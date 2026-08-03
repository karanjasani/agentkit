<!-- Thanks for contributing! Please fill out the sections below. -->

## Summary

<!-- What does this PR do and why? Focus on the "why". -->

## Related issues

<!-- e.g. Fixes #123 -->

## Type of change

- [ ] Bug fix
- [ ] New feature
- [ ] Documentation
- [ ] Refactor / internal
- [ ] CI / build

## Checklist

- [ ] `go test ./...` passes
- [ ] `go vet ./...` and `golangci-lint run` pass
- [ ] Golden files updated (`make golden`) if CLI output changed, and the diff reviewed
- [ ] Docs and `CHANGELOG.md` (Unreleased) updated if user-facing
- [ ] Output remains deterministic (slices sorted, no map-order leakage, no absolute paths)
- [ ] Changes are backward compatible with the `repomap.v1` schema (additive only)
