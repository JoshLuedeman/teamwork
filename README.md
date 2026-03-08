# Teamwork

**An agent-native development template for teams where AI agents are first-class contributors.**

> This is a GitHub template repository. Click **"Use this template"** to create your own project based on this framework.

## Philosophy

AI coding agents are powerful but directionless without structure. Teamwork provides that structure — roles, workflows, conventions, and quality gates — so agents can contribute reliably.

- **Role-based, not tool-based.** A "coder" is a behavioral contract. Any AI agent (or a human) can fill it. Swap models, switch tools — the roles stay the same.
- **Human as executive.** You set goals, approve results, and make judgment calls. Agents do the implementation work between these decisions.
- **Separation of concerns.** No agent both writes and reviews code. Roles have clear boundaries and explicit handoffs.
- **Phase 2 complete.** The orchestration layer is built — a Go CLI (`teamwork`) automates workflow coordination, task dispatching, validation, and handoff management. See [Phase 2](#phase-2-orchestration-app) below.

## Quick Start

### Option A: Using the gh CLI extension (recommended)

If you have the [GitHub CLI](https://cli.github.com) installed:

```bash
gh extension install JoshLuedeman/gh-teamwork
gh teamwork init
```

### Option B: Using the teamwork binary directly

1. **Install the Teamwork CLI** — Build from source or use Docker:
   ```bash
   # Via go install
   go install github.com/JoshLuedeman/teamwork/cmd/teamwork@latest
   
   # Or via Docker
   docker build -t teamwork .
   alias teamwork='docker run --rm -v "$(pwd):/project" teamwork'
   ```

2. **Create your repo** — Click "Use this template" on GitHub, or clone and remove the `.git` directory.

3. **Initialize Teamwork in your project** — In your new repo, run:
   ```bash
   teamwork install
   ```
   This fetches framework files and creates the `.teamwork/` directory structure.

### After Installation

4. **Read the onboarding guide** — [`docs/onboarding.md`](docs/onboarding.md) covers first steps for both humans and agents.

5. **Customize for your project** — Edit the files listed in [Customization Guide](#customization-guide) below.

## Enhance with MCP

Teamwork agents become significantly more capable when paired with MCP (Model Context Protocol) servers. Configure them in `.teamwork/config.yaml` and agents automatically use them for GitHub operations, library lookups, security scanning, web research, and more.

See **[docs/mcp.md](docs/mcp.md)** for full setup instructions and client configuration.

### Teamwork MCP servers

Five custom MCP servers are included in [`mcp-servers/`](mcp-servers/README.md), purpose-built for Teamwork workflows:

| Server | Install | What it unlocks |
|--------|---------|-----------------| 
| [Coverage](mcp-servers/coverage/) | `pip install teamwork-mcp-coverage` | Coverage report analysis (lcov, Istanbul, Go) |
| [Commits](mcp-servers/commits/) | `pip install teamwork-mcp-commits` | Conventional commit generation and validation |
| [ADR](mcp-servers/adr/) | `pip install teamwork-mcp-adr` | Architecture Decision Record management |
| [Changelog](mcp-servers/changelog/) | `pip install teamwork-mcp-changelog` | Changelog and release notes via git-cliff |
| [Complexity](mcp-servers/complexity/) | `pip install teamwork-mcp-complexity` | Cyclomatic complexity analysis (30+ languages) |

### Recommended third-party servers

| Server | Install | What it unlocks |
|--------|---------|-----------------| 
| [GitHub MCP](https://github.com/github/github-mcp-server) | `gh extension install github/gh-mcp` | PRs, issues, CI, Dependabot alerts |
| [Context7](https://github.com/upstash/context7) | `npx -y @upstash/context7-mcp` | Accurate, up-to-date library docs |
| [Semgrep](https://github.com/semgrep/mcp) | `pip install semgrep-mcp` | SAST security scanning |
| [Tavily](https://github.com/tavily-ai/tavily-mcp) | `npx -y tavily-mcp` | Web search and research |
| [E2B](https://github.com/e2b-dev/e2b-mcp) | `pip install e2b-mcp` | Sandboxed code execution |
| [Terraform](https://github.com/hashicorp/terraform-mcp-server) | `npx -y terraform-mcp-server@latest` | Terraform Registry and IaC |

After installing, run `teamwork mcp list` to see which servers are configured and ready.

## Repository Structure

```
teamwork/
├── .github/                       # GitHub & Copilot Configuration
│   ├── agents/                    # Custom Agents (behavioral contracts)
│   │   ├── planner.agent.md
│   │   ├── architect.agent.md
│   │   ├── coder.agent.md
│   │   ├── tester.agent.md
│   │   ├── reviewer.agent.md
│   │   ├── security-auditor.agent.md
│   │   ├── documenter.agent.md
│   │   ├── orchestrator.agent.md
│   │   ├── lint-agent.agent.md
│   │   ├── api-agent.agent.md
│   │   └── dba-agent.agent.md
│   ├── skills/                    # Skills (end-to-end workflow definitions)
│   │   ├── feature/SKILL.md
│   │   ├── bugfix/SKILL.md
│   │   ├── refactor/SKILL.md
│   │   ├── hotfix/SKILL.md
│   │   ├── security-response/SKILL.md
│   │   ├── dependency-update/SKILL.md
│   │   ├── documentation/SKILL.md
│   │   ├── spike/SKILL.md
│   │   ├── release/SKILL.md
│   │   └── rollback/SKILL.md
│   ├── instructions/              # Path-specific instructions
│   ├── copilot-instructions.md    # GitHub Copilot custom instructions
│   ├── ISSUE_TEMPLATE/            # Issue templates (bug, task, planning)
│   └── PULL_REQUEST_TEMPLATE.md   # PR template
├── docs/                          # Documentation
│   ├── onboarding.md              # Getting started for humans and agents
│   ├── conventions.md             # Code, git, and testing standards
│   ├── glossary.md                # Framework terminology
│   ├── architecture.md            # ADRs and design decisions
│   ├── cli.md                     # Teamwork CLI command reference
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
├── MEMORY.md                      # Project context (read at session start)
├── CHANGELOG.md                   # Project changelog
├── Makefile                       # Central command interface
├── .editorconfig                  # Editor formatting standards
├── .pre-commit-config.yaml        # Pre-commit hook configuration
└── .gitignore
```

## Agents

Eleven agents cover the software development lifecycle. Each agent file (Custom Agent) is a complete behavioral contract defining identity, responsibilities, inputs, outputs, rules, quality bar, and escalation policy.

| Agent | File | Description |
|-------|------|-------------|
| Planner | [`planner.agent.md`](.github/agents/planner.agent.md) | Breaks goals into actionable tasks with acceptance criteria |
| Architect | [`architect.agent.md`](.github/agents/architect.agent.md) | Makes design decisions, evaluates tradeoffs, produces ADRs |
| Coder | [`coder.agent.md`](.github/agents/coder.agent.md) | Implements tasks by writing code and tests, opens PRs |
| Tester | [`tester.agent.md`](.github/agents/tester.agent.md) | Writes and runs tests with an adversarial mindset |
| Reviewer | [`reviewer.agent.md`](.github/agents/reviewer.agent.md) | Reviews PRs for quality, correctness, and standards |
| Security Auditor | [`security-auditor.agent.md`](.github/agents/security-auditor.agent.md) | Checks for vulnerabilities, secret leaks, and unsafe patterns |
| Documenter | [`documenter.agent.md`](.github/agents/documenter.agent.md) | Keeps docs in sync with code changes |
| Orchestrator | [`orchestrator.agent.md`](.github/agents/orchestrator.agent.md) | Coordinates workflows, dispatches roles, validates handoffs |
| Lint Agent | [`lint-agent.agent.md`](.github/agents/lint-agent.agent.md) | Runs linters and auto-fixes code style issues |
| API Agent | [`api-agent.agent.md`](.github/agents/api-agent.agent.md) | Designs, implements, and validates API endpoints |
| DBA Agent | [`dba-agent.agent.md`](.github/agents/dba-agent.agent.md) | Manages database schemas, migrations, and query optimization |

## Skills (Workflows)

Ten Skills define how agents coordinate to deliver work end-to-end.

| Skill | File | When to Use |
|-------|------|-------------|
| Feature | [`feature/SKILL.md`](.github/skills/feature/SKILL.md) | New functionality from a goal or requirement |
| Bugfix | [`bugfix/SKILL.md`](.github/skills/bugfix/SKILL.md) | Fixing a reported defect |
| Refactor | [`refactor/SKILL.md`](.github/skills/refactor/SKILL.md) | Improving code quality without changing behavior |
| Hotfix | [`hotfix/SKILL.md`](.github/skills/hotfix/SKILL.md) | Urgent production fix requiring immediate resolution |
| Security Response | [`security-response/SKILL.md`](.github/skills/security-response/SKILL.md) | Responding to a discovered security vulnerability |
| Dependency Update | [`dependency-update/SKILL.md`](.github/skills/dependency-update/SKILL.md) | Updating third-party dependencies |
| Documentation | [`documentation/SKILL.md`](.github/skills/documentation/SKILL.md) | Standalone documentation creation or updates |
| Spike | [`spike/SKILL.md`](.github/skills/spike/SKILL.md) | Research or technical investigation |
| Release | [`release/SKILL.md`](.github/skills/release/SKILL.md) | Preparing and publishing a release |
| Rollback | [`rollback/SKILL.md`](.github/skills/rollback/SKILL.md) | Rolling back failed deployments or changes |

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

Everything runs locally in your development environment. No GitHub Actions are consumed. You invoke agents through your AI coding tool of choice (Copilot, Claude Code, Cursor, etc.) and point them at the relevant agent file.

## Customization Guide

### Add a new agent

1. Create a file in `.github/agents/` named `<agent-name>.agent.md` following the standard structure: Identity, Responsibilities, Inputs, Outputs, Rules, Quality Bar, Escalation.
2. Add the agent to any Skills that should include it.
3. Update `docs/role-selector.md` with the new agent's tier and selection criteria.

### Add a new Skill (workflow)

1. Create a directory in `.github/skills/<skill-name>/` with a `SKILL.md` file following the existing pattern: Overview, Trigger, Steps (table), Handoff Contracts, Completion Criteria, Notes.
2. Reference the agents involved and define explicit handoffs between each step.

### Adapt for your language/stack

- Edit `scripts/*.sh` to call your actual linters, test runners, and build tools.
- Update `docs/conventions.md` with your project's coding standards.
- Add dependencies and setup steps to `scripts/setup.sh`.
- Modify `.pre-commit-config.yaml` for your language's hooks.

### Add CI/CD

- Add GitHub Actions workflows in `.github/workflows/` to run `make check` on PRs.
- Activate the optional **DevOps** agent (`.github/agents/devops.agent.md`) for deployment coordination.
- The `Makefile` targets (`lint`, `test`, `build`, `check`) work identically in CI and locally.

## Phase 2: Orchestration App

Phase 2 is complete. The Teamwork CLI (`teamwork`) automates workflow coordination, task dispatching, state management, and handoff validation. The CLI reads and writes protocol files in `.teamwork/` to manage workflow state, providing human visibility and control over the entire lifecycle.

**Features:**
- **Workflow management** — `teamwork start`, `status`, `next`, `approve`, `block`, `cancel`, `fail`, `complete`, `history`
- **Validation** — `teamwork validate` with JSON output for CI integration
- **Environment diagnostics** — `teamwork doctor` checks prerequisites and reports issues with actionable fixes
- **Installation** — `teamwork install` and `teamwork update` for framework setup and upgrades
- **Memory management** — `teamwork memory add`, `search`, `list`, `sync` for structured project memory
- **Metrics reporting** — `teamwork metrics summary` and `roles` for workflow analytics
- **Multi-repo coordination** — `teamwork repos` for hub-spoke multi-repository setups
- **GitHub CLI integration** — `gh teamwork init` and `gh teamwork update` via the `gh-teamwork` extension
- **Interactive dashboard** — `teamwork dashboard` for real-time workflow monitoring

See [`docs/cli.md`](docs/cli.md) for command reference and [`docs/decisions/004-validate-command-design.md`](docs/decisions/004-validate-command-design.md) and [`docs/decisions/005-install-update-design.md`](docs/decisions/005-install-update-design.md) for design details.

## Phase 3: Auto-Install GitHub App

Phase 3 adds automatic framework installation. A GitHub App + Cloudflare Worker detects new repository creation and pushes Teamwork framework files automatically — no manual `teamwork install` needed.

See [`docs/github-app-setup.md`](docs/github-app-setup.md) for setup instructions.

## License

[MIT](LICENSE)
