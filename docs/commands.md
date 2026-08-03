# Command reference

All commands share two persistent flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--dir` | `.` | Module root directory to analyze. |
| `--format` | `json` | Output format: `json` or `text`. |

`--version` prints the tool version and exits.

Diagnostics are written to **stderr**; **stdout** carries only the JSON contract,
so piping into `jq` is always safe.

Each command's `result` payload is documented by a JSON Schema under
[`schemas/repomap.v1/`](../schemas/repomap.v1) and summarized in
[schema.md](schema.md).

---

## `overview`

```
repomap overview
```

High-level map of the module: packages, entrypoints, generated files (detected by
filename heuristics such as `.pb.go`, `_generated.go`), vendored directories, and
aggregate stats. Result schema: `overview.schema.json`.

## `package <import-path|dir>`

```
repomap package ./service
repomap package example.com/sample/service
```

A single package's `import_path`, `name`, `dir`, direct `imports`, module-local
`imported_by`, exported `exports`, and `test_files`. Result schema:
`package.schema.json`.

## `symbol <name>`

```
repomap symbol FetchWidget --signature-only
repomap symbol FetchWidget --body
repomap symbol FetchWidget --doc
repomap symbol Widget --shape
```

Locate a symbol (func, type, var, const, method) and return details about it.

| Flag | Description |
| --- | --- |
| `--signature-only` | Return only the signature. |
| `--body` | Return the full declaration source. |
| `--doc` | Return only the doc comment. |
| `--shape` | Return the recursive JSON contract (struct types only). |

With no flag, the location and kind are returned. Result schema:
`symbol.schema.json`.

## `struct <name>`

```
repomap struct models.Widget
```

The recursive JSON contract of a named struct type: field names, JSON tags,
types, optionality, and nested structs (with cycle detection and a depth cap).
Result schema: `struct.schema.json`.

## `deps <import-path|dir>`

```
repomap deps example.com/sample
```

The intra-module dependency graph: `nodes`, directed `edges`, and the maximum
`depth`. External dependencies are excluded. Result schema: `deps.schema.json`.

## `callers <name>`

```
repomap callers Helper
```

Direct call sites (with one-line source context) and indirect caller chains,
computed via SSA + CHA call-graph analysis. Because CHA over-approximates through
interfaces, every edge carries a `confidence` of `direct` or `possible`. Result
schema: `callers.schema.json`.

## `tests <name>`

```
repomap tests FetchWidget
```

Test functions that transitively reach the symbol, classified as `unit`,
`integration`, or `benchmark`. Result schema: `tests.schema.json`.

## `impact`

```
repomap impact --base main
```

| Flag | Default | Description |
| --- | --- | --- |
| `--base` | `HEAD` | Git base ref to diff against. |

Diffs the working tree against the base ref, then reports changed files, changed
packages, transitively affected packages, public API deltas, recommended tests,
and a deterministic risk score/level. Requires a `git` binary; returns a
`GIT_UNAVAILABLE` error otherwise. Result schema: `impact.schema.json`.

## `endpoint <method> <path>`

```
repomap endpoint GET /api/v1/widgets/{id}
```

Traces a route through its handler to its orchestration chain and upstream calls.
Route detection is pluggable across `net/http` (Go 1.22+ method patterns), `chi`,
`gin`, `echo`, and `gorilla/mux`. Every result carries a `confidence` field and a
`file:line` anchor. Result schema: `endpoint.schema.json`.

## `upstreams <import-path|dir>`

```
repomap upstreams ./service
```

Outbound calls under a package path, with resolved URL constants and inferred
decode types where possible. Coverage:

- **All HTTP verbs** (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`)
  via `net/http` (functions and `*http.Client` methods) and request construction
  (`http.NewRequest`/`NewRequestWithContext`).
- **Well-known third-party HTTP clients** (resty, go-retryablehttp, fasthttp,
  req, gorequest, sling, gentleman, grequests, heimdall, monaco-io) reported with
  `direct` confidence.
- **Any other client or hand-rolled wrapper** whose call passes a literal/const
  URL — recognized via the URL and reported with `possible` confidence.
- **gRPC** connection setup (`grpc.Dial`/`DialContext`/`NewClient`), reported
  with `method: "GRPC"` and the dial target as the URL.

Each call carries `confidence` and a `file:line` location. Result schema:
`upstreams.schema.json`.

---

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | Generic error (e.g. load failure, git failure, internal). |
| `2` | Not found (`SYMBOL_NOT_FOUND`, `PACKAGE_NOT_FOUND`, `TYPE_NOT_FOUND`, `ENDPOINT_NOT_FOUND`). |
| `3` | Usage/argument error (`INVALID_ARGUMENT`). |

On failure the JSON envelope has `"ok": false` and an `error` object; see
[schema.md](schema.md#errors).
