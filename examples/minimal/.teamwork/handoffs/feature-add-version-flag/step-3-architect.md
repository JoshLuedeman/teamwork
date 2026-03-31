# Handoff: architect → coder

**Workflow:** feature/add-version-flag  
**Step:** 3  
**Date:** 2025-01-15T09:35:00Z

## Summary

The `--version` flag implementation is straightforward — no new dependencies or
packages required. The version string will be injected at link time using the
standard Go `-ldflags` mechanism. No ADR needed for this scope.

## Design Decision

Use a package-level `var version = "dev"` as the default so the binary works
even when built without ldflags (e.g., `go run`). The Makefile `build` target
will set it via:

```
go build -ldflags "-X main.version=$(git describe --tags --always)" -o greeter ./src
```

## Scope

- Modify `src/main.go` only (add `version` var and `--version` branch).
- Modify `Makefile` (update `build` target).
- No new files needed.

## Quality Gate

- [x] ADR written (N/A — no architecture change)
- [x] Scope bounded to two files
- [x] Design guidance provided
