## 一、代码注释与逻辑解读

先总体概括一下这段代码的功能：

- 在 `main.go` 中，我们创建了两个本地的传输节点 (`trLocal`、`trRemote`)，它们通过本地模拟的方式互相连接，然后 `trRemote` 会定时向 `trLocal` 发送消息。
- 我们通过 `network.NewServer(opts)` 初始化了一个服务器 (`Server`)，并在服务器中接收消息打印、以及定时做一些事情（每 5 秒一次）。
- 服务器里有一个通道 `rpcCh`，通过它来收集所有来自 `Transport` 层（本地或者将来实际网络）的消息，并进行处理（目前只是打印出来）。

下面我们分文件来解读。

### 1. `main.go`

```go
package main

import (
	"time"

	"github.com/DylanJinx/blockchain_1_0/network"
)

// main 函数是程序入口
func main() {
	// 1. 创建两个本地模拟的 Transport
	trLocal := network.NewLocalTransport("LOCAL")
	trRemote := network.NewLocalTransport("REMOTE")

	// 2. 两个 Transport 互相连接
	trLocal.Connect(trRemote)
	trRemote.Connect(trLocal)

	// 3. 启动一个协程，定时向 trLocal 发送消息
	go func() {
		for {
			trRemote.SendMessage(trLocal.Addr(), []byte("Hello, LOCAL"))
			time.Sleep(2 * time.Second)
		}
	}()

	// 4. 定义一个 ServerOpts，里面存了所有我们需要的传输层实例
	opts := network.ServerOpts{
		Transports: []network.Transport{trLocal},
	}

	// 5. 新建一个 Server，并启动
	s := network.NewServer(opts)
	s.Start()
}
```

> **流程概括**：
>
> - 创建了两个本地节点（Transport），并互相连通。
> - `trRemote` 每隔 2 秒给 `trLocal` 发送消息。
> - 将 `trLocal` 注册到 Server 中，服务器通过这个 Transport 来接收消息并打印。

这里的关键在于：

- `trRemote` 不在 `ServerOpts` 中，但它通过和 `trLocal` 连接后，一样可以向 `trLocal` 发送消息，然后 `trLocal` 会把消息往 `Server` 里送。
- 服务器启动后，会调用 `s.initTransports()` 去监听所有 `Transport` 的消息（即 `tr.Consume()`），然后做处理。

### 2. `server.go`

```go
package network

import (
	"fmt"
	"time"
)

// ServerOpts 用来配置 Server，比如需要哪些传输层
type ServerOpts struct {
	Transports []Transport
}

// Server 用来整合各种模块，比如区块链管理、交易管理、以及最重要的网络通信
type Server struct {
	ServerOpts

	rpcCh  chan RPC      // 收集所有 Transport 发来的消息
	quitCh chan struct{} // 用来通知关闭服务器的通道
}

// NewServer 创建一个 Server 并返回指针
func NewServer(opts ServerOpts) *Server {
	return &Server{
		ServerOpts: opts,
		rpcCh:      make(chan RPC),
		quitCh:     make(chan struct{}, 1),
	}
}

// Start 启动服务器，在这里面会有一个 select 监听多个通道
func (s *Server) Start() {
	// 1. 首先初始化传输层，让服务器监听每一个 Transport 的消息
	s.initTransports()

	// 2. 定义一个 5 秒间隔的定时器
	ticker := time.NewTicker(5 * time.Second)

free:
	for {
		select {
		case rpc := <-s.rpcCh:
			// 当从 rpcCh 通道获取到消息时，就把它打印出来
			fmt.Printf("%+v\n", rpc)
		case <-s.quitCh:
			// 收到退出信号，就退出循环
			break free
		case <-ticker.C:
			// 每 5 秒做一次额外的事情，这里只是打印一句话
			fmt.Println("do stuff every 5 seconds")
		}
	}
	fmt.Println("Server shutdown")
}

// initTransports 会把每个 Transport 的消息都转发到 s.rpcCh
func (s *Server) initTransports() {
	// 遍历所有 s.Transports
	for _, tr := range s.Transports {
		go func(tr Transport) {
			// range 住 tr.Consume() ，每读到一次消息，就放到 s.rpcCh
			for rpc := range tr.Consume() {
				s.rpcCh <- rpc
			}
		}(tr)
	}
}
```

> **关键点**：
>
> - `rpcCh` 是 `Server` 用来统一接收消息的管道。它相当于一个汇总点，收到了就可以打印，或者将来可以做别的处理。
> - `initTransports()` 里面开了多个 goroutine，每个 goroutine 负责把自己那个 `Transport` 收到的消息搬到 `Server.rpcCh`。
> - `ticker.C` 机制用于“隔一段时间做一次事情”。在区块链的真实场景中可能是定时打包区块、定时检查节点在线情况等等。

### 3. `transport.go`

```go
package network

// NetAddr 定义了网络地址，在这里用 string 表示
type NetAddr string

// RPC 表示一个远程调用/消息（“From” 是发送者，“Payload” 是内容）
type RPC struct {
	From    NetAddr
	Payload []byte
}

// Transport 接口：所有传输层要实现这些方法，Server 就可以统一管理
type Transport interface {
	Consume() <-chan RPC
	Connect(Transport) error
	SendMessage(NetAddr, []byte) error
	Addr() NetAddr
}
```

> **接口意义**：
>
> - 只要实现了 `Transport` 接口，服务器就可以把它当作一个“通信模块”来使用，不管底层是本地模拟还是 TCP、UDP，都不影响上层的逻辑。
> - `Consume()` 返回一个只读通道，让外部（Server）可以不断读取消息。

### 4. `local_transport.go`

```go
package network

import (
	"fmt"
	"sync"
)

// LocalTransport 用来模拟本地传输。（真实环境中可能用 TCP/UDP）
type LocalTransport struct {
	addr     NetAddr
	consumeCh chan RPC
	lock     sync.RWMutex
	peers    map[NetAddr]*LocalTransport
}

// NewLocalTransport 创建一个本地传输实例，并初始化它的各个字段
func NewLocalTransport(addr NetAddr) Transport {
	return &LocalTransport{
		addr:      addr,
		consumeCh: make(chan RPC, 1024),
		peers:     make(map[NetAddr]*LocalTransport),
	}
}

// Consume 返回只读通道，让外界可以读取到本节点收到的消息
func (t *LocalTransport) Consume() <-chan RPC {
	return t.consumeCh
}

// Connect 把自己和对方互相添加到各自的 peers 字典里
func (t *LocalTransport) Connect(tr Transport) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// 这里用类型断言把 Transport 转成 *LocalTransport
	t.peers[tr.Addr()] = tr.(*LocalTransport)
	return nil
}

// SendMessage 向指定地址发送消息。其实就是往对方的 consumeCh 放一条记录
func (t *LocalTransport) SendMessage(to NetAddr, payload []byte) error {
	t.lock.RLock()
	defer t.lock.RUnlock()

	peer, ok := t.peers[to]
	if !ok {
		return fmt.Errorf("%s: could not send message to %s", t.addr, to)
	}
	peer.consumeCh <- RPC{
		From:    t.addr,
		Payload: payload,
	}
	return nil
}

// Addr 返回当前本地节点的地址
func (t *LocalTransport) Addr() NetAddr {
	return t.addr
}
```

> **关键点**：
>
> - `Consume()` 是从 `consumeCh` 读消息；`SendMessage()` 是往对方 `consumeCh` 写消息。
> - `peers` 用于记录自己连接了哪些节点；在真实环境下，这个 peers 可能就是“活跃连接列表”，可以是 TCP 连接或者 websocket 连接。
> - 这里用 `sync.RWMutex` 来保证并发安全，因为发送消息和连接操作都有可能在不同的 goroutine 同时发生。

### 5. `local_transport_test.go`

```go
package network

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	tra := NewLocalTransport("A")
	trb := NewLocalTransport("B")

	tra.Connect(trb)
	trb.Connect(tra)
	assert.Equal(t, tra.(*LocalTransport).peers[trb.Addr()], trb.(*LocalTransport))
	assert.Equal(t, trb.(*LocalTransport).peers[tra.Addr()], tra.(*LocalTransport))
}

func TestSendMessage(t *testing.T) {
	tra := NewLocalTransport("A")
	trb := NewLocalTransport("B")

	tra.Connect(trb)
	trb.Connect(tra)

	msg := []byte("hello")
	assert.Nil(t, tra.SendMessage(trb.Addr(), msg))

	rpc := <-trb.Consume()
	assert.Equal(t, rpc.From, tra.Addr())
	assert.Equal(t, rpc.Payload, msg)
}
```

> **单元测试**：
>
> - 测试了 `Connect` 是否能成功连通。
> - 测试了 `SendMessage` 是否能把消息成功送到对方的 `Consume`。
> - 使用了 `testify/assert` 库的 `Equal` 和 `Nil` 来断言结果。

---

## 二、为什么代码里这样写？（核心原理）

1. **Server 只关心来自所有 Transport 的消息**  
   服务器本身并不想处理 Transport 的底层细节，它只想知道：

   - “谁发来的？”
   - “发的什么消息？”  
     于是就定义了一个 `rpcCh`（chan RPC）来接收所有消息。每当有消息进来（`<-s.rpcCh`），它就可以做一些处理或打印。

2. **Transport 负责消息的具体发送和接收**

   - `LocalTransport` 在这里仅做本地模拟，把要发的消息直接放进对方的 `consumeCh`。
   - 如果换成 TCP，你可能就要打开 socket，连接对方 IP/Port，然后使用 `net.Conn.Write()` 发送数据，再在对方节点的 `Conn.Read()` 中接收数据、放进一个 channel 里。
   - 这种设计思路让上层（Server）不用关心具体实现细节。

3. **Go 语言并发场景下的通道和锁**

   - `Consume()` 返回的是一个只读 channel，方便 Server 或其他组件直接 `for rpc := range tr.Consume()` 读取。
   - `RWMutex` 用来保证 `peers` map 的并发读写安全。
   - 当 `local_transport_test.go` 中互相连接、互相发送消息时，需要确保不会在并发读写 `peers` 结构时出现问题。

4. **协程与定时器**
   - 在 `main.go` 里，`trRemote` 每隔 2 秒向 `trLocal` 发一条消息，用了 `time.Sleep()` 模拟周期发送。
   - 在 `server.go` 里，用了 `time.NewTicker(5 * time.Second)` 模拟一个周期任务，这种逻辑对很多区块链项目来说很常见（比如定时打包区块、定时检查节点状态等）。

---

## 三、从“会看”到“会改”的思路

1. **先搞清楚各自职责**

   - `Server`：管理全局消息，处理区块链的逻辑（目前还没写）。
   - `Transport`：真正做网络/模拟网络通信的模块。
   - `rpcCh`：Server 与 Transport 之间的管道“桥梁”。

2. **如果要新增一个 TCPTransport**

   - 你可以新写一个结构体 `TCPTransport`，实现 `Transport` 接口的所有方法（`Consume()`, `Connect()`, `SendMessage()`, `Addr()`）。
   - 这样 `Server` 就可以直接用 `TCPTransport`，而不用改任何处理消息的逻辑。

3. **区块链真正的处理逻辑**
   - 目前 `Server` 里只是在 `select` 里打印消息或定时打印一句话。
   - 将来你会加入更多处理逻辑，比如收到交易后先验证签名、校验 UTXO；收到新区块后检查区块哈希和交易等。
   - 这些就是往 `Server`（或 `Node`）里不断扩展功能的过程。

---

## 四、总结

- **架构层面**：目前的代码主要是搭建了“传输层 + 服务器”基础框架。它通过抽象接口把网络通信封装起来，Server 只专注于如何“处理消息”，而不关心到底是用什么方式接收到的消息。
- **并发模型**：Go 语言里最常见的是通道（channel）+ goroutine。一个协程负责从 `Transport` 的 `Consume()` 通道里读数据，然后统一塞进 `Server.rpcCh`。这在实际生产环境也非常常见（消息队列、微服务、事件驱动）。
