// Package endpoints implements the heuristic repomap commands: endpoint and
// upstreams. Route detection is pluggable via the RouteDetector interface, with
// built-in detectors for net/http, chi, gin, echo and gorilla/mux.
//
// These commands pattern-match framework idioms, so every result carries a
// confidence field and a file:line anchor; they never present a guess as fact.
package endpoints

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// Route is a detected route registration.
type Route struct {
	Method     string
	Path       string
	Handler    ast.Expr
	Framework  string
	Confidence string
	Pos        token.Pos
}

// RouteDetector recognizes route registrations for a particular framework.
type RouteDetector interface {
	// Name is the framework identifier (e.g. "chi").
	Name() string
	// Detect returns routes found in a single file. info may be nil if type
	// information is unavailable, in which case detectors fall back to syntax.
	Detect(file *ast.File, info *types.Info) []Route
}

// Detectors returns the built-in detectors in a stable order.
func Detectors() []RouteDetector {
	return []RouteDetector{
		netHTTPDetector{},
		chiDetector{},
		ginDetector{},
		echoDetector{},
		gorillaDetector{},
	}
}

// httpVerbs are the recognized HTTP method selector names (case-insensitive).
var httpVerbs = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"PATCH": true, "HEAD": true, "OPTIONS": true, "CONNECT": true, "TRACE": true,
}

// selectorCall extracts the selector name and call expression if n is a method
// call like recv.Sel(args...).
func selectorCall(n ast.Node) (*ast.SelectorExpr, *ast.CallExpr, bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, nil, false
	}
	return sel, call, true
}

// stringLit returns the unquoted value of a string literal expression.
func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	return strings.Trim(lit.Value, "`\""), true
}

// receiverPkgPath returns the import path of the type on the left of a selector,
// using type info when available.
func receiverPkgPath(sel *ast.SelectorExpr, info *types.Info) string {
	if info == nil {
		return ""
	}
	tv, ok := info.Types[sel.X]
	if !ok || tv.Type == nil {
		// Try Uses for identifiers.
		if id, ok := sel.X.(*ast.Ident); ok {
			if obj := info.Uses[id]; obj != nil && obj.Type() != nil {
				return typePkgPath(obj.Type())
			}
		}
		return ""
	}
	return typePkgPath(tv.Type)
}

func typePkgPath(t types.Type) string {
	switch u := t.(type) {
	case *types.Pointer:
		return typePkgPath(u.Elem())
	case *types.Named:
		if u.Obj() != nil && u.Obj().Pkg() != nil {
			return u.Obj().Pkg().Path()
		}
	}
	return ""
}

// handlerInfo resolves a handler expression to a display name.
func handlerName(e ast.Expr, info *types.Info) string {
	switch h := e.(type) {
	case *ast.Ident:
		return h.Name
	case *ast.SelectorExpr:
		if x, ok := h.X.(*ast.Ident); ok {
			return x.Name + "." + h.Sel.Name
		}
		return h.Sel.Name
	case *ast.FuncLit:
		return "func(...)"
	case *ast.CallExpr:
		// e.g. middleware(handler) — best-effort.
		return handlerName(h.Fun, info)
	}
	return ""
}

// splitMethodPath splits a Go 1.22 ServeMux pattern like "GET /a/b" into method
// and path. If no method prefix is present, method is "" and path is the whole
// pattern.
func splitMethodPath(pattern string) (method, path string) {
	pattern = strings.TrimSpace(pattern)
	if i := strings.IndexByte(pattern, ' '); i > 0 {
		m := strings.ToUpper(pattern[:i])
		if httpVerbs[m] {
			return m, strings.TrimSpace(pattern[i+1:])
		}
	}
	return "", pattern
}
