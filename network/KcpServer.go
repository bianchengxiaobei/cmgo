package network

import (
	"github.com/bianchengxiaobei/cmgo/log4g"
	"gitee.com/Caomao/kcp-go"
	"sync"
	"time"
)

type KcpServer struct {
	Listener      *kcp.Listener
	lock          sync.Mutex
	waitGroup     sync.WaitGroup
	SessionConfig *SocketSessionConfig
	codec         Protocol
	handler       EventHandleInterface
	sync.Once
	done     chan struct{}
	Sessions map[uint32]SocketSessionInterface
}

func (server *KcpServer) Bind(addr string) error {
	if addr == "" {
		return ConnectAddressNilError
	}
	l, err := kcp.Listen(addr)
	if err != nil {
		return err
	}
	server.Listener = l.(*kcp.Listener)
	server.Listener.SetReadBuffer(1024)
	server.Listener.SetWriteBuffer(1024)
	server.Listener.SetDSCP(46)
	go server.run()
	return nil
}
func (server *KcpServer) Connect(addr string) error {
	return nil
}
func (server *KcpServer) run() {
	for {
		if server.IsClosed() {
			log4g.Info("服务器已经关闭!")
			return
		}
		session, err := server.accept()
		if err != nil {
			if session == nil {
				continue
			}
		}
		server.waitGroup.Add(1)
		session.run(server)
	}
	////阻塞线程，等全部完成之后结束
	//server.waitGroup.Wait()
}
func (server *KcpServer) accept() (*SocketSession, error) {
	var (
		ok      bool
		kcpSess *kcp.UDPSession
	)
	kcpConn, err := server.Listener.Accept()
	if err != nil {
		if kcpConn != nil {
			kcpConn.Close()
		}
		return nil, err
	}
	if kcpSess, ok = kcpConn.(*kcp.UDPSession); !ok {
		return nil, NotKcpConnError
	}
	session, err := CreateKcpSocketSession(kcpSess)
	if err != nil {
		return nil, err
	}
	session.SetProtocol(server.codec)
	session.SetHandler(server.handler)
	session.SetReadChan(server.SessionConfig.ReadChanLen)
	session.SetWriteChan(server.SessionConfig.WriteChanLen)
	session.SetPeriod(server.SessionConfig.PeriodTime)
	kcpSess.SetReadBuffer(server.SessionConfig.TcpReadBuffSize)
	kcpSess.SetWriteBuffer(server.SessionConfig.TcpWriteBuffSize)
	kcpSess.SetStreamMode(true)
	kcpSess.SetWindowSize(300, 300)
	kcpSess.SetNoDelay(1, 10, 2, 1)
	//kcpSess.SetDSCP(46)
	kcpSess.SetMtu(1400)
	kcpSess.SetACKNoDelay(false)
	kcpSess.SetReadDeadline(time.Now().Add(time.Hour))
	kcpSess.SetWriteDeadline(time.Now().Add(time.Hour))
	server.Sessions[session.Id()] = session
	return session, nil
}
func (server *KcpServer) Close() {
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
			for _, session := range server.Sessions {
				session.CloseChan()
			}
		})
	}
	server.waitGroup.Wait()
}

//是否已经关闭
func (server *KcpServer) IsClosed() bool {
	select {
	case <-server.done:
		return true
	default:
		return false
	}
}
func (server *KcpServer) DoneWaitGroup() {
	server.waitGroup.Done()
}

//创建一个新的KcpServer
func NewKcpServer(sessionConfig *SocketSessionConfig) *KcpServer {
	server := &KcpServer{
		SessionConfig: sessionConfig,
		Sessions:      make(map[uint32]SocketSessionInterface),
		done:          make(chan struct{}),
	}
	return server
}

//设置编解码
func (server *KcpServer) SetProtocolCodec(protocol Protocol) {
	server.codec = protocol
}

//设置消息处理器
func (server *KcpServer) SetMessageHandler(handler EventHandleInterface) {
	server.handler = handler
}
func (server *KcpServer) GetSessionConfig() *SocketSessionConfig {
	return server.SessionConfig
}
func (server *KcpServer) RemoveSession(sessionId uint32) {
	delete(server.Sessions, sessionId)
}
