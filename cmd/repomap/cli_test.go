package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixture = "../../testdata/fixtures/sample"

func fixtureDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// runCLI executes the CLI in-process with os.Stdout redirected to a pipe and
// returns the exit code and captured stdout. A draining goroutine prevents the
// child write from blocking on a full pipe buffer.
func runCLI(t *testing.T, args ...string) (int, string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w

	outCh := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		outCh <- string(b)
	}()

	code := run(args)

	_ = w.Close()
	os.Stdout = old
	out := <-outCh
	_ = r.Close()
	return code, out
}

func withDir(dir string, rest ...string) []string {
	return append([]string{"--dir", dir}, rest...)
}

// TestCLICommandsJSON exercises every subcommand's happy path in JSON format.
func TestCLICommandsJSON(t *testing.T) {
	dir := fixtureDir(t)
	cases := []struct {
		name string
		args []string
	}{
		{"overview", []string{"overview"}},
		{"package", []string{"package", "example.com/sample/service"}},
		{"symbol_signature", []string{"symbol", "FetchWidget", "--signature-only"}},
		{"symbol_body", []string{"symbol", "Helper", "--body"}},
		{"symbol_doc", []string{"symbol", "FetchWidget", "--doc"}},
		{"callers", []string{"callers", "Helper"}},
		{"deps", []string{"deps", "example.com/sample"}},
		{"tests", []string{"tests", "FetchWidget"}},
		{"endpoint", []string{"endpoint", "GET", "/api/v1/widgets/{id}"}},
		{"upstreams", []string{"upstreams", "."}},
		{"struct", []string{"struct", "models.Widget"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, out := runCLI(t, withDir(dir, tc.args...)...)
			if code != 0 {
				t.Fatalf("exit = %d, want 0\noutput:\n%s", code, out)
			}
			if !strings.Contains(out, "\"schema\"") {
				t.Errorf("output missing schema envelope:\n%s", out)
			}
			if !strings.Contains(out, "\"ok\": true") {
				t.Errorf("expected ok=true envelope:\n%s", out)
			}
		})
	}
}

// TestCLICommandsText exercises the text renderer path through the CLI for
// every subcommand.
func TestCLICommandsText(t *testing.T) {
	dir := fixtureDir(t)
	cases := [][]string{
		{"overview"},
		{"package", "example.com/sample/service"},
		{"symbol", "FetchWidget", "--body"},
		{"callers", "Helper"},
		{"deps", "example.com/sample"},
		{"tests", "FetchWidget"},
		{"endpoint", "GET", "/api/v1/widgets/{id}"},
		{"upstreams", "."},
		{"struct", "models.Widget"},
	}
	for _, args := range cases {
		t.Run(args[0], func(t *testing.T) {
			full := append([]string{"--format", "text"}, withDir(dir, args...)...)
			code, out := runCLI(t, full...)
			if code != 0 {
				t.Fatalf("exit = %d, want 0\noutput:\n%s", code, out)
			}
			if strings.TrimSpace(out) == "" {
				t.Errorf("text output was empty")
			}
			if strings.Contains(out, "\"schema\"") {
				t.Errorf("text output unexpectedly contained JSON envelope:\n%s", out)
			}
		})
	}
}

// TestCLIImpact runs impact against the fixture. Depending on the working-tree
// state it may succeed or return a structured error; either way it must emit a
// well-formed JSON envelope and a sane exit code.
func TestCLIImpact(t *testing.T) {
	dir := fixtureDir(t)
	code, out := runCLI(t, withDir(dir, "impact", "--base", "HEAD")...)
	if !strings.Contains(out, "\"schema\"") {
		t.Errorf("impact output missing schema envelope:\n%s", out)
	}
	switch code {
	case 0, 1, 2:
	default:
		t.Errorf("unexpected impact exit code %d", code)
	}
}

func TestCLISymbolNotFound(t *testing.T) {
	dir := fixtureDir(t)
	code, out := runCLI(t, withDir(dir, "symbol", "DoesNotExist")...)
	if code != 2 {
		t.Errorf("exit = %d, want 2 (not found)\n%s", code, out)
	}
	if !strings.Contains(out, "\"ok\": false") {
		t.Errorf("expected ok=false envelope:\n%s", out)
	}
	if !strings.Contains(out, "SYMBOL_NOT_FOUND") {
		t.Errorf("expected SYMBOL_NOT_FOUND code:\n%s", out)
	}
}

func TestCLIInvalidFormat(t *testing.T) {
	dir := fixtureDir(t)
	code, out := runCLI(t, withDir(dir, "overview", "--format", "yaml")...)
	if code != 3 {
		t.Errorf("exit = %d, want 3 (usage)\n%s", code, out)
	}
	if !strings.Contains(out, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT code:\n%s", out)
	}
}

func TestCLIInvalidDir(t *testing.T) {
	code, out := runCLI(t, "overview", "--dir", "/nonexistent/path/xyz")
	if code != 3 {
		t.Errorf("exit = %d, want 3\n%s", code, out)
	}
	if !strings.Contains(out, "INVALID_ARGUMENT") {
		t.Errorf("expected INVALID_ARGUMENT:\n%s", out)
	}
}

// TestCLIAllCommandsInvalidDir drives every subcommand with an invalid module
// directory so each command's analyzer-construction error path is exercised.
// Valid argument counts are supplied so cobra's arg validation passes and the
// RunE body runs.
func TestCLIAllCommandsInvalidDir(t *testing.T) {
	const bad = "/nonexistent/path/xyz"
	cmds := [][]string{
		{"overview"},
		{"package", "x"},
		{"symbol", "X"},
		{"callers", "X"},
		{"deps", "x"},
		{"impact"},
		{"tests", "X"},
		{"endpoint", "GET", "/x"},
		{"upstreams", "x"},
		{"struct", "X"},
	}
	for _, base := range cmds {
		args := append([]string{"--dir", bad}, base...)
		code, out := runCLI(t, args...)
		if code != 3 {
			t.Errorf("%v exit = %d, want 3\n%s", base, code, out)
		}
		if !strings.Contains(out, "INVALID_ARGUMENT") {
			t.Errorf("%v expected INVALID_ARGUMENT:\n%s", base, out)
		}
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	code, _ := runCLI(t, "bogus-command")
	if code != 3 {
		t.Errorf("exit = %d, want 3 (usage error)", code)
	}
}

func TestCLIMissingArg(t *testing.T) {
	code, _ := runCLI(t, "symbol")
	if code != 3 {
		t.Errorf("exit = %d, want 3 (usage error)", code)
	}
}

func TestCLIVersion(t *testing.T) {
	code, out := runCLI(t, "--version")
	if code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("expected version output")
	}
}
