# teamwork-mcp-adr

An MCP server for searching, creating, and managing **Architecture Decision Records** (ADRs) stored as [MADR](https://adr.github.io/madr/) markdown files.

## What it does

- **List** all ADRs with optional status filtering
- **Read** a single ADR with fully parsed sections
- **Search** across all ADR content (case-insensitive full-text)
- **Create** new ADRs in MADR format with auto-incremented IDs
- **Update** ADR status (draft → accepted → superseded → deprecated)
- **Link** ADRs to source code via inline comments

## Installation

### pip

```bash
pip install .
```

### uvx (no install)

```bash
uvx --from . teamwork-mcp-adr
```

### Docker

```bash
docker build -t teamwork-mcp-adr .
docker run -i --rm -v "$(pwd)/docs/decisions:/app/docs/decisions" teamwork-mcp-adr
```

## MCP client configuration

Add to your MCP client config (e.g. Claude Desktop, VS Code):

```json
{
  "mcpServers": {
    "teamwork-mcp-adr": {
      "command": "teamwork-mcp-adr",
      "args": []
    }
  }
}
```

Or with Docker:

```json
{
  "mcpServers": {
    "teamwork-mcp-adr": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "${workspaceFolder}/docs/decisions:/app/docs/decisions",
        "teamwork-mcp-adr"
      ]
    }
  }
}
```

## Tools

| Tool | Description |
|------|-------------|
| `list_adrs` | List all ADRs, optionally filtered by status |
| `get_adr` | Get full parsed content of a specific ADR |
| `search_adrs` | Full-text search across all ADR content |
| `create_adr` | Create a new ADR in MADR format with auto-incremented ID |
| `update_adr_status` | Update the status field in an existing ADR |
| `link_adr_to_code` | Add a code comment linking source code to an ADR |

### Parameters

**`list_adrs(status?, decisions_dir?)`**
- `status` — Filter: `draft`, `accepted`, `superseded`, `deprecated`
- `decisions_dir` — Path to ADR directory (default: `docs/decisions`)

**`get_adr(id, decisions_dir?)`**
- `id` — ADR identifier, e.g. `"001"` or `"1"`

**`search_adrs(query, decisions_dir?)`**
- `query` — Search string (case-insensitive substring match)

**`create_adr(title, context, decision, consequences, status?, decisions_dir?)`**
- `title` — Short title for the decision
- `context` — Why this decision is needed
- `decision` — What was decided
- `consequences` — What happens as a result
- `status` — Initial status (default: `"accepted"`)

**`update_adr_status(id, status, superseded_by?, decisions_dir?)`**
- `status` — New status value
- `superseded_by` — Optional ID of the superseding ADR

**`link_adr_to_code(adr_id, file, line?, reason?)`**
- `file` — Path to the source file
- `line` — Line number to insert before (default: top of file)
- `reason` — Description of why this code relates to the ADR

## MADR format

ADR files use [Markdown Architectural Decision Records](https://adr.github.io/madr/) format with YAML frontmatter:

```markdown
---
id: "001"
title: "Use React for frontend"
status: "accepted"
date: "2024-01-15"
---

# ADR-001: Use React for frontend

## Status
Accepted

## Context
We need a frontend framework for the dashboard.

## Decision
Use React 18 with TypeScript.

## Consequences
Team needs React expertise. Good ecosystem support.
```

Files are named `NNN-slug.md` (e.g. `001-use-react.md`) with zero-padded 3-digit IDs.

## Development

```bash
# Install dev dependencies
pip install -e ".[dev]" pytest pytest-asyncio

# Run tests
PYTHONPATH=src python -m pytest tests/ -v
```
