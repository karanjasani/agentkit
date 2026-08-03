package pathutil

import (
	"go/token"
	"path/filepath"
	"testing"
)

func TestRel(t *testing.T) {
	moduleDir := filepath.FromSlash("/home/user/proj")
	cases := []struct {
		file string
		want string
	}{
		{filepath.Join(moduleDir, "internal", "auth", "token.go"), "internal/auth/token.go"},
		{filepath.Join(moduleDir, "main.go"), "main.go"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := Rel(moduleDir, tc.file); got != tc.want {
			t.Errorf("Rel(%q) = %q, want %q", tc.file, got, tc.want)
		}
	}
}

func TestRelOutsideModule(t *testing.T) {
	got := Rel(filepath.FromSlash("/a/b"), filepath.FromSlash("/c/d/e.go"))
	if got != "/c/d/e.go" {
		t.Errorf("Rel outside module = %q", got)
	}
}

func TestLocInvalidPos(t *testing.T) {
	fset := token.NewFileSet()
	loc := Loc(fset, "/x", token.NoPos)
	if loc.File != "" || loc.Line != 0 {
		t.Errorf("expected empty location, got %+v", loc)
	}
}

func TestLocValid(t *testing.T) {
	fset := token.NewFileSet()
	f := fset.AddFile(filepath.FromSlash("/x/main.go"), -1, 100)
	pos := f.Pos(0)
	loc := Loc(fset, filepath.FromSlash("/x"), pos)
	if loc.File != "main.go" || loc.Line != 1 {
		t.Errorf("Loc = %+v, want main.go:1", loc)
	}
}
