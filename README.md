# Teamwork

**An agent-native development template for teams where AI agents are first-class contributors.**

> This is a GitHub template repository. Click **"Use this template"** to create your own project based on this framework.

## Philosophy

AI coding agents are powerful but directionless without structure. Teamwork provides that structure — roles, workflows, conventions, and quality gates — so agents can contribute reliably.

- **Role-based, not tool-based.** A "coder" is a behavioral contract. Any AI agent (or a human) can fill it. Swap models, switch tools — the roles stay the same.
- **Human as executive.** You set goals, approve results, and make judgment calls. Agents do the implementation work between those decisions.
- **Separation of concerns.** No agent both writes and reviews code. Roles have clear boundaries and explicit handoffs.
- **Phase 1 of two.** This template is the foundation — static role files and workflows that agents read. Phase 2 will add an orchestration layer to coordinate agents automatically. See [Phase 2](#phase-2-orchestration-app) below.

## Quick Start

1. **Create your repo** — Click "Use this template" on GitHub, or clone and remove the `.git` directory.
2. **Run setup** — `make setup` runs the one-time development environment setup.
3. **Read the onboarding guide** — [`docs/onboarding.md`](docs/onboarding.md) covers first steps for both humans and agents.
4. **Customize for your project** — Edit the files listed in [Customization Guide](#customization-guide) below.

## Repository Structure

```
teamwork/
├── agents/                        # Agent Framework
│   ├── README.md                  # How the role system works
│   ├── roles/                     # Behavioral contracts for each role
│   │   ├── planner.md
│   │   ├── architect.md
│   │   ├── coder.md
│   │   ├── tester.md
│   │   ├── reviewer.md
│   │   ├── security-auditor.md
│   │   ├── documenter.md
│   │   └── optional/              # Add-on roles for larger projects
│   │       ├── triager.md
│   │       ├── devops.md
│   │       ├── dependency-manager.md
│   │       └── refactorer.md
│   └── workflows/                 # End-to-end process definitions
│       ├── feature.md
│       ├── bugfix.md
│       ├── refactor.md
│       ├── hotfix.md
│       ├── security-response.md
│       ├── dependency-update.md
│       ├── documentation.md
│       ├── spike.md
│       ├── release.md
│       └── rollback.md
├── docs/                          # Documentation
│   ├── onboarding.md              # Getting started for humans and agents
│   ├── conventions.md             # Code, git, and testing standards
│   ├── glossary.md                # Framework terminology
│   ├── architecture.md            # ADRs and design decisions
│   ├── workflow-selector.md       # Guide for choosing the right workflow
│   ├── conflict-resolution.md     # Resolving conflicting instructions
│   ├── secrets-policy.md          # Rules for handling secrets and credentials
│   ├── cost-policy.md             # Guidelines for managing AI agent costs
│   └── decisions/                 # Architecture Decision Records (ADRs)
│       └── 001-role-based-agent-framework.md
├── scripts/                       # Tooling (called by Makefile)
│   ├── setup.sh                   # Dev environment setup
│   ├── lint.sh                    # Run linters
│   ├── test.sh                    # Run tests
│   ├── build.sh                   # Build the project
│   ├── plan.sh                    # Invoke planning agent
│   └── review.sh                  # Invoke review agent
├── .github/                       # GitHub Templates
│   ├── ISSUE_TEMPLATE/            # Issue templates (bug, task, planning)
│   ├── PULL_REQUEST_TEMPLATE.md   # PR template
│   └── copilot-instructions.md   # GitHub Copilot custom instructions
├── MEMORY.md                      # Project context (read at session start)
├── CHANGELOG.md                   # Project changelog
├── Makefile                       # Central command interface
├── CLAUDE.md                      # Claude Code custom instructions
├── .cursorrules                   # Cursor custom instructions
├── .editorconfig                  # Editor formatting standards
├── .pre-commit-config.yaml        # Pre-commit hook configuration
└── .gitignore
```

## Agent Roles

Seven core roles cover the software development lifecycle. Each role file is a complete behavioral contract defining identity, responsibilities, inputs, outputs, rules, quality bar, and escalation policy.

| Role | File | Description |
|------|------|-------------|
| Planner | [`planner.md`](agents/roles/planner.md) | Breaks goals into actionable tasks with acceptance criteria |
| Architect | [`architect.md`](agents/roles/architect.md) | Makes design decisions, evaluates tradeoffs, produces ADRs |
| Coder | [`coder.md`](agents/roles/coder.md) | Implements tasks by writing code and tests, opens PRs |
| Tester | [`tester.md`](agents/roles/tester.md) | Writes and runs tests with an adversarial mindset |
| Reviewer | [`reviewer.md`](agents/roles/reviewer.md) | Reviews PRs for quality, correctness, and standards |
| Security Auditor | [`security-auditor.md`](agents/roles/security-auditor.md) | Checks for vulnerabilities, secret leaks, and unsafe patterns |
| Documenter | [`documenter.md`](agents/roles/documenter.md) | Keeps docs in sync with code changes |

Four optional roles are available in [`agents/roles/optional/`](agents/roles/optional/) for projects that need them: **Triager**, **DevOps**, **Dependency Manager**, and **Refactorer**. See [`agents/README.md`](agents/README.md) for full details.

## Workflows

Ten workflows define how roles coordinate to deliver work end-to-end.

| Workflow | File | When to Use |
|----------|------|-------------|
| Feature | [`feature.md`](agents/workflows/feature.md) | New functionality from a goal or requirement |
| Bugfix | [`bugfix.md`](agents/workflows/bugfix.md) | Fixing a reported defect |
| Refactor | [`refactor.md`](agents/workflows/refactor.md) | Improving code quality without changing behavior |
| Hotfix | [`hotfix.md`](agents/workflows/hotfix.md) | Urgent production fix requiring immediate resolution |
| Security Response | [`security-response.md`](agents/workflows/security-response.md) | Responding to a discovered security vulnerability |
| Dependency Update | [`dependency-update.md`](agents/workflows/dependency-update.md) | Updating third-party dependencies |
| Documentation | [`documentation.md`](agents/workflows/documentation.md) | Standalone documentation creation or updates |
| Spike | [`spike.md`](agents/workflows/spike.md) | Research or technical investigation |
| Release | [`release.md`](agents/workflows/release.md) | Preparing and publishing a release |
| Rollback | [`rollback.md`](agents/workflows/rollback.md) | Rolling back failed deployments or changes |

The **feature workflow** is the most common and follows this progression:

```
Human          Planner        Architect       Coder          Tester
  │               │               │              │              │
  │─── goal ─────>│               │              │              │
  │               │─── tasks ────>│              │              │
  │               │               │─── design ──>│              │
  │               │               │              │─── PR ──────>│
  │               │               │              │              │
  │           Security Auditor   Reviewer       Human        Documenter
  │               │               │              │              │
  │               │<── PR ────────│              │              │
  │               │─── findings ─>│              │              │
  │               │               │─── approved ─>│             │
  │               │               │              │─── merged ──>│
  │               │               │              │              │─── docs
```

## How Work Gets Done

1. **Human sets a goal** — describe what you want built, fixed, or improved.
2. **Planner breaks it down** — decomposes the goal into tasks with acceptance criteria and a dependency graph.
3. **Agents execute** — each role picks up work in sequence (or in parallel for independent tasks), following the appropriate workflow.
4. **Human approves** — review the PR, merge, and the Documenter updates docs.

Everything runs locally in your development environment. No GitHub Actions are consumed. You invoke agents through your AI coding tool of choice (Copilot, Claude Code, Cursor, etc.) and point them at the relevant role file.

## Customization Guide

### Add a new role

1. Create a file in `agents/roles/` (or `agents/roles/optional/`) following the [standard structure](agents/README.md#role-file-structure): Identity, Responsibilities, Inputs, Outputs, Rules, Quality Bar, Escalation.
2. Add the role to any workflows that should include it.
3. Document the role in `agents/README.md`.

### Add a new workflow

1. Create a file in `agents/workflows/` following the existing pattern: Overview, Trigger, Steps (table), Handoff Contracts, Completion Criteria, Notes.
2. Reference the roles involved and define explicit handoffs between each step.

### Adapt for your language/stack

- Edit `scripts/*.sh` to call your actual linters, test runners, and build tools.
- Update `docs/conventions.md` with your project's coding standards.
- Add dependencies and setup steps to `scripts/setup.sh`.
- Modify `.pre-commit-config.yaml` for your language's hooks.

### Add CI/CD

- Add GitHub Actions workflows in `.github/workflows/` to run `make check` on PRs.
- Activate the optional **DevOps** role (`agents/roles/optional/devops.md`) for deployment coordination.
- The `Makefile` targets (`lint`, `test`, `build`, `check`) work identically in CI and locally.

## Phase 2: Orchestration App

This template is Phase 1 — the structural foundation. Phase 2 will introduce an orchestration CLI that coordinates agents automatically: reading the dependency graph, dispatching tasks to the right roles, tracking progress, and managing handoffs without manual intervention.

Coming soon.

## License

[MIT](LICENSE)
