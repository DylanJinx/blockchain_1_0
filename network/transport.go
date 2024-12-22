package network

// NetAddr 定义了网络地址的类型，使用 string 来表示
type NetAddr string

// RPC是将消息通过transport传送在传输层上
type RPC struct {
	From	 NetAddr // 消息发送方地址
	Payload []byte // 消息内容，本质就是一段二进制数据
}

// Transport是server上的一个模块，服务器需要能够访问通过传输层发送的所有消息
// 一个完整的区块链系统必须支持多种 Transport 实现，比如 TCP、UDP 或者 本地模拟等
type Transport interface {
	// consume 返回一个只读的通道，用于接收其他节点发送过来的消息
	Consume() <-chan RPC

	// Connect 用于连接到其他 Transport 实例，从而实现节点之间的对等连接。 例如，如果我们使用 TCP 传输，那么这个方法将会建立一个 TCP 连接
	Connect(Transport) error
	
	// SendMessage 用于发送消息到指定地址的节点
	SendMessage(NetAddr, []byte) error

	// Addr 返回当前 Transport 实例的地址
	Addr() NetAddr
}