package util_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/taills/leaf/util"
)

func TestMap(t *testing.T) {
	var m util.Map[string, int]

	if m.Len() != 0 {
		t.Fatalf("empty Len: got %d, want 0", m.Len())
	}
	if v := m.Get("missing"); v != 0 {
		t.Fatalf("Get missing: got %d, want zero value", v)
	}

	m.Set("a", 1)
	m.Set("b", 2)
	if v := m.Get("a"); v != 1 {
		t.Fatalf("Get a: got %d, want 1", v)
	}
	if m.Len() != 2 {
		t.Fatalf("Len: got %d, want 2", m.Len())
	}

	// TestAndSet returns the existing value and does not overwrite.
	if v := m.TestAndSet("a", 99); v != 1 {
		t.Fatalf("TestAndSet existing: got %d, want 1", v)
	}
	if v := m.Get("a"); v != 1 {
		t.Fatalf("TestAndSet must not overwrite: got %d, want 1", v)
	}

	m.Del("a")
	if m.Len() != 1 {
		t.Fatalf("Len after Del: got %d, want 1", m.Len())
	}

	sum := 0
	m.RLockRange(func(_ string, v int) { sum += v })
	if sum != 2 {
		t.Fatalf("RLockRange sum: got %d, want 2", sum)
	}
}

func TestMapConcurrent(t *testing.T) {
	var m util.Map[int, int]
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m.Set(i, i*i)
			_ = m.Get(i)
		}(i)
	}
	wg.Wait()
	if m.Len() != 100 {
		t.Fatalf("concurrent Len: got %d, want 100", m.Len())
	}
}

func BenchmarkMapGet(b *testing.B) {
	var m util.Map[string, int]
	for i := 0; i < 1000; i++ {
		m.Set(strconv.Itoa(i), i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = m.Get(strconv.Itoa(i % 1000))
			i++
		}
	})
}
