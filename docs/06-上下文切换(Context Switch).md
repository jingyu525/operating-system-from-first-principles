# 06 — 上下文切换 (Context Switch)

---

## 这一章回答什么问题？

> 进程/线程切换时到底发生了什么？为什么说 Context Switch 很昂贵？

---

## 第一性原理

CPU 运行一个程序时，依赖大量"现场"：

```text
寄存器：rax, rbx, rcx, rdx, rsp, rip...
程序计数器 (PC)：下一条指令的地址
栈指针 (SP)：当前栈的位置
状态寄存器：标志位（进位、零位等）
MMU 状态：页表基地址寄存器 (CR3)
```

**切换进程 = 保存当前进程的全部现场 + 恢复下一个进程的全部现场。**

---

## 推导过程

### Context Switch 的完整流程

```text
        进程 A 正在运行
              │
              ▼
      [定时器中断 / 系统调用]
              │
              ▼
    CPU 切换到内核态
              │
              ▼
    保存进程 A 的上下文：
      - 通用寄存器 (rax, rbx, ...)
      - 段寄存器
      - 程序计数器 (rip)
      - 栈指针 (rsp)
      - 状态寄存器 (rflags)
      - 浮点寄存器
      - CR3 (页表基地址)
              │
              ▼
    选择进程 B（调度器决策）
              │
              ▼
    恢复进程 B 的上下文
              │
              ▼
    iretq 返回用户态
              │
              ▼
        进程 B 开始运行
```

### 为什么昂贵？

```text
1. 保存/恢复几十个寄存器 → CPU 时间
2. 切换页表 (CR3) → TLB 全部失效！
3. 新进程的 Cache 是冷的 → Cache Miss 暴增
4. 调度器本身的决策开销
```

**TLB (Translation Lookaside Buffer) 失效是最大的隐性成本。**

```text
TLB：虚拟地址 → 物理地址的快表（Cache of Page Table）

进程 A：TLB 缓存了 A 的虚拟地址映射
↓ 切换到进程 B
TLB 中的 A 的条目全部失效！（或被刷掉）
↓
进程 B 每次访问内存都要走完整的页表查询（慢！）
```

### 不同切换的开销对比

| 切换类型 | 做了什么 | 大致开销 |
|----------|---------|---------|
| 函数调用 | 保存/恢复几个寄存器 | ~ns |
| goroutine 切换 | Go Runtime 用户态切换 | ~200ns |
| 同进程线程切换 | 保存/恢复寄存器，不切页表 | ~1-5μs |
| 不同进程切换 | 保存/恢复寄存器 + 切页表 + TLB flush | ~5-10μs |
| VM Exit | 虚拟化场景 | ~10-50μs |

---

## 核心概念

| 概念 | 本质 |
|------|------|
| **Context Switch** | OS 暂停当前任务、恢复下一个任务的过程 |
| **TLB** | 虚拟→物理地址转换的快表 |
| **TLB Flush** | 切换地址空间时让旧的 TLB 条目失效 |
| **Cache Miss** | CPU Cache 中没有需要的数据 |
| **自愿切换** | 进程主动放弃 CPU（如调用 `sleep()`） |
| **非自愿切换** | 时间片到了，被强制切换 |

---

## Linux 是怎么实现的？

```c
// 内核中 context_switch() 的核心逻辑（极度简化）
static __always_inline struct rq *
context_switch(struct rq *rq, struct task_struct *prev,
               struct task_struct *next) {
    
    // 1. 切换内存上下文（切换页表）
    switch_mm_irqs_off(prev->active_mm, next->mm, next);
    
    // 2. 切换寄存器上下文
    switch_to(prev, next, prev);
    
    return rq;
}
```

`switch_mm`：切换 CR3 寄存器 → 切换页表 → TLB 失效。

`switch_to`：用汇编保存当前寄存器、恢复下一个进程的寄存器。（这部分极其依赖架构，x86-64 和 ARM 完全不同。）

---

## Go 是怎么利用它的？

Go 的设计理念：**减少 Context Switch**。

### 1. GOMAXPROCS = CPU 核数

避免过多 OS Thread 导致的上下文切换：

```go
// 默认已经是 CPU 核数
runtime.GOMAXPROCS(runtime.NumCPU())
```

### 2. 系统调用时 M 不变，P 换 M

```text
M1 执行 G1         M1 被系统调用阻塞
    │                    │
    ▼                    ▼
P 绑定 M1           P 解绑 M1，找空闲 M2
    │                    │
    ▼                    ▼
正常执行            M1 阻塞等 IO
                   P+M2 继续调度其他 G
```

### 3. netpoller：避免 IO 阻塞 goroutine

```text
G1 执行网络 IO → G1 注册到 netpoller → G1 被挂起
                               │
                         P 继续调度 G2
                               │
                         IO 就绪 → netpoller 唤醒 G1
```

---

## 常见面试题

**Q: Context Switch 具体做了哪些事？**

A:
1. 保存当前进程/线程的 CPU 寄存器
2. 更新 PCB/TCB 状态
3. 把当前进程放到合适的队列
4. 选择下一个进程
5. 更新内存管理结构（切换页表 → 可能导致 TLB flush）
6. 恢复下一个进程的寄存器上下文
7. 返回用户态

**Q: 为什么进程切换比线程切换贵？**

A: 核心区别是**页表切换**。线程共享地址空间，不需要切换 CR3 → 不需要 TLB flush。进程切换需要切换 CR3 → TLB 全部失效 → 内存访问变慢。

**Q: Go 怎么减少上下文切换？**

A:
- GOMAXPROCS 限制 OS Thread 数量
- goroutine 切换在用户态（不涉及系统调用）
- netpoller 避免 IO 阻塞 M
- work-stealing 减少线程空闲

---

## 实战

```bash
# 查看每秒上下文切换次数
vmstat 1 5
# cs 列就是上下文切换次数

# 更详细的切换统计
pidstat -w 1

# cswch/s  = 自愿切换（voluntary）
# nvcswch/s = 非自愿切换（involuntary）

# 查看特定进程的切换
pidstat -w -p <PID> 1
```

```go
// labs/context-switch/main.go
package main

import (
    "fmt"
    "runtime"
    "sync"
    "time"
)

func main() {
    // 场景 1：绑定到 1 个 OS Thread → 很多 goroutine 切换（用户态，开销小）
    runtime.GOMAXPROCS(1)
    var wg sync.WaitGroup

    start := time.Now()
    for i := 0; i < 10000; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // 模拟工作
        }()
    }
    wg.Wait()
    fmt.Printf("GOMAXPROCS=1, 10000 goroutines: %v\n", time.Since(start))

    // 观察 mtomatic switching
    runtime.GOMAXPROCS(runtime.NumCPU())
}
```

---

## 总结

> Context Switch 是 OS 多任务的基础，代价主要在寄存器保存/恢复和 TLB 失效。Go 通过用户态调度和 GOMAXPROCS 控制来减少内核级上下文切换。

---

## 与后端开发的联系

```text
理解上下文切换 → 理解为什么线程池设太大反而变慢
              → 理解 Go 的 goroutine 模型为什么适合高并发
              → 理解容器 CPU 限流(vmstat cs 暴增 → 可能是 CPU throttle)
              → 理解为什么 Redis 单线程反而快(没有上下文切换)
              → 理解 NUMA 架构对性能的影响(跨 NUMA 节点切换更贵)
```
