package service

import "testing"

// TestFetchWidget exercises FetchWidget (transitively covers Helper via nothing,
// but reaches the upstream call path).
func TestFetchWidget(t *testing.T) {
	_ = FetchWidget("1")
}

// TestHelper exercises Helper directly.
func TestHelper(t *testing.T) {
	if Helper() == "" {
		t.Fatal("expected non-empty")
	}
}

// BenchmarkHelper benchmarks Helper.
func BenchmarkHelper(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Helper()
	}
}
