// Package output turns API results and errors into the stable repomap.v1 output
// contract. It applies the JSON envelope, maps typed API errors to structured
// error payloads and exit codes, and provides a human-readable text renderer.
//
// The CLI is the only place the envelope is applied; the library itself returns
// typed Go values so other adapters can reuse it without re-parsing JSON.
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/karanjasani/agentkit/internal/repomap/rerr"
	"github.com/karanjasani/agentkit/internal/version"
	"github.com/karanjasani/agentkit/pkg/models"
)

// Format selects the output rendering.
type Format string

const (
	// FormatJSON is the default machine-readable format.
	FormatJSON Format = "json"
	// FormatText is the human-readable format.
	FormatText Format = "text"
)

// ParseFormat validates and normalizes a format string.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case FormatJSON:
		return FormatJSON, nil
	case FormatText:
		return FormatText, nil
	default:
		return "", fmt.Errorf("invalid format %q: want json or text", s)
	}
}

// Exit codes returned by the CLI.
const (
	ExitOK         = 0
	ExitError      = 1
	ExitNotFound   = 2
	ExitUsageError = 3
)

// Success writes a successful result to w in the requested format.
func Success(w io.Writer, format Format, result any) error {
	if format == FormatText {
		return renderText(w, result)
	}
	env := models.Envelope{
		Schema:      models.Schema,
		ToolVersion: version.Version(),
		Ok:          true,
		Result:      result,
	}
	return writeJSON(w, env)
}

// Failure writes a structured error to w and returns the process exit code that
// should be used.
func Failure(w io.Writer, format Format, err error) int {
	e := rerr.As(err)
	if format == FormatText {
		fmt.Fprintf(w, "error [%s]: %s\n", e.Code, e.Message)
		return exitFor(e)
	}
	env := models.Envelope{
		Schema:      models.Schema,
		ToolVersion: version.Version(),
		Ok:          false,
		Error: &models.Error{
			Code:        string(e.Code),
			Message:     e.Message,
			Recoverable: e.Recoverable,
		},
	}
	_ = writeJSON(w, env)
	return exitFor(e)
}

func exitFor(e *rerr.Error) int {
	switch e.Code {
	case rerr.InvalidArgument:
		return ExitUsageError
	case rerr.SymbolNotFound, rerr.PackageNotFound, rerr.TypeNotFound,
		rerr.EndpointNotFound:
		return ExitNotFound
	default:
		return ExitError
	}
}

// writeJSON marshals v deterministically. encoding/json sorts map keys and
// preserves struct field order; slices must already be sorted by the analyzer.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
