package api_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/karanjasani/agentkit/pkg/api"
)

// TestImpactIntegration builds a throwaway git repo from a small module, commits
// it, modifies a package, and verifies change-impact analysis.
func TestImpactIntegration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()

	writeFile(t, dir, "go.mod", "module example.com/impact\n\ngo 1.24\n")
	writeFile(t, dir, "core/core.go", `package core

// Value returns a constant.
func Value() int { return 1 }
`)
	writeFile(t, dir, "app/app.go", `package app

import "example.com/impact/core"

// Run uses core.
func Run() int { return core.Value() }
`)

	gitInit(t, dir)

	// Modify the core package (a dependency of app).
	writeFile(t, dir, "core/core.go", `package core

// Value returns a constant.
func Value() int { return 2 }
`)

	a, err := api.New(context.Background(), api.WithDir(dir))
	if err != nil {
		t.Fatal(err)
	}
	imp, err := a.Impact(context.Background(), api.ImpactOptions{Base: "HEAD"})
	if err != nil {
		t.Fatalf("impact: %v", err)
	}

	if !contains(imp.ChangedPackages, "example.com/impact/core") {
		t.Errorf("expected core in changed packages, got %v", imp.ChangedPackages)
	}
	if !contains(imp.AffectedPackages, "example.com/impact/app") {
		t.Errorf("expected app in affected packages, got %v", imp.AffectedPackages)
	}
	if imp.RiskScore <= 0 {
		t.Errorf("expected positive risk score, got %d", imp.RiskScore)
	}
	if imp.RiskLevel == "" {
		t.Error("expected a risk level")
	}
}

// TestImpactModuleInSubdir places the Go module in a subdirectory of the git
// repo and changes one file of a package that also has an unchanged file. This
// exercises the repo-root-vs-module-root path translation and the public-API
// detection that only reports symbols defined in changed files.
func TestImpactModuleInSubdir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	repo := t.TempDir()

	// A file at the repo root, outside the module, that will also change; it must
	// be dropped by the module-relative translation.
	writeFile(t, repo, "README.md", "v1\n")

	writeFile(t, repo, "svc/go.mod", "module example.com/svc\n\ngo 1.24\n")
	writeFile(t, repo, "svc/core/core.go", `package core

// Value returns a constant.
func Value() int { return 1 }
`)
	// A second, unchanged file in the same package with its own exported symbol.
	writeFile(t, repo, "svc/core/extra.go", `package core

// Extra is unchanged and must not appear in the public-API diff.
func Extra() int { return 0 }
`)
	writeFile(t, repo, "svc/app/app.go", `package app

import "example.com/svc/core"

// Run uses core.
func Run() int { return core.Value() }
`)

	gitInit(t, repo)

	// Change README (outside module) and core.go (inside module).
	writeFile(t, repo, "README.md", "v2\n")
	writeFile(t, repo, "svc/core/core.go", `package core

// Value returns a constant.
func Value() int { return 2 }
`)

	moduleDir := filepath.Join(repo, "svc")
	a, err := api.New(context.Background(), api.WithDir(moduleDir))
	if err != nil {
		t.Fatal(err)
	}
	imp, err := a.Impact(context.Background(), api.ImpactOptions{Base: "HEAD"})
	if err != nil {
		t.Fatalf("impact: %v", err)
	}

	if !contains(imp.ChangedPackages, "example.com/svc/core") {
		t.Errorf("expected core in changed packages, got %v", imp.ChangedPackages)
	}
	if !contains(imp.AffectedPackages, "example.com/svc/app") {
		t.Errorf("expected app in affected packages, got %v", imp.AffectedPackages)
	}
	// The README change lives outside the module and must be translated away.
	for _, f := range imp.ChangedFiles {
		if f == "README.md" || filepath.IsAbs(f) {
			t.Errorf("unexpected changed file leaked: %q", f)
		}
	}
	// Value changed; Extra did not.
	var names []string
	for _, s := range imp.PublicAPIChanged {
		names = append(names, s.Name)
	}
	if !contains(names, "Value") {
		t.Errorf("expected Value in public API changes, got %v", names)
	}
	if contains(names, "Extra") {
		t.Errorf("Extra is unchanged and should not appear, got %v", names)
	}
}

func TestImpactOutsideRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	// A module dir that is not a git repo should return a typed git error.
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/nogit\n\ngo 1.24\n")
	writeFile(t, dir, "x.go", "package nogit\n")

	a, err := api.New(context.Background(), api.WithDir(dir))
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Impact(context.Background(), api.ImpactOptions{Base: "HEAD"})
	if err == nil {
		t.Fatal("expected error impact outside git repo")
	}
	if e := api.AsError(err); e.Code != api.CodeGitUnavailable {
		t.Errorf("code = %s, want %s", e.Code, api.CodeGitUnavailable)
	}
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("add", ".")
	run("commit", "-m", "initial")
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
