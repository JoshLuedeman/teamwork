package validate_test

import (
	"strings"
	"os"
	"path/filepath"
	"testing"

	"github.com/joshluedeman/teamwork/internal/validate"
)

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func validConfig() string {
	return `project:
  name: "test-project"
  repo: "owner/repo"
roles:
  core:
    - planner
    - coder
`
}

func validState() string {
	return `id: "feature/42"
type: "feature"
status: "active"
current_step: 1
created_at: "2025-01-01T00:00:00Z"
`
}

func countPassed(results []validate.Result) int {
	n := 0
	for _, r := range results {
		if r.Passed {
			n++
		}
	}
	return n
}

func countFailed(results []validate.Result) int {
	n := 0
	for _, r := range results {
		if !r.Passed {
			n++
		}
	}
	return n
}

func failedMessages(results []validate.Result) []string {
	var msgs []string
	for _, r := range results {
		if !r.Passed {
			msgs = append(msgs, r.Message)
		}
	}
	return msgs
}

// TestRun_MissingTeamworkDir returns error when .teamwork/ doesn't exist.
func TestRun_MissingTeamworkDir(t *testing.T) {
	dir := t.TempDir()
	_, err := validate.Run(dir)
	if err == nil {
		t.Fatal("expected error when .teamwork/ is missing, got nil")
	}
}

// TestRun_ValidMinimalSetup passes with valid config and empty subdirs.
func TestRun_ValidMinimalSetup(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "handoffs"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "memory"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if countFailed(results) != 0 {
		t.Errorf("expected 0 failures, got: %v", failedMessages(results))
	}
}

// TestRun_MissingConfig fails when config.yaml is absent.
func TestRun_MissingConfig(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if countFailed(results) == 0 {
		t.Error("expected failure for missing config.yaml")
	}
}

// TestRun_InvalidConfigYAML fails when config.yaml has malformed YAML.
func TestRun_InvalidConfigYAML(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", ":\tinvalid: yaml: [}")

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := failedMessages(results)
	if len(msgs) == 0 {
		t.Error("expected failure for invalid YAML config")
	}
}

// TestRun_ConfigMissingProjectName fails when project.name is empty.
func TestRun_ConfigMissingProjectName(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", `project:
  name: ""
  repo: "owner/repo"
roles:
  core:
    - coder
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := failedMessages(results)
	found := false
	for _, m := range msgs {
		if contains(m, "project.name") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected failure mentioning project.name, got: %v", msgs)
	}
}

// TestRun_ConfigEmptyRolesCore fails when roles.core is empty.
func TestRun_ConfigEmptyRolesCore(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", `project:
  name: "test"
  repo: "owner/repo"
roles:
  core: []
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := failedMessages(results)
	found := false
	for _, m := range msgs {
		if contains(m, "roles.core") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected failure mentioning roles.core, got: %v", msgs)
	}
}

// TestRun_ValidStateFile passes for a valid state file.
func TestRun_ValidStateFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "handoffs"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "memory"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())
	writeFile(t, dir, ".teamwork/state/feature-42.yaml", validState())

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if countFailed(results) != 0 {
		t.Errorf("expected 0 failures, got: %v", failedMessages(results))
	}
}

// TestRun_StateInvalidStatus fails for an unknown status value.
func TestRun_StateInvalidStatus(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "handoffs"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "memory"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())
	writeFile(t, dir, ".teamwork/state/bad.yaml", `id: "x"
type: "feature"
status: "running"
current_step: 0
created_at: "2025-01-01"
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := failedMessages(results)
	found := false
	for _, m := range msgs {
		if contains(m, "running") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected failure mentioning invalid status 'running', got: %v", msgs)
	}
}

// TestRun_StateAllValidStatuses passes for each valid status.
func TestRun_StateAllValidStatuses(t *testing.T) {
	statuses := []string{"active", "blocked", "completed", "failed", "cancelled"}
	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			dir := t.TempDir()
			os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755)
			os.MkdirAll(filepath.Join(dir, ".teamwork", "handoffs"), 0o755)
			os.MkdirAll(filepath.Join(dir, ".teamwork", "memory"), 0o755)
			writeFile(t, dir, ".teamwork/config.yaml", validConfig())
			writeFile(t, dir, ".teamwork/state/w.yaml", `id: "x"
type: "feature"
status: "`+status+`"
current_step: 0
created_at: "2025-01-01"
`)
			results, err := validate.Run(dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if countFailed(results) != 0 {
				t.Errorf("status %q: expected 0 failures, got: %v", status, failedMessages(results))
			}
		})
	}
}

// TestRun_StateMissingID fails when id is absent.
func TestRun_StateMissingID(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "handoffs"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "memory"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())
	writeFile(t, dir, ".teamwork/state/noid.yaml", `type: "feature"
status: "active"
current_step: 0
created_at: "2025-01-01"
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if countFailed(results) == 0 {
		t.Error("expected failure for missing id field")
	}
}

// TestRun_EmptyHandoffFails fails for a zero-byte handoff file.
func TestRun_EmptyHandoffFails(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "handoffs", "feature-42"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "memory"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())
	writeFile(t, dir, ".teamwork/handoffs/feature-42/01-planner.md", "")

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := failedMessages(results)
	found := false
	for _, m := range msgs {
		if contains(m, "empty") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected failure for empty handoff, got: %v", msgs)
	}
}

// TestRun_NonEmptyHandoffPasses passes for a handoff with content.
func TestRun_NonEmptyHandoffPasses(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "handoffs", "feature-42"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "memory"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())
	writeFile(t, dir, ".teamwork/handoffs/feature-42/01-planner.md", "# Handoff\nContent here.")

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if countFailed(results) != 0 {
		t.Errorf("expected 0 failures, got: %v", failedMessages(results))
	}
}

// TestRun_EmptyMemoryFilePasses passes for a zero-byte memory file (created by init).
func TestRun_EmptyMemoryFilePasses(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "handoffs"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "memory"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())
	writeFile(t, dir, ".teamwork/memory/patterns.yaml", "")

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if countFailed(results) != 0 {
		t.Errorf("expected 0 failures for empty memory file, got: %v", failedMessages(results))
	}
}

// TestRun_InvalidMemoryYAMLFails fails for a memory file with invalid YAML.
func TestRun_InvalidMemoryYAMLFails(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "handoffs"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".teamwork", "memory"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())
	writeFile(t, dir, ".teamwork/memory/patterns.yaml", ":\tbad: [yaml}")

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if countFailed(results) == 0 {
		t.Error("expected failure for invalid memory YAML")
	}
}

// writeMCPTestConfig writes a config.yaml with MCP content into dir/.teamwork/.
func writeMCPTestConfig(t *testing.T, dir string, configYAML string) {
	t.Helper()
	twDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(twDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte(configYAML), 0o644); err != nil {
		t.Fatal(err)
	}
}

// filterMCPResults returns only results with Check == "mcp_servers".
func filterMCPResults(results []validate.Result) []validate.Result {
	var out []validate.Result
	for _, r := range results {
		if r.Check == "mcp_servers" {
			out = append(out, r)
		}
	}
	return out
}

func TestCheckMCPConfig_NoSection(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	if len(mcpResults) != 0 {
		t.Errorf("expected no MCP results, got %d: %+v", len(mcpResults), mcpResults)
	}
}

func TestCheckMCPConfig_EmptySection(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers: {}
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	for _, r := range mcpResults {
		if !r.Passed {
			t.Errorf("expected all MCP results to pass, got failure: %s", r.Message)
		}
	}
}

func TestCheckMCPConfig_ValidURL(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  test-server:
    description: "Test server"
    url: "https://example.com/mcp"
    roles: [coder]
    env_vars: []
    install: "npm install test"
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	if len(mcpResults) == 0 {
		t.Fatal("expected MCP results, got none")
	}
	for _, r := range mcpResults {
		if !r.Passed {
			t.Errorf("expected pass, got failure: %s", r.Message)
		}
	}
}

func TestCheckMCPConfig_ValidCommand(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  local-tool:
    description: "Local tool"
    command: "npx @modelcontextprotocol/server"
    roles: [coder]
    env_vars: []
    install: "npm install tool"
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	if len(mcpResults) == 0 {
		t.Fatal("expected MCP results, got none")
	}
	for _, r := range mcpResults {
		if !r.Passed {
			t.Errorf("expected pass, got failure: %s", r.Message)
		}
	}
}

func TestCheckMCPConfig_MissingURLAndCommand(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  broken:
    description: "No transport"
    roles: [coder]
    env_vars: []
    install: ""
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	if countFailed(mcpResults) == 0 {
		t.Error("expected failure for missing url and command")
	}
}

func TestCheckMCPConfig_BothURLAndCommand(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  both:
    description: "Has both"
    url: "https://example.com/mcp"
    command: "npx server"
    roles: [coder]
    env_vars: []
    install: ""
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	if countFailed(mcpResults) == 0 {
		t.Error("expected failure for having both url and command")
	}
}

func TestCheckMCPConfig_InvalidURL(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  bad-url:
    description: "Bad URL"
    url: "not-a-url"
    roles: [coder]
    env_vars: []
    install: ""
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	if countFailed(mcpResults) == 0 {
		t.Error("expected failure for invalid URL")
	}
}

func TestCheckMCPConfig_UnknownRole(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  wizard-server:
    description: "Unknown role"
    url: "https://example.com/mcp"
    roles: [wizard]
    env_vars: []
    install: ""
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	if countFailed(mcpResults) == 0 {
		t.Error("expected failure for unknown role")
	}
	msgs := failedMessages(mcpResults)
	found := false
	for _, m := range msgs {
		if contains(m, "invalid role") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected message mentioning 'invalid role', got: %v", msgs)
	}
}

func TestCheckMCPConfig_MissingDescription(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  no-desc:
    url: "https://example.com/mcp"
    roles: [coder]
    env_vars: []
    install: ""
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	if countFailed(mcpResults) == 0 {
		t.Error("expected failure for missing description")
	}
}

func TestCheckMCPConfig_MissingEnvVar(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  env-server:
    description: "Env test"
    url: "https://example.com/mcp"
    roles: [coder]
    env_vars: [TAVILY_API_KEY]
    install: ""
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	// Should pass (warning, not failure)
	if countFailed(mcpResults) != 0 {
		t.Errorf("expected no failures for missing env var (warning only), got: %v", failedMessages(mcpResults))
	}
	// Should contain a WARN message
	foundWarn := false
	for _, r := range mcpResults {
		if contains(r.Message, "WARN") && contains(r.Message, "TAVILY_API_KEY") {
			foundWarn = true
		}
	}
	if !foundWarn {
		t.Error("expected a WARN message mentioning TAVILY_API_KEY")
	}
}

func TestCheckMCPConfig_SetEnvVar(t *testing.T) {
	t.Setenv("TAVILY_API_KEY", "test-value")
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  env-server:
    description: "Env test"
    url: "https://example.com/mcp"
    roles: [coder]
    env_vars: [TAVILY_API_KEY]
    install: ""
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	for _, r := range mcpResults {
		if contains(r.Message, "WARN") {
			t.Errorf("expected no WARN when env var is set, got: %s", r.Message)
		}
	}
}

func TestCheckMCPConfig_MultipleServers(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  server-a:
    description: "Server A"
    url: "https://a.example.com/mcp"
    roles: [coder]
    env_vars: []
    install: ""
  server-b:
    description: "Server B"
    url: "https://b.example.com/mcp"
    roles: [tester]
    env_vars: []
    install: ""
  server-c:
    description: "Server C"
    command: "npx server-c"
    roles: [reviewer]
    env_vars: []
    install: ""
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	if countFailed(mcpResults) != 0 {
		t.Errorf("expected no failures, got: %v", failedMessages(mcpResults))
	}
	// Check for summary mentioning 3 servers
	foundSummary := false
	for _, r := range mcpResults {
		if contains(r.Message, "3 servers configured") {
			foundSummary = true
		}
	}
	if !foundSummary {
		var msgs []string
		for _, r := range mcpResults {
			msgs = append(msgs, r.Message)
		}
		t.Errorf("expected summary mentioning '3 servers configured', got: %v", msgs)
	}
}

func TestCheckMCPConfig_MixedValid(t *testing.T) {
	dir := t.TempDir()
	writeMCPTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  good-server:
    description: "Good server"
    url: "https://example.com/mcp"
    roles: [coder]
    env_vars: []
    install: ""
  bad-server:
    url: "https://example.com/mcp"
    roles: [coder]
    env_vars: []
    install: ""
`)
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mcpResults := filterMCPResults(results)
	if countFailed(mcpResults) == 0 {
		t.Error("expected failure for the server missing description")
	}
	// The valid server should still produce a pass
	if countPassed(mcpResults) == 0 {
		t.Error("expected at least one pass for the valid server")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

// --- Agent file validation tests ---

// validAgentFile returns a minimal valid agent file with all required parts.
func validAgentFile(name, tier string) string {
	return `---
name: ` + name + `
description: Test agent
---

# Role: Test

## Identity

You are a test agent.

## Model Requirements

- **Tier:** ` + tier + `
- **Why:** Test.
- **Key capabilities needed:** Testing

## Responsibilities

- Do things

## Boundaries

- ✅ **Always:** Do the right thing
`
}

// writeAgentFile writes an agent file into the test directory's .github/agents/.
func writeAgentFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	agentDir := filepath.Join(dir, ".github", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", agentDir, err)
	}
	path := filepath.Join(agentDir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// filterAgentResults returns only results with an agent-related Check.
func filterAgentResults(results []validate.Result) []validate.Result {
	var out []validate.Result
	for _, r := range results {
		if contains(r.Check, "agent_") {
			out = append(out, r)
		}
	}
	return out
}

func filterCustomWorkflowResults(results []validate.Result) []validate.Result {
	var out []validate.Result
	for _, r := range results {
		if r.Check == "custom_workflows" { out = append(out, r) }
	}
	return out
}

func TestAgentFile_ValidFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())
	writeAgentFile(t, dir, "coder.agent.md", validAgentFile("coder", "Premium"))

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agentResults := filterAgentResults(results)
	if len(agentResults) == 0 {
		t.Fatal("expected agent results, got none")
	}
	for _, r := range agentResults {
		if !r.Passed {
			t.Errorf("expected pass, got failure: %s", r.Message)
		}
	}
}

func TestAgentFile_CUSTOMIZEPlaceholder(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())

	content := `---
name: coder
description: Test agent
---

# Role: Coder

## Identity

You are the Coder.

## Project Knowledge
<!-- CUSTOMIZE: Replace the placeholders below with your project's details -->
- **Tech Stack:** [e.g., React 18]

## Model Requirements

- **Tier:** Premium
- **Why:** Test

## Responsibilities

- Do things

## Boundaries

- Do things
`
	writeAgentFile(t, dir, "coder.agent.md", content)

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agentResults := filterAgentResults(results)

	// Should have no failures (CUSTOMIZE is a warning, not a failure).
	if countFailed(agentResults) != 0 {
		t.Errorf("expected no failures for CUSTOMIZE placeholder (warning only), got: %v", failedMessages(agentResults))
	}

	// Should contain a WARN message about CUSTOMIZE.
	foundWarn := false
	for _, r := range agentResults {
		if contains(r.Message, "WARN") && contains(r.Message, "CUSTOMIZE") {
			foundWarn = true
		}
	}
	if !foundWarn {
		t.Error("expected a WARN message mentioning CUSTOMIZE")
	}
}

func TestAgentFile_MissingSections(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())

	// File with no required sections at all.
	content := `---
name: coder
description: Missing sections
---

# Role: Coder

## Model Requirements

- **Tier:** Premium

Some content but no Identity, Responsibilities, or Boundaries sections.
`
	writeAgentFile(t, dir, "coder.agent.md", content)

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agentResults := filterAgentResults(results)

	// Should have a failure for missing sections.
	msgs := failedMessages(agentResults)
	found := false
	for _, m := range msgs {
		if contains(m, "missing required sections") {
			found = true
			// Should mention Identity, Responsibilities, and Boundaries.
			if !contains(m, "Identity") {
				t.Errorf("expected missing section 'Identity' mentioned, got: %s", m)
			}
			if !contains(m, "Responsibilities") {
				t.Errorf("expected missing section 'Responsibilities' mentioned, got: %s", m)
			}
			if !contains(m, "Boundaries") {
				t.Errorf("expected missing section 'Boundaries' mentioned, got: %s", m)
			}
		}
	}
	if !found {
		t.Errorf("expected failure mentioning 'missing required sections', got: %v", msgs)
	}
}

func TestAgentFile_UnknownRole(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())
	writeAgentFile(t, dir, "wizard.agent.md", validAgentFile("wizard", "Premium"))

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agentResults := filterAgentResults(results)

	msgs := failedMessages(agentResults)
	found := false
	for _, m := range msgs {
		if contains(m, "unknown role") && contains(m, "wizard") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected failure mentioning unknown role 'wizard', got: %v", msgs)
	}
}

func TestAgentFile_InvalidModelTier(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())
	writeAgentFile(t, dir, "coder.agent.md", validAgentFile("coder", "Ultra"))

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agentResults := filterAgentResults(results)

	msgs := failedMessages(agentResults)
	found := false
	for _, m := range msgs {
		if contains(m, "invalid model tier") && contains(m, "Ultra") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected failure mentioning invalid model tier 'Ultra', got: %v", msgs)
	}
}

func TestAgentFile_MissingFrontmatterName(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())

	content := `---
description: No name field
---

# Role: Mystery

## Identity

You are nobody.

## Model Requirements

- **Tier:** Standard

## Responsibilities

- Nothing

## Boundaries

- None
`
	writeAgentFile(t, dir, "mystery.agent.md", content)

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agentResults := filterAgentResults(results)

	msgs := failedMessages(agentResults)
	found := false
	for _, m := range msgs {
		if contains(m, "missing or empty") && contains(m, "name") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected failure mentioning missing 'name', got: %v", msgs)
	}
}

func TestAgentFile_MissingModelTier(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())

	content := `---
name: coder
description: No tier
---

# Role: Coder

## Identity

You are the Coder.

## Model Requirements

No tier line here.

## Responsibilities

- Do things

## Boundaries

- Do things
`
	writeAgentFile(t, dir, "coder.agent.md", content)

	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agentResults := filterAgentResults(results)

	msgs := failedMessages(agentResults)
	found := false
	for _, m := range msgs {
		if contains(m, "no model tier found") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected failure mentioning 'no model tier found', got: %v", msgs)
	}
}

func TestAgentFile_NoAgentsDir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
	writeFile(t, dir, ".teamwork/config.yaml", validConfig())

	// No .github/agents/ directory at all — should pass silently.
	results, err := validate.Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agentResults := filterAgentResults(results)
	if len(agentResults) != 0 {
		t.Errorf("expected no agent results when directory is absent, got %d: %+v", len(agentResults), agentResults)
	}
}

func TestAgentFile_AllValidTiers(t *testing.T) {
	tiers := []string{"Premium", "Standard", "Fast"}
	for _, tier := range tiers {
		t.Run(tier, func(t *testing.T) {
			dir := t.TempDir()
			os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755)
			writeFile(t, dir, ".teamwork/config.yaml", validConfig())
			writeAgentFile(t, dir, "coder.agent.md", validAgentFile("coder", tier))

			results, err := validate.Run(dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			agentResults := filterAgentResults(results)
			for _, r := range agentResults {
				if !r.Passed {
					t.Errorf("tier %q: expected pass, got failure: %s", tier, r.Message)
				}
			}
		})
	}
}

func TestRun_ValidCustomWorkflows(t *testing.T) {
	dir := t.TempDir()
	twDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(twDir, 0o755); err != nil { t.Fatal(err) }
	cfg := "model:\n  provider: openai\n  name: gpt-4\ncustom_workflows:\n  data-pipeline:\n    steps:\n      - role: planner\n        description: Design\n      - role: coder\n        description: Build\n"
	if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte(cfg), 0o644); err != nil { t.Fatal(err) }
	results, _ := validate.Run(dir)
	cwResults := filterCustomWorkflowResults(results)
	if len(cwResults) == 0 { t.Fatal("expected results") }
	for _, r := range cwResults {
		if !r.Passed { t.Errorf("expected pass: %s", r.Message) }
	}
}

func TestRun_CustomWorkflowConflictsWithBuiltin(t *testing.T) {
	dir := t.TempDir()
	twDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(twDir, 0o755); err != nil { t.Fatal(err) }
	cfg := "model:\n  provider: openai\n  name: gpt-4\ncustom_workflows:\n  feature:\n    steps:\n      - role: coder\n        description: Override\n"
	if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte(cfg), 0o644); err != nil { t.Fatal(err) }
	results, _ := validate.Run(dir)
	cwResults := filterCustomWorkflowResults(results)
	found := false
	for _, r := range cwResults {
		if !r.Passed && strings.Contains(r.Message, "conflicts with built-in") { found = true }
	}
	if !found { t.Error("expected conflict") }
}

func TestRun_CustomWorkflowEmptySteps(t *testing.T) {
	dir := t.TempDir()
	twDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(twDir, 0o755); err != nil { t.Fatal(err) }
	cfg := "model:\n  provider: openai\n  name: gpt-4\ncustom_workflows:\n  empty-wf:\n    steps: []\n"
	if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte(cfg), 0o644); err != nil { t.Fatal(err) }
	results, _ := validate.Run(dir)
	cwResults := filterCustomWorkflowResults(results)
	found := false
	for _, r := range cwResults {
		if !r.Passed && strings.Contains(r.Message, "at least one step") { found = true }
	}
	if !found { t.Error("expected error for empty steps") }
}

func TestRun_CustomWorkflowMissingRole(t *testing.T) {
	dir := t.TempDir()
	twDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(twDir, 0o755); err != nil { t.Fatal(err) }
	cfg := "model:\n  provider: openai\n  name: gpt-4\ncustom_workflows:\n  bad-wf:\n    steps:\n      - description: missing role\n"
	if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte(cfg), 0o644); err != nil { t.Fatal(err) }
	results, _ := validate.Run(dir)
	cwResults := filterCustomWorkflowResults(results)
	found := false
	for _, r := range cwResults {
		if !r.Passed && strings.Contains(r.Message, "role") { found = true }
	}
	if !found { t.Error("expected error for missing role") }
}

func TestRun_NoCustomWorkflowsSection(t *testing.T) {
	dir := t.TempDir()
	twDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(twDir, 0o755); err != nil { t.Fatal(err) }
	cfg := "model:\n  provider: openai\n  name: gpt-4\n"
	if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte(cfg), 0o644); err != nil { t.Fatal(err) }
	results, _ := validate.Run(dir)
	cwResults := filterCustomWorkflowResults(results)
	if len(cwResults) != 0 { t.Errorf("got %d, want 0", len(cwResults)) }
}
