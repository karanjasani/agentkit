// Package version exposes the tool version, resolved either from linker flags
// injected at release time (GoReleaser) or, failing that, from the module build
// info recorded by `go install`.
package version

import "runtime/debug"

// These variables are overridden at build time via -ldflags -X. When the binary
// is produced by `go install` (which does not set them), they stay at their
// zero/default values and we fall back to debug.ReadBuildInfo.
var (
	version = ""
	commit  = ""
	date    = ""
)

// devVersion is reported when no version information is available at all (for
// example, `go run` from a working tree).
const devVersion = "0.0.0-dev"

// Version returns the semantic version string of the tool. It prefers the
// ldflags-injected value and falls back to the module version recorded in the
// build info for `go install`-ed binaries.
func Version() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return devVersion
}

// Commit returns the VCS commit the binary was built from, if known.
func Commit() string {
	if commit != "" {
		return commit
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				return s.Value
			}
		}
	}
	return ""
}

// Date returns the commit/build date the binary was built from, if known.
func Date() string {
	if date != "" {
		return date
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.time" {
				return s.Value
			}
		}
	}
	return ""
}
