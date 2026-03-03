# Role Selector

A guide to choosing the right role for your task. When no role is specified, use this to determine which role to adopt. See `agents/roles/` for full role definitions.

## Decision Tree

Start at the top and follow the first question that matches your situation.

```
Is this coordinating a multi-step workflow?
├── Yes → Orchestrator
└── No ↓

Does this involve breaking a goal into tasks or scoping work?
├── Yes → Planner
└── No ↓

Does this involve system design, architecture, or evaluating tradeoffs?
├── Yes → Architect
└── No ↓

Does this involve writing or modifying code?
├── Yes → Coder
└── No ↓

Does this involve writing tests or validating behavior?
├── Yes → Tester
└── No ↓

Does this involve reviewing a PR or someone else's code?
├── Yes → Reviewer
└── No ↓

Does this involve security analysis or vulnerability assessment?
├── Yes → Security Auditor
└── No ↓

Does this involve writing or updating documentation?
├── Yes → Documenter
└── No ↓

Is the task too large or ambiguous to fit one role?
├── Yes → Planner (break it down first)
└── No → Coder (default for implementation)
```

## Intent-to-Role Table

| What the user is asking | Role | Why |
|---|---|---|
| "Break down this goal" / "Plan this feature" / "Create tasks for..." | **Planner** | Decomposing goals into actionable tasks |
| "How should we architect..." / "Design the system for..." / "Evaluate tradeoffs" | **Architect** | System design and decision-making |
| "Build this" / "Implement..." / "Fix this bug" / "Add a feature" / "Write code for..." | **Coder** | Writing and modifying code |
| "Write tests for..." / "Verify this works" / "Test the edge cases" | **Tester** | Testing and validation |
| "Review this PR" / "Check this code" / "Is this implementation correct?" | **Reviewer** | Code review and quality checks |
| "Check for vulnerabilities" / "Audit security" / "Is this safe?" | **Security Auditor** | Security analysis |
| "Update the docs" / "Write a README" / "Document this API" | **Documenter** | Documentation |
| "Start the feature workflow" / "What's the status?" / "What should happen next?" | **Orchestrator** | Workflow coordination |

## Context-Based Routing

When intent alone isn't enough, use context clues:

| Context | Route to | Reasoning |
|---|---|---|
| There's a PR open for review | **Reviewer** | PRs need review before merge |
| A CVE or vulnerability was reported | **Security Auditor** | Security issues need triage first |
| There's a GitHub issue with acceptance criteria | **Coder** | Issue is ready for implementation |
| The request spans multiple repos or components | **Planner** | Complex work needs decomposition |
| There's a design question with multiple valid approaches | **Architect** | Architecture decisions need ADRs |
| Tests are failing after a code change | **Tester** | Test failures need investigation |
| A feature was just merged | **Documenter** | New features need documentation |
| Multiple workflows are active | **Orchestrator** | Coordination needed |

## Compound Tasks

Some requests span multiple roles. When that happens:

1. **Route to Planner first** — have it break the task into role-specific subtasks.
2. **If the task is small** (single file, clear scope) — route to the primary role and let it handle adjacent concerns.
3. **If the task is large** (multiple files, unclear scope) — always plan first.

Examples:
- "Add OAuth and document it" → **Planner** (spans coder + documenter)
- "Fix the login bug" → **Coder** (single role, clear scope)
- "Redesign the API and migrate existing clients" → **Planner** (large scope, multiple roles)
- "Review PR #42 for security issues" → **Security Auditor** (security takes priority over general review)

## Model Tier Guide

Each role has a recommended model tier based on cognitive complexity. Use the cheapest model that reliably handles the role's demands (Principle of Least Power).

| Role | Tier | Rationale |
|---|---|---|
| **Planner** | Premium | Complex goal decomposition, dependency analysis, scope judgment |
| **Architect** | Premium | Deep tradeoff analysis, multi-system reasoning, consequential decisions |
| **Coder** | Premium | Code generation quality, edge case handling, tool use |
| **Tester** | Standard | Structured task (criteria → tests), bounded scope |
| **Reviewer** | Standard | Bounded diff review, pattern matching against criteria |
| **Security Auditor** | Premium | Specialized domain knowledge, high cost of missed vulnerabilities |
| **Documenter** | Fast | Structured prose, summarization, low reasoning depth |
| **Orchestrator** | Fast | Rule-following, state management, routing logic |

**Tier definitions** (configure actual models in `.teamwork/config.yaml`):

- **Premium** — Strongest reasoning. Use for roles where errors are expensive to fix or where deep analysis is required. Examples: Claude Opus, GPT-5.2 Codex, Gemini 2.5 Pro.
- **Standard** — Balanced cost and capability. Use for roles with well-defined scope and structured inputs. Examples: Claude Sonnet, GPT-4.1, Gemini 2.5 Flash.
- **Fast** — Optimized for speed and cost. Use for roles that follow clear rules and produce structured outputs. Examples: Claude Haiku, GPT-5 mini, Gemini 2.0 Flash.

> **Cost impact:** Using fast-tier models for documenter and orchestrator (which together handle ~20% of workflow steps) can reduce overall model costs by 30-40% with no quality loss.

## Within a Workflow

If you're already in an active workflow (check `.teamwork/state/`), role selection is predetermined — follow the step sequence defined in the workflow file. The orchestrator or the state file tells you which role is next.

Only use this selector for:
- **Ad-hoc tasks** not part of a tracked workflow
- **First contact** when no workflow has been started yet
- **Determining whether to start a workflow** (if the task maps to a known workflow type, consider starting one via `teamwork start`)
