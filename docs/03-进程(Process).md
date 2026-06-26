# 03 — 进程 (Process)

---

## 这一章回答什么问题？

> 进程到底是什么？为什么每个程序都觉得自己独占 CPU 和内存？

---

## 第一性原理

```text
CPU 一次只能运行一个程序。
内存只有一块。
```

但是我们需要同时运行微信、Chrome、终端……

所以必须有一个机制，让多个程序**轮流使用 CPU，并让它们互不干扰**。

这个机制就是**进程**。

一句话：

> **进程是操作系统对正在运行程序的抽象。** 它给每个程序营造"整个计算机都是我的"的幻觉。

---

## 推导过程

### 没有进程的世界

```text
时间 →
CPU: [程序A全部跑完] → [程序B全部跑完] → [程序C全部跑完]
```

问题：
- 程序 A 卡住了，B 和 C 永远轮不到
- 程序 A 访问了程序 B 的内存，B 崩溃

### 有进程的世界

```text
时间 →
CPU: [A] [B] [C] [A] [B] [C] [A] ...
     ↑   ↑   ↑   ↑   ↑   ↑   ↑
     时间片轮流切换
```

- 每个进程有自己的**虚拟地址空间**（看不到别人的内存）
- 每个进程有自己的**上下文**（寄存器、PC、栈等）
- OS 通过**调度器**决定谁在什么时候运行

---

## 进程的状态

```
        ┌──────────┐
        │ Created  │  (刚创建，还没准备好)
        └────┬─────┘
             │
             ▼
        ┌──────────┐
  ┌────→│  Ready   │  (准备好了，排队等 CPU)
  │     └────┬─────┘
  │          │ 调度器分配 CPU
  │          ▼
  │     ┌──────────┐
  │     │ Running  │  (正在 CPU 上执行)
  │     └──┬───┬───┘
  │        │   │
  │   等待IO  时间片用完
  │        │   │
  │        ▼   │
  │  ┌────────┐│
  │  │Blocked ││  (等待 IO、锁、信号等)
  │  └───┬────┘│
  │      │      │
  │   IO完成    │
  │      │      │
  │      ▼      │
  │  ┌────────┐ │
  └──│ Ready  │←┘
     └────────┘

              ┌──────────┐
              │Terminated│  (进程结束)
              └──────────┘
```

---

## PCB (Process Control Block)

操作系统用 PCB 记录每个进程的一切：

```c
// 逻辑结构（不是真实代码）
struct task_struct {     // Linux 中叫 task_struct
    pid_t pid;           // 进程 ID
    long state;          // 进程状态
    void *stack;         // 内核栈
    struct mm_struct *mm; // 地址空间
    struct files_struct *files; // 打开的文件
    struct signal_struct *signal; // 信号处理
    // ... 还有很多很多
};
```

---

## 进程的创建：fork + exec

```text
        父进程
          │
      fork()        ← 克隆一个一模一样的子进程
          │
    ┌─────┴─────┐
    │           │
  父进程      子进程（和父进程相同的内存、文件描述符等）
    │           │
  wait()     exec()   ← 替换子进程自己的内存为新的程序
    │           │
    └─────┬─────┘
          │
        继续
```

为什么 fork + exec 而不是直接创建？

> 因为 fork 之后可以在 exec 之前做一些事：重定向 stdin/stdout、设置环境变量等。
> Shell 就是这么实现的：

```bash
ls | grep foo
```

Shell 先 fork，然后在子进程中把 stdout 重定向到管道，再 exec 执行 `ls`。

---

## 核心概念

| 概念 | 本质 |
|------|------|
| **进程** | 运行中的程序 + 它的资源（地址空间、文件、信号等） |
| **PCB** | OS 记录进程信息的数据结构 |
| **fork** | 克隆当前进程 |
| **exec** | 替换当前进程为新程序 |
| **进程状态** | Ready / Running / Blocked / Terminated |
| **孤儿进程** | 父进程挂了，由 init (PID=1) 收养 |
| **僵尸进程** | 子进程已结束但父进程还没回收 |

---

## Linux 是怎么实现的？

```bash
# Linux 中一切进程都是 fork 出来的
# 进程 0 (idle) → 进程 1 (init/systemd) → 所有其他进程

# PID 1 是 systemd，所有进程的"祖先"
pstree -p 1
```

`/proc` 文件系统暴露了每个进程的 PCB 信息：

```bash
ls /proc/self/       # 查看自己进程的信息
cat /proc/self/maps  # 查看地址空间布局
cat /proc/self/status # 查看进程状态
ls /proc/self/fd/    # 查看打开的文件描述符
```

---

## Go 是怎么利用它的？

Go 程序中你看不到进程的创建，因为：

```go
// Go 程序是单进程多 goroutine
// goroutine ≠ 进程

// 但 Go 可以创建子进程
cmd := exec.Command("ls", "-l")
cmd.Run()

// 底层走的还是 fork + exec
```

Go runtime 初始化时：

```text
Go 程序就是一个普通进程
  │
  ├── 多个 OS Thread (M)
  ├── 多个 Goroutine (G)
  └── 一个进程地址空间
```

---

## 常见面试题

**Q: 进程和程序的区别？**

A: 程序是磁盘上的文件（死的），进程是内存中运行的程序实例（活的）。同一个程序可以运行多次，每次是一个单独的进程。

**Q: 孤儿进程和僵尸进程的区别？**

A:
- 孤儿进程：父进程死了，子进程还在 → 被 init 收养，无害
- 僵尸进程：子进程死了，父进程没调用 wait() → PCB 还留着，占资源

**Q: 为什么 Go 不支持 fork？**

A: Go 程序是多线程的，fork 只复制当前线程，其他线程的锁状态会不一致，会导致死锁。Go 用 `exec.Command` 代替。

---

## 实战

```bash
# 查看进程树
pstree -p

# 查看进程详情
ps aux

# 实时查看进程
top

# 查看 /proc
cat /proc/1/status
ls /proc/1/fd

# 查看僵尸进程
ps aux | grep Z
```

```go
// labs/process/main.go
package main

import (
    "fmt"
    "os"
    "os/exec"
)

func main() {
    fmt.Printf("我的 PID: %d\n", os.Getpid())
    fmt.Printf("父进程 PID: %d\n", os.Getppid())

    cmd := exec.Command("echo", "我是子进程")
    cmd.Stdout = os.Stdout
    cmd.Run()
}
```

---

## 总结

> 进程是 OS 对运行程序的抽象，通过 PCB 管理状态，通过虚拟地址空间实现隔离，通过 fork+exec 创建。

---

## 与后端开发的联系

```text
Go 服务进程 → 单进程多 goroutine 模型
             → 利用 fork+exec 执行外部命令
             → /proc 用于监控和诊断
             → 理解进程生命周期 → 理解容器 (namespace + cgroup)
```
