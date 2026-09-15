package main

import (
	"fmt"
	"io"
	"net"
	"sync"
)

type Server struct {
	Ip   string
	Port int

	// 在线用户的列表
	OnlineMap map[string]*User
	maplock sync.RWMutex

	// 消息传播的channel
	Message chan string
}

// 创建一个server接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:   ip,
		Port: port,
		OnlineMap: make(map[string]*User),
		Message: make(chan string),
	}

	return server
}

// 启动服务器的接口
func (this *Server) Start() {
	// socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}

	// close listen socket
	defer listener.Close()

	// 启动监听Message的goroutine
	go this.ListenMessage()

	for {
		// accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}
			
		// do handler
		go this.Handler(conn)
	}
}

func (this *Server) Handler(conn net.Conn) {
	// fmt.Println("连接建立成功")

	user := NewUser(conn, this)

	user.Online()

	// 接受客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}

			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err:", err)
				return
			}

			// 提取用户的消息（去换行符）
			msg := string(buf[:n-1])

			// 将得到的消息进行广播
			user.HandleMessage(msg)
		}
	}()

	// 当前Handler阻塞
	select{}

}

// 监听message广播消息channel的goroutine，一旦有消息就发送给全部的在线user
func (this *Server) ListenMessage() {
	for {
		msg := <- this.Message

		// 将msg发送到所有在线的user
		this.maplock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.maplock.Unlock()
	}
}

// 将信息发送到message管道
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg

	this.Message <- sendMsg
}