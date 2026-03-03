# Project Memory

This file captures project learnings that persist across agent sessions. It serves as
institutional memory so agents don't repeat mistakes or rediscover established patterns.

**How to update this file:** When you learn something that future agents should know —
a pattern that works well, a mistake to avoid, a key decision — add it to the appropriate
section below. Keep entries concise (one or two lines). Include dates for decisions.
Do not remove entries unless they are explicitly obsolete.

---

## Patterns That Work

- **Role-model delegation:** Router (Sonnet) reads task → identifies role → spawns sub-agent with correct model tier via `task` tool `model` parameter. Works well.
- **httptest.NewServer** for mocking GitHub tarball endpoint in installer tests — lets unit tests cover install/update without network calls.

## Patterns to Avoid

- **gh issue create --milestone <number>** does NOT work; must use `gh api repos/.../issues` with `-F milestone=<number>`.
- **Unauthenticated tarball fetch** fails for private repos — always set `Authorization: Bearer $GH_TOKEN` header in HTTP requests to GitHub API.

## Key Decisions

Record important architectural and process decisions with rationale. Link to ADRs when
they exist.

- **2025-07-18:** ADR-004 — `teamwork validate` command. Exits 0/1/2, supports `--json`/`--quiet`, validates config+state+handoffs+memory in one pass. See `docs/decisions/004-validate-command-design.md`.
- **2025-07-18:** ADR-005 — `teamwork install` and `teamwork update` commands. Tarball fetch (needs GH_TOKEN for private repos), manifest-based conflict detection (SHA-256), framework vs starter file classification. See `docs/decisions/005-install-update-design.md`.
- **2026-03-03:** `gh-teamwork` CLI extension created at JoshLuedeman/gh-teamwork. Wraps `teamwork install`/`teamwork update` behind `gh teamwork init`/`gh teamwork update`. Falls back to Docker if binary not found.
- **2026-03-03:** GitHub milestone numbering: Milestone #1 = pre-existing Phase 2 Orchestration App. New: #2=Phase 1 install/update, #3=Phase 2 gh extension, #4=Phase 3 GitHub App.

## Common Mistakes

Things agents frequently get wrong. Check this section before starting work.

- Do not use `gh issue create --milestone` — it silently ignores the milestone. Use `gh api` instead.

## Reviewer Feedback

Persistent feedback from code reviews that applies broadly, not just to a single PR.

- *(No entries yet — add broadly applicable review feedback here)*
