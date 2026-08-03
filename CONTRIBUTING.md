# Contributing to AgentKit

Thanks for your interest in improving AgentKit. This document explains how to get set
up, the standards we hold code to, and how to submit changes.

## Code of Conduct

This project is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By
participating you agree to uphold it. Report unacceptable behavior via the process in
that document.

## Getting started

Requirements:

- Go 1.24 or newer
- `git`
- (optional) `golangci-lint`, `govulncheck` for local linting

```bash
git clone https://github.com/karanjasani/agentkit
cd agentkit
go build ./...
go test ./...
```

Useful `make` targets:

| Target | Description |
| --- | --- |
| `make build` | Build the `repomap` binary |
| `make test` | Run all tests |
| `make golden` | Regenerate golden test fixtures (`-update`) |
| `make cover` | Run tests with coverage |
| `make lint` | Run `golangci-lint` |
| `make vet` | Run `go vet` |
| `make vuln` | Run `govulncheck` |
| `make fmt` | Format with `gofmt -s` |

## Design principles

Please keep changes consistent with the project's core principles:

1. **Deterministic** — identical input must produce byte-identical output. Sort every
   slice before it reaches output; never let map iteration order leak.
2. **Read-only** — commands must never modify source, git, or logs.
3. **Evidence-driven** — every result carries a `file:line` anchor.
4. **Thin adapters** — business logic lives in `internal/` and `pkg/`, never in
   `cmd/`. The CLI only parses flags, calls the library, and marshals.
5. **Stable schemas** — once released, JSON schemas are backward compatible. Additive
   changes only within a schema version.

## Coding standards

- No global mutable state.
- Context-aware APIs: `context.Context` as the first argument where appropriate.
- Public APIs documented with Go doc comments.
- Prefer composition; avoid reflection unless necessary.
- Aim for 90%+ unit-test coverage on core packages, with golden tests for CLI JSON.
- Run `gofmt -s`, `go vet`, and `golangci-lint run` before pushing.

## Making changes

1. Fork and create a topic branch (`git checkout -b feature/my-change`).
2. Make your change with tests. If you change CLI output, update golden files with
   `make golden` and review the diff carefully.
3. Update docs and `CHANGELOG.md` under the `Unreleased` heading.
4. Ensure `go test ./...`, `go vet ./...`, and `golangci-lint run` pass.
5. Open a pull request using the PR template. Describe the *why*, not just the *what*.

## Commit and PR conventions

- Keep commits focused and messages descriptive.
- Reference related issues (e.g. `Fixes #123`).
- Small, reviewable PRs are strongly preferred.

## Reporting bugs and requesting features

Use the issue templates. For security vulnerabilities, do **not** open a public issue;
follow [SECURITY.md](SECURITY.md) instead.
