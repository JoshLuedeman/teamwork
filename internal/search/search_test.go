package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMemoryFile creates a minimal memory YAML file for testing.
func writeMemoryFile(t *testing.T, dir, name, content string) {
	t.Helper()
	memDir := filepath.Join(dir, ".teamwork", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memDir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeHandoffFile creates a handoff markdown file for testing.
func writeHandoffFile(t *testing.T, dir, workflowID, filename, content string) {
	t.Helper()
	p := filepath.Join(dir, ".teamwork", "handoffs", workflowID, filename)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeADRFile creates an ADR markdown file for testing.
func writeADRFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	adrDir := filepath.Join(dir, "docs", "decisions")
	if err := os.MkdirAll(adrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adrDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeStateFile creates a minimal state YAML file for testing.
func writeStateFile(t *testing.T, dir, workflowID, content string) {
	t.Helper()
	stateDir := filepath.Join(dir, ".teamwork", "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, workflowID+".yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestQuery_EmptyQueryReturnsError(t *testing.T) {
	dir := t.TempDir()
	_, err := Query(dir, "", QueryOptions{})
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestQuery_NoResultsReturnsEmptySlice(t *testing.T) {
	dir := t.TempDir()
	results, err := Query(dir, "nonexistent term xyz", QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestQuery_MatchingMemoryEntry(t *testing.T) {
	dir := t.TempDir()
	writeMemoryFile(t, dir, "patterns.yaml", `entries:
  - id: pattern-001
    domain: [golang, testing]
    content: "Use table-driven tests in Go for better coverage"
    context: "Applied in the authentication module"
`)

	results, err := Query(dir, "table-driven tests", QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	found := false
	for _, r := range results {
		if r.Type == "memory" && r.Score > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a memory result in results")
	}
}

func TestQuery_DomainFilterExcludesNonMatching(t *testing.T) {
	dir := t.TempDir()
	writeMemoryFile(t, dir, "patterns.yaml", `entries:
  - id: pattern-001
    domain: [golang]
    content: "Go patterns for testing"
    context: ""
  - id: pattern-002
    domain: [python]
    content: "Python testing patterns"
    context: ""
`)

	results, err := Query(dir, "testing patterns", QueryOptions{Domain: "golang"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, r := range results {
		if r.Type == "memory" && strings.Contains(r.Snippet, "Python") {
			t.Error("expected python entry to be excluded by domain filter")
		}
	}
}

func TestQuery_TypeFilterMemory(t *testing.T) {
	dir := t.TempDir()
	writeMemoryFile(t, dir, "patterns.yaml", `entries:
  - id: pattern-001
    domain: [golang]
    content: "OAuth authentication flow"
    context: ""
`)
	writeHandoffFile(t, dir, "feature/1", "01-coder.md", "# OAuth handoff\nImplemented OAuth flow")

	results, err := Query(dir, "OAuth", QueryOptions{Type: "memory"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, r := range results {
		if r.Type != "memory" {
			t.Errorf("expected only memory results with type filter, got type: %s", r.Type)
		}
	}
}

func TestQuery_TypeFilterHandoff(t *testing.T) {
	dir := t.TempDir()
	writeHandoffFile(t, dir, "feature/1", "01-coder.md", "# Handoff\nAuthentication implementation complete")

	results, err := Query(dir, "Authentication", QueryOptions{Type: "handoff"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one handoff result")
	}
	for _, r := range results {
		if r.Type != "handoff" {
			t.Errorf("expected only handoff results, got: %s", r.Type)
		}
	}
}

func TestQuery_TypeFilterADR(t *testing.T) {
	dir := t.TempDir()
	writeADRFile(t, dir, "0001-use-postgres.md", "# Use PostgreSQL\nDecided to use PostgreSQL for storage")

	results, err := Query(dir, "PostgreSQL", QueryOptions{Type: "adr"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one ADR result")
	}
	for _, r := range results {
		if r.Type != "adr" {
			t.Errorf("expected only adr results, got: %s", r.Type)
		}
	}
}

func TestQuery_TypeFilterState(t *testing.T) {
	dir := t.TempDir()
	writeStateFile(t, dir, "feature__1-auth", `id: feature/1-auth
type: feature
status: active
goal: Implement JWT authentication
current_step: 3
current_role: coder
steps: []
`)

	results, err := Query(dir, "JWT authentication", QueryOptions{Type: "state"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one state result")
	}
	for _, r := range results {
		if r.Type != "state" {
			t.Errorf("expected only state results, got: %s", r.Type)
		}
	}
}

func TestQuery_HigherScoreRanksFirst(t *testing.T) {
	dir := t.TempDir()
	writeMemoryFile(t, dir, "patterns.yaml", `entries:
  - id: pattern-001
    domain: [golang]
    content: "authentication authentication authentication"
    context: ""
  - id: pattern-002
    domain: [golang]
    content: "authentication once"
    context: ""
`)

	results, err := Query(dir, "authentication", QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) < 2 {
		t.Skip("need at least 2 results to check ordering")
	}
	if results[0].Score < results[1].Score {
		t.Errorf("expected higher score first: got %d then %d", results[0].Score, results[1].Score)
	}
}
