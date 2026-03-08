#!/usr/bin/env bash
# E2E Test: GitHub App Auto-Install
#
# Prerequisites:
#   - GitHub App registered and installed on your account
#   - Cloudflare Worker deployed and configured
#   - gh CLI authenticated
#
# Usage:
#   ./test-auto-install.sh [test-repo-name]

set -euo pipefail

REPO_NAME="${1:-teamwork-e2e-test-$(date +%s)}"
OWNER=$(gh api user --jq '.login')
MAX_WAIT=60
POLL_INTERVAL=5

echo "=== Teamwork Auto-Install E2E Test ==="
echo ""
echo "Owner: $OWNER"
echo "Test repo: $REPO_NAME"
echo ""

# Step 1: Create test repo
echo "Step 1: Creating private test repo..."
gh repo create "$REPO_NAME" --private --description "Teamwork E2E test — safe to delete" 2>&1
echo "  ✓ Created $OWNER/$REPO_NAME"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up: deleting $OWNER/$REPO_NAME..."
    gh repo delete "$OWNER/$REPO_NAME" --yes 2>/dev/null || echo "  ⚠ Could not delete repo (may need manual cleanup)"
}
trap cleanup EXIT

# Step 2: Wait for framework files
echo "Step 2: Waiting for auto-install (max ${MAX_WAIT}s)..."
elapsed=0
installed=false

while [ $elapsed -lt $MAX_WAIT ]; do
    sleep $POLL_INTERVAL
    elapsed=$((elapsed + POLL_INTERVAL))

    # Check if .github/agents directory exists
    if gh api "repos/$OWNER/$REPO_NAME/contents/.github/agents" --silent 2>/dev/null; then
        installed=true
        break
    fi
    echo "  Polling... (${elapsed}s elapsed)"
done

if [ "$installed" = false ]; then
    echo ""
    echo "  ✗ FAIL: Framework files not found after ${MAX_WAIT}s"
    echo ""
    echo "  Troubleshooting:"
    echo "    - Check webhook deliveries: GitHub App settings → Advanced"
    echo "    - Check Worker logs: wrangler tail"
    echo "    - Ensure Worker is deployed and secrets are configured"
    exit 1
fi

echo "  ✓ Framework files detected (${elapsed}s)"
echo ""

# Step 3: Verify framework files
echo "Step 3: Verifying framework files..."
ERRORS=0

check_path() {
    local path="$1"
    if gh api "repos/$OWNER/$REPO_NAME/contents/$path" --silent 2>/dev/null; then
        echo "  ✓ $path"
    else
        echo "  ✗ $path (missing)"
        ERRORS=$((ERRORS + 1))
    fi
}

check_path ".github/agents"
check_path ".github/skills"
check_path ".github/copilot-instructions.md"
check_path "docs"
check_path "Makefile"
check_path "MEMORY.md"
check_path "CHANGELOG.md"
echo ""

# Step 4: Verify commit author
echo "Step 4: Verifying commit author..."
AUTHOR=$(gh api "repos/$OWNER/$REPO_NAME/commits" --jq '.[0].commit.author.name' 2>/dev/null || echo "unknown")
echo "  Commit author: $AUTHOR"

# GitHub App commits show as the App name or "teamwork-installer[bot]"
if echo "$AUTHOR" | grep -qi "teamwork\|bot\|app"; then
    echo "  ✓ Commit appears to be from GitHub App"
else
    echo "  ⚠ Commit author may not be the GitHub App (got: $AUTHOR)"
    echo "    This is acceptable if the App name doesn't contain 'teamwork'"
fi
echo ""

# Results
echo "=== Results ==="
if [ $ERRORS -eq 0 ]; then
    echo "  ✓ PASS: All framework files present"
    exit 0
else
    echo "  ✗ FAIL: $ERRORS file(s) missing"
    exit 1
fi
