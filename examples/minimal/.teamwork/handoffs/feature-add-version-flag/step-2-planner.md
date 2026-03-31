# Handoff: planner → architect

**Workflow:** feature/add-version-flag  
**Step:** 2  
**Date:** 2025-01-15T09:18:00Z

## Summary

Broke down the feature request into two concrete tasks: (1) add the `--version`
flag to the `greeter` CLI reading a build-time variable via ldflags, and (2)
update the Makefile `build` target to inject the version from `git describe`.

## Artifacts

- `.teamwork/handoffs/feature-add-version-flag/step-2-planner.md` (this file)

## Tasks

1. **Add `--version` flag** — In `src/main.go`, declare a package-level `version`
   variable and handle `os.Args[1] == "--version"`.
   - Acceptance criteria: `./greeter --version` prints `greeter version <tag>`.
2. **Update Makefile** — Pass `-X main.version=$(git describe --tags --always)`
   to `go build`.
   - Acceptance criteria: `make build && ./greeter --version` shows the current
     git tag.

## Quality Gate

- [x] Tasks have acceptance criteria
- [x] No blocking dependencies
