# 11 — Go Runtime 与操作系统

---

## 这一章回答什么问题？

> GMP 模型和 OS 的调度/内存/IO 是什么关系？Go Runtime 到底在 OS 之上做了什么？

---

## 第一性原理

```text
OS 提供的：
- 进程 / 线程
- 虚拟内存
- 系统调用 (read, write, mmap, futex...)
- 调度器 (CFS)
- 网络栈 (epoll)

Go Runtime 在 OS 之上构建：
- goroutine (用户态"线程")
- GMP 调度器
- 自动内存管理 (GC)
- Netpoller
- Channel 等并发原语
```

> **Go Runtime 本质是在 OS 之上的一层"用户态操作系统"。**

---

## 推导过程

### 为什么 Go 需要 Runtime？

```text
OS Thread 的问题：
1. 创建/销毁需要系统调用 → 慢
2. 栈固定 ~8MB → 浪费内存
3. 调度器在内核态 → 切换开销大
4. 没有内置的并发通信机制

Go 的解决方案：
Runtime 在用户态管理这一切：
- 用户态调度 → goroutine 切换 ~200ns
- 动态栈 → 初始 2KB，按需增长
- 内置 Channel → CSP 并发模型
```

### Go 程序的完整启动链路

```text
go build main.go
  ↓
ELF 静态二进制
  ↓
./main
  ↓
Linux Loader 加载 ELF
  ↓
Go Runtime 入口 (_rt0_amd64_linux)
  ↓
runtime.args()       — 解析命令行
runtime.osinit()     — CPU 核数、页大小等
runtime.schedinit()  — 初始化调度器
runtime.newproc()    — 创建 main goroutine
runtime.mstart()     — 启动调度
  ↓
Goroutine 执行 main.main()
```

---

## GMP 模型的 OS 视角

```text
┌────────────────────────────────────────────┐
│              Go User Space                 │
│                                            │
│   ┌───┐ ┌───┐ ┌───┐    ┌───┐              │
│   │ G │ │ G │ │ G │... │ G │  goroutines   │
│   └─┬─┘ └─┬─┘ └─┬─┘    └─┬─┘              │
│      │     │     │        │                │
│   ┌──┴─────┴─────┴────────┴──┐             │
│   │      P (逻辑处理器)       │             │
│   │   本地 G 队列 (runq)      │             │
│   └─────────────┬────────────┘             │
│                 │                          │
│   ┌─────────────┴────────────┐             │
│   │   Netpoller (epoll)     │             │
│   └─────────────────────────┘             │
│                 │                          │
└─────────────────┼──────────────────────────┘
                  │
 ─ ─ ─ ─ ─ ─ ─ ─ ┼ ─ ─ OS Boundary ─ ─ ─ ─ ─
                  │
┌─────────────────┼──────────────────────────┐
│            Linux Kernel                    │
│                 │                          │
│   ┌─────────────┴────────────┐             │
│   │   M1   M2   M3  (Threads)│             │
│   └─────────────┬────────────┘             │
│                 │                          │
│        Linux Scheduler (CFS)               │
│                 │                          │
│              CPU                           │
└────────────────────────────────────────────┘
```

---

## goroutine 的三种调度触发

```text
1. 主动让出 (Cooperative)
   - time.Sleep()
   - channel 操作 (阻塞)
   - sync.Mutex.Lock() (阻塞)
   - runtime.Gosched()

2. 系统调用 (Syscall)
   G 执行系统调用 (如 os.Open)
   → M 进入内核态
   → P 解绑，找其他 M
   → 系统调用返回后，G 重新排队

3. 抢占式调度 (Preemptive)
   Go 1.14+: 信号抢占
   → sysmon 线程检测到 G 运行超过 10ms
   → 发送 SIGURG 信号
   → G 被安全地抢占
```

---

## Go 内存管理与 OS

```text
Go Memory Allocator 的层次：

Go Allocator (mcache → mcentral → mheap)
  │  ← 小对象在 Go 内部管理
  ▼
mheap
  │  ← mmap 向 OS 申请虚拟地址空间
  ▼
OS Virtual Memory
  │  ← Demand Paging: 真正写入时才分配物理页
  ▼
Physical Memory
```

```text
Go GC 与 OS：

标记阶段 (STW)     → 扫描所有存活对象
并发标记            → 和用户代码并行
并发清扫 (Sweep)    → 释放未使用的内存
  │
  ▼
madvise(MADV_FREE) → 告诉 OS："这些物理页可以回收"
  │                    (虚拟地址空间还在，物理内存还给 OS)
  ▼
下次使用时重新分配物理页
```

---

## 核心概念

| 概念 | OS 层 | Go Runtime 层 |
|------|-------|-------------|
| **调度单位** | Thread (task_struct) | Goroutine |
| **创建开销** | ~10μs (clone) | ~几十 ns |
| **栈** | ~8MB 固定 | ~2KB 动态增长 |
| **调度器** | CFS (内核态) | GMP (用户态) |
| **IO 模型** | epoll / blocking | Netpoller (封装 epoll) |
| **内存管理** | mmap / brk | 分层分配器 + GC |
| **锁** | futex | sync.Mutex (基于 futex) |

---

## Go 是怎么利用它的？

### futex：Go 的 sync.Mutex 底层

```text
sync.Mutex.Lock() 两个层面：

用户态 (快速路径)：
atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked)
  → 抢到了！不需要系统调用

内核态 (慢速路径)：
抢了 N 次没抢到 → futex(FUTEX_WAIT) → 内核挂起 goroutine
释放锁 → futex(FUTEX_WAKE) → 唤醒等待者
```

### Channel 的底层

```text
ch := make(chan int, 10)
ch <- 42     // 发送
v := <-ch    // 接收

底层：环形队列 + 等待队列 (sudog) + runtime 调度

缓冲 channel：
├── 有空间 → 数据入队 → 返回
└── 满了 → goroutine 挂起 → 加入 sendq

无缓冲 channel：
发送者 → 挂起 → 等待接收者
接收者 → 取出数据 → 唤醒发送者
```

---

## 常见面试题

**Q: GMP 中的 P 的作用？**

A: P (Processor) 是 Go 调度器的核心。它管理 goroutine 的本地队列，数量和 `GOMAXPROCS` 对应。P 可以被 M 借走执行 G，也可以和 M 解绑。P 让 Go 的调度更加灵活。

**Q: Go 的 goroutine 会被抢占吗？**

A: Go 1.14 之前，goroutine 只有在函数调用/系统调用/阻塞时才会切换（协作式）。Go 1.14+ 引入基于信号的抢占式调度，`sysmon` 会发送 `SIGURG` 抢占运行超过 10ms 的 goroutine。

**Q: Go 的 GC 会影响性能到什么程度？**

A: Go GC 是并发标记-清扫，STW 时间通常在微秒级别。但大量堆分配会导致 GC 频率增加、CPU 消耗增加。调优方向：减少堆分配、使用 `sync.Pool`、预分配切片容量。

---

## 实战

```bash
# Go 相关环境变量
GOMAXPROCS=4 go run main.go     # 限制 P 数量
GODEBUG=schedtrace=1000 ./main  # 打印调度器追踪信息
GODEBUG=gctrace=1 ./main        # 打印 GC 追踪信息

# 性能分析
go tool pprof http://localhost:6060/debug/pprof/profile
go tool trace trace.out
```

```go
// labs/runtime/main.go
package main

import (
    "fmt"
    "runtime"
    "runtime/debug"
    "time"
)

func main() {
    // 查看 GC 信息
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    fmt.Printf("Alloc: %v MB\n", m.Alloc/1024/1024)
    fmt.Printf("NumGC: %v\n", m.NumGC)

    // 设置 GC 百分比（默认 100，即堆增长 100% 触发 GC）
    debug.SetGCPercent(100)

    // 手动触发 GC
    runtime.GC()

    // 查看 goroutine 数量
    fmt.Printf("Goroutines: %d\n", runtime.NumGoroutine())
    fmt.Printf("CPU cores: %d\n", runtime.NumCPU())
    fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))

    // 查看调用栈（调试时有用）
    buf := make([]byte, 1024)
    n := runtime.Stack(buf, true)
    fmt.Printf("Stack: %s\n", buf[:n])

    time.Sleep(time.Second)
}
```

---

## 总结

> Go Runtime 是 OS 之上的一层"用户态操作系统"：GMP 调度器补充 Linux CFS，Netpoller 封装 epoll，GC 管理内存生命周期，Channel 提供无锁并发通信。

---

## 与后端开发的联系

```text
GOMAXPROCS → 理解为什么容器中 Go 程序可能受限
           → 理解 CPU 限制和 GMP 的关系

GC 调优   → GOGC 参数调整
        → 线上内存泄漏排查（pprof heap）
        → goroutine 泄漏排查（pprof goroutine）

Netpoller → 理解为什么 Go HTTP Server 不要前面加 Nginx
          → 但大文件下载仍需反向代理

futex    → Go 的 Mutex 竞争分析
        → pprof mutex profile
```
