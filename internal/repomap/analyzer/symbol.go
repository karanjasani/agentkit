package analyzer

import (
	"bytes"
	"context"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/karanjasani/agentkit/internal/repomap/loader"
	"github.com/karanjasani/agentkit/internal/repomap/rerr"
	"github.com/karanjasani/agentkit/pkg/models"
)

// SymbolDetail selects which detail a symbol lookup returns.
type SymbolDetail struct {
	Body          bool
	SignatureOnly bool
	Doc           bool
	Shape         bool
}

// candidate is an internal match for a symbol name.
type candidate struct {
	pkg  *packages.Package
	decl ast.Decl
	spec ast.Spec // for GenDecl-based decls (type/var/const)
	name string
	recv string
	obj  types.Object
	kind string
}

// LookupSymbol locates a symbol by name and returns the requested detail. The
// name may be bare ("ValidateToken"), package-qualified ("models.FabricStatus"),
// or a method ("Type.Method").
func LookupSymbol(ctx context.Context, l *loader.Loader, name string, detail SymbolDetail) (models.Symbol, error) {
	res, err := l.Load(ctx, loader.TierTypes, false)
	if err != nil {
		return models.Symbol{}, rerr.New(rerr.LoadFailed, false, "%v", err)
	}

	cands := findSymbolCandidates(res, name)
	if len(cands) == 0 {
		return models.Symbol{}, rerr.New(rerr.SymbolNotFound, true, "symbol not found: %s", name)
	}
	c := cands[0]

	sym := models.Symbol{
		Name:     c.name,
		Kind:     c.kind,
		Package:  c.pkg.PkgPath,
		Recv:     c.recv,
		Location: location(res.Fset, res.ModuleDir, c.obj.Pos()),
	}

	sig := signatureString(res.Fset, c)
	doc := docString(c)

	switch {
	case detail.SignatureOnly:
		sym.Signature = sig
	case detail.Doc:
		sym.Doc = doc
	case detail.Body:
		sym.Signature = sig
		sym.Doc = doc
		sym.Body = bodyString(res.Fset, c)
	default:
		sym.Signature = sig
		sym.Doc = doc
	}

	if detail.Shape {
		if tn, ok := c.obj.(*types.TypeName); ok {
			if named, ok := tn.Type().(*types.Named); ok {
				if st, ok := named.Underlying().(*types.Struct); ok {
					shape := buildStruct(tn.Name(), st, 0, map[string]bool{})
					sym.Shape = &shape
				}
			}
		}
	}

	return sym, nil
}

func findSymbolCandidates(res *loader.Result, name string) []candidate {
	pkgQual, symName := splitQualified(name)
	// Distinguish "Type.Method" from "pkg.Symbol": we treat the qualifier as a
	// receiver type when the qualifier is capitalized (exported type) AND no
	// package with that name is present. We check both interpretations.
	var out []candidate

	for _, p := range res.Packages {
		if p.Types == nil || isTestVariant(p.PkgPath) {
			continue
		}
		pkgName := p.Types.Name()
		for _, f := range p.Syntax {
			for _, decl := range f.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					recv := receiverName(d)
					// Bare method or function match.
					if d.Name.Name == symName {
						// Interpret qualifier as package or receiver.
						if pkgQual == "" || pkgQual == pkgName || pkgQual == recv {
							out = append(out, funcCandidate(p, d, recv))
						}
					}
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						if cand, ok := genCandidate(p, d, spec, symName, pkgQual, pkgName); ok {
							out = append(out, cand)
						}
					}
				}
			}
		}
	}
	// Deterministic ordering: by package path then by name.
	stableSortCandidates(out)
	return out
}

func funcCandidate(p *packages.Package, d *ast.FuncDecl, recv string) candidate {
	kind := "func"
	if recv != "" {
		kind = "method"
	}
	var obj types.Object
	if p.TypesInfo != nil {
		obj = p.TypesInfo.Defs[d.Name]
	}
	return candidate{pkg: p, decl: d, name: d.Name.Name, recv: recv, obj: obj, kind: kind}
}

func genCandidate(p *packages.Package, d *ast.GenDecl, spec ast.Spec, symName, pkgQual, pkgName string) (candidate, bool) {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		if s.Name.Name == symName && (pkgQual == "" || pkgQual == pkgName) {
			var obj types.Object
			if p.TypesInfo != nil {
				obj = p.TypesInfo.Defs[s.Name]
			}
			return candidate{pkg: p, decl: d, spec: s, name: s.Name.Name, obj: obj, kind: "type"}, true
		}
	case *ast.ValueSpec:
		for _, id := range s.Names {
			if id.Name == symName && (pkgQual == "" || pkgQual == pkgName) {
				var obj types.Object
				if p.TypesInfo != nil {
					obj = p.TypesInfo.Defs[id]
				}
				kind := "var"
				if d.Tok == token.CONST {
					kind = "const"
				}
				return candidate{pkg: p, decl: d, spec: s, name: id.Name, obj: obj, kind: kind}, true
			}
		}
	}
	return candidate{}, false
}

func receiverName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	return baseTypeName(d.Recv.List[0].Type)
}

func baseTypeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return baseTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr: // generic receiver: T[P]
		return baseTypeName(t.X)
	case *ast.IndexListExpr:
		return baseTypeName(t.X)
	}
	return ""
}

// signatureString renders a declaration's signature without its body.
func signatureString(fset *token.FileSet, c candidate) string {
	switch d := c.decl.(type) {
	case *ast.FuncDecl:
		tmp := *d
		tmp.Body = nil
		tmp.Doc = nil
		return printNode(fset, &tmp)
	case *ast.GenDecl:
		switch s := c.spec.(type) {
		case *ast.TypeSpec:
			// Print "type Name <underlying-name>" compactly for struct/interface.
			if _, ok := s.Type.(*ast.StructType); ok {
				return "type " + s.Name.Name + " struct{...}"
			}
			if _, ok := s.Type.(*ast.InterfaceType); ok {
				return "type " + s.Name.Name + " interface{...}"
			}
			cp := *s
			cp.Doc = nil
			cp.Comment = nil
			return "type " + printNode(fset, &cp)
		case *ast.ValueSpec:
			return c.kind + " " + c.name
		}
	}
	return ""
}

// bodyString renders the full declaration source.
func bodyString(fset *token.FileSet, c candidate) string {
	switch d := c.decl.(type) {
	case *ast.FuncDecl:
		tmp := *d
		tmp.Doc = nil
		return printNode(fset, &tmp)
	case *ast.GenDecl:
		if c.spec != nil {
			switch s := c.spec.(type) {
			case *ast.TypeSpec:
				cp := *s
				cp.Doc = nil
				cp.Comment = nil
				return "type " + printNode(fset, &cp)
			case *ast.ValueSpec:
				return printNode(fset, s)
			}
		}
		return printNode(fset, d)
	}
	return ""
}

func docString(c candidate) string {
	switch d := c.decl.(type) {
	case *ast.FuncDecl:
		return strings.TrimSpace(d.Doc.Text())
	case *ast.GenDecl:
		if s, ok := c.spec.(*ast.TypeSpec); ok && s.Doc != nil {
			return strings.TrimSpace(s.Doc.Text())
		}
		return strings.TrimSpace(d.Doc.Text())
	}
	return ""
}

func printNode(fset *token.FileSet, node any) string {
	var buf bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 4}
	if err := cfg.Fprint(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}

func stableSortCandidates(cs []candidate) {
	// Simple insertion-free stable ordering via sort on (pkgpath, kind, name).
	for i := 1; i < len(cs); i++ {
		for j := i; j > 0 && less(cs[j], cs[j-1]); j-- {
			cs[j], cs[j-1] = cs[j-1], cs[j]
		}
	}
}

func less(a, b candidate) bool {
	if a.pkg.PkgPath != b.pkg.PkgPath {
		return a.pkg.PkgPath < b.pkg.PkgPath
	}
	if a.name != b.name {
		return a.name < b.name
	}
	return a.recv < b.recv
}
