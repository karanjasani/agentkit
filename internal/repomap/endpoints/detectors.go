package endpoints

import (
	"go/ast"
	"go/types"
	"strings"
)

// verbRoutes finds selector calls whose method name is an HTTP verb and whose
// first argument is a string path, e.g. r.GET("/x", handler) or r.Get(...).
// pkgHint, when non-empty, is matched against the receiver package path (via
// type info) to raise confidence and set the framework name.
func verbRoutes(file *ast.File, info *types.Info, framework, pkgHint string, caseExact bool) []Route {
	var routes []Route
	ast.Inspect(file, func(n ast.Node) bool {
		sel, call, ok := selectorCall(n)
		if !ok {
			return true
		}
		name := sel.Sel.Name
		verb := strings.ToUpper(name)
		if !httpVerbs[verb] {
			return true
		}
		if len(call.Args) < 2 {
			return true
		}
		path, ok := stringLit(call.Args[0])
		if !ok {
			return true
		}

		confidence := "possible"
		recvPkg := receiverPkgPath(sel, info)
		switch {
		case pkgHint != "" && strings.Contains(recvPkg, pkgHint):
			confidence = "direct"
		case caseExact && name == verb:
			// chi uses Title-case verbs (e.g. "Get"). Skip ALLCAPS verbs here so
			// they are attributed to gin/echo instead of double-counted.
			return true
		}

		routes = append(routes, Route{
			Method:     verb,
			Path:       path,
			Handler:    call.Args[len(call.Args)-1],
			Framework:  framework,
			Confidence: confidence,
			Pos:        call.Lparen,
		})
		return true
	})
	return routes
}

// netHTTPDetector recognizes ServeMux.HandleFunc / HandleFunc / Handle, including
// Go 1.22 method patterns ("GET /path").
type netHTTPDetector struct{}

func (netHTTPDetector) Name() string { return "net/http" }

func (netHTTPDetector) Detect(file *ast.File, info *types.Info) []Route {
	var routes []Route
	ast.Inspect(file, func(n ast.Node) bool {
		sel, call, ok := selectorCall(n)
		if !ok {
			return true
		}
		if sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle" {
			return true
		}
		if len(call.Args) < 2 {
			return true
		}
		pattern, ok := stringLit(call.Args[0])
		if !ok {
			return true
		}
		method, path := splitMethodPath(pattern)
		if method == "" {
			method = "ANY"
		}
		confidence := "possible"
		recvPkg := receiverPkgPath(sel, info)
		if strings.Contains(recvPkg, "net/http") {
			confidence = "direct"
		}
		routes = append(routes, Route{
			Method:     method,
			Path:       path,
			Handler:    call.Args[len(call.Args)-1],
			Framework:  "net/http",
			Confidence: confidence,
			Pos:        call.Lparen,
		})
		return true
	})
	return routes
}

// chiDetector recognizes chi routers: r.Get/Post/... (Title case) and r.Method.
type chiDetector struct{}

func (chiDetector) Name() string { return "chi" }

func (chiDetector) Detect(file *ast.File, info *types.Info) []Route {
	routes := verbRoutes(file, info, "chi", "go-chi/chi", true)
	// r.Method("GET", "/path", handler)
	ast.Inspect(file, func(n ast.Node) bool {
		sel, call, ok := selectorCall(n)
		if !ok || sel.Sel.Name != "Method" || len(call.Args) < 3 {
			return true
		}
		method, ok := stringLit(call.Args[0])
		if !ok {
			return true
		}
		path, ok := stringLit(call.Args[1])
		if !ok {
			return true
		}
		confidence := "possible"
		if strings.Contains(receiverPkgPath(sel, info), "go-chi/chi") {
			confidence = "direct"
		}
		routes = append(routes, Route{
			Method:     strings.ToUpper(method),
			Path:       path,
			Handler:    call.Args[2],
			Framework:  "chi",
			Confidence: confidence,
			Pos:        call.Lparen,
		})
		return true
	})
	return routes
}

// ginDetector recognizes gin routers: r.GET/POST/... (upper case).
type ginDetector struct{}

func (ginDetector) Name() string { return "gin" }

func (ginDetector) Detect(file *ast.File, info *types.Info) []Route {
	return verbRoutes(file, info, "gin", "gin-gonic/gin", false)
}

// echoDetector recognizes echo routers: e.GET/POST/... (upper case).
type echoDetector struct{}

func (echoDetector) Name() string { return "echo" }

func (echoDetector) Detect(file *ast.File, info *types.Info) []Route {
	return verbRoutes(file, info, "echo", "labstack/echo", false)
}

// gorillaDetector recognizes gorilla/mux: r.HandleFunc("/x", h).Methods("GET").
type gorillaDetector struct{}

func (gorillaDetector) Name() string { return "gorilla/mux" }

func (gorillaDetector) Detect(file *ast.File, info *types.Info) []Route {
	var routes []Route
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for the outer .Methods(...) call wrapping a HandleFunc call.
		outerSel, outerCall, ok := selectorCall(n)
		if !ok || outerSel.Sel.Name != "Methods" {
			return true
		}
		inner, ok := outerSel.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		innerSel, ok := inner.Fun.(*ast.SelectorExpr)
		if !ok || innerSel.Sel.Name != "HandleFunc" || len(inner.Args) < 2 {
			return true
		}
		path, ok := stringLit(inner.Args[0])
		if !ok {
			return true
		}
		confidence := "possible"
		if strings.Contains(receiverPkgPath(innerSel, info), "gorilla/mux") {
			confidence = "direct"
		}
		for _, arg := range outerCall.Args {
			if method, ok := stringLit(arg); ok {
				routes = append(routes, Route{
					Method:     strings.ToUpper(method),
					Path:       path,
					Handler:    inner.Args[len(inner.Args)-1],
					Framework:  "gorilla/mux",
					Confidence: confidence,
					Pos:        inner.Lparen,
				})
			}
		}
		return true
	})
	return routes
}
