# Scheduler Lab

## 实验目标

观察 CPU 调度和 Go 调度器的行为。

## 实验 1：观察上下文切换

```bash
# 运行程序
go run main.go

# 在另一个终端
vmstat 1
pidstat -w -p $(pgrep sched-lab) 1
```

## 实验 2：GOMAXPROCS 的影响

```bash
GOMAXPROCS=1 go run main.go   # 单 OS Thread，观察 Go 调度器
GOMAXPROCS=4 go run main.go   # 4 个 OS Thread，观察并行
```

## 实验 3：CPU 亲和性

```bash
taskset -c 0,1 go run main.go  # 绑定到 CPU 0 和 1
```

## 关键问题

- GOMAXPROCS=1 时，所有 goroutine 跑在一个 OS Thread 上，谁在做调度？
- GOMAXPROCS > 1 时，goroutine 怎么分配到不同的 OS Thread？
- busy loop 的 goroutine 为什么在 Go 1.14 之前不会自动让出 CPU？
