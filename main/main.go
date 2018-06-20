package main

import (
	"network"
)
func main(){
	serverConfig := network.SocketSessionConfig{
		TcpNoDelay:true,
		TcpKeepAlive:true,
		TcpReadBuffSize:1024,
		TcpWriteBuffSize:1024,
		ReadChanLen:1,
		WriteChanLen:1,
	}
	var serverHandler network.EventHandleInterface
	serverHandler = new(DefaultHandler)
	var clientHandler network.EventHandleInterface
	clientHandler = new(DefaultHandler)
	server := network.NewTcpServer("tcp4",&serverConfig)
	server.SetMessageHandler(serverHandler)
	server.Bind(":8080")
	client := network.NewTcpClient("tcp",&serverConfig)
	client.SetMessageHandler(clientHandler)
	client.Connect("127.0.0.1:8080")
	network.WaitSignal()
	server.Close()
	client.Close()
}