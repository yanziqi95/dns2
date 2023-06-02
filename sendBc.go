package main

//func sendBc(ip string) {
//	// 打开要发送的文件
//	file, err := os.Open(dbFile)
//	if err != nil {
//		fmt.Println("无法打开文件:", err)
//		return
//	}
//	defer file.Close()
//
//	// 建立TCP连接
//	conn, err := net.Dial("tcp", ip+":9888")
//	if err != nil {
//		fmt.Println("无法建立TCP连接:", err)
//		return
//	}
//	defer conn.Close()
//
//	// 创建缓冲区
//	buffer := make([]byte, 1024)
//
//	// 逐块读取文件并发送
//	for {
//		// 读取文件数据到缓冲区
//		n, err := file.Read(buffer)
//		if err != nil {
//			if err != io.EOF {
//				fmt.Println("读取文件错误:", err)
//			}
//			break
//		}
//
//		// 发送数据块到服务器
//		_, err = conn.Write(buffer[:n])
//		if err != nil {
//			fmt.Println("发送数据错误:", err)
//			return
//		}
//	}
//
//	fmt.Println("文件发送完成")
//}
