package network

import (
	"net"
	"sync"
	"time"
	"github.com/bianchengxiaobei/cmgo/cmattribute"
	"bytes"
	"sync/atomic"
	"errors"
	"github.com/xtaci/kcp-go"
)

const (
	maxReadBufferLen = 4*1024
	period		= 5e9			//5s
	waitDuration = 3e9				//3s
)
//var wheel = cmtime.NewWheel(time.Duration(100 * float64(time.Millisecond)),1200)
var timer = time.NewTicker(5 * time.Second)
var ErrSessionClosed = errors.New("Session已经关闭!")
var ErrSessionBlocked = errors.New("Session阻塞!")
type SocketSession struct{
	SocketConnectInterface
	once          sync.Once
	done          chan struct{}
	lockNum       int32
	readQueue     chan interface{}
	writeQueue    chan interface{}
	handler       EventHandleInterface
	period        time.Duration
	closeWaitTime time.Duration
	rwLock        sync.RWMutex
	attrs         *cmattribute.ValuesContext
	coder         Protocol
}
type WriteMessage struct {
	MsgId 		int
	MsgData		interface{}
}
type InnerWriteMessage struct {
	RoleId	int64
	MsgData interface{}
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
	SetPeriod(duration time.Duration)
	SetSesionCloseWaitTime(wait time.Duration)
	CloseChan()
	IsClosed() bool
	WriteMsg(msgId int,message interface{}) error
	WriteBytes([]byte)error
}
//创建SocketSession和SocketConnection
func CreateTcpSocketSession(conn net.Conn) (*SocketSession,error){
	connect,err := CreateTcpConnection(conn)
	if err != nil{
		return nil, err
	}
	session := &SocketSession{
		SocketConnectInterface : connect,
		done:                    make(chan struct{}),
		period:                  period,
		closeWaitTime:           waitDuration,
		attrs:                   cmattribute.NewValuesContext(nil),
	}
	session.SocketConnectInterface.SetSocketSession(session)
	session.SetWriteTimeout(IOTimeout)
	session.SetReadTimeout(IOTimeout)
	return session,nil
}
func CreateKcpSocketSession(conn *kcp.UDPSession)(*SocketSession,error){
	connect,err := CreateKcpConnection(conn)
	if err != nil{
		return nil, err
	}
	session := &SocketSession{
		SocketConnectInterface : connect,
		done:                    make(chan struct{}),
		period:                  period,
		closeWaitTime:           waitDuration,
		attrs:                   cmattribute.NewValuesContext(nil),
	}
	session.SocketConnectInterface.SetSocketSession(session)
	session.SetWriteTimeout(IOTimeout)
	session.SetReadTimeout(IOTimeout)
	return session,nil
}
func (session *SocketSession) GetConn() net.Conn{
	if tcpConn,ok := session.SocketConnectInterface.(*SocketTcpConnect);ok{
		return tcpConn.conn
	}else if kcpConn,ok := session.SocketConnectInterface.(*SocketKcpConnect);ok{
		return kcpConn.conn
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
//设置关闭session等待时间
func (session *SocketSession)SetSesionCloseWaitTime(wait time.Duration)  {
	if wait < 1{
		panic("cloaseWaitTime < 1")
	}
	session.rwLock.Lock()
	session.closeWaitTime = wait
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
func (session *SocketSession)SetPeriod(duration time.Duration){
	if duration <= 0{
		panic("period < 0")
	}
	session.period = duration
}
//写消息
func (session *SocketSession)WriteMsg(msgId int,message interface{}) error{
	if session.IsClosed(){
		return	ErrSessionClosed
	}
	writeMsg := WriteMessage{
		MsgId:msgId,
		MsgData:message,
	}
	select {
	case session.writeQueue<-writeMsg:
		break
	case <-time.After(waitDuration):
			return ErrSessionBlocked
	}
	return nil
}
//写数据
func (session *SocketSession)WriteBytes(data []byte) error{
	if session.IsClosed(){
		return	ErrSessionClosed
	}
	if _,err := session.SocketConnectInterface.Write(data);err != nil{
		return err
	}
	return nil
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
			now := time.Now()
			if conn := session.GetConn(); conn != nil {
				if _,ok := conn.(*kcp.UDPSession);ok == false{
					conn.SetReadDeadline(now.Add(session.GetReadTimeout()))
					conn.SetWriteDeadline(now.Add(session.GetWriteTimeout()))
				}
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
	)
	//关闭session
	defer func() {
		//log4g.Infof("[Session:id:%d]关闭!",session.Id())
		//防止messageLoop关闭
		atomic.AddInt32(&(session.lockNum),-1)
		session.handler.SessionClosed(session,err)
		session.gc()
		socket.DoneWaitGroup()
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
				if time.Since(counter).Nanoseconds() > session.closeWaitTime.Nanoseconds(){
					break LOOP
				}
			}
		//case <-wheel.After(session.period)://定时
		//	session.handler.SessionPeriod(session)
		case <- timer.C:
			session.handler.SessionPeriod(session)
		case inData = <-session.readQueue:
			session.handler.MessageReceived(session,inData)
		case outData = <-session.writeQueue:
			if err = session.coder.Encode(session,outData);err != nil{
				session.CloseChan()
			}
			session.handler.MessageSent(session,outData)
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
		data = nil
		dataBuffer = nil
		atomic.AddInt32(&(session.lockNum),-1)
		session.CloseChan() //close(done)
		if r := recover(); r != nil {
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
		dataBuffer.Write(data[:dataLen])
		for{
			if dataBuffer.Len() <= 0{
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
			if chanMessage,ok:=message.(WriteMessage);ok{
				session.UpdateActiveTime()
				session.readQueue<-chanMessage
				dataBuffer.Next(messageLen)
			}else {
				panic("not a chanMessage")
			}
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
		session.SocketConnectInterface.Close((int)((int64)(session.closeWaitTime)))
	}
	session.rwLock.Unlock()
}