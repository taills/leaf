package network

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// memConn is an in-memory net.Conn whose reads are served from r and whose
// writes are discarded. It is used to drive MsgParser without real sockets.
type memConn struct {
	r *bytes.Reader
}

func (c *memConn) Read(b []byte) (int, error)         { return c.r.Read(b) }
func (c *memConn) Write(b []byte) (int, error)        { return len(b), nil }
func (c *memConn) Close() error                       { return nil }
func (c *memConn) LocalAddr() net.Addr                { return nil }
func (c *memConn) RemoteAddr() net.Addr               { return nil }
func (c *memConn) SetDeadline(t time.Time) error      { return nil }
func (c *memConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *memConn) SetWriteDeadline(t time.Time) error { return nil }

// buildFrames concatenates n length-prefixed (2-byte big-endian) frames.
func buildFrames(n int, payload []byte) []byte {
	var buf bytes.Buffer
	var hdr [2]byte
	for i := 0; i < n; i++ {
		binary.BigEndian.PutUint16(hdr[:], uint16(len(payload)))
		buf.Write(hdr[:])
		buf.Write(payload)
	}
	return buf.Bytes()
}

func TestMsgParserRoundTrip(t *testing.T) {
	p := NewMsgParser()
	payload := []byte("hello leaf framing")

	// Write frames the payload and enqueues it on the write channel.
	wc := &TCPConn{conn: &memConn{}, writeChan: make(chan []byte, 1)}
	if err := p.Write(wc, payload); err != nil {
		t.Fatalf("Write: %v", err)
	}
	framed := <-wc.writeChan
	if len(framed) != len(payload)+2 {
		t.Fatalf("framed len: got %d, want %d", len(framed), len(payload)+2)
	}

	// Read parses the framed bytes back into the original payload.
	rc := &TCPConn{conn: &memConn{r: bytes.NewReader(framed)}, msgParser: p}
	got, err := p.Read(rc)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("round trip mismatch: got %q, want %q", got, payload)
	}
}

func TestMsgParserTooLong(t *testing.T) {
	p := NewMsgParser()
	p.SetMsgLen(2, 1, 8)
	wc := &TCPConn{conn: &memConn{}, writeChan: make(chan []byte, 1)}
	if err := p.Write(wc, bytes.Repeat([]byte("x"), 9)); err == nil {
		t.Fatal("expected error for over-long message")
	}
}

func BenchmarkMsgParserRead(b *testing.B) {
	p := NewMsgParser()
	payload := bytes.Repeat([]byte("x"), 128)
	frames := buildFrames(b.N, payload)
	conn := &TCPConn{conn: &memConn{r: bytes.NewReader(frames)}, msgParser: p}

	b.SetBytes(int64(len(payload) + 2))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.Read(conn); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMsgParserWrite(b *testing.B) {
	p := NewMsgParser()
	payload := bytes.Repeat([]byte("x"), 128)
	// Large buffer so doWrite never reports the channel as full during the run.
	conn := &TCPConn{conn: &memConn{}, writeChan: make(chan []byte, b.N+1)}

	b.SetBytes(int64(len(payload) + 2))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := p.Write(conn, payload); err != nil {
			b.Fatal(err)
		}
	}
}
