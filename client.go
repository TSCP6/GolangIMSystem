package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

type Client struct {
	ServerIp string
	ServerPort int
	Name string
	conn net.Conn
	flag int // 当前client模式

	// 控制是否允许把服务器返回的消息打印到屏幕
	// 因为 DealServerResponse 运行在单独的 goroutine 中，需要用读写锁保护
	printMu sync.RWMutex
	printEnabled bool
}

func NewClient(serverIp string, serverPort int) *Client {
	// 创建客户端对象
	client := &Client{
		ServerIp: serverIp,
		ServerPort: serverPort,
		flag: 666,
	}

	// 链接server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", serverIp,serverPort))
	if err != nil {
		fmt.Println("net.Dial error:", err)
		return nil
	}

	client.conn = conn

	// 返回对象
	return client
}

func (client *Client) menu() bool {
	var flag int

	fmt.Println("1.公聊模式")
	fmt.Println("2.私聊模式")
	fmt.Println("3.更新用户名")
	fmt.Println("0.退出")

	// 检查输入，避免非法输入(非数字)时 flag 保持为 0 被误当成"退出"
	if _, err := fmt.Scanln(&flag); err != nil {
		fmt.Println("---请输入合法的数字---")
		return false
	}

	if flag >= 0 && flag <= 3 {
		client.flag = flag
		return true
	} else {
		fmt.Println("---请输入合法的数字---")
		return false
	}
}

// 设置是否允许把服务器消息打印到屏幕
func (client *Client) SetPrint(enabled bool) {
	client.printMu.Lock()
	defer client.printMu.Unlock()
	client.printEnabled = enabled
}

// 查询当前是否允许把服务器消息打印到屏幕
func (client *Client) CanPrint() bool {
	client.printMu.RLock()
	defer client.printMu.RUnlock()
	return client.printEnabled
}

// 处理server回应消息，仅在特定模式下才显示到标准输出
func (client *Client) DealServerResponse() {
	buf := make([]byte, 4096)
	for {
		// 永久监听client.conn的数据
		n, err := client.conn.Read(buf)
		if err != nil {
			fmt.Println("conn.Read err:", err)
			return
		}

		msg := string(buf[:n])
		// 私聊消息无视模式开关，始终显示；其他消息(公聊广播/who/改名等)
		// 仅在允许打印的模式下输出，菜单等其他状态下只读取不打印，避免干扰输入
		if strings.HasPrefix(msg, "[私聊]") || client.CanPrint() {
			os.Stdout.Write(buf[:n])
		}
	}
}

func (client *Client) PublicChat() {
	// 提示用户输入消息
	var chatMsg string

	fmt.Println("请输入聊天内容, exit退出.")
	fmt.Scanln(&chatMsg)

	for chatMsg != "exit" {
		// 消息不为空就发送
		if len(chatMsg) != 0 {
			msg := chatMsg + "\n"
			_, err := client.conn.Write([]byte(msg))
			if err != nil {
				fmt.Println("conn.Write err:", err)
				break
			}
		}

		chatMsg = ""
		fmt.Println("请输入聊天内容, exit退出.")
		fmt.Scanln(&chatMsg)
	}
}

func (client *Client) SelectOnlineUsers() {
	msg := "who\n"
	_, err := client.conn.Write([]byte(msg))
	if err != nil {
		fmt.Println("conn.Write err:", err)
		return
	}
}

func (client *Client) PrivateChat() {
	var remoteName string
	var chatMsg string

	client.SelectOnlineUsers()
	fmt.Println("请输入聊天对象[用户名], exit退出:")
	fmt.Scanln(&remoteName)

	for remoteName != "exit" {
		fmt.Println("请输入聊天内容, exit退出.")
		fmt.Scanln(&chatMsg)

		for chatMsg != "exit" {
			// 消息不为空就发送
			if len(chatMsg) != 0 {
				msg := "to|" + remoteName + "|" + chatMsg + "\n"
				_, err := client.conn.Write([]byte(msg))
				if err != nil {
					fmt.Println("conn.Write err:", err)
					break
				}
			}

			chatMsg = ""
			fmt.Println("请输入聊天内容, exit退出.")
			fmt.Scanln(&chatMsg)
		}

		client.SelectOnlineUsers()
		fmt.Println("请输入聊天对象[用户名], exit退出:")
		fmt.Scanln(&remoteName)
	}
}

func (client *Client) UpdateName() bool {
	fmt.Println("请输入用户名:")
	fmt.Scanln(&client.Name)

	msg := "rename|" + client.Name + "\n"
	_, err := client.conn.Write([]byte(msg))
	if err != nil {
		fmt.Println("conn.Write err:", err)
		return false
	}
	return true
}



func (client *Client) Run() {
	for client.flag != 0 {
		// 处于菜单状态，关闭打印，避免服务器广播被输出到屏幕干扰输入
		client.SetPrint(false)

		if !client.menu() {
			// 输入不合法，重新显示菜单
			continue
		}

		// 根据不同的模式处理不同业务
		switch client.flag {
		case 1 :
			// 进入公聊模式，允许打印服务器消息
			client.SetPrint(true)
			client.PublicChat()
		case 2 :
			// 进入私聊模式，允许打印服务器消息
			client.SetPrint(true)
			client.PrivateChat()
		case 3 :
			// 进入改名模式，允许打印服务器消息
			client.SetPrint(true)
			client.UpdateName()
		}
	}
}

var serverIp string
var serverPort int

func init() {
	flag.StringVar(&serverIp, "ip", "127.0.0.1", "设置服务器IP地址(默认是127.0.0.1)")
	flag.IntVar(&serverPort, "port", 8888, "设置服务器端口(默认是8888)")
}

func main() {
	// 命令行解析
	flag.Parse()

	client := NewClient(serverIp, serverPort)
	if client == nil {
		fmt.Println("---服务器连接失败---")
		return 
	}

	// 单独开启一个goroutine处理server返回消息
	go client.DealServerResponse()

	fmt.Println("---服务器连接成功---")

	// 启动客户端的业务
	client.Run()
}