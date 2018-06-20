package network

type EventHandleInterface interface {
	MessageReceived(session SocketSessionInterface,message interface{}) error
	MessageSent(session SocketSessionInterface,message interface{}) error
	SessionOpened(session SocketSessionInterface) error
	SessionClosed(session SocketSessionInterface)
	SessionPeriod(session SocketSessionInterface)
	ExceptionCaught(session SocketSessionInterface,err error)
}