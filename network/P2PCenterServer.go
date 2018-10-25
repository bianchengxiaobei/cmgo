package network

import (
	"net"
	"github.com/xtaci/kcp-go"
	"fmt"
	"sync"
	"strconv"
)

type P2PCenterServer struct {
	readBuffer	[]byte
	serverId	int32
	clientId	int32
	ServerConnMap	map[int32]*kcp.UDPSession
	ClientConnMap	map[int32]*kcp.UDPSession
	ServerAddrByteMap	map[int32][]byte
	ServerAddrMap	map[int32]*net.UDPAddr
	ClientAddrMap	map[int32]*net.UDPAddr
	sync.Locker
}
func NewP2PCenterServer()*P2PCenterServer{
	server := &P2PCenterServer{
		readBuffer:make([]byte,512),
		serverId:0,
		clientId:0,
		ServerConnMap:make(map[int32]*kcp.UDPSession),
		ClientConnMap:make(map[int32]*kcp.UDPSession),
		ServerAddrByteMap:make(map[int32][]byte),
		ServerAddrMap:make(map[int32]*net.UDPAddr),
		ClientAddrMap:make(map[int32]*net.UDPAddr),
	}
	return server
}
func (server *P2PCenterServer)Bind(addr string) error{
	conn,err := kcp.Listen(addr)
	if err != nil{
		return  err
	}
	for {
		s,err := conn.Accept()
		if err != nil{
			continue
		}
		if kcpSess,ok :=s.(*kcp.UDPSession);ok{
			//读取消息
			go server.HandlerConn(kcpSess)
		}
	}
}
func (server *P2PCenterServer)HandlerConn(session *kcp.UDPSession){
	for {
		len, err := session.Read(server.readBuffer)
		if err != nil {
			fmt.Println(err)
			return
		}
		if len > 0 {
			content := string(server.readBuffer[:len])
			if content == "Server" {
				id := server.GetServerId()
				udpAddr := session.RemoteAddr().(*net.UDPAddr)
				server.ServerAddrMap[id] = udpAddr
				server.ServerAddrByteMap[id] = []byte(udpAddr.String())
				server.ServerConnMap[id] = session
			} else if content == "Client" {
				id := server.GetClientId()
				udpAddr := session.RemoteAddr().(*net.UDPAddr)
				server.ClientAddrMap[id] = udpAddr
				server.ClientConnMap[id] = session
				clientAddrAndId := fmt.Sprintf("%s;%d", udpAddr.String(), id)
				server.ServerConnMap[1].Write([]byte(clientAddrAndId))//这里1写死，本来是动态负载均衡网关id
				break
			} else {
				//服务器发送的客户端id，请求客户端开始向他发送验证打洞数据
				id, err := strconv.Atoi(content)
				if err == nil {
					id3 := int32(id)
					con := server.ClientConnMap[id3]
					if con != nil {
						len, err = con.Write(server.ServerAddrByteMap[1])
						if err != nil {
							fmt.Println(err)
						}
						server.DeleteClient(id3)
					}
				} else {
					fmt.Println(err)
				}
			}
		}
	}
}
func (server *P2PCenterServer)GetServerId()int32  {
	server.Lock()
	server.serverId++
	server.Unlock()
	return server.serverId
}
func (server *P2PCenterServer)GetClientId()int32{
	server.Lock()
	server.clientId++
	server.Unlock()
	return server.clientId
}
func (server *P2PCenterServer)DeleteClient(id int32){
	server.Lock()
	defer server.Unlock()
	delete(server.ClientConnMap, id)
	delete(server.ClientAddrMap, id)
	server.clientId--
}