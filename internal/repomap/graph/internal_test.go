package graph

import (
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"

	"github.com/karanjasani/agentkit/internal/repomap/loader"
	"github.com/karanjasani/agentkit/pkg/models"
)

func TestFnPkgPathAndInModule(t *testing.T) {
	fn := &ssa.Function{} // no package
	if got := fnPkgPath(fn); got != "" {
		t.Errorf("fnPkgPath(no pkg) = %q, want empty", got)
	}
	// No module path: everything is considered in-module.
	if !inModule(&loader.Result{}, fn) {
		t.Error("inModule with empty ModulePath should be true")
	}
	// Module path set but the function has no resolvable package path.
	if inModule(&loader.Result{ModulePath: "m"}, fn) {
		t.Error("inModule with unresolved package path should be false")
	}
}

func newTestFunc(pkgPath, name string, withRecv bool) *types.Func {
	var pkg *types.Package
	if pkgPath != "" {
		pkg = types.NewPackage(pkgPath, name)
	}
	var recv *types.Var
	if withRecv {
		recv = types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])
	}
	sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)
	return types.NewFunc(token.NoPos, pkg, name, sig)
}

func TestAppendUnique(t *testing.T) {
	f := newTestFunc("m", "A", false)
	g := newTestFunc("m", "B", false)
	out := appendUnique(nil, f)
	out = appendUnique(out, f) // duplicate: must not grow
	if len(out) != 1 {
		t.Fatalf("after duplicate append len = %d, want 1", len(out))
	}
	out = appendUnique(out, g)
	if len(out) != 2 {
		t.Fatalf("after distinct append len = %d, want 2", len(out))
	}
}

func TestPkgPath(t *testing.T) {
	if got := pkgPath(nil); got != "" {
		t.Errorf("pkgPath(nil) = %q, want empty", got)
	}
	if got := pkgPath(newTestFunc("", "x", false)); got != "" {
		t.Errorf("pkgPath(nil pkg) = %q, want empty", got)
	}
	if got := pkgPath(newTestFunc("m/a", "x", false)); got != "m/a" {
		t.Errorf("pkgPath = %q, want m/a", got)
	}
}

func TestModulePackages(t *testing.T) {
	res := &loader.Result{
		ModulePath: "m",
		Packages: []*packages.Package{
			{PkgPath: ""},        // skipped: empty path
			{PkgPath: "m"},       // kept: module root
			{PkgPath: "m/a"},     // kept: under module
			{PkgPath: "other/b"}, // skipped: outside module
		},
	}
	got := modulePackages(res)
	if len(got) != 2 {
		t.Fatalf("modulePackages len = %d, want 2 (%v)", len(got), got)
	}

	// With no module path, all non-empty packages are returned.
	resNoMod := &loader.Result{Packages: []*packages.Package{{PkgPath: ""}, {PkgPath: "x"}, {PkgPath: "y"}}}
	if got := modulePackages(resNoMod); len(got) != 2 {
		t.Fatalf("modulePackages(no module) len = %d, want 2", len(got))
	}
}

func TestObjKind(t *testing.T) {
	pkg := types.NewPackage("m", "m")
	cases := []struct {
		obj  types.Object
		want string
	}{
		{newTestFunc("m", "F", false), "func"},
		{newTestFunc("m", "M", true), "method"},
		{types.NewTypeName(token.NoPos, pkg, "T", nil), "type"},
		{types.NewVar(token.NoPos, pkg, "V", types.Typ[types.Int]), "var"},
		{types.NewConst(token.NoPos, pkg, "C", types.Typ[types.Int], nil), "const"},
		{types.NewLabel(token.NoPos, pkg, "L"), "symbol"},
	}
	for _, tc := range cases {
		if got := objKind(tc.obj); got != tc.want {
			t.Errorf("objKind(%T) = %q, want %q", tc.obj, got, tc.want)
		}
	}
}

func TestBaseOr(t *testing.T) {
	if got := baseOr(""); got != "HEAD" {
		t.Errorf("baseOr(\"\") = %q, want HEAD", got)
	}
	if got := baseOr("main"); got != "main" {
		t.Errorf("baseOr(main) = %q, want main", got)
	}
}

func TestSortTests(t *testing.T) {
	ts := []models.Test{
		{Name: "TestB", Location: models.Location{File: "b.go", Line: 1}},
		{Name: "TestA", Location: models.Location{File: "a.go", Line: 9}},
		{Name: "TestA", Location: models.Location{File: "a.go", Line: 9}, Package: "p2"},
		{Name: "TestA2", Location: models.Location{File: "a.go", Line: 9}},
	}
	sortTests(ts)
	if ts[0].Location.File != "a.go" || ts[len(ts)-1].Location.File != "b.go" {
		t.Fatalf("sortTests did not order by file: %+v", ts)
	}
	// Within a.go: by name then package.
	if ts[0].Name != "TestA" || ts[1].Name != "TestA" || ts[2].Name != "TestA2" {
		t.Errorf("sortTests name/package tiebreak wrong: %+v", ts)
	}
	if ts[0].Package != "" || ts[1].Package != "p2" {
		t.Errorf("sortTests package tiebreak wrong: %+v", ts)
	}
}

func TestItoa(t *testing.T) {
	cases := map[int]string{0: "0", 5: "5", 42: "42", -7: "-7", 1000: "1000"}
	for in, want := range cases {
		if got := itoa(in); got != want {
			t.Errorf("itoa(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitQualified(t *testing.T) {
	if p, s := splitQualified("Type.Method"); p != "Type" || s != "Method" {
		t.Errorf("got %q %q", p, s)
	}
	if p, s := splitQualified("Bare"); p != "" || s != "Bare" {
		t.Errorf("got %q %q", p, s)
	}
}

func TestIsTestVariant(t *testing.T) {
	if !isTestVariant("example.com/p.test") {
		t.Error("expected test variant")
	}
	if !isTestVariant("example.com/p [example.com/p.test]") {
		t.Error("expected bracketed test variant")
	}
	if isTestVariant("example.com/p") {
		t.Error("did not expect test variant")
	}
}

func TestCallerKeyAndSort(t *testing.T) {
	m := map[string]models.Caller{}
	a := models.Caller{Symbol: "B", Package: "p", Location: models.Location{File: "a.go", Line: 2}}
	b := models.Caller{Symbol: "A", Package: "p", Location: models.Location{File: "a.go", Line: 1}}
	m[callerKey(a)] = a
	m[callerKey(b)] = b
	got := sortedCallers(m)
	if len(got) != 2 || got[0].Location.Line != 1 {
		t.Fatalf("sortedCallers not sorted by line: %+v", got)
	}
}

func TestRiskScore(t *testing.T) {
	cases := []struct {
		changed, affected, api int
		wantLevel              string
	}{
		{0, 0, 0, "low"},
		{1, 1, 1, "low"},    // 10+5+8 = 23
		{2, 2, 2, "medium"}, // 20+10+16 = 46
		{5, 5, 5, "high"},   // capped 100
	}
	for _, tc := range cases {
		score, level := riskScore(tc.changed, tc.affected, tc.api)
		if level != tc.wantLevel {
			t.Errorf("riskScore(%d,%d,%d) level = %s, want %s (score %d)",
				tc.changed, tc.affected, tc.api, level, tc.wantLevel, score)
		}
		if score > 100 {
			t.Errorf("score exceeded 100: %d", score)
		}
	}
}
