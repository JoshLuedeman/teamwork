"""Tests for teamwork-mcp-commits server logic.

Tests exercise the pure-function helpers directly — no MCP protocol needed.
"""

from __future__ import annotations

import pytest

from teamwork_mcp_commits.server import (
    COMMIT_TYPES,
    VALID_TYPES,
    build_commit_message,
    detect_breaking_change,
    detect_commit_type,
    detect_scope,
    generate_pr_description_text,
    generate_subject,
    parse_diff_files,
    validate_message,
)

# ---------------------------------------------------------------------------
# Sample diffs used across multiple tests
# ---------------------------------------------------------------------------

DIFF_TEST_FILE = """\
diff --git a/tests/test_auth.py b/tests/test_auth.py
new file mode 100644
--- /dev/null
+++ b/tests/test_auth.py
@@ -0,0 +1,10 @@
+import pytest
+
+def test_login():
+    assert True
"""

DIFF_DOC_FILE = """\
diff --git a/docs/guide.md b/docs/guide.md
--- a/docs/guide.md
+++ b/docs/guide.md
@@ -1,3 +1,5 @@
 # Guide
+
+New section about authentication.
"""

DIFF_CI_FILE = """\
diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -10,3 +10,5 @@
     steps:
       - uses: actions/checkout@v4
+      - name: Run lint
+        run: make lint
"""

DIFF_MULTI_FILES = """\
diff --git a/internal/config/loader.go b/internal/config/loader.go
--- a/internal/config/loader.go
+++ b/internal/config/loader.go
@@ -5,3 +5,6 @@
 func Load() {}
+func Reload() {}
+func Validate() {}
diff --git a/internal/config/types.go b/internal/config/types.go
--- a/internal/config/types.go
+++ b/internal/config/types.go
@@ -1,3 +1,5 @@
 package config
+
+type Settings struct {}
"""

DIFF_NEW_FEAT = """\
diff --git a/internal/auth/oauth.go b/internal/auth/oauth.go
new file mode 100644
--- /dev/null
+++ b/internal/auth/oauth.go
@@ -0,0 +1,20 @@
+package auth
+
+func OAuthLogin() {}
"""

DIFF_DELETED = """\
diff --git a/internal/legacy/old.go b/internal/legacy/old.go
deleted file mode 100644
--- a/internal/legacy/old.go
+++ /dev/null
@@ -1,5 +0,0 @@
-package legacy
-
-func Old() {}
"""

DIFF_DOCKERFILE = """\
diff --git a/Dockerfile b/Dockerfile
--- a/Dockerfile
+++ b/Dockerfile
@@ -1,3 +1,4 @@
 FROM golang:1.22
+RUN apt-get update
 WORKDIR /app
"""

DIFF_DEPS = """\
diff --git a/go.mod b/go.mod
--- a/go.mod
+++ b/go.mod
@@ -3,3 +3,4 @@
 module github.com/example/project
+require github.com/new/dep v1.0.0
"""

DIFF_BREAKING = """\
diff --git a/internal/api/handler.go b/internal/api/handler.go
--- a/internal/api/handler.go
+++ b/internal/api/handler.go
@@ -10,5 +10,8 @@
-func HandleV1() {}
+func HandleV2() {}
+// BREAKING CHANGE: removed HandleV1
"""

DIFF_MARKDOWN_ROOT = """\
diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1,2 +1,3 @@
 # Project
+Updated readme.
"""


# ===================================================================
# parse_diff_files
# ===================================================================


class TestParseDiffFiles:
    def test_single_new_file(self) -> None:
        files = parse_diff_files(DIFF_TEST_FILE)
        assert len(files) == 1
        assert files[0]["path"] == "tests/test_auth.py"
        assert files[0]["status"] == "added"
        assert files[0]["insertions"] == 4

    def test_modified_file(self) -> None:
        files = parse_diff_files(DIFF_DOC_FILE)
        assert len(files) == 1
        assert files[0]["status"] == "modified"
        assert files[0]["insertions"] == 2

    def test_deleted_file(self) -> None:
        files = parse_diff_files(DIFF_DELETED)
        assert len(files) == 1
        assert files[0]["status"] == "deleted"
        assert files[0]["deletions"] == 3

    def test_multiple_files(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        assert len(files) == 2
        assert files[0]["path"] == "internal/config/loader.go"
        assert files[1]["path"] == "internal/config/types.go"

    def test_empty_diff(self) -> None:
        assert parse_diff_files("") == []


# ===================================================================
# detect_commit_type
# ===================================================================


class TestDetectCommitType:
    def test_test_files(self) -> None:
        files = parse_diff_files(DIFF_TEST_FILE)
        assert detect_commit_type(files) == "test"

    def test_doc_files(self) -> None:
        files = parse_diff_files(DIFF_DOC_FILE)
        assert detect_commit_type(files) == "docs"

    def test_ci_files(self) -> None:
        files = parse_diff_files(DIFF_CI_FILE)
        assert detect_commit_type(files) == "ci"

    def test_build_files(self) -> None:
        files = parse_diff_files(DIFF_DOCKERFILE)
        assert detect_commit_type(files) == "build"

    def test_dep_files(self) -> None:
        files = parse_diff_files(DIFF_DEPS)
        assert detect_commit_type(files) == "build"

    def test_all_new_files_feat(self) -> None:
        files = parse_diff_files(DIFF_NEW_FEAT)
        assert detect_commit_type(files) == "feat"

    def test_all_deleted_refactor(self) -> None:
        files = parse_diff_files(DIFF_DELETED)
        assert detect_commit_type(files) == "refactor"

    def test_hint_fix(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        assert detect_commit_type(files, hint="fix null pointer bug") == "fix"

    def test_hint_crash(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        assert detect_commit_type(files, hint="crash on startup") == "fix"

    def test_hint_error(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        assert detect_commit_type(files, hint="handle error in loader") == "fix"

    def test_hint_refactor(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        assert detect_commit_type(files, hint="refactor config module") == "refactor"

    def test_hint_perf(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        assert detect_commit_type(files, hint="improve performance") == "perf"

    def test_hint_style(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        assert detect_commit_type(files, hint="format code style") == "style"

    def test_hint_revert(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        assert detect_commit_type(files, hint="revert last merge") == "revert"

    def test_empty_files(self) -> None:
        assert detect_commit_type([]) == "chore"

    def test_markdown_root_is_docs(self) -> None:
        files = parse_diff_files(DIFF_MARKDOWN_ROOT)
        assert detect_commit_type(files) == "docs"

    def test_test_file_patterns(self) -> None:
        """Various test-file naming conventions are recognised."""
        test_paths = [
            "src/auth_test.go",
            "lib/parser.test.js",
            "test/integration/api.py",
            "spec/models/user_spec.rb",
        ]
        for path in test_paths:
            files = [{"path": path, "status": "modified", "insertions": 1, "deletions": 0}]
            assert detect_commit_type(files) == "test", f"Expected 'test' for {path}"


# ===================================================================
# detect_scope
# ===================================================================


class TestDetectScope:
    def test_scope_from_directory(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        assert detect_scope(files) == "config"

    def test_scope_from_auth(self) -> None:
        files = parse_diff_files(DIFF_NEW_FEAT)
        assert detect_scope(files) == "auth"

    def test_no_scope_single_segment(self) -> None:
        """Files at root level should return None."""
        files = [{"path": "README.md", "status": "modified", "insertions": 1, "deletions": 0}]
        assert detect_scope(files) is None

    def test_empty_files(self) -> None:
        assert detect_scope([]) is None

    def test_cmd_mapped_to_cli(self) -> None:
        files = [{"path": "cmd/teamwork/cmd/root.go", "status": "modified", "insertions": 1, "deletions": 0}]
        scope = detect_scope(files)
        assert scope == "cli"


# ===================================================================
# validate_message
# ===================================================================


class TestValidateMessage:
    def test_valid_simple(self) -> None:
        result = validate_message("feat: add login endpoint")
        assert result["valid"] is True
        assert result["type"] == "feat"
        assert result["scope"] is None
        assert result["errors"] == []

    def test_valid_with_scope(self) -> None:
        result = validate_message("fix(auth): handle expired tokens")
        assert result["valid"] is True
        assert result["type"] == "fix"
        assert result["scope"] == "auth"

    def test_valid_with_body(self) -> None:
        msg = "docs: update readme\n\nAdded installation instructions."
        result = validate_message(msg)
        assert result["valid"] is True

    def test_valid_breaking_bang(self) -> None:
        result = validate_message("feat(api)!: remove deprecated v1 endpoints")
        assert result["valid"] is True
        assert result["type"] == "feat"
        assert result["scope"] == "api"

    def test_valid_breaking_footer(self) -> None:
        msg = "feat: new API\n\nSome body.\n\nBREAKING CHANGE: removed old endpoints"
        result = validate_message(msg)
        assert result["valid"] is True

    def test_invalid_uppercase_subject(self) -> None:
        result = validate_message("feat: Add login endpoint")
        assert result["valid"] is False
        assert any("lowercase" in e for e in result["errors"])

    def test_invalid_trailing_period(self) -> None:
        result = validate_message("feat: add login endpoint.")
        assert result["valid"] is False
        assert any("period" in e for e in result["errors"])

    def test_invalid_no_type(self) -> None:
        result = validate_message("added a new feature")
        assert result["valid"] is False
        assert result["type"] is None

    def test_invalid_unknown_type(self) -> None:
        result = validate_message("yolo: do something")
        assert result["valid"] is False
        assert any("Unknown commit type" in e for e in result["errors"])

    def test_invalid_missing_colon(self) -> None:
        result = validate_message("feat add something")
        assert result["valid"] is False
        assert any("colon" in e.lower() for e in result["errors"])

    def test_invalid_body_no_blank_line(self) -> None:
        msg = "feat: add feature\nsome body text"
        result = validate_message(msg)
        assert result["valid"] is False
        assert any("blank line" in e for e in result["errors"])

    def test_invalid_long_header(self) -> None:
        long_subject = "a" * 70
        result = validate_message(f"feat: {long_subject}")
        assert result["valid"] is False
        assert any("72" in e for e in result["errors"])

    def test_both_breaking_indicators(self) -> None:
        msg = "feat!: new api\n\nBREAKING CHANGE: old api removed"
        result = validate_message(msg)
        assert result["valid"] is True
        assert any("both" in s.lower() for s in result["suggestions"])


# ===================================================================
# list_commit_types
# ===================================================================


class TestListCommitTypes:
    def test_all_types_present(self) -> None:
        expected = {"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"}
        actual = {t["type"] for t in COMMIT_TYPES}
        assert actual == expected

    def test_semver_impact_values(self) -> None:
        valid_impacts = {"major", "minor", "patch", "none"}
        for t in COMMIT_TYPES:
            assert t["semver_impact"] in valid_impacts, f"{t['type']} has invalid semver_impact"

    def test_feat_is_minor(self) -> None:
        feat = next(t for t in COMMIT_TYPES if t["type"] == "feat")
        assert feat["semver_impact"] == "minor"

    def test_fix_is_patch(self) -> None:
        fix = next(t for t in COMMIT_TYPES if t["type"] == "fix")
        assert fix["semver_impact"] == "patch"

    def test_docs_is_none(self) -> None:
        docs = next(t for t in COMMIT_TYPES if t["type"] == "docs")
        assert docs["semver_impact"] == "none"

    def test_all_have_description(self) -> None:
        for t in COMMIT_TYPES:
            assert t["description"], f"{t['type']} is missing a description"


# ===================================================================
# generate_subject
# ===================================================================


class TestGenerateSubject:
    def test_with_hint(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        subject = generate_subject(files, hint="Reload configuration on SIGHUP")
        assert subject == "reload configuration on SIGHUP"
        assert not subject[0].isupper()

    def test_hint_trailing_period_stripped(self) -> None:
        subject = generate_subject([], hint="Fix the bug.")
        assert not subject.endswith(".")

    def test_single_new_file(self) -> None:
        files = parse_diff_files(DIFF_NEW_FEAT)
        subject = generate_subject(files)
        assert "oauth.go" in subject

    def test_single_deleted_file(self) -> None:
        files = parse_diff_files(DIFF_DELETED)
        subject = generate_subject(files)
        assert "remove" in subject

    def test_multiple_files_scope(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        subject = generate_subject(files)
        assert "config" in subject or "update" in subject

    def test_empty_files(self) -> None:
        assert generate_subject([]) == "update project files"

    def test_long_hint_truncated(self) -> None:
        long_hint = "a" * 100
        subject = generate_subject([], hint=long_hint)
        assert len(subject) <= 68


# ===================================================================
# detect_breaking_change
# ===================================================================


class TestDetectBreakingChange:
    def test_breaking_in_hint(self) -> None:
        assert detect_breaking_change("", hint="breaking change to API") is True

    def test_breaking_in_diff(self) -> None:
        assert detect_breaking_change(DIFF_BREAKING) is True

    def test_no_breaking(self) -> None:
        assert detect_breaking_change(DIFF_MULTI_FILES) is False

    def test_breaking_change_footer_string(self) -> None:
        diff_with_footer = "some diff\nBREAKING CHANGE: removed old interface"
        assert detect_breaking_change(diff_with_footer) is True

    def test_breaking_hyphen_variant(self) -> None:
        diff_with_footer = "some diff\nBREAKING-CHANGE: removed old interface"
        assert detect_breaking_change(diff_with_footer) is True


# ===================================================================
# build_commit_message
# ===================================================================


class TestBuildCommitMessage:
    def test_simple(self) -> None:
        msg = build_commit_message("feat", None, "add new endpoint")
        assert msg == "feat: add new endpoint"

    def test_with_scope(self) -> None:
        msg = build_commit_message("fix", "auth", "handle expired tokens")
        assert msg == "fix(auth): handle expired tokens"

    def test_breaking(self) -> None:
        msg = build_commit_message("feat", "api", "remove v1", breaking_change=True)
        assert msg == "feat(api)!: remove v1"

    def test_with_body(self) -> None:
        msg = build_commit_message("feat", None, "add login", body="Detailed description.")
        assert "feat: add login" in msg
        assert "\n\nDetailed description." in msg

    def test_with_footer(self) -> None:
        msg = build_commit_message("feat", None, "new api", footer="BREAKING CHANGE: old api removed")
        assert msg.endswith("BREAKING CHANGE: old api removed")

    def test_full_message(self) -> None:
        msg = build_commit_message(
            "feat",
            "api",
            "add v2 endpoints",
            body="Added new REST endpoints for v2.",
            breaking_change=True,
            footer="BREAKING CHANGE: v1 endpoints removed",
        )
        lines = msg.split("\n")
        assert lines[0] == "feat(api)!: add v2 endpoints"
        assert lines[1] == ""
        assert lines[2] == "Added new REST endpoints for v2."
        assert lines[3] == ""
        assert lines[4] == "BREAKING CHANGE: v1 endpoints removed"


# ===================================================================
# generate_pr_description_text
# ===================================================================


class TestGeneratePrDescription:
    def test_contains_expected_sections(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        result = generate_pr_description_text(files, DIFF_MULTI_FILES)
        assert "## " in result  # title
        assert "### Summary" in result
        assert "### Motivation" in result
        assert "### Changes Made" in result
        assert "### Test Plan" in result
        assert "### Breaking Changes" in result

    def test_files_listed(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        result = generate_pr_description_text(files, DIFF_MULTI_FILES)
        assert "loader.go" in result
        assert "types.go" in result

    def test_stats_in_summary(self) -> None:
        files = parse_diff_files(DIFF_MULTI_FILES)
        result = generate_pr_description_text(files, DIFF_MULTI_FILES)
        assert "2" in result  # 2 files
        assert "insertion" in result

    def test_breaking_change_noted(self) -> None:
        files = parse_diff_files(DIFF_BREAKING)
        result = generate_pr_description_text(files, DIFF_BREAKING)
        assert "BREAKING" in result

    def test_no_breaking_says_none(self) -> None:
        files = parse_diff_files(DIFF_DOC_FILE)
        result = generate_pr_description_text(files, DIFF_DOC_FILE)
        assert "None" in result

    def test_custom_template(self) -> None:
        tmpl = "# {title}\n{summary}\n{changes}"
        files = parse_diff_files(DIFF_DOC_FILE)
        result = generate_pr_description_text(files, DIFF_DOC_FILE, template=tmpl)
        assert result.startswith("# ")
        assert "guide.md" in result
