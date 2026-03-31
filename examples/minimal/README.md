# Example: Minimal Feature Workflow

This example shows a complete Teamwork **feature workflow** applied to a tiny
Go CLI tool called `greeter`. The workflow adds a `--version` flag to the binary
so users can check which version is installed.

## What's in this example

```
examples/minimal/
├── README.md                          — this file
├── src/
│   └── main.go                        — the greeter CLI
└── .teamwork/
    ├── config.yaml                    — project configuration
    ├── state/
    │   └── feature-add-version-flag.yaml  — completed workflow state
    ├── handoffs/
    │   └── feature-add-version-flag/  — role-to-role handoff artifacts
    │       ├── step-2-planner.md
    │       ├── step-3-architect.md
    │       ├── step-4-coder.md
    │       ├── step-5-tester.md
    │       ├── step-6-security-auditor.md
    │       └── step-7-reviewer.md
    └── memory/
        └── patterns.yaml              — learnings captured after the workflow
```

## How the workflow ran

### Step 1 — Human starts the workflow

The developer ran:

```bash
teamwork start feature "Add --version flag to the greeter CLI so users can check which version is installed" --issue 12
```

Teamwork created `.teamwork/state/feature-add-version-flag.yaml` and set
`current_step: 1`, `current_role: planner`.

### Step 2 — Planner

The orchestrator invoked `@planner`, which:
- Broke the request into two tasks with acceptance criteria
- Confirmed there were no blocking dependencies
- Wrote the handoff to `step-2-planner.md`
- Ran `teamwork complete feature/add-version-flag` to advance to step 3

See [`step-2-planner.md`](.teamwork/handoffs/feature-add-version-flag/step-2-planner.md).

### Step 3 — Architect

The orchestrator invoked `@architect`, which:
- Confirmed no new dependencies or packages were needed
- Decided on the ldflags injection approach
- Wrote design guidance for the coder
- Ran `teamwork complete feature/add-version-flag` to advance to step 4

See [`step-3-architect.md`](.teamwork/handoffs/feature-add-version-flag/step-3-architect.md).

### Step 4 — Coder

The orchestrator invoked `@coder`, which:
- Added `var version = "dev"` to `src/main.go`
- Added the `--version` argument handler
- Updated `Makefile` to inject the version via ldflags
- Opened PR #7
- Ran `teamwork complete feature/add-version-flag` to advance to step 5

See [`step-4-coder.md`](.teamwork/handoffs/feature-add-version-flag/step-4-coder.md).

### Step 5 — Tester

The orchestrator invoked `@tester`, which:
- Wrote `TestVersionFlag_PrintsVersion` and `TestVersionFlag_DefaultWhenNotSet`
- Confirmed 94% statement coverage
- Verified all acceptance criteria
- Ran `teamwork complete feature/add-version-flag` to advance to step 6

See [`step-5-tester.md`](.teamwork/handoffs/feature-add-version-flag/step-5-tester.md).

### Step 6 — Security Auditor

The orchestrator invoked `@security-auditor`, which:
- Ran a secrets scan (clean)
- Confirmed no user input reaches the version variable
- Cleared the PR for reviewer
- Ran `teamwork complete feature/add-version-flag` to advance to step 7

See [`step-6-security-auditor.md`](.teamwork/handoffs/feature-add-version-flag/step-6-security-auditor.md).

### Step 7 — Reviewer

The orchestrator invoked `@reviewer`, which:
- Reviewed PR #7 and approved it
- The developer merged the PR

See [`step-7-reviewer.md`](.teamwork/handoffs/feature-add-version-flag/step-7-reviewer.md).

## What was produced

| Artifact | Location |
|----------|---------|
| Updated `src/main.go` | `src/main.go` |
| Updated `Makefile` | (not included in example) |
| Workflow state | `.teamwork/state/feature-add-version-flag.yaml` |
| 6 handoff documents | `.teamwork/handoffs/feature-add-version-flag/` |
| Memory patterns | `.teamwork/memory/patterns.yaml` |

## Trying it yourself

1. Copy this example into a new directory.
2. Run `teamwork init` to install the framework files.
3. Start a new workflow:

```bash
teamwork start feature "Add a --help subcommand"
teamwork next
```

The orchestrator will guide each agent through the steps.
