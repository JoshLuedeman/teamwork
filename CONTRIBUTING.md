# Contributing to Teamwork

Thank you for your interest in contributing to Teamwork! This guide covers everything you need to get started.

## Development Setup

### Prerequisites

- **Go 1.22+** (for building from source)
- **Docker** (alternative: build and test without Go installed)
- **Git** (with `user.name` and `user.email` configured)
- **GitHub CLI** (`gh`) — optional, for issue/PR management

### Getting Started

```bash
# Clone the repository
git clone https://github.com/JoshLuedeman/teamwork.git
cd teamwork

# Run initial setup
make setup

# Verify everything works
make check
```

### Building

```bash
# Build the CLI binary (requires Go)
make build-cli       # outputs bin/teamwork

# Or build with Docker (no Go required)
make docker-build    # builds teamwork:latest image
```

### Running Tests

```bash
# Run Go tests (requires Go)
make test-cli

# Run tests via Docker (no Go required)
docker run --rm -v "$(pwd):/src" -w /src golang:1.24-alpine go test ./...
```

### Running Linters

```bash
make lint
```

### Full Check (lint + test + build)

```bash
make check
```

## Project Structure

```
cmd/teamwork/         — CLI entry point and cobra commands
internal/
  config/             — Config parsing (.teamwork/config.yaml)
  handoff/            — Handoff artifact management
  installer/          — Install/update framework files
  memory/             — Structured project memory
  metrics/            — JSONL metrics logging and aggregation
  state/              — Workflow state machine
  tui/                — Terminal UI dashboard
  validate/           — Validation checks
  workflow/           — Workflow engine (ties everything together)
agents/
  roles/              — Role definitions (8 core roles)
  workflows/          — Workflow step guides (10 types)
docs/                 — Documentation and ADRs
.teamwork/            — Orchestration state (config, handoffs, memory, metrics)
scripts/              — Build, test, and lint scripts
```

## Making Changes

### Branch Naming

Use prefixed branch names with lowercase kebab-case:

- `feature/<short-description>` — new functionality
- `bugfix/<short-description>` — fixing broken behavior
- `refactor/<short-description>` — restructuring without behavior change
- `docs/<short-description>` — documentation-only changes
- `chore/<short-description>` — tooling, dependencies, CI

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short summary>
```

**Types:** `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`

Examples:
```
feat(cli): add teamwork doctor command
fix(state): handle missing state directory
docs(contributing): add development setup guide
test(metrics): add defect tracking tests
```

### Pull Request Requirements

- **One logical change per PR** — don't mix features, fixes, and refactors
- **Tests required** — every behavioral change needs a corresponding test
- **Passing checks** — `make check` must pass before submitting
- **Clear description** — explain what changed, why, and how to verify
- **Link issues** — reference related GitHub issues (e.g., `Fixes #42`)

## Adding a New Role

Roles live in `agents/roles/`. To add a new role:

1. Create `agents/roles/<role-name>.md` following the existing template
2. Include all required sections: Identity, Model Requirements, Responsibilities, Inputs, Outputs, Rules, Quality Bar, Escalation
3. Add the role to `docs/role-selector.md` with its tier and selection criteria
4. Update `docs/glossary.md` if the role introduces new terminology
5. Run `teamwork validate` to check the new role file

## Adding a New Workflow

Workflows live in `agents/workflows/`. To add a new workflow:

1. Create `agents/workflows/<workflow-type>.md` with step-by-step instructions
2. Add the workflow type to the `definitions` map in `internal/workflow/engine.go`
3. Define the step sequence (step number, role, action description)
4. Update `docs/cli.md` to list the new workflow type under `teamwork start`
5. Add tests for the new workflow type

## Review Process

1. **Open a PR** with a clear title and description
2. **Automated checks** run lint, test, and build
3. **Human review** — the repository owner reviews for correctness, style, and scope
4. **Address feedback** — make requested changes as new commits (don't force-push during review)
5. **Merge** — the reviewer merges once all checks pass and feedback is addressed

## Issue Labels and Milestones

Issues use prefix labels in their titles:

- `[CODE]` — implementation tasks
- `[DOCS]` — documentation tasks
- `[PLAN]` — planning/design tasks
- `[DX]` — developer experience improvements
- `[TEST]` — testing tasks

Milestones track phased work (Phase 3, Phase 4, Backlog).

## Reporting Bugs

Open a GitHub issue with:

1. What you expected to happen
2. What actually happened
3. Steps to reproduce
4. Environment details (OS, Go version, Docker version)

## Security Vulnerabilities

**Do not open a public issue for security vulnerabilities.** Instead, please report them privately:

1. Use [GitHub's private vulnerability reporting](https://github.com/JoshLuedeman/teamwork/security/advisories/new)
2. Or email the repository owner directly

See `docs/secrets-policy.md` for the project's secrets and credentials policy.

## Code of Conduct

Be respectful and constructive. We follow the [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/) code of conduct.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
