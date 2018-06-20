package main

import (
	"network"
	"fmt"
)

type DefaultHandler struct {

}
func (handler DefaultHandler)MessageReceived(session network.SocketSessionInterface,message interface{}) error{
	return nil
}
func (handler DefaultHandler)MessageSent(session network.SocketSessionInterface,message interface{}) error{
	return nil
}
func (handler DefaultHandler)SessionOpened(session network.SocketSessionInterface) error{
	fmt.Println("kaishile !")
	return nil
}
func (handler DefaultHandler)SessionClosed(session network.SocketSessionInterface){

}
func (handler DefaultHandler)SessionPeriod(session network.SocketSessionInterface){
	fmt.Println("222")
}
func (handler DefaultHandler)ExceptionCaught(session network.SocketSessionInterface,err error){

}