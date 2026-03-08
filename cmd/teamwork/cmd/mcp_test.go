package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func writeTestConfig(t *testing.T, dir string, yaml string) {
	t.Helper()
	twDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(twDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

const threeServerConfig = `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers:
  github:
    description: "GitHub operations"
    url: "https://api.githubcopilot.com/mcp/"
    roles: [coder, tester]
    env_vars: [GH_TOKEN]
    install: "gh extension install github/gh-mcp"
  semgrep:
    description: "Security scanning"
    command: "uvx semgrep-mcp"
    roles: [security-auditor, coder]
    env_vars: [SEMGREP_APP_TOKEN]
    install: "pip install semgrep-mcp"
  osv:
    description: "Vulnerability lookup"
    command: "uvx osv-mcp"
    roles: [security-auditor]
    env_vars: []
    install: "pip install osv-mcp"
`

// executeMCPCmd runs a subcommand of "teamwork mcp" and captures stdout.
func executeMCPCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	// Reset flag state to avoid pollution between tests.
	for _, c := range mcpCmd.Commands() {
		c.Flags().VisitAll(func(f *pflag.Flag) {
			f.Value.Set(f.DefValue)
			f.Changed = false
		})
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"mcp", "--dir", dir}, args...))

	err := rootCmd.Execute()
	return buf.String(), err
}

// --- list subcommand tests ---

func TestMCPList_ShowsAllServers(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, threeServerConfig)

	out, err := executeMCPCmd(t, dir, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range []string{"github", "semgrep", "osv"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected output to contain %q, got:\n%s", name, out)
		}
	}
}

func TestMCPList_EnvVarSet(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, threeServerConfig)
	t.Setenv("GH_TOKEN", "test-token")

	out, err := executeMCPCmd(t, dir, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The github line should show "✓ set" since GH_TOKEN is set.
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "github") {
			if !strings.Contains(line, "✓ set") {
				t.Errorf("expected github line to contain '✓ set', got: %s", line)
			}
			return
		}
	}
	t.Errorf("github server not found in output:\n%s", out)
}

func TestMCPList_EnvVarMissing(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, threeServerConfig)
	// Do NOT set SEMGREP_APP_TOKEN.

	out, err := executeMCPCmd(t, dir, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "semgrep") {
			if !strings.Contains(line, "✗ missing") {
				t.Errorf("expected semgrep line to contain '✗ missing', got: %s", line)
			}
			return
		}
	}
	t.Errorf("semgrep server not found in output:\n%s", out)
}

func TestMCPList_NoEnvVars(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, threeServerConfig)

	out, err := executeMCPCmd(t, dir, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "osv") {
			if !strings.Contains(line, "✓ ready") {
				t.Errorf("expected osv line to contain '✓ ready', got: %s", line)
			}
			return
		}
	}
	t.Errorf("osv server not found in output:\n%s", out)
}

func TestMCPList_NoMCPSection(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
`)

	out, err := executeMCPCmd(t, dir, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "No MCP servers configured. See docs/mcp.md for setup instructions."
	if !strings.Contains(out, expected) {
		t.Errorf("expected helpful message, got:\n%s", out)
	}
}

func TestMCPList_MissingConfig(t *testing.T) {
	dir := t.TempDir()
	// No config.yaml written.

	_, err := executeMCPCmd(t, dir, "list")
	if err == nil {
		t.Fatal("expected error for missing config, got nil")
	}

	var exitErr *ExitError
	if e, ok := err.(*ExitError); ok {
		exitErr = e
	} else {
		// Cobra may wrap the error message; check for our message.
		if !strings.Contains(err.Error(), "failed to load config") {
			t.Fatalf("expected ExitError or config error, got: %v", err)
		}
		return
	}
	if exitErr.Code != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.Code)
	}
}

func TestMCPList_RoleFilter(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, threeServerConfig)

	out, err := executeMCPCmd(t, dir, "list", "--role", "coder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// github and semgrep have "coder" in their roles; osv does not.
	if !strings.Contains(out, "github") {
		t.Errorf("expected output to contain 'github'")
	}
	if !strings.Contains(out, "semgrep") {
		t.Errorf("expected output to contain 'semgrep'")
	}
	if strings.Contains(out, "osv") {
		t.Errorf("expected output NOT to contain 'osv', got:\n%s", out)
	}
}

func TestMCPList_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, threeServerConfig)

	out, err := executeMCPCmd(t, dir, "list", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var servers []mcpServerJSON
	if err := json.Unmarshal([]byte(out), &servers); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}

	if len(servers) != 3 {
		t.Errorf("expected 3 servers, got %d", len(servers))
	}
}

// --- config subcommand tests ---

func TestMCPConfig_ClaudeDesktopFormat(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, threeServerConfig)

	out, err := executeMCPCmd(t, dir, "config")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	servers, ok := result["mcpServers"]
	if !ok {
		t.Fatal("expected 'mcpServers' key in output")
	}

	// github should be http type.
	var ghEntry map[string]any
	if err := json.Unmarshal(servers["github"], &ghEntry); err != nil {
		t.Fatalf("parsing github entry: %v", err)
	}
	if ghEntry["type"] != "http" {
		t.Errorf("expected github type 'http', got %v", ghEntry["type"])
	}
	if ghEntry["url"] != "https://api.githubcopilot.com/mcp/" {
		t.Errorf("expected github url, got %v", ghEntry["url"])
	}

	// semgrep should be stdio type with command/args.
	var sgEntry map[string]any
	if err := json.Unmarshal(servers["semgrep"], &sgEntry); err != nil {
		t.Fatalf("parsing semgrep entry: %v", err)
	}
	if sgEntry["type"] != "stdio" {
		t.Errorf("expected semgrep type 'stdio', got %v", sgEntry["type"])
	}
	if sgEntry["command"] != "uvx" {
		t.Errorf("expected semgrep command 'uvx', got %v", sgEntry["command"])
	}
	args, ok := sgEntry["args"].([]any)
	if !ok || len(args) != 1 || args[0] != "semgrep-mcp" {
		t.Errorf("expected semgrep args ['semgrep-mcp'], got %v", sgEntry["args"])
	}
}

func TestMCPConfig_VSCodeFormat(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, threeServerConfig)

	out, err := executeMCPCmd(t, dir, "config", "--format", "vscode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	if _, ok := result["servers"]; !ok {
		t.Fatal("expected 'servers' key in vscode output")
	}
	if _, ok := result["mcpServers"]; ok {
		t.Fatal("unexpected 'mcpServers' key in vscode output")
	}
}

func TestMCPConfig_OnlyReady(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, threeServerConfig)

	// Set GH_TOKEN but not SEMGREP_APP_TOKEN.
	t.Setenv("GH_TOKEN", "test-token")

	out, err := executeMCPCmd(t, dir, "config", "--only-ready")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	servers := result["mcpServers"]

	// github (env set) and osv (no env needed) should be present.
	if _, ok := servers["github"]; !ok {
		t.Error("expected github in output (env var is set)")
	}
	if _, ok := servers["osv"]; !ok {
		t.Error("expected osv in output (no env vars needed)")
	}
	// semgrep should be excluded (env var missing).
	if _, ok := servers["semgrep"]; ok {
		t.Error("expected semgrep to be excluded (env var missing)")
	}
}

func TestMCPConfig_EmptyServers(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, `project:
  name: test
  repo: test/test
roles:
  core: [coder]
mcp_servers: {}
`)

	out, err := executeMCPCmd(t, dir, "config")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	servers := result["mcpServers"]
	if len(servers) != 0 {
		t.Errorf("expected empty mcpServers, got %d entries", len(servers))
	}
}
