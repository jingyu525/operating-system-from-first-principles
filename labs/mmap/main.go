package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"time"
)

const fileSize = 100 * 1024 * 1024 // 100MB

func main() {
	// 创建测试文件
	filename := "test_mmap.dat"
	f, _ := os.Create(filename)
	data := make([]byte, fileSize)
	f.Write(data)
	f.Close()
	defer os.Remove(filename)

	// 测试 1: 传统 read 方式
	start := time.Now()
	f1, _ := os.Open(filename)
	for i := 0; i < 10; i++ {
		buf := make([]byte, fileSize)
		_, _ = f1.ReadAt(buf, 0)
	}
	f1.Close()
	fmt.Printf("传统 Read: %v\n", time.Since(start))

	// 测试 2: mmap 方式
	start = time.Now()
	f2, _ := os.Open(filename)
	defer f2.Close()
	for i := 0; i < 10; i++ {
		mmapData, err := syscall.Mmap(int(f2.Fd()), 0, fileSize,
			syscall.PROT_READ, syscall.MAP_SHARED)
		if err != nil {
			fmt.Printf("mmap 失败: %v\n", err)
			return
		}
		// 遍历数据（强制访问以触发真实的页加载）
		sum := 0
		for j := 0; j < len(mmapData); j += 4096 {
			sum += int(mmapData[j])
		}
		_ = sum
		syscall.Munmap(mmapData)
	}
	fmt.Printf("mmap: %v\n", time.Since(start))

	// 测试 3: ioutil
	start = time.Now()
	for i := 0; i < 10; i++ {
		_, _ = ioutil.ReadFile(filename)
	}
	fmt.Printf("ioutil.ReadFile: %v\n", time.Since(start))

	fmt.Println("\n关键结论:")
	fmt.Println("  mmap 将文件直接映射到虚拟地址空间")
	fmt.Println("  访问 mmap 区域 = 直接访问 Page Cache")
	fmt.Println("  不需要 read() 系统调用, 不需要内核→用户态拷贝")
}
