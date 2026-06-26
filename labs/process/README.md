# Process Lab

## 实验目标

理解进程的创建、状态和父子关系。

## 实验 1：观察进程信息

```bash
# 编译运行 labs/process/main.go
go run main.go

# 输出：
# 我的 PID: 12345
# 父进程 PID: 1234
```

## 实验 2：用 /proc 探索进程

```bash
# 运行程序后，在另一个终端
ps aux | grep myapp
cat /proc/<PID>/status
cat /proc/<PID>/maps
ls /proc/<PID>/fd
```

## 实验 3：观察 fork+exec

参考 `exec_demo.go`，观察 Go 如何创建子进程。
