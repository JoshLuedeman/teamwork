package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
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

// makeEntries generates n test entries for the given category.
func makeEntries(cat Category, n int) []Entry {
	prefix := categoryPrefix(cat)
	entries := make([]Entry, n)
	for i := range entries {
		entries[i] = Entry{
			ID:      fmt.Sprintf("%s%03d", prefix, i+1),
			Date:    fmt.Sprintf("2026-01-%02d", (i%28)+1),
			Source:  fmt.Sprintf("test-%d", i+1),
			Domain:  []string{"testing"},
			Content: fmt.Sprintf("entry %d", i+1),
		}
	}
	return entries
}

// seedCategory writes entries to a category file and rebuilds the index.
func seedCategory(t *testing.T, dir string, cat Category, entries []Entry) {
	t.Helper()
	mf := &MemoryFile{Entries: entries}
	if err := SaveCategory(dir, cat, mf); err != nil {
		t.Fatalf("saving seed category: %v", err)
	}
	idx := &Index{}
	if err := idx.Rebuild(dir); err != nil {
		t.Fatalf("rebuilding index: %v", err)
	}
}

func TestArchive(t *testing.T) {
	fixedTime := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixedTime }
	t.Cleanup(func() { nowFunc = origNow })

	tests := []struct {
		name            string
		entryCount      int
		threshold       int
		wantKept        int
		wantArchived    int
		wantArchiveFile bool
	}{
		{
			name:            "no archive when under threshold",
			entryCount:      3,
			threshold:       5,
			wantKept:        3,
			wantArchived:    0,
			wantArchiveFile: false,
		},
		{
			name:            "no archive when at threshold",
			entryCount:      5,
			threshold:       5,
			wantKept:        5,
			wantArchived:    0,
			wantArchiveFile: false,
		},
		{
			name:            "archive triggered when over threshold",
			entryCount:      8,
			threshold:       5,
			wantKept:        5,
			wantArchived:    3,
			wantArchiveFile: true,
		},
		{
			name:            "archive with threshold of 1",
			entryCount:      4,
			threshold:       1,
			wantKept:        1,
			wantArchived:    3,
			wantArchiveFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMemoryDir(t)
			entries := makeEntries(Patterns, tt.entryCount)
			seedCategory(t, dir, Patterns, entries)

			err := Archive(dir, Patterns, tt.threshold)
			if err != nil {
				t.Fatalf("Archive() error: %v", err)
			}

			// Check main file.
			mf, err := LoadCategory(dir, Patterns)
			if err != nil {
				t.Fatalf("LoadCategory() error: %v", err)
			}
			if len(mf.Entries) != tt.wantKept {
				t.Errorf("kept entries = %d, want %d", len(mf.Entries), tt.wantKept)
			}

			// Check archive file.
			archPath := archiveFilePath(dir, Patterns, fixedTime)
			archFile, err := loadMemoryFile(archPath)
			if err != nil {
				t.Fatalf("loadMemoryFile() error: %v", err)
			}
			if tt.wantArchiveFile {
				if len(archFile.Entries) != tt.wantArchived {
					t.Errorf("archived entries = %d, want %d", len(archFile.Entries), tt.wantArchived)
				}
			} else {
				if _, err := os.Stat(archPath); !os.IsNotExist(err) {
					t.Errorf("archive file should not exist, but does")
				}
			}
		})
	}
}

func TestArchive_PreservesCorrectEntries(t *testing.T) {
	fixedTime := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixedTime }
	t.Cleanup(func() { nowFunc = origNow })

	dir := setupMemoryDir(t)
	entries := makeEntries(Patterns, 7)
	seedCategory(t, dir, Patterns, entries)

	if err := Archive(dir, Patterns, 5); err != nil {
		t.Fatalf("Archive() error: %v", err)
	}

	// The most recent 5 entries (pattern-003 through pattern-007) should be kept.
	mf, err := LoadCategory(dir, Patterns)
	if err != nil {
		t.Fatalf("LoadCategory() error: %v", err)
	}
	for i, e := range mf.Entries {
		wantID := fmt.Sprintf("pattern-%03d", i+3) // 003, 004, 005, 006, 007
		if e.ID != wantID {
			t.Errorf("kept entry[%d].ID = %q, want %q", i, e.ID, wantID)
		}
	}

	// The oldest 2 entries (pattern-001, pattern-002) should be archived.
	archPath := archiveFilePath(dir, Patterns, fixedTime)
	archFile, err := loadMemoryFile(archPath)
	if err != nil {
		t.Fatalf("loadMemoryFile() error: %v", err)
	}
	for i, e := range archFile.Entries {
		wantID := fmt.Sprintf("pattern-%03d", i+1) // 001, 002
		if e.ID != wantID {
			t.Errorf("archived entry[%d].ID = %q, want %q", i, e.ID, wantID)
		}
	}
}

func TestArchive_RebuildIndex(t *testing.T) {
	fixedTime := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixedTime }
	t.Cleanup(func() { nowFunc = origNow })

	dir := setupMemoryDir(t)

	// Create entries with distinct domains so we can verify the index.
	entries := []Entry{
		{ID: "pattern-001", Date: "2026-01-01", Domain: []string{"old-domain"}, Content: "old"},
		{ID: "pattern-002", Date: "2026-01-02", Domain: []string{"old-domain"}, Content: "old"},
		{ID: "pattern-003", Date: "2026-01-03", Domain: []string{"new-domain"}, Content: "new"},
	}
	seedCategory(t, dir, Patterns, entries)

	// Verify index has both domains before archive.
	idx, err := LoadIndex(dir)
	if err != nil {
		t.Fatalf("LoadIndex() error: %v", err)
	}
	if _, ok := idx.Domains["old-domain"]; !ok {
		t.Fatal("index missing old-domain before archive")
	}
	if _, ok := idx.Domains["new-domain"]; !ok {
		t.Fatal("index missing new-domain before archive")
	}

	// Archive with threshold=1 so only the newest entry remains.
	if err := Archive(dir, Patterns, 1); err != nil {
		t.Fatalf("Archive() error: %v", err)
	}

	// After archiving, only "new-domain" should remain in the index.
	idx, err = LoadIndex(dir)
	if err != nil {
		t.Fatalf("LoadIndex() error: %v", err)
	}
	if _, ok := idx.Domains["old-domain"]; ok {
		t.Error("index should not contain old-domain after archive")
	}
	if ids, ok := idx.Domains["new-domain"]; !ok {
		t.Error("index missing new-domain after archive")
	} else if len(ids) != 1 || ids[0] != "pattern-003" {
		t.Errorf("new-domain ids = %v, want [pattern-003]", ids)
	}
}

func TestAdd_TriggersArchive(t *testing.T) {
	fixedTime := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixedTime }
	t.Cleanup(func() { nowFunc = origNow })

	dir := setupMemoryDir(t)
	threshold := 3

	// Seed with exactly threshold entries (no archive yet).
	entries := makeEntries(Patterns, threshold)
	seedCategory(t, dir, Patterns, entries)

	// Add one more entry to exceed the threshold.
	newEntry := Entry{
		Domain:  []string{"testing"},
		Content: "the straw that breaks the camel",
	}
	if err := Add(dir, Patterns, newEntry, threshold); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	// After the add, the main file should have exactly threshold entries.
	mf, err := LoadCategory(dir, Patterns)
	if err != nil {
		t.Fatalf("LoadCategory() error: %v", err)
	}
	if len(mf.Entries) != threshold {
		t.Errorf("entries after archive = %d, want %d", len(mf.Entries), threshold)
	}

	// The archive file should contain the oldest entry.
	archPath := archiveFilePath(dir, Patterns, fixedTime)
	archFile, err := loadMemoryFile(archPath)
	if err != nil {
		t.Fatalf("loadMemoryFile() error: %v", err)
	}
	if len(archFile.Entries) != 1 {
		t.Errorf("archived entries = %d, want 1", len(archFile.Entries))
	}
}

func TestAdd_NoArchiveWhenThresholdZero(t *testing.T) {
	dir := setupMemoryDir(t)

	// Seed with entries and add more with threshold=0 (archiving disabled).
	entries := makeEntries(Patterns, 5)
	seedCategory(t, dir, Patterns, entries)

	newEntry := Entry{
		Domain:  []string{"testing"},
		Content: "should not trigger archive",
	}
	if err := Add(dir, Patterns, newEntry, 0); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	// All 6 entries should remain in the main file.
	mf, err := LoadCategory(dir, Patterns)
	if err != nil {
		t.Fatalf("LoadCategory() error: %v", err)
	}
	if len(mf.Entries) != 6 {
		t.Errorf("entries = %d, want 6", len(mf.Entries))
	}
}

func TestArchive_AppendsToExistingArchiveFile(t *testing.T) {
	fixedTime := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixedTime }
	t.Cleanup(func() { nowFunc = origNow })

	dir := setupMemoryDir(t)

	// Pre-populate an archive file with one entry.
	archPath := archiveFilePath(dir, Patterns, fixedTime)
	existingArchive := &MemoryFile{
		Entries: []Entry{
			{ID: "pattern-100", Date: "2026-02-01", Domain: []string{"old"}, Content: "already archived"},
		},
	}
	data, err := yaml.Marshal(existingArchive)
	if err != nil {
		t.Fatalf("marshaling archive: %v", err)
	}
	if err := os.WriteFile(archPath, data, 0o644); err != nil {
		t.Fatalf("writing archive: %v", err)
	}

	// Seed the main file with entries that exceed the threshold.
	entries := makeEntries(Patterns, 4)
	seedCategory(t, dir, Patterns, entries)

	if err := Archive(dir, Patterns, 2); err != nil {
		t.Fatalf("Archive() error: %v", err)
	}

	// Archive should now have the pre-existing entry plus 2 new ones.
	archFile, err := loadMemoryFile(archPath)
	if err != nil {
		t.Fatalf("loadMemoryFile() error: %v", err)
	}
	if len(archFile.Entries) != 3 {
		t.Errorf("archive entries = %d, want 3 (1 existing + 2 new)", len(archFile.Entries))
	}
	if archFile.Entries[0].ID != "pattern-100" {
		t.Errorf("first archive entry = %q, want pattern-100", archFile.Entries[0].ID)
	}
}

func TestArchiveFilePath(t *testing.T) {
	ts := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	got := archiveFilePath("/proj", Patterns, ts)
	want := filepath.Join("/proj", ".teamwork", "memory", "patterns.archive-2026-03.yaml")
	if got != want {
		t.Errorf("archiveFilePath() = %q, want %q", got, want)
	}
}
