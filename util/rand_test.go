package util_test

import (
	"testing"

	"github.com/name5566/leaf/util"
)

func TestRandGroupDeterministic(t *testing.T) {
	// A single non-zero weight must always select its index.
	if got := util.RandGroup(0, 0, 5, 0); got != 2 {
		t.Fatalf("RandGroup single-weight: got %d, want 2", got)
	}
	// All-zero weights return index 0.
	if got := util.RandGroup(0, 0, 0); got != 0 {
		t.Fatalf("RandGroup all-zero: got %d, want 0", got)
	}
}

func TestRandGroupDistribution(t *testing.T) {
	// Weights 1:3 over many samples should land roughly 25% / 75%.
	const n = 100000
	var counts [2]int
	for i := 0; i < n; i++ {
		counts[util.RandGroup(1, 3)]++
	}
	ratio := float64(counts[0]) / float64(n)
	if ratio < 0.22 || ratio > 0.28 {
		t.Fatalf("RandGroup distribution off: index0 ratio=%.3f, want ~0.25", ratio)
	}
}

func BenchmarkRandGroup(b *testing.B) {
	weights := []uint32{10, 20, 30, 25, 15, 5, 40, 35}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = util.RandGroup(weights...)
	}
}
