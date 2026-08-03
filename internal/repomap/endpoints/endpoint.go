package endpoints

import (
	"context"
	"go/ast"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/karanjasani/agentkit/internal/repomap/loader"
	"github.com/karanjasani/agentkit/internal/repomap/pathutil"
	"github.com/karanjasani/agentkit/internal/repomap/rerr"
	"github.com/karanjasani/agentkit/pkg/models"
)

// Trace resolves a route to its handler and, from the handler body, its
// orchestration chain and upstream calls.
func Trace(ctx context.Context, l *loader.Loader, method, path string) (models.Endpoint, error) {
	res, err := l.Load(ctx, loader.TierTypes, false)
	if err != nil {
		return models.Endpoint{}, rerr.New(rerr.LoadFailed, false, "%v", err)
	}
	method = strings.ToUpper(strings.TrimSpace(method))

	type located struct {
		route Route
		pkg   *packages.Package
	}
	var matches []located

	detectors := Detectors()
	for _, p := range res.Packages {
		if isTestVariant(p.PkgPath) {
			continue
		}
		for _, f := range p.Syntax {
			for _, d := range detectors {
				for _, r := range d.Detect(f, p.TypesInfo) {
					if routeMatches(r, method, path) {
						matches = append(matches, located{route: r, pkg: p})
					}
				}
			}
		}
	}

	if len(matches) == 0 {
		return models.Endpoint{}, rerr.New(rerr.EndpointNotFound, true,
			"no route found for %s %s", method, path)
	}

	// Deterministic, quality-ordered selection: direct confidence first, then by
	// location.
	sort.Slice(matches, func(i, j int) bool {
		if (matches[i].route.Confidence == "direct") != (matches[j].route.Confidence == "direct") {
			return matches[i].route.Confidence == "direct"
		}
		li := res.Fset.Position(matches[i].route.Pos)
		lj := res.Fset.Position(matches[j].route.Pos)
		if li.Filename != lj.Filename {
			return li.Filename < lj.Filename
		}
		return li.Line < lj.Line
	})
	best := matches[0]

	out := models.Endpoint{
		Method:     best.route.Method,
		Path:       best.route.Path,
		Framework:  best.route.Framework,
		Route:      pathutil.Loc(res.Fset, res.ModuleDir, best.route.Pos),
		Confidence: best.route.Confidence,
	}

	// Resolve handler declaration.
	hName := handlerName(best.route.Handler, best.pkg.TypesInfo)
	if fn, fpkg := findHandlerDecl(res, best.pkg, best.route.Handler, hName); fn != nil {
		out.Handler = &models.Symbol{
			Name:     fn.Name.Name,
			Kind:     "func",
			Package:  fpkg.PkgPath,
			Location: pathutil.Loc(res.Fset, res.ModuleDir, fn.Name.Pos()),
		}
		if fn.Body != nil {
			out.Orchestration = orchestrationChain(res, fpkg, fn.Body)
			out.Upstreams = analyzeFuncUpstreams(res, fpkg.TypesInfo, fn.Body)
			out.RequestType, out.ResponseType = requestResponseTypes(fpkg.TypesInfo, fn)
		}
	} else if hName != "" {
		out.Handler = &models.Symbol{Name: hName, Kind: "func", Package: best.pkg.PkgPath}
	}

	return out, nil
}

func routeMatches(r Route, method, path string) bool {
	if r.Path != path {
		return false
	}
	if method == "" || r.Method == "ANY" || r.Method == method {
		return true
	}
	return false
}

// findHandlerDecl locates the *ast.FuncDecl for a handler expression by name,
// searching the route's package first, then all module packages.
func findHandlerDecl(res *loader.Result, pkg *packages.Package, handler ast.Expr, name string) (*ast.FuncDecl, *packages.Package) {
	// If the handler is an identifier resolvable via type info, use its object.
	if info := pkg.TypesInfo; info != nil {
		if obj := handlerObject(handler, info); obj != nil {
			if fn, fpkg := declForObject(res, obj); fn != nil {
				return fn, fpkg
			}
		}
	}
	// Fall back to name match within the route's package.
	short := name
	if i := strings.LastIndex(short, "."); i >= 0 {
		short = short[i+1:]
	}
	if fn := findFuncDeclByName(pkg, short); fn != nil {
		return fn, pkg
	}
	return nil, nil
}

func handlerObject(handler ast.Expr, info *types.Info) types.Object {
	switch h := handler.(type) {
	case *ast.Ident:
		if obj := info.Uses[h]; obj != nil {
			return obj
		}
		return info.Defs[h]
	case *ast.SelectorExpr:
		return info.Uses[h.Sel]
	}
	return nil
}

func declForObject(res *loader.Result, obj types.Object) (*ast.FuncDecl, *packages.Package) {
	pos := obj.Pos()
	for _, p := range res.Packages {
		if isTestVariant(p.PkgPath) {
			continue
		}
		for _, f := range p.Syntax {
			for _, d := range f.Decls {
				if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Pos() == pos {
					return fn, p
				}
			}
		}
	}
	return nil, nil
}

func findFuncDeclByName(pkg *packages.Package, name string) *ast.FuncDecl {
	for _, f := range pkg.Syntax {
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == name {
				return fn
			}
		}
	}
	return nil
}

// orchestrationChain returns the module-internal functions called directly within
// a handler body, in source order.
func orchestrationChain(res *loader.Result, pkg *packages.Package, body ast.Node) []models.Symbol {
	var out []models.Symbol
	seen := map[types.Object]bool{}
	info := pkg.TypesInfo
	if info == nil {
		return out
	}
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var obj types.Object
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			obj = info.Uses[fn]
		case *ast.SelectorExpr:
			obj = info.Uses[fn.Sel]
		}
		fnObj, ok := obj.(*types.Func)
		if !ok || seen[fnObj] || fnObj.Pkg() == nil {
			return true
		}
		if res.ModulePath == "" || !strings.HasPrefix(fnObj.Pkg().Path(), res.ModulePath) {
			return true
		}
		seen[fnObj] = true
		out = append(out, models.Symbol{
			Name:     fnObj.Name(),
			Kind:     "func",
			Package:  fnObj.Pkg().Path(),
			Location: pathutil.Loc(res.Fset, res.ModuleDir, fnObj.Pos()),
		})
		return true
	})
	return out
}

// requestResponseTypes is a best-effort guess: the type decoded from the request
// body and the type encoded into the response, based on json usage in the body.
func requestResponseTypes(info *types.Info, fn *ast.FuncDecl) (reqType, respType string) {
	if info == nil || fn.Body == nil {
		return "", ""
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch sel.Sel.Name {
		case "Decode", "Unmarshal":
			idx := 0
			if sel.Sel.Name == "Unmarshal" {
				idx = 1
			}
			if len(call.Args) > idx {
				if t := derefTypeString(info, call.Args[idx]); t != "" && reqType == "" {
					reqType = t
				}
			}
		case "Encode", "Marshal":
			idx := 0
			if sel.Sel.Name == "Marshal" {
				idx = 0
			}
			if len(call.Args) > idx {
				if t := derefTypeString(info, call.Args[idx]); t != "" && respType == "" {
					respType = t
				}
			}
		}
		return true
	})
	return reqType, respType
}
