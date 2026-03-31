# Handoff: coder → tester

**Workflow:** feature/add-version-flag  
**Step:** 4  
**Date:** 2025-01-15T10:05:00Z

## Summary

Implemented the `--version` flag in `src/main.go` and updated the `Makefile`
`build` target to inject the version from `git describe --tags --always`. The
binary built with `make build` now correctly reports the version. All existing
tests pass; `go vet` is clean.

## Artifacts

- `src/main.go` — added `var version = "dev"` and `--version` argument handler
- `Makefile` — updated `build` target with ldflags

## PR

Opened PR #7: "feat: add --version flag to greeter CLI"

## Quality Gate

- [x] Tests pass (`go test ./...`)
- [x] Lint passes (`go vet ./...`)
- [x] PR opened with CI green
