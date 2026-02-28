# Teamwork — Agent-Native Development Framework

Teamwork is a framework where AI coding agents are primary contributors. Roles define behavior, not tools. Each agent reads its role file to understand its identity, responsibilities, constraints, and quality bar — then acts accordingly within defined workflows.

## How It Works

### Roles Define Behavior

Each role file is a complete behavioral contract. An agent assigned a role reads that file and follows it. Roles are deliberately separated so that no single agent has conflicting incentives — the coder doesn't review their own code, the reviewer doesn't modify it, and the planner never implements.

### Agents Read Their Role File

When an agent starts work, it reads exactly one role file. That file tells it:

- **Who it is** and what it's responsible for
- **What inputs** it needs before starting
- **What outputs** it must produce
- **What rules** constrain its behavior
- **What quality bar** it must meet
- **When to escalate** to a human

### Workflows Coordinate Roles

Roles don't operate in isolation. Workflows define how roles hand off work to each other — what triggers each role, what artifacts flow between them, and what the end-to-end process looks like. See the [workflows/](workflows/) directory for defined workflows.

| Workflow | File | When to Use |
|----------|------|-------------|
| Feature | [workflows/feature.md](workflows/feature.md) | New functionality from a goal or requirement |
| Bugfix | [workflows/bugfix.md](workflows/bugfix.md) | Fixing a reported defect |
| Refactor | [workflows/refactor.md](workflows/refactor.md) | Improving code quality without changing behavior |
| Hotfix | [workflows/hotfix.md](workflows/hotfix.md) | Urgent production fix requiring immediate resolution |
| Security Response | [workflows/security-response.md](workflows/security-response.md) | Responding to a discovered security vulnerability |
| Dependency Update | [workflows/dependency-update.md](workflows/dependency-update.md) | Updating third-party dependencies |
| Documentation | [workflows/documentation.md](workflows/documentation.md) | Standalone documentation creation or updates |
| Spike | [workflows/spike.md](workflows/spike.md) | Research or technical investigation |
| Release | [workflows/release.md](workflows/release.md) | Preparing and publishing a release |

## Core Roles

These roles cover the essential software development lifecycle:

| Role | File | Description |
|------|------|-------------|
| **Planner** | [roles/planner.md](roles/planner.md) | Breaks goals into actionable tasks with acceptance criteria |
| **Architect** | [roles/architect.md](roles/architect.md) | Makes design decisions, evaluates tradeoffs, produces ADRs |
| **Coder** | [roles/coder.md](roles/coder.md) | Implements tasks by writing code and tests, opens PRs |
| **Tester** | [roles/tester.md](roles/tester.md) | Writes and runs tests with an adversarial mindset |
| **Reviewer** | [roles/reviewer.md](roles/reviewer.md) | Reviews PRs for quality, correctness, and standards |
| **Security Auditor** | [roles/security-auditor.md](roles/security-auditor.md) | Checks for vulnerabilities, secret leaks, and unsafe patterns |
| **Documenter** | [roles/documenter.md](roles/documenter.md) | Keeps docs in sync with code changes |

## Optional Roles

Add these when your project needs them:

| Role | File | Description |
|------|------|-------------|
| **Triager** | [roles/optional/triager.md](roles/optional/triager.md) | Categorizes issues, assigns priority, routes to workflows |
| **DevOps** | [roles/optional/devops.md](roles/optional/devops.md) | Manages CI/CD, deployments, and infrastructure-as-code |
| **Dependency Manager** | [roles/optional/dependency-manager.md](roles/optional/dependency-manager.md) | Monitors and updates dependencies safely |
| **Refactorer** | [roles/optional/refactorer.md](roles/optional/refactorer.md) | Improves code quality without changing behavior |

## Role File Structure

Every role file follows this exact structure:

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
