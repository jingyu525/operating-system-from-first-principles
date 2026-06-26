package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	fmt.Printf("我的 PID: %d\n", os.Getpid())
	fmt.Printf("父进程 PID: %d\n", os.Getppid())
	fmt.Printf("我的 UID: %d\n", os.Getuid())

	// 执行外部命令（底层 fork + exec）
	cmd := exec.Command("echo", "我是 fork + exec 创建的子进程")
	cmd.Stdout = os.Stdout
	cmd.Run()

	// 查看 /proc/self 信息
	fmt.Printf("\n进程名: %s\n", os.Args[0])
	for _, e := range os.Environ() {
		if len(e) > 4 && e[:4] == "HOME" {
			fmt.Printf("HOME: %s\n", e[5:])
			break
		}
	}

	fmt.Println("\n提示：在另一个终端运行以下命令：")
	fmt.Printf("  cat /proc/%d/status\n", os.Getpid())
	fmt.Printf("  cat /proc/%d/maps\n", os.Getpid())
	fmt.Printf("  ls /proc/%d/fd/\n", os.Getpid())

	// 暂停，给时间观察 /proc
	fmt.Println("\n按 Enter 键退出...")
	fmt.Scanln()
}
