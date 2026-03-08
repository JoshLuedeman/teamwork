"""MCP server for changelog generation using git-cliff.

Provides tools to generate changelogs, preview release notes, list unreleased
commits, and suggest the next semantic version based on conventional commits.
"""

from __future__ import annotations

import asyncio
import json
import re
from typing import Any

from mcp.server.fastmcp import FastMCP

# ---------------------------------------------------------------------------
# Conventional-commit parsing
# ---------------------------------------------------------------------------

CONVENTIONAL_COMMIT_RE = re.compile(
    r"^(?P<type>\w+)(?:\((?P<scope>[^)]*)\))?(?P<breaking>!)?:\s*(?P<subject>.+)$"
)


def parse_conventional_commit(subject: str) -> dict[str, Any] | None:
    """Parse a conventional-commit subject line.

    Returns a dict with keys ``type``, ``scope``, ``subject``, and
    ``breaking`` (bool), or *None* if the subject doesn't match the
    conventional-commit format.
    """
    match = CONVENTIONAL_COMMIT_RE.match(subject.strip())
    if not match:
        return None
    return {
        "type": match.group("type"),
        "scope": match.group("scope") or None,
        "subject": match.group("subject"),
        "breaking": match.group("breaking") == "!",
    }


# ---------------------------------------------------------------------------
# Version helpers
# ---------------------------------------------------------------------------


def parse_version(version: str) -> tuple[int, int, int]:
    """Parse a semver string like ``v1.2.3`` or ``1.2.3`` into a 3-tuple."""
    v = version.lstrip("v")
    parts = v.split(".")
    if len(parts) != 3:
        raise ValueError(f"Invalid semver version: {version}")
    try:
        return int(parts[0]), int(parts[1]), int(parts[2])
    except ValueError as exc:
        raise ValueError(f"Invalid semver version: {version}") from exc


def format_version(major: int, minor: int, patch: int) -> str:
    """Format a semver 3-tuple back to a ``v``-prefixed string."""
    return f"v{major}.{minor}.{patch}"


def compute_next_version(
    current: str,
    commits: list[dict[str, Any]],
) -> dict[str, Any]:
    """Determine the next version and explain why.

    Returns ``{version, bump, reasoning}``.
    """
    major, minor, patch = parse_version(current)

    has_breaking = any(c.get("breaking") for c in commits)
    has_feat = any(c.get("type") == "feat" for c in commits)

    if has_breaking:
        bump = "major"
        next_v = format_version(major + 1, 0, 0)
        breaking_subjects = [
            c.get("subject", c.get("raw_subject", ""))
            for c in commits
            if c.get("breaking")
        ]
        reasoning = (
            f"BREAKING CHANGE detected in: {'; '.join(breaking_subjects)}. "
            f"Bumping major version from {current} to {next_v}."
        )
    elif has_feat:
        bump = "minor"
        next_v = format_version(major, minor + 1, 0)
        feat_subjects = [
            c.get("subject", c.get("raw_subject", ""))
            for c in commits
            if c.get("type") == "feat"
        ]
        reasoning = (
            f"New feature(s) detected: {'; '.join(feat_subjects)}. "
            f"Bumping minor version from {current} to {next_v}."
        )
    else:
        bump = "patch"
        next_v = format_version(major, minor, patch + 1)
        reasoning = (
            f"Only fix/chore/docs commits found. "
            f"Bumping patch version from {current} to {next_v}."
        )

    return {"version": next_v, "bump": bump, "reasoning": reasoning}


# ---------------------------------------------------------------------------
# Git / git-cliff subprocess helpers
# ---------------------------------------------------------------------------


async def _run(
    *args: str,
    check: bool = True,
) -> tuple[int, str, str]:
    """Run a subprocess and return ``(returncode, stdout, stderr)``."""
    proc = await asyncio.create_subprocess_exec(
        *args,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )
    stdout_bytes, stderr_bytes = await proc.communicate()
    stdout = stdout_bytes.decode(errors="replace")
    stderr = stderr_bytes.decode(errors="replace")

    if check and proc.returncode != 0:
        raise RuntimeError(
            f"Command {args!r} failed (rc={proc.returncode}): {stderr.strip()}"
        )

    return proc.returncode, stdout, stderr


async def _git_cliff_available() -> bool:
    """Return True if ``git-cliff`` is on the PATH."""
    try:
        rc, _, _ = await _run("git-cliff", "--version", check=False)
        return rc == 0
    except FileNotFoundError:
        return False


async def _in_git_repo() -> bool:
    """Return True if the cwd is inside a git repository."""
    try:
        rc, _, _ = await _run(
            "git", "rev-parse", "--is-inside-work-tree", check=False
        )
        return rc == 0
    except FileNotFoundError:
        return False


async def _ref_exists(ref: str) -> bool:
    """Return True if *ref* resolves to a valid git object."""
    rc, _, _ = await _run("git", "rev-parse", "--verify", ref, check=False)
    return rc == 0


async def _latest_tag() -> str | None:
    """Return the most recent tag reachable from HEAD, or None."""
    rc, stdout, _ = await _run(
        "git", "describe", "--tags", "--abbrev=0", check=False
    )
    if rc != 0:
        return None
    return stdout.strip() or None


GIT_CLIFF_INSTALL_MSG = (
    "git-cliff is not installed. "
    "Install it with: cargo install git-cliff, "
    "brew install git-cliff, or see https://git-cliff.org/docs/installation"
)


async def _parse_git_log(from_ref: str | None) -> list[dict[str, Any]]:
    """Run ``git log`` and parse conventional commits since *from_ref*.

    If *from_ref* is None the full history is used.
    """
    range_spec = f"{from_ref}..HEAD" if from_ref else "HEAD"
    log_format = "%H|%s|%an|%aI"

    _, stdout, _ = await _run(
        "git", "log", f"--format={log_format}", range_spec
    )

    commits: list[dict[str, Any]] = []
    for line in stdout.strip().splitlines():
        if not line.strip():
            continue
        parts = line.split("|", 3)
        if len(parts) < 4:
            continue
        hash_, subject, author, date = parts
        parsed = parse_conventional_commit(subject)
        if parsed:
            commits.append(
                {
                    "hash": hash_,
                    "type": parsed["type"],
                    "scope": parsed["scope"],
                    "subject": parsed["subject"],
                    "breaking": parsed["breaking"],
                    "author": author,
                    "date": date,
                }
            )
        else:
            # Include non-conventional commits too, but flag them.
            commits.append(
                {
                    "hash": hash_,
                    "type": "other",
                    "scope": None,
                    "subject": subject,
                    "breaking": False,
                    "author": author,
                    "date": date,
                }
            )

    # Also check bodies for BREAKING CHANGE trailers.
    for commit in commits:
        if commit["breaking"]:
            continue
        rc, body, _ = await _run(
            "git", "log", "-1", "--format=%b", commit["hash"], check=False
        )
        if rc == 0 and "BREAKING CHANGE:" in body:
            commit["breaking"] = True

    return commits


# ---------------------------------------------------------------------------
# MCP Server
# ---------------------------------------------------------------------------

server = FastMCP(
    "teamwork-mcp-changelog",
    instructions="Changelog generation tools powered by git-cliff",
)


@server.tool()
async def generate_changelog(
    from_ref: str,
    to_ref: str = "HEAD",
    format: str = "markdown",
) -> str | list:
    """Generate a changelog between two git refs using git-cliff.

    Args:
        from_ref: Starting git ref (tag, branch, or commit SHA).
        to_ref: Ending git ref. Defaults to HEAD.
        format: Output format — ``"markdown"`` (default) returns a string,
                ``"json"`` returns a structured list of versions with commits.

    Returns:
        Markdown string or JSON list depending on *format*.
    """
    if not await _in_git_repo():
        return "Error: not inside a git repository."

    if not await _git_cliff_available():
        return f"Error: {GIT_CLIFF_INSTALL_MSG}"

    for ref in (from_ref, to_ref):
        if not await _ref_exists(ref):
            return f"Error: ref '{ref}' does not exist."

    range_arg = f"{from_ref}..{to_ref}"

    if format == "json":
        _, stdout, _ = await _run(
            "git-cliff", "--range", range_arg, "--context"
        )
        try:
            return json.loads(stdout)
        except json.JSONDecodeError:
            return stdout
    else:
        _, stdout, _ = await _run("git-cliff", "--range", range_arg)
        return stdout


@server.tool()
async def preview_release_notes(tag: str | None = None) -> dict:
    """Generate release notes for the next (unreleased) version.

    Args:
        tag: Optional explicit tag to generate notes for. If omitted the
             unreleased changes since the last tag are used.

    Returns:
        A dict with ``title``, ``body``, ``breaking_changes``, and
        ``highlights``.
    """
    if not await _in_git_repo():
        return {"error": "Not inside a git repository."}

    if not await _git_cliff_available():
        return {"error": GIT_CLIFF_INSTALL_MSG}

    cmd: list[str] = ["git-cliff", "--unreleased"]
    if tag:
        cmd.extend(["--tag", tag])

    _, body, _ = await _run(*cmd)

    # Also grab the structured context for breaking/highlights.
    cmd_ctx = cmd + ["--context"]
    _, ctx_raw, _ = await _run(*cmd_ctx)

    breaking_changes: list[str] = []
    highlights: list[str] = []
    try:
        context = json.loads(ctx_raw)
        for version_block in context:
            for commit in version_block.get("commits", []):
                msg = commit.get("message", "")
                if commit.get("breaking", False):
                    breaking_changes.append(msg)
                if commit.get("group", "").lower() in ("features", "feat"):
                    highlights.append(msg)
    except (json.JSONDecodeError, TypeError, KeyError):
        pass

    title = f"Release {tag}" if tag else "Next Release"

    return {
        "title": title,
        "body": body.strip(),
        "breaking_changes": breaking_changes,
        "highlights": highlights,
    }


@server.tool()
async def get_unreleased_commits(from_tag: str | None = None) -> list:
    """Get a structured list of commits since the last release tag.

    Args:
        from_tag: Start counting from this tag. If omitted the most recent
                  tag is detected automatically.

    Returns:
        A list of commit dicts with ``hash``, ``type``, ``scope``,
        ``subject``, ``breaking``, ``author``, and ``date``.
    """
    if not await _in_git_repo():
        return [{"error": "Not inside a git repository."}]

    ref = from_tag
    if ref is None:
        ref = await _latest_tag()
        # ref may still be None if there are no tags — that's OK, we'll
        # log the full history in that case.

    if ref is not None and not await _ref_exists(ref):
        return [{"error": f"Tag '{ref}' does not exist."}]

    return await _parse_git_log(ref)


@server.tool()
async def suggest_next_version(current: str) -> dict:
    """Suggest the next semantic version based on unreleased commits.

    Args:
        current: The current version string (e.g. ``"v1.2.3"`` or
                 ``"1.2.3"``).

    Returns:
        A dict with ``version``, ``bump`` (``"major"``, ``"minor"``, or
        ``"patch"``), and ``reasoning``.
    """
    if not await _in_git_repo():
        return {"error": "Not inside a git repository."}

    try:
        parse_version(current)
    except ValueError as exc:
        return {"error": str(exc)}

    # Determine the tag to use as the starting point.
    tag = current if current.startswith("v") else f"v{current}"
    ref: str | None = tag
    if not await _ref_exists(tag):
        # If the exact tag doesn't exist, try without the 'v' prefix.
        alt = current.lstrip("v")
        if await _ref_exists(alt):
            ref = alt
        else:
            # Fall back to latest tag or full history.
            ref = await _latest_tag()

    commits = await _parse_git_log(ref)
    if not commits:
        return {
            "version": current,
            "bump": "none",
            "reasoning": "No new commits found since the current version.",
        }

    return compute_next_version(current, commits)


# ---------------------------------------------------------------------------
# Entrypoint
# ---------------------------------------------------------------------------


def main() -> None:
    """Run the MCP server over stdio."""
    server.run()


if __name__ == "__main__":
    main()
