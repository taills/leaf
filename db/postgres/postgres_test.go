package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/name5566/leaf/db"
	"github.com/name5566/leaf/db/postgres"
)

// These tests require a real PostgreSQL instance. Set LEAF_PG_DSN, e.g.
//
//	LEAF_PG_DSN="postgres://leaf:leaf@localhost:5432/leaf?sslmode=disable" go test ./db/postgres/...
//
// When the variable is unset, the tests are skipped so CI stays green without
// a database.
func newStore(t testing.TB) db.Store {
	t.Helper()
	dsn := os.Getenv("LEAF_PG_DSN")
	if dsn == "" {
		t.Skip("LEAF_PG_DSN not set; skipping PostgreSQL integration test")
	}
	s, err := postgres.Open(dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestNextSeq(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	first, err := s.NextSeq(ctx, "pg_test_seq")
	if err != nil {
		t.Fatalf("NextSeq: %v", err)
	}
	second, err := s.NextSeq(ctx, "pg_test_seq")
	if err != nil {
		t.Fatalf("NextSeq: %v", err)
	}
	if second != first+1 {
		t.Fatalf("NextSeq not monotonic: %d then %d", first, second)
	}
}

func TestKV(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	key := "pg_test_kv"
	if err := s.Set(ctx, key, []byte("v1")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	v, ok, err := s.Get(ctx, key)
	if err != nil || !ok || string(v) != "v1" {
		t.Fatalf("Get: v=%q ok=%v err=%v", v, ok, err)
	}
	if err := s.Del(ctx, key); err != nil {
		t.Fatalf("Del: %v", err)
	}
}
