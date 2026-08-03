// Package models defines the stable, serializable types that make up the
// `repomap.v1` JSON contract. Every CLI command and every future adapter (such
// as an MCP server) emits one of these result types wrapped in an Envelope.
//
// These types are the public data contract of the tool. Once released they are
// changed additively only, to preserve backward compatibility.
package models

// Schema is the identifier embedded in every response so consumers can detect
// the contract version.
const Schema = "repomap.v1"

// Envelope is the outer wrapper for every response. Exactly one of Result or
// Error is populated, indicated by Ok.
type Envelope struct {
	Schema      string `json:"schema"`
	ToolVersion string `json:"tool_version"`
	Ok          bool   `json:"ok"`
	// Result holds the command payload on success. It is one of the result
	// types declared in this package.
	Result any `json:"result,omitempty"`
	// Error holds structured error information on failure.
	Error *Error `json:"error,omitempty"`
}

// Error is the structured error payload returned when Ok is false.
type Error struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Recoverable bool   `json:"recoverable"`
}

// Location is an evidence anchor: a file (relative to the module root, using
// forward slashes) and a 1-based line, optionally a column.
type Location struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Col  int    `json:"col,omitempty"`
}
