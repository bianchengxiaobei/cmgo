package network

type Protocol interface {
	Encode(SocketSessionInterface,[]byte)(interface{},int,error)
	Decode(SocketConnectInterface,interface{}) error
}
