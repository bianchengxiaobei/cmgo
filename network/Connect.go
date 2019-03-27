package network

import (
	"net"
	"io"
	"time"
	"sync/atomic"
	"errors"
	"github.com/xtaci/kcp-go"
)

const (
	IOTimeout = 1e9	//1s
)
var serverInitTime time.Time = time.Now()//服务器启动时刻
var TcpConnNullError = errors.New("TcpConn == null!")
var KcpConnNullError = errors.New("KcpConn == null!")
type SocketConnect struct{
	id	uint32
	activeTime	int64
	readTimeout	time.Duration
	writeTimeout time.Duration
	readLastDeadTime time.Time
	writeLastDeadTime time.Time
	localAddr string
	remoteAddr string
	session SocketSessionInterface
}
type SocketTcpConnect struct{
	conn	net.Conn
	reader	io.Reader
	writer	io.Writer
	SocketConnect
}
type SocketKcpConnect struct {
	conn *kcp.UDPSession
	SocketConnect
}

type SocketConnectInterface interface {
	Id() uint32
	LocalAddr() string
	RemoteAddr() string
	UpdateActiveTime()
	GetActiveTime() time.Time
	GetReadTimeout() time.Duration
	SetReadTimeout(time.Duration)
	GetWriteTimeout() time.Duration
	SetWriteTimeout(time.Duration)
	SetSocketSession(SocketSessionInterface)
	Read([]byte)(int,error)
	Write([]byte)(int,error)
	Close(waitSecondTime int)
}
var connectId uint32
//创建TcpConnect
func CreateTcpConnection(conn net.Conn) (*SocketTcpConnect,error) {
	if conn == nil{
		return nil,TcpConnNullError
	}
	localAddr := conn.LocalAddr().String()
	peerAddr := conn.RemoteAddr().String()
	return &SocketTcpConnect{
		conn:conn,
		reader:io.Reader(conn),
		writer:io.Writer(conn),
		SocketConnect:SocketConnect{
			id:atomic.AddUint32(&connectId,1),
			readTimeout:IOTimeout,
			writeTimeout:IOTimeout,
			localAddr:localAddr,
			remoteAddr:peerAddr,
		},
	},nil
}
//创建KCP连接
func CreateKcpConnection(conn *kcp.UDPSession)(*SocketKcpConnect,error){
	if conn == nil{
		return nil,KcpConnNullError
	}
	localAddr := conn.LocalAddr().String()
	peerAddr := conn.RemoteAddr().String()
	return &SocketKcpConnect{
		conn:conn,
		SocketConnect:SocketConnect{
			id:atomic.AddUint32(&connectId,1),
			readTimeout:IOTimeout,
			writeTimeout:IOTimeout,
			localAddr:localAddr,
			remoteAddr:peerAddr,
		},
	},nil
}
func (connect *SocketConnect) Id() uint32{
	return connect.id
}
func (connect *SocketConnect) LocalAddr() string{
	return connect.localAddr
}
func (connect *SocketConnect) RemoteAddr() string{
	return connect.remoteAddr
}
func (connect *SocketConnect) UpdateActiveTime() {
	atomic.StoreInt64(&(connect.activeTime),int64(time.Since(serverInitTime)))
}
func (connect *SocketConnect) GetActiveTime() time.Time{
	return serverInitTime.Add(time.Duration(atomic.LoadInt64(&(connect.activeTime))))
}
func (connect *SocketConnect) GetReadTimeout() time.Duration{
	return connect.readTimeout
}
func (connect *SocketConnect) SetReadTimeout(readTime time.Duration){
	if readTime < 1{
		panic("readTime < 1")
	}
	connect.readTimeout = readTime
	if connect.writeTimeout == 0{
		connect.writeTimeout = readTime
	}
}
func (connect *SocketConnect) GetWriteTimeout() time.Duration{
	return connect.writeTimeout
}
func (connect *SocketConnect) SetWriteTimeout(writeTime time.Duration){
	if writeTime < 1{
		panic("writeTime < 1")
	}
	connect.writeTimeout = writeTime
	if connect.readTimeout == 0{
		connect.readTimeout = writeTime
	}
}
//关闭连接
func (connect *SocketTcpConnect) Close(waitSecondTime int){
	if connect.conn != nil{
		connect.conn.(*net.TCPConn).SetLinger(waitSecondTime)
		connect.conn.Close()
		connect.conn = nil
	}
}
func (connect *SocketKcpConnect)Close(waitSecondTime int){
	if connect.conn != nil{
		connect.conn.Close()
		connect.conn = nil
	}
}
func (connect *SocketConnect) SetSocketSession(session SocketSessionInterface){
	connect.session = session
}
func (connect *SocketTcpConnect) Read(data []byte)(int,error){
	if connect.readTimeout > 0{
		currentTime := time.Now()
		if currentTime.Sub(connect.readLastDeadTime) > (connect.readTimeout >> 2){
			if err := connect.conn.SetWriteDeadline(currentTime.Add(connect.readTimeout));err!=nil{
				return 0,err
			}
			connect.readLastDeadTime = currentTime
		}
	}
	if length,err:=connect.reader.Read(data);err == nil{
		//atomic.AddUint32(&connect.readBytes,uint32(length))
		return length,err
	}
	return 0,nil
}
func (connect *SocketKcpConnect)Read(data []byte)(int,error)  {
	if connect.readTimeout > 0{
		currentTime := time.Now()
		if currentTime.Sub(connect.readLastDeadTime) > (connect.readTimeout >> 2){
			if err := connect.conn.SetWriteDeadline(currentTime.Add(connect.readTimeout));err!=nil{
				return 0,err
			}
			connect.readLastDeadTime = currentTime
		}
	}
	if length,err:=connect.conn.Read(data);err == nil{
		//atomic.AddUint32(&connect.readBytes,uint32(length))
		return length,err
	}
	return 0,nil
}
func (connect *SocketTcpConnect) Write(data []byte)(int, error){
	if connect.writeTimeout > 0{
		currentTime := time.Now()
		if currentTime.Sub(connect.writeLastDeadTime) > (connect.writeTimeout >> 2){
			if err := connect.conn.SetWriteDeadline(currentTime.Add(connect.readTimeout));err!=nil{
				return 0,err
			}
			connect.writeLastDeadTime = currentTime
		}
	}
	len,err:=connect.writer.Write(data)
	//if err == nil{
	//	atomic.AddUint32(&connect.writeBytes,uint32(len))
	//}
	return len,err
}
func (connect *SocketKcpConnect)Write(data []byte)(int,error)  {
	if connect.writeTimeout > 0{
		currentTime := time.Now()
		if currentTime.Sub(connect.writeLastDeadTime) > (connect.writeTimeout >> 2){
			if err := connect.conn.SetWriteDeadline(currentTime.Add(connect.readTimeout));err!=nil{
				return 0,err
			}
			connect.writeLastDeadTime = currentTime
		}
	}
	len,err:=connect.conn.Write(data)
	//if err == nil{
	//	atomic.AddUint32(&connect.writeBytes,uint32(len))
	//}
	return len,err
}
























