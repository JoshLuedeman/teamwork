"""MCP server for code complexity analysis using lizard."""

from __future__ import annotations

import os
from pathlib import Path

import lizard
from mcp.server.fastmcp import FastMCP

server = FastMCP(
    "teamwork-mcp-complexity",
    instructions=(
        "Code complexity analysis server. Provides per-function cyclomatic "
        "complexity metrics (CCN) for 30+ languages using the lizard library."
    ),
)


def _analyze_file(file: str) -> lizard.FileInformation:
    """Analyze a single file and return the lizard result."""
    path = Path(file)
    if not path.is_file():
        raise FileNotFoundError(f"File not found: {file}")
    return lizard.analyze_file(str(path))


def _analyze_path(path: str) -> list[lizard.FileInformation]:
    """Analyze all supported files under *path* (file or directory)."""
    p = Path(path)
    if not p.exists():
        raise FileNotFoundError(f"Path not found: {path}")
    results: list[lizard.FileInformation] = []
    if p.is_file():
        results.append(lizard.analyze_file(str(p)))
    else:
        for r in lizard.analyze(paths=[str(p)], threads=4):
            results.append(r)
    return results


def _func_dict(func: object) -> dict:
    """Convert a lizard FunctionInfo into a serialisable dict."""
    return {
        "name": func.name,
        "line": func.start_line,
        "ccn": func.cyclomatic_complexity,
        "length": func.nloc,
        "parameters": len(func.parameters),
        "token_count": func.token_count,
    }


# ── Tools ────────────────────────────────────────────────────────────────────


@server.tool()
async def get_file_complexity(file: str) -> dict:
    """Get per-function complexity metrics for a single file.

    Returns file path, average cyclomatic complexity, total number of
    functions, and a list of per-function metrics.
    """
    result = _analyze_file(file)
    functions = [_func_dict(f) for f in result.function_list]
    avg_ccn = (
        sum(f["ccn"] for f in functions) / len(functions)
        if functions
        else 0.0
    )
    return {
        "file": file,
        "avg_ccn": round(avg_ccn, 2),
        "total_functions": len(functions),
        "functions": functions,
    }


@server.tool()
async def get_high_complexity_functions(
    path: str,
    threshold: int = 10,
) -> list[dict]:
    """Get all functions exceeding a cyclomatic-complexity threshold.

    Scans every supported file under *path* and returns matches sorted
    by CCN descending.
    """
    results = _analyze_path(path)
    hits: list[dict] = []
    for r in results:
        for f in r.function_list:
            if f.cyclomatic_complexity > threshold:
                hits.append(
                    {
                        "file": r.filename,
                        "function": f.name,
                        "line": f.start_line,
                        "ccn": f.cyclomatic_complexity,
                        "length": f.nloc,
                    }
                )
    hits.sort(key=lambda h: h["ccn"], reverse=True)
    return hits


@server.tool()
async def get_project_complexity_report(
    path: str,
    threshold: int = 10,
) -> dict:
    """Project-wide complexity summary.

    Returns aggregate statistics, the top-10 most complex functions,
    and per-file average CCN.
    """
    results = _analyze_path(path)

    all_functions: list[dict] = []
    file_stats: list[dict] = []

    for r in results:
        funcs = [_func_dict(f) for f in r.function_list]
        all_functions.extend(
            [{**fd, "file": r.filename} for fd in funcs]
        )
        if funcs:
            file_avg = round(
                sum(f["ccn"] for f in funcs) / len(funcs), 2
            )
            file_stats.append({"file": r.filename, "avg_ccn": file_avg})

    total = len(all_functions)
    exceeding = [f for f in all_functions if f["ccn"] > threshold]
    avg_ccn = (
        round(sum(f["ccn"] for f in all_functions) / total, 2)
        if total
        else 0.0
    )
    pct = round(len(exceeding) / total * 100, 2) if total else 0.0

    top_10 = sorted(all_functions, key=lambda f: f["ccn"], reverse=True)[
        :10
    ]

    file_stats.sort(key=lambda s: s["avg_ccn"], reverse=True)

    return {
        "avg_ccn": avg_ccn,
        "total_functions": total,
        "functions_exceeding_threshold": len(exceeding),
        "pct_exceeding": pct,
        "top_10_most_complex": [
            {
                "file": f["file"],
                "function": f["name"],
                "line": f["line"],
                "ccn": f["ccn"],
            }
            for f in top_10
        ],
        "files_by_avg_ccn": file_stats,
    }


@server.tool()
async def get_function_complexity(file: str, function: str) -> dict:
    """Detailed complexity breakdown for a specific function.

    Looks up *function* by name inside *file* and returns its metrics.
    Raises an error when the function is not found.
    """
    result = _analyze_file(file)
    for f in result.function_list:
        if f.name == function:
            return _func_dict(f)
    raise ValueError(
        f"Function '{function}' not found in {file}. "
        f"Available: {[f.name for f in result.function_list]}"
    )


@server.tool()
async def compare_complexity(
    base_path: str,
    head_path: str,
    threshold: int = 10,
) -> dict:
    """Compare complexity between two directory trees.

    Matches functions by (relative-file, function-name) and reports
    regressions, improvements, and newly introduced high-complexity
    functions.
    """
    base_results = _analyze_path(base_path)
    head_results = _analyze_path(head_path)

    def _build_index(
        results: list[lizard.FileInformation], root: str
    ) -> dict[tuple[str, str], dict]:
        idx: dict[tuple[str, str], dict] = {}
        for r in results:
            rel = os.path.relpath(r.filename, root)
            for f in r.function_list:
                idx[(rel, f.name)] = {
                    "file": rel,
                    "function": f.name,
                    "line": f.start_line,
                    "ccn": f.cyclomatic_complexity,
                }
        return idx

    base_idx = _build_index(base_results, base_path)
    head_idx = _build_index(head_results, head_path)

    regressions: list[dict] = []
    improvements: list[dict] = []
    new_high: list[dict] = []

    # Functions present in both
    for key in base_idx.keys() & head_idx.keys():
        b = base_idx[key]
        h = head_idx[key]
        delta = h["ccn"] - b["ccn"]
        if delta > 0:
            regressions.append(
                {
                    "file": h["file"],
                    "function": h["function"],
                    "base_ccn": b["ccn"],
                    "head_ccn": h["ccn"],
                    "delta": delta,
                }
            )
        elif delta < 0:
            improvements.append(
                {
                    "file": h["file"],
                    "function": h["function"],
                    "base_ccn": b["ccn"],
                    "head_ccn": h["ccn"],
                    "delta": delta,
                }
            )

    # New functions only in head
    for key in head_idx.keys() - base_idx.keys():
        h = head_idx[key]
        if h["ccn"] > threshold:
            new_high.append(h)

    regressions.sort(key=lambda r: r["delta"], reverse=True)
    improvements.sort(key=lambda i: i["delta"])
    new_high.sort(key=lambda n: n["ccn"], reverse=True)

    return {
        "regressions": regressions,
        "improvements": improvements,
        "new_high_complexity": new_high,
    }


# ── Entrypoint ───────────────────────────────────────────────────────────────


def main() -> None:
    """Run the MCP server over stdio."""
    server.run(transport="stdio")


if __name__ == "__main__":
    main()
