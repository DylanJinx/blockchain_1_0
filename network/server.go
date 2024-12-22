package network

// ServerOpts 用来存储服务器配置的选项，比如传输层配置等等
type ServerOpts struct {
	Transports []Transport
}

// Server 是我们的服务端结构，可以在此汇总我们需要的各种模块
// 例如区块链管理、交易管理、网络通信模块等等。
type Server struct {

}

