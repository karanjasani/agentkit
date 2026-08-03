package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/karanjasani/agentkit/pkg/models"
)

func renderToString(t *testing.T, result any) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Success(&buf, FormatText, result); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestRenderAllTypes(t *testing.T) {
	loc := models.Location{File: "a.go", Line: 10}

	cases := []struct {
		name     string
		result   any
		contains string
	}{
		{"overview", models.Overview{
			Module:      "m",
			Packages:    []models.PkgRef{{ImportPath: "m/p", Name: "p", Dir: "p"}},
			Entrypoints: []models.PkgRef{{ImportPath: "m", Name: "main", Dir: "."}},
			Stats:       models.Stats{Packages: 1, Files: 2, Entrypoints: 1},
		}, "module: m"},
		{"package", models.Package{
			ImportPath: "m/p", Name: "p", Dir: "p",
			Imports: []string{"fmt"}, ImportedBy: []string{"m"},
			Exports:   []models.Symbol{{Kind: "func", Name: "F", Location: loc}},
			TestFiles: []string{"p_test.go"},
		}, "package p"},
		{"symbol", models.Symbol{
			Name: "F", Kind: "func", Package: "m/p", Location: loc,
			Signature: "func F()", Doc: "docs", Body: "func F(){}",
			Shape:   &models.Struct{Name: "S", Fields: []models.Field{{Name: "X", Type: "int"}}},
			Callers: []models.Caller{{Location: loc, Context: "F()", Confidence: "direct"}},
		}, "func F"},
		{"callers", models.Callers{
			Symbol: "F", Package: "m/p",
			Direct:   []models.Caller{{Symbol: "G", Location: loc, Context: "F()", Confidence: "direct"}},
			Indirect: []models.Caller{{Symbol: "H", Location: loc, Confidence: "possible"}},
		}, "callers of F"},
		{"deps", models.Deps{
			Root: "m", Depth: 1, Nodes: []string{"m", "m/p"},
			Edges: []models.DepEdge{{From: "m", To: "m/p"}},
		}, "deps of m"},
		{"impact", models.Impact{
			Base: "main", RiskScore: 42, RiskLevel: "medium",
			ChangedPackages: []string{"m/p"}, AffectedPackages: []string{"m"},
			RecommendedTests: []string{"p_test.go"},
		}, "risk: 42"},
		{"tests", models.Tests{
			Symbol: "F",
			Unit:   []models.Test{{Name: "TestF", Location: loc}},
		}, "tests for F"},
		{"endpoint", models.Endpoint{
			Method: "GET", Path: "/x", Framework: "chi", Confidence: "direct",
			Route:       loc,
			Handler:     &models.Symbol{Name: "h", Location: loc},
			RequestType: "Req", ResponseType: "Resp",
			Orchestration: []models.Symbol{{Name: "svc", Location: loc}},
			Upstreams:     []models.Upstream{{Method: "GET", URL: "u", Location: loc, Confidence: "possible"}},
		}, "GET /x"},
		{"upstreams", models.Upstreams{
			Root:  ".",
			Calls: []models.Upstream{{Method: "GET", URL: "u", DecodeType: "T", Location: loc, Confidence: "direct"}},
		}, "upstreams in ."},
		{"struct", models.Struct{
			Name: "S",
			Fields: []models.Field{
				{Name: "X", JSONName: "x", Type: "int"},
				{Name: "Y", JSONName: "y", Type: "N", Optional: true, Nested: &models.Struct{Name: "N", Fields: []models.Field{{Name: "Z", Type: "int"}}}},
			},
		}, "S {"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := renderToString(t, tc.result)
			if !strings.Contains(out, tc.contains) {
				t.Errorf("text output missing %q:\n%s", tc.contains, out)
			}
		})
	}
}

func TestRenderUnknownType(t *testing.T) {
	out := renderToString(t, struct{ X int }{X: 1})
	if out == "" {
		t.Error("expected fallback output for unknown type")
	}
}
