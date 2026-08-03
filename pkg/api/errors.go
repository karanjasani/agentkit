package api

import "github.com/karanjasani/agentkit/internal/repomap/rerr"

// ErrorCode is a stable, machine-readable error identifier that appears in the
// JSON error envelope. Codes are part of the public contract.
type ErrorCode = rerr.Code

// Error codes returned by the API.
const (
	CodeSymbolNotFound   = rerr.SymbolNotFound
	CodePackageNotFound  = rerr.PackageNotFound
	CodeTypeNotFound     = rerr.TypeNotFound
	CodeNotAStruct       = rerr.NotAStruct
	CodeEndpointNotFound = rerr.EndpointNotFound
	CodeLoadFailed       = rerr.LoadFailed
	CodeGitUnavailable   = rerr.GitUnavailable
	CodeInvalidArgument  = rerr.InvalidArgument
	CodeInternal         = rerr.Internal
)

// Error is the typed error returned by the API. The CLI and other adapters map
// it into their own error format. It is deliberately not JSON itself: the
// library returns typed Go values, and callers apply the transport envelope.
type Error = rerr.Error

// AsError extracts an *Error from err if present, otherwise wraps it as an
// internal, non-recoverable error so every failure maps to a stable code.
func AsError(err error) *Error { return rerr.As(err) }

// newError is an internal convenience for building typed API errors.
func newError(code ErrorCode, recoverable bool, format string, args ...any) *Error {
	return rerr.New(code, recoverable, format, args...)
}
