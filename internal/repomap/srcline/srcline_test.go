package srcline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLine(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.go")
	content := "line one\n  line two  \nline three\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New()
	if got := c.Line(file, 2); got != "line two" {
		t.Errorf("Line(2) = %q, want %q", got, "line two")
	}
	// Cached read returns the same result.
	if got := c.Line(file, 1); got != "line one" {
		t.Errorf("Line(1) = %q", got)
	}
	if got := c.Line(file, 99); got != "" {
		t.Errorf("Line(99) = %q, want empty", got)
	}
	if got := c.Line("", 1); got != "" {
		t.Errorf("Line(empty file) = %q", got)
	}
	if got := c.Line(filepath.Join(dir, "missing.go"), 1); got != "" {
		t.Errorf("Line(missing) = %q", got)
	}
}
