package api_test

import (
	"context"
	"testing"

	"github.com/karanjasani/agentkit/pkg/api"
)

func TestSymbolDocAndShape(t *testing.T) {
	a := newFixtureAnalyzer(t)
	ctx := context.Background()

	doc, err := a.Symbol(ctx, "FetchWidget", api.SymbolOptions{Doc: true})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Doc == "" {
		t.Error("expected doc comment for FetchWidget")
	}
	if doc.Body != "" {
		t.Error("doc-only lookup should not include body")
	}

	shape, err := a.Symbol(ctx, "models.Widget", api.SymbolOptions{Shape: true})
	if err != nil {
		t.Fatal(err)
	}
	if shape.Shape == nil || len(shape.Shape.Fields) == 0 {
		t.Fatal("expected struct shape for Widget")
	}
}

func TestPackageByDirectory(t *testing.T) {
	a := newFixtureAnalyzer(t)
	pkg, err := a.Package(context.Background(), "./service")
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Name != "service" {
		t.Errorf("name = %q, want service", pkg.Name)
	}
	foundExport := false
	for _, s := range pkg.Exports {
		if s.Name == "FetchWidget" {
			foundExport = true
		}
	}
	if !foundExport {
		t.Error("expected FetchWidget in exports")
	}
	if len(pkg.TestFiles) == 0 {
		t.Error("expected test files listed")
	}
}

func TestStructNestedShape(t *testing.T) {
	a := newFixtureAnalyzer(t)
	st, err := a.Struct(context.Background(), "models.Widget")
	if err != nil {
		t.Fatal(err)
	}
	var owner *string
	for _, f := range st.Fields {
		if f.JSONName == "owner" {
			if f.Nested == nil {
				t.Fatal("owner field should have nested shape")
			}
			name := f.Nested.Name
			owner = &name
		}
		if f.Name == "secret" {
			t.Error("unexported field should be omitted from shape")
		}
	}
	if owner == nil || *owner != "Owner" {
		t.Error("expected nested Owner struct")
	}
}

func TestCallersOfHelper(t *testing.T) {
	a := newFixtureAnalyzer(t)
	c, err := a.Callers(context.Background(), "Helper")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range c.Direct {
		if d.Symbol == "UsesHelper" {
			found = true
			if d.Confidence != "direct" {
				t.Errorf("expected direct confidence, got %s", d.Confidence)
			}
		}
	}
	if !found {
		t.Error("expected UsesHelper among direct callers of Helper")
	}
}

func TestTestsClassification(t *testing.T) {
	a := newFixtureAnalyzer(t)
	ts, err := a.Tests(context.Background(), "Helper")
	if err != nil {
		t.Fatal(err)
	}
	// Helper is used by TestHelper (unit) and BenchmarkHelper (benchmark).
	if len(ts.Unit) == 0 {
		t.Error("expected at least one unit test")
	}
	if len(ts.Benchmark) == 0 {
		t.Error("expected at least one benchmark")
	}
}

func TestDepsFromRoot(t *testing.T) {
	a := newFixtureAnalyzer(t)
	d, err := a.Deps(context.Background(), "example.com/sample")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Nodes) < 2 {
		t.Errorf("expected multiple nodes, got %d", len(d.Nodes))
	}
	if d.Depth < 1 {
		t.Errorf("expected depth >= 1, got %d", d.Depth)
	}
}

func TestUpstreamsByDir(t *testing.T) {
	a := newFixtureAnalyzer(t)
	u, err := a.Upstreams(context.Background(), "./service")
	if err != nil {
		t.Fatal(err)
	}
	if len(u.Calls) == 0 {
		t.Fatal("expected at least one upstream call")
	}
	if u.Calls[0].Method != "GET" {
		t.Errorf("method = %s, want GET", u.Calls[0].Method)
	}
}
