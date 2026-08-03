package api_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/karanjasani/agentkit/pkg/api"
)

const fixture = "../../testdata/fixtures/sample"

func newFixtureAnalyzer(t *testing.T) *api.Analyzer {
	t.Helper()
	abs, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}
	a, err := api.New(context.Background(), api.WithDir(abs))
	if err != nil {
		t.Fatalf("api.New: %v", err)
	}
	return a
}

func TestNewDefaultDir(t *testing.T) {
	// With no WithDir option, New falls back to the current working directory.
	a, err := api.New(context.Background())
	if err != nil {
		t.Fatalf("api.New (default dir): %v", err)
	}
	if a == nil {
		t.Fatal("expected analyzer")
	}
}

func TestNewInvalidDir(t *testing.T) {
	_, err := api.New(context.Background(), api.WithDir("/nonexistent/path/xyz"))
	if err == nil {
		t.Fatal("expected error for nonexistent dir")
	}
	if e := api.AsError(err); e.Code != api.CodeInvalidArgument {
		t.Errorf("code = %s, want %s", e.Code, api.CodeInvalidArgument)
	}
}

func TestSymbolNotFound(t *testing.T) {
	a := newFixtureAnalyzer(t)
	_, err := a.Symbol(context.Background(), "DoesNotExist", api.SymbolOptions{})
	if err == nil {
		t.Fatal("expected not found")
	}
	if e := api.AsError(err); e.Code != api.CodeSymbolNotFound {
		t.Errorf("code = %s", e.Code)
	}
}

func TestStructNotAStruct(t *testing.T) {
	a := newFixtureAnalyzer(t)
	// FetchWidget is a func, not a struct type.
	_, err := a.Struct(context.Background(), "FetchWidget")
	if err == nil {
		t.Fatal("expected error")
	}
	e := api.AsError(err)
	if e.Code != api.CodeTypeNotFound && e.Code != api.CodeNotAStruct {
		t.Errorf("code = %s", e.Code)
	}
}

func TestEndpointNotFound(t *testing.T) {
	a := newFixtureAnalyzer(t)
	_, err := a.Endpoint(context.Background(), "GET", "/does/not/exist")
	if err == nil {
		t.Fatal("expected not found")
	}
	if e := api.AsError(err); e.Code != api.CodeEndpointNotFound {
		t.Errorf("code = %s", e.Code)
	}
}

func TestAnalyzerCachesAcrossCalls(t *testing.T) {
	a := newFixtureAnalyzer(t)
	ctx := context.Background()
	if _, err := a.Overview(ctx); err != nil {
		t.Fatal(err)
	}
	// Second call should reuse the cached shallow load and be fast.
	start := time.Now()
	if _, err := a.Overview(ctx); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("cached overview took %v, expected fast", elapsed)
	}
}
