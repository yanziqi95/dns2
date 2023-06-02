package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func recvBc() {
	// 监听TCP连接
	listener, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("无法监听端口:", err)
		return
	}
	defer listener.Close()

	fmt.Println("服务器已启动，等待连接...")

	// 接受连接并处理
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("接受连接错误:", err)
			continue
		}

		// 启动goroutine处理连接
		go handler_file(conn)
	}
}

func handler_file(conn net.Conn) {
	defer conn.Close()

	// 创建保存数据库文件的目标文件
	file, err := os.Create(dbFile)
	if err != nil {
		fmt.Println("无法创建文件:", err)
		return
	}
	defer file.Close()

	// 创建缓冲区
	buffer := make([]byte, 1024)

	// 逐块接收数据并写入文件
	for {
		// 从连接中读取数据块
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Println("读取数据错误:", err)
			}
			break
		}

		// 将数据块写入文件
		_, err = file.Write(buffer[:n])
		if err != nil {
			fmt.Println("写入文件错误:", err)
			return
		}
	}

	fmt.Println("数据库文件接收完成")
}
