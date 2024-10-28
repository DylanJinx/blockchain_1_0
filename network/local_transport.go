package network

import (
	"fmt"
	"sync"
)

type LocalTransport struct {
	addr NetAddr
	consumeCh chan RPC
	lock sync.RWMutex
	peers map[NetAddr]*LocalTransport // 传输将负责维护和连接到的对等方，我们需要一个映射来存储对等方的地址
}

func NewLocalTransport(addr NetAddr) *LocalTransport {
	return &LocalTransport{
		addr: addr,
		consumeCh: make(chan RPC, 1024),
		peers: make(map[NetAddr]*LocalTransport),
	}
}

func (t *LocalTransport) Consume() <-chan RPC {
	return t.consumeCh
}

func (t *LocalTransport) Connect(tr *LocalTransport) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.peers[tr.Addr()] = tr

	return nil // 这里我们不需要做任何事情，因为我们只是在本地传输中连接到对等方
}

func (t *LocalTransport) SendMessage(to NetAddr, payload []byte) error {
	t.lock.RLock() // 读锁
	defer t.lock.RUnlock() // 释放读锁

	peer, ok := t.peers[to]
	if !ok {
		return fmt.Errorf("%s: could not send message to %s", t.addr, to)
	}

	peer.consumeCh <- RPC{
		From: t.addr,
		Payload: payload,
	}

	return nil
}

func (t *LocalTransport) Addr() NetAddr {
	return t.addr
}