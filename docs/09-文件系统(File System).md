# 09 — 文件系统 (File System)

---

## 这一章回答什么问题？

> 为什么 `rm hugefile.log` 几乎是瞬间完成的？文件系统到底怎么管理磁盘空间的？

---

## 第一性原理

```text
磁盘 = 一块巨大的字节数组。
问题：怎么在这块大数组里快速找到某个文件？
```

> **文件系统本质是磁盘空间的索引。** 它告诉你"文件 X 的数据在磁盘的哪些块上"。

---

## 推导过程

### 最简单的文件系统

```text
方案 1：所有文件连续存储
[file_a.txt][file_b.txt][file_c.txt]

问题：
- file_a.txt 变大了怎么办？后面的文件要整体移动
- 删除 file_b.txt 留出空洞 → 碎片
```

### 现代文件系统：inode + 数据块

```text
超级块 (Superblock)：文件系统元信息
  ↓
inode 表：每个文件一个 inode
  ↓
数据块：实际存储文件内容

一个 inode 包含：
  - 文件类型 (普通文件/目录/链接)
  - 权限 (rwx)
  - 大小
  - 时间戳 (创建/修改/访问)
  - 数据块指针
```

### 一个文件对应一个 inode

```text
"test.txt"
  ↓ 路径查找
inode 42
  ├── 权限: rw-r--r--
  ├── 大小: 1024 bytes
  ├── 数据块指针: [128, 256, 512]
  └── ...

块 128: "Hello"
块 256: " Worl"
块 512: "d!\n"
```

### 为什么删除文件是瞬时的？

```text
rm file.txt 做的事情：
1. 找到 file.txt 的 inode
2. 标记 inode 为"未使用"
3. 标记数据块为"空闲"

注意：没有擦除数据块的内容！
     真正的数据还在磁盘上（直到被覆盖）。
```

---

## 目录的实质

```text
目录 = 一张表，记录"名字 → inode 号"

ls /home/user
  ↓
读取目录 inode 的数据块：
  ┌──────────┬──────────┐
  │  名称     │  inode   │
  ├──────────┼──────────┤
  │  .       │   100    │
  │  ..      │   50     │
  │  docs    │   200    │
  │  main.go │   201    │
  └──────────┴──────────┘
```

---

## 软链接 vs 硬链接

```text
硬链接 (hard link)：
┌─────────────┐
│ "name1"     │ ──→ inode 42 ←── 数据块
│             │
│ "name2"     │ ──→ inode 42 ←──┘
└─────────────┘
两个名字指向同一个 inode，删除一个不影响另一个。
只有引用计数 = 0 时才真正删除。

软链接 (symlink)：
"shortcut" → 一个特殊 inode，内容是目标路径字符串
删除原文件 → 软链接变成"死链接"。
```

---

## 核心概念

| 概念 | 本质 |
|------|------|
| **inode** | 文件的元数据（权限、大小、数据块位置） |
| **数据块** | 存储文件实际内容的磁盘块（通常 4KB） |
| **超级块** | 文件系统的全局信息 |
| **目录** | 特殊的文件，内容是"名字→inode"的映射 |
| **硬链接** | 同一个 inode 的另一个名字 |
| **软链接** | 指向路径的快捷方式 |
| **Page Cache** | 内核用空闲内存缓存文件数据 |

---

## Linux 是怎么实现的？

### VFS (Virtual File System)

```text
用户程序
  │  read() / write() / open()
  ▼
VFS (统一的文件系统接口)
  ├── ext4
  ├── xfs
  ├── btrfs
  ├── procfs (/proc)
  ├── sysfs (/sys)
  └── tmpfs (/tmp, 在内存中)
```

所有文件系统实现同一套接口，上层代码不用关心下层是什么。

### Page Cache

```text
read("test.txt")
  │
  ▼
检查 Page Cache 中有没有该文件的页
  ├── 有 (Cache Hit) → 直接从内存返回 (快！)
  └── 没有 (Cache Miss) → 从磁盘读 → 放入 Page Cache → 返回
```

---

## Go 是怎么利用它的？

```go
f, _ := os.OpenFile("test.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
f.Write([]byte("hello\n"))
// 此时数据可能还在 Page Cache 中！
// 没有落到磁盘！

f.Sync()  // 强制 fsync → 数据写入磁盘
```

Go 标准库的 `os.File` 直接封装系统调用：

```go
// os/file.go (简化)
func (f *File) Write(b []byte) (n int, err error) {
    // → syscall.Write(f.fd, b)
    // → 内核 → Page Cache → (异步刷盘)
}

func (f *File) Sync() error {
    // → syscall.Fsync(f.fd)
    // → 内核 → 强制刷盘
}
```

---

## 常见面试题

**Q: `rm` 为什么很快？**

A: `rm` 只是把 inode 和数据块标记为"空闲"，不擦除数据。真正的数据还在磁盘上，只是文件系统不认了。

**Q: `mv` 和 `cp` 的区别？**

A:
- **同分区 `mv`**：只修改目录中的 inode 映射，数据块不动 → 瞬间完成
- **跨分区 `mv`** = `cp` + `rm` → 需要复制数据
- **`cp`**：读数据 → 写新文件 → 数据实际复制

**Q: 硬链接和软链接的区别？**

A:
- 硬链接：同一个 inode 的两个名字，删除任何一个都不影响数据
- 软链接：一个指向路径的特殊文件，原文件删除后变成死链

**Q: `fsync` 做了什么？为什么它对 MySQL/Redis 很重要？**

A: `fsync` 强制把 Page Cache 中的脏页写回磁盘。没有 `fsync`，数据可能还在内存中，断电就丢了。MySQL 的 redo log 和 Redis 的 AOF 都依赖 `fsync` 保证持久性。

---

## 实战

```bash
# 查看文件 inode
ls -i file.txt
stat file.txt

# 查看磁盘使用
df -h
df -i        # inode 使用情况

# 查看目录（目录也是文件）
ls -ld /home/user    # d 开头 = 目录
ls -lai              # -i 显示 inode 号

# 创建 hard link / symlink
ln file.txt hardlink.txt    # 硬链接
ln -s file.txt softlink.txt # 软链接

# 强制刷 Page Cache
sync
echo 3 > /proc/sys/vm/drop_caches  # 清理 Page Cache（慎用！）
```

```go
// labs/filesystem/main.go
package main

import (
    "fmt"
    "io/ioutil"
    "os"
    "syscall"
)

func main() {
    // 写文件
    f, _ := os.Create("test.txt")
    f.Write([]byte("hello, filesystem!"))
    
    // 获取 inode 信息
    fi, _ := f.Stat()
    stat := fi.Sys().(*syscall.Stat_t)
    fmt.Printf("Inode: %d\n", stat.Ino)
    fmt.Printf("Block count: %d\n", stat.Blocks)
    
    // 强制刷盘
    f.Sync()
    f.Close()

    // ioutil.ReadFile 内部用 Read 系统调用 → Page Cache
    data, _ := ioutil.ReadFile("test.txt")
    fmt.Printf("读取: %s\n", data)
}
```

---

## 总结

> 文件系统本质是磁盘空间的索引。inode 记录文件元数据，目录是名字→inode 的映射表，Page Cache 是内核用内存加速文件访问的利器。

---

## 与后端开发的联系

```text
fsync  → MySQL 的 redo log / binlog 持久化
       → Redis AOF 持久化
       → Kafka 日志段写入

Page Cache → Linux 用空闲内存缓存文件
          → 导致 OOM 误判（cache 是可以释放的！）
          → 影响数据库的 Buffer Pool 设计

硬链接  → Docker 镜像分层存储（copy-on-write + hard link）

inode 耗尽 → 磁盘有空间但无法创建文件 → 排查思路
```
