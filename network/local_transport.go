package network

import (
	"fmt"
	"sync"
)

// LocalTransport 用来模拟本地传输。
// 在真实的区块链系统中，这里会用 TCP 或 UDP 的 socket 来进行网络通信。
// 在本地测试或单机模拟时，我们用 LocalTransport 来模拟多个节点之间的网络通信。
type LocalTransport struct {
	addr 	  NetAddr  // 当前节点的地址
	consumeCh chan RPC // 传入消息队列，其他节点发来的消息会被放到这个队列里
	lock 	  sync.RWMutex
	peers 	  map[NetAddr]*LocalTransport  // 记录和哪些节点建立了连接
}

func NewLocalTransport(addr NetAddr) *LocalTransport {
	return &LocalTransport{
		addr     : addr,
		consumeCh: make(chan RPC, 1024),
		peers    : make(map[NetAddr]*LocalTransport),
	}
}

// Consume 返回一个只读的通道，让外部可以读取到本节点收到的消息。
func (t *LocalTransport) Consume() <-chan RPC {
	return t.consumeCh
}

// Connect 连接到另一个 LocalTransport。
func (t *LocalTransport) Connect(tr *LocalTransport) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.peers[tr.Addr()] = tr

	return nil // 这里我们不需要做任何事情，因为我们只是在本地传输中连接到对等方
}

// SendMessage 用来向指定地址（to）发送消息（payload）。
// 如果 peers 中没有这个地址，就说明没有连接到对方，返回错误。
// 如果找到，则把消息放入对方的 consumeCh，让对方读取。
func (t *LocalTransport) SendMessage(to NetAddr, payload []byte) error {
	t.lock.RLock() // 使用读锁，允许并发读但阻塞写
	defer t.lock.RUnlock() // 释放读锁

	peer, ok := t.peers[to]
	if !ok {
		return fmt.Errorf("%s: could not send message to %s", t.addr, to)
	}

	peer.consumeCh <- RPC{
		From   : t.addr,
		Payload: payload,
	}

	return nil
}

func (t *LocalTransport) Addr() NetAddr {
	return t.addr
}