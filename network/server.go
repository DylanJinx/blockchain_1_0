package network

import (
	"fmt"
	"time"
)

// ServerOpts 用来存储服务器配置的选项，比如传输层配置等等
type ServerOpts struct {
	Transports []Transport
}

// Server 是我们的服务端结构，可以在此汇总我们需要的各种模块
// 例如区块链管理、交易管理、网络通信模块等等。
type Server struct {
	ServerOpts

	rpcCh  chan RPC       // 定义了一个 RPC 通道，用于接收其他节点发送过来的消息（消息类型为 RPC）
	quitCh chan struct{}  // 定义了一个退出通道，用于退出服务器
}

func NewServer(opts ServerOpts) *Server {
	return &Server{
		ServerOpts: opts,
		rpcCh     : make(chan RPC),
		quitCh    : make(chan struct{}, 1), // struct{} 是空结构体，占用 0 字节，用于通道的信号通知
	}
}

func (s *Server) Start() {
	s.initTransports()
	ticker := time.NewTicker(5 * time.Second) // 定义一个定时器，每 5 秒触发一次

free: // free 是一个标签，用于 break 退出 for 循环
	for {
		select {
		case rpc := <-s.rpcCh: // 从 rpcCh 通道中读取到消息
			fmt.Printf("%+v\n", rpc) // +v 格式化输出结构体
		case <-s.quitCh:
			break free
		case <-ticker.C: // C 是定时器的通道，每次定时器触发，都会从这个通道中读取到一个值
			fmt.Println("do stuff every 5 seconds")
		}
	}
	fmt.Println("Server shutdown")
}

func (s *Server) initTransports() {
	for _, tr := range s.Transports { //为什么不是s.ServerOpts.Transports? 因为ServerOpts是Server的一个属性，所以可以直接访问
		go func(tr Transport) {
			for rpc := range tr.Consume() {
				// handle
				s.rpcCh <- rpc
			}
		}(tr)
	}
}