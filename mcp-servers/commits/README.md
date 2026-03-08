# teamwork-mcp-commits

An MCP server that generates and validates structured, conventional-commit-compliant messages from git diffs.

## What It Does

- **Generate commit messages** — analyzes a `git diff` and produces a conventional commit with detected type, scope, subject, body, and footer.
- **Generate PR descriptions** — creates structured, markdown-formatted pull request descriptions from diffs.
- **Validate commit messages** — checks messages against the [Conventional Commits](https://www.conventionalcommits.org/) specification and returns actionable errors and suggestions.
- **List commit types** — returns all valid conventional commit types with descriptions and semver impact.

## Installation

### pip

```bash
cd mcp-servers/commits
pip install .
```

### uvx (ephemeral)

```bash
cd mcp-servers/commits
uvx --from . teamwork-mcp-commits
```

### Docker

```bash
cd mcp-servers/commits
docker build -t teamwork-mcp-commits .
docker run -i teamwork-mcp-commits
```

## MCP Client Configuration

Add to your MCP client config (e.g. Claude Desktop, VS Code):

```json
{
  "mcpServers": {
    "commits": {
      "command": "teamwork-mcp-commits",
      "args": []
    }
  }
}
```

Or with Docker:

```json
{
  "mcpServers": {
    "commits": {
      "command": "docker",
      "args": ["run", "-i", "teamwork-mcp-commits"]
    }
  }
}
```

## Tools

### `generate_commit_message`

Analyze a git diff and produce a conventional-commit message.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `diff` | `str` | Yes | Unified diff string (output of `git diff`) |
| `hint` | `str` | No | Human hint to guide type/subject detection |

**Returns:** `{type, scope, subject, body, breaking_change, footer, full_message}`

### `generate_pr_description`

Generate a structured PR description from a diff.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `diff` | `str` | Yes | Unified diff string |
| `template` | `str` | No | Custom format-string template |

**Returns:** Markdown string with sections: Title, Summary, Motivation, Changes Made, Test Plan, Breaking Changes.

### `validate_commit_message`

Validate a message against the Conventional Commits spec.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `message` | `str` | Yes | Full commit message to validate |

**Returns:** `{valid, type, scope, errors, suggestions}`

### `list_commit_types`

List all valid conventional commit types.

**Returns:** `[{type, description, semver_impact}]`

## Examples

### Generate a commit message

```python
result = await generate_commit_message(
    diff="diff --git a/tests/test_auth.py b/tests/test_auth.py\nnew file mode 100644\n...",
    hint="add authentication tests"
)
# {
#   "type": "test",
#   "scope": None,
#   "subject": "add authentication tests",
#   "full_message": "test: add authentication tests"
# }
```

### Validate a commit message

```python
result = await validate_commit_message("feat: add login endpoint")
# {"valid": True, "type": "feat", "scope": None, "errors": [], "suggestions": []}

result = await validate_commit_message("Added login endpoint.")
# {"valid": False, "type": None, "errors": ["Header does not match..."], ...}
```

## Running Tests

```bash
cd mcp-servers/commits
pip install mcp pytest
python -m pytest tests/ -v
```
