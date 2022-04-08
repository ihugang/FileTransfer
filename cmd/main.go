package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var mode = flag.String("mode", "server", "运行模式")
var filename = flag.String("filename", "", "要发送的文件名")
var port = flag.Int("port", 10101, "端口号")
var localIp = ""
var remoteIp = ""

// 限制goroutine数量
var limitChan = make(chan bool, 1000)

func main() {
	flag.Parse()
	fmt.Println("快捷局域网文件传输工具 V1.0, shrek@Codans设计编码")
	fmt.Println("Simple File Transfer Tool, Desined by shrek@Codans 2022")
	fmt.Println("--------------------------------------------------------")

	LOG_FILE := "log.txt"
	// open log file
	logFile, err := os.OpenFile(LOG_FILE, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer logFile.Close()

	// Set log out put and enjoy :)
	log.SetOutput(logFile)

	// optional: log date-time, filename, and line number
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	localIp = getIp()
	fmt.Println("Local IP:", localIp)
	log.Println("Local IP:", localIp)

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

			udpAddr, _ := net.ResolveUDPAddr("udp4", "0.0.0.0:10101")
			//监听端口

			udpRun := false

			udpConn, err := net.ListenUDP("udp", udpAddr)
			defer udpConn.Close()
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("udp listening ... ", "Press Ctrl+C to exit")
				//udp不需要Accept
				udpRun = true
			}

			for {
				if udpRun {
					limitChan <- true
					go handleUdpConnection(udpConn)
				}

				// 等待客户端连接
				conn, err := s.Accept()
				if err != nil {
					fmt.Println("Accept Error:", err)
					return
				} else {
					fmt.Println("tcp connected :", conn.RemoteAddr())
					// 创建一个协程处理客户端请求
					go handleClient(conn)
				}
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
			listenUdpServerIp()
			sendFile()
		}

	}
}

// 读取udp消息
func handleUdpConnection(udpConn *net.UDPConn) {
	buf := make([]byte, 1024)
	// 读取数据

	len, udpAddr, err := udpConn.ReadFromUDP(buf)
	if err != nil {
		return
	}
	logContent := strings.Replace(string(buf), "\n", "", 1)
	log.Println("udp server read data:", logContent)

	// 发送数据
	len, err = udpConn.WriteToUDP([]byte(localIp+"\r\n"), udpAddr)
	if err != nil {
		return
	}

	log.Println("udp socket", udpAddr, "write len:", len)
	<-limitChan
}

// 获得本地IP的广播地址
func getBroadcastIp(ip string) string {
	broadcastIps := strings.Split(ip, ".")
	broadcastIp := ""
	for i := 0; i < len(broadcastIps)-1; i++ {
		broadcastIp += broadcastIps[i]
		if i != len(broadcastIps)-1 {
			broadcastIp += "."
		}
	}

	broadcastIp += "255"
	return broadcastIp
}

// 侦听服务端的IP
func listenUdpServerIp() {
	broadcastIp := getBroadcastIp(localIp)
	fmt.Println("broadcast ip:", broadcastIp)
	ip := net.ParseIP(broadcastIp)

	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	dstAddr := &net.UDPAddr{IP: ip, Port: 10101}

	conn, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	fmt.Println("udp listen ok ")

	// 发送数据
	len, err := conn.WriteToUDP([]byte("where?\r\n"), dstAddr)
	if err != nil {
		return
	}
	fmt.Println("udp client write len:", len)
	time.Sleep(time.Duration(1) * time.Second)
	buf := make([]byte, 1024)
	//读取数据

	len, _ = conn.Read(buf)
	if len > 0 {
		fmt.Println("udp client read len:", len)
		remoteIp = strings.ReplaceAll(string(buf[:len]), "\r\n", "")
		fmt.Println("udp client read data:", remoteIp)
		conn.Close()
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
	log.Println("Receive:", filename)

	// 创建一个文件
	file, err := os.Create(filename)
	if err != nil {
		log.Println("Create File Error:", err)
		return
	}
	defer file.Close()
	length := 0
	// 接收文件内容
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Receive File Successfully!")
				return
			}
			log.Println("Read Error:", err)
			return
		}
		// 写入文件
		file.Write(buf[:n])
		length = length + n
		fmt.Printf("%d bytes recieved.\r", length)
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
	for {
		if len(remoteIp) > 0 {
			break
		}
	}

	addr := fmt.Sprintf("%s:%d", remoteIp, *port)
	fmt.Println("connect to:", addr)

	conn, err := net.Dial("tcp", addr)
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

	time.Sleep(time.Duration(1) * time.Second)

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
