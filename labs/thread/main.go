package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

func main() {
	// 实验 1：创建大量 goroutine 的代价
	start := time.Now()
	var wg sync.WaitGroup

	const numGoroutines = 100000
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("创建了 %d 个 goroutine:\n", numGoroutines)
	fmt.Printf("  耗时: %v\n", elapsed)
	fmt.Printf("  当前 goroutine 数: %d\n", runtime.NumGoroutine())
	fmt.Printf("  OS Thread 数: %d (通过 runtime 设置: %d)\n",
		runtime.GOMAXPROCS(0), runtime.GOMAXPROCS(0))
	fmt.Printf("  堆内存: %d MB\n", m.Alloc/1024/1024)

	// 实验 2：对比
	fmt.Println("\n--- 对比 ---")
	fmt.Println("如果是 OS Thread:")
	fmt.Println("  100,000 × 8MB 栈 = 800GB 虚拟内存 → 不可能")
	fmt.Println("  100,000 个线程的上下文切换 → CPU 全耗在切换上")
	fmt.Println()
	fmt.Println("Goroutine:")
	fmt.Println("  100,000 × ~2KB 初始栈 = ~200MB → 完全可行")
	fmt.Println("  切换在用户态 → 极低开销")

	time.Sleep(100 * time.Millisecond)
}
