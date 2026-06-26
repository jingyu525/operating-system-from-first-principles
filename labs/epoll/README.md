# epoll Lab

## 实验目标

理解 epoll：用 Go netpoller 实现高并发 TCP 服务器。

## 实验 1：TCP Echo Server

```bash
# 启动服务器
go run main.go

# 在另一个终端测试
echo "hello" | nc localhost 8080
```

## 实验 2：观察 epoll

```bash
# 查看服务器打开的文件描述符
lsof -p $(pgrep epoll-lab) | grep sock

# 用 strace 追踪
strace -e epoll_wait,accept4 -p $(pgrep epoll-lab)
```

## 实验 3：压测

```bash
# 用 Apache Bench 压测
ab -n 10000 -c 100 http://localhost:8080/
```

## 关键问题

- `lsof` 看到的每个 socket fd 都注册在 epoll 中
- 为什么一个 accept goroutine + N 个 reader goroutine 能处理几千个连接？
- 底层只有一个 epoll_wait，Go Runtime 怎么把事件分发给正确的 goroutine？
