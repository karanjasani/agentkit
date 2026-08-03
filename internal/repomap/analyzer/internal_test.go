package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestKindOf(t *testing.T) {
	pkg := types.NewPackage("m", "m")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	recv := types.NewVar(token.NoPos, pkg, "r", types.Typ[types.Int])
	methodSig := types.NewSignatureType(recv, nil, nil, nil, nil, false)
	cases := []struct {
		obj  types.Object
		want string
	}{
		{types.NewFunc(token.NoPos, pkg, "F", sig), "func"},
		{types.NewFunc(token.NoPos, pkg, "M", methodSig), "method"},
		{types.NewTypeName(token.NoPos, pkg, "T", nil), "type"},
		{types.NewVar(token.NoPos, pkg, "V", types.Typ[types.Int]), "var"},
		{types.NewConst(token.NoPos, pkg, "C", types.Typ[types.Int], nil), "const"},
		{types.NewLabel(token.NoPos, pkg, "L"), "symbol"},
	}
	for _, tc := range cases {
		if got := kindOf(tc.obj); got != tc.want {
			t.Errorf("kindOf(%T) = %q, want %q", tc.obj, got, tc.want)
		}
	}
}

func TestLooksGenerated(t *testing.T) {
	yes := []string{"api/foo.pb.go", "x_generated.go", "zz_generated.deepcopy.go", "bindata.go", "types_string.go", "a/b/c.gen.go"}
	no := []string{"main.go", "service/service.go", "foo_test.go"}
	for _, f := range yes {
		if !looksGenerated(f) {
			t.Errorf("looksGenerated(%q) = false, want true", f)
		}
	}
	for _, f := range no {
		if looksGenerated(f) {
			t.Errorf("looksGenerated(%q) = true, want false", f)
		}
	}
}

func TestPackageDir(t *testing.T) {
	// No files: empty directory.
	if got := packageDir("/m", &packages.Package{}); got != "" {
		t.Errorf("packageDir(no files) = %q, want empty", got)
	}
	// File at module root: "." (no slash in the relative path).
	if got := packageDir("/m", &packages.Package{GoFiles: []string{"/m/main.go"}}); got != "." {
		t.Errorf("packageDir(root file) = %q, want .", got)
	}
	// File in a subdirectory.
	if got := packageDir("/m", &packages.Package{GoFiles: []string{"/m/svc/x.go"}}); got != "svc" {
		t.Errorf("packageDir(subdir) = %q, want svc", got)
	}
	// Fallback to CompiledGoFiles when GoFiles is empty.
	if got := packageDir("/m", &packages.Package{CompiledGoFiles: []string{"/m/gen/y.go"}}); got != "gen" {
		t.Errorf("packageDir(compiled) = %q, want gen", got)
	}
}

func TestVendorDir(t *testing.T) {
	if got := vendorDir("vendor/github.com/x/y/z.go"); got != "vendor/github.com" {
		t.Errorf("vendorDir = %q", got)
	}
	if got := vendorDir("internal/x.go"); got != "" {
		t.Errorf("vendorDir non-vendor = %q", got)
	}
}

func TestBasePkgPath(t *testing.T) {
	cases := map[string]string{
		"example.com/p":                      "example.com/p",
		"example.com/p.test":                 "example.com/p",
		"example.com/p [example.com/p.test]": "example.com/p",
	}
	for in, want := range cases {
		if got := basePkgPath(in); got != want {
			t.Errorf("basePkgPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitQualified(t *testing.T) {
	if p, s := splitQualified("pkg.Name"); p != "pkg" || s != "Name" {
		t.Errorf("got %q %q", p, s)
	}
	if p, s := splitQualified("Bare"); p != "" || s != "Bare" {
		t.Errorf("got %q %q", p, s)
	}
}

func TestParseJSONTag(t *testing.T) {
	name, omit, skip := parseJSONTag(reflect.StructTag(`json:"id,omitempty"`), "ID")
	if name != "id" || !omit || skip {
		t.Errorf("got %q %v %v", name, omit, skip)
	}
	name, _, skip = parseJSONTag(reflect.StructTag(`json:"-"`), "ID")
	if !skip {
		t.Errorf("expected skip for json:-")
	}
	name, _, _ = parseJSONTag(reflect.StructTag(""), "ID")
	if name != "ID" {
		t.Errorf("no tag: got %q", name)
	}
	name, _, _ = parseJSONTag(reflect.StructTag(`json:",omitempty"`), "ID")
	if name != "ID" {
		t.Errorf("empty name tag: got %q", name)
	}
}

func TestReceiverAndBaseTypeName(t *testing.T) {
	src := `package p
func (c *Client) Do() {}
func (s Server) Handle() {}
func Plain() {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"Do": "Client", "Handle": "Server", "Plain": ""}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if got := receiverName(fd); got != want[fd.Name.Name] {
			t.Errorf("receiverName(%s) = %q, want %q", fd.Name.Name, got, want[fd.Name.Name])
		}
	}
}

// TestBaseTypeNameDefault covers the fall-through (unrecognized expression) of
// baseTypeName, which yields an empty string.
func TestBaseTypeNameDefault(t *testing.T) {
	e, err := parser.ParseExpr("map[string]int")
	if err != nil {
		t.Fatal(err)
	}
	if got := baseTypeName(e); got != "" {
		t.Errorf("baseTypeName(map) = %q, want empty", got)
	}
}

// TestBaseTypeNameGenerics covers the generic-receiver branches of baseTypeName
// (IndexExpr for a single type param, IndexListExpr for multiple).
func TestBaseTypeNameGenerics(t *testing.T) {
	src := `package p
func (b *Box[T]) M() {}
func (p Pair[K, V]) N() {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "g.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"M": "Box", "N": "Pair"}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if got := receiverName(fd); got != want[fd.Name.Name] {
			t.Errorf("receiverName(%s) = %q, want %q", fd.Name.Name, got, want[fd.Name.Name])
		}
	}
}

// TestSignatureBodyDocVariants exercises signatureString, bodyString and
// docString across interface, alias, struct, var and const declarations,
// including a grouped decl whose doc attaches to the spec rather than the
// GenDecl.
func TestSignatureBodyDocVariants(t *testing.T) {
	src := `package p
// TDoc.
type T interface{ M() }
// ADoc.
type A = int
// SDoc.
type S struct{ X int }
// VDoc.
var V = 1
// CDoc.
const C = 2
type (
	// GroupDoc.
	G struct{}
)
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "v.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	find := func(name string) (*ast.GenDecl, ast.Spec) {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, sp := range gd.Specs {
				switch s := sp.(type) {
				case *ast.TypeSpec:
					if s.Name.Name == name {
						return gd, s
					}
				case *ast.ValueSpec:
					for _, id := range s.Names {
						if id.Name == name {
							return gd, s
						}
					}
				}
			}
		}
		t.Fatalf("decl %q not found", name)
		return nil, nil
	}

	gd, sp := find("T")
	if got := signatureString(fset, candidate{decl: gd, spec: sp, name: "T", kind: "type"}); got != "type T interface{...}" {
		t.Errorf("interface sig = %q", got)
	}

	gd, sp = find("A")
	c := candidate{decl: gd, spec: sp, name: "A", kind: "type"}
	if got := signatureString(fset, c); got != "type A = int" {
		t.Errorf("alias sig = %q", got)
	}
	if got := bodyString(fset, c); got != "type A = int" {
		t.Errorf("alias body = %q", got)
	}

	gd, sp = find("S")
	if got := signatureString(fset, candidate{decl: gd, spec: sp, name: "S", kind: "type"}); got != "type S struct{...}" {
		t.Errorf("struct sig = %q", got)
	}

	gd, sp = find("V")
	c = candidate{decl: gd, spec: sp, name: "V", kind: "var"}
	if got := signatureString(fset, c); got != "var V" {
		t.Errorf("var sig = %q", got)
	}
	if got := bodyString(fset, c); got != "V = 1" {
		t.Errorf("var body = %q", got)
	}
	if got := docString(c); got != "VDoc." {
		t.Errorf("var doc = %q", got)
	}

	gd, sp = find("C")
	if got := docString(candidate{decl: gd, spec: sp, name: "C", kind: "const"}); got != "CDoc." {
		t.Errorf("const doc = %q", got)
	}

	// Grouped type: the doc comment attaches to the TypeSpec, exercising the
	// s.Doc != nil branch of docString.
	gd, sp = find("G")
	if got := docString(candidate{decl: gd, spec: sp, name: "G", kind: "type"}); got != "GroupDoc." {
		t.Errorf("grouped type doc = %q", got)
	}
}

// TestStableSortCandidates covers all three comparison branches of less and the
// reordering loop in stableSortCandidates.
func TestStableSortCandidates(t *testing.T) {
	mk := func(pkg, name, recv string) candidate {
		return candidate{pkg: &packages.Package{PkgPath: pkg}, name: name, recv: recv}
	}
	cs := []candidate{
		mk("b/pkg", "A", ""),
		mk("a/pkg", "B", ""),
		mk("a/pkg", "A", "Y"),
		mk("a/pkg", "A", "X"),
	}
	stableSortCandidates(cs)
	want := []string{"a/pkg|A|X", "a/pkg|A|Y", "a/pkg|B|", "b/pkg|A|"}
	for i, c := range cs {
		got := c.pkg.PkgPath + "|" + c.name + "|" + c.recv
		if got != want[i] {
			t.Errorf("pos %d = %q, want %q", i, got, want[i])
		}
	}
}
