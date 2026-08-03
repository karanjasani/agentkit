package version

import "testing"

func TestVersionFallback(t *testing.T) {
	// With no ldflags injected (test binary), Version falls back to build info
	// or the dev sentinel. It must never be empty.
	if got := Version(); got == "" {
		t.Fatal("Version() returned empty string")
	}
}

func TestVersionPrefersLdflags(t *testing.T) {
	old := version
	defer func() { version = old }()
	version = "1.2.3"
	if got := Version(); got != "1.2.3" {
		t.Fatalf("Version() = %q, want 1.2.3", got)
	}
}

func TestCommitAndDate(t *testing.T) {
	oldC, oldD := commit, date
	defer func() { commit, date = oldC, oldD }()
	commit, date = "abc123", "2026-01-01"
	if Commit() != "abc123" {
		t.Errorf("Commit() = %q", Commit())
	}
	if Date() != "2026-01-01" {
		t.Errorf("Date() = %q", Date())
	}
}

// TestCommitDateFallback exercises the debug.ReadBuildInfo fallback path taken
// when no ldflags were injected (as in a plain `go test` binary). It asserts the
// calls are safe and return a string; the value depends on the build's VCS
// settings and is not otherwise constrained.
func TestCommitDateFallback(t *testing.T) {
	oldC, oldD := commit, date
	defer func() { commit, date = oldC, oldD }()
	commit, date = "", ""
	_ = Commit()
	_ = Date()
}
