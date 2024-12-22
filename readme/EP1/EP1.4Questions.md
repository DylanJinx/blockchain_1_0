# `local_transport.go` 和 `local_transport_test.go`代码前后对比

`local_transport.go`:

```go
package network

import (
	"fmt"
	"sync"
)

type LocalTransport struct {
	addr 	  NetAddr
	consumeCh chan RPC
	lock 	  sync.RWMutex
	peers 	  map[NetAddr]*LocalTransport  // 传输将负责维护和连接到的对等方，我们需要一个映射来存储对等方的地址
}

func NewLocalTransport(addr NetAddr) *LocalTransport {
	return &LocalTransport{
		addr     : addr,
		consumeCh: make(chan RPC, 1024),
		peers    : make(map[NetAddr]*LocalTransport),
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
		From   : t.addr,
		Payload: payload,
	}

	return nil
}

func (t *LocalTransport) Addr() NetAddr {
	return t.addr
}
```

`local_transport_test.go`:

```go
package network

import (
	"testing"
	"github.com/stretchr/testify/assert" // go get github.com/stretchr/testify
)

func TestConnect(t *testing.T) {
	tra := NewLocalTransport("A")
	trb := NewLocalTransport("B")

	tra.Connect(trb)
	trb.Connect(tra)
	assert.Equal(t, tra.peers[trb.addr], trb)
	assert.Equal(t, trb.peers[tra.addr], tra)
}

func TestSendMessage(t *testing.T) {
	tra := NewLocalTransport("A")
	trb := NewLocalTransport("B")

	tra.Connect(trb)
	trb.Connect(tra)

	msg := []byte("hello")
	assert.Nil(t, tra.SendMessage(trb.addr, msg))

	rpc := <-trb.Consume()
	assert.Equal(t, rpc.From, tra.addr)
	assert.Equal(t, rpc.Payload, msg)
}
```

现在的代码：
`local_transport.go`:

```go
package network

import (
	"fmt"
	"sync"
)

type LocalTransport struct {
	addr 	  NetAddr
	consumeCh chan RPC
	lock 	  sync.RWMutex
	peers 	  map[NetAddr]*LocalTransport
}

func NewLocalTransport(addr NetAddr) Transport {
	return &LocalTransport{
		addr     : addr,
		consumeCh: make(chan RPC, 1024),
		peers    : make(map[NetAddr]*LocalTransport),
	}
}

func (t *LocalTransport) Consume() <-chan RPC {
	return t.consumeCh
}

func (t *LocalTransport) Connect(tr Transport) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.peers[tr.Addr()] = tr.(*LocalTransport)

	return nil
}

func (t *LocalTransport) SendMessage(to NetAddr, payload []byte) error {
	t.lock.RLock()
	defer t.lock.RUnlock()

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
```

`local_transport_test.go`:

```go
package network

import (
	"testing"

	"github.com/stretchr/testify/assert" // go get github.com/stretchr/testify
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

## 一、前后代码对比

### **之前的版本**

**`local_transport.go`**（主要不同点：`NewLocalTransport` 返回 `*LocalTransport`）

```go
func NewLocalTransport(addr NetAddr) *LocalTransport {
    return &LocalTransport{
        addr:      addr,
        consumeCh: make(chan RPC, 1024),
        peers:     make(map[NetAddr]*LocalTransport),
    }
}
```

### **现在的版本**

**`local_transport.go`**（主要不同点：`NewLocalTransport` 返回的是 `Transport` 接口）

```go
func NewLocalTransport(addr NetAddr) Transport {
    return &LocalTransport{
        addr:      addr,
        consumeCh: make(chan RPC, 1024),
        peers:     make(map[NetAddr]*LocalTransport),
    }
}
```

从表面上看，只是把函数返回值的类型从 `*LocalTransport` 改成了 `Transport` 接口。但这个改动带来了**后续的连锁影响**，特别是在测试文件 `local_transport_test.go` 里。

---

## 二、为什么要这样改？

### 1. 面向接口编程的需要

- **老版本**：`NewLocalTransport(addr) *LocalTransport`  
  返回的是一个具体的结构体指针。这意味着所有使用者都能直接拿到 `LocalTransport` 的实现细节，比如 `peers` 字段等。

- **新版本**：`NewLocalTransport(addr) Transport`  
  返回的是一个 `Transport` 接口。这是一种“更抽象、更通用”的写法，在 Go 里也很常见。它能让上层代码只关心“它是一个符合 `Transport` 接口的东西”，而不必依赖于其具体实现（`LocalTransport` 还是别的什么 Transport）。

**好处**：

- 将来如果你要扩展一个 `TCPTransport` 或者 `UDPTransport`，都能同样实现 `Transport` 接口，然后也能返回 `Transport`。上层只用 `Transport` 就够了，不用每次都回到 `*LocalTransport`。

### 2. 解耦、可扩展

- 现在上层代码（比如 `server.go` 或者某些初始化逻辑）只需要写：
  ```go
  tr := NewLocalTransport("A") // tr 是 Transport
  ```
  它就知道拿到一个 “Transport 能力” 的东西，能 `SendMessage()`、`Consume()`……至于底层如何实现，都可以隐藏起来。
- 这使得我们的程序更符合“依赖抽象不依赖具体实现”的设计原则。

---

## 三、为什么测试文件要做 `tr.(*LocalTransport)` 的类型断言？

**新版本**的测试里，你会看到：

```go
func TestConnect(t *testing.T) {
    tra := NewLocalTransport("A")
    trb := NewLocalTransport("B")

    tra.Connect(trb)
    trb.Connect(tra)

    // 这里要做类型断言，否则 tra.peers[trb.Addr()] 无法访问
    assert.Equal(t, tra.(*LocalTransport).peers[trb.Addr()], trb.(*LocalTransport))
    assert.Equal(t, trb.(*LocalTransport).peers[tra.Addr()], tra.(*LocalTransport))
}
```

原因在于：

- 现在 `tra` 的静态类型是 `Transport`（接口类型），其中并没有 `peers` 这个字段。只有实现了这个接口的“具体类型” `*LocalTransport` 才有 `peers`。
- 当你想在测试中访问 `tra.peers[...]` 时，就必须先把它“转回”具体类型 `*LocalTransport` 才行。
- Go 语言里，`接口类型` → `具体类型指针` 需要使用 `类型断言`（`.(type)` 或者 `.( *LocalTransport )`）的写法。
- 断言写法： `tra.(*LocalTransport)` 的意思是：“请把接口变量 `tra` 转换回 `*LocalTransport`，否则我就没法访问它特有的字段 `peers`”。

### **为什么之前不用类型断言？**

- **老版本**：`tra` 就是 `*LocalTransport`，直接 `tra.peers[...]` 就能访问。
- **新版本**：`tra` 是 `Transport`，要先 `(tra.(*LocalTransport))` 才能得到 `peers`。

## `tr.addr` -> `tr.Addr()`

因为现在的`tra`和`trb` 是 `Transport`类型，这个变量就只能访问接口中定义的方法，不能直接访问 `LocalTransport` 的私有字段（如 `addr`），所以要用 Addr()方法来调用。

```go
func (t *LocalTransport) Addr() NetAddr {
	return t.addr
}
```

调用 `tr.Addr()` 并不是真正地“把 `Transport` 转成了 `LocalTransport`”，而是“通过接口的动态调度，调用了它背后实际的 `LocalTransport` 对象的方法”。

- 在 Go 中，接口变量（`tr`）包含**两个信息**：一是“具体值”（底层实际类型，比如 `LocalTransport`），二是“具体类型描述”。当你对接口调用某个方法时，会在运行时根据“具体类型”找到对应的方法实现，并执行它。
- 这跟手动做类型断言（`tr.(*LocalTransport)`) 不一样。后者是真的在代码层面显式地“把接口变量转回到具体类型”之后，你才能访问到 `LocalTransport` 特有的字段（比如 `peers`）。
- 而 `tr.Addr()` 之所以能工作，是因为 `Transport` 接口中声明了 `Addr()` 方法，所以 Go 编译器知道：“只要满足 `Transport` 接口的具体类型，都一定有 `Addr()` 这个方法。” 这个调用是通过 **接口的动态分发** 完成的，而不是把 `Transport` “改成” `LocalTransport`。

# `range channel`为什么可以实现

在 Go 语言中，`range` 可以用来遍历「可迭代」的对象，比如数组、切片、映射（map），**也**可以用来从「通道（channel）」中**持续读取**元素，直到通道被关闭或循环被打断。

---

## 为什么对通道也能用 `range`？

1. **通道本质上是一种“队列”**

   - 当你往一个通道里不断 `send`（比如 `ch <- data`），通道就相当于在排队存储那些发送进来的数据。
   - 当你 `range ch` 时，就是在“遍历”这条队列，每次从队列取出一个值来赋给循环变量，直到通道被关闭或没有新数据了。

2. **只读通道 vs. 可读可写通道**

   - 在语法上，`chan T` 可以是双向通道（可读可写），也可以是 `<-chan T`（只读）或 `chan<- T`（只写）。
   - 这里的“只读”只是限制了**当前作用域**里**你只能读，不能写**。但是对于 `range ch` 来说，只要能读就行，它会持续地从该通道里**接收**消息。
   - “只读”并不代表这个通道本身没人往里写。你可能在另一个 goroutine 里向通道里写数据。

3. **`for ... range ch` 的工作机制**

- 当我们写 for rpc := range ch { ... }，本质是 Go 帮我们做了不断地 rpc, ok := <-ch 的工作。
- 如果通道 ch 没有数据可读，但是也没被关闭，rpc := <-ch 会阻塞等待。
- 只要有其他 goroutine 向通道写入数据（ch <- something），阻塞的读取操作就会被唤醒，获得数据后继续下一次循环。
- 如果通道被关闭（close(ch)），那么下一次读取时 ok 就会为 false，循环也就结束了。

---

## 简单示例

一个常见的例子：

```go
func producer(ch chan<- int) {
    for i := 0; i < 5; i++ {
        ch <- i
    }
    close(ch) // 生产者结束后关闭通道
}

func main() {
    ch := make(chan int)
    go producer(ch)

    // 消费者用 range 读所有数据，直到通道关闭
    for val := range ch {
        fmt.Println("Received:", val)
    }
}
```

- `producer` 不断往通道里发送数据，然后 `close(ch)`。
- `main` 中 `for val := range ch` 就可以一条一条地把数据取出来。

# 多个传输层写法：`Transport: []network.Transport{trLocal, tcpTransport, ...}`

**简短回答**：  
完全可以这么写，只要你的 `tcpTransport` 也实现了 `Transport` 接口，就可以和 `trLocal` 一起被放进同一个切片 `[]network.Transport{...}` 里。然后 `Server` 会把它们都初始化、都消费消息，实现多种传输方式的并存。

---

## 为什么可以这样做？

1. **`Transports` 是一个切片 (slice)**  
   在这段代码里：

   ```go
   opts := network.ServerOpts{
       Transports: []network.Transport{trLocal},
   }
   ```

   只是示例写法，表示：**我们给 `ServerOpts` 赋值的 `Transports` 字段，是一个 `[]network.Transport` 切片，里面只有 `trLocal` 这一个元素**。

2. **可以往切片里添加多个实现同一接口的元素**

   - 如果你实现了另一个传输层，比如 `tcpTransport`，并且它同样满足 `Transport` 接口定义的方法：
     ```go
     type Transport interface {
         Consume() <-chan RPC
         Connect(Transport) error
         SendMessage(NetAddr, []byte) error
         Addr() NetAddr
     }
     ```
     那么它和 `trLocal` 在类型上是“同一级别”的东西——都属于 `Transport` 接口的实现。
   - 于是你可以把它们都放到 `Transports` 切片里，比如：
     ```go
     opts := network.ServerOpts{
         Transports: []network.Transport{trLocal, tcpTransport},
     }
     ```

3. **`Server` 会同时管理这些 Transport**  
   在 `server.go` 中，`initTransports()` 会遍历这个切片里的所有 Transport，用 goroutine 去 “range tr.Consume()”，把消息发送到统一的 `rpcCh`。也就是说，**无论消息来自哪个 Transport**，最终都能被 `Server` 的 `select { case rpc := <-s.rpcCh: ... }` 读到。

---

## 代码示例（伪代码）

假设你真的实现了一个新的 `TCPTransport`，像下面这样（伪代码）：

```go
type TCPTransport struct {
    // 一些字段，比如 listener, connections, etc.
    // ...
}

func NewTCPTransport(addr NetAddr) Transport {
    return &TCPTransport{ /* ... */ }
}

func (t *TCPTransport) Consume() <-chan RPC {
    // 返回一个只读通道
    // ...
}

func (t *TCPTransport) Connect(other Transport) error {
    // 真正建立 TCP 连接的逻辑
    // ...
}

func (t *TCPTransport) SendMessage(to NetAddr, payload []byte) error {
    // 通过 TCP 连接把消息发出去
    // ...
}

func (t *TCPTransport) Addr() NetAddr {
    // 返回自己的地址
    // ...
}
```

然后在你的 `main.go` 里，你完全可以写成：

```go
func main() {
    trLocal := network.NewLocalTransport("LOCAL")
    tcpTransport := network.NewTCPTransport("127.0.0.1:9000")

    opts := network.ServerOpts{
        // 同时加本地 Transport 和 TCP Transport
        Transports: []network.Transport{trLocal, tcpTransport},
    }
    s := network.NewServer(opts)
    s.Start()
}
```

这样 `Server` 就能处理来自两个不同传输层的消息了。

---

## 总结

- “在上面代码中只写了 `Transports: []network.Transport{trLocal}`，是不是就不能加别的？”  
  **并不是**。这只是示例。如果你还有别的满足 `Transport` 接口的实现，都可以往这个切片里放。
- Go 的多态是通过接口来实现的。只要实现了相同的接口，就能被视作同一种“类型”在高层使用（这里指的就是 `Transport`）。
- 服务器 (`Server`) 会遍历整个 `Transports` 切片，逐一启动监听，因此**可以在同一个节点中使用多个不同的传输层**。

**所以**，你所说的「我们是不是就不能再这样写了」其实恰恰相反，**我们正是可以这样做**。只要多种传输都实现了 `Transport` 接口，就能通过这个切片一并交给 `Server` 来使用。
