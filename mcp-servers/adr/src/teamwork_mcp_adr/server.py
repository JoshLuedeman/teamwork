"""MCP server for Architecture Decision Record management.

Provides tools for listing, searching, creating, and managing ADRs
stored as Markdown Architectural Decision Records (MADR) files.
"""

from __future__ import annotations

import os
import re
from datetime import date
from pathlib import Path

import frontmatter
from mcp.server.fastmcp import FastMCP

server = FastMCP("teamwork-mcp-adr")

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# Pattern for ADR filenames: NNN-slug.md
_ADR_PATTERN = re.compile(r"^(\d{3})-.+\.md$")

# Comment prefixes by file extension
_COMMENT_STYLES: dict[str, str] = {
    ".go": "//",
    ".js": "//",
    ".ts": "//",
    ".tsx": "//",
    ".jsx": "//",
    ".java": "//",
    ".c": "//",
    ".cpp": "//",
    ".cs": "//",
    ".rs": "//",
    ".swift": "//",
    ".kt": "//",
    ".py": "#",
    ".rb": "#",
    ".sh": "#",
    ".bash": "#",
    ".yaml": "#",
    ".yml": "#",
    ".toml": "#",
    ".r": "#",
    ".pl": "#",
    ".css": "/*",  # needs closing */
    ".html": "<!--",  # needs closing -->
}

_COMMENT_CLOSERS: dict[str, str] = {
    ".css": " */",
    ".html": " -->",
}


def _resolve_dir(decisions_dir: str) -> Path:
    """Resolve the decisions directory to an absolute path."""
    return Path(decisions_dir).resolve()


def _parse_adr(path: Path) -> dict | None:
    """Parse a single ADR markdown file, returning structured data.

    Supports YAML frontmatter via python-frontmatter. Falls back to
    extracting sections from the markdown body.
    """
    try:
        post = frontmatter.load(str(path))
    except Exception:
        return None

    match = _ADR_PATTERN.match(path.name)
    if not match:
        return None

    file_id = match.group(1)
    body: str = post.content

    # Frontmatter values take priority; fall back to body parsing.
    adr_id = str(post.get("id", file_id))
    title = post.get("title", "")
    status = post.get("status", "")
    adr_date = str(post.get("date", ""))

    # Extract sections from body
    sections = _extract_sections(body)

    if not title:
        title = sections.get("title", path.stem)
    if not status:
        status = sections.get("status", "unknown")

    return {
        "id": adr_id,
        "title": title,
        "status": status.lower().strip(),
        "date": adr_date,
        "file": str(path),
        "context": sections.get("context", ""),
        "decision": sections.get("decision", ""),
        "consequences": sections.get("consequences", ""),
        "content": body,
    }


def _extract_sections(body: str) -> dict[str, str]:
    """Extract named ## sections from the markdown body."""
    sections: dict[str, str] = {}

    # Grab the H1 title if present
    h1 = re.search(r"^#\s+(.+)$", body, re.MULTILINE)
    if h1:
        sections["title"] = h1.group(1).strip()

    # Split on ## headings
    parts = re.split(r"^##\s+", body, flags=re.MULTILINE)
    for part in parts[1:]:  # skip text before first ##
        lines = part.split("\n", 1)
        heading = lines[0].strip().lower()
        content = lines[1].strip() if len(lines) > 1 else ""
        sections[heading] = content

    return sections


def _list_adr_files(decisions_dir: Path) -> list[Path]:
    """Return sorted list of ADR files in the directory."""
    if not decisions_dir.is_dir():
        return []
    files = [f for f in decisions_dir.iterdir() if _ADR_PATTERN.match(f.name)]
    return sorted(files)


def _next_id(decisions_dir: Path) -> str:
    """Compute the next auto-incremented ADR ID (zero-padded to 3 digits)."""
    files = _list_adr_files(decisions_dir)
    max_id = 0
    for f in files:
        m = _ADR_PATTERN.match(f.name)
        if m:
            max_id = max(max_id, int(m.group(1)))
    return f"{max_id + 1:03d}"


def _slugify(title: str) -> str:
    """Convert a title to a filename-safe slug."""
    slug = title.lower().strip()
    slug = re.sub(r"[^a-z0-9]+", "-", slug)
    slug = slug.strip("-")
    return slug


# ---------------------------------------------------------------------------
# MCP Tools
# ---------------------------------------------------------------------------


@server.tool()
async def list_adrs(
    status: str | None = None,
    decisions_dir: str = "docs/decisions",
) -> list[dict]:
    """List all ADRs, optionally filtered by status.

    Args:
        status: Filter by status (draft/accepted/superseded/deprecated).
                If None, returns all ADRs.
        decisions_dir: Path to the decisions directory.

    Returns:
        List of dicts with keys: id, title, status, date, file.
    """
    dirpath = _resolve_dir(decisions_dir)
    results: list[dict] = []
    for path in _list_adr_files(dirpath):
        adr = _parse_adr(path)
        if adr is None:
            continue
        if status and adr["status"] != status.lower().strip():
            continue
        results.append({
            "id": adr["id"],
            "title": adr["title"],
            "status": adr["status"],
            "date": adr["date"],
            "file": adr["file"],
        })
    return results


@server.tool()
async def get_adr(
    id: str,
    decisions_dir: str = "docs/decisions",
) -> dict:
    """Get full parsed content of a specific ADR.

    Args:
        id: The ADR identifier (e.g. "001").
        decisions_dir: Path to the decisions directory.

    Returns:
        Dict with keys: id, title, status, date, context, decision,
        consequences, content.

    Raises:
        ValueError: If the ADR is not found.
    """
    dirpath = _resolve_dir(decisions_dir)
    padded = id.zfill(3)
    for path in _list_adr_files(dirpath):
        m = _ADR_PATTERN.match(path.name)
        if m and m.group(1) == padded:
            adr = _parse_adr(path)
            if adr:
                return adr
    raise ValueError(f"ADR '{id}' not found in {decisions_dir}")


@server.tool()
async def search_adrs(
    query: str,
    decisions_dir: str = "docs/decisions",
) -> list[dict]:
    """Full-text search across all ADR content.

    Performs case-insensitive substring matching against the full body
    of every ADR. Results are scored by number of occurrences and
    include an excerpt around the first match.

    Args:
        query: Search string.
        decisions_dir: Path to the decisions directory.

    Returns:
        List of matching ADRs sorted by relevance (descending).
        Each dict has: id, title, status, date, excerpt, relevance.
    """
    dirpath = _resolve_dir(decisions_dir)
    pattern = re.compile(re.escape(query), re.IGNORECASE)
    results: list[dict] = []

    for path in _list_adr_files(dirpath):
        adr = _parse_adr(path)
        if adr is None:
            continue

        content = adr["content"]
        matches = list(pattern.finditer(content))
        if not matches:
            continue

        # Build excerpt around first match (±100 chars)
        m = matches[0]
        start = max(0, m.start() - 100)
        end = min(len(content), m.end() + 100)
        excerpt = content[start:end]
        if start > 0:
            excerpt = "..." + excerpt
        if end < len(content):
            excerpt = excerpt + "..."

        results.append({
            "id": adr["id"],
            "title": adr["title"],
            "status": adr["status"],
            "date": adr["date"],
            "excerpt": excerpt,
            "relevance": len(matches),
        })

    results.sort(key=lambda r: r["relevance"], reverse=True)
    return results


@server.tool()
async def create_adr(
    title: str,
    context: str,
    decision: str,
    consequences: str,
    status: str = "accepted",
    decisions_dir: str = "docs/decisions",
) -> dict:
    """Create a new ADR file in MADR format with auto-incremented ID.

    Args:
        title: Short title for the decision.
        context: Why this decision is needed.
        decision: What was decided.
        consequences: What happens as a result.
        status: Initial status (default: "accepted").
        decisions_dir: Path to the decisions directory.

    Returns:
        Dict with keys: id, file, content.
    """
    dirpath = _resolve_dir(decisions_dir)
    dirpath.mkdir(parents=True, exist_ok=True)

    adr_id = _next_id(dirpath)
    slug = _slugify(title)
    filename = f"{adr_id}-{slug}.md"
    filepath = dirpath / filename

    today = date.today().isoformat()
    content = (
        f"---\n"
        f'id: "{adr_id}"\n'
        f'title: "{title}"\n'
        f'status: "{status.lower()}"\n'
        f'date: "{today}"\n'
        f"---\n"
        f"\n"
        f"# ADR-{adr_id}: {title}\n"
        f"\n"
        f"## Status\n"
        f"{status.capitalize()}\n"
        f"\n"
        f"## Context\n"
        f"{context}\n"
        f"\n"
        f"## Decision\n"
        f"{decision}\n"
        f"\n"
        f"## Consequences\n"
        f"{consequences}\n"
    )

    filepath.write_text(content)

    return {
        "id": adr_id,
        "file": str(filepath),
        "content": content,
    }


@server.tool()
async def update_adr_status(
    id: str,
    status: str,
    superseded_by: str | None = None,
    decisions_dir: str = "docs/decisions",
) -> dict:
    """Update the status field in an existing ADR.

    Updates both the YAML frontmatter and the ## Status section in the
    markdown body. If superseded_by is provided, adds a note.

    Args:
        id: The ADR identifier (e.g. "001").
        status: New status value.
        superseded_by: Optional ID of the superseding ADR.
        decisions_dir: Path to the decisions directory.

    Returns:
        Dict with keys: file, updated (bool).

    Raises:
        ValueError: If the ADR is not found.
    """
    dirpath = _resolve_dir(decisions_dir)
    padded = id.zfill(3)

    for path in _list_adr_files(dirpath):
        m = _ADR_PATTERN.match(path.name)
        if not (m and m.group(1) == padded):
            continue

        post = frontmatter.load(str(path))
        post["status"] = status.lower()

        # Update ## Status section in body
        status_text = status.capitalize()
        if superseded_by:
            sup_id = superseded_by.zfill(3)
            status_text += f"\n\nSuperseded by [ADR-{sup_id}](./{sup_id}-*.md)"

        body = post.content
        body = re.sub(
            r"(## Status\n).*?(?=\n## |\Z)",
            rf"\g<1>{status_text}\n",
            body,
            count=1,
            flags=re.DOTALL,
        )
        post.content = body

        path.write_text(frontmatter.dumps(post))
        return {"file": str(path), "updated": True}

    raise ValueError(f"ADR '{id}' not found in {decisions_dir}")


@server.tool()
async def link_adr_to_code(
    adr_id: str,
    file: str,
    line: int | None = None,
    reason: str = "",
) -> dict:
    """Add a code comment linking a source file to an ADR.

    Inserts a comment like ``// ADR-004: reason`` at the specified line
    (or at the top of the file if no line is given).

    Args:
        adr_id: The ADR identifier (e.g. "004").
        file: Path to the source file to annotate.
        line: Line number (1-based) to insert the comment before.
              If None, inserts at the top of the file.
        reason: Optional description of why this code relates to the ADR.

    Returns:
        Dict with key comment_added (bool).
    """
    filepath = Path(file).resolve()
    if not filepath.is_file():
        return {"comment_added": False}

    ext = filepath.suffix.lower()
    prefix = _COMMENT_STYLES.get(ext, "//")
    closer = _COMMENT_CLOSERS.get(ext, "")

    padded = adr_id.zfill(3)
    comment_text = f"ADR-{padded}"
    if reason:
        comment_text += f": {reason}"

    comment_line = f"{prefix} {comment_text}{closer}\n"

    lines = filepath.read_text().splitlines(keepends=True)

    if line is not None:
        idx = max(0, line - 1)  # convert to 0-based
        lines.insert(idx, comment_line)
    else:
        lines.insert(0, comment_line)

    filepath.write_text("".join(lines))
    return {"comment_added": True}


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------


def main() -> None:
    """Run the MCP server using stdio transport."""
    server.run()


if __name__ == "__main__":
    main()
