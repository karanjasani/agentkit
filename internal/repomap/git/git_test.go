package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karanjasani/agentkit/internal/repomap/rerr"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
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

// TestChangedFilesRejectsOptionInjection ensures a base ref that looks like a
// command-line option is rejected rather than passed through to git, where it
// could be interpreted as e.g. --output=<path> and write to an arbitrary file.
func TestChangedFilesRejectsOptionInjection(t *testing.T) {
	if !Available() {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	runGit(t, dir, "init")

	for _, bad := range []string{"--output=/tmp/pwned", "-O/tmp/x", "--ext-diff"} {
		_, err := ChangedFiles(context.Background(), dir, bad)
		if err == nil {
			t.Fatalf("expected rejection for base %q", bad)
		}
		if e := rerr.As(err); e.Code != rerr.InvalidArgument {
			t.Errorf("base %q: code = %s, want %s", bad, e.Code, rerr.InvalidArgument)
		}
	}
}

// TestChangedFilesInRepo exercises the success paths of ChangedFiles (default
// base, tracked-modification diff, and untracked listing) and RepoRoot against
// a real throwaway repository.
func TestChangedFilesInRepo(t *testing.T) {
	if !Available() {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "init")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")

	// Modify the tracked file and add an untracked one.
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Empty base must default to HEAD.
	files, err := ChangedFiles(context.Background(), dir, "")
	if err != nil {
		t.Fatalf("ChangedFiles: %v", err)
	}
	joined := strings.Join(files, ",")
	if !strings.Contains(joined, "tracked.txt") {
		t.Errorf("expected modified tracked.txt in %v", files)
	}
	if !strings.Contains(joined, "untracked.txt") {
		t.Errorf("expected untracked.txt in %v", files)
	}

	root, err := RepoRoot(context.Background(), dir)
	if err != nil {
		t.Fatalf("RepoRoot: %v", err)
	}
	if root == "" {
		t.Error("expected non-empty repo root")
	}
}

func TestRepoRootErrorsOutsideRepo(t *testing.T) {
	if !Available() {
		t.Skip("git not available")
	}
	// A temp dir is not a git repository.
	_, err := RepoRoot(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for non-repo directory")
	}
	if e := rerr.As(err); e.Code != rerr.GitUnavailable {
		t.Errorf("code = %s, want %s", e.Code, rerr.GitUnavailable)
	}
}

func TestChangedFilesErrorsOutsideRepo(t *testing.T) {
	if !Available() {
		t.Skip("git not available")
	}
	_, err := ChangedFiles(context.Background(), t.TempDir(), "HEAD")
	if err == nil {
		t.Fatal("expected error diffing in non-repo directory")
	}
}
