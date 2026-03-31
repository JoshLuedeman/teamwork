# Handoff: coder → tester

**Workflow:** bugfix/off-by-one  
**Step:** 2  
**Date:** 2025-02-03T14:22:00Z

## Summary

Fixed the off-by-one error in `counter.Sum`. The loop now starts at index `0`
instead of index `1`. The root cause was a copy-paste error from a 1-indexed
loop in an adjacent function.

## Root Cause

```go
// Before (buggy):
for i := 1; i < len(s); i++ {

// After (fixed):
for i := 0; i < len(s); i++ {
```

## Artifacts

- `src/counter.go` — fixed loop start index

## PR

Opened PR #4: "fix(counter): correct off-by-one in Sum loop"

## Quality Gate

- [x] Tests pass
- [x] Lint passes
- [x] PR opened with CI green
