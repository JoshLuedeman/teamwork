# teamwork-mcp-coverage

MCP server that parses test coverage reports (lcov, Istanbul JSON, Go `cover.out`) and surfaces coverage gaps to AI agents.

## Installation

```bash
# pip
pip install -e mcp-servers/coverage

# uvx (run without installing)
uvx --from ./mcp-servers/coverage teamwork-mcp-coverage

# Docker
docker build -t teamwork-mcp-coverage mcp-servers/coverage
docker run -i --rm teamwork-mcp-coverage
```

## MCP Client Configuration

### VS Code / GitHub Copilot

Add to `.vscode/mcp.json`:

```json
{
  "servers": {
    "coverage": {
      "command": "uvx",
      "args": [
        "--from", "./mcp-servers/coverage",
        "teamwork-mcp-coverage"
      ]
    }
  }
}
```

Or with Docker:

```json
{
  "servers": {
    "coverage": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "${workspaceFolder}:/workspace",
        "teamwork-mcp-coverage"
      ]
    }
  }
}
```

## Tools

| Tool | Description |
|------|-------------|
| `load_coverage_report(path, format)` | Load a coverage report file. Format: `lcov`, `istanbul`, or `go`. |
| `get_coverage_summary(file?)` | Get coverage summary — overall or per-file. Returns line/branch percentages and uncovered functions. |
| `get_uncovered_files(threshold?)` | List files below a coverage threshold (default 80%). Sorted by coverage ascending. |
| `get_function_coverage(file, function)` | Get line-level coverage detail for a specific function — executed lines, missed lines, branch coverage. |

## Supported Formats

| Format | File | Notes |
|--------|------|-------|
| **lcov** | `lcov.info`, `coverage.lcov` | Parses `SF`, `DA`, `BRDA`, `FN`, `FNDA` records |
| **Istanbul** | `coverage-summary.json` | Istanbul / nyc summary format |
| **Go** | `cover.out` | `go test -coverprofile` output |

## Development

```bash
cd mcp-servers/coverage
pip install -e .
pip install pytest
python -m pytest tests/ -v
```
