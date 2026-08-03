package endpoints

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestNetHTTPDetector(t *testing.T) {
	src := `package p
import "net/http"
func reg(mux *http.ServeMux) {
	mux.HandleFunc("GET /a", handler)
	mux.HandleFunc("/b", handler)
}
func handler(http.ResponseWriter, *http.Request) {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	routes := netHTTPDetector{}.Detect(f, nil)
	if len(routes) != 2 {
		t.Fatalf("got %d routes, want 2", len(routes))
	}
	if routes[0].Method != "GET" || routes[0].Path != "/a" {
		t.Errorf("route0 = %s %s", routes[0].Method, routes[0].Path)
	}
	if routes[1].Method != "ANY" || routes[1].Path != "/b" {
		t.Errorf("route1 = %s %s", routes[1].Method, routes[1].Path)
	}
}

func TestChiDetector(t *testing.T) {
	src := `package p
func reg(r Router) {
	r.Get("/x", h)
	r.Post("/y", h)
	r.Method("PUT", "/z", h)
}
`
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "x.go", src, 0)
	routes := chiDetector{}.Detect(f, nil)
	methods := map[string]string{}
	for _, r := range routes {
		methods[r.Method] = r.Path
	}
	if methods["GET"] != "/x" || methods["POST"] != "/y" || methods["PUT"] != "/z" {
		t.Errorf("chi routes = %+v", methods)
	}
}

func TestGinDetector(t *testing.T) {
	src := `package p
func reg(r Engine) {
	r.GET("/x", h)
	r.DELETE("/y", h)
}
`
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "x.go", src, 0)
	routes := ginDetector{}.Detect(f, nil)
	if len(routes) != 2 {
		t.Fatalf("got %d routes, want 2", len(routes))
	}
}

func TestEchoDetector(t *testing.T) {
	src := `package p
func reg(e Echo) {
	e.GET("/x", h)
}
`
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "x.go", src, 0)
	routes := echoDetector{}.Detect(f, nil)
	if len(routes) != 1 || routes[0].Path != "/x" {
		t.Errorf("echo routes = %+v", routes)
	}
}

func TestGorillaDetector(t *testing.T) {
	src := `package p
func reg(r Router) {
	r.HandleFunc("/x", h).Methods("GET", "POST")
}
`
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "x.go", src, 0)
	routes := gorillaDetector{}.Detect(f, nil)
	if len(routes) != 2 {
		t.Fatalf("got %d routes, want 2 (GET, POST)", len(routes))
	}
	if routes[0].Path != "/x" {
		t.Errorf("gorilla route path = %s", routes[0].Path)
	}
}

func TestSplitMethodPath(t *testing.T) {
	m, p := splitMethodPath("GET /a/b")
	if m != "GET" || p != "/a/b" {
		t.Errorf("got %s %s", m, p)
	}
	m, p = splitMethodPath("/only/path")
	if m != "" || p != "/only/path" {
		t.Errorf("got %q %q", m, p)
	}
}

func TestDetectorsRegistered(t *testing.T) {
	got := map[string]bool{}
	for _, d := range Detectors() {
		got[d.Name()] = true
	}
	for _, want := range []string{"net/http", "chi", "gin", "echo", "gorilla/mux"} {
		if !got[want] {
			t.Errorf("missing detector %q", want)
		}
	}
}
