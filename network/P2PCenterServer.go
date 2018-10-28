package network

import (
	"fmt"
	"net"
	"strconv"
	"sync"
)

type P2PCenterServer struct {
	readBuffer        []byte
	serverId          int32
	clientId          int32
	Conn              *net.UDPConn
	ServerAddrByteMap map[int32][]byte
	ServerAddrMap     map[int32]*net.UDPAddr
	ClientAddrMap     map[int32]*net.UDPAddr
	lock              sync.Mutex
}

func NewP2PCenterServer() *P2PCenterServer {
	server := &P2PCenterServer{
		readBuffer:        make([]byte, 512),
		serverId:          0,
		clientId:          0,
		ServerAddrByteMap: make(map[int32][]byte),
		ServerAddrMap:     make(map[int32]*net.UDPAddr),
		ClientAddrMap:     make(map[int32]*net.UDPAddr),
	}
	return server
}
func (server *P2PCenterServer) Bind(addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	server.Conn = conn
	fmt.Println("服务器开始运行")
	for {
		len, tempAddr, err := conn.ReadFromUDP(server.readBuffer)
		if err != nil {
			fmt.Println(err)
		}
		if len > 0 {
			//读取消息
			server.HandlerConn(tempAddr, len)
		}
	}

}
func (server *P2PCenterServer) HandlerConn(udpAddr *net.UDPAddr, len int) {
	if len > 0 {
		content := string(server.readBuffer[:len])
		if content == "Server" {
			id := server.GetServerId()
			fmt.Println("服务器远程地址:" + udpAddr.String())
			server.ServerAddrMap[id] = udpAddr
			server.ServerAddrByteMap[id] = []byte(udpAddr.String())
		} else if content == "Client" {
			id := server.GetClientId()
			fmt.Println("客户端远程地址:" + udpAddr.String())
			server.ClientAddrMap[id] = udpAddr
			clientAddrAndId := fmt.Sprintf("%s;%d", udpAddr.String(), id)
			server.Conn.WriteTo([]byte(clientAddrAndId), server.ServerAddrMap[1]) //这里1写死，本来是动态负载均衡网关id
		} else {
			//服务器发送的客户端id，请求客户端开始向他发送验证打洞数据
			id, err := strconv.Atoi(content)
			if err == nil {
				id3 := int32(id)
				con := server.ClientAddrMap[id3]
				if con != nil {
					len, err = server.Conn.WriteTo(server.ServerAddrByteMap[1], con)
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
func (server *P2PCenterServer) GetServerId() int32 {
	server.lock.Lock()
	server.serverId++
	server.lock.Unlock()
	return server.serverId
}
func (server *P2PCenterServer) GetClientId() int32 {
	server.lock.Lock()
	server.clientId++
	server.lock.Unlock()
	return server.clientId
}
func (server *P2PCenterServer) DeleteClient(id int32) {
	server.lock.Lock()
	defer server.lock.Unlock()
	delete(server.ClientAddrMap, id)
	server.clientId--
}
