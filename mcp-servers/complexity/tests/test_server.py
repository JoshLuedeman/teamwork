"""Tests for teamwork-mcp-complexity MCP server."""

from __future__ import annotations

import asyncio
import os
import shutil
from pathlib import Path

import pytest

from teamwork_mcp_complexity.server import (
    get_file_complexity,
    get_function_complexity,
    get_high_complexity_functions,
    get_project_complexity_report,
    compare_complexity,
)

FIXTURES = str(Path(__file__).parent / "fixtures")


def _run(coro):
    """Run an async tool function synchronously."""
    return asyncio.run(coro)


# -- get_file_complexity -----------------------------------------------------


class TestGetFileComplexity:
    def test_simple_python(self):
        result = _run(get_file_complexity(os.path.join(FIXTURES, "simple.py")))
        assert result["total_functions"] == 2
        assert result["avg_ccn"] == 1.0
        names = [f["name"] for f in result["functions"]]
        assert "hello" in names
        assert "add" in names
        for func in result["functions"]:
            assert func["ccn"] == 1

    def test_complex_go(self):
        result = _run(get_file_complexity(os.path.join(FIXTURES, "complex.go")))
        assert result["total_functions"] == 1
        funcs = result["functions"]
        assert funcs[0]["name"] == "complexFunction"
        assert funcs[0]["ccn"] >= 10

    def test_nonexistent_file(self):
        with pytest.raises(FileNotFoundError):
            _run(get_file_complexity("/nonexistent/file.py"))


# -- get_high_complexity_functions -------------------------------------------


class TestGetHighComplexityFunctions:
    def test_threshold_5(self):
        result = _run(get_high_complexity_functions(FIXTURES, threshold=5))
        functions = [r["function"] for r in result]
        assert "complexFunction" in functions
        assert "processData" in functions
        # simple functions should NOT appear
        assert "hello" not in functions
        assert "add" not in functions

    def test_threshold_20_empty(self):
        result = _run(get_high_complexity_functions(FIXTURES, threshold=20))
        assert result == []

    def test_sorted_descending(self):
        result = _run(get_high_complexity_functions(FIXTURES, threshold=1))
        ccns = [r["ccn"] for r in result]
        assert ccns == sorted(ccns, reverse=True)


# -- get_project_complexity_report -------------------------------------------


class TestGetProjectComplexityReport:
    def test_fixture_totals(self):
        report = _run(get_project_complexity_report(FIXTURES, threshold=10))
        # 5 total functions: 2 in simple.py, 1 in complex.go, 2 in mixed.js
        assert report["total_functions"] == 5
        # Only complexFunction (ccn=11) exceeds threshold 10
        assert report["functions_exceeding_threshold"] == 1
        assert report["pct_exceeding"] == pytest.approx(20.0)
        assert report["avg_ccn"] > 0

    def test_top_10(self):
        report = _run(get_project_complexity_report(FIXTURES))
        top = report["top_10_most_complex"]
        assert len(top) <= 10
        assert top[0]["function"] == "complexFunction"

    def test_files_by_avg_ccn(self):
        report = _run(get_project_complexity_report(FIXTURES))
        files = report["files_by_avg_ccn"]
        assert len(files) == 3
        # complex.go has the highest average CCN
        assert "complex.go" in files[0]["file"]


# -- get_function_complexity -------------------------------------------------


class TestGetFunctionComplexity:
    def test_known_function(self):
        result = _run(
            get_function_complexity(
                os.path.join(FIXTURES, "simple.py"), "hello"
            )
        )
        assert result["name"] == "hello"
        assert result["ccn"] == 1
        assert result["parameters"] == 1
        assert result["line"] == 1

    def test_nonexistent_function(self):
        with pytest.raises(ValueError, match="not found"):
            _run(
                get_function_complexity(
                    os.path.join(FIXTURES, "simple.py"), "nonexistent"
                )
            )


# -- compare_complexity ------------------------------------------------------


class TestCompareComplexity:
    def test_regression_detected(self, tmp_path: Path):
        """A function whose CCN increases should appear in regressions."""
        base_dir = tmp_path / "base"
        head_dir = tmp_path / "head"
        base_dir.mkdir()
        head_dir.mkdir()

        # base: simple function
        (base_dir / "code.py").write_text(
            "def foo(x):\n    return x\n"
        )
        # head: more complex version
        (head_dir / "code.py").write_text(
            "def foo(x):\n"
            "    if x > 0:\n"
            "        if x > 10:\n"
            "            return x * 2\n"
            "        else:\n"
            "            return x + 1\n"
            "    elif x == 0:\n"
            "        return 0\n"
            "    else:\n"
            "        return -x\n"
        )

        result = _run(compare_complexity(str(base_dir), str(head_dir)))
        assert len(result["regressions"]) == 1
        reg = result["regressions"][0]
        assert reg["function"] == "foo"
        assert reg["delta"] > 0
        assert reg["head_ccn"] > reg["base_ccn"]

    def test_improvement_detected(self, tmp_path: Path):
        """A function whose CCN decreases should appear in improvements."""
        base_dir = tmp_path / "base"
        head_dir = tmp_path / "head"
        base_dir.mkdir()
        head_dir.mkdir()

        (base_dir / "code.py").write_text(
            "def bar(x):\n"
            "    if x > 0:\n"
            "        if x > 10:\n"
            "            return 1\n"
            "        else:\n"
            "            return 2\n"
            "    else:\n"
            "        return 3\n"
        )
        (head_dir / "code.py").write_text(
            "def bar(x):\n    return x\n"
        )

        result = _run(compare_complexity(str(base_dir), str(head_dir)))
        assert len(result["improvements"]) == 1
        imp = result["improvements"][0]
        assert imp["function"] == "bar"
        assert imp["delta"] < 0

    def test_new_high_complexity(self, tmp_path: Path):
        """A brand-new function with high CCN should appear in new_high_complexity."""
        base_dir = tmp_path / "base"
        head_dir = tmp_path / "head"
        base_dir.mkdir()
        head_dir.mkdir()

        (base_dir / "code.py").write_text(
            "def simple(x):\n    return x\n"
        )

        # Copy base and add a complex function
        shutil.copy(base_dir / "code.py", head_dir / "code.py")
        (head_dir / "complex.py").write_text(
            "def big(a, b, c):\n"
            "    if a > 0:\n"
            "        if b > 0:\n"
            "            if c > 0:\n"
            "                return 1\n"
            "            elif c == 0:\n"
            "                return 2\n"
            "            else:\n"
            "                return 3\n"
            "        elif b == 0:\n"
            "            return 4\n"
            "        else:\n"
            "            if c > 0:\n"
            "                return 5\n"
            "            elif c == 0:\n"
            "                return 6\n"
            "            else:\n"
            "                return 7\n"
            "    elif a == 0:\n"
            "        return 8\n"
            "    else:\n"
            "        if b > 0:\n"
            "            return 9\n"
            "        else:\n"
            "            return 10\n"
        )

        result = _run(
            compare_complexity(str(base_dir), str(head_dir), threshold=5)
        )
        new_funcs = [f["function"] for f in result["new_high_complexity"]]
        assert "big" in new_funcs

    def test_nonexistent_path(self):
        with pytest.raises(FileNotFoundError):
            _run(compare_complexity("/no/such/base", "/no/such/head"))
