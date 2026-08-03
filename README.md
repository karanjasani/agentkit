# AgentKit — RepoMap

[![CI](https://github.com/karanjasani/agentkit/actions/workflows/ci.yml/badge.svg)](https://github.com/karanjasani/agentkit/actions/workflows/ci.yml)
<!-- Hidden until the repo is public and Scorecard has published a score worth showing.
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/karanjasani/agentkit/badge)](https://securityscorecards.dev/viewer/?uri=github.com/karanjasani/agentkit)
-->
[![Go Reference](https://pkg.go.dev/badge/github.com/karanjasani/agentkit.svg)](https://pkg.go.dev/github.com/karanjasani/agentkit)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> Deterministic repository intelligence for AI coding agents (and the humans who work with them).

`repomap` is a read-only CLI that answers the questions coding agents waste the most
context on — *where is this symbol defined, who calls it, what does this struct
serialize to, which tests should I run* — with deterministic, evidence-backed JSON.

---

## The problem

AI coding agents spend a large fraction of their context budget rediscovering facts
that already exist in the code: which files matter, where a symbol lives, what depends
on it, which tests cover a change. Every model re-derives this from `grep` and
whole-file reads, burning tokens on discovery instead of the actual task.

## Why RepoMap

Instead of letting every agent rediscover the repository from scratch, `repomap`
provides deterministic answers:

- **Deterministic** — same repo + same command = byte-identical output.
- **Structured** — every command emits stable, versioned JSON (`schema: repomap.v1`).
- **Evidence-backed** — every result carries a `file:line` anchor, so the agent's
  fallback read stays narrow.
- **Read-only & offline** — no mutation of source, git, or logs; no network calls.
  Safe to auto-approve in an agent's permission settings.
- **Fast** — target < 100 ms on small repos, < 2 s on large ones.

RepoMap is not another LLM, indexer, or IDE. It produces structured information; the
agent interprets it.

## Installation

```bash
# From source (requires Go 1.24+)
go install github.com/karanjasani/agentkit/cmd/repomap@latest

# Homebrew (tap)
brew install karanjasani/tap/repomap

# Or download a prebuilt binary from the Releases page.
```

## Quick start

```bash
# High-level map of the module in the current directory
repomap overview

# Find a symbol, with callers and covering tests
repomap symbol ValidateToken

# See the JSON contract of a struct
repomap struct models.FabricStatus

# Blast radius of your branch vs. main, plus tests to run
repomap impact --base main
```

Output is JSON by default. Add `--format text` for human-readable output.

## Examples

```bash
repomap symbol ValidateToken --format json
```

```json
{
  "schema": "repomap.v1",
  "tool_version": "0.1.0",
  "ok": true,
  "result": {
    "name": "ValidateToken",
    "kind": "func",
    "package": "github.com/acme/svc/internal/auth",
    "location": { "file": "internal/auth/token.go", "line": 42 },
    "signature": "func ValidateToken(ctx context.Context, raw string) (*Claims, error)"
  }
}
```

See [docs/](docs/) for a full per-command reference and [examples/](examples/) for
copy-pasteable agent integration snippets (`AGENTS.md`, Cursor rules).

## Commands

| Command | Purpose |
| --- | --- |
| `overview` | Packages, entrypoints, modules, generated/vendor folders |
| `package <path>` | Imports, imported-by, exported symbols, tests |
| `symbol <name>` | Location, package, signature/body/doc/shape |
| `callers <name>` | Direct + indirect callers with one-line context |
| `deps <path>` | Package/import graph and dependency depth |
| `impact --base <ref>` | Changed + affected packages, tests, risk score |
| `tests <name>` | Unit / integration / benchmark tests for a symbol |
| `endpoint <method> <path>` | Route → handler → upstream vertical slice |
| `upstreams <path>` | Outbound REST/adapter calls and decode targets |
| `struct <name>` | Recursive JSON contract of a type |

## Architecture

All logic lives in reusable Go packages. The CLI is a thin adapter that parses flags,
calls `pkg/api`, and marshals the result. This keeps a future MCP server a pure
adapter over the same libraries.

```
Core libraries (pkg/api, pkg/models, internal/repomap/*)
        │
   ┌────┴────┐
  CLI     (future) MCP server
        │
   Human / AI agents
```

See [docs/architecture.md](docs/architecture.md).

## JSON design

Every successful response is wrapped in a stable envelope:

```json
{ "schema": "repomap.v1", "tool_version": "0.1.0", "ok": true, "result": {} }
```

Errors are structured too:

```json
{
  "schema": "repomap.v1",
  "tool_version": "0.1.0",
  "ok": false,
  "error": { "code": "SYMBOL_NOT_FOUND", "message": "ValidateToken not found", "recoverable": true }
}
```

JSON Schemas for every command live under [schemas/repomap.v1/](schemas/).

## Roadmap

v0.1 (this release) is the `repomap` CLI for Go modules. LogSlice (log intelligence)
and the MCP servers are planned for later releases. See [ROADMAP.md](ROADMAP.md).

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) and our
[Code of Conduct](CODE_OF_CONDUCT.md). Security issues: see [SECURITY.md](SECURITY.md).

## License

MIT License. See [LICENSE](LICENSE).
