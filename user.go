package main

import (
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C    chan string
	conn net.Conn

	server *Server
}


// 创建一个用户的API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name: userAddr,
		Addr: userAddr,
		C:    make(chan string),
		conn: conn,

		server : server,
	}

	// 启动监听当前user channel的goroutine
	go user.ListenMessage()

	return user 
}

// 用户上线业务
func (this *User) Online() {
	// 将用户加入到OnlineMap中
	this.server.maplock.Lock()
	this.server.OnlineMap[this.Name] = this
	this.server.maplock.Unlock()

	// 广播当前用户上线信息
	this.server.BroadCast(this, "已上线")
}

// 用户下线业务
func (this *User) Offline() {
	// 将用户从OnlineMap中删除
	this.server.maplock.Lock()
	delete(this.server.OnlineMap, this.Name)
	this.server.maplock.Unlock()
	
	// 广播当前用户上线信息
	this.server.BroadCast(this, "下线")
}

func (this *User) SendMessage(msg string){
	this.conn.Write([]byte(msg))
}

// 用户处理消息业务
func (this *User) HandleMessage(msg string) {
	if msg == "who"{
		// 查询当前的在线用户
		this.server.maplock.Lock()
		for _, user := range this.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线...\n"
			this.SendMessage(onlineMsg)
		}
		this.server.maplock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		// 消息格式: rename|your_name
		newName := strings.Split(msg, "|")[1]

		// 判断name是否存在
		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.SendMessage("当前用户名已被使用\n")
		} else {
			this.server.maplock.Lock()
			delete(this.server.OnlineMap, this.Name)
			this.server.OnlineMap[newName] = this
			this.server.maplock.Unlock()

			this.Name = newName
			this.SendMessage("已更新用户名:" + this.Name + "\n")
		}
	} else {
		this.server.BroadCast(this, msg)
	}
}

// 监听当前user channel的方法，一旦有消息立即发送到客户端
func (this *User) ListenMessage() {
	for {
		msg := <- this.C

		this.conn.Write([]byte(msg + "\n"))
	}
}