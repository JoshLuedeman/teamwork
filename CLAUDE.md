# CLAUDE.md — Teamwork

Teamwork is an agent-native development template that structures AI-human collaboration through defined roles, workflows, and conventions. This file provides guidance for Claude Code when working in this repository.

## First Steps (Always Do These)

1. **Identify your role.** Determine which role you are performing and read the matching file in `agents/roles/`:
   - `planner.md` — Break goals into tasks. Never write code.
   - `architect.md` — Design systems, write ADRs. Never write code.
   - `coder.md` — Implement tasks, write tests, open PRs.
   - `tester.md` — Write adversarial tests. Never modify production code.
   - `reviewer.md` — Review PRs for quality. Never modify code.
   - `security-auditor.md` — Audit for vulnerabilities. Never modify code.
   - `documenter.md` — Keep documentation accurate and current.

2. **Read the conventions.** `docs/conventions.md` defines coding standards, branch naming (`feature/`, `bugfix/`, `refactor/`), conventional commit format, PR requirements, file naming, and directory structure.

3. **Check architecture decisions.** `docs/architecture.md` contains ADRs documenting prior design decisions and their rationale. Read relevant ADRs before proposing structural changes.

4. **Use correct terminology.** `docs/glossary.md` defines project terms (role, workflow, handoff, escalation, etc.). Use these terms consistently.

## Workflows

For multi-step tasks, check `agents/workflows/` for structured guides:
- `feature.md` — Adding new functionality
- `bugfix.md` — Diagnosing and fixing bugs
- `refactor.md` — Restructuring existing code
- `hotfix.md` — Urgent production fixes
- `security-response.md` — Responding to security vulnerabilities
- `dependency-update.md` — Updating third-party dependencies
- `documentation.md` — Standalone documentation updates
- `spike.md` — Research or technical investigation
- `release.md` — Preparing and publishing releases

Follow the workflow steps in order. Each workflow defines which roles participate and when handoffs occur.

## Key Rules

1. **Minimal changes.** Only modify what is necessary to complete the task. Do not refactor adjacent code, fix unrelated issues, or make speculative improvements.
2. **Test everything.** Run existing tests before and after changes. Write new tests for new behavior. Verify all tests pass before submitting.
3. **Conventional commits.** Format: `type(scope): description` (e.g., `feat(auth): add token refresh`, `fix(api): handle null response`).
4. **One task per PR.** Each pull request should address exactly one task. Keep changes focused and reviewable.
5. **Respect role boundaries.** Each role file specifies what actions are allowed and prohibited. Follow these constraints strictly.
6. **Document as you go.** Update relevant documentation when your changes affect behavior, APIs, or architecture.
7. **Keep scope small.** Target ~300 lines changed and ~10 files maximum per task.

## When to Escalate

Stop and ask the human when:

- Requirements are ambiguous, contradictory, or missing
- A change would affect system architecture or public APIs
- Tests fail and the root cause is unclear
- You need to choose between competing approaches with significant tradeoffs
- Security concerns arise that require human judgment
- The task crosses role boundaries (e.g., a coder being asked to make architectural decisions)
- You are unsure which role or workflow applies to the current task

## Project Structure

```
agents/
  roles/              — Role definitions (7 files — read yours first)
  workflows/          — Step-by-step workflow guides
docs/
  conventions.md      — Coding standards and project conventions
  architecture.md     — Architecture Decision Records (ADRs)
  glossary.md         — Terminology definitions
```

## Tips

- When starting work, state which role you are performing and confirm you have read the role file.
- If no role is specified, default to `coder.md` for implementation tasks or `planner.md` for planning tasks.
- Prefer reading existing code and tests before writing new code.
- When in doubt, check the glossary — terms like "handoff," "escalation," and "quality bar" have specific meanings in this project.
