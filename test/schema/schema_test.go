// Package schema validates that the committed golden outputs conform to the
// published repomap.v1 JSON Schema files. This is what makes the stability
// promise enforceable: any drift between the emitted shape and the documented
// contract fails CI.
package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	schemaDir = "../../schemas/repomap.v1"
	goldenDir = "../golden/testdata"
	idBase    = "https://github.com/karanjasani/agentkit/schemas/repomap.v1/"
)

// goldenSchema maps each golden fixture (without the .json suffix) to the
// command schema that describes its result payload.
var goldenSchema = map[string]string{
	"overview":          "overview",
	"package_service":   "package",
	"symbol_signature":  "symbol",
	"symbol_body":       "symbol",
	"struct_widget":     "struct",
	"deps":              "deps",
	"callers_helper":    "callers",
	"tests_fetchwidget": "tests",
	"tests_helper":      "tests",
	"endpoint":          "endpoint",
	"upstreams":         "upstreams",
}

func newCompiler(t *testing.T) *jsonschema.Compiler {
	t.Helper()
	c := jsonschema.NewCompiler()
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		t.Fatalf("reading schema dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		doc := loadJSON(t, filepath.Join(schemaDir, e.Name()))
		m, ok := doc.(map[string]any)
		if !ok {
			t.Fatalf("%s: schema root is not an object", e.Name())
		}
		id, ok := m["$id"].(string)
		if !ok || id == "" {
			t.Fatalf("%s: missing $id", e.Name())
		}
		if err := c.AddResource(id, doc); err != nil {
			t.Fatalf("adding resource %s: %v", id, err)
		}
	}
	return c
}

func loadJSON(t *testing.T, path string) any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	doc, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return doc
}

func TestGoldenAgainstSchema(t *testing.T) {
	c := newCompiler(t)
	for golden, schemaName := range goldenSchema {
		t.Run(golden, func(t *testing.T) {
			sch, err := c.Compile(idBase + schemaName + ".schema.json")
			if err != nil {
				t.Fatalf("compile %s: %v", schemaName, err)
			}
			inst := loadJSON(t, filepath.Join(goldenDir, golden+".json"))
			if err := sch.Validate(inst); err != nil {
				t.Errorf("%s does not satisfy %s.schema.json:\n%v", golden, schemaName, err)
			}
		})
	}
}

// TestGoldenCoverage guards against a new golden file being added without a
// corresponding schema mapping.
func TestGoldenCoverage(t *testing.T) {
	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("reading golden dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		name := e.Name()[:len(e.Name())-len(".json")]
		if _, ok := goldenSchema[name]; !ok {
			t.Errorf("golden %q has no schema mapping in goldenSchema", name)
		}
	}
}

func TestEnvelopeSchema(t *testing.T) {
	c := newCompiler(t)
	sch, err := c.Compile(idBase + "envelope.schema.json")
	if err != nil {
		t.Fatalf("compile envelope: %v", err)
	}

	ok := map[string]any{
		"schema":       "repomap.v1",
		"tool_version": "0.1.0",
		"ok":           true,
		"result":       map[string]any{"anything": true},
	}
	if err := sch.Validate(ok); err != nil {
		t.Errorf("valid success envelope rejected: %v", err)
	}

	fail := map[string]any{
		"schema":       "repomap.v1",
		"tool_version": "0.1.0",
		"ok":           false,
		"error": map[string]any{
			"code":        "symbol_not_found",
			"message":     "no symbol named Foo",
			"recoverable": false,
		},
	}
	if err := sch.Validate(fail); err != nil {
		t.Errorf("valid error envelope rejected: %v", err)
	}

	// ok=true but no result must fail the oneOf.
	bad := map[string]any{
		"schema":       "repomap.v1",
		"tool_version": "0.1.0",
		"ok":           true,
	}
	if err := sch.Validate(bad); err == nil {
		t.Error("envelope with ok=true and no result should be rejected")
	}
}
