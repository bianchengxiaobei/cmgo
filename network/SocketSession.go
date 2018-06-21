package network

import (
	"net"
	"sync"
	"time"
	"cmattribute"
	"cmtime"
	"bytes"
	"fmt"
	"sync/atomic"
)

const (
	maxReadBufferLen = 4*1024
	period		= 60 * 1e9
	waitDuration = 3e9
)
var wheel = cmtime.NewWheel(time.Duration(100 * float64(time.Millisecond)),1200)
type SocketSession struct{
	SocketConnectInterface
	once       sync.Once
	done       chan struct{}
	lockNum    int32
	readQueue  chan interface{}
	writeQueue chan interface{}
	handler    EventHandleInterface
	period     time.Duration
	wait       time.Duration
	rwLock     sync.RWMutex
	attrs      *cmattribute.ValuesContext
	coder      Protocol
}
type SocketSessionInterface interface {
	SocketConnectInterface
	GetConn() net.Conn
	SetProtocol(coder Protocol)
	SetHandler(handler EventHandleInterface)
	GetAttribute(key interface{}) interface{}
	RemoveAttribute(key interface{})
	SetAttribute(key interface{}, value interface{})
	SetReadChan(int)
	SetWriteChan(int)
	CloseChan()
}
//创建SocketSession和SocketConnection
func CreateSocketSession(conn net.Conn) *SocketSession{
	connect := CreateTcpConnection(conn)
	session := &SocketSession{
		SocketConnectInterface : connect,
		done:make(chan struct{}),
		period:period,
		wait:waitDuration,
		attrs:cmattribute.NewValuesContext(nil),
	}
	session.SocketConnectInterface.SetSocketSession(session)
	session.SetWriteTimeout(IOTimeout)
	session.SetReadTimeout(IOTimeout)
	return session
}
func (session *SocketSession) GetConn() net.Conn{
	if tcpConn,ok := session.SocketConnectInterface.(*SocketTcpConnect);ok{
		return tcpConn.conn
	}
	return nil
}
//设置编解码
func (session *SocketSession) SetProtocol(coder Protocol){
	session.coder = coder
}
//设置事件处理器
func (session *SocketSession) SetHandler(handler EventHandleInterface){
	session.handler = handler
}
//取得session参数
func (session *SocketSession) GetAttribute(key interface{}) interface{} {
	session.rwLock.RLock()
	ret, flag := session.attrs.Get(key)
	session.rwLock.RUnlock()

	if !flag {
		return nil
	}
	return ret
}
// 设置session参数
func (session *SocketSession) SetAttribute(key interface{}, value interface{}) {
	session.rwLock.Lock()
	session.attrs.Set(key, value)
	session.rwLock.Unlock()
}

// 移除session参数
func (session *SocketSession) RemoveAttribute(key interface{}) {
	session.rwLock.Lock()
	session.attrs.Delete(key)
	session.rwLock.Unlock()
}
func (session *SocketSession) SetReadChan(len int){
	if len < 1{
		panic("readChanLen < 1")
	}
	session.rwLock.Lock()
	session.readQueue = make(chan interface{},len)
	session.rwLock.Unlock()
}
func (session *SocketSession) SetWriteChan(len int){
	if len < 1{
		panic("writeChanLen < 1")
	}
	session.rwLock.Lock()
	session.writeQueue = make(chan interface{},len)
	session.rwLock.Unlock()
}
//是否Session关闭了
func (session *SocketSession) IsClosed() bool{
	select {
	case <-session.done:
		return true
	default:
		return false
	}
}
func (session *SocketSession) CloseChan(){
	select {
	case <-session.done:
		return
	default:
		session.once.Do(func() {
			now := wheel.Now()
			if conn := session.GetConn(); conn != nil {
				conn.SetReadDeadline(now.Add(session.GetReadTimeout()))
				conn.SetWriteDeadline(now.Add(session.GetWriteTimeout()))
			}
			close(session.done)
		})
	}
}
func (session *SocketSession) run(socket ISocket){
	if session.readQueue == nil || session.writeQueue == nil{
		panic("readWrite == nil")
	}
	if session.SocketConnectInterface == nil || session.handler == nil{
		panic("ConnectHandler == nil")
	}
	session.UpdateActiveTime()
	if err:=session.handler.SessionOpened(session);err !=nil{
		session.CloseChan()
		return
	}
	atomic.AddInt32(&(session.lockNum),2)
	//loopHandle
	go session.workLoop(socket)
	//handlemessage
	go session.messageLoop()
}
func (session *SocketSession) workLoop(socket ISocket){
	var (
		inData interface{}
		outData interface{}
		counter time.Time
		err error
		server *TcpServer
		client *TcpClient
	)
	server,_ = socket.(*TcpServer)
	client,_ = socket.(*TcpClient)
	//关闭session
	defer func() {
		fmt.Println("session closed!")
		//防止messageLoop关闭
		atomic.AddInt32(&(session.lockNum),-1)
		session.handler.SessionClosed(session)
		session.gc()
		if server != nil{
			server.waitGroup.Done()
		}else{
			client.waitGroup.Done()
		}
	}()
LOOP:
	for{
		select {
		case <-session.done:
			if atomic.LoadInt32(&(session.lockNum)) == 1{
				if len(session.readQueue) == 0 && len(session.writeQueue) == 0{
					break LOOP
				}
				counter = time.Now()
				if time.Since(counter).Nanoseconds() > session.wait.Nanoseconds(){
					break LOOP
				}
			}
		case inData = <-session.readQueue:
			session.handler.MessageReceived(session,inData)
		case outData = <-session.writeQueue:
			if err = session.coder.Encode(session,outData);err != nil{
				
			}
			session.handler.MessageSent(session,outData)
		case <-wheel.After(session.period)://定时
			session.handler.SessionPeriod(session)

		}
	}
}
func (session *SocketSession) messageLoop(){
	var(
		err error
		data []byte
		dataLen int
		stop bool
		dataBuffer *bytes.Buffer
		message interface{}
		messageLen int
	)
	//close第一次
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Error")
		}
		atomic.AddInt32(&(session.lockNum),-1)
		session.CloseChan() //close(done)
		if err != nil{
			session.handler.ExceptionCaught(session,err)
		}
	}()
	data = make([]byte,maxReadBufferLen)
	dataBuffer = new(bytes.Buffer)
	for{
		if session.IsClosed(){
			err = nil
			break
		}
		dataLen = 0
		for{
			dataLen,err = session.Read(data)
			if err != nil{
				if netError,ok := err.(net.Error);ok && netError.Timeout(){
					break
				}
				stop = true
			}
			break
		}
		if stop{
			break
		}
		if dataLen == 0{
			continue
		}
		dataBuffer.Write(data)
		for{
			if dataBuffer.Len() < 0{
				break
			}
			message,messageLen,err = session.coder.Decode(session,dataBuffer.Bytes())
			if err != nil{
				stop = true
				break
			}
			if message == nil{
				break
			}
			session.UpdateActiveTime()
			session.readQueue<-message
			dataBuffer.Next(messageLen)
		}
		if stop{
			break
		}
	}
}
//关闭session(gc)
func (session *SocketSession) gc(){
	session.rwLock.Lock()
	if session.attrs != nil{
		session.attrs = nil
		close(session.writeQueue)
		session.writeQueue = nil
		close(session.readQueue)
		session.readQueue = nil
		session.SocketConnectInterface.Close((int)((int64)(session.wait)))
	}
	session.rwLock.Unlock()
}