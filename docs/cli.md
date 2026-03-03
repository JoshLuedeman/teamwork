# Teamwork CLI

## Overview

The Teamwork CLI (`teamwork`) provides human oversight and control over agent workflows. It is the primary interface for developers to initialize, monitor, approve, and manage the lifecycle of agent-driven tasks.

## Installation

### Docker (Recommended)

```bash
# Build the image
make docker-build

# Run any command
docker run --rm -v "$(pwd):/project" teamwork status

# Or use the Makefile shortcut
make docker-run CMD="status"
```

Set up a shell alias for convenience:

```bash
alias teamwork='docker run --rm -u "$(id -u):$(id -g)" -v "$(pwd):/project" teamwork'
```

### From Source

Requires Go 1.22+.

```bash
make build-cli    # builds to bin/teamwork
make install-cli  # installs to GOPATH/bin

# Or directly
go install github.com/JoshLuedeman/teamwork/cmd/teamwork@latest
```

## Commands

### `teamwork init`

Initialize the `.teamwork/` directory in the current project.

```bash
teamwork init
```

Creates the `.teamwork/` directory structure with default configuration files needed for workflow coordination.

### `teamwork validate`

Validate the `.teamwork/` directory structure and contents.

```bash
teamwork validate [flags]
```

Checks that configuration, state files, handoff documents, and memory files are well-formed and contain required fields.

**Flags:**
- `--json` — Output results as a JSON array
- `--quiet` — Suppress passing checks (only show failures)

**Exit codes:**
- `0` — All checks passed
- `1` — One or more checks failed
- `2` — Cannot run validation (e.g., `.teamwork/` directory not found)

**Example:**
```bash
# Human-readable output
teamwork validate

# JSON output for CI pipelines
teamwork validate --json

# Only show failures
teamwork validate --quiet
```

### `teamwork start <type> <goal>`

Start a new workflow.

```bash
teamwork start <type> <goal> [flags]
```

**Arguments:**
- `type` — Workflow type (e.g., `feature`, `bugfix`, `refactor`, `hotfix`)
- `goal` — Description of what the workflow should accomplish

**Flags:**
- `--issue <number>` — Link the workflow to a GitHub issue

**Example:**
```bash
teamwork start feature "Add user authentication" --issue 42
```

### `teamwork status`

Show the status of all active workflows.

```bash
teamwork status
```

Displays each active workflow with its current phase, pending actions, and overall progress.

### `teamwork next`

Show pending actions that need human attention.

```bash
teamwork next
```

Lists quality gates awaiting approval, blocked workflows, and other items requiring human input.

### `teamwork approve <id>`

Approve a quality gate to advance a workflow.

```bash
teamwork approve <id>
```

**Arguments:**
- `id` — The workflow identifier (e.g., `feature/42-add-user-authentication`)

### `teamwork block <id>`

Block a workflow with a reason.

```bash
teamwork block <id> --reason <text>
```

**Arguments:**
- `id` — The workflow identifier

**Flags:**
- `--reason <text>` — Explanation for why the workflow is blocked (required)

### `teamwork complete <id>`

Mark a workflow as complete.

```bash
teamwork complete <id>
```

**Arguments:**
- `id` — The workflow identifier

### `teamwork history <id>`

Show the full history of a workflow.

```bash
teamwork history <id>
```

**Arguments:**
- `id` — The workflow identifier

Displays all state transitions, approvals, blocks, and agent actions for the workflow.

### `teamwork dashboard`

Open the interactive TUI dashboard.

```bash
teamwork dashboard
```

Provides a terminal-based interface for monitoring and managing all workflows in real time.

## Examples

A typical workflow session:

```bash
# Initialize in a new project
teamwork init

# Start a feature workflow
teamwork start feature "Add user authentication" --issue 42

# Check what needs to happen next
teamwork next

# After an agent completes a step, check status
teamwork status

# Approve the work and advance
teamwork approve feature/42-add-user-authentication

# View the full history
teamwork history feature/42-add-user-authentication

# Open the interactive dashboard
teamwork dashboard
```

## Configuration

The CLI reads configuration from `.teamwork/config.yaml` in the project root. See [`docs/protocols.md`](protocols.md) for details on the protocol file format and coordination model.

## How It Works

The CLI reads and writes `.teamwork/` protocol files to manage workflow state. It does not invoke AI tools directly — that is the orchestrator agent's responsibility. The CLI provides human visibility and control over the workflow lifecycle, acting as the interface between developers and the agent coordination layer.
