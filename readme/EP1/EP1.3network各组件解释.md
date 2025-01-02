### 1. 代码里的 `Transport`, `Server`, `ServerOpts` 都是什么，它们之间关系如何？

1. **Transport**：

   - 负责节点之间通信的“底层”。它抽象了一个通用接口，里面有 `Connect`, `SendMessage`, `Consume`, `Addr` 等方法。
   - 在区块链或者任意分布式系统中，各节点需要能互相通信才行。`Transport` 就是抽象出“如何给某个地址发送消息，以及如何从其他节点收到消息”。
   - 例子：`LocalTransport` 是在本地用一个 map 和 channel 来模拟节点之间的连接和消息传递；将来可能会有 `TCPTransport` 或别的实现。

2. **Server**：

   - 你可以把它理解成“区块链节点”的主要逻辑容器。它里面持有了 `rpcCh`（一个消息总线），以及一堆 Transport。
   - 当其他节点给这个节点发消息时，Transport 会把消息投递给 `Server` 的通道（`rpcCh`），`Server` 就能处理这些消息（比如打印、验证交易、打包区块等等）。
   - 同时，`Server` 也可以定时做一些事情，比如启动共识算法、广播最新区块等。

3. **ServerOpts**（Options）：
   - “Opts” 一般是 “Options” 的缩写，指“配置项”或者“选项”。
   - 这里是用来传入构造 `Server` 的一些必要参数——例如要用哪些 Transport、后续可能加上别的配置项，比如节点名称、端口、日志级别等。
   - 为什么不用直接把 Transports 传进 `NewServer`？当然可以，但一般在 Go 中，我们喜欢用一个 `opts` 结构体来收集和管理配置参数。这样当配置项变多时，代码结构会更有条理。

### 2. 在区块链中扮演什么角色？

- **Transport**：节点间的网络层。
- **Server**：节点自身的核心服务器，可能管理区块、交易、共识、P2P 通讯等——它是一个“主程序”或“节点程序”。
- **RPC**：消息传递的格式。区块链节点之间需要传交易或区块，这都属于“消息”，所以可以定义成 `RPC`，本质上是“From + Payload”。

用一句话概括：

> **`Server`** 用 **`Transport`** 来与其他节点通信，将消息收集到自己内部的通道 `rpcCh`，并在内部做具体的区块链逻辑处理。

### 3. 当前这个代码的架构为什么要设计成这样？

- **面向接口编程**：让 `Server` 不关心 Transport 的实现细节，只关心它能否正常“发送消息”和“接收消息”。将来可以替换或新增不同的通信方式。
- **解耦**：区块链本身处理逻辑（验证交易、打包区块、维护状态机）和网络传输逻辑分离。出现问题时更易定位；想要换种方式传输也不会破坏整体结构。
- **并发模型**：Go 语言下，使用 channel+goroutine 是非常常见的。每个 Transport 用一个 goroutine，死循环等消息，收到后就投递给 `Server` 的 `rpcCh`。

### 4. 文件如何相互穿插、流程是什么？

1. **`main.go`**
   - 创建 Transport → 互相连接 → 启动一个协程定时发送消息 → 创建 `ServerOpts` → `NewServer(opts)` → `s.Start()`
2. **`server.go`**
   - `NewServer` 构造一个 `Server`，持有了 `rpcCh` 和要用的 Transport 列表。
   - `s.Start()` 时先 `initTransports()`：给每个 Transport 启一个 goroutine，负责把 Transport 收到的消息塞到 `s.rpcCh`。
   - 之后 `select {}` 监听 `s.rpcCh`、定时器、或者退出通道，当 `rpcCh` 收到消息就打印/处理。
3. **`local_transport.go`**
   - 本地模拟的实现：提供 `Connect(Transport)`, `SendMessage(...)`, `Consume()`, `Addr()` 的具体代码。
   - `Consume()` 返回一个 channel，Server 在这里面“range”就可以获取消息。
   - `SendMessage()` 往对方的 `consumeCh` 里塞消息。
4. **`transport.go`**
   - 定义 Transport 接口和 RPC 结构，让 “任何类型的 Transport” 都必须提供 `Connect`, `SendMessage`, `Consume`, `Addr`。
   - `RPC` 代表了一个消息对象，里面有 `From + Payload`。

**从代码调用顺序上看**：

- `main.go` 里：`trRemote.SendMessage(trLocal.Addr(), [])` →  
  `local_transport.go` 里：找到了 `trLocal` 的 peers，然后写消息到 `trLocal.consumeCh` →  
  `server.go` 里：在 `initTransports()` 中有个协程 `for rpc := range tr.Consume() { s.rpcCh <- rpc }`，因此 `trLocal` 的 consumeCh 有新消息时，`s.rpcCh` 会收到 →  
  `server.go` 里的 `select { case rpc := <-s.rpcCh: ... }` 收到后打印。
