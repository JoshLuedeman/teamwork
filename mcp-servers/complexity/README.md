# teamwork-mcp-complexity

An MCP server that provides structured code-complexity metrics using the [lizard](https://github.com/terryyin/lizard) static analysis library.

## What It Does

Wraps lizard's cyclomatic-complexity analysis behind five MCP tools so agents can programmatically inspect code quality across **30+ languages** including C, C++, Go, Java, JavaScript, TypeScript, Python, Rust, Ruby, C#, Swift, and more.

## Installation

### pip

```bash
pip install -e mcp-servers/complexity/
```

### uvx (no install)

```bash
uvx --from ./mcp-servers/complexity teamwork-mcp-complexity
```

### Docker

```bash
docker build -t teamwork-mcp-complexity mcp-servers/complexity/
docker run -i --rm -v "$(pwd):/workspace" teamwork-mcp-complexity
```

## MCP Client Configuration

Add to your MCP client config (e.g. `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "complexity": {
      "command": "teamwork-mcp-complexity",
      "args": []
    }
  }
}
```

Or with Docker:

```json
{
  "mcpServers": {
    "complexity": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "/your/project:/workspace",
        "teamwork-mcp-complexity"
      ]
    }
  }
}
```

## Tools

### `get_file_complexity`

Get per-function complexity metrics for a single file.

| Parameter | Type | Description |
|-----------|------|-------------|
| `file`    | str  | Path to the source file |

Returns: `{file, avg_ccn, total_functions, functions: [{name, line, ccn, length, parameters, token_count}]}`

### `get_high_complexity_functions`

Find all functions exceeding a CCN threshold across a codebase path.

| Parameter   | Type | Default | Description |
|-------------|------|---------|-------------|
| `path`      | str  |         | File or directory to scan |
| `threshold` | int  | 10      | CCN threshold |

Returns: `[{file, function, line, ccn, length}]` sorted by CCN descending.

### `get_project_complexity_report`

Project-wide complexity summary with aggregates and rankings.

| Parameter   | Type | Default | Description |
|-------------|------|---------|-------------|
| `path`      | str  |         | Directory to scan |
| `threshold` | int  | 10      | CCN threshold |

Returns: `{avg_ccn, total_functions, functions_exceeding_threshold, pct_exceeding, top_10_most_complex, files_by_avg_ccn}`

### `get_function_complexity`

Detailed metrics for a specific function by name.

| Parameter  | Type | Description |
|------------|------|-------------|
| `file`     | str  | Path to the source file |
| `function` | str  | Function name to look up |

Returns: `{name, line, ccn, length, parameters, token_count}`

### `compare_complexity`

Compare complexity between two directory trees (e.g., base vs. head branch).

| Parameter   | Type | Default | Description |
|-------------|------|---------|-------------|
| `base_path` | str  |         | Base directory |
| `head_path` | str  |         | Head directory |
| `threshold` | int  | 10      | Threshold for new functions |

Returns: `{regressions: [{file, function, base_ccn, head_ccn, delta}], improvements: [...], new_high_complexity: [...]}`

## CCN Thresholds

| CCN Range | Risk Level | Recommendation |
|-----------|------------|----------------|
| 1–5       | Low        | Simple, well-tested code |
| 6–10      | Moderate   | Acceptable but review carefully |
| 11–20     | High       | Consider refactoring |
| 21–50     | Very High  | Must refactor — hard to test |
| 50+       | Unmaintainable | Refactor immediately |

## Development

```bash
cd mcp-servers/complexity
pip install -e ".[dev]" 2>/dev/null || pip install -e .
pip install pytest
python -m pytest tests/ -v
```
