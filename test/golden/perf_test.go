package golden

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/karanjasani/agentkit/pkg/api"
)

// TestPerformanceBudget asserts that a metadata-only command completes well
// within the small-repository budget. It establishes the harness that would be
// pointed at a large synthetic module in CI; here it guards against gross
// regressions on the fixture.
func TestPerformanceBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in -short mode")
	}
	abs, _ := filepath.Abs(fixture)
	ctx := context.Background()

	a, err := api.New(ctx, api.WithDir(abs))
	if err != nil {
		t.Fatal(err)
	}
	// Warm the load once, then measure a cached metadata query.
	if _, err := a.Overview(ctx); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := a.Overview(ctx); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Errorf("cached overview took %v, budget 100ms", d)
	}
}
