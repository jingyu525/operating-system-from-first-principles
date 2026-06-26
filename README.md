# Operating System From First Principles

## 一句话介绍

> 从第一性原理理解操作系统，而不是死记硬背概念。

---

## 适合读者

- Go 后端工程师
- Java 后端工程师
- C++ 后端工程师
- Linux 学习者
- MySQL / Redis / Elasticsearch 使用者
- 分布式系统学习者

---

## 为什么写这份教程？

很多人学习操作系统，知道这些概念：

```
进程、线程、页表、虚拟内存、系统调用
```

**全都会。**

但是不知道：

**为什么要有这些东西。**

更不知道它们如何影响：

```
MySQL、Redis、Go Runtime、epoll、Kubernetes
```

本教程尝试：

> 从第一性原理重新推导整个操作系统，解释现代互联网系统为什么能够运行。

---

## 一页纸知识地图

```
                           Operating System
                                   │
          ┌────────────────────────┼────────────────────────┐
          │                        │                        │
       CPU 管理                 Memory 管理               IO 管理
          │                        │                        │
      Process                 Virtual Memory           File System
          │                        │                        │
      Thread                  Page Table               Driver
          │                        │                        │
   Scheduler               Physical Memory           Socket
          │                        │                        │
 Context Switch             mmap/brk                  epoll
                                   │
                             System Call
                                   │
                          User Mode / Kernel Mode
                                   │
                           Go Runtime / MySQL / Redis
```

---

## 学习路线

| 章节 | 标题 | 核心问题 |
|------|------|----------|
| 00 | [为什么需要操作系统](docs/00-为什么需要操作系统.md) | 没有 OS 的计算机是什么样子？ |
| 01 | [计算机是如何工作的](docs/01-计算机是如何工作的.md) | CPU、内存、总线到底是什么关系？ |
| 02 | [程序为什么能够运行](docs/02-程序为什么能够运行.md) | 从源码到进程，中间发生了什么？ |
| 03 | [进程 (Process)](docs/03-进程(Process).md) | 为什么每个程序都觉得自己独占 CPU？ |
| 04 | [线程 (Thread)](docs/04-线程(Thread).md) | 为什么需要比进程更轻的调度单位？ |
| 05 | [CPU 调度 (Scheduler)](docs/05-CPU调度(Scheduler).md) | 一个 CPU 怎么同时运行多个程序？ |
| 06 | [上下文切换 (Context Switch)](docs/06-上下文切换(Context%20Switch).md) | 切换进程的代价到底有多大？ |
| 07 | [虚拟内存 (Virtual Memory)](docs/07-虚拟内存(Virtual%20Memory).md) | 为什么每个程序都有 0x1000？ |
| 08 | [系统调用 (System Call)](docs/08-系统调用(System%20Call).md) | `os.Open()` 到底是怎么打开文件的？ |
| 09 | [文件系统 (File System)](docs/09-文件系统(File%20System).md) | 删除文件为什么几乎是瞬间完成的？ |
| 10 | [网络 IO — Socket & epoll](docs/10-网络IO(Socket%20&%20epoll).md) | Redis 怎么支持几十万并发连接？ |
| 11 | [Go Runtime 与操作系统](docs/11-Go%20Runtime%20与操作系统.md) | GMP 模型和 OS 的调度是什么关系？ |
| 12 | [性能分析](docs/12-性能分析.md) | `top` / `perf` / `strace` 背后的原理？ |

---

## 最后一章：一行 Go 代码的完整旅程

把全书知识串起来：

```
main()
  │
  ▼
Go Compiler
  │
  ▼
ELF 二进制
  │
  ▼
Linux Loader
  │
  ▼
Process → Thread → Scheduler → CPU
  │
  ▼
read() → System Call → Kernel
  │
  ▼
Socket → NIC → Network
  │
  ▼
Redis → MySQL → Disk
  │
  ▼
返回 → JSON → HTTP → Browser
```

---

## 每章统一结构

1. **这一章回答什么问题？**
2. **第一性原理** — 从最基础的物理/逻辑事实出发
3. **推导过程** — 画图展示为什么概念必然出现
4. **核心概念** — 不超过两页
5. **Linux 是怎么实现的？** — 不展开源码，只讲设计思路
6. **Go 是怎么利用它的？** — 联系 GMP、netpoll、GC 等
7. **常见面试题** — 用第一性原理回答
8. **实战** — Linux 命令观察
9. **总结** — 一句话
10. **与后端开发的联系**

---

## 项目定位

本仓库不是传统的《操作系统教程》，而是《**后端研发第一性原理**》系列的第一卷：

| 卷 | 主题 | 状态 |
|----|------|------|
| **第 1 卷** | **操作系统 (OS)** | 🚧 施工中 |
| 第 2 卷 | 计算机网络 (Network) | 📋 规划中 |
| 第 3 卷 | Go Runtime | 📋 规划中 |
| 第 4 卷 | MySQL | 📋 规划中 |
| 第 5 卷 | Redis | 📋 规划中 |
| 第 6 卷 | Elasticsearch | 📋 规划中 |
| 第 7 卷 | 分布式系统 | 📋 规划中 |
| 第 8 卷 | 高并发与高可用 | 📋 规划中 |

每一卷都采用相同的结构：**第一性原理 → 推导过程 → Linux/Go 实现 → 工程实践 → 与后端系统的联系**。

---

## 如何使用

1. **按顺序阅读**：每一章基于前面的知识
2. **做实验**：`labs/` 下有对应的动手实验
3. **结合自己的工作**：思考你写的每一行代码和操作系统的关系

---

## 贡献

欢迎 PR 和 Issue。如果你想参与其他卷的编写，请先开 Issue 讨论。

---

## License

[MIT](LICENSE)
