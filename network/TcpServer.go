package network

import (
	"net"
	"sync"
	"time"
	"github.com/bianchengxiaobei/cmgo/log4g"
)

type TcpServer struct {
	SocketAddress string
	Listener      net.Listener
	lock          sync.Mutex
	TcpVersion    string
	waitGroup     sync.WaitGroup
	SessionConfig *SocketSessionConfig
	codec         Protocol
	handler       EventHandleInterface
	sync.Once
	done chan struct{}
	Sessions 	map[uint32]SocketSessionInterface
}
type SocketSessionConfig struct {
	SocketAddr			string
	TcpNoDelay         bool
	TcpKeepAlive       bool
	TcpKeepAlivePeriod time.Duration
	TcpReadBuffSize    int
	TcpWriteBuffSize   int
	ReadChanLen			int
	WriteChanLen		int
}
type ISocket interface {
	run()
	Close()
	IsClosed()	bool
}
//服务器开始监听断开
func (server *TcpServer) Bind(addr string) error{
	if addr == "" {
		log4g.Error("服务器监听地址为空!")
	}
	server.SocketAddress = addr
	listener, err := net.Listen(server.TcpVersion, server.SocketAddress)
	if err != nil {
		log4g.Errorf("服务器[%s]监听出错,错误:%s",server.SocketAddress,err.Error())
		return err
	}
	server.Listener = listener
	go server.run()
	return nil
}
func (server *TcpServer) run() {
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
		server.waitGroup.Add(1)
		session.run(server)
	}
	////阻塞线程，等全部完成之后结束
	//server.waitGroup.Wait()
}
func (server *TcpServer) accept() (*SocketSession, error) {
	var (
		ok   bool
		conn *net.TCPConn
	)
	tcpConn, err := server.Listener.Accept()
	if err != nil {
		if tcpConn != nil{
			tcpConn.Close()
		}
		return nil, err
	}
	session := CreateSocketSession(tcpConn)
	if conn, ok = session.GetConn().(*net.TCPConn); !ok {
		panic("not a tcpConn")
	}
	session.SetProtocol(server.codec)
	session.SetHandler(server.handler)
	session.SetReadChan(server.SessionConfig.ReadChanLen)
	session.SetWriteChan(server.SessionConfig.WriteChanLen)
	conn.SetNoDelay(server.SessionConfig.TcpNoDelay)
	conn.SetKeepAlive(server.SessionConfig.TcpKeepAlive)
	conn.SetKeepAlivePeriod(server.SessionConfig.TcpKeepAlivePeriod)
	conn.SetReadBuffer(server.SessionConfig.TcpReadBuffSize)
	conn.SetWriteBuffer(server.SessionConfig.TcpWriteBuffSize)
	server.Sessions[session.Id()] = session
	return session, nil
}

//创建一个新的TcpServer
func NewTcpServer(tcpVersion string, sessionConfig *SocketSessionConfig) *TcpServer {
	if tcpVersion == "" {
		log4g.Error("tcpVersion: == null")
	}
	server := &TcpServer{
		TcpVersion:    	tcpVersion,
		SessionConfig: 	sessionConfig,
		Sessions:make(map[uint32]SocketSessionInterface),
		done:			make(chan struct{}),
	}
	return server
}

//设置编解码
func (server *TcpServer) SetProtocolCodec(protocol Protocol) {
	server.codec = protocol
}

//设置消息处理器
func (server *TcpServer) SetMessageHandler(handler EventHandleInterface) {
	server.handler = handler
}

//关闭服务器
func (server *TcpServer) Close() {
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
//是否已经关闭
func (server  *TcpServer) IsClosed() bool{
	select {
	case <-server.done:
		return true
	default:
		return false
	}
}
