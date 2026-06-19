package sqlite_test

import (
	"context"
	"testing"

	"github.com/name5566/leaf/db"
	"github.com/name5566/leaf/db/sqlite"
)

func newStore(t testing.TB) db.Store {
	t.Helper()
	s, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestNextSeq(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	for want := int64(1); want <= 5; want++ {
		got, err := s.NextSeq(ctx, "player")
		if err != nil {
			t.Fatalf("NextSeq: %v", err)
		}
		if got != want {
			t.Fatalf("NextSeq: got %d, want %d", got, want)
		}
	}

	// Independent sequences do not interfere.
	if got, _ := s.NextSeq(ctx, "room"); got != 1 {
		t.Fatalf("independent seq: got %d, want 1", got)
	}
}

func TestKV(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	if _, ok, err := s.Get(ctx, "k"); err != nil || ok {
		t.Fatalf("Get missing: ok=%v err=%v", ok, err)
	}

	if err := s.Set(ctx, "k", []byte("v1")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	v, ok, err := s.Get(ctx, "k")
	if err != nil || !ok || string(v) != "v1" {
		t.Fatalf("Get: v=%q ok=%v err=%v", v, ok, err)
	}

	// Set upserts.
	if err := s.Set(ctx, "k", []byte("v2")); err != nil {
		t.Fatalf("Set upsert: %v", err)
	}
	v, _, _ = s.Get(ctx, "k")
	if string(v) != "v2" {
		t.Fatalf("upsert: got %q, want v2", v)
	}

	if err := s.Del(ctx, "k"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	if _, ok, _ := s.Get(ctx, "k"); ok {
		t.Fatal("Get after Del: still present")
	}
	// Deleting a missing key is not an error.
	if err := s.Del(ctx, "missing"); err != nil {
		t.Fatalf("Del missing: %v", err)
	}
}

func TestIsDup(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	if _, err := s.Exec(ctx, `CREATE TABLE players (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.Exec(ctx, `INSERT INTO players (id) VALUES (?)`, 1); err != nil {
		t.Fatalf("insert: %v", err)
	}
	_, err := s.Exec(ctx, `INSERT INTO players (id) VALUES (?)`, 1)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
	if !s.IsDup(err) {
		t.Fatalf("IsDup: got false for %v", err)
	}
}

func TestRawQuery(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	var n int
	if err := s.QueryRow(ctx, `SELECT ? + ?`, 40, 2).Scan(&n); err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if n != 42 {
		t.Fatalf("QueryRow: got %d, want 42", n)
	}
}

func BenchmarkNextSeq(b *testing.B) {
	s := newStore(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.NextSeq(ctx, "bench"); err != nil {
			b.Fatal(err)
		}
	}
}
