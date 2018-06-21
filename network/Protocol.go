package network

type Protocol interface {
	Decode(SocketSessionInterface,[]byte)(interface{},int,error)
	Encode(SocketSessionInterface,interface{}) error
}
