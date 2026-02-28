# Copilot Instructions — Teamwork

Teamwork is an agent-native development template that structures AI-human collaboration through defined roles, workflows, and conventions.

## Before You Start

1. **Identify your role.** Determine which role you are performing and read the corresponding file in `agents/roles/`:
   - `planner.md` — Break goals into tasks. Never write code.
   - `architect.md` — Design systems, write ADRs. Never write code.
   - `coder.md` — Implement tasks, write tests, open PRs.
   - `tester.md` — Write adversarial tests. Never modify production code.
   - `reviewer.md` — Review PRs for quality. Never modify code.
   - `security-auditor.md` — Audit for vulnerabilities. Never modify code.
   - `documenter.md` — Keep documentation accurate and current.
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
agents/roles/       — Role definitions (read yours before starting)
agents/workflows/   — Multi-step workflow guides
docs/conventions.md — Coding standards and project conventions
docs/architecture.md — Architecture decisions (ADRs)
docs/glossary.md    — Project terminology
```
