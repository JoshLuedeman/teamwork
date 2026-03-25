# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Changed

- **Merged `install` into `init`** — `teamwork init` now fetches framework files (agents, skills, docs, instructions) from upstream AND creates the `.teamwork/` config directory in a single command. Previously, users needed to run both `teamwork install` and `teamwork init` separately. The `install` command is now a deprecated alias that delegates to `init`. (#170)

## [v1.3.1] — 2026-03-16

### Fixed

- **Windows installer path separator bug** — Replaced `filepath.Clean` with `path.Clean` in tarball extraction so that framework files in subdirectories (`.github/agents/`, `docs/`) are correctly installed on Windows. `filepath.Clean` converted forward slashes to backslashes, causing `isFrameworkFile()` to silently skip subdirectory paths.

## [v1.3.0] — 2026-03-16

### Fixed

- **Lowercase Go module path** — Changed module declaration from `github.com/JoshLuedeman/teamwork` to `github.com/joshluedeman/teamwork` so that `go install` works with the natural lowercase GitHub URL. Go module paths are case-sensitive; the mixed-case path caused `sum.golang.org` 404 errors when users typed lowercase. Updated all imports and `--source` flag defaults across 30 files. (PR #125)

## [v1.2.0] — 2026-03-08

### Added

- **Phase 5: Developer Experience & Template Polish**
  - **Agent auto-dispatch guidance** — Added `## Agent & Skill Usage` section to `copilot-instructions.md` instructing Copilot to automatically dispatch Custom Agents when requests match their purpose (#82, PR #107)
  - **Product Owner and QA Lead roles** — Two new optional agent roles in `.github/agents/`: Product Owner for backlog management and business priorities; QA Lead for test strategy, quality gates, and release readiness (#60, PR #108)
  - **`teamwork role create` command** — Scaffolds new Custom Agent files with YAML frontmatter, all required sections, and TODO placeholders; validates kebab-case names and rejects built-in role conflicts; supports `--description` and `--tier` flags (10 tests) (#42, PR #109)
  - **GitHub Actions CI template** — `.github/workflows/teamwork-ci.yaml.example` template for validating Teamwork structure on push/PR; added `--ci` flag to `teamwork validate` for machine-readable PASS/FAIL output (4 tests) (#59, PR #110)
  - **Interactive `teamwork init` wizard** — Prompts for project name, GitHub repo, and optional roles when stdin is a TTY; auto-detects non-TTY for CI/piped usage; supports `--non-interactive` flag (10 tests) (#52, PR #111)
  - **`teamwork logs` command** — Views and filters `.teamwork/metrics/` JSONL activity logs with `--role`, `--action`, `--tail`, and `--since` (ISO dates and relative durations like `24h`, `7d`) (13 tests) (#51, PR #112)
  - **`teamwork start --dry-run` flag** — Previews workflow steps, roles, model tiers, quality gates, and skip conditions without creating state files (6 tests) (#50, PR #113)

## [v1.1.0] — 2026-03-08

### Added

- **Phase 4: MCP Integration** — Full MCP server support for Teamwork agents
  - **ADR-007 update** — Updated MCP integration strategy with 8 server definitions and role mappings (#89)
  - **MCP config section** — Added `mcp_servers` config in `.teamwork/config.yaml` with 8 server entries; added `MCPServer` struct to Go config package (#91)
  - **MCP agent tools** — Added `## MCP Tools` section to all 15 agent files with role-specific server assignments (#92)
  - **MCP instructions** — Added MCP tools guidance to `copilot-instructions.md` (#90)
  - **MCP setup guide** — Created `docs/mcp.md` with comprehensive setup, configuration, and role-to-server mapping (#93)
  - **MCP validation** — Extended `teamwork validate` with MCP config checks: URL/command mutual exclusivity, valid roles, env var format (#94)
  - **`teamwork mcp list`** — CLI command listing configured MCP servers with role filtering (#95)
  - **`teamwork mcp config`** — CLI command generating MCP client configuration for VS Code and Claude Desktop (#95)
  - **MCP README section** — Added MCP overview to project README (#96)
- **5 custom Python MCP servers** in `mcp-servers/` — each with pyproject.toml, Dockerfile, README, and tests
  - **teamwork-mcp-coverage** — Test coverage report analysis for lcov, Istanbul, and Go cover.out formats (30 tests) (#101)
  - **teamwork-mcp-adr** — Architecture Decision Record search, creation, and management (24 tests) (#98)
  - **teamwork-mcp-commits** — Conventional commit message generation and validation from diffs (70 tests) (#99)
  - **teamwork-mcp-changelog** — Changelog generation and release notes using git-cliff (43 tests) (#100)
  - **teamwork-mcp-complexity** — Code complexity analysis using lizard for 30+ languages (15 tests) (#97)
  - **MCP servers config update** — Added all 5 servers to config.yaml, docs/mcp.md, agent files, and README (#102)
- **Auto-create setup issue** — `teamwork update` now creates a GitHub issue assigned to Copilot when unfilled `<!-- CUSTOMIZE -->` placeholders are detected (#83)
- **Release process documentation** — `docs/releasing.md` covering semver strategy, release checklist, CHANGELOG conventions, and dual-repo sync (#104)
- **`make release` target** — Automated release process: tests, cross-compilation, CHANGELOG verification, git tag, GitHub Release (#103)
- **`teamwork --version` flag** — Version embedded in binary via ldflags (#103)

## [v1.0.0] — 2026-03-08

### Added

- **Phase 3: GitHub App + Cloudflare Worker auto-install** — Automatic Teamwork framework installation for new repositories
  - **ADR-006** — GitHub App + Cloudflare Worker design document (#15)
  - **Cloudflare Worker** — TypeScript webhook handler at `workers/github-app/` (#16)
    - HMAC-SHA256 webhook signature verification via Web Crypto API
    - GitHub App JWT → installation token authentication
    - Git Data API for atomic single-commit file push
    - Fork detection and `.teamwork-skip` opt-out support
    - Zero runtime dependencies (44 Vitest tests)
  - **Deployment config** — `wrangler.toml` with secrets documentation (#17)
  - **Setup guide** — Step-by-step instructions at `docs/github-app-setup.md` (#18)
  - **E2E test** — Manual verification script at `workers/github-app/e2e/` (#19)
- **`teamwork cancel` CLI command** — Cancel active or blocked workflows with optional reason (#63)
- **`teamwork fail` CLI command** — Mark workflows as failed with required reason (#63)
- **`teamwork doctor` CLI command** — Environment diagnostics with actionable fix suggestions (#49)
- **`CONTRIBUTING.md`** — Contribution guide covering setup, standards, and PR process (#56)
- **ADR-007** — MCP integration strategy design document (#20)
- **`teamwork memory` CLI command** — `add`, `search`, `list`, `sync` subcommands for managing structured project memory (#66)
- **`teamwork metrics` CLI command** — `summary` and `roles` subcommands for workflow analytics (#67)
- **`teamwork repos` CLI command** — List configured repositories and their status (#73)
- **Defect escape rate tracking** — `defect_source` field on metrics events, `LogDefect()` function, `DefectEscapeRate()` on Summary (#68)
- **Cost tracking in metrics** — `LogWithCost()` function, `TotalCost` aggregation in Summary (#72)
- **Multi-repo config** — `repos` section in config.yaml with hub-spoke coordination model (#70)
- **Repo field in workflow state** — `StepRecord` and `NextAction` track target repo (#75)
- **Cross-repo status/next** — `teamwork status` and `teamwork next` show repo context when configured (#74, #76)
- **Hub-spoke memory sync** — `teamwork memory sync --repo <name> --domain <domains>` copies entries between repos (#77)
- **Copilot-native restructure** — Agents at `.github/agents/*.agent.md`, Skills at `.github/skills/*/SKILL.md`, Instructions at `.github/instructions/`
- **15 Custom Agents** — planner, architect, coder, tester, reviewer, security-auditor, documenter, orchestrator, triager, devops, dependency-manager, refactorer, lint-agent, api-agent, dba-agent
- **10 Skills** — feature, bugfix, refactor, hotfix, security-response, dependency-update, documentation, spike, release, rollback
- **Tests** — Config tests for repos parsing, metrics tests for defect/cost tracking

### Changed

- Updated `docs/cli.md` with memory, metrics, repos, and multi-repo documentation (#78)

### Security

- **Fixed zip-slip vulnerability** in tarball extraction — path traversal via `..` now rejected (CWE-22)
- **Fixed path traversal** via workflow IDs in state and handoff operations (CWE-22)
- **Added file size limits** (10MB) to tarball extraction to prevent decompression bombs (CWE-400)
- **Added HTTP timeout** (120s) to tarball fetch to prevent indefinite hangs (CWE-400)
- **Fixed workflow ID validation** — reject `..` and absolute paths in state.Load/Save and handoff.Save

### Fixed

- **Fixed panic** on short commit SHA in installer (`[:12]` without length check)
- **Fixed `os.Exit()` in cobra RunE handlers** — validate and doctor now return `ExitError` instead of calling `os.Exit` directly, enabling proper cleanup and testability
- **Fixed `Approve()` missing metrics** — now logs `LogComplete` and `LogStart` when advancing steps
- **Fixed latent panic** in `truncate()` when `n <= 3`
- **Fixed `os.Stat` error handling** in init command — now properly distinguishes "not exists" from permission errors
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
