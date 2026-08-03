# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-08-02

### Added

- Initial release of the `repomap` CLI for Go modules.
- Commands: `overview`, `package`, `symbol`, `callers`, `deps`, `impact`, `tests`,
  `endpoint`, `upstreams`, `struct`.
- Stable `repomap.v1` JSON envelope with structured errors.
- `symbol` flags: `--body`, `--signature-only`, `--doc`, `--shape`.
- `endpoint`/`upstreams` route detection for net/http, chi, gin, echo, gorilla/mux.
- Deterministic, read-only, offline analysis with `file:line` evidence anchors.
- JSON Schemas under `schemas/repomap.v1/`.
- Agent integration examples (`AGENTS.md`, Cursor rules).
- Cross-platform releases (Linux, macOS, Windows) via GoReleaser, with Homebrew tap,
  deb/rpm/apk packages, SBOM, and cosign signatures.

[Unreleased]: https://github.com/karanjasani/agentkit/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/karanjasani/agentkit/releases/tag/v0.1.0
