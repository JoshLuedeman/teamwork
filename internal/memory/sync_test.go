package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupMemoryDir creates a temp directory with the .teamwork/memory structure
// and returns the root dir path.
func setupMemoryDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	memDir := filepath.Join(dir, ".teamwork", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatalf("creating memory dir: %v", err)
	}
	return dir
}

func TestSyncToMemoryMD_BasicContent(t *testing.T) {
	dir := setupMemoryDir(t)

	// Add entries to two categories.
	entries := []Entry{
		{ID: "pattern-001", Date: "2026-01-01", Source: "PR #1", Domain: []string{"api"}, Content: "Use middleware for auth", Context: "Cleaner than per-route"},
		{ID: "pattern-002", Date: "2026-01-02", Source: "PR #5", Domain: []string{"testing"}, Content: "Always test error paths"},
	}
	if err := SaveCategory(dir, Patterns, &MemoryFile{Entries: entries}); err != nil {
		t.Fatalf("saving patterns: %v", err)
	}

	feedback := []Entry{
		{ID: "feedback-001", Date: "2026-02-01", Source: "review", Domain: []string{"testing"}, Content: "Add table-driven tests", Context: "More maintainable"},
	}
	if err := SaveCategory(dir, Feedback, &MemoryFile{Entries: feedback}); err != nil {
		t.Fatalf("saving feedback: %v", err)
	}

	if err := SyncToMemoryMD(dir); err != nil {
		t.Fatalf("SyncToMemoryMD() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "MEMORY.md"))
	if err != nil {
		t.Fatalf("reading MEMORY.md: %v", err)
	}
	content := string(data)

	// Check markers.
	if !strings.Contains(content, beginMarker) {
		t.Error("missing begin marker")
	}
	if !strings.Contains(content, endMarker) {
		t.Error("missing end marker")
	}

	// Check category headings.
	if !strings.Contains(content, "## Patterns That Work") {
		t.Error("missing Patterns That Work heading")
	}
	if !strings.Contains(content, "## Patterns to Avoid") {
		t.Error("missing Patterns to Avoid heading")
	}
	if !strings.Contains(content, "## Key Decisions") {
		t.Error("missing Key Decisions heading")
	}
	if !strings.Contains(content, "## Reviewer Feedback") {
		t.Error("missing Reviewer Feedback heading")
	}

	// Check entries rendered.
	if !strings.Contains(content, "**Use middleware for auth**") {
		t.Error("missing pattern-001 content")
	}
	if !strings.Contains(content, "— Cleaner than per-route") {
		t.Error("missing pattern-001 context")
	}
	if !strings.Contains(content, "*(PR #1)*") {
		t.Error("missing pattern-001 source")
	}
	if !strings.Contains(content, "**Always test error paths**") {
		t.Error("missing pattern-002 content")
	}
	if !strings.Contains(content, "**Add table-driven tests**") {
		t.Error("missing feedback-001 content")
	}

	// Empty categories should show placeholder.
	if !strings.Contains(content, "*(No entries yet)*") {
		t.Error("empty categories should show placeholder text")
	}
}

func TestSyncToMemoryMD_PreservesExistingContent(t *testing.T) {
	dir := setupMemoryDir(t)

	// Write a MEMORY.md with existing content and markers.
	existing := "# Project Memory\n\nSome intro text that should be preserved.\n\n" +
		beginMarker + "\n## old content that will be replaced\n" + endMarker +
		"\n\n## Manual Section\n\nThis was added by hand and should be preserved.\n"
	mdPath := filepath.Join(dir, "MEMORY.md")
	if err := os.WriteFile(mdPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("writing MEMORY.md: %v", err)
	}

	if err := SyncToMemoryMD(dir); err != nil {
		t.Fatalf("SyncToMemoryMD() error: %v", err)
	}

	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("reading MEMORY.md: %v", err)
	}
	content := string(data)

	// Content before markers should be preserved.
	if !strings.Contains(content, "Some intro text that should be preserved.") {
		t.Error("content before markers was not preserved")
	}

	// Content after markers should be preserved.
	if !strings.Contains(content, "## Manual Section") {
		t.Error("content after markers was not preserved")
	}
	if !strings.Contains(content, "This was added by hand and should be preserved.") {
		t.Error("manual section content was not preserved")
	}

	// Old structured content should be replaced.
	if strings.Contains(content, "old content that will be replaced") {
		t.Error("old structured content was not replaced")
	}

	// New structured content should be present.
	if !strings.Contains(content, "## Patterns That Work") {
		t.Error("new structured content not present")
	}
}

func TestSyncToMemoryMD_AppendsToExistingWithoutMarkers(t *testing.T) {
	dir := setupMemoryDir(t)

	// Write a MEMORY.md without markers.
	existing := "# Project Memory\n\nSome existing content.\n"
	mdPath := filepath.Join(dir, "MEMORY.md")
	if err := os.WriteFile(mdPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("writing MEMORY.md: %v", err)
	}

	if err := SyncToMemoryMD(dir); err != nil {
		t.Fatalf("SyncToMemoryMD() error: %v", err)
	}

	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("reading MEMORY.md: %v", err)
	}
	content := string(data)

	// Original content should be preserved.
	if !strings.Contains(content, "# Project Memory") {
		t.Error("original heading was not preserved")
	}
	if !strings.Contains(content, "Some existing content.") {
		t.Error("original content was not preserved")
	}

	// Structured section should be appended.
	if !strings.Contains(content, beginMarker) {
		t.Error("begin marker not added")
	}
	if !strings.Contains(content, endMarker) {
		t.Error("end marker not added")
	}

	// Markers should come after the existing content.
	existingIdx := strings.Index(content, "Some existing content.")
	markerIdx := strings.Index(content, beginMarker)
	if markerIdx < existingIdx {
		t.Error("structured section should be after existing content")
	}
}

func TestSyncToMemoryMD_CreatesNewFile(t *testing.T) {
	dir := setupMemoryDir(t)
	mdPath := filepath.Join(dir, "MEMORY.md")

	// Ensure MEMORY.md does not exist.
	os.Remove(mdPath)

	if err := SyncToMemoryMD(dir); err != nil {
		t.Fatalf("SyncToMemoryMD() error: %v", err)
	}

	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("reading MEMORY.md: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, beginMarker) {
		t.Error("missing begin marker in new file")
	}
	if !strings.Contains(content, endMarker) {
		t.Error("missing end marker in new file")
	}
}

func TestSyncToMemoryMD_Idempotent(t *testing.T) {
	dir := setupMemoryDir(t)

	entries := []Entry{
		{ID: "pattern-001", Date: "2026-01-01", Domain: []string{"api"}, Content: "Test entry", Source: "test"},
	}
	if err := SaveCategory(dir, Patterns, &MemoryFile{Entries: entries}); err != nil {
		t.Fatalf("saving patterns: %v", err)
	}

	// Sync twice.
	if err := SyncToMemoryMD(dir); err != nil {
		t.Fatalf("first SyncToMemoryMD() error: %v", err)
	}
	if err := SyncToMemoryMD(dir); err != nil {
		t.Fatalf("second SyncToMemoryMD() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "MEMORY.md"))
	if err != nil {
		t.Fatalf("reading MEMORY.md: %v", err)
	}
	content := string(data)

	// Should have exactly one pair of markers.
	if strings.Count(content, beginMarker) != 1 {
		t.Errorf("expected 1 begin marker, got %d", strings.Count(content, beginMarker))
	}
	if strings.Count(content, endMarker) != 1 {
		t.Errorf("expected 1 end marker, got %d", strings.Count(content, endMarker))
	}
}

func TestSyncToMemoryMD_EntryWithoutContext(t *testing.T) {
	dir := setupMemoryDir(t)

	entries := []Entry{
		{ID: "decision-001", Date: "2026-01-01", Domain: []string{"arch"}, Content: "Use PostgreSQL", Source: "ADR-001"},
	}
	if err := SaveCategory(dir, Decisions, &MemoryFile{Entries: entries}); err != nil {
		t.Fatalf("saving decisions: %v", err)
	}

	if err := SyncToMemoryMD(dir); err != nil {
		t.Fatalf("SyncToMemoryMD() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "MEMORY.md"))
	if err != nil {
		t.Fatalf("reading MEMORY.md: %v", err)
	}
	content := string(data)

	// Entry without context should not have a dash separator.
	if strings.Contains(content, "**Use PostgreSQL** —") {
		t.Error("entry without context should not have dash separator")
	}
	if !strings.Contains(content, "**Use PostgreSQL**") {
		t.Error("missing decision content")
	}
	if !strings.Contains(content, "*(ADR-001)*") {
		t.Error("missing decision source")
	}
}

func TestSyncToMemoryMD_EntryWithoutSource(t *testing.T) {
	dir := setupMemoryDir(t)

	entries := []Entry{
		{ID: "pattern-001", Date: "2026-01-01", Domain: []string{"api"}, Content: "Use middleware", Context: "Cleaner code"},
	}
	if err := SaveCategory(dir, Patterns, &MemoryFile{Entries: entries}); err != nil {
		t.Fatalf("saving patterns: %v", err)
	}

	if err := SyncToMemoryMD(dir); err != nil {
		t.Fatalf("SyncToMemoryMD() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "MEMORY.md"))
	if err != nil {
		t.Fatalf("reading MEMORY.md: %v", err)
	}
	content := string(data)

	// Entry without source should not have source annotation.
	if strings.Contains(content, "*(*") {
		t.Error("entry without source should not have empty source annotation")
	}
	if !strings.Contains(content, "— Cleaner code") {
		t.Error("missing context")
	}
}
