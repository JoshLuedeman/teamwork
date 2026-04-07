# Teamwork — Claude Code Project Configuration

## Session Start Checklist
1. Read `MEMORY.md` for current project state, recent decisions, and active context
2. Check `.teamwork/state/` for any active workflows relevant to your task
3. Check `.teamwork/handoffs/` for prior step context if resuming a workflow

## Project Overview
Teamwork is an agent-native development template that structures AI-human collaboration
through defined roles, workflows, and conventions. This repo is the framework itself — 
written in Go.

- **Language:** Go (see `.github/instructions/go.instructions.md` for conventions)
- **Build:** `make build`
- **Test:** `make test` / `go test ./...`
- **Lint:** `golangci-lint run`
- **Release:** `make release VERSION=vX.Y.Z`

## Role System (Claude Code Equivalent of Copilot Custom Agents)

Copilot uses `.github/agents/*.agent.md` files selectable from a dropdown.
In Claude Code, invoke them as subagents using `/agents/<name>` or reference the
agent file directly. The agent files are fully compatible — Claude Code can read them.

**Core agents** (read `.github/agents/<name>.agent.md` for full persona + boundaries):
- `planner` — Break goals into tasks. Never write code.
- `architect` — Design systems, write ADRs. Never write code.
- `coder` — Implement tasks, write tests, open PRs.
- `tester` — Write adversarial tests. Never modify production code.
- `reviewer` — Review PRs for quality. Never modify code.
- `security-auditor` — Audit for vulnerabilities. Never modify code.
- `documenter` — Keep documentation accurate and current.
- `orchestrator` — Coordinate workflows, dispatch roles. Never write code.

**Extended agents:** triager, devops, dependency-manager, refactorer, lint-agent,
api-agent, dba-agent, product-owner, qa-lead (see `.github/agents/` for all files)

**Dispatch rule:** When a task clearly belongs to an agent's domain, adopt that
agent's persona and follow its boundaries. State which agent you're acting as.

## Workflow Skills (Claude Code Equivalent of Copilot Skills)

Copilot Skills live in `.github/skills/*/SKILL.md` and are invoked via `/skill-name`.
In Claude Code, read the relevant SKILL.md and follow its step table explicitly.

Available skills:
- `/feature-workflow` — New functionality end-to-end
- `/bugfix-workflow` — Diagnose and fix bugs
- `/refactor-workflow` — Restructure without behavior change
- `/hotfix-workflow` — Urgent production fixes
- `/security-response` — Respond to security vulnerabilities
- `/dependency-update` — Update third-party dependencies
- `/documentation-workflow` — Standalone documentation updates
- `/spike-workflow` — Research or technical investigation
- `/release-workflow` — Prepare and publish releases
- `/rollback-workflow` — Roll back failed deployments
- `/setup-teamwork` — Fill all CUSTOMIZE placeholders by analyzing the repo

To invoke: read `.github/skills/<name>/SKILL.md` and execute its steps in order.

## Path-Specific Instructions (Auto-loaded in Copilot; read explicitly in Claude Code)

| Path pattern | File to read |
|---|---|
| `**/*.go` | `.github/instructions/go.instructions.md` |
| `docs/**/*.md`, `README.md`, `CHANGELOG.md` | `.github/instructions/docs.instructions.md` |
| `.teamwork/**` | `.github/instructions/teamwork-config.instructions.md` |

**Always read the relevant instructions file before working in those paths.**

## Protocol Integration (.teamwork/ system)

**During work:**
- Follow the agent file's boundaries and quality bar
- Reference handoffs from prior steps in `.teamwork/handoffs/<workflow-id>/`

**At session end (for active workflows):**
- Write handoff artifact to `.teamwork/handoffs/<workflow-id>/<step>-<role>.md`
- Update state file in `.teamwork/state/<workflow-id>.yaml`
- Add broadly applicable learnings to `.teamwork/memory/`

## Key Rules
- **Minimal changes** — only touch what the task requires; no opportunistic refactors
- **Conventional commits** — `type(scope): description` (e.g., `feat(auth): add token refresh`)
- **One task per PR** — keep pull requests focused on a single change
- **Respect agent boundaries** — each `.agent.md` defines ✅ Always / ⚠️ Ask first / 🚫 Never
- **Scope ceiling** — ~300 lines changed, ~10 files max per task

## Known Gotchas (from MEMORY.md)
- `gh issue create --milestone <number>` silently ignores the milestone — use `gh api` instead
- Unauthenticated tarball fetch fails for private repos — always set `Authorization: Bearer $GH_TOKEN`
- httptest.NewServer is the right approach for mocking GitHub tarball endpoints in unit tests

## When to Escalate
Stop and ask when:
- Requirements are ambiguous or contradictory
- A change would affect architecture or public APIs
- Tests fail and the fix is unclear
- The task crosses agent boundaries
- Security concerns need human judgment
