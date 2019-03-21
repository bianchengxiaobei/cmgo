package network

import (
	"fmt"
	"net"
	"strconv"
	"sync"
)

type P2PCenterServer struct {
	readBuffer    []byte
	serverId      int
	clientId      int
	Conn          *net.UDPConn
	ServerConn    *net.UDPAddr
	chanel        chan int
	//serverAddrChannel   chan string
	ServerAddrMap map[int]*net.UDPAddr
	ClientAddrMap map[int]*net.UDPAddr
	lock          sync.Mutex
}

func NewP2PCenterServer() *P2PCenterServer {
	server := &P2PCenterServer{
		readBuffer:    make([]byte, 512),
		serverId:      0,
		clientId:      0,
		chanel:        make(chan int, 1),
		//serverAddrChannel:make(chan string,100),
		ServerAddrMap: make(map[int]*net.UDPAddr),
		ClientAddrMap: make(map[int]*net.UDPAddr),
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
			content := string(server.readBuffer[:len])
			if content == "Server" {
				server.ServerConn = tempAddr
			}else if content == "DSClient" {
				id := server.GetClientId()
				fmt.Println("客户端DS远程地址:" + tempAddr.String())
				port := tempAddr.Port
				port += 1;
				tempAddr.Port = port
				server.ClientAddrMap[id] = tempAddr
				clientAddrAndId := fmt.Sprintf("%s;%d", tempAddr.String(), id)
				server.Conn.WriteToUDP([]byte(clientAddrAndId), server.ServerConn)
				server.chanel<-id
				//开始转发试试
			}else if content == "WSClient" {
				id := server.GetClientId()
				fmt.Println("客户端WS远程地址:" + tempAddr.String())
				port := tempAddr.Port
				port += 1;
				tempAddr.Port = port
				server.ClientAddrMap[id] = tempAddr
				clientAddrAndId := fmt.Sprintf("%s;%d", tempAddr.String(), id)
				server.Conn.WriteToUDP([]byte(clientAddrAndId), server.ServerConn)
				server.chanel<-id
			}else if content == "Client"{
				id := server.GetClientId()
				fmt.Println("客户端远程地址:" + tempAddr.String())
				server.ClientAddrMap[id] = tempAddr
				clientAddrAndId := fmt.Sprintf("%s;%d", tempAddr.String(), id)
				server.Conn.WriteToUDP([]byte(clientAddrAndId), server.ServerConn)
				server.chanel<-id
			}else if content == "P" {
				continue
			} else {
				go server.HandlerConn(tempAddr, content)
			}
		}
	}
}
func (server *P2PCenterServer) HandlerConn(udpAddr *net.UDPAddr, content string) {
	if content == "ServerClient" {
		id := server.GetServerId()
		fmt.Printf("服务器[%d]远程地址:[%s]", id, udpAddr.String())
		fmt.Println()
		server.ServerAddrMap[id] = udpAddr
		server.chanel <- id
	} else {
		//服务器发送的客户端id，请求客户端开始向他发送验证打洞数据
		id, err := strconv.Atoi(content)
		if err == nil {
			con := server.ClientAddrMap[id]
			if con != nil {
				id2 := <-server.chanel
				if id2 != id {
					fmt.Println(id2)
				}
				serverAddr := server.ServerAddrMap[id]
				fmt.Printf("发送给客户端[%d]服务器的地址:[%s]", id, serverAddr.String())
				_, err := server.Conn.WriteToUDP([]byte(serverAddr.String()), con)
				if err != nil {
					fmt.Println(err)
				}
				server.DeleteClient(id)
			}
		} else {
			fmt.Println(content)
			fmt.Println(err)
		}
	}
}
func (server *P2PCenterServer) GetServerId() int {
	server.lock.Lock()
	server.serverId++
	server.lock.Unlock()
	return server.serverId
}
func (server *P2PCenterServer) GetClientId() int {
	server.lock.Lock()
	server.clientId++
	server.lock.Unlock()
	return server.clientId
}
func (server *P2PCenterServer) DeleteClient(id int) {
	server.lock.Lock()
	defer server.lock.Unlock()
	delete(server.ClientAddrMap, id)
	delete(server.ServerAddrMap, id)
}
