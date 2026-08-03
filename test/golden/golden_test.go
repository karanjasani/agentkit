// Package golden runs the public API against a fixture module and compares the
// JSON-marshaled results against stored golden files. Run with -update to
// regenerate the golden files after an intentional change.
package golden

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/karanjasani/agentkit/pkg/api"
)

var update = flag.Bool("update", false, "update golden files")

const fixture = "../../testdata/fixtures/sample"

type testCase struct {
	name string
	run  func(ctx context.Context, a *api.Analyzer) (any, error)
}

func cases() []testCase {
	return []testCase{
		{"overview", func(ctx context.Context, a *api.Analyzer) (any, error) { return a.Overview(ctx) }},
		{"package_service", func(ctx context.Context, a *api.Analyzer) (any, error) {
			return a.Package(ctx, "example.com/sample/service")
		}},
		{"symbol_signature", func(ctx context.Context, a *api.Analyzer) (any, error) {
			return a.Symbol(ctx, "FetchWidget", api.SymbolOptions{SignatureOnly: true})
		}},
		{"symbol_body", func(ctx context.Context, a *api.Analyzer) (any, error) {
			return a.Symbol(ctx, "Helper", api.SymbolOptions{Body: true})
		}},
		{"struct_widget", func(ctx context.Context, a *api.Analyzer) (any, error) {
			return a.Struct(ctx, "models.Widget")
		}},
		{"deps", func(ctx context.Context, a *api.Analyzer) (any, error) {
			return a.Deps(ctx, "example.com/sample")
		}},
		{"callers_helper", func(ctx context.Context, a *api.Analyzer) (any, error) {
			return a.Callers(ctx, "Helper")
		}},
		{"tests_fetchwidget", func(ctx context.Context, a *api.Analyzer) (any, error) {
			return a.Tests(ctx, "FetchWidget")
		}},
		{"tests_helper", func(ctx context.Context, a *api.Analyzer) (any, error) {
			return a.Tests(ctx, "Helper")
		}},
		{"endpoint", func(ctx context.Context, a *api.Analyzer) (any, error) {
			return a.Endpoint(ctx, "GET", "/api/v1/widgets/{id}")
		}},
		{"upstreams", func(ctx context.Context, a *api.Analyzer) (any, error) {
			return a.Upstreams(ctx, ".")
		}},
	}
}

func TestGolden(t *testing.T) {
	absFixture, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatalf("resolving fixture path: %v", err)
	}
	if _, err := os.Stat(absFixture); err != nil {
		t.Fatalf("fixture module not found at %s: %v", absFixture, err)
	}

	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			a, err := api.New(ctx, api.WithDir(absFixture))
			if err != nil {
				t.Fatalf("api.New: %v", err)
			}
			result, err := tc.run(ctx, a)
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			got := mustMarshal(t, result)

			goldenPath := filepath.Join("testdata", tc.name+".json")
			if *update {
				writeGolden(t, goldenPath, got)
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("reading golden %s (run with -update): %v", goldenPath, err)
			}
			if !bytes.Equal(normalize(got), normalize(want)) {
				t.Errorf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", tc.name, got, want)
			}
		})
	}
}

// TestDeterminism runs each case twice and asserts byte-identical output.
func TestDeterminism(t *testing.T) {
	absFixture, _ := filepath.Abs(fixture)
	ctx := context.Background()
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			a1, _ := api.New(ctx, api.WithDir(absFixture))
			r1, err := tc.run(ctx, a1)
			if err != nil {
				t.Fatalf("run1: %v", err)
			}
			a2, _ := api.New(ctx, api.WithDir(absFixture))
			r2, err := tc.run(ctx, a2)
			if err != nil {
				t.Fatalf("run2: %v", err)
			}
			if !bytes.Equal(mustMarshal(t, r1), mustMarshal(t, r2)) {
				t.Errorf("%s: output not deterministic across runs", tc.name)
			}
		})
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return buf.Bytes()
}

func writeGolden(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write golden: %v", err)
	}
}

// normalize trims a trailing newline so editor settings do not cause spurious
// diffs.
func normalize(b []byte) []byte {
	return bytes.TrimRight(b, "\n")
}
