# 08 — 系统调用 (System Call)

---

## 这一章回答什么问题？

> `os.Open()` 到底是怎么打开文件的？用户程序为什么不能直接操作磁盘？

---

## 第一性原理

```text
用户程序运行在 Ring 3 (用户态)
内核运行在 Ring 0 (内核态)

用户程序不能直接访问硬件（磁盘、网卡等）。
必须通过"门"进入内核 → 这就是系统调用。
```

> **系统调用 = 用户程序请求内核服务的唯一入口。**

---

## 推导过程

### 为什么不能直接操作硬件？

```text
如果程序 A 可以随意写磁盘任意扇区：
  → 程序 A 可能覆盖程序 B 的数据
  → 程序 A 可能破坏文件系统
  → 程序 A 可能读取其他用户的文件

所以必须有一个"守门人" → 内核。
```

### 系统调用的完整流程

```text
用户程序：
  os.Open("test.txt")
    │
    ▼
glibc / Go Runtime 封装：
  openat(AT_FDCWD, "test.txt", O_RDONLY)
    │
    ▼
SYSCALL 指令 (切换到内核态)：
  mov rax, 257      ← 系统调用号 (openat)
  mov rdi, AT_FDCWD ← 参数1
  mov rsi, path     ← 参数2
  mov rdx, flags    ← 参数3
  syscall           ← 触发！
    │
    ▼
内核态：
  1. 保存用户态寄存器
  2. 通过系统调用号查表 → sys_openat()
  3. 权限检查（你能打开这个文件吗？）
  4. 在文件系统中查找文件
  5. 分配文件描述符
  6. 返回 fd 到 rax
    │
    ▼
SYSRET 指令 (切换回用户态)：
  恢复 rax (包含返回值：fd 或 -errno)
    │
    ▼
Go Runtime：
  if fd < 0 → 返回 error
  else → 返回 *os.File{fd: fd}
```

---

## 切换的关键：用户态 ↔ 内核态

```text
                  ┌─────────────────┐
                  │     内核态       │
                  │  (Ring 0)       │
                  │                 │
 SYSCALL ────────→│  内核代码执行    │
                  │                 │
                  │                 │
                  │  返回结果        │
 SYSRET ←─────────│                 │
                  └─────────────────┘
                          │
                          │ 不可直接访问
                          │
                  ┌─────────────────┐
                  │     用户态       │
                  │  (Ring 3)       │
                  │                 │
                  │  你的程序代码    │
                  │                 │
                  └─────────────────┘
```

### 什么是"陷入"？

"陷入内核" 就是用户程序从**用户态**进入**内核态**的过程。CPU 执行 `SYSCALL` 指令时自动完成以下动作：

```text
SYSCALL 指令（一条 CPU 指令）自动做：
  ↓
1. 把 CPU 权限从 Ring 3 提升到 Ring 0
2. 保存返回地址到 rcx，保存 rflags 到 r11
3. 加载内核入口地址到 rip
4. 加载内核栈指针到 rsp
  ↓
现在 CPU 在内核态，执行内核代码
```

### 为什么从用户态到内核态的切换有代价？

```text
动作                             大致耗时
─────────────────────────────────────────
SYSCALL / SYSRET 指令本身         ~100ns
保存/恢复通用寄存器 (rax, rbx...)  ~几十 ns
保存/恢复浮点寄存器 (xmm, ymm...)  ~100-200ns
切换栈 (用户栈 ↔ 内核栈)          ~几十 ns
权限检查                          ~几十 ns
TLB 可能部分失效                   ~0-500ns（取决于是否有 PCID）
Spectre/Meltdown 缓解措施          ~200-500ns（Intel CPU 上更明显）
─────────────────────────────────────────
一次"陷入-返回"总共：              ~500ns - 1.5μs
```

对比一下：

```text
普通函数调用 (call/ret)：          ~1-5ns       ← 不过权限门
系统调用 (syscall/sysret)：        ~500-1500ns  ← 要过权限门

差 100-300 倍！
```

**一句话总结：** "陷入"的本质就是 CPU 执行 `SYSCALL` 指令穿过用户态→内核态这道权限门，过门本身就有一系列硬件开销，更不用说内核还要做安全检查。

### goroutine 切换为什么不需要陷入？

goroutine 的切换由 Go Runtime 在**用户态**完成——保存几个寄存器、换栈指针、跳转到新 goroutine 的代码继续执行，全程不经过 `SYSCALL` 指令。所以 goroutine 切换和普通函数调用差不多快（~200ns），而 OS Thread 切换必须"陷入"内核（~1-10μs），差 50 倍以上。

---

## 常见系统调用

| 系统调用 | Go 中对应 | 作用 |
|---------|----------|------|
| `read` | `f.Read()` | 读文件/socket |
| `write` | `f.Write()` | 写文件/socket |
| `openat` | `os.Open()` | 打开文件 |
| `close` | `f.Close()` | 关闭文件描述符 |
| `mmap` | `syscall.Mmap()` | 内存映射 |
| `brk` | (Go runtime 内部) | 调整堆大小 |
| `fork` | — | 创建子进程 |
| `execve` | `exec.Command()` | 执行新程序 |
| `clone` | (Go runtime 内部) | 创建线程 |
| `futex` | `sync.Mutex` 底层 | 用户态锁 |
| `epoll_wait` | `netpoller` 底层 | 等待 IO 事件 |
| `nanosleep` | `time.Sleep()` | 睡眠 |

---

## 核心概念

| 概念 | 本质 |
|------|------|
| **系统调用号** | 数字 ID，告诉内核你要什么服务 |
| **用户态 / 内核态** | CPU 的权限级别 |
| **SYSCALL / SYSRET** | x86-64 的快速系统调用指令 |
| **errno** | 系统调用失败时返回的错误码 |
| **vDSO** | 某些系统调用的用户态加速（避免切换） |

---

## Linux 是怎么实现的？

Linux 系统调用表：

```c
// arch/x86/entry/syscalls/syscall_64.tbl (简化)
0    read
1    write
2    open
...
257  openat
...
```

每个系统调用在内核中对应一个函数：

```c
SYSCALL_DEFINE3(openat, int, dfd, const char __user *, filename,
                int, flags, umode_t, mode) {
    // ... 具体实现
}
```

**注意 `__user` 标记**：内核不能直接访问用户态指针，需要通过 `copy_from_user` / `copy_to_user` 拷贝数据。

---

## Go 是怎么利用它的？

Go 绕过了 glibc，直接做系统调用：

```go
// Go 内部这样发起系统调用（简化）
func syscall.Syscall(trap uintptr, a1, a2, a3 uintptr) (uintptr, uintptr, uintptr)

// 例如：打开文件
func Open(path string, mode int, perm uint32) (fd int, err error) {
    // Go Runtime 直接发起 SYSCALL 指令
    r0, _, e1 := syscall.Syscall(syscall.SYS_OPEN, 
        uintptr(unsafe.Pointer(&path[0])), 
        uintptr(mode), 
        uintptr(perm))
    // ...
}
```

**为什么 Go 绕过 glibc？**

```text
1. 减少一层封装 → 更快
2. glibc 是 C 的，Go 有自己的调度器和栈管理
3. 静态链接 → 不依赖系统的 glibc 版本（避免 glibc 版本地狱）
```

在 Go 程序中，系统调用会导致 M 和 P 分离：

```text
G1 在 M1 上执行系统调用
  ↓
M1 进入内核 → P 被释放
  ↓
P 找到空闲 M2，继续调度其他 G
  ↓
M1 系统调用完成 → G1 找 P 重新排队
```

---

## 常见面试题

**Q: 用户态和内核态的区别？**

A: CPU 的权限级别（x86 的 Ring 3 vs Ring 0）。用户态不能执行特权指令（如修改页表、直接访问硬件），必须通过系统调用进入内核态。

**Q: 一次系统调用大概多少开销？**

A: 通常 ~100-500ns (现代 CPU)，但如果要等待 IO 则取决于 IO 本身。开销来源：寄存器保存/恢复、栈切换、权限检查、可能的 TLB 影响。

**Q: Go 为什么不直接用 glibc？**

A:
1. 静态链接，避免依赖
2. 更轻量的调用封装
3. 和 Go Runtime 调度器深度整合（如异步系统调用）

---

## 实战

```bash
# 查看进程的系统调用
strace ls                  # 追踪所有系统调用
strace -c ls               # 统计系统调用次数和时间
strace -e openat ls        # 只追踪 openat

# 查看系统调用表
man syscalls

# 用 perf 统计系统调用
perf trace ls
```

```go
// labs/syscall/main.go  
package main

import (
    "fmt"
    "syscall"
)

func main() {
    // 直接用系统调用打开文件
    path := "/etc/hostname"
    fd, err := syscall.Open(path, syscall.O_RDONLY, 0)
    if err != nil {
        fmt.Printf("open failed: %v\n", err)
        return
    }
    defer syscall.Close(fd)

    buf := make([]byte, 256)
    n, err := syscall.Read(fd, buf)
    if err != nil {
        fmt.Printf("read failed: %v\n", err)
        return
    }
    fmt.Printf("读到 %d 字节: %s\n", n, string(buf[:n]))
}
```

---

## 总结

> 系统调用是用户程序进入内核的唯一入口。Go 直接发起系统调用（绕过 glibc），并在系统调用时做 M-P 分离以保持并发性能。

---

## 与后端开发的联系

```text
strace    → 线上问题排障利器（哪个系统调用慢？卡在哪里？）
futex     → Go sync.Mutex 的底层实现
epoll_wait → Go netpoller 的基础
fsync     → MySQL/Redis 持久化的关键系统调用
write     → 日志框架最终都调用 write()
sendfile  → Nginx 零拷贝的底层
```
