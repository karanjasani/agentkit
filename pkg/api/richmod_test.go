package api_test

import (
	"context"
	"testing"

	"github.com/karanjasani/agentkit/pkg/api"
)

// richModule writes a small but feature-rich temp module used to exercise the
// analyzer branches that the primary fixture does not reach: exported const/var
// kinds, importedBy, package resolution by suffix, nested struct shapes through
// pointers/slices/named types, generated-file detection, and full endpoint
// tracing (handler resolution, orchestration, request/response types).
//
// JSON struct tags are intentionally omitted here (tag handling is covered by
// the primary Widget fixture); field names double as JSON names.
func richModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, dir, "go.mod", "module example.com/rich\n\ngo 1.24\n")

	writeFile(t, dir, "model/model.go", `package model

// Status is an exported constant.
const Status = "ok"

// DefaultName is an exported variable.
var DefaultName = "widget"

// Kind is an exported interface.
type Kind interface{ Name() string }

// Item is the primary domain type with nested shapes.
type Item struct {
	ID     string
	Owner  Owner
	Parent *Owner
	Tags   []Owner
	Matrix [2]Owner
	Anon   struct{ X int }
	Labels map[string]string
}

// Owner is a nested struct.
type Owner struct {
	Email string
}

// Celsius is a named non-struct type, used to exercise the not-a-struct path.
type Celsius float64

// Generic is an uninstantiated generic function; its origin has no SSA body, so
// caller analysis takes the no-SSA-function path.
func Generic[T any](x T) T { return x }

// Describe is a method on Owner.
func (o Owner) Describe() string { return o.Email }

// Build constructs an Item.
func Build() Item { return Item{} }
`)

	// A generated file that is part of a loaded package, to exercise the
	// generated-file branch of Overview.
	writeFile(t, dir, "model/types.pb.go", `package model

// Generated is a machine-generated type.
type Generated struct{}
`)

	writeFile(t, dir, "store/store.go", `package store

import (
	"net/http"

	"example.com/rich/model"
)

// Save persists an item.
func Save(i model.Item) model.Item { return i }

// Handler is an exported HTTP handler used to exercise selector-based handler
// resolution across packages.
func Handler(w http.ResponseWriter, r *http.Request) {}
`)

	writeFile(t, dir, "main.go", `package main

import (
	"encoding/json"
	"net/http"

	"example.com/rich/model"
	"example.com/rich/store"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /items", createItem)
	// Selector handler from another package.
	mux.HandleFunc("GET /ext", store.Handler)
	// Wrapped handler: the handler expression is a call, so resolution falls
	// back to matching the wrapping function's name within the package.
	mux.HandleFunc("GET /wrapped", wrap(createItem))
	// Handler wrapped in http.HandlerFunc: no local declaration matches the
	// resolved name, so only the name is reported (no handler decl).
	mux.HandleFunc("GET /std", http.HandlerFunc(createItem))
	warmup()
	_ = http.ListenAndServe(":0", mux)
}

// wrap returns its handler unchanged.
func wrap(h http.HandlerFunc) http.HandlerFunc { return h }

// warmup calls createItem so that Save has a two-hop caller chain
// (Save <- createItem <- warmup), exercising indirect-caller analysis.
func warmup() { createItem(nil, nil) }

// createItem decodes an Item, saves it, and encodes the result.
func createItem(w http.ResponseWriter, r *http.Request) {
	var it model.Item
	_ = json.NewDecoder(r.Body).Decode(&it)
	saved := store.Save(it)
	_ = json.NewEncoder(w).Encode(saved)
}
`)

	return dir
}

func richAnalyzer(t *testing.T) *api.Analyzer {
	t.Helper()
	a, err := api.New(context.Background(), api.WithDir(richModule(t)))
	if err != nil {
		t.Fatalf("api.New: %v", err)
	}
	return a
}

func TestRichOverviewGenerated(t *testing.T) {
	a := richAnalyzer(t)
	ov, err := a.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Entrypoints) == 0 {
		t.Error("expected a main entrypoint")
	}
	foundGen := false
	for _, g := range ov.Generated {
		if g == "model/types.pb.go" {
			foundGen = true
		}
	}
	if !foundGen {
		t.Errorf("expected generated file detected, got %v", ov.Generated)
	}
}

func TestRichPackageKindsAndImporters(t *testing.T) {
	a := richAnalyzer(t)
	pkg, err := a.Package(context.Background(), "example.com/rich/model")
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]string{}
	for _, s := range pkg.Exports {
		kinds[s.Name] = s.Kind
	}
	want := map[string]string{
		"Status":      "const",
		"DefaultName": "var",
		"Kind":        "type",
		"Item":        "type",
		"Build":       "func",
	}
	for name, k := range want {
		if kinds[name] != k {
			t.Errorf("export %s kind = %q, want %q", name, kinds[name], k)
		}
	}
	importedByStore := false
	for _, ib := range pkg.ImportedBy {
		if ib == "example.com/rich/store" {
			importedByStore = true
		}
	}
	if !importedByStore {
		t.Errorf("expected store among importers, got %v", pkg.ImportedBy)
	}
}

func TestRichPackageResolveBySuffixAndDir(t *testing.T) {
	a := richAnalyzer(t)
	// Suffix match: not an exact path or a directory, but a path suffix.
	bySuffix, err := a.Package(context.Background(), "rich/store")
	if err != nil {
		t.Fatalf("suffix resolve: %v", err)
	}
	if bySuffix.Name != "store" {
		t.Errorf("suffix name = %q, want store", bySuffix.Name)
	}
	// Directory match.
	byDir, err := a.Package(context.Background(), "./store")
	if err != nil {
		t.Fatalf("dir resolve: %v", err)
	}
	if byDir.ImportPath != "example.com/rich/store" {
		t.Errorf("dir import path = %q", byDir.ImportPath)
	}
}

func TestRichStructNestedVariants(t *testing.T) {
	a := richAnalyzer(t)
	st, err := a.Struct(context.Background(), "model.Item")
	if err != nil {
		t.Fatal(err)
	}
	nestedByName := map[string]bool{}
	for _, f := range st.Fields {
		nestedByName[f.Name] = f.Nested != nil
	}
	// Owner (named), Parent (pointer), Tags (slice), Matrix (array) and Anon
	// (anonymous struct) all reach a nested struct.
	for _, name := range []string{"Owner", "Parent", "Tags", "Matrix", "Anon"} {
		if !nestedByName[name] {
			t.Errorf("field %q should have a nested struct shape", name)
		}
	}
	// Labels is a map and should not produce a nested struct.
	if nestedByName["Labels"] {
		t.Error("map field should not have a nested struct")
	}
}

func TestRichSymbolInterfaceShapeIsNil(t *testing.T) {
	a := richAnalyzer(t)
	sym, err := a.Symbol(context.Background(), "model.Kind", api.SymbolOptions{Shape: true})
	if err != nil {
		t.Fatal(err)
	}
	if sym.Shape != nil {
		t.Error("interface should not yield a struct shape")
	}
}

func TestRichEndpointTrace(t *testing.T) {
	a := richAnalyzer(t)
	ep, err := a.Endpoint(context.Background(), "POST", "/items")
	if err != nil {
		t.Fatal(err)
	}
	if ep.Handler == nil || ep.Handler.Name != "createItem" {
		t.Fatalf("handler = %+v, want createItem", ep.Handler)
	}
	if ep.RequestType == "" || ep.ResponseType == "" {
		t.Errorf("expected request/response types, got req=%q resp=%q", ep.RequestType, ep.ResponseType)
	}
	foundSave := false
	for _, s := range ep.Orchestration {
		if s.Name == "Save" {
			foundSave = true
		}
	}
	if !foundSave {
		t.Errorf("expected Save in orchestration chain, got %+v", ep.Orchestration)
	}
}

func TestRichEndpointSelectorHandler(t *testing.T) {
	a := richAnalyzer(t)
	ep, err := a.Endpoint(context.Background(), "GET", "/ext")
	if err != nil {
		t.Fatal(err)
	}
	if ep.Handler == nil || ep.Handler.Name != "Handler" {
		t.Fatalf("handler = %+v, want Handler", ep.Handler)
	}
	if ep.Handler.Package != "example.com/rich/store" {
		t.Errorf("handler package = %q, want example.com/rich/store", ep.Handler.Package)
	}
}

func TestRichEndpointWrappedHandler(t *testing.T) {
	a := richAnalyzer(t)
	ep, err := a.Endpoint(context.Background(), "GET", "/wrapped")
	if err != nil {
		t.Fatal(err)
	}
	// The handler expression is wrap(createItem); resolution falls back to the
	// wrapping function name within the route's package.
	if ep.Handler == nil || ep.Handler.Name != "wrap" {
		t.Fatalf("handler = %+v, want wrap", ep.Handler)
	}
}

func TestRichCallersOfSave(t *testing.T) {
	a := richAnalyzer(t)
	c, err := a.Callers(context.Background(), "Save")
	if err != nil {
		t.Fatal(err)
	}
	direct := false
	for _, d := range c.Direct {
		if d.Symbol == "createItem" {
			direct = true
		}
	}
	if !direct {
		t.Errorf("expected createItem among direct callers of Save, got %+v", c.Direct)
	}
	indirect := false
	for _, d := range c.Indirect {
		if d.Symbol == "warmup" {
			indirect = true
		}
	}
	if !indirect {
		t.Errorf("expected warmup among indirect callers of Save, got %+v", c.Indirect)
	}
}

func TestRichEndpointNameOnlyHandler(t *testing.T) {
	a := richAnalyzer(t)
	ep, err := a.Endpoint(context.Background(), "GET", "/std")
	if err != nil {
		t.Fatal(err)
	}
	// The handler resolves to a name with no local declaration, so only the
	// name is populated.
	if ep.Handler == nil || ep.Handler.Name == "" {
		t.Fatalf("expected a name-only handler, got %+v", ep.Handler)
	}
}

func TestRichCallersGenericOrigin(t *testing.T) {
	a := richAnalyzer(t)
	// Generic is uninstantiated, so its origin resolves to no SSA function.
	// Callers must return cleanly with empty caller sets.
	c, err := a.Callers(context.Background(), "Generic")
	if err != nil {
		t.Fatal(err)
	}
	if c.Symbol != "Generic" {
		t.Errorf("symbol = %q, want Generic", c.Symbol)
	}
	if len(c.Direct) != 0 || len(c.Indirect) != 0 {
		t.Errorf("expected empty caller sets, got direct=%v indirect=%v", c.Direct, c.Indirect)
	}
}

func TestRichStructNotAStruct(t *testing.T) {
	a := richAnalyzer(t)
	_, err := a.Struct(context.Background(), "model.Celsius")
	if err == nil {
		t.Fatal("expected error for non-struct named type")
	}
	if e := api.AsError(err); e.Code != api.CodeNotAStruct {
		t.Errorf("code = %s, want %s", e.Code, api.CodeNotAStruct)
	}
}
