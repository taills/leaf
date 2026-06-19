package network

import (
	"net"
	"testing"
	"time"
)

// nopAgent 是用于测试的最小 Agent 实现。
type nopAgent struct {
	conn *TCPConn
}

func (a *nopAgent) Run()     { a.conn.ReadMsg() } // 阻塞直到连接关闭
func (a *nopAgent) OnClose() {}

// TestTCPServerCloseDoesNotHang 回归测试：监听器被 Close 后，run() 的 accept
// 循环应及时退出，Close 不应永久阻塞（此前因 accept 错误无限重试而挂起）。
func TestTCPServerCloseDoesNotHang(t *testing.T) {
	server := &TCPServer{
		Addr:            "127.0.0.1:0",
		MaxConnNum:      10,
		PendingWriteNum: 10,
		LenMsgLen:       2,
		MaxMsgLen:       4096,
		NewAgent:        func(c *TCPConn) Agent { return &nopAgent{conn: c} },
	}
	server.init()
	go server.run()

	// 确认正常接受连接。
	addr := server.ln.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	conn.Close()

	// Close 必须在合理时间内返回。
	done := make(chan struct{})
	go func() {
		server.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("TCPServer.Close hung: accept loop did not exit on closed listener")
	}
}
