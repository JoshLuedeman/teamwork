"""Tests for the teamwork-mcp-changelog server.

All subprocess interactions are mocked so that no real ``git`` or ``git-cliff``
binary is required.
"""

from __future__ import annotations

import json
from unittest.mock import AsyncMock, patch

import pytest

from teamwork_mcp_changelog.server import (
    compute_next_version,
    format_version,
    generate_changelog,
    get_unreleased_commits,
    parse_conventional_commit,
    parse_version,
    preview_release_notes,
    suggest_next_version,
)


# =========================================================================
# 1. Conventional-commit parsing
# =========================================================================


class TestParseConventionalCommit:
    """Test the conventional-commit regex and parser."""

    def test_simple_fix(self):
        result = parse_conventional_commit("fix: resolve crash on startup")
        assert result is not None
        assert result["type"] == "fix"
        assert result["scope"] is None
        assert result["subject"] == "resolve crash on startup"
        assert result["breaking"] is False

    def test_feat_with_scope(self):
        result = parse_conventional_commit("feat(auth): add OAuth2 login")
        assert result is not None
        assert result["type"] == "feat"
        assert result["scope"] == "auth"
        assert result["subject"] == "add OAuth2 login"
        assert result["breaking"] is False

    def test_breaking_with_bang(self):
        result = parse_conventional_commit("feat!: remove legacy endpoint")
        assert result is not None
        assert result["type"] == "feat"
        assert result["breaking"] is True

    def test_breaking_with_scope_and_bang(self):
        result = parse_conventional_commit("refactor(api)!: redesign auth flow")
        assert result is not None
        assert result["type"] == "refactor"
        assert result["scope"] == "api"
        assert result["subject"] == "redesign auth flow"
        assert result["breaking"] is True

    def test_chore_no_scope(self):
        result = parse_conventional_commit("chore: update deps")
        assert result is not None
        assert result["type"] == "chore"
        assert result["scope"] is None

    def test_docs_type(self):
        result = parse_conventional_commit("docs: update README")
        assert result is not None
        assert result["type"] == "docs"

    def test_multi_word_scope(self):
        result = parse_conventional_commit("feat(user auth): add MFA support")
        assert result is not None
        assert result["scope"] == "user auth"
        assert result["subject"] == "add MFA support"

    def test_non_conventional_returns_none(self):
        result = parse_conventional_commit("just a regular commit message")
        assert result is None

    def test_missing_colon_returns_none(self):
        result = parse_conventional_commit("feat add new thing")
        assert result is None

    def test_empty_string(self):
        result = parse_conventional_commit("")
        assert result is None

    def test_with_leading_whitespace(self):
        result = parse_conventional_commit("  fix: trim whitespace")
        assert result is not None
        assert result["type"] == "fix"
        assert result["subject"] == "trim whitespace"


# =========================================================================
# 2. Version parsing and formatting
# =========================================================================


class TestVersionParsing:
    """Test semver parsing and formatting."""

    def test_parse_with_v_prefix(self):
        assert parse_version("v1.2.3") == (1, 2, 3)

    def test_parse_without_v_prefix(self):
        assert parse_version("1.2.3") == (1, 2, 3)

    def test_parse_v0_0_1(self):
        assert parse_version("v0.0.1") == (0, 0, 1)

    def test_parse_large_numbers(self):
        assert parse_version("v10.200.3000") == (10, 200, 3000)

    def test_parse_invalid_raises(self):
        with pytest.raises(ValueError, match="Invalid semver"):
            parse_version("not-a-version")

    def test_parse_two_parts_raises(self):
        with pytest.raises(ValueError, match="Invalid semver"):
            parse_version("1.2")

    def test_format_version(self):
        assert format_version(1, 2, 3) == "v1.2.3"

    def test_format_version_zeros(self):
        assert format_version(0, 0, 1) == "v0.0.1"


# =========================================================================
# 3. suggest_next_version / compute_next_version logic
# =========================================================================


class TestComputeNextVersion:
    """Test the pure-logic version bump computation."""

    def test_breaking_change_bumps_major(self):
        commits = [
            {"type": "feat", "subject": "add thing", "breaking": True},
        ]
        result = compute_next_version("v1.2.3", commits)
        assert result["version"] == "v2.0.0"
        assert result["bump"] == "major"

    def test_feat_bumps_minor(self):
        commits = [
            {"type": "feat", "subject": "new feature", "breaking": False},
        ]
        result = compute_next_version("v1.2.3", commits)
        assert result["version"] == "v1.3.0"
        assert result["bump"] == "minor"

    def test_only_fixes_bumps_patch(self):
        commits = [
            {"type": "fix", "subject": "fix bug", "breaking": False},
            {"type": "chore", "subject": "cleanup", "breaking": False},
        ]
        result = compute_next_version("v1.2.3", commits)
        assert result["version"] == "v1.2.4"
        assert result["bump"] == "patch"

    def test_breaking_takes_precedence_over_feat(self):
        commits = [
            {"type": "feat", "subject": "new api", "breaking": True},
            {"type": "feat", "subject": "add widget", "breaking": False},
        ]
        result = compute_next_version("v1.2.3", commits)
        assert result["bump"] == "major"

    def test_feat_takes_precedence_over_fix(self):
        commits = [
            {"type": "feat", "subject": "new feature", "breaking": False},
            {"type": "fix", "subject": "fix bug", "breaking": False},
        ]
        result = compute_next_version("v1.2.3", commits)
        assert result["bump"] == "minor"

    def test_reasoning_mentions_breaking(self):
        commits = [
            {"type": "feat", "subject": "drop v1 api", "breaking": True},
        ]
        result = compute_next_version("v1.0.0", commits)
        assert "BREAKING CHANGE" in result["reasoning"]
        assert "drop v1 api" in result["reasoning"]

    def test_reasoning_mentions_features(self):
        commits = [
            {"type": "feat", "subject": "add search", "breaking": False},
        ]
        result = compute_next_version("v1.0.0", commits)
        assert "add search" in result["reasoning"]

    def test_reasoning_patch_message(self):
        commits = [
            {"type": "fix", "subject": "typo", "breaking": False},
        ]
        result = compute_next_version("v1.0.0", commits)
        assert "fix/chore/docs" in result["reasoning"]

    def test_version_without_v_prefix(self):
        commits = [
            {"type": "fix", "subject": "bug", "breaking": False},
        ]
        result = compute_next_version("2.0.0", commits)
        assert result["version"] == "v2.0.1"
        assert result["bump"] == "patch"


# =========================================================================
# 4. get_unreleased_commits — mock subprocess
# =========================================================================


def _arg_matches(key_part: str, args: tuple[str, ...]) -> bool:
    """Check if *key_part* matches any element of *args*.

    Supports both exact matches and substring matches so that a key like
    ``"--format"`` matches an arg like ``"--format=%H|%s|%an|%aI"``.
    """
    return any(key_part == a or key_part in a for a in args)


def _make_run_mock(responses: dict[tuple[str, ...], tuple[int, str, str]]):
    """Create an async mock for ``_run`` that returns canned responses.

    *responses* maps key-tuples to ``(returncode, stdout, stderr)``.  Each
    element in the key-tuple is checked against the actual args via
    substring matching, and the **longest** (most specific) matching key
    wins.  Any command not matched returns ``(0, "", "")``.
    """

    async def _fake_run(*args, check=True):
        # Find the best (longest) matching key.
        best_key: tuple[str, ...] | None = None
        best_value: tuple[int, str, str] | None = None
        for key, value in responses.items():
            if all(_arg_matches(k, args) for k in key):
                if best_key is None or len(key) > len(best_key):
                    best_key = key
                    best_value = value
        if best_value is not None:
            if check and best_value[0] != 0:
                raise RuntimeError(f"Command failed: {args}")
            return best_value
        return (0, "", "")

    return _fake_run


class TestGetUnreleasedCommits:
    """Test get_unreleased_commits with mocked subprocess."""

    @pytest.mark.asyncio
    async def test_parses_git_log_output(self):
        log_output = (
            "abc123|feat(ui): add dark mode|Alice|2024-01-15T10:00:00+00:00\n"
            "def456|fix: resolve crash|Bob|2024-01-14T09:00:00+00:00\n"
        )
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (0, "true\n", ""),
            ("describe", "--tags"): (0, "v1.0.0\n", ""),
            ("rev-parse", "--verify", "v1.0.0"): (0, "aaa\n", ""),
            ("git", "log", "--format"): (0, log_output, ""),
            # Body lookups (no BREAKING CHANGE)
            ("log", "-1", "--format=%b", "abc123"): (0, "", ""),
            ("log", "-1", "--format=%b", "def456"): (0, "", ""),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await get_unreleased_commits()

        assert len(result) == 2
        assert result[0]["type"] == "feat"
        assert result[0]["scope"] == "ui"
        assert result[0]["subject"] == "add dark mode"
        assert result[0]["author"] == "Alice"
        assert result[1]["type"] == "fix"

    @pytest.mark.asyncio
    async def test_detects_breaking_change_in_body(self):
        log_output = "abc123|feat: new api|Alice|2024-01-15T10:00:00+00:00\n"
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (0, "true\n", ""),
            ("describe", "--tags"): (0, "v1.0.0\n", ""),
            ("rev-parse", "--verify", "v1.0.0"): (0, "aaa\n", ""),
            ("git", "log", "--format"): (0, log_output, ""),
            ("log", "-1", "--format=%b", "abc123"): (
                0,
                "Some details\n\nBREAKING CHANGE: old api removed\n",
                "",
            ),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await get_unreleased_commits()

        assert len(result) == 1
        assert result[0]["breaking"] is True

    @pytest.mark.asyncio
    async def test_not_in_git_repo(self):
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (1, "", "not a repo"),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await get_unreleased_commits()

        assert len(result) == 1
        assert "error" in result[0]

    @pytest.mark.asyncio
    async def test_invalid_tag_returns_error(self):
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (0, "true\n", ""),
            ("rev-parse", "--verify", "v999.0.0"): (1, "", "bad ref"),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await get_unreleased_commits(from_tag="v999.0.0")

        assert len(result) == 1
        assert "error" in result[0]


# =========================================================================
# 5. generate_changelog — git-cliff not installed
# =========================================================================


class TestGenerateChangelog:
    """Test generate_changelog with mocked subprocess."""

    @pytest.mark.asyncio
    async def test_error_when_git_cliff_not_installed(self):
        async def _fake_run(*args, check=True):
            if "git-cliff" in args:
                raise FileNotFoundError("git-cliff not found")
            if "rev-parse" in args and "--is-inside-work-tree" in args:
                return (0, "true\n", "")
            if "rev-parse" in args and "--verify" in args:
                return (0, "aaa\n", "")
            return (0, "", "")

        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_fake_run,
        ):
            result = await generate_changelog(from_ref="v1.0.0", to_ref="HEAD")

        assert isinstance(result, str)
        assert "git-cliff" in result.lower() or "not installed" in result.lower()

    @pytest.mark.asyncio
    async def test_error_when_not_in_git_repo(self):
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (1, "", "not a repo"),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await generate_changelog(from_ref="v1.0.0")

        assert "not inside a git repository" in result.lower()

    @pytest.mark.asyncio
    async def test_error_when_ref_not_found(self):
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (0, "true\n", ""),
            ("git-cliff", "--version"): (0, "git-cliff 2.0\n", ""),
            ("rev-parse", "--verify", "v999.0.0"): (1, "", "bad"),
            ("rev-parse", "--verify", "HEAD"): (0, "aaa\n", ""),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await generate_changelog(from_ref="v999.0.0")

        assert "does not exist" in result.lower()

    @pytest.mark.asyncio
    async def test_markdown_format_returns_string(self):
        changelog_md = "# Changelog\n\n## v1.1.0\n\n- feat: new thing\n"
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (0, "true\n", ""),
            ("git-cliff", "--version"): (0, "git-cliff 2.0\n", ""),
            ("rev-parse", "--verify", "v1.0.0"): (0, "aaa\n", ""),
            ("rev-parse", "--verify", "HEAD"): (0, "bbb\n", ""),
            ("git-cliff", "--range"): (0, changelog_md, ""),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await generate_changelog(
                from_ref="v1.0.0", to_ref="HEAD", format="markdown"
            )

        assert isinstance(result, str)
        assert "Changelog" in result

    @pytest.mark.asyncio
    async def test_json_format_returns_list(self):
        json_data = [{"version": "1.1.0", "commits": []}]
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (0, "true\n", ""),
            ("git-cliff", "--version"): (0, "git-cliff 2.0\n", ""),
            ("rev-parse", "--verify", "v1.0.0"): (0, "aaa\n", ""),
            ("rev-parse", "--verify", "HEAD"): (0, "bbb\n", ""),
            ("git-cliff", "--range", "--context"): (
                0,
                json.dumps(json_data),
                "",
            ),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await generate_changelog(
                from_ref="v1.0.0", to_ref="HEAD", format="json"
            )

        assert isinstance(result, list)
        assert result[0]["version"] == "1.1.0"


# =========================================================================
# 6. preview_release_notes
# =========================================================================


class TestPreviewReleaseNotes:
    """Test preview_release_notes with mocked subprocess."""

    @pytest.mark.asyncio
    async def test_error_when_git_cliff_not_installed(self):
        async def _fake_run(*args, check=True):
            if "git-cliff" in args:
                raise FileNotFoundError("git-cliff not found")
            if "rev-parse" in args:
                return (0, "true\n", "")
            return (0, "", "")

        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_fake_run,
        ):
            result = await preview_release_notes()

        assert "error" in result

    @pytest.mark.asyncio
    async def test_returns_structured_notes(self):
        body_md = "## Unreleased\n\n- feat: add search\n"
        ctx_json = json.dumps(
            [
                {
                    "version": None,
                    "commits": [
                        {
                            "message": "add search",
                            "group": "Features",
                            "breaking": False,
                        },
                        {
                            "message": "drop old api",
                            "group": "Breaking",
                            "breaking": True,
                        },
                    ],
                }
            ]
        )
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (0, "true\n", ""),
            ("git-cliff", "--version"): (0, "git-cliff 2.0\n", ""),
            ("git-cliff", "--unreleased"): (0, body_md, ""),
            ("git-cliff", "--unreleased", "--context"): (0, ctx_json, ""),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await preview_release_notes()

        assert result["title"] == "Next Release"
        assert "Unreleased" in result["body"]
        assert len(result["breaking_changes"]) == 1
        assert "drop old api" in result["breaking_changes"][0]


# =========================================================================
# 7. suggest_next_version (integration with mocked git)
# =========================================================================


class TestSuggestNextVersion:
    """Test suggest_next_version tool with mocked subprocess."""

    @pytest.mark.asyncio
    async def test_suggests_major_for_breaking(self):
        log_output = "abc123|feat!: remove old api|Alice|2024-01-15T10:00:00+00:00\n"
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (0, "true\n", ""),
            ("rev-parse", "--verify", "v1.0.0"): (0, "aaa\n", ""),
            ("git", "log", "--format"): (0, log_output, ""),
            ("log", "-1", "--format=%b", "abc123"): (0, "", ""),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await suggest_next_version(current="v1.0.0")

        assert result["version"] == "v2.0.0"
        assert result["bump"] == "major"

    @pytest.mark.asyncio
    async def test_suggests_minor_for_feat(self):
        log_output = "abc123|feat: add search|Alice|2024-01-15T10:00:00+00:00\n"
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (0, "true\n", ""),
            ("rev-parse", "--verify", "v1.0.0"): (0, "aaa\n", ""),
            ("git", "log", "--format"): (0, log_output, ""),
            ("log", "-1", "--format=%b", "abc123"): (0, "", ""),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await suggest_next_version(current="v1.0.0")

        assert result["version"] == "v1.1.0"
        assert result["bump"] == "minor"

    @pytest.mark.asyncio
    async def test_suggests_patch_for_fixes_only(self):
        log_output = (
            "abc123|fix: typo|Alice|2024-01-15T10:00:00+00:00\n"
            "def456|chore: update deps|Bob|2024-01-14T09:00:00+00:00\n"
        )
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (0, "true\n", ""),
            ("rev-parse", "--verify", "v1.0.0"): (0, "aaa\n", ""),
            ("git", "log", "--format"): (0, log_output, ""),
            ("log", "-1", "--format=%b", "abc123"): (0, "", ""),
            ("log", "-1", "--format=%b", "def456"): (0, "", ""),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await suggest_next_version(current="v1.0.0")

        assert result["version"] == "v1.0.1"
        assert result["bump"] == "patch"

    @pytest.mark.asyncio
    async def test_invalid_version_string(self):
        responses = {
            ("rev-parse", "--is-inside-work-tree"): (0, "true\n", ""),
        }
        with patch(
            "teamwork_mcp_changelog.server._run",
            side_effect=_make_run_mock(responses),
        ):
            result = await suggest_next_version(current="not-a-version")

        assert "error" in result
