package analyzer

import (
	"context"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/karanjasani/agentkit/internal/repomap/loader"
	"github.com/karanjasani/agentkit/internal/repomap/rerr"
	"github.com/karanjasani/agentkit/pkg/models"
)

// PackageInfo returns information about a single package, addressed by import
// path or by directory (e.g. "./internal/auth").
func PackageInfo(ctx context.Context, l *loader.Loader, path string) (models.Package, error) {
	res, err := l.Load(ctx, loader.TierTypes, true)
	if err != nil {
		return models.Package{}, rerr.New(rerr.LoadFailed, false, "%v", err)
	}

	target := resolvePackage(res, path)
	if target == nil {
		return models.Package{}, rerr.New(rerr.PackageNotFound, true, "package not found: %s", path)
	}

	out := models.Package{
		ImportPath: target.PkgPath,
		Name:       target.Name,
		Dir:        packageDir(res.ModuleDir, target),
		Imports:    importPaths(target),
		ImportedBy: importedBy(res, target.PkgPath),
		Exports:    exportedSymbols(res, target),
		TestFiles:  testFiles(res, target.PkgPath),
	}
	return out, nil
}

// resolvePackage finds a package by exact import path, then by module-relative
// directory, then by import-path suffix.
func resolvePackage(res *loader.Result, path string) *packages.Package {
	clean := strings.TrimPrefix(path, "./")
	clean = strings.TrimSuffix(clean, "/")

	var byDir, bySuffix *packages.Package
	for _, p := range res.Packages {
		if p.PkgPath == "" || isTestVariant(p.PkgPath) {
			continue
		}
		if p.PkgPath == path || p.PkgPath == clean {
			return p
		}
		if packageDir(res.ModuleDir, p) == clean && byDir == nil {
			byDir = p
		}
		if strings.HasSuffix(p.PkgPath, "/"+clean) && bySuffix == nil {
			bySuffix = p
		}
	}
	if byDir != nil {
		return byDir
	}
	return bySuffix
}

// isTestVariant reports whether a package path is a synthetic test variant
// produced by loading with Tests: true (e.g. "foo [foo.test]", "foo.test").
func isTestVariant(pkgPath string) bool {
	return strings.Contains(pkgPath, ".test") || strings.Contains(pkgPath, "[")
}

func importPaths(p *packages.Package) []string {
	out := make([]string, 0, len(p.Imports))
	for path := range p.Imports {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func importedBy(res *loader.Result, target string) []string {
	set := map[string]bool{}
	for _, p := range res.Packages {
		if p.PkgPath == "" || isTestVariant(p.PkgPath) || p.PkgPath == target {
			continue
		}
		if _, ok := p.Imports[target]; ok {
			set[p.PkgPath] = true
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func exportedSymbols(res *loader.Result, p *packages.Package) []models.Symbol {
	out := []models.Symbol{}
	if p.Types == nil {
		return out
	}
	scope := p.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj == nil || !obj.Exported() {
			continue
		}
		out = append(out, models.Symbol{
			Name:     name,
			Kind:     kindOf(obj),
			Package:  p.PkgPath,
			Location: location(res.Fset, res.ModuleDir, obj.Pos()),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func kindOf(obj types.Object) string {
	switch o := obj.(type) {
	case *types.Func:
		if sig, ok := o.Type().(*types.Signature); ok && sig.Recv() != nil {
			return "method"
		}
		return "func"
	case *types.TypeName:
		return "type"
	case *types.Var:
		return "var"
	case *types.Const:
		return "const"
	default:
		return "symbol"
	}
}

// testFiles collects _test.go files for a package, including those that only
// appear in the synthetic test-variant package produced when loading with Tests.
func testFiles(res *loader.Result, pkgPath string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, p := range res.Packages {
		if basePkgPath(p.PkgPath) != pkgPath {
			continue
		}
		for _, f := range p.GoFiles {
			rel := relPath(res.ModuleDir, f)
			if strings.HasSuffix(rel, "_test.go") && !seen[rel] {
				seen[rel] = true
				out = append(out, rel)
			}
		}
	}
	sort.Strings(out)
	return out
}

// basePkgPath strips the synthetic test-variant suffix/annotation from a package
// path (e.g. "p [p.test]" or "p.test" -> "p").
func basePkgPath(pkgPath string) string {
	if i := strings.Index(pkgPath, " ["); i >= 0 {
		pkgPath = pkgPath[:i]
	}
	return strings.TrimSuffix(pkgPath, ".test")
}
