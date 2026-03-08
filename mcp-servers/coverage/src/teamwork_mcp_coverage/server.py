"""MCP server for test coverage report analysis.

Parses lcov, Istanbul JSON, and Go cover.out reports and surfaces
coverage gaps to agents via MCP tools.
"""

from __future__ import annotations

import json
import os
import re
from pathlib import Path

from mcp.server.fastmcp import FastMCP

# ---------------------------------------------------------------------------
# Module-level state – persists across tool calls within a session
# ---------------------------------------------------------------------------

_coverage_data: dict[str, dict] = {}
"""
Keyed by file path.  Each value:
{
    "lines": {line_number: execution_count, ...},
    "branches": [(line, block, branch, count), ...],
    "functions": {name: {"line": int, "count": int}, ...},
}
"""

# ---------------------------------------------------------------------------
# Parsing helpers  (pure functions – easily testable)
# ---------------------------------------------------------------------------


def parse_lcov(text: str) -> dict[str, dict]:
    """Parse an lcov-format coverage report.

    Returns a dict keyed by source file with line, branch, and function data.
    """
    result: dict[str, dict] = {}
    current_file: str | None = None
    lines: dict[int, int] = {}
    branches: list[tuple[int, int, int, int]] = []
    functions: dict[str, dict] = {}

    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line:
            continue

        if line.startswith("SF:"):
            current_file = line[3:]
            lines = {}
            branches = []
            functions = {}

        elif line.startswith("DA:"):
            parts = line[3:].split(",")
            line_no = int(parts[0])
            count = int(parts[1])
            lines[line_no] = count

        elif line.startswith("BRDA:"):
            parts = line[5:].split(",")
            branch_line = int(parts[0])
            block = int(parts[1])
            branch = int(parts[2])
            count = 0 if parts[3] == "-" else int(parts[3])
            branches.append((branch_line, block, branch, count))

        elif line.startswith("FN:"):
            parts = line[3:].split(",", 1)
            fn_line = int(parts[0])
            fn_name = parts[1]
            functions[fn_name] = {"line": fn_line, "count": 0}

        elif line.startswith("FNDA:"):
            parts = line[5:].split(",", 1)
            fn_count = int(parts[0])
            fn_name = parts[1]
            if fn_name in functions:
                functions[fn_name]["count"] = fn_count

        elif line == "end_of_record":
            if current_file is not None:
                result[current_file] = {
                    "lines": dict(lines),
                    "branches": list(branches),
                    "functions": dict(functions),
                }
            current_file = None

    return result


def parse_istanbul(text: str) -> dict[str, dict]:
    """Parse an Istanbul / nyc coverage-summary.json report.

    Returns a dict keyed by source file (the ``"total"`` key is included
    with file path ``"total"``).
    """
    raw: dict = json.loads(text)
    result: dict[str, dict] = {}

    for file_key, metrics in raw.items():
        lines_info = metrics.get("lines", {})
        branches_info = metrics.get("branches", {})
        functions_info = metrics.get("functions", {})

        total_lines = lines_info.get("total", 0)
        covered_lines = lines_info.get("covered", 0)

        # Build synthetic line data (we don't have line-level detail from
        # summary format, so we store aggregate counts only).
        line_data: dict[int, int] = {}
        for i in range(1, total_lines + 1):
            line_data[i] = 1 if i <= covered_lines else 0

        total_branches = branches_info.get("total", 0)
        covered_branches = branches_info.get("covered", 0)
        branch_data: list[tuple[int, int, int, int]] = []
        for i in range(total_branches):
            count = 1 if i < covered_branches else 0
            branch_data.append((i + 1, 0, 0, count))

        # Functions – summary doesn't give names, store counts only.
        total_fns = functions_info.get("total", 0)
        covered_fns = functions_info.get("covered", 0)
        fn_data: dict[str, dict] = {}
        for i in range(total_fns):
            name = f"fn_{i + 1}"
            fn_data[name] = {
                "line": i + 1,
                "count": 1 if i < covered_fns else 0,
            }

        result[file_key] = {
            "lines": line_data,
            "branches": branch_data,
            "functions": fn_data,
        }

    return result


_GO_LINE_RE = re.compile(
    r"^(.+?):(\d+)\.\d+,(\d+)\.\d+\s+(\d+)\s+(\d+)$"
)


def parse_go_cover(text: str) -> dict[str, dict]:
    """Parse a Go ``cover.out`` file.

    Each data line has the form:
        file.go:startLine.startCol,endLine.endCol numStatements count
    """
    result: dict[str, dict] = {}

    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line or line.startswith("mode:"):
            continue

        m = _GO_LINE_RE.match(line)
        if not m:
            continue

        file_path = m.group(1)
        start_line = int(m.group(2))
        end_line = int(m.group(3))
        num_stmts = int(m.group(4))
        count = int(m.group(5))

        if file_path not in result:
            result[file_path] = {
                "lines": {},
                "branches": [],
                "functions": {},
            }

        entry = result[file_path]
        for ln in range(start_line, end_line + 1):
            # Keep the max count if lines overlap between blocks.
            existing = entry["lines"].get(ln, 0)
            entry["lines"][ln] = max(existing, count)

    return result


# ---------------------------------------------------------------------------
# Aggregate helpers
# ---------------------------------------------------------------------------


def _line_coverage(file_data: dict) -> tuple[int, int]:
    """Return (total_lines, covered_lines) for a single file entry."""
    lines = file_data.get("lines", {})
    total = len(lines)
    covered = sum(1 for c in lines.values() if c > 0)
    return total, covered


def _branch_coverage(file_data: dict) -> tuple[int, int]:
    """Return (total_branches, covered_branches) for a single file entry."""
    branches = file_data.get("branches", [])
    total = len(branches)
    covered = sum(1 for (_, _, _, c) in branches if c > 0)
    return total, covered


def _line_pct(file_data: dict) -> float:
    total, covered = _line_coverage(file_data)
    return (covered / total * 100) if total else 100.0


def _branch_pct(file_data: dict) -> float:
    total, covered = _branch_coverage(file_data)
    return (covered / total * 100) if total else 100.0


def _uncovered_functions(file_data: dict, file_path: str) -> list[dict]:
    functions = file_data.get("functions", {})
    return [
        {"name": name, "file": file_path, "line": info["line"]}
        for name, info in functions.items()
        if info["count"] == 0
    ]


# ---------------------------------------------------------------------------
# MCP server
# ---------------------------------------------------------------------------

server = FastMCP("teamwork-mcp-coverage")


@server.tool()
async def load_coverage_report(path: str, format: str) -> dict:
    """Load a coverage report file.

    Args:
        path: Path to the coverage report file.
        format: Report format – ``'lcov'``, ``'istanbul'``, or ``'go'``.

    Returns:
        Summary with *files_loaded*, *total_lines*, and *covered_lines*.
    """
    fmt = format.lower()
    if fmt not in ("lcov", "istanbul", "go"):
        raise ValueError(f"Unsupported format: {format!r}. Use 'lcov', 'istanbul', or 'go'.")

    resolved = Path(path).expanduser().resolve()
    if not resolved.is_file():
        raise FileNotFoundError(f"Coverage report not found: {path}")

    text = resolved.read_text(encoding="utf-8")

    if fmt == "lcov":
        parsed = parse_lcov(text)
    elif fmt == "istanbul":
        parsed = parse_istanbul(text)
    else:
        parsed = parse_go_cover(text)

    _coverage_data.update(parsed)

    total_lines = 0
    covered_lines = 0
    for fd in parsed.values():
        t, c = _line_coverage(fd)
        total_lines += t
        covered_lines += c

    return {
        "files_loaded": len(parsed),
        "total_lines": total_lines,
        "covered_lines": covered_lines,
    }


@server.tool()
async def get_coverage_summary(file: str | None = None) -> dict:
    """Get coverage summary — overall or for a specific file.

    Args:
        file: Optional file path.  If omitted, returns aggregate summary.

    Returns:
        Dict with *line_pct*, *branch_pct*, *total_lines*, *covered_lines*,
        and *uncovered_functions*.
    """
    if not _coverage_data:
        raise RuntimeError("No coverage data loaded. Call load_coverage_report first.")

    if file is not None:
        if file not in _coverage_data:
            raise KeyError(f"No coverage data for file: {file!r}")
        fd = _coverage_data[file]
        total, covered = _line_coverage(fd)
        return {
            "line_pct": round(_line_pct(fd), 2),
            "branch_pct": round(_branch_pct(fd), 2),
            "total_lines": total,
            "covered_lines": covered,
            "uncovered_functions": _uncovered_functions(fd, file),
        }

    # Aggregate across all files
    total_lines = 0
    covered_lines = 0
    total_branches = 0
    covered_branches = 0
    uncovered_fns: list[dict] = []

    for fp, fd in _coverage_data.items():
        t, c = _line_coverage(fd)
        total_lines += t
        covered_lines += c
        bt, bc = _branch_coverage(fd)
        total_branches += bt
        covered_branches += bc
        uncovered_fns.extend(_uncovered_functions(fd, fp))

    line_pct = (covered_lines / total_lines * 100) if total_lines else 100.0
    branch_pct = (covered_branches / total_branches * 100) if total_branches else 100.0

    return {
        "line_pct": round(line_pct, 2),
        "branch_pct": round(branch_pct, 2),
        "total_lines": total_lines,
        "covered_lines": covered_lines,
        "uncovered_functions": uncovered_fns,
    }


@server.tool()
async def get_uncovered_files(threshold: float = 80.0) -> list:
    """Get files with line coverage below a threshold.

    Args:
        threshold: Minimum acceptable coverage percentage (default 80).

    Returns:
        List of dicts ``{file, line_pct, branch_pct}`` sorted by coverage
        ascending.
    """
    if not _coverage_data:
        raise RuntimeError("No coverage data loaded. Call load_coverage_report first.")

    results: list[dict] = []
    for fp, fd in _coverage_data.items():
        lp = round(_line_pct(fd), 2)
        if lp < threshold:
            results.append({
                "file": fp,
                "line_pct": lp,
                "branch_pct": round(_branch_pct(fd), 2),
            })

    results.sort(key=lambda r: r["line_pct"])
    return results


@server.tool()
async def get_function_coverage(file: str, function: str) -> dict:
    """Get line-level coverage for a specific function.

    Args:
        file: Source file path.
        function: Function name.

    Returns:
        Dict with *covered* (bool), *executed_lines*, *missed_lines*, and
        *branch_pct*.
    """
    if not _coverage_data:
        raise RuntimeError("No coverage data loaded. Call load_coverage_report first.")

    if file not in _coverage_data:
        raise KeyError(f"No coverage data for file: {file!r}")

    fd = _coverage_data[file]
    functions = fd.get("functions", {})
    if function not in functions:
        raise KeyError(f"Function {function!r} not found in {file!r}")

    fn_info = functions[function]
    fn_line = fn_info["line"]
    fn_count = fn_info["count"]

    # Collect lines belonging to this function.  Heuristic: lines from the
    # function start until the next function or end-of-file.
    all_fn_lines = sorted(info["line"] for info in functions.values())
    fn_idx = all_fn_lines.index(fn_line)
    if fn_idx + 1 < len(all_fn_lines):
        end_line = all_fn_lines[fn_idx + 1] - 1
    else:
        end_line = max(fd["lines"].keys()) if fd["lines"] else fn_line

    executed: list[int] = []
    missed: list[int] = []
    for ln in range(fn_line, end_line + 1):
        if ln in fd["lines"]:
            if fd["lines"][ln] > 0:
                executed.append(ln)
            else:
                missed.append(ln)

    # Branch coverage scoped to the function's line range
    fn_branches = [
        b for b in fd.get("branches", []) if fn_line <= b[0] <= end_line
    ]
    br_total = len(fn_branches)
    br_covered = sum(1 for (_, _, _, c) in fn_branches if c > 0)
    br_pct = (br_covered / br_total * 100) if br_total else 100.0

    return {
        "covered": fn_count > 0,
        "executed_lines": executed,
        "missed_lines": missed,
        "branch_pct": round(br_pct, 2),
    }


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------


def main() -> None:
    """Run the MCP server via stdio transport."""
    server.run()


if __name__ == "__main__":
    main()
