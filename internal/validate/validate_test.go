package validate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JoshLuedeman/teamwork/internal/validate"
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
