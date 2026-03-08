"""MCP server for conventional commit message generation and validation.

Provides tools to generate and validate structured, conventional-commit-compliant
messages from git diffs.
"""

from __future__ import annotations

import re
from pathlib import PurePosixPath
from typing import Any

from mcp.server.fastmcp import FastMCP

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

COMMIT_TYPES: list[dict[str, str]] = [
    {"type": "feat", "description": "A new feature", "semver_impact": "minor"},
    {"type": "fix", "description": "A bug fix", "semver_impact": "patch"},
    {"type": "docs", "description": "Documentation only changes", "semver_impact": "none"},
    {
        "type": "style",
        "description": "Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)",
        "semver_impact": "none",
    },
    {
        "type": "refactor",
        "description": "A code change that neither fixes a bug nor adds a feature",
        "semver_impact": "none",
    },
    {"type": "perf", "description": "A code change that improves performance", "semver_impact": "patch"},
    {"type": "test", "description": "Adding missing tests or correcting existing tests", "semver_impact": "none"},
    {
        "type": "build",
        "description": "Changes that affect the build system or external dependencies",
        "semver_impact": "patch",
    },
    {
        "type": "ci",
        "description": "Changes to CI configuration files and scripts",
        "semver_impact": "none",
    },
    {"type": "chore", "description": "Other changes that don't modify src or test files", "semver_impact": "none"},
    {"type": "revert", "description": "Reverts a previous commit", "semver_impact": "patch"},
]

VALID_TYPES: set[str] = {t["type"] for t in COMMIT_TYPES}

# Regex for the first line of a conventional commit message.
_HEADER_RE = re.compile(
    r"^(?P<type>[a-z]+)"
    r"(?:\((?P<scope>[a-zA-Z0-9_./-]+)\))?"
    r"(?P<breaking>!)?"
    r":\s(?P<subject>.+)$"
)

# File-path patterns for type detection (order matters – first match wins).
_TEST_PATTERNS: tuple[str, ...] = ("test/", "tests/", "_test.", ".test.", "test_", "_spec.", ".spec.")
_DOC_PATTERNS: tuple[str, ...] = ("docs/", "doc/")
_DOC_EXTENSIONS: tuple[str, ...] = (".md", ".rst", ".txt", ".adoc")
_CI_PATTERNS: tuple[str, ...] = (".github/workflows/", ".github/actions/", ".circleci/", ".travis", "Jenkinsfile")
_BUILD_FILES: tuple[str, ...] = ("Makefile", "Dockerfile", "docker-compose", "Taskfile")
_DEP_FILES: tuple[str, ...] = (
    "package.json",
    "package-lock.json",
    "go.mod",
    "go.sum",
    "requirements.txt",
    "Pipfile",
    "Cargo.toml",
    "Cargo.lock",
    "pyproject.toml",
    "poetry.lock",
    "pnpm-lock.yaml",
    "yarn.lock",
    "Gemfile",
    "Gemfile.lock",
)

_FIX_HINTS: tuple[str, ...] = ("fix", "bug", "error", "crash", "issue", "broken", "regression", "patch")

# ---------------------------------------------------------------------------
# Diff parsing helpers
# ---------------------------------------------------------------------------


def parse_diff_files(diff: str) -> list[dict[str, Any]]:
    """Extract file information from a unified diff.

    Returns a list of dicts with keys: path, status ("added", "deleted",
    "modified"), insertions, deletions.
    """
    files: list[dict[str, Any]] = []
    current: dict[str, Any] | None = None

    for line in diff.splitlines():
        if line.startswith("diff --git"):
            if current is not None:
                files.append(current)
            # Extract b-side path (new path)
            parts = line.split(" b/", 1)
            path = parts[1] if len(parts) == 2 else ""
            current = {"path": path, "status": "modified", "insertions": 0, "deletions": 0}
        elif current is not None:
            if line.startswith("new file"):
                current["status"] = "added"
            elif line.startswith("deleted file"):
                current["status"] = "deleted"
            elif line.startswith("+") and not line.startswith("+++"):
                current["insertions"] += 1
            elif line.startswith("-") and not line.startswith("---"):
                current["deletions"] += 1

    if current is not None:
        files.append(current)
    return files


# ---------------------------------------------------------------------------
# Type & scope detection (pure functions)
# ---------------------------------------------------------------------------


def detect_commit_type(files: list[dict[str, Any]], hint: str | None = None) -> str:
    """Determine the conventional commit type from file paths and an optional hint."""
    if not files:
        return "chore"

    # If a hint strongly suggests a fix, prefer that.
    if hint:
        hint_lower = hint.lower()
        if any(h in hint_lower for h in _FIX_HINTS):
            return "fix"

    paths = [f["path"] for f in files]
    statuses = [f["status"] for f in files]

    all_tests = all(_is_test_file(p) for p in paths)
    all_docs = all(_is_doc_file(p) for p in paths)
    all_ci = all(_is_ci_file(p) for p in paths)
    all_build = all(_is_build_or_dep_file(p) for p in paths)

    if all_tests:
        return "test"
    if all_docs:
        return "docs"
    if all_ci:
        return "ci"
    if all_build:
        return "build"
    if all(s == "deleted" for s in statuses):
        return "refactor"
    if all(s == "added" for s in statuses):
        return "feat"

    # Mixed — fall back to hint or majority
    if hint:
        hint_lower = hint.lower()
        if "refactor" in hint_lower:
            return "refactor"
        if "perf" in hint_lower or "performance" in hint_lower:
            return "perf"
        if "style" in hint_lower or "format" in hint_lower:
            return "style"
        if "revert" in hint_lower:
            return "revert"

    return "feat"


def _is_test_file(path: str) -> bool:
    return any(pat in path for pat in _TEST_PATTERNS)


def _is_doc_file(path: str) -> bool:
    if any(path.startswith(pat) for pat in _DOC_PATTERNS):
        return True
    suffix = PurePosixPath(path).suffix.lower()
    return suffix in _DOC_EXTENSIONS


def _is_ci_file(path: str) -> bool:
    return any(pat in path for pat in _CI_PATTERNS)


def _is_build_or_dep_file(path: str) -> bool:
    name = PurePosixPath(path).name
    if name in _DEP_FILES:
        return True
    return any(name.startswith(pat) for pat in _BUILD_FILES)


def detect_scope(files: list[dict[str, Any]]) -> str | None:
    """Detect scope from the most common top-level directory of changed files.

    Returns ``None`` when no meaningful scope can be determined.
    """
    if not files:
        return None

    scopes: list[str] = []
    for f in files:
        parts = PurePosixPath(f["path"]).parts
        if len(parts) >= 2:
            # Use the deepest meaningful directory.  For paths like
            # ``internal/config/foo.go`` → "config"; for ``cmd/teamwork/cmd/`` → "cli".
            candidate = parts[-2]  # parent dir of the file
            # Map well-known directories
            if candidate == "cmd":
                candidate = "cli"
            scopes.append(candidate)

    if not scopes:
        return None

    # Pick the most common scope.
    scope_counts: dict[str, int] = {}
    for s in scopes:
        scope_counts[s] = scope_counts.get(s, 0) + 1
    return max(scope_counts, key=scope_counts.get)  # type: ignore[arg-type]


# ---------------------------------------------------------------------------
# Subject generation
# ---------------------------------------------------------------------------


def generate_subject(files: list[dict[str, Any]], hint: str | None = None) -> str:
    """Produce a short, lowercase subject line summarising the change."""
    if hint:
        # Clean the hint into a usable subject.
        subject = hint.strip().rstrip(".")
        # Ensure lowercase start
        subject = subject[0].lower() + subject[1:] if subject else subject
        # Truncate to 72 chars (minus space for type prefix later accounted by caller)
        if len(subject) > 68:
            subject = subject[:65] + "..."
        return subject

    if not files:
        return "update project files"

    statuses = {f["status"] for f in files}
    paths = [f["path"] for f in files]

    if statuses == {"added"}:
        if len(paths) == 1:
            return f"add {PurePosixPath(paths[0]).name}"
        return f"add {len(paths)} new files"
    if statuses == {"deleted"}:
        if len(paths) == 1:
            return f"remove {PurePosixPath(paths[0]).name}"
        return f"remove {len(paths)} files"

    if len(paths) == 1:
        return f"update {PurePosixPath(paths[0]).name}"
    # Summarise by common directory
    scope = detect_scope(files)
    if scope:
        return f"update {scope} module"
    return f"update {len(paths)} files"


# ---------------------------------------------------------------------------
# Breaking-change detection
# ---------------------------------------------------------------------------


def detect_breaking_change(diff: str, hint: str | None = None) -> bool:
    """Return ``True`` when the diff or hint indicates a breaking change."""
    if hint and "breaking" in hint.lower():
        return True
    if "BREAKING CHANGE" in diff or "BREAKING-CHANGE" in diff:
        return True
    return False


# ---------------------------------------------------------------------------
# Commit message assembly
# ---------------------------------------------------------------------------


def build_commit_message(
    commit_type: str,
    scope: str | None,
    subject: str,
    body: str | None = None,
    breaking_change: bool = False,
    footer: str | None = None,
) -> str:
    """Assemble a full conventional-commit message string."""
    header = commit_type
    if scope:
        header += f"({scope})"
    if breaking_change:
        header += "!"
    header += f": {subject}"

    parts = [header]
    if body:
        parts.append("")  # blank line
        parts.append(body)
    if footer:
        parts.append("")
        parts.append(footer)
    return "\n".join(parts)


# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------


def validate_message(message: str) -> dict[str, Any]:
    """Validate a commit message against the Conventional Commits spec.

    Returns a dict with keys: valid, type, scope, errors, suggestions.
    """
    errors: list[str] = []
    suggestions: list[str] = []
    detected_type: str | None = None
    detected_scope: str | None = None

    lines = message.split("\n")
    header = lines[0] if lines else ""

    match = _HEADER_RE.match(header)
    if not match:
        errors.append("Header does not match conventional commit format: type(scope): description")
        # Try to provide more specific feedback
        if ":" not in header:
            errors.append("Missing colon separator after type")
        elif not header.split(":")[0].strip().replace("(", "").replace(")", "").replace("!", "").isalpha():
            errors.append("Type must contain only lowercase letters")
        return {
            "valid": False,
            "type": None,
            "scope": None,
            "errors": errors,
            "suggestions": ["Use format: type(scope): description  or  type: description"],
        }

    detected_type = match.group("type")
    detected_scope = match.group("scope")
    subject = match.group("subject")

    if detected_type not in VALID_TYPES:
        errors.append(f"Unknown commit type '{detected_type}'. Valid types: {', '.join(sorted(VALID_TYPES))}")

    # Subject checks
    if subject and subject[0].isupper():
        errors.append("Subject must start with a lowercase letter")
        suggestions.append(f"Change to: {subject[0].lower()}{subject[1:]}")
    if subject and subject.endswith("."):
        errors.append("Subject must not end with a period")
        suggestions.append(f"Remove trailing period: {subject[:-1]}")
    if len(header) > 72:
        errors.append(f"Header exceeds 72 characters ({len(header)} chars)")
        suggestions.append("Shorten the subject line to fit within 72 characters")

    # Body separation check
    if len(lines) > 1 and lines[1].strip():
        errors.append("Body must be separated from header by a blank line")
        suggestions.append("Add an empty line between the header and body")

    # BREAKING CHANGE footer format check
    has_breaking_bang = match.group("breaking") == "!"
    has_breaking_footer = False
    for line in lines:
        if line.startswith("BREAKING CHANGE:") or line.startswith("BREAKING-CHANGE:"):
            has_breaking_footer = True
            break

    if has_breaking_footer and has_breaking_bang:
        suggestions.append("Both '!' and BREAKING CHANGE footer present — both are valid, but pick one for clarity")

    return {
        "valid": len(errors) == 0,
        "type": detected_type,
        "scope": detected_scope,
        "errors": errors,
        "suggestions": suggestions,
    }


# ---------------------------------------------------------------------------
# PR description generation
# ---------------------------------------------------------------------------

_DEFAULT_PR_TEMPLATE = """\
## {title}

### Summary
{summary}

### Motivation
{motivation}

### Changes Made
{changes}

### Test Plan
{test_plan}

### Breaking Changes
{breaking}
"""


def generate_pr_description_text(
    files: list[dict[str, Any]],
    diff: str,
    template: str | None = None,
) -> str:
    """Generate a structured PR description in markdown."""
    total_insertions = sum(f["insertions"] for f in files)
    total_deletions = sum(f["deletions"] for f in files)

    # Group files by top-level directory
    groups: dict[str, list[str]] = {}
    for f in files:
        parts = PurePosixPath(f["path"]).parts
        group = parts[0] if parts else "root"
        groups.setdefault(group, []).append(f["path"])

    # Build changes list
    changes_lines: list[str] = []
    for group, paths in sorted(groups.items()):
        changes_lines.append(f"- **{group}/**")
        for p in sorted(paths):
            changes_lines.append(f"  - `{p}`")

    commit_type = detect_commit_type(files)
    scope = detect_scope(files)
    subject = generate_subject(files)
    title = f"{commit_type}"
    if scope:
        title += f"({scope})"
    title += f": {subject}"

    summary = (
        f"This PR modifies **{len(files)}** file(s) "
        f"with **{total_insertions}** insertion(s) and **{total_deletions}** deletion(s)."
    )

    motivation = "Improve project functionality and maintainability."
    test_plan = "- [ ] Unit tests pass\n- [ ] Manual verification"
    breaking = "None" if not detect_breaking_change(diff) else "See BREAKING CHANGE notes in the diff."

    tmpl = template if template else _DEFAULT_PR_TEMPLATE
    return tmpl.format(
        title=title,
        summary=summary,
        motivation=motivation,
        changes="\n".join(changes_lines),
        test_plan=test_plan,
        breaking=breaking,
    )


# ---------------------------------------------------------------------------
# MCP server definition
# ---------------------------------------------------------------------------

server = FastMCP("teamwork-mcp-commits")


@server.tool()
async def generate_commit_message(diff: str, hint: str | None = None) -> dict[str, Any]:
    """Analyze a git diff and produce a conventional-commit-compliant message.

    Args:
        diff: A unified diff string (output of ``git diff``).
        hint: Optional human hint to guide type/subject detection.

    Returns:
        A dict with keys: type, scope, subject, body, breaking_change, footer,
        full_message.
    """
    files = parse_diff_files(diff)
    commit_type = detect_commit_type(files, hint)
    scope = detect_scope(files)
    subject = generate_subject(files, hint)
    breaking = detect_breaking_change(diff, hint)
    footer = "BREAKING CHANGE: see commit body for details" if breaking else None
    body = None

    full_message = build_commit_message(commit_type, scope, subject, body, breaking, footer)

    return {
        "type": commit_type,
        "scope": scope,
        "subject": subject,
        "body": body,
        "breaking_change": breaking,
        "footer": footer,
        "full_message": full_message,
    }


@server.tool()
async def generate_pr_description(diff: str, template: str | None = None) -> str:
    """Generate a structured PR description with markdown sections.

    Sections include: Title, Summary, Motivation, Changes Made, Test Plan,
    and Breaking Changes.

    Args:
        diff: A unified diff string.
        template: Optional Python format-string template with placeholders:
                  {title}, {summary}, {motivation}, {changes}, {test_plan},
                  {breaking}.

    Returns:
        A markdown-formatted PR description string.
    """
    files = parse_diff_files(diff)
    return generate_pr_description_text(files, diff, template)


@server.tool()
async def validate_commit_message(message: str) -> dict[str, Any]:
    """Validate a commit message against the Conventional Commits specification.

    Args:
        message: The full commit message to validate.

    Returns:
        A dict with keys: valid (bool), type, scope, errors (list[str]),
        suggestions (list[str]).
    """
    return validate_message(message)


@server.tool()
async def list_commit_types() -> list[dict[str, str]]:
    """List all valid conventional commit types with descriptions.

    Returns:
        A list of dicts, each with keys: type, description, semver_impact
        (one of "major", "minor", "patch", or "none").
    """
    return COMMIT_TYPES


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------


def main() -> None:
    """Run the MCP server over stdio."""
    server.run()


if __name__ == "__main__":
    main()
