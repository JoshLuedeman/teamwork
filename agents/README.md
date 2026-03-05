# Teamwork — Agent-Native Development Framework

> **⚠️ Migration Notice:** Agent files have moved to `.github/agents/*.agent.md` (Custom Agents) and workflow files have moved to `.github/skills/*/SKILL.md` (Skills). The files below in `roles/` and `workflows/` are legacy copies. See `.github/agents/` and `.github/skills/` for the current versions.

Teamwork is a framework where AI coding agents are primary contributors. Agents are defined as Custom Agent files in `.github/agents/`. Each agent reads its agent file to understand its identity, responsibilities, constraints, and quality bar — then acts accordingly within defined Skills (workflows).

## How It Works

### Agents Define Behavior

Each agent file (`.github/agents/*.agent.md`) is a complete behavioral contract. An agent assigned a role reads that file and follows it. Agents are deliberately separated so that no single agent has conflicting incentives — the coder doesn't review their own code, the reviewer doesn't modify it, and the planner never implements.

### Agents Read Their Agent File

When an agent starts work, it reads exactly one agent file. That file tells it:

- **Who it is** and what it's responsible for
- **What inputs** it needs before starting
- **What outputs** it must produce
- **What rules** constrain its behavior
- **What quality bar** it must meet
- **When to escalate** to a human

### Skills Coordinate Agents

Agents don't operate in isolation. Skills (in `.github/skills/`) define how agents hand off work to each other — what triggers each agent, what artifacts flow between them, and what the end-to-end process looks like.

| Skill | File | When to Use |
|-------|------|-------------|
| Feature | [.github/skills/feature/SKILL.md](../.github/skills/feature/SKILL.md) | New functionality from a goal or requirement |
| Bugfix | [.github/skills/bugfix/SKILL.md](../.github/skills/bugfix/SKILL.md) | Fixing a reported defect |
| Refactor | [.github/skills/refactor/SKILL.md](../.github/skills/refactor/SKILL.md) | Improving code quality without changing behavior |
| Hotfix | [.github/skills/hotfix/SKILL.md](../.github/skills/hotfix/SKILL.md) | Urgent production fix requiring immediate resolution |
| Security Response | [.github/skills/security-response/SKILL.md](../.github/skills/security-response/SKILL.md) | Responding to a discovered security vulnerability |
| Dependency Update | [.github/skills/dependency-update/SKILL.md](../.github/skills/dependency-update/SKILL.md) | Updating third-party dependencies |
| Documentation | [.github/skills/documentation/SKILL.md](../.github/skills/documentation/SKILL.md) | Standalone documentation creation or updates |
| Spike | [.github/skills/spike/SKILL.md](../.github/skills/spike/SKILL.md) | Research or technical investigation |
| Release | [.github/skills/release/SKILL.md](../.github/skills/release/SKILL.md) | Preparing and publishing a release |
| Rollback | [.github/skills/rollback/SKILL.md](../.github/skills/rollback/SKILL.md) | Rolling back failed deployments or changes |

## Core Agents

These agents cover the essential software development lifecycle:

| Agent | File | Description |
|-------|------|-------------|
| **Planner** | [.github/agents/planner.agent.md](../.github/agents/planner.agent.md) | Breaks goals into actionable tasks with acceptance criteria |
| **Architect** | [.github/agents/architect.agent.md](../.github/agents/architect.agent.md) | Makes design decisions, evaluates tradeoffs, produces ADRs |
| **Coder** | [.github/agents/coder.agent.md](../.github/agents/coder.agent.md) | Implements tasks by writing code and tests, opens PRs |
| **Tester** | [.github/agents/tester.agent.md](../.github/agents/tester.agent.md) | Writes and runs tests with an adversarial mindset |
| **Reviewer** | [.github/agents/reviewer.agent.md](../.github/agents/reviewer.agent.md) | Reviews PRs for quality, correctness, and standards |
| **Security Auditor** | [.github/agents/security-auditor.agent.md](../.github/agents/security-auditor.agent.md) | Checks for vulnerabilities, secret leaks, and unsafe patterns |
| **Documenter** | [.github/agents/documenter.agent.md](../.github/agents/documenter.agent.md) | Keeps docs in sync with code changes |
| **Orchestrator** | [.github/agents/orchestrator.agent.md](../.github/agents/orchestrator.agent.md) | Coordinates workflows, dispatches roles, validates handoffs, enforces quality gates |

## Specialized Agents

| Agent | File | Description |
|-------|------|-------------|
| **@lint-agent** | [.github/agents/lint-agent.agent.md](../.github/agents/lint-agent.agent.md) | Runs linters and auto-fixes code style issues |
| **@api-agent** | [.github/agents/api-agent.agent.md](../.github/agents/api-agent.agent.md) | Designs, implements, and validates API endpoints |
| **@dba-agent** | [.github/agents/dba-agent.agent.md](../.github/agents/dba-agent.agent.md) | Manages database schemas, migrations, and query optimization |

## Agent File Structure

Every agent file follows this exact structure:

1. **Identity** — Who you are and your purpose
2. **Responsibilities** — What you do
3. **Inputs** — What you need to start work
4. **Outputs** — What you produce
5. **Rules** — Constraints and boundaries
6. **Quality Bar** — Minimum standard for your work
7. **Escalation** — When to ask the human for help

## Principles

- **Separation of concerns**: Each role has a single focus. No role both writes and reviews code.
- **Explicit handoffs**: Artifacts flow between roles through defined interfaces, not implicit knowledge.
- **Human in the loop**: Every role has an escalation policy. Agents ask for help rather than guessing.
- **Minimal authority**: Each role does only what it's responsible for and nothing more.
- **Traceability**: Every output links back to the input that triggered it.
