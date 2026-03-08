# teamwork-mcp-changelog

An MCP server that wraps [git-cliff](https://git-cliff.org/) for structured changelog generation. It exposes tools for generating changelogs, previewing release notes, listing unreleased commits, and suggesting the next semantic version — all via the [Model Context Protocol](https://modelcontextprotocol.io/).

## Prerequisites

- **Python 3.10+**
- **git** — must be available on `PATH`
- **git-cliff** — required by `generate_changelog` and `preview_release_notes` (the other tools work without it)

Install git-cliff:

```bash
# macOS
brew install git-cliff

# Cargo
cargo install git-cliff

# See https://git-cliff.org/docs/installation for more options
```

## Installation

### pip

```bash
pip install .
```

### uvx (ephemeral)

```bash
uvx --from . teamwork-mcp-changelog
```

### Docker

```bash
docker build -t teamwork-mcp-changelog .
docker run --rm -i -v "$(pwd):/repo" -w /repo teamwork-mcp-changelog
```

## MCP Client Configuration

Add to your MCP client config (e.g. Claude Desktop, VS Code):

```json
{
  "mcpServers": {
    "changelog": {
      "command": "teamwork-mcp-changelog"
    }
  }
}
```

Or with Docker:

```json
{
  "mcpServers": {
    "changelog": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-v", "${workspaceFolder}:/repo",
        "-w", "/repo",
        "teamwork-mcp-changelog"
      ]
    }
  }
}
```

## Tool Reference

### `generate_changelog`

Generate a changelog between two git refs using git-cliff.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `from_ref` | `str` | *required* | Starting git ref (tag, branch, or SHA) |
| `to_ref` | `str` | `"HEAD"` | Ending git ref |
| `format` | `str` | `"markdown"` | `"markdown"` for string, `"json"` for structured output |

**Requires:** git-cliff

### `preview_release_notes`

Generate release notes for the next (unreleased) version since the last tag.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `tag` | `str \| None` | `None` | Optional tag name for the release |

**Returns:** `{title, body, breaking_changes: [str], highlights: [str]}`

**Requires:** git-cliff

### `get_unreleased_commits`

Get a structured list of commits since the last release tag.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `from_tag` | `str \| None` | `None` | Start from this tag (auto-detects latest if omitted) |

**Returns:** `[{hash, type, scope, subject, breaking, author, date}]`

### `suggest_next_version`

Analyze unreleased commits and suggest the next semantic version.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `current` | `str` | *required* | Current version (e.g. `"v1.2.3"` or `"1.2.3"`) |

**Returns:** `{version, bump: "major"|"minor"|"patch", reasoning}`

**Rules:**
- `BREAKING CHANGE` → major bump
- `feat` → minor bump
- `fix` / `chore` / `docs` → patch bump

## Development

```bash
# Install dev dependencies
pip install -e ".[dev]" pytest pytest-asyncio mcp

# Run tests
python -m pytest tests/ -v
```
