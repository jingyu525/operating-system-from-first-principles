# Thread Lab

## 实验目标

对比 OS Thread 和 Goroutine 的创建开销和内存占用。

## 实验 1：创建大量 goroutine

```bash
go run main.go
```

观察：
- 创建 10000 个 goroutine 的时间和内存
- 创建 10000 个 OS Thread 的时间对比（理论分析）

## 实验 2：监控线程数

```bash
# 运行程序，在另一个终端
cat /proc/$(pgrep thread-lab)/status | grep Threads
```

## 关键问题

- 为什么 goroutine 比 OS Thread 轻这么多？
- 如果你创建 10000 个 OS Thread，会发生什么？
