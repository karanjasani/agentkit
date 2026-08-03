<!--
Copy the block below into your repository's AGENTS.md (or append it to an
existing one). It teaches coding agents to reach for `repomap` instead of
grepping, and to scope changes before running tests.
-->

## Repository intelligence: use `repomap`

This repo ships with [`repomap`](https://github.com/karanjasani/agentkit), a
read-only, deterministic Go analysis CLI. Prefer it over ad-hoc `grep`/`find`
for structural questions. It outputs stable JSON on stdout (add `--format text`
for humans).

Prefer `repomap` when you need to:

- **Find a definition** instead of grepping:
  - `repomap symbol <Name> --signature-only` — signature + location.
  - `repomap symbol <Name> --body` — full source of the declaration.
  - `repomap symbol <Name> --doc` — doc comment only.
  - `repomap struct <pkg.Type>` — the recursive JSON contract of a type.
- **Understand structure**:
  - `repomap overview` — packages, entrypoints, generated/vendored code.
  - `repomap package <path>` — imports, importers, exports, tests.
  - `repomap deps <path>` — intra-module dependency graph.
- **Trace relationships** (each edge carries a `confidence`):
  - `repomap callers <Name>` — who calls a symbol (direct + indirect).
  - `repomap tests <Name>` — which tests exercise a symbol.
- **Scope a change before running the whole suite**:
  - `repomap impact --base <ref>` — changed files → affected packages →
    public-API delta → **recommended tests** → risk score. Run this first and
    prefer the recommended tests over running everything.
- **Map HTTP surface area**:
  - `repomap endpoint <METHOD> <path>` — route → handler → upstream calls.
  - `repomap upstreams <path>` — outbound calls under a path.

Guidelines:

- Trust `repomap` output over grep for "where is X defined / who calls X".
- Before running tests for a change, run `repomap impact --base <baseref>` and
  run the `recommended_tests` it returns.
- Every result includes `file:line` evidence anchors — cite them.
- The tool is read-only and offline; it never mutates the repo.
