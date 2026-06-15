package cluster

import (
	"github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/network"
	"time"
)

var (
	server  *network.TCPServer
	clients []*network.TCPClient
)

func Init() {
	if conf.ListenAddr != "" {
		server = new(network.TCPServer)
		server.Addr = conf.ListenAddr
		server.MaxConnNum = conf.MaxConnNum
		if server.MaxConnNum <= 0 {
			server.MaxConnNum = 1024
		}
		server.PendingWriteNum = conf.PendingWriteNum
		server.LenMsgLen = 4
		server.MaxMsgLen = conf.MaxMsgLen
		if server.MaxMsgLen <= 0 {
			server.MaxMsgLen = 1024 * 1024 // 1MB
		}
		server.NewAgent = newAgent

		server.Start()
	}

	for _, addr := range conf.ConnAddrs {
		client := new(network.TCPClient)
		client.Addr = addr
		client.ConnNum = 1
		client.ConnectInterval = 3 * time.Second
		client.PendingWriteNum = conf.PendingWriteNum
		client.LenMsgLen = 4
		client.MaxMsgLen = conf.MaxMsgLen
		if client.MaxMsgLen <= 0 {
			client.MaxMsgLen = 1024 * 1024 // 1MB
		}
		client.NewAgent = newAgent

		client.Start()
		clients = append(clients, client)
	}
}

func Destroy() {
	if server != nil {
		server.Close()
	}

	for _, client := range clients {
		client.Close()
	}
}

type Agent struct {
	conn *network.TCPConn
}

func newAgent(conn *network.TCPConn) network.Agent {
	a := new(Agent)
	a.conn = conn
	return a
}

func (a *Agent) Run() {}

func (a *Agent) OnClose() {}
