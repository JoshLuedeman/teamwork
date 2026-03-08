# Teamwork MCP Servers

Custom [Model Context Protocol](https://modelcontextprotocol.io/) servers built specifically for Teamwork agent workflows. Each server exposes focused tools that agents invoke during planning, coding, reviewing, and releasing.

## Servers

| Server | Description | Issue |
|--------|-------------|-------|
| [coverage](coverage/) | Test coverage report analysis — lcov, Istanbul, Go `cover.out` | [#33](https://github.com/JoshLuedeman/teamwork/issues/33) |
| [commits](commits/) | Conventional commit message generation and validation from diffs | [#31](https://github.com/JoshLuedeman/teamwork/issues/31) |
| [adr](adr/) | Architecture Decision Record search, creation, and management | [#34](https://github.com/JoshLuedeman/teamwork/issues/34) |
| [changelog](changelog/) | Changelog generation and release notes using git-cliff | [#30](https://github.com/JoshLuedeman/teamwork/issues/30) |
| [complexity](complexity/) | Code complexity analysis — cyclomatic complexity for 30+ languages | [#32](https://github.com/JoshLuedeman/teamwork/issues/32) |

## Installation

Each server is a standalone Python package. Install whichever servers your workflow needs.

### pip install

```bash
pip install teamwork-mcp-coverage
pip install teamwork-mcp-commits
pip install teamwork-mcp-adr
pip install teamwork-mcp-changelog
pip install teamwork-mcp-complexity
```

### uvx (run without installing)

```bash
uvx teamwork-mcp-coverage
uvx teamwork-mcp-commits
uvx teamwork-mcp-adr
uvx teamwork-mcp-changelog
uvx teamwork-mcp-complexity
```

### Docker

Each server includes a `Dockerfile`. Build and run individually:

```bash
cd mcp-servers/coverage
docker build -t teamwork-mcp-coverage .
docker run --rm teamwork-mcp-coverage

cd mcp-servers/commits
docker build -t teamwork-mcp-commits .
docker run --rm teamwork-mcp-commits
```

## Configuration

After installing, add the servers to `.teamwork/config.yaml` under `mcp_servers:` — they are already pre-configured in this repository. Agents discover them automatically from the config.

See [docs/mcp.md](../docs/mcp.md) for full setup instructions, client configuration (Claude Desktop, VS Code), and role mappings.

## Development

Each server lives in its own directory with:

```
mcp-servers/<name>/
├── Dockerfile
├── pyproject.toml
├── README.md
├── src/
│   └── teamwork_mcp_<name>/
└── tests/
```

Run tests for a single server:

```bash
cd mcp-servers/coverage
pip install -e ".[dev]"
pytest
```
