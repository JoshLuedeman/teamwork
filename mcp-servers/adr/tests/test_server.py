"""Tests for the teamwork-mcp-adr MCP server."""

from __future__ import annotations

import shutil
from pathlib import Path

import pytest

from teamwork_mcp_adr.server import (
    create_adr,
    get_adr,
    link_adr_to_code,
    list_adrs,
    search_adrs,
    update_adr_status,
    _parse_adr,
    _next_id,
    _slugify,
)

FIXTURES_DIR = Path(__file__).parent / "fixtures" / "decisions"


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


@pytest.fixture()
def fixture_dir() -> Path:
    """Return the path to the test fixtures decisions directory."""
    return FIXTURES_DIR


@pytest.fixture()
def tmp_decisions(tmp_path: Path) -> Path:
    """Copy fixture ADRs into a temporary directory for mutation tests."""
    dest = tmp_path / "decisions"
    shutil.copytree(FIXTURES_DIR, dest)
    return dest


# ---------------------------------------------------------------------------
# list_adrs
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_list_adrs_returns_all(fixture_dir: Path) -> None:
    """list_adrs returns all ADRs from the fixture directory."""
    results = await list_adrs(decisions_dir=str(fixture_dir))
    assert len(results) == 3
    ids = {r["id"] for r in results}
    assert ids == {"001", "002", "003"}


@pytest.mark.asyncio
async def test_list_adrs_filter_accepted(fixture_dir: Path) -> None:
    """list_adrs with status='accepted' returns only accepted ADRs."""
    results = await list_adrs(status="accepted", decisions_dir=str(fixture_dir))
    assert len(results) == 2
    for r in results:
        assert r["status"] == "accepted"


@pytest.mark.asyncio
async def test_list_adrs_filter_draft(fixture_dir: Path) -> None:
    """list_adrs with status='draft' returns only draft ADRs."""
    results = await list_adrs(status="draft", decisions_dir=str(fixture_dir))
    assert len(results) == 1
    assert results[0]["id"] == "003"


@pytest.mark.asyncio
async def test_list_adrs_empty_dir(tmp_path: Path) -> None:
    """list_adrs returns empty list for a directory with no ADRs."""
    results = await list_adrs(decisions_dir=str(tmp_path))
    assert results == []


# ---------------------------------------------------------------------------
# get_adr
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_get_adr_returns_parsed_content(fixture_dir: Path) -> None:
    """get_adr returns parsed content with all sections."""
    adr = await get_adr(id="001", decisions_dir=str(fixture_dir))
    assert adr["id"] == "001"
    assert adr["title"] == "Use React for frontend"
    assert adr["status"] == "accepted"
    assert adr["date"] == "2024-01-15"
    assert "frontend framework" in adr["context"]
    assert "React 18" in adr["decision"]
    assert "React expertise" in adr["consequences"]
    assert adr["content"]  # non-empty


@pytest.mark.asyncio
async def test_get_adr_not_found(fixture_dir: Path) -> None:
    """get_adr raises ValueError for a nonexistent ID."""
    with pytest.raises(ValueError, match="not found"):
        await get_adr(id="999", decisions_dir=str(fixture_dir))


@pytest.mark.asyncio
async def test_get_adr_unpadded_id(fixture_dir: Path) -> None:
    """get_adr accepts unpadded IDs like '1' and pads them."""
    adr = await get_adr(id="1", decisions_dir=str(fixture_dir))
    assert adr["id"] == "001"


# ---------------------------------------------------------------------------
# search_adrs
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_search_adrs_finds_react(fixture_dir: Path) -> None:
    """search_adrs('React') finds ADR-001."""
    results = await search_adrs(query="React", decisions_dir=str(fixture_dir))
    assert len(results) >= 1
    assert results[0]["id"] == "001"
    assert results[0]["excerpt"]


@pytest.mark.asyncio
async def test_search_adrs_finds_authentication(fixture_dir: Path) -> None:
    """search_adrs('authentication') finds ADR-002."""
    results = await search_adrs(query="authentication", decisions_dir=str(fixture_dir))
    ids = {r["id"] for r in results}
    assert "002" in ids


@pytest.mark.asyncio
async def test_search_adrs_case_insensitive(fixture_dir: Path) -> None:
    """search_adrs is case-insensitive."""
    upper = await search_adrs(query="REACT", decisions_dir=str(fixture_dir))
    lower = await search_adrs(query="react", decisions_dir=str(fixture_dir))
    assert len(upper) == len(lower)


@pytest.mark.asyncio
async def test_search_adrs_no_results(fixture_dir: Path) -> None:
    """search_adrs returns empty list when nothing matches."""
    results = await search_adrs(query="xyznonexistent", decisions_dir=str(fixture_dir))
    assert results == []


# ---------------------------------------------------------------------------
# create_adr
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_create_adr_auto_increments(tmp_decisions: Path) -> None:
    """create_adr generates the next ID (004) after existing 001-003."""
    result = await create_adr(
        title="Use Redis for caching",
        context="Need a fast caching layer.",
        decision="Use Redis 7.",
        consequences="Team needs Redis knowledge.",
        decisions_dir=str(tmp_decisions),
    )
    assert result["id"] == "004"
    assert Path(result["file"]).exists()
    assert "004-use-redis-for-caching.md" in result["file"]


@pytest.mark.asyncio
async def test_create_adr_valid_madr_format(tmp_decisions: Path) -> None:
    """create_adr produces valid MADR format with frontmatter and sections."""
    result = await create_adr(
        title="Use GraphQL",
        context="Need flexible API queries.",
        decision="Adopt GraphQL for the public API.",
        consequences="Higher complexity, but better client flexibility.",
        status="draft",
        decisions_dir=str(tmp_decisions),
    )
    content = result["content"]

    # Check frontmatter
    assert 'id: "004"' in content
    assert 'title: "Use GraphQL"' in content
    assert 'status: "draft"' in content

    # Check sections
    assert "## Status" in content
    assert "## Context" in content
    assert "## Decision" in content
    assert "## Consequences" in content
    assert "Draft" in content
    assert "flexible API queries" in content


@pytest.mark.asyncio
async def test_create_adr_empty_dir(tmp_path: Path) -> None:
    """create_adr works in an empty directory, starting at 001."""
    new_dir = tmp_path / "empty_decisions"
    result = await create_adr(
        title="First decision",
        context="Starting fresh.",
        decision="Do the thing.",
        consequences="Things happen.",
        decisions_dir=str(new_dir),
    )
    assert result["id"] == "001"
    assert Path(result["file"]).exists()


# ---------------------------------------------------------------------------
# update_adr_status
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_update_adr_status_changes_status(tmp_decisions: Path) -> None:
    """update_adr_status changes the status in the file."""
    result = await update_adr_status(
        id="003", status="accepted", decisions_dir=str(tmp_decisions)
    )
    assert result["updated"] is True

    # Verify the file was updated
    adr = await get_adr(id="003", decisions_dir=str(tmp_decisions))
    assert adr["status"] == "accepted"


@pytest.mark.asyncio
async def test_update_adr_status_with_superseded_by(tmp_decisions: Path) -> None:
    """update_adr_status with superseded_by adds a link."""
    result = await update_adr_status(
        id="001",
        status="superseded",
        superseded_by="004",
        decisions_dir=str(tmp_decisions),
    )
    assert result["updated"] is True

    # Read the raw file to check the superseded_by note
    filepath = Path(result["file"])
    raw = filepath.read_text()
    assert "superseded" in raw.lower()
    assert "ADR-004" in raw


@pytest.mark.asyncio
async def test_update_adr_status_not_found(tmp_decisions: Path) -> None:
    """update_adr_status raises ValueError for nonexistent ID."""
    with pytest.raises(ValueError, match="not found"):
        await update_adr_status(
            id="999", status="deprecated", decisions_dir=str(tmp_decisions)
        )


# ---------------------------------------------------------------------------
# link_adr_to_code
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_link_adr_to_code_go_file(tmp_path: Path) -> None:
    """link_adr_to_code inserts a // comment in a Go file."""
    go_file = tmp_path / "main.go"
    go_file.write_text("package main\n\nfunc main() {}\n")

    result = await link_adr_to_code(
        adr_id="001", file=str(go_file), line=1, reason="Uses React pattern"
    )
    assert result["comment_added"] is True

    content = go_file.read_text()
    assert "// ADR-001: Uses React pattern" in content
    # Original content preserved
    assert "package main" in content


@pytest.mark.asyncio
async def test_link_adr_to_code_python_file(tmp_path: Path) -> None:
    """link_adr_to_code inserts a # comment in a Python file."""
    py_file = tmp_path / "app.py"
    py_file.write_text("import os\n\ndef run():\n    pass\n")

    result = await link_adr_to_code(
        adr_id="002", file=str(py_file), reason="JWT auth implementation"
    )
    assert result["comment_added"] is True

    content = py_file.read_text()
    assert "# ADR-002: JWT auth implementation" in content


@pytest.mark.asyncio
async def test_link_adr_to_code_nonexistent_file(tmp_path: Path) -> None:
    """link_adr_to_code returns comment_added=False for missing files."""
    result = await link_adr_to_code(
        adr_id="001", file=str(tmp_path / "nope.go")
    )
    assert result["comment_added"] is False


# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------


def test_slugify() -> None:
    """_slugify produces clean filename slugs."""
    assert _slugify("Use React for frontend") == "use-react-for-frontend"
    assert _slugify("  Spaces & Symbols!  ") == "spaces-symbols"


def test_next_id(fixture_dir: Path) -> None:
    """_next_id returns 004 for fixtures containing 001-003."""
    assert _next_id(fixture_dir) == "004"


def test_next_id_empty(tmp_path: Path) -> None:
    """_next_id returns 001 for an empty directory."""
    assert _next_id(tmp_path) == "001"


def test_parse_adr(fixture_dir: Path) -> None:
    """_parse_adr extracts all fields from a fixture file."""
    path = fixture_dir / "001-use-react.md"
    adr = _parse_adr(path)
    assert adr is not None
    assert adr["id"] == "001"
    assert adr["title"] == "Use React for frontend"
    assert adr["status"] == "accepted"
