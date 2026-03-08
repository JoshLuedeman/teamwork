"""Tests for teamwork-mcp-coverage parsing and query functions."""

from __future__ import annotations

import json
import os
from pathlib import Path

import pytest

from teamwork_mcp_coverage.server import (
    _branch_coverage,
    _branch_pct,
    _coverage_data,
    _line_coverage,
    _line_pct,
    _uncovered_functions,
    parse_go_cover,
    parse_istanbul,
    parse_lcov,
)

FIXTURES = Path(__file__).parent / "fixtures"


# ---- Helpers ---------------------------------------------------------------


@pytest.fixture(autouse=True)
def _clear_state():
    """Reset module-level coverage state between tests."""
    _coverage_data.clear()
    yield
    _coverage_data.clear()


# ---- lcov parsing ----------------------------------------------------------


class TestParseLcov:
    def test_parses_files(self):
        text = (FIXTURES / "sample.lcov").read_text()
        result = parse_lcov(text)
        assert set(result.keys()) == {"src/utils.js", "src/main.js"}

    def test_line_counts(self):
        text = (FIXTURES / "sample.lcov").read_text()
        result = parse_lcov(text)

        utils = result["src/utils.js"]
        total, covered = _line_coverage(utils)
        assert total == 9
        assert covered == 6  # DA lines 1-3 and 9-11 are covered; 5-7 are not

    def test_branch_counts(self):
        text = (FIXTURES / "sample.lcov").read_text()
        result = parse_lcov(text)

        utils = result["src/utils.js"]
        total, covered = _branch_coverage(utils)
        assert total == 6
        assert covered == 4  # BRDA counts > 0

    def test_function_data(self):
        text = (FIXTURES / "sample.lcov").read_text()
        result = parse_lcov(text)

        utils = result["src/utils.js"]
        assert "add" in utils["functions"]
        assert utils["functions"]["add"]["count"] == 10
        assert utils["functions"]["subtract"]["count"] == 0

    def test_uncovered_functions(self):
        text = (FIXTURES / "sample.lcov").read_text()
        result = parse_lcov(text)

        utils = result["src/utils.js"]
        uncovered = _uncovered_functions(utils, "src/utils.js")
        names = [f["name"] for f in uncovered]
        assert "subtract" in names
        assert "add" not in names

    def test_line_pct(self):
        text = (FIXTURES / "sample.lcov").read_text()
        result = parse_lcov(text)
        pct = _line_pct(result["src/utils.js"])
        assert abs(pct - 66.67) < 0.1

    def test_branch_pct(self):
        text = (FIXTURES / "sample.lcov").read_text()
        result = parse_lcov(text)
        pct = _branch_pct(result["src/utils.js"])
        assert abs(pct - 66.67) < 0.1

    def test_main_full_coverage(self):
        text = (FIXTURES / "sample.lcov").read_text()
        result = parse_lcov(text)
        main = result["src/main.js"]
        total, covered = _line_coverage(main)
        assert total == 8
        assert covered == 8  # All lines covered


# ---- Istanbul JSON parsing -------------------------------------------------


class TestParseIstanbul:
    def test_parses_files(self):
        text = (FIXTURES / "coverage-summary.json").read_text()
        result = parse_istanbul(text)
        assert "total" in result
        assert "src/utils.js" in result
        assert "src/main.js" in result
        assert "src/helpers.js" in result

    def test_total_line_counts(self):
        text = (FIXTURES / "coverage-summary.json").read_text()
        result = parse_istanbul(text)

        total_entry = result["total"]
        total, covered = _line_coverage(total_entry)
        assert total == 200
        assert covered == 160

    def test_line_percentage(self):
        text = (FIXTURES / "coverage-summary.json").read_text()
        result = parse_istanbul(text)

        pct = _line_pct(result["total"])
        assert pct == 80.0

    def test_branch_counts(self):
        text = (FIXTURES / "coverage-summary.json").read_text()
        result = parse_istanbul(text)

        total_entry = result["total"]
        bt, bc = _branch_coverage(total_entry)
        assert bt == 50
        assert bc == 35

    def test_per_file_coverage(self):
        text = (FIXTURES / "coverage-summary.json").read_text()
        result = parse_istanbul(text)

        utils = result["src/utils.js"]
        total, covered = _line_coverage(utils)
        assert total == 100
        assert covered == 70

    def test_file_line_pct(self):
        text = (FIXTURES / "coverage-summary.json").read_text()
        result = parse_istanbul(text)
        pct = _line_pct(result["src/utils.js"])
        assert pct == 70.0


# ---- Go cover.out parsing --------------------------------------------------


class TestParseGoCover:
    def test_parses_files(self):
        text = (FIXTURES / "cover.out").read_text()
        result = parse_go_cover(text)
        assert "github.com/example/pkg/math.go" in result
        assert "github.com/example/pkg/utils.go" in result

    def test_math_coverage(self):
        text = (FIXTURES / "cover.out").read_text()
        result = parse_go_cover(text)

        math = result["github.com/example/pkg/math.go"]
        total, covered = _line_coverage(math)
        # Lines 5-7, 9-11, 13-15, 16-17, 18 -> 12 unique lines
        # Line 15 overlaps two blocks; max(1,0)=1 so it counts as covered
        # Lines 16-17 (count=0), rest covered -> 10 covered
        assert total == 12
        assert covered == 10

    def test_utils_coverage(self):
        text = (FIXTURES / "cover.out").read_text()
        result = parse_go_cover(text)

        utils = result["github.com/example/pkg/utils.go"]
        total, covered = _line_coverage(utils)
        # Lines 5-7 (count=1), 9-11 (count=0), 13-15 (count=0)
        assert total == 9
        assert covered == 3

    def test_utils_line_pct(self):
        text = (FIXTURES / "cover.out").read_text()
        result = parse_go_cover(text)
        pct = _line_pct(result["github.com/example/pkg/utils.go"])
        assert abs(pct - 33.33) < 0.1

    def test_skips_mode_line(self):
        text = "mode: set\n"
        result = parse_go_cover(text)
        assert result == {}

    def test_handles_empty(self):
        result = parse_go_cover("")
        assert result == {}


# ---- get_uncovered_files logic ---------------------------------------------


class TestGetUncoveredFiles:
    def test_threshold_filtering(self):
        text = (FIXTURES / "sample.lcov").read_text()
        parsed = parse_lcov(text)
        _coverage_data.update(parsed)

        # utils.js has ~66.7% coverage, main.js has 100%
        from teamwork_mcp_coverage.server import get_uncovered_files

        import asyncio

        result = asyncio.run(
            get_uncovered_files(threshold=80.0)
        )
        files = [r["file"] for r in result]
        assert "src/utils.js" in files
        assert "src/main.js" not in files

    def test_sorted_ascending(self):
        # Load Istanbul data which has multiple files below threshold
        text = (FIXTURES / "coverage-summary.json").read_text()
        parsed = parse_istanbul(text)
        _coverage_data.update(parsed)

        from teamwork_mcp_coverage.server import get_uncovered_files

        import asyncio

        result = asyncio.run(
            get_uncovered_files(threshold=85.0)
        )
        pcts = [r["line_pct"] for r in result]
        assert pcts == sorted(pcts)

    def test_empty_when_all_above(self):
        text = (FIXTURES / "sample.lcov").read_text()
        parsed = parse_lcov(text)
        _coverage_data.update(parsed)

        from teamwork_mcp_coverage.server import get_uncovered_files

        import asyncio

        result = asyncio.run(
            get_uncovered_files(threshold=0.0)
        )
        assert result == []


# ---- get_coverage_summary logic --------------------------------------------


class TestGetCoverageSummary:
    def test_overall_summary(self):
        text = (FIXTURES / "sample.lcov").read_text()
        parsed = parse_lcov(text)
        _coverage_data.update(parsed)

        from teamwork_mcp_coverage.server import get_coverage_summary

        import asyncio

        result = asyncio.run(
            get_coverage_summary()
        )
        assert "line_pct" in result
        assert "branch_pct" in result
        assert "total_lines" in result
        assert "covered_lines" in result
        assert "uncovered_functions" in result
        assert result["total_lines"] == 17  # 9 + 8
        assert result["covered_lines"] == 14  # 6 + 8

    def test_per_file_summary(self):
        text = (FIXTURES / "sample.lcov").read_text()
        parsed = parse_lcov(text)
        _coverage_data.update(parsed)

        from teamwork_mcp_coverage.server import get_coverage_summary

        import asyncio

        result = asyncio.run(
            get_coverage_summary(file="src/utils.js")
        )
        assert result["total_lines"] == 9
        assert result["covered_lines"] == 6

    def test_missing_file_raises(self):
        text = (FIXTURES / "sample.lcov").read_text()
        parsed = parse_lcov(text)
        _coverage_data.update(parsed)

        from teamwork_mcp_coverage.server import get_coverage_summary

        import asyncio

        with pytest.raises(KeyError):
            asyncio.run(
                get_coverage_summary(file="nonexistent.js")
            )

    def test_no_data_raises(self):
        from teamwork_mcp_coverage.server import get_coverage_summary

        import asyncio

        with pytest.raises(RuntimeError):
            asyncio.run(
                get_coverage_summary()
            )


# ---- Error handling --------------------------------------------------------


class TestErrorHandling:
    def test_invalid_format(self):
        from teamwork_mcp_coverage.server import load_coverage_report

        import asyncio

        with pytest.raises(ValueError, match="Unsupported format"):
            asyncio.run(
                load_coverage_report(
                    path=str(FIXTURES / "sample.lcov"),
                    format="xml",
                )
            )

    def test_missing_file(self):
        from teamwork_mcp_coverage.server import load_coverage_report

        import asyncio

        with pytest.raises(FileNotFoundError):
            asyncio.run(
                load_coverage_report(
                    path="/nonexistent/coverage.lcov",
                    format="lcov",
                )
            )

    def test_load_lcov_integration(self):
        """Verify load_coverage_report populates module state."""
        from teamwork_mcp_coverage.server import load_coverage_report

        import asyncio

        result = asyncio.run(
            load_coverage_report(
                path=str(FIXTURES / "sample.lcov"),
                format="lcov",
            )
        )
        assert result["files_loaded"] == 2
        assert result["total_lines"] == 17
        assert result["covered_lines"] == 14
        assert len(_coverage_data) == 2
