package network

type NetAddr string

// RPC是将消息通过transport传送在传输层上
type RPC struct {
	From NetAddr
	Payload []byte
}

// Transport是server上的一个模块，服务器需要能够访问通过传输层发送的所有消息
type Transport interface {
	Consume() <-chan RPC
	Connect(Transport) error
	SendMessage(NetAddr, []byte) error
	Addr() NetAddr
}