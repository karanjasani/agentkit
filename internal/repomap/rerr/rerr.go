// Package rerr defines the typed error used throughout repomap analysis. It is a
// dependency-free leaf package so both the internal analyzers and the output
// layer can use it without import cycles. The public pkg/api package re-exports
// these identifiers as the stable error contract.
package rerr

import "fmt"

// Code is a stable, machine-readable error identifier surfaced in JSON output.
type Code string

const (
	SymbolNotFound   Code = "SYMBOL_NOT_FOUND"
	PackageNotFound  Code = "PACKAGE_NOT_FOUND"
	TypeNotFound     Code = "TYPE_NOT_FOUND"
	NotAStruct       Code = "NOT_A_STRUCT"
	EndpointNotFound Code = "ENDPOINT_NOT_FOUND"
	LoadFailed       Code = "LOAD_FAILED"
	GitUnavailable   Code = "GIT_UNAVAILABLE"
	InvalidArgument  Code = "INVALID_ARGUMENT"
	Internal         Code = "INTERNAL"
)

// Error is the typed analysis error.
type Error struct {
	Code        Code
	Message     string
	Recoverable bool
}

func (e *Error) Error() string { return e.Message }

// New builds a typed error.
func New(code Code, recoverable bool, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...), Recoverable: recoverable}
}

// As extracts an *Error from err, wrapping unknown errors as internal.
func As(err error) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return e
	}
	return &Error{Code: Internal, Message: err.Error(), Recoverable: false}
}
