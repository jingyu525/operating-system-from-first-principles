package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

func main() {
	gomaxprocs := runtime.GOMAXPROCS(0)
	fmt.Printf("PID: %d\n", os.Getpid())
	fmt.Printf("GOMAXPROCS: %d\n", gomaxprocs)
	fmt.Printf("CPU 核数: %d\n", runtime.NumCPU())

	fmt.Println("\n=== 实验 1: CPU 密集型 goroutine 的调度 ===")
	var wg sync.WaitGroup

	// 创建 CPU 核数 × 2 个 CPU 密集型 goroutine
	for i := 0; i < gomaxprocs*2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			start := time.Now()
			// 忙等，测试抢占式调度
			count := 0
			for time.Since(start) < 2*time.Second {
				count++
			}
			fmt.Printf("  goroutine %d: %d 次循环\n", id, count)
		}(i)
	}

	wg.Wait()

	fmt.Println("\n=== 实验 2: IO 密集型 goroutine 的调度 ===")
	for i := 0; i < 10; i++ {
		go func(id int) {
			// IO 密集型：大部分时间在等
			time.Sleep(500 * time.Millisecond)
			fmt.Printf("  IO goroutine %d 完成\n", id)
		}(i)
	}

	time.Sleep(1 * time.Second)

	fmt.Println("\n=== 调度观察完成 ===")
	fmt.Printf("最终 goroutine 数: %d\n", runtime.NumGoroutine())
	fmt.Println("\n提示：在另一个终端运行以下命令观察调度行为：")
	fmt.Printf("  vmstat 1                    # 看 cs 列（上下文切换）\n")
	fmt.Printf("  pidstat -w -p %d 1          # 看进程级切换\n", os.Getpid())
	fmt.Printf("  top -H -p %d                # 看线程视图\n", os.Getpid())
}
