# Copilot Instructions — Teamwork

Teamwork is an agent-native development template that structures AI-human collaboration through defined roles, workflows, and conventions.

## Before You Start

0. **Read project context.** Start every session by reading `MEMORY.md` for current project state, recent decisions, and active context.

1. **Identify your role.** Determine which role you are performing and read the corresponding file in `agents/roles/`:
   - `planner.md` — Break goals into tasks. Never write code.
   - `architect.md` — Design systems, write ADRs. Never write code.
   - `coder.md` — Implement tasks, write tests, open PRs.
   - `tester.md` — Write adversarial tests. Never modify production code.
   - `reviewer.md` — Review PRs for quality. Never modify code.
   - `security-auditor.md` — Audit for vulnerabilities. Never modify code.
   - `documenter.md` — Keep documentation accurate and current.
   - `orchestrator.md` — Coordinate workflows, dispatch roles. Never write code.

   **If no role is specified:** Use `docs/role-selector.md` to determine the right role from the task. Quick defaults: implementation tasks → Coder, planning/scoping → Planner, code review → Reviewer, multi-role tasks → Planner (break it down first).
2. **Read the conventions.** Review `docs/conventions.md` for coding standards, branch naming, commit format, and PR requirements.
3. **Understand the architecture.** Check `docs/architecture.md` for prior design decisions (ADRs) before proposing structural changes.
4. **Learn the vocabulary.** Use terminology as defined in `docs/glossary.md`.
5. **Check for workflows.** For multi-step tasks, see `agents/workflows/` for step-by-step guides (feature, bugfix, refactor, hotfix, security-response, dependency-update, documentation, spike, release).

## Key Rules

- **Minimal changes.** Change only what is necessary. Do not refactor unrelated code.
- **Test before submitting.** Run all relevant tests and verify they pass before opening a PR.
- **Conventional commits.** Use the format: `type(scope): description` (e.g., `feat(auth): add token refresh`).
- **One task per PR.** Keep pull requests focused on a single task or change.
- **Follow role boundaries.** If your role says "never write code" or "never modify code," respect that constraint.
- **Check for workflow files.** If a multi-step workflow applies (feature, bugfix, refactor, hotfix, security-response, dependency-update, documentation, spike, release), check `agents/workflows/` for step-by-step guidance.

## When to Escalate

Stop and ask the human when:

- Requirements are ambiguous or contradictory
- A change would affect architecture or public APIs
- Tests fail and the fix is unclear
- You are unsure which role or workflow applies
- Security concerns arise that need human judgment

## Project Structure

```
MEMORY.md             — Project context (read at session start)
agents/roles/         — Role definitions (read yours before starting)
agents/workflows/     — Multi-step workflow guides
.teamwork/            — Orchestration state (handoffs, memory, metrics)
docs/conventions.md   — Coding standards and project conventions
docs/architecture.md  — Architecture decisions (ADRs)
docs/protocols.md     — Coordination protocol specification
docs/glossary.md      — Project terminology
docs/conflict-resolution.md — Resolving conflicting instructions
docs/secrets-policy.md — Rules for handling secrets and credentials
docs/cost-policy.md   — Guidelines for managing AI agent costs
```

## Protocol Integration

When working in a workflow, integrate with the `.teamwork/` protocol system:

### At Session Start
1. Check `.teamwork/state/` for active workflows relevant to your task.
2. If a workflow exists for your task, read the state file to find your step and role.
3. Read the previous handoff artifact from `.teamwork/handoffs/<workflow-id>/` for context from the prior role.
4. Check `.teamwork/memory/` for patterns, decisions, and feedback relevant to the domain you're working in.

### During Work
- Follow your role file's rules and quality bar.
- Reference the handoff from the previous step — it contains the context, decisions, and artifacts you need.

### At Session End
1. Write a handoff artifact to `.teamwork/handoffs/<workflow-id>/<step>-<role>.md` following the format in `docs/protocols.md`.
2. Update the workflow state file in `.teamwork/state/<workflow-id>.yaml` — record your step as completed and set the next step/role.
3. If you learned something broadly applicable, add it to the relevant file in `.teamwork/memory/` (patterns, antipatterns, decisions, or feedback).

### If No Workflow Exists
If the task is ad-hoc (not part of a tracked workflow), you don't need to read/write `.teamwork/` files. Just follow your role file and conventions.
