package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

var mode = flag.String("mode", "server", "运行模式")
var filename = flag.String("filename", "", "要发送的文件名")
var port = flag.Int("port", 10101, "端口号")

func main() {
	flag.Parse()
	fmt.Println("Simple File Transfer Tool, Desined by shrek@Codans 2022")
	fmt.Println("--------------------------------------------------------")
	localIp := getIp()
	fmt.Println("Local IP:", localIp)
	if *mode == "server" {
		fmt.Println("Server Mode...")

		// 创建一个监听
		s, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
		defer s.Close()

		if err != nil {
			fmt.Println("Listen Error:", err)
			return
		} else {
			fmt.Println("Server Start Successfully!")
			for {
				// 等待客户端连接
				conn, err := s.Accept()
				if err != nil {
					fmt.Println("Accept Error:", err)
					return
				}
				// 创建一个协程处理客户端请求
				go handleClient(conn)
			}
		}
	} else {
		fmt.Println("Client Mode...")
		if *filename == "" {
			fmt.Println("请指定要发送的文件名")
			fmt.Println("-filename:", *filename)
			return
		} else {
			fmt.Println("filename:", *filename)
			sendFile()
		}

	}
}

// tcp handle
func handleClient(conn net.Conn) {
	defer conn.Close()
	// 读取客户端发送的数据
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Read Error:", err)
		return
	}
	filename := string(buf[:n])
	filename = filepath.Base(filename)
	fmt.Println("Receive:", filename)
	// 创建一个文件
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Create File Error:", err)
		return
	}
	defer file.Close()
	// 接收文件内容
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Receive File Successfully!")
				return
			}
			fmt.Println("Read Error:", err)
			return
		}
		// 写入文件
		file.Write(buf[:n])
	}
}

func getIp() string {
	localIp := GetLocalNicIP()

	if localIp != "" {
		return localIp
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, i := range ifaces {
		stdIf := &i
		if !IsUp(stdIf) || !IsBroadcast(stdIf) || IsProblematicInterface(stdIf) || IsLoopback(stdIf) {
			// Skip down interfaces and ones that are
			// problematic that we don't want to try to
			// send Tailscale traffic over.
			continue
		}

		fmt.Println("Interface:", stdIf.Name)

		addrs, err := i.Addrs()
		if err != nil {
			return ""
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil {

			}

			//fmt.Println("ip:", ip.To4().String(), addr.String())
			if isPrivateIP(ip) {
				return ip.To4().String()
			}
		}
	}

	return ""
}

func isPrivateIP(ip net.IP) bool {
	var privateIPBlocks []*net.IPNet
	for _, cidr := range []string{
		// don't check loopback ips
		//"127.0.0.0/8",    // IPv4 loopback
		//"::1/128",        // IPv6 loopback
		//"fe80::/10",      // IPv6 link-local
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

// client 发送文件
func sendFile() {
	// 创建一个连接
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", *port))
	defer conn.Close()
	if err != nil {
		fmt.Println("Dial Error:", err)
		return
	}
	// 发送文件名
	_, err = conn.Write([]byte(*filename))
	if err != nil {
		fmt.Println("Write Error:", err)
		return
	}

	var fileSize int64 = 0
	fi, err := os.Stat(*filename)
	if err == nil {
		fileSize = fi.Size()
		fmt.Println("file size is ", fileSize)
	} else {
		fmt.Println("File Does Not Exist!")
		return
	}

	// 发送文件内容
	file, err := os.Open(*filename)
	if err != nil {
		fmt.Println("Open File Error:", err)
		return
	}
	defer file.Close()
	buf := make([]byte, 1024)
	length := 0
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Send File Successfully!")
				return
			}
			fmt.Println("Read File Error:", err)
			return
		}

		length = length + n
		fmt.Fprintf(os.Stdout, "%d/%d %d%%\r", length, fileSize, int64(length*100)/fileSize)
		_, err = conn.Write(buf[:n])
		if err != nil {
			fmt.Println("Write File Error:", err)
			return
		}
	}
}
