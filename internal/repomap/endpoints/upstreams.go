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

// UpstreamCalls maps outbound calls originating under a package path.
func UpstreamCalls(ctx context.Context, l *loader.Loader, path string) (models.Upstreams, error) {
	res, err := l.Load(ctx, loader.TierTypes, false)
	if err != nil {
		return models.Upstreams{}, rerr.New(rerr.LoadFailed, false, "%v", err)
	}

	clean := strings.TrimSuffix(strings.TrimPrefix(path, "./"), "/")
	var calls []models.Upstream

	for _, p := range res.Packages {
		if p.Types == nil || isTestVariant(p.PkgPath) {
			continue
		}
		if !matchesPath(res, p, clean, path) {
			continue
		}
		for _, f := range p.Syntax {
			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				calls = append(calls, analyzeFuncUpstreams(res, p.TypesInfo, fn.Body)...)
			}
		}
	}

	sortUpstreams(calls)
	return models.Upstreams{Root: clean, Calls: calls}, nil
}

func matchesPath(res *loader.Result, p *packages.Package, clean, raw string) bool {
	if raw == "" || raw == "." || clean == "" {
		return true
	}
	if p.PkgPath == raw || p.PkgPath == clean {
		return true
	}
	if strings.HasPrefix(p.PkgPath, clean) {
		return true
	}
	// Directory match.
	dir := ""
	files := p.GoFiles
	if len(files) > 0 {
		rel := pathutil.Rel(res.ModuleDir, files[0])
		if i := strings.LastIndex(rel, "/"); i >= 0 {
			dir = rel[:i]
		}
	}
	return dir == clean || strings.HasPrefix(dir, clean+"/")
}

// analyzeFuncUpstreams scans a function body for outbound HTTP calls and pairs
// them with any JSON decode targets found in the same body.
func analyzeFuncUpstreams(res *loader.Result, info *types.Info, body ast.Node) []models.Upstream {
	var requests []models.Upstream
	var decodeType string

	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		name := sel.Sel.Name

		// Request construction / issuance.
		switch {
		case name == "NewRequest" || name == "NewRequestWithContext":
			method, url := requestMethodURL(call, info, name)
			requests = append(requests, models.Upstream{
				Method:     method,
				URL:        url,
				Location:   pathutil.Loc(res.Fset, res.ModuleDir, call.Lparen),
				Confidence: confidenceFor(info, sel, "net/http"),
			})
		case isGRPCDial(name):
			if u, ok := grpcUpstream(res, info, sel, call); ok {
				requests = append(requests, u)
			}
		case verbMethod(name) != "":
			if u, ok := httpVerbUpstream(res, info, sel, call, name); ok {
				requests = append(requests, u)
			}
		case name == "Decode":
			if t := decodeArgType(call, info); t != "" {
				decodeType = t
			}
		case name == "Unmarshal":
			if len(call.Args) >= 2 {
				if t := derefTypeString(info, call.Args[1]); t != "" {
					decodeType = t
				}
			}
		}
		return true
	})

	for i := range requests {
		if requests[i].DecodeType == "" {
			requests[i].DecodeType = decodeType
		}
	}
	return requests
}

func requestMethodURL(call *ast.CallExpr, info *types.Info, fnName string) (string, string) {
	// NewRequest(method, url, body); NewRequestWithContext(ctx, method, url, body).
	base := 0
	if fnName == "NewRequestWithContext" {
		base = 1
	}
	method := ""
	url := ""
	if len(call.Args) > base {
		method = strings.ToUpper(strings.Trim(resolveURL(call.Args[base], info), "\""))
	}
	if len(call.Args) > base+1 {
		url = resolveURL(call.Args[base+1], info)
	}
	return method, url
}

// resolveURL returns a string literal or a resolved constant value, else "".
func resolveURL(e ast.Expr, info *types.Info) string {
	if s, ok := stringLit(e); ok {
		return s
	}
	if info != nil {
		if tv, ok := info.Types[e]; ok && tv.Value != nil {
			return strings.Trim(tv.Value.String(), "\"")
		}
	}
	// Fall back to the identifier name so there is still an anchorable hint.
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return v.Sel.Name
	}
	return ""
}

func decodeArgType(call *ast.CallExpr, info *types.Info) string {
	if len(call.Args) == 0 {
		return ""
	}
	return derefTypeString(info, call.Args[0])
}

func derefTypeString(info *types.Info, e ast.Expr) string {
	if info == nil {
		return ""
	}
	t := info.TypeOf(e)
	if t == nil {
		return ""
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	return types.TypeString(t, func(p *types.Package) string { return p.Name() })
}

func verbIsHTTP(info *types.Info, sel *ast.SelectorExpr) bool {
	pkg := receiverPkgPath(sel, info)
	if strings.Contains(pkg, "net/http") {
		return true
	}
	// http.Get style: X is the http package identifier.
	if id, ok := sel.X.(*ast.Ident); ok && id.Name == "http" {
		return true
	}
	return false
}

func confidenceFor(info *types.Info, sel *ast.SelectorExpr, pkgHint string) string {
	if strings.Contains(receiverPkgPath(sel, info), pkgHint) {
		return "direct"
	}
	if id, ok := sel.X.(*ast.Ident); ok && id.Name == "http" {
		return "direct"
	}
	return "possible"
}

// httpClientPkgs are import-path substrings of well-known Go HTTP client
// libraries. A verb call whose receiver resolves to one of these is treated as
// a first-party outbound HTTP call with "direct" confidence. The list is a
// best-effort allow-list; unknown clients are still caught by the URL-literal
// fallback in httpVerbUpstream, just with "possible" confidence.
var httpClientPkgs = []string{
	"net/http",
	"go-resty/resty",
	"hashicorp/go-retryablehttp",
	"valyala/fasthttp",
	"imroc/req",
	"parnurzeal/gorequest",
	"dghubble/sling",
	"h2non/gentleman",
	"levigross/grequests",
	"gojek/heimdall",
	"monaco-io/request",
	"go-kit/kit/transport/http",
}

// verbMethod maps a selector method name to an HTTP method, or "" if the name is
// not an HTTP verb. PostForm is treated as POST.
func verbMethod(name string) string {
	switch strings.ToUpper(name) {
	case "GET":
		return "GET"
	case "POST", "POSTFORM":
		return "POST"
	case "PUT":
		return "PUT"
	case "DELETE":
		return "DELETE"
	case "PATCH":
		return "PATCH"
	case "HEAD":
		return "HEAD"
	case "OPTIONS":
		return "OPTIONS"
	}
	return ""
}

// knownHTTPClient reports whether the selector's receiver type resolves to one
// of the recognized HTTP client packages.
func knownHTTPClient(sel *ast.SelectorExpr, info *types.Info) bool {
	pkg := receiverPkgPath(sel, info)
	if pkg == "" {
		return false
	}
	for _, k := range httpClientPkgs {
		if strings.Contains(pkg, k) {
			return true
		}
	}
	return false
}

// httpVerbUpstream builds an Upstream for a verb-style call such as
// http.Get(url), client.Post(url, ...), or resty R().Get(url). It records the
// call when it can tell the call is HTTP: either the receiver resolves to
// net/http or a known client package (→ "direct"), or the first argument is a
// literal/const URL (→ "possible"). This lets it cover arbitrary third-party
// clients and hand-rolled wrappers without an exhaustive package list.
func httpVerbUpstream(res *loader.Result, info *types.Info, sel *ast.SelectorExpr, call *ast.CallExpr, name string) (models.Upstream, bool) {
	url := ""
	if len(call.Args) > 0 {
		url = resolveURL(call.Args[0], info)
	}
	isHTTP := verbIsHTTP(info, sel)
	known := knownHTTPClient(sel, info)
	if !isHTTP && !known && !isURLString(url) {
		return models.Upstream{}, false
	}
	confidence := "possible"
	if isHTTP || known {
		confidence = "direct"
	}
	return models.Upstream{
		Method:     verbMethod(name),
		URL:        url,
		Location:   pathutil.Loc(res.Fset, res.ModuleDir, call.Lparen),
		Confidence: confidence,
	}, true
}

// isURLString reports whether s looks like an absolute HTTP(S) URL. It is the
// signal used to recognize outbound calls made through unknown clients.
func isURLString(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// isGRPCDial reports whether a selector name is a gRPC connection constructor.
func isGRPCDial(name string) bool {
	return name == "Dial" || name == "DialContext" || name == "NewClient"
}

// grpcUpstream builds an Upstream for a grpc.Dial / grpc.DialContext /
// grpc.NewClient call, using the dial target as the URL and "GRPC" as the
// method. Only calls that resolve to google.golang.org/grpc (or a receiver
// identifier literally named "grpc") are recorded, to avoid matching unrelated
// Dial methods.
func grpcUpstream(res *loader.Result, info *types.Info, sel *ast.SelectorExpr, call *ast.CallExpr) (models.Upstream, bool) {
	isGRPC := strings.Contains(receiverPkgPath(sel, info), "google.golang.org/grpc")
	if id, ok := sel.X.(*ast.Ident); ok && id.Name == "grpc" {
		isGRPC = true
	}
	if !isGRPC {
		return models.Upstream{}, false
	}
	target, _ := firstStringArg(info, call.Args)
	return models.Upstream{
		Service:    "grpc",
		Method:     "GRPC",
		URL:        target,
		Location:   pathutil.Loc(res.Fset, res.ModuleDir, call.Lparen),
		Confidence: "direct",
	}, true
}

// firstStringArg returns the resolved value of the first string-typed argument.
// It uses type info to skip non-string arguments (e.g. the context passed to
// DialContext), falling back to the first string literal when type info is
// unavailable.
func firstStringArg(info *types.Info, args []ast.Expr) (string, bool) {
	for _, a := range args {
		if info != nil {
			if t := info.TypeOf(a); t != nil {
				if b, ok := t.Underlying().(*types.Basic); ok && b.Info()&types.IsString != 0 {
					return resolveURL(a, info), true
				}
				continue
			}
		}
		if s, ok := stringLit(a); ok {
			return s, true
		}
	}
	return "", false
}

func sortUpstreams(u []models.Upstream) {
	sort.Slice(u, func(i, j int) bool {
		if u[i].Location.File != u[j].Location.File {
			return u[i].Location.File < u[j].Location.File
		}
		if u[i].Location.Line != u[j].Location.Line {
			return u[i].Location.Line < u[j].Location.Line
		}
		return u[i].URL < u[j].URL
	})
}

func isTestVariant(pkgPath string) bool {
	return strings.Contains(pkgPath, ".test") || strings.Contains(pkgPath, "[")
}
