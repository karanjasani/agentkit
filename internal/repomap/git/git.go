// Package git provides the minimal, read-only git interaction needed for change
// impact analysis. It shells out to the git binary for speed and to avoid a
// heavy pure-Go git dependency; when git is unavailable it returns a typed
// GIT_UNAVAILABLE error so callers can surface a clear message.
package git

import (
	"context"
	"os/exec"
	"sort"
	"strings"

	"github.com/karanjasani/agentkit/internal/repomap/rerr"
)

// Available reports whether the git binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// ChangedFiles returns the set of files that differ between base and the current
// working tree (committed diff plus uncommitted changes), as paths relative to
// the repository root. Results are sorted and de-duplicated.
func ChangedFiles(ctx context.Context, dir, base string) ([]string, error) {
	if !Available() {
		return nil, rerr.New(rerr.GitUnavailable, false, "git is required for impact analysis but was not found on PATH")
	}
	if base == "" {
		base = "HEAD"
	}
	if err := validateRef(base); err != nil {
		return nil, err
	}

	set := map[string]bool{}

	// Committed changes: diff base against the working tree. "--end-of-options"
	// forces base to be parsed as a revision, never as an option, as a second
	// layer of defense behind validateRef.
	if out, err := run(ctx, dir, "diff", "--name-only", "--end-of-options", base); err == nil {
		addLines(set, out)
	} else {
		return nil, rerr.New(rerr.GitUnavailable, true, "git diff failed: %v", err)
	}

	// Untracked files.
	if out, err := run(ctx, dir, "ls-files", "--others", "--exclude-standard"); err == nil {
		addLines(set, out)
	}

	out := make([]string, 0, len(set))
	for f := range set {
		out = append(out, f)
	}
	sort.Strings(out)
	return out, nil
}

// RepoRoot returns the absolute path to the repository root containing dir.
func RepoRoot(ctx context.Context, dir string) (string, error) {
	if !Available() {
		return "", rerr.New(rerr.GitUnavailable, false, "git not found on PATH")
	}
	out, err := run(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", rerr.New(rerr.GitUnavailable, true, "not a git repository: %v", err)
	}
	return strings.TrimSpace(out), nil
}

// validateRef rejects revision strings that could be interpreted as command
// options. A legitimate git revision (branch, tag, SHA, or range such as
// "HEAD~3" or "a..b") never begins with "-", so rejecting a leading dash blocks
// argument/option injection into the git invocation (for example
// "--output=<path>", which would let git write to an arbitrary file).
func validateRef(ref string) error {
	if strings.HasPrefix(ref, "-") {
		return rerr.New(rerr.InvalidArgument, true,
			"invalid base ref %q: must not begin with '-'", ref)
	}
	return nil
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}

func addLines(set map[string]bool, out string) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			set[line] = true
		}
	}
}
