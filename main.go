package main

import (
	"time"

	"github.com/DylanJinx/blockchain_1_0/network"
)

// Server
// Transport => tcp, udp
// Block
// TX
// Keypair

// 后面我们会在这里初始化节点、服务器，以及后续的区块链相关组件。
func main() {
	trLocal := network.NewLocalTransport("LOCAL")
	trRemote := network.NewLocalTransport("REMOTE")

	trLocal.Connect(trRemote)
	trRemote.Connect(trLocal)

	go func() {
		for {
			trRemote.SendMessage(trLocal.Addr(), []byte("Hello, LOCAL"))
			time.Sleep(2 * time.Second)
		}
	}()

	opts := network.ServerOpts{
		Transports: []network.Transport{trLocal},
	}

	s := network.NewServer(opts)
	s.Start()
}