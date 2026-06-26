package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	_ "net/http/pprof"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	connections int64
	bytesRead   int64
)

func main() {
	port := flag.Int("port", 8080, "监听端口")
	flag.Parse()

	go func() {
		for {
			time.Sleep(5 * time.Second)
			fmt.Fprintf(os.Stderr,
				"连接数: %d | 字节数: %d | Goroutines: %d\n",
				atomic.LoadInt64(&connections),
				atomic.LoadInt64(&bytesRead),
				runtime.NumGoroutine())
		}
	}()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Printf("Echo 服务器启动在 :%d\n", *port)
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("PID: %d\n", os.Getpid())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		atomic.AddInt64(&connections, 1)

		// 每个连接一个 goroutine — 底层用的就是 epoll！
		go func(c net.Conn) {
			defer func() {
				c.Close()
				atomic.AddInt64(&connections, -1)
			}()

			buf := make([]byte, 4096)
			for {
				n, err := c.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Println(err)
					}
					return
				}
				atomic.AddInt64(&bytesRead, int64(n))
				// Echo back
				c.Write(buf[:n])
			}
		}(conn)
	}
}
