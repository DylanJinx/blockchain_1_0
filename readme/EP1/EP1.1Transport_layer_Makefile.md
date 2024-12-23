# 1. Makefile
**什么是 Makefile？**

Makefile 是一个自动化构建工具，用于管理项目的编译、运行和测试等任务。通过定义一系列的规则和命令，Makefile 可以简化和自动化常见的开发流程。当你在命令行输入 `make` 时，它会根据 Makefile 中的指令执行相应的任务。

**Makefile 的基本结构**

一个典型的 Makefile 由多个“目标”（target）组成，每个目标都定义了要执行的命令。其基本语法如下：

```
目标: 依赖项
	命令
```

- **目标（Target）**：要构建或执行的任务名称，例如 `build`、`run`、`test`。
- **依赖项（Dependencies）**：目标所依赖的其他目标或文件。
- **命令（Commands）**：在目标下要执行的具体命令，这些命令通常以 Tab 键开头。

**Makefile 内容解析**

逐行分析 Makefile：

---

**1. build:**

```
build:
	go build -o ./bin/blockchain_1_0
```

- **目标**：`build`
- **命令**：`go build -o ./bin/blockchain_1_0`
- **解释**：
  - 这个目标用于编译你的 Go 代码。
  - `go build` 是 Go 的编译命令。
  - `-o ./bin/blockchain_1_0` 指定了输出的可执行文件的路径和名称，意味着编译后的程序将被保存为 `./bin/blockchain_1_0`。

---

**2. run: build**

```
run: build
	./bin/blockchain_1_0
```

- **目标**：`run`
- **依赖项**：`build`
- **命令**：`./bin/blockchain_1_0`
- **解释**：
  - 这个目标用于运行编译后的程序。
  - `run` 依赖于 `build`，这意味着在执行 `run` 之前，`make` 会先执行 `build` 目标，确保程序已被编译。
  - `./bin/blockchain_1_0` 是运行编译后的可执行文件的命令。

---

**3. test:**

```
test:
	go test -v ./...
```

- **目标**：`test`
- **命令**：`go test -v ./...`
- **解释**：
  - 这个目标用于运行项目的测试。
  - `go test` 是 Go 的测试命令。
  - `-v` 表示以详细模式运行测试，会输出测试的详细信息。
  - `./...` 指定了测试的包路径，`./...` 表示当前目录及其所有子目录。

---

**如何使用这个 Makefile？**

- **编译项目**：在终端中输入 `make build`，这将执行 `build` 目标，编译你的 Go 项目。
- **运行项目**：输入 `make run`，这将先编译项目（因为 `run` 依赖于 `build`），然后运行编译后的可执行文件。
- **运行测试**：输入 `make test`，这将执行项目中的所有测试。

