# Example: Bugfix Workflow

This example shows a complete Teamwork **bugfix workflow** applied to a small Go
library. The library has a deliberate off-by-one bug in its `Sum` function.

## The Bug

The `Sum` function in `src/counter.go` skips the first element of the slice
because the loop starts at index `1` instead of `0`:

```go
// BUG: should start at 0
for i := 1; i < len(s); i++ {
    total += s[i]
}
```

For example, `Sum([]int{1, 2, 3})` returns `5` instead of `6`.

## What's in this example

```
examples/bugfix/
├── README.md                          — this file
├── src/
│   └── counter.go                     — Go package with the deliberate bug
└── .teamwork/
    ├── config.yaml                    — project configuration
    ├── state/
    │   └── bugfix-off-by-one.yaml     — completed bugfix workflow state
    └── handoffs/
        └── bugfix-off-by-one/         — handoff artifacts
            ├── step-2-coder.md
            ├── step-3-tester.md
            └── step-4-reviewer.md
```

## How the workflow ran

### Step 1 — Human starts the workflow

The developer spotted the bug in a CI failure and ran:

```bash
teamwork start bugfix "Fix off-by-one error in counter.Sum — the loop starts at index 1 and skips the first element" --issue 3
```

### Step 2 — Coder

`@coder` diagnosed the bug, fixed the loop start index from `1` to `0`, and
opened PR #4.

See [`step-2-coder.md`](.teamwork/handoffs/bugfix-off-by-one/step-2-coder.md).

### Step 3 — Tester

`@tester` added regression tests: `TestSum_Empty`, `TestSum_SingleElement`,
`TestSum_FirstElementIncluded`, and `TestSum_AllNegative`. 100% coverage on
the fixed function.

See [`step-3-tester.md`](.teamwork/handoffs/bugfix-off-by-one/step-3-tester.md).

### Step 4 — Reviewer

`@reviewer` approved PR #4. The developer merged it.

See [`step-4-reviewer.md`](.teamwork/handoffs/bugfix-off-by-one/step-4-reviewer.md).

## What was produced

| Artifact | Notes |
|----------|-------|
| Fixed `src/counter.go` | Loop now starts at `i := 0` |
| 4 regression tests | Prevent the bug from reappearing |
| Workflow state | `.teamwork/state/bugfix-off-by-one.yaml` |
| 3 handoff documents | `.teamwork/handoffs/bugfix-off-by-one/` |

## Trying it yourself

1. Open `src/counter.go` and confirm the bug (`i := 1`).
2. Run `teamwork start bugfix "Fix off-by-one in Sum"`.
3. Follow the orchestrator prompts — `@coder` will fix it and `@tester` will
   add regression coverage.
