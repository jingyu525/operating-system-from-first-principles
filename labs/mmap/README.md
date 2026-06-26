# mmap Lab

## 实验目标

理解 mmap：零拷贝文件读写。

## 实验 1：对比 read 和 mmap

```bash
go run main.go
```

观察两种方式的性能差异。

## 实验 2：观察进程的虚拟内存

```bash
# 运行程序，在另一个终端
cat /proc/$(pgrep mmap-lab)/maps  # 查看虚拟内存映射
cat /proc/$(pgrep mmap-lab)/smaps # 详细内存统计
```

## 关键问题

- mmap 为什么比 read 快？
- mmap 映射的文件修改后，什么时候写回磁盘？
