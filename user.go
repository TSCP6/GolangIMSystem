package main

import (
	"net"
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

// 用户处理消息业务
func (this *User) HandleMessage(msg string) {
	this.server.BroadCast(this, msg)
}

// 监听当前user channel的方法，一旦有消息立即发送到客户端
func (this *User) ListenMessage() {
	for {
		msg := <- this.C

		this.conn.Write([]byte(msg + "\n"))
	}
}