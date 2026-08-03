# Quick start

`repomap` answers structural questions about a Go module as stable, deterministic
JSON. It is read-only and runs entirely offline.

By default every command analyzes the current directory. Use `--dir` to point at
another module root.

## 1. Get the lay of the land

```bash
repomap overview
```

This lists packages, entrypoints (`package main`), generated files, and vendored
directories.

## 2. Find a symbol instead of grepping

```bash
# Just the signature (cheap, no body)
repomap symbol FetchWidget --signature-only

# The full declaration source
repomap symbol FetchWidget --body

# Only the doc comment
repomap symbol FetchWidget --doc

# The JSON contract of a struct type
repomap symbol Widget --shape
```

## 3. Understand a package

```bash
repomap package ./service
repomap deps ./...          # intra-module dependency graph + depth
```

## 4. Trace call relationships

```bash
repomap callers Helper       # who calls this (direct + indirect)
repomap tests FetchWidget    # which tests exercise this symbol
```

## 5. Scope a change before running the whole suite

```bash
# What does my working tree touch, and which tests should I run?
repomap impact --base main
```

`impact` uses `git` to diff against the base ref, maps changed files to packages,
walks the reverse-import graph, flags public API changes, and recommends the
minimal set of tests plus a risk score.

## 6. Map HTTP surface area

```bash
repomap endpoint GET /api/v1/widgets/{id}   # route -> handler -> upstreams
repomap upstreams ./service                  # outbound calls under a path
```

## Output format

JSON is the default and is the contract. Add `--format text` for a compact
human-readable rendering:

```bash
repomap overview --format text
```

Every JSON response is wrapped in an envelope:

```json
{
  "schema": "repomap.v1",
  "tool_version": "0.1.0",
  "ok": true,
  "result": { }
}
```

See [commands.md](commands.md) for the full command reference and
[schema.md](schema.md) for the contract.
