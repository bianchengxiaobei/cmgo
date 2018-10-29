package network

import (
	"github.com/xtaci/kcp-go"
	"sync"
	"strings"
	"strconv"
	"github.com/bianchengxiaobei/cmgo/log4g"
	"fmt"
	"net"
	"time"
)

type P2PKcpServer struct {
	Listener      *net.UDPConn
	localAddr	*net.UDPAddr
	remoteAddr  *net.UDPAddr
	lock          sync.Mutex
	waitGroup     sync.WaitGroup
	readBuffer		[]byte
	SessionConfig *SocketSessionConfig
	codec         Protocol
	handler       EventHandleInterface
	sync.Once
	done chan struct{}
	Sessions 	map[uint32]SocketSessionInterface
}
func (server *P2PKcpServer) Bind(addr string) error{
	if addr == "" {
		return ConnectAddressNilError
	}
	udpAddr,err := net.ResolveUDPAddr("udp",addr)
	server.remoteAddr = udpAddr
	l,err := net.DialUDP("udp",nil,udpAddr)
	//l,err := net.ListenUDP("udp",localAddr)
	if err != nil{
		return err
	}
	server.Listener = l
	server.localAddr = l.LocalAddr().(*net.UDPAddr)
	log4g.Infof("本地地址:%s",server.localAddr.String())
	server.Listener.SetReadBuffer(512)
	server.Listener.SetWriteBuffer(512)
	//发送自己网关的id
	server.Listener.Write([]byte("Server"))
	go server.run()
	return nil
}
func (server *P2PKcpServer)Connect(addr string) error{
	return nil
}
func (server *P2PKcpServer) run() {
	for {
		if server.IsClosed(){
			log4g.Info("服务器已经关闭!")
			return
		}
		session, err := server.accept()
		if err != nil {
			if session == nil{
				continue
			}
		}
		if session == nil{
			continue
		}
		server.waitGroup.Add(1)
		session.run(server)
	}
	////阻塞线程，等全部完成之后结束
	//server.waitGroup.Wait()
}
func (server *P2PKcpServer) accept() (*SocketSession, error) {
	var (
		ok               bool
		kcpSess          *kcp.UDPSession
		clientAddrString string
		clientId         int
		clientAddr	*net.UDPAddr
		len int
		err error
	)
	len ,err = server.Listener.Read(server.readBuffer)
	if err != nil{
		fmt.Println(err.Error())
		return nil, err
	}
	if len > 0{
		content := string(server.readBuffer[:len])
		if strings.Contains(content,";"){
			a := strings.Split(content, ";")
			clientId, err = strconv.Atoi(a[1])
			if err != nil {
				fmt.Println(err.Error())
			}
			clientAddrString = a[0]
		}else{
			return nil, nil
		}
	}
	l,err := net.DialUDP("udp",nil,server.remoteAddr)
	if err != nil{
		return nil,err
	}
	l.Write([]byte("ServerClient"))
	clientAddr, err = net.ResolveUDPAddr("udp", clientAddrString)
	if err != nil {
		fmt.Println(err)
	}
	clientC, err := net.DialUDP("udp", l.LocalAddr().(*net.UDPAddr), clientAddr)
	if err != nil{
		fmt.Println(err.Error())
	}
	defer l.Close()
	//随意往客户端发送一个确认消息
	clientC.Write([]byte("Hello"))
	server.Listener.Write([]byte(strconv.Itoa(clientId)))
	len,err = clientC.Read(server.readBuffer)
	if err == nil && len > 0{
		clientContent := string(server.readBuffer[:len])
		if clientContent != "Hello"{
			return nil, nil
		}
	}
	kcpConn, err := kcp.DialWithLocal(clientC,clientAddr)
	if err != nil {
		if kcpConn != nil{
			kcpConn.Close()
		}
		return nil, err
	}
	if kcpSess, ok = kcpConn.(*kcp.UDPSession); !ok {
		return nil, NotKcpConnError
	}
	session,err := CreateKcpSocketSession(kcpSess)
	if err != nil{
		return nil, err
	}
	session.SetProtocol(server.codec)
	session.SetHandler(server.handler)
	session.SetReadChan(server.SessionConfig.ReadChanLen)
	session.SetWriteChan(server.SessionConfig.WriteChanLen)
	session.SetPeriod(server.SessionConfig.PeriodTime)
	kcpSess.SetReadBuffer(server.SessionConfig.TcpReadBuffSize)
	kcpSess.SetWriteBuffer(server.SessionConfig.TcpWriteBuffSize)
	kcpSess.SetStreamMode(false)
	kcpSess.SetWindowSize(300, 300)
	kcpSess.SetNoDelay(1, 10, 2, 1)
	kcpSess.SetDSCP(46)
	kcpSess.SetMtu(1400)
	kcpSess.SetACKNoDelay(false)
	kcpSess.SetReadDeadline(time.Now().Add(time.Hour))
	kcpSess.SetWriteDeadline(time.Now().Add(time.Hour))
	server.Sessions[session.Id()] = session
	kcpSess.Write([]byte("Start"))
	return session, nil
}
//是否已经关闭
func (server  *P2PKcpServer) IsClosed() bool{
	select {
	case <-server.done:
		return true
	default:
		return false
	}
}
func (server *P2PKcpServer)Close(){
	select {
	case <-server.done:
		return
	default:
		server.Once.Do(func() {
			close(server.done)
			if server.Listener != nil {
				server.Listener.Close()
				server.Listener = nil
			}
			for _,session:= range server.Sessions {
				session.CloseChan()
			}
		})
	}
	server.waitGroup.Wait()
}
func (server *P2PKcpServer) DoneWaitGroup(){
	server.waitGroup.Done()
}
//创建一个新的KcpServer
func NewP2PKcpServer(sessionConfig *SocketSessionConfig) *P2PKcpServer {
	server := &P2PKcpServer{
		readBuffer:make([]byte,1024),
		SessionConfig: 	sessionConfig,
		Sessions:make(map[uint32]SocketSessionInterface),
		done:			make(chan struct{}),
	}
	return server
}
//设置编解码
func (server *P2PKcpServer) SetProtocolCodec(protocol Protocol) {
	server.codec = protocol
}

//设置消息处理器
func (server *P2PKcpServer) SetMessageHandler(handler EventHandleInterface) {
	server.handler = handler
}
func (server *P2PKcpServer)GetSessionConfig()*SocketSessionConfig{
	return server.SessionConfig
}