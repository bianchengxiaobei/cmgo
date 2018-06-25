package network


import (
	"sync"
	"net"
	"github.com/bianchengxiaobei/cmgo/log4g"
)

const (
	connectTimeout = 5e9
)
type TcpClient struct {
	lock          sync.Mutex
	TcpVersion    string
	waitGroup     sync.WaitGroup
	SessionConfig *SocketSessionConfig
	codec         Protocol
	handler       EventHandleInterface
	Session       *SocketSession
	sync.Once
	done          chan struct{}
}
//创建一个新的TcpServer
func NewTcpClient(tcpVersion string, sessionConfig *SocketSessionConfig) *TcpClient {
	if tcpVersion == "" {
		panic("tcpVersion: == null")
	}
	client := &TcpClient{
		TcpVersion:    		tcpVersion,
		SessionConfig: 		sessionConfig,
		done:				make(chan struct{}),
	}
	return client
}
func (client *TcpClient)Connect(addr string){
	var(
		err error
		conn net.Conn
	)
	if addr == "" {
		log4g.Error("客户端连接地址为空！")
	}
	conn,err = net.DialTimeout(client.TcpVersion,addr,connectTimeout)
	if err != nil{
		panic("连接失败")
	}
	session := client.CreateSessionConnect(conn)
	if session != nil{
		client.Session = session
		client.waitGroup.Add(1)
		client.run()
	}
	log4g.Info("客户端连接成功！")
}
func (client *TcpClient)run(){
	if client.Session != nil{
		if client.IsClosed(){
			log4g.Info("客户端已经关闭!")
			return
		}
		client.Session.run(client)
	}
}
//创建客户端session
func (client *TcpClient)CreateSessionConnect(conn net.Conn) *SocketSession{
	var (
		tcpConn *net.TCPConn
		ok bool
	)
	session := CreateSocketSession(conn)
	if tcpConn,ok = conn.(*net.TCPConn);!ok{
		panic("not tcpConn")
	}
	session.SetProtocol(client.codec)
	session.SetReadChan(client.SessionConfig.ReadChanLen)
	session.SetWriteChan(client.SessionConfig.WriteChanLen)
	session.SetHandler(client.handler)
	tcpConn.SetNoDelay(client.SessionConfig.TcpNoDelay)
	tcpConn.SetKeepAlive(client.SessionConfig.TcpKeepAlive)
	tcpConn.SetKeepAlivePeriod(client.SessionConfig.TcpKeepAlivePeriod)
	tcpConn.SetReadBuffer(client.SessionConfig.TcpReadBuffSize)
	tcpConn.SetWriteBuffer(client.SessionConfig.TcpWriteBuffSize)
	return session
}
//设置编解码
func (client *TcpClient) SetProtocolCodec(protocol Protocol) {
	client.codec = protocol
}

//设置消息处理器
func (client *TcpClient) SetMessageHandler(handler EventHandleInterface) {
	client.handler = handler
}
func (client *TcpClient) Close(){
	select {
	case <-client.done:
		return
	default:
		client.Once.Do(func() {
			log4g.Info("关闭客户端")
			close(client.done)
			client.Session.CloseChan()
		})
	}
	client.waitGroup.Wait()
}
func (client *TcpClient) IsClosed() bool{
	select {
	case <-client.done:
		return true
	default:
		return false
	}
}