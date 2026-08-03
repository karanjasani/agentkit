package endpoints

import (
	"go/ast"
	"go/parser"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/karanjasani/agentkit/internal/repomap/loader"
	"github.com/karanjasani/agentkit/pkg/models"
)

func TestSortUpstreams(t *testing.T) {
	u := []models.Upstream{
		{URL: "b", Location: models.Location{File: "a.go", Line: 2}},
		{URL: "z", Location: models.Location{File: "b.go", Line: 1}},
		{URL: "a", Location: models.Location{File: "a.go", Line: 2}},
		{URL: "a", Location: models.Location{File: "a.go", Line: 1}},
	}
	sortUpstreams(u)
	got := make([]string, len(u))
	for i, c := range u {
		got[i] = c.Location.File + ":" + itoaLocal(c.Location.Line) + ":" + c.URL
	}
	want := []string{"a.go:1:a", "a.go:2:a", "a.go:2:b", "b.go:1:z"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pos %d = %q, want %q (%v)", i, got[i], want[i], got)
		}
	}
}

func itoaLocal(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return "?"
}

func TestMatchesPath(t *testing.T) {
	res := &loader.Result{ModuleDir: "/m"}
	p := &packages.Package{PkgPath: "example.com/m/svc", GoFiles: []string{"/m/svc/x.go"}}

	cases := []struct {
		clean, raw string
		want       bool
	}{
		{"", ".", true}, // wildcard
		{"example.com/m/svc", "example.com/m/svc", true}, // exact path
		{"example.com/m", "example.com/m", true},         // path prefix
		{"svc", "svc", true},                             // directory match
		{"other", "other", false},                        // no match
	}
	for _, tc := range cases {
		if got := matchesPath(res, p, tc.clean, tc.raw); got != tc.want {
			t.Errorf("matchesPath(clean=%q, raw=%q) = %v, want %v", tc.clean, tc.raw, got, tc.want)
		}
	}
}

func parseExpr(t *testing.T, src string) ast.Expr {
	t.Helper()
	e, err := parser.ParseExpr(src)
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	return e
}

func TestHandlerName(t *testing.T) {
	cases := map[string]string{
		"handler":             "handler",
		"pkg.Handler":         "pkg.Handler",
		"middleware(handler)": "middleware",
	}
	for src, want := range cases {
		if got := handlerName(parseExpr(t, src)); got != want {
			t.Errorf("handlerName(%q) = %q, want %q", src, got, want)
		}
	}
	// Function literal.
	if got := handlerName(parseExpr(t, "func(){}")); got != "func(...)" {
		t.Errorf("handlerName(funclit) = %q", got)
	}
	// Nested selector: X is itself a selector, so only the final name is used.
	if got := handlerName(parseExpr(t, "a.b.Handler")); got != "Handler" {
		t.Errorf("handlerName(nested selector) = %q, want Handler", got)
	}
}

func TestResolveURLLiteralAndIdent(t *testing.T) {
	if got := resolveURL(parseExpr(t, `"https://x"`), nil); got != "https://x" {
		t.Errorf("literal url = %q", got)
	}
	if got := resolveURL(parseExpr(t, "baseURL"), nil); got != "baseURL" {
		t.Errorf("ident url = %q", got)
	}
	if got := resolveURL(parseExpr(t, "cfg.URL"), nil); got != "URL" {
		t.Errorf("selector url = %q", got)
	}
}

func TestRequestMethodURL(t *testing.T) {
	call := parseExpr(t, `http.NewRequest("POST", "https://x", nil)`).(*ast.CallExpr)
	m, u := requestMethodURL(call, nil, "NewRequest")
	if m != "POST" || u != "https://x" {
		t.Errorf("got %q %q", m, u)
	}

	callCtx := parseExpr(t, `http.NewRequestWithContext(ctx, "GET", "https://y", nil)`).(*ast.CallExpr)
	m, u = requestMethodURL(callCtx, nil, "NewRequestWithContext")
	if m != "GET" || u != "https://y" {
		t.Errorf("withctx got %q %q", m, u)
	}
}

func TestVerbIsHTTP(t *testing.T) {
	sel := parseExpr(t, "http.Get").(*ast.SelectorExpr)
	if !verbIsHTTP(nil, sel) {
		t.Error("expected http.Get to be recognized")
	}
	other := parseExpr(t, "client.Get").(*ast.SelectorExpr)
	if verbIsHTTP(nil, other) {
		t.Error("client.Get should not match without type info")
	}
}

func TestConfidenceFor(t *testing.T) {
	sel := parseExpr(t, "http.Get").(*ast.SelectorExpr)
	if got := confidenceFor(nil, sel, "net/http"); got != "direct" {
		t.Errorf("confidence = %q, want direct", got)
	}
	other := parseExpr(t, "c.Get").(*ast.SelectorExpr)
	if got := confidenceFor(nil, other, "net/http"); got != "possible" {
		t.Errorf("confidence = %q, want possible", got)
	}
}

func TestVerbMethod(t *testing.T) {
	cases := map[string]string{
		"Get": "GET", "get": "GET",
		"Post": "POST", "PostForm": "POST",
		"Put": "PUT", "Delete": "DELETE", "Patch": "PATCH",
		"Head": "HEAD", "Options": "OPTIONS",
		"Do": "", "Fetch": "", "": "",
	}
	for in, want := range cases {
		if got := verbMethod(in); got != want {
			t.Errorf("verbMethod(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsGRPCDialAndURLString(t *testing.T) {
	for _, n := range []string{"Dial", "DialContext", "NewClient"} {
		if !isGRPCDial(n) {
			t.Errorf("isGRPCDial(%q) = false, want true", n)
		}
	}
	if isGRPCDial("Get") {
		t.Error("isGRPCDial(Get) = true, want false")
	}
	if !isURLString("https://x") || !isURLString("http://y") {
		t.Error("expected http(s) URLs to be recognized")
	}
	if isURLString("ftp://z") || isURLString("orders:50051") {
		t.Error("non-http scheme should not be a URL string")
	}
}

func TestFirstStringArg(t *testing.T) {
	// With nil type info, firstStringArg falls back to the first string literal.
	args := []ast.Expr{parseExpr(t, "ctx"), parseExpr(t, `"orders:50051"`)}
	if got, ok := firstStringArg(nil, args); !ok || got != "orders:50051" {
		t.Errorf("firstStringArg = %q, %v; want orders:50051, true", got, ok)
	}
	none := []ast.Expr{parseExpr(t, "ctx"), parseExpr(t, "opts")}
	if got, ok := firstStringArg(nil, none); ok {
		t.Errorf("firstStringArg(no literal) = %q, %v; want \"\", false", got, ok)
	}
}

func TestDecodeArgTypeAndDerefNilInfo(t *testing.T) {
	call := parseExpr(t, `d.Decode(&x)`).(*ast.CallExpr)
	if got := decodeArgType(call, nil); got != "" {
		t.Errorf("decodeArgType(nil info) = %q, want empty", got)
	}
	empty := parseExpr(t, `d.Decode()`).(*ast.CallExpr)
	if got := decodeArgType(empty, nil); got != "" {
		t.Errorf("decodeArgType(no args) = %q, want empty", got)
	}
	if got := derefTypeString(nil, parseExpr(t, "x")); got != "" {
		t.Errorf("derefTypeString(nil info) = %q, want empty", got)
	}
}

func TestKnownHTTPClientNilInfo(t *testing.T) {
	sel := parseExpr(t, "client.Get").(*ast.SelectorExpr)
	if knownHTTPClient(sel, nil) {
		t.Error("knownHTTPClient with nil info should be false")
	}
}
