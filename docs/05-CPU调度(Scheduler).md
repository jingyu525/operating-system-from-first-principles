# 05 — CPU 调度 (Scheduler)

---

## 这一章回答什么问题？

> 为什么一个 CPU 能同时运行多个程序？Scheduler 到底在做什么？

---

## 第一性原理

```text
CPU 一次只能执行一条指令。
```

所以多个程序**必须轮流运行**。

Scheduler 就是决定"下一个是谁"的人：

> **Scheduler 本质是 CPU 资源分配器。**

---

## 推导过程

### 如果没有调度器

```text
时间 →
CPU: [程序 A 一直跑... 直到结束] → [程序 B 一直跑...]
```

- A 卡住了 → 整个系统卡死
- A 是死循环 → B 永远轮不到

### 有了调度器

```text
时间 →
CPU: [A][B][C][A][B][D][A][C]...

A got: 3 slices
B got: 2 slices
C got: 2 slices
D got: 1 slice
```

每个程序获得**CPU 时间片**，轮流执行。

为什么我们感觉所有程序"同时在运行"？

```text
QQ     [1ms 开始执行]
       ↓ (1ms 后)
Chrome [收到1ms CPU时间]
       ↓ (1ms 后)
MySQL  [收到1ms CPU时间]
       ↓
QQ     [又拿到了！]
```

**切换太快（毫秒级），人眼根本分辨不出。**

---

## 调度器的目标

```text
1. 公平性：每个进程都能拿到 CPU
2. 吞吐量：单位时间内完成的任务尽可能多
3. 响应时间：交互式任务尽快响应
4. 开销低：调度本身不能太吃 CPU
```

矛盾：公平 vs 吞吐量、响应速度 vs 上下文切换开销。

---

## 调度算法的演进

### 1. FCFS (First Come First Served)

```text
[任务A: 100s][任务B: 1s][任务C: 1s]
平均等待时间：(0 + 100 + 101) / 3 = 67s
```
问题：长任务堵死短任务（护航效应）。

### 2. SJF (Shortest Job First)

```text
[任务B: 1s][任务C: 1s][任务A: 100s]
平均等待时间：(0 + 1 + 2) / 3 = 1s
```
问题：需要预知任务时间，实际做不到。

### 3. Round-Robin

```text
时间片 = 1s
[A1][B1][C1][A2][B2][C2][A3]...
```
公平，但上下文切换太频繁 → 开销大。

### 4. MLFQ (Multi-Level Feedback Queue)

```text
优先级高（时间片短）  ← 适合 IO 密集型
  Q1: [A] → 用完了？ → 降级
  Q2: [A] → 又用完了？ → 降级
  Q3: [A] → 继续降级
优先级低（时间片长）  ← 适合 CPU 密集型
```

现代操作系统（Linux、macOS、Windows）都基于 MLFQ 思想。

---

## 核心概念

| 概念 | 本质 |
|------|------|
| **时间片 (Time Slice)** | 进程在 CPU 上连续运行的最大时间 |
| **就绪队列 (Ready Queue)** | 所有等待 CPU 的进程排队的地方 |
| **抢占 (Preemption)** | 时间片到了，强制切换 |
| **非抢占** | 进程主动放弃 CPU（如等待 IO） |
| **CPU 密集型** | 大部分时间在计算 |
| **IO 密集型** | 大部分时间在等 IO |

---

## Linux 是怎么实现的？

### CFS (Completely Fair Scheduler)

核心思想：

> **给每个进程分配虚拟运行时间 (vruntime)，谁跑得少谁优先。**

```text
vruntime = 实际运行时间 × (1024 / 进程权重)

例子：
进程 A (nice=0, 权重=1024)：跑了 10ms → vruntime = 10ms
进程 B (nice=5, 权重=335)： 跑了 10ms → vruntime = 30.5ms

A 的 vruntime 更小 → A 优先！
```

用**红黑树**管理就绪队列，基于 vruntime 排序。

```bash
# 查看进程的调度策略和优先级
chrt -p <PID>

# 其他策略
# SCHED_OTHER (CFS, 默认)
# SCHED_FIFO  (实时，先入先出)
# SCHED_RR    (实时，轮转)
```

---

## Go 是怎么利用它的？

### GMP 调度模型

```text
G (Goroutine)  ← 用户态协程
P (Processor)  ← 逻辑处理器（= GOMAXPROCS，默认 CPU 核数）
M (Machine)    ← OS Thread
```

```text
GMP 视角的调度层次：

用户态 (Go Runtime)：
  Goroutine → P → M
  
内核态 (Linux)：
  M (OS Thread) → Linux CFS → CPU
```

两层调度：

```text
1. Go 调度器决定：哪个 G 放到哪个 M 上
2. Linux 调度器决定：哪个 M（OS Thread）放到哪个 CPU 核上
```

Go 调度器的特点：

- **work-stealing**：P 的本地 G 队列空了，去别的 P 偷
- **handoff**：M 执行 G 时阻塞了（如系统调用），P 找其他 M
- **抢占式调度**：Go 1.14+ 支持基于信号的异步抢占

---

## 常见面试题

**Q: goroutine 为什么比线程轻？**

A: 从调度角度：
- 创建/销毁在用户态 → 不需要系统调用
- 切换在用户态 → 不需要保存/恢复大量内核寄存器
- 栈动态增长 → 初始只有 2KB

**Q: 线程太多为什么 CPU 利用率下降？**

A:
1. 上下文切换开销 → 切换本身吃 CPU
2. Cache miss → 切换后 Cache 全是冷数据
3. 调度器开销 → 更多 task_struct 要管理

**Q: IO 密集型和 CPU 密集型任务应该用不同的线程池大小吗？**

A: 是的：
- CPU 密集型：线程数 ≈ CPU 核数（多了浪费在切换上）
- IO 密集型：线程数可以更多（因为大部分时间在等 IO，CPU 空闲）

---

## 实战

```bash
# 观察 CPU 调度
top -1       # 查看 CPU 使用率
htop         # 更直观的进程管理

# 把进程绑定到特定 CPU
taskset -c 0,1 myapp

# 调整优先级（nice：-20 最高，19 最低）
nice -n -10 myapp

# 查看上下文切换
vmstat 1     # cs 列是每秒上下文切换次数

# 性能分析
perf stat -e context-switches,cpu-migrations myapp
```

```go
// labs/scheduler/main.go
package main

import (
    "fmt"
    "runtime"
    "time"
)

func cpuIntensive() {
    for i := 0; i < 1e9; i++ {
    }
    fmt.Println("CPU 密集型任务完成")
}

func ioIntensive() {
    time.Sleep(1 * time.Second)
    fmt.Println("IO 密集型任务完成")
}

func main() {
    runtime.GOMAXPROCS(1) // 只用一个 OS Thread，观察 Go 调度器如何切换

    go cpuIntensive()
    go ioIntensive()
    go cpuIntensive()

    time.Sleep(2 * time.Second)
}
```

---

## 总结

> Scheduler 的目标就是让 CPU 尽可能一直工作，同时兼顾公平性和响应速度。

---

## 与后端开发的联系

```text
Go GMP Scheduler → 理解 goroutine 的调度行为
                → 知道什么时候需要 runtime.Gosched()
                → 理解 sync.Mutex 阻塞时的调度行为

Linux CFS       → 理解容器 CPU 限制 (cgroup cpu.shares)
                → 理解 nice 值对服务稳定性的影响
                → CPU throttle 问题排障
```
