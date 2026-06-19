package chanrpc

import (
	"testing"
)

// runServer starts a goroutine that drains the server's ChanCall until it is
// closed, returning a stop function.
func runServer(s *Server) (stop func()) {
	done := make(chan struct{})
	go func() {
		for ci := range s.ChanCall {
			s.Exec(ci)
		}
		close(done)
	}()
	return func() {
		s.Close()
		<-done
	}
}

func TestCall(t *testing.T) {
	s := NewServer(10)
	s.Register("add", func(args []interface{}) interface{} {
		return args[0].(int) + args[1].(int)
	})
	s.Register("noop", func(args []interface{}) {})
	s.Register("pair", func(args []interface{}) []interface{} {
		return []interface{}{args[0], args[1]}
	})
	stop := runServer(s)
	defer stop()

	c := s.Open(10)

	if err := c.Call0("noop"); err != nil {
		t.Fatalf("Call0: %v", err)
	}

	r, err := c.Call1("add", 2, 3)
	if err != nil {
		t.Fatalf("Call1: %v", err)
	}
	if r.(int) != 5 {
		t.Fatalf("Call1 add: got %v, want 5", r)
	}

	rs, err := c.CallN("pair", 7, 8)
	if err != nil {
		t.Fatalf("CallN: %v", err)
	}
	if len(rs) != 2 || rs[0].(int) != 7 || rs[1].(int) != 8 {
		t.Fatalf("CallN pair: got %v, want [7 8]", rs)
	}
}

func TestCallUnregistered(t *testing.T) {
	s := NewServer(10)
	stop := runServer(s)
	defer stop()

	c := s.Open(10)
	if _, err := c.Call1("missing", 1); err == nil {
		t.Fatal("expected error for unregistered function")
	}
}

func BenchmarkCall1(b *testing.B) {
	s := NewServer(100)
	s.Register("add", func(args []interface{}) interface{} {
		return args[0].(int) + args[1].(int)
	})
	stop := runServer(s)
	defer stop()

	c := s.Open(100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.Call1("add", 1, 2); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGo(b *testing.B) {
	s := NewServer(100)
	done := make(chan struct{}, 1)
	s.Register("ping", func(args []interface{}) {
		select {
		case done <- struct{}{}:
		default:
		}
	})
	stop := runServer(s)
	defer stop()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Go("ping", i)
	}
}
