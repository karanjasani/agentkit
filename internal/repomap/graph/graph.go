// Package graph implements the call-graph-backed repomap commands: callers,
// tests and impact. It builds an SSA representation and a CHA call graph.
//
// A note on precision: CHA (class hierarchy analysis) works on library code
// without a whole-program entry point, but it over-approximates calls made
// through interfaces. Every reported edge therefore carries a confidence marker
// ("direct" for statically resolved calls, "possible" for dynamic ones) rather
// than silently presenting unreachable callers as facts.
package graph

import (
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/karanjasani/agentkit/internal/repomap/loader"
)

// funcObjects returns the *types.Func objects matching a name across the loaded
// module packages. The name may be bare ("Do"), package-qualified ("pkg.Do") or
// a method ("Type.Method").
func funcObjects(res *loader.Result, name string) []*types.Func {
	pkgQual, symName := splitQualified(name)
	var out []*types.Func

	for _, p := range res.Packages {
		if p.Types == nil || isTestVariant(p.PkgPath) {
			continue
		}
		pkgName := p.Types.Name()
		scope := p.Types.Scope()

		// Top-level function match.
		if pkgQual == "" || pkgQual == pkgName {
			if obj := scope.Lookup(symName); obj != nil {
				if fn, ok := obj.(*types.Func); ok {
					out = append(out, fn)
				}
			}
		}

		// Method match: qualifier interpreted as receiver type name.
		for _, tn := range scope.Names() {
			obj := scope.Lookup(tn)
			typeName, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			if pkgQual != "" && pkgQual != typeName.Name() && pkgQual != pkgName {
				continue
			}
			named, ok := typeName.Type().(*types.Named)
			if !ok {
				continue
			}
			for i := 0; i < named.NumMethods(); i++ {
				m := named.Method(i)
				if m.Name() == symName {
					out = append(out, m)
				}
			}
			// Pointer method set.
			ptr := types.NewPointer(named)
			ms := types.NewMethodSet(ptr)
			for i := 0; i < ms.Len(); i++ {
				m := ms.At(i).Obj()
				if fn, ok := m.(*types.Func); ok && fn.Name() == symName {
					out = appendUnique(out, fn)
				}
			}
		}
	}
	return out
}

func appendUnique(fns []*types.Func, fn *types.Func) []*types.Func {
	for _, f := range fns {
		if f == fn {
			return fns
		}
	}
	return append(fns, fn)
}

// pkgPath returns the import path of a func object's package, if any.
func pkgPath(fn *types.Func) string {
	if fn != nil && fn.Pkg() != nil {
		return fn.Pkg().Path()
	}
	return ""
}

func splitQualified(name string) (pkg, sym string) {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[:i], name[i+1:]
	}
	return "", name
}

func isTestVariant(pkgPath string) bool {
	return strings.Contains(pkgPath, ".test") || strings.Contains(pkgPath, "[")
}

// modulePackages returns loaded packages that belong to the main module.
func modulePackages(res *loader.Result) []*packages.Package {
	var out []*packages.Package
	for _, p := range res.Packages {
		if p.PkgPath == "" {
			continue
		}
		if res.ModulePath != "" && !strings.HasPrefix(p.PkgPath, res.ModulePath) {
			continue
		}
		out = append(out, p)
	}
	return out
}
