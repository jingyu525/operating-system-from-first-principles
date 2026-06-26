# 10 — 网络 IO：Socket & epoll

---

## 这一章回答什么问题？

> Redis 为什么能支持几十万并发连接？`epoll` 到底是什么？为什么 Go 的网络 IO 这么强？

---

## 第一性原理

```text
网络通信的本质：
一台机器的内存 ↔ 网卡 ↔ 网络 ↔ 网卡 ↔ 另一台机器的内存

操作系统需要解决的问题：
1. 怎么从网卡收数据？
2. 怎么高效地同时处理几万个连接？
```

> **Socket 是网络通信的抽象，epoll 是高效等待多个 Socket 事件的机制。**

---

## 推导过程

### Socket 的本质

```text
Socket = 文件描述符 + 网络协议栈

服务端创建 Socket：
socket()  → 创建一个"网络文件"
bind()    → 绑定 IP + 端口
listen()  → 标记为被动连接（监听）
accept()  → 接受连接 → 返回新的 fd
read()    → 读数据（和读文件一样！）
write()   → 写数据
```

### 从最差到最好：IO 模型的演进

```text
方案 1：BIO (Blocking IO) — 一个连接一个线程
┌──────────┐  ┌──────────┐  ┌──────────┐
│ Thread 1 │  │ Thread 2 │  │ Thread N │
│   accept │  │  read()  │  │  read()  │
│          │  │  (阻塞)   │  │  (阻塞)   │
└──────────┘  └──────────┘  └──────────┘

问题：10000 连接 = 10000 线程 → 内存爆炸 + 上下文切换灾难

方案 2：NIO (Non-blocking IO) — 一个线程轮询所有 fd
while (true) {
    for fd in all_fds {
        if fd is ready {  // 非阻塞检查
            read(fd)
        }
    }
}

问题：大多数 fd 没数据，白遍历 → CPU 浪费

方案 3：IO Multiplexing — select / poll
select(1000 fds) → 内核告诉你有哪几个就绪了
问题：每次调用都要传入完整的 fd 列表（拷贝开销）
      需要遍历所有 fd 才知道谁就绪（O(n)）

方案 4：epoll — 事件驱动 ✅ 现代标准答案
epoll_create()  → 创建 epoll 实例
epoll_ctl()     → 告诉 epoll："帮我监视这个 fd"
epoll_wait()    → "哪些 fd 有事件？"  ← 只返回就绪的 fd O(1)！
```

---

## epoll 的核心设计

```text
epoll 两个关键数据结构：

1. 红黑树：存储所有被监视的 fd
   插入/删除/修改：O(log N)

2. 就绪链表：存储所有就绪的 fd
   epoll_wait() 直接从链表取：O(1)

                            ┌─────────────┐
epoll_ctl(add fd) ────────→│  红黑树       │
                            │  (所有 fd)    │
                            └──────┬───────┘
                                   │ 数据到达
                                   ▼
                            ┌─────────────┐
epoll_wait() ←─────────────│  就绪链表     │
  (O(1) 返回就绪 fd)        │  (就绪的 fd)  │
                            └─────────────┘
```

### 两种触发模式

```text
LT (Level Triggered) — 水平触发（默认）
  fd 就绪，epoll_wait 每次都返回这个 fd，直到你读完数据
  简单但可能重复通知

ET (Edge Triggered) — 边缘触发
  fd 从"不就绪"变成"就绪"时才通知一次
  高效但要求一次读完所有数据（否则丢事件）
```

---

## 核心概念

| 概念 | 本质 |
|------|------|
| **Socket** | 网络通信的 fd，read/write 像操作文件 |
| **BIO** | 阻塞 IO，一个连接一个线程 |
| **NIO** | 非阻塞 IO，轮询所有 fd |
| **select/poll** | 内核帮你检查 fd 就绪状态 |
| **epoll** | 事件驱动，只返回就绪 fd，O(1) |
| **LT / ET** | 水平触发（默认）vs 边缘触发（高效） |
| **C10K 问题** | 如何在一台机器上处理 10000 个并发连接 |

---

## Linux 是怎么实现的？

```c
// epoll 使用示例 (C 语言)
int epfd = epoll_create(1);      // 创建 epoll 实例
struct epoll_event ev;
ev.events = EPOLLIN;             // 关心"可读"事件
ev.data.fd = server_fd;
epoll_ctl(epfd, EPOLL_CTL_ADD, server_fd, &ev); // 添加 fd

while (1) {
    struct epoll_event events[MAX_EVENTS];
    int n = epoll_wait(epfd, events, MAX_EVENTS, -1); // 等待事件
    for (int i = 0; i < n; i++) {
        if (events[i].data.fd == server_fd) {
            // 新连接
            int client = accept(server_fd, ...);
            // 把 client 也加入 epoll
        } else {
            // 已有连接上有数据可读
            read(events[i].data.fd, buf, sizeof(buf));
        }
    }
}
```

---

## Go 是怎么利用它的？

### Netpoller：Go 的异步网络 IO

```text
用户代码：
conn.Read(buf)           ← 看起来是同步的
  │
  ▼
Go Runtime netpoller：
  │
  ├── fd 就绪 → 直接读
  └── fd 未就绪 → 把 goroutine 挂起
                   │
              epoll_wait 等待
                   │
              fd 就绪 → 唤醒 goroutine
                   │
              conn.Read 返回数据
```

整个过程，goroutine 对用户来说是"阻塞"的，但对 Go Runtime 是异步的！

```text
传统 epoll 模型（回调地狱）：
epoll_wait → fd 就绪 → 回调函数 → 状态机 → 难写

Go Netpoller 模型：
conn.Read() → 看似同步 → 底层异步 → 代码简单
```

### Go Netpoller 架构

```text
┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
│  G1  │ │  G2  │ │  G3  │ │  G4  │  ← goroutine
└──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘
   │        │        │        │
   │ conn.Read() 看起来是同步的
   │        │        │        │
┌──┴────────┴────────┴────────┴──┐
│         Netpoller              │
│  (epoll_wait 在独立 OS Thread)  │
└───────────────┬────────────────┘
                │
   ┌────────────┼────────────┐
   ▼            ▼            ▼
┌──────┐    ┌──────┐    ┌──────┐
│  fd1  │    │  fd2  │    │  fdN  │  ← 套接字
└──────┘    └──────┘    └──────┘
```

---

## 常见面试题

**Q: select、poll、epoll 的区别？**

A:

| | select | poll | epoll |
|------|--------|------|-------|
| fd 上限 | 1024 (FD_SETSIZE) | 无限制 | 无限制 |
| 数据结构 | 位图 | 链表 | 红黑树+就绪链表 |
| 遍历方式 | O(n) 遍历所有 | O(n) 遍历所有 | O(1) 只返回就绪的 |
| 拷贝开销 | 每次传入全部 fd | 每次传入全部 fd | 只添加一次 |
| 适用场景 | 少量连接 | 少量连接 | 大量连接 |

**Q: epoll ET vs LT 的区别？**

A:
- LT (默认)：只要 fd 就绪就会一直通知，不会漏事件，但可能重复通知
- ET：只在状态变化时通知一次，效率高但必须读到 EAGAIN 才算读完

Go 的 netpoller 使用 ET 模式。

**Q: Redis 为什么单线程也能支持几十万连接？**

A:
- 使用 epoll 做 IO 多路复用
- 单线程处理请求（没有锁 + 没有上下文切换）
- 数据在内存中，操作极快
- 瓶颈在网卡带宽，不是 CPU

---

## 实战

```bash
# 查看系统网络连接
ss -tlnp           # 查看监听端口
ss -s              # 统计信息

# 查看 socket 统计
cat /proc/net/sockstat

# 查看连接状态分布
ss -ant | awk '{print $1}' | sort | uniq -c

# 调整系统参数
sysctl net.core.somaxconn        # 最大连接队列
sysctl net.ipv4.tcp_tw_reuse     # TIME_WAIT 复用
```

```go
// labs/epoll/main.go — 简单的 TCP Echo 服务器，展示 Go netpoller
package main

import (
    "fmt"
    "io"
    "log"
    "net"
)

func main() {
    listener, err := net.Listen("tcp", ":8080")
    if err != nil {
        log.Fatal(err)
    }
    defer listener.Close()

    fmt.Println("Echo 服务器运行在 :8080")

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Println(err)
            continue
        }

        // 每个连接一个 goroutine — 底层用的就是 epoll！
        go func(c net.Conn) {
            defer c.Close()
            // 这个 Read 看起来是阻塞的，但底层 netpoller 是异步的
            io.Copy(c, c)
        }(conn)
    }
}
```

---

## 总结

> epoll 通过事件驱动实现 O(1) 的就绪 fd 获取，Go 的 netpoller 把 epoll 异步本质隐藏在同步 API 之下，让高并发网络编程变得简单。

---

## 与后端开发的联系

```text
Go Netpoller  → 每个连接一个 goroutine，百万连接不是梦
              → 理解为什么 goroutine 泄露会导致网络连接异常

Redis    → 单线程 + epoll = 极简高效的 IO 模型
Nginx    → 多进程 + epoll = 高性能反向代理

epoll 本质 → 理解事件驱动架构 (Event Loop)
          → Node.js 的 libuv 也基于 epoll/kqueue/IOCP
          → 理解 Reactor / Proactor 模式
```
