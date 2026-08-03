# Roadmap

AgentKit is built CLI-first. The command-line tool is the primary product; the Go
libraries under `pkg/` are the reusable core, and any future MCP server is a thin
adapter over those same libraries.

> Note on scope: The "zero external dependencies" success metric refers to zero
> external *services* (AgentKit is fully offline). The project does depend on a small
> set of vetted Go modules (cobra, `golang.org/x/tools`); these ship compiled into the
> binary and require no runtime services.

## v0.1 — RepoMap CLI (current)

Go module analysis via a deterministic, read-only CLI.

- `overview`, `package`, `symbol`, `callers`, `deps`, `impact`, `tests`, `endpoint`,
  `upstreams`, `struct`
- Stable `repomap.v1` JSON schemas
- Cross-platform releases and `go install`

**Deferred from v0.1:** LogSlice and the MCP servers (see below).

## v0.2 — LogSlice + performance

- LogSlice tool: `summarize`, `errors`, `correlate`, `timeline`, `redact`
- Caching and repository indexing
- Additional performance work

## v0.3 — MCP servers

- `repomap-mcp` (and later `logslice-mcp`) as thin adapters over `pkg/api`
- Reference integrations for Claude, Cursor, and VS Code

The library exposes an `Analyzer` handle (`api.New(...)`) precisely so a long-lived
MCP server can load a repository once and answer many queries cheaply.

## v0.5 — Multi-language

- Additional language backends: TypeScript, Python, Rust, Java

## v1.0 — Stability

- Frozen JSON schemas and public Go APIs
- Long-term support

## Ordering note

The MCP servers are built *before* LogSlice in implementation terms: they validate the
`pkg/api` library boundary while RepoMap is still the only consumer. The version
numbers above reflect release grouping, not a strict build order.
