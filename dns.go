package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

//func handleStatus(conn net.Conn) string {
//	defer conn.Close()
//
//	// 创建缓冲区
//	buffer := make([]byte, 1024)
//
//	// 读取接收到的文字数据
//	n, err := conn.Read(buffer)
//	if err != nil {
//		fmt.Println("读取数据错误:", err)
//	}
//	// 提取文字内容
//	status := string(buffer[:n])
//	fmt.Println("接收到的文字:", status)
//	return status
//}
//
//func dns() {
//	listener, err := net.Listen("tcp", ":9888")
//	if err != nil {
//		fmt.Println("无法监听端口:", err)
//	}
//	defer listener.Close()
//
//	fmt.Println("服务器已启动，等待连接...")
//	conn, err := listener.Accept()
//	if err != nil {
//		fmt.Println("接受连接错误:", err)
//	}
//	//接受状态码，如果是同步区块链请求，则发送给该ip
//	info := handleStatus(conn)
//	status := info[0:3]
//	ip := info[3:]
//	fmt.Printf(ip)
//	if status == "bal" {
//		//返回余额
//		bal := getBalance(ip)
//		_, err = conn.Write([]byte(string(bal)))
//		if err != nil {
//			fmt.Println("Error sending response:", err.Error())
//			return
//		}
//		fmt.Printf("Responded %x balance request succed", status[3:])
//	}
//	if status == "syn" {
//		//同步区块链
//		sendBc(ip)
//	}
//	if status == "upd" {
//		//收到来自节点的更新请求
//		recvBc()
//	}
//}

func handleData(conn net.Conn) {
	// 读取状态码
	statusCodeBuffer := make([]byte, 3)
	_, err := conn.Read(statusCodeBuffer)
	if err != nil {
		fmt.Println("Error reading status code:", err.Error())
		return
	}
	statusCode := string(statusCodeBuffer)

	// 根据状态码接收不同类型的数据
	switch statusCode {
	case "bal":
		// 接收字符串类型的数据
		dataBuffer := make([]byte, 1024)
		n, err := conn.Read(dataBuffer)
		if err != nil {
			fmt.Println("Error reading data:", err.Error())
			return
		}
		data := string(dataBuffer[:n])
		fmt.Printf("Requesting balance:%x", data)

		bal := getBalance(data)
		_, err = conn.Write([]byte(string(bal)))
		if err != nil {
			fmt.Println("Error sending response:", err.Error())
			return
		}
		fmt.Printf("Responded %x balance request: %x", data, bal)

	case "syn":
		file, err := os.Open(dbFile)
		if err != nil {
			fmt.Println("Error opening Bolt file:", err)
			return
		}
		defer file.Close()

		_, err = conn.Write([]byte("002\n"))
		if err != nil {
			fmt.Println("Error sending status code:", err)
			return
		}

		_, err = io.Copy(conn, file)
		if err != nil {
			fmt.Println("Error sending Bolt file:", err)
			return
		}

	case "upd":
		file, err := os.Create(dbFile)
		if err != nil {
			fmt.Println("Error creating file:", err.Error())
			return
		}
		defer file.Close()
		for {
			fileBuffer := make([]byte, 1024)
			n, err := conn.Read(fileBuffer)
			if err != nil {
				if err != io.EOF {
					fmt.Println("Error reading file data:", err.Error())
				}
				break
			}

			_, err = file.Write(fileBuffer[:n])
			if err != nil {
				fmt.Println("Error writing file data:", err.Error())
				return
			}
		}
		fmt.Println("Bolt file data received successfully")

	default:
		fmt.Println("Invalid status code:", statusCode)
	}
}

func dns() {
	listener, err := net.Listen("tcp", ":9888")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer listener.Close()

	fmt.Println("Server listening on port : 9888")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			return
		}

		// 在新的goroutine中处理客户端连接
		go handleData(conn)
	}
}
