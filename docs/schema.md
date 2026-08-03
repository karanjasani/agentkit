# JSON schema reference

Every response conforms to the `repomap.v1` contract. The machine-readable JSON
Schema files live in [`schemas/repomap.v1/`](../schemas/repomap.v1) and are
validated against the committed golden outputs in CI, which is what makes the
stability promise enforceable.

## Envelope

Schema: `envelope.schema.json`.

```json
{
  "schema": "repomap.v1",
  "tool_version": "0.1.0",
  "ok": true,
  "result": { }
}
```

| Field | Type | Notes |
| --- | --- | --- |
| `schema` | string | Always `repomap.v1`. |
| `tool_version` | string | The emitting binary's version. |
| `ok` | boolean | `true` on success, `false` on error. |
| `result` | object | Present when `ok` is `true`; one of the result types below. |
| `error` | object | Present when `ok` is `false`; see [Errors](#errors). |

## Shared definitions

Schema: `defs.schema.json`. Reusable shapes referenced by the command schemas.

- **location** — `{ "file": string, "line": integer, "col": integer? }`. `file`
  is module-root-relative and forward-slash normalized.
- **symbol** — name, kind (`func` / `type` / `var` / `const` / `method`),
  package, location, plus optional `signature`, `doc`, `body`, `recv`, `shape`,
  `callers`, `tests`.
- **struct** / **field** — recursive JSON contract of a type.
- **caller** — a call site with `context` and `confidence` (`direct` /
  `possible`).
- **test** — a test function with `kind` (`unit` / `integration` / `benchmark`).
- **upstream** — an outbound call with optional `service`/`method`/`url`/
  `decode_type`, a location, and `confidence`.
- **pkgref** — `{ "import_path", "name", "dir" }`.

## Result schemas

| Command | Schema file |
| --- | --- |
| `overview` | `overview.schema.json` |
| `package` | `package.schema.json` |
| `symbol` | `symbol.schema.json` |
| `struct` | `struct.schema.json` |
| `deps` | `deps.schema.json` |
| `callers` | `callers.schema.json` |
| `tests` | `tests.schema.json` |
| `impact` | `impact.schema.json` |
| `endpoint` | `endpoint.schema.json` |
| `upstreams` | `upstreams.schema.json` |

## Errors

When `ok` is `false`, `error` is:

```json
{
  "code": "SYMBOL_NOT_FOUND",
  "message": "no symbol named Foo",
  "recoverable": true
}
```

| Code | Exit | Meaning |
| --- | --- | --- |
| `SYMBOL_NOT_FOUND` | 2 | No matching symbol. |
| `PACKAGE_NOT_FOUND` | 2 | No matching package. |
| `TYPE_NOT_FOUND` | 2 | No matching named type. |
| `NOT_A_STRUCT` | 1 | The named type is not a struct. |
| `ENDPOINT_NOT_FOUND` | 2 | No matching route. |
| `LOAD_FAILED` | 1 | The module could not be loaded/type-checked. |
| `GIT_UNAVAILABLE` | 1 | `git` is required (for `impact`) but unavailable. |
| `INVALID_ARGUMENT` | 3 | Bad flag or argument. |
| `INTERNAL` | 1 | Unexpected internal error. |

## Compatibility

`repomap.v1` evolves **additively** only. New optional fields may be added within
the version; existing fields are never removed or repurposed. A breaking change
would ship as a new schema identifier (`repomap.v2`). Consumers should ignore
unknown fields.
