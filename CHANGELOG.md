# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- **`teamwork cancel` CLI command** — Cancel active or blocked workflows with optional reason (#63)
- **`teamwork fail` CLI command** — Mark workflows as failed with required reason (#63)
- **`teamwork doctor` CLI command** — Environment diagnostics with actionable fix suggestions (#49)
- **`CONTRIBUTING.md`** — Contribution guide covering setup, standards, and PR process (#56)
- **`teamwork memory` CLI command** — `add`, `search`, `list`, `sync` subcommands for managing structured project memory (#66)
- **`teamwork metrics` CLI command** — `summary` and `roles` subcommands for workflow analytics (#67)
- **`teamwork repos` CLI command** — List configured repositories and their status (#73)
- **Defect escape rate tracking** — `defect_source` field on metrics events, `LogDefect()` function, `DefectEscapeRate()` on Summary (#68)
- **Cost tracking in metrics** — `LogWithCost()` function, `TotalCost` aggregation in Summary (#72)
- **Multi-repo config** — `repos` section in config.yaml with hub-spoke coordination model (#70)
- **Repo field in workflow state** — `StepRecord` and `NextAction` track target repo (#75)
- **Cross-repo status/next** — `teamwork status` and `teamwork next` show repo context when configured (#74, #76)
- **Hub-spoke memory sync** — `teamwork memory sync --repo <name> --domain <domains>` copies entries between repos (#77)
- **Tests** — Config tests for repos parsing, metrics tests for defect/cost tracking

### Changed

- Updated `docs/cli.md` with memory, metrics, repos, and multi-repo documentation (#78)
- Updated `docs/protocols.md` with multi-repo hub-spoke model and repos config schema
- Updated README with new CLI features

## [Phase 2] — 2026-03-03

### Added

- **Orchestrator role** — New 8th core role for coordinating workflow state machines
- **Go CLI application** — `teamwork` CLI for workflow lifecycle management
  - `teamwork validate` — Validate `.teamwork/` directory structure (exit codes: 0=pass, 1=fail, 2=cannot run)
  - `teamwork install` — Install Teamwork framework files into a project
  - `teamwork update` — Update framework files to a new version
  - `teamwork init` — Initialize `.teamwork/` directory structure
  - `teamwork start` — Start a new workflow instance
  - `teamwork status` — Show active workflow status
  - `teamwork next` — List pending actions requiring human attention
  - `teamwork approve` — Approve a quality gate to advance a workflow
  - `teamwork block` — Block a workflow with a reason
  - `teamwork complete` — Mark a workflow as complete
  - `teamwork history` — Show full workflow history
  - `teamwork dashboard` — Interactive TUI dashboard for workflow monitoring
- **gh-teamwork CLI extension** — GitHub CLI extension wrapping `teamwork install`/`teamwork update`
  - `gh teamwork init` — Initialize Teamwork via GitHub CLI
  - `gh teamwork update` — Update framework files via GitHub CLI
  - Falls back to Docker if binary not found
- **Model tier recommendations** — Each role has a "Model Requirements" section specifying optimal model tier (premium/standard/fast)
- **ADR-004** — Validate command design with protocol validation and exit codes
- **ADR-005** — Install and Update commands with tarball fetching and conflict detection

### Changed

- Updated README with orchestrator role and Phase 2 progress
- Added model escalation instructions to Claude, Cursor, and Copilot instructions
- GitHub milestone numbering: #1=Orchestration (pre-existing), #2=Phase 1 install/update, #3=Phase 2 gh extension, #4=Phase 3 GitHub App

### Fixed

- Authenticate HTTP requests to GitHub with GH_TOKEN/GITHUB_TOKEN for private repos

## [Phase 1] — 2025-07-18

### Added

- Initial project template with role-based agent framework
- Eight core agent roles in `agents/roles/`:
  - Planner, Architect, Coder, Tester, Reviewer, Security Auditor, Documenter, Orchestrator
- Optional roles in `agents/roles/optional/`:
  - Triager, DevOps, Dependency Manager, Refactorer
- Ten workflow definitions in `agents/workflows/`:
  - Feature, Bugfix, Refactor, Hotfix, Security Response, Dependency Update, Documentation, Spike, Release, Rollback
- Agent framework documentation:
  - `agents/README.md` — Role system overview
  - `docs/conventions.md` — Code, git, and testing standards
  - `docs/glossary.md` — Framework terminology
  - `docs/architecture.md` — ADR guidance and storage
- GitHub issue and PR templates
- Customizable shell scripts for linting, testing, and building
- CI/CD Makefile with targets for lint, test, build, check
- Architecture Decision Records (ADRs 001-003)
