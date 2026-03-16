package installer

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// serveAndCheckDrift calls CheckDrift() via a mock HTTP test server.
func serveAndCheckDrift(t *testing.T, dir string, tb []byte) (*DriftResult, error) {
	t.Helper()
	srv := newTestServer(tb)
	t.Cleanup(srv.Close)
	original := http.DefaultTransport
	http.DefaultTransport = &redirectTransport{target: srv.URL, base: &http.Transport{}}
	t.Cleanup(func() { http.DefaultTransport = original })
	return CheckDrift(dir, "JoshLuedeman", "teamwork", "main")
}

// TestCheckDrift_NoDrift verifies that identical local and upstream files produce no drift.
func TestCheckDrift_NoDrift(t *testing.T) {
	dir := t.TempDir()
	tb := makeTarball(testPrefix, sampleFrameworkContent())
	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("serveAndInstall: %v", err)
	}
	result, err := serveAndCheckDrift(t, dir, tb)
	if err != nil {
		t.Fatalf("CheckDrift: %v", err)
	}
	if result.HasDrift() {
		t.Errorf("expected no drift, got Added=%v Modified=%v Removed=%v",
			result.Added, result.Modified, result.Removed)
	}
}

// TestCheckDrift_DetectsAddedFiles verifies files in upstream but not local are in Added.
func TestCheckDrift_DetectsAddedFiles(t *testing.T) {
	dir := t.TempDir()
	baseFiles := sampleFrameworkContent()
	tb := makeTarball(testPrefix, baseFiles)
	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("serveAndInstall: %v", err)
	}
	newFile := ".github/agents/new-agent.agent.md"
	upstreamFiles := make(map[string]string, len(baseFiles)+1)
	for k, v := range baseFiles {
		upstreamFiles[k] = v
	}
	upstreamFiles[newFile] = "# New Agent\n"
	upstreamTb := makeTarball(testPrefix, upstreamFiles)
	result, err := serveAndCheckDrift(t, dir, upstreamTb)
	if err != nil {
		t.Fatalf("CheckDrift: %v", err)
	}
	if !containsStr(result.Added, newFile) {
		t.Errorf("expected %q in Added, got %v", newFile, result.Added)
	}
	if len(result.Modified) != 0 {
		t.Errorf("expected no Modified, got %v", result.Modified)
	}
}

// TestCheckDrift_DetectsModifiedFiles verifies content changes are in Modified.
func TestCheckDrift_DetectsModifiedFiles(t *testing.T) {
	dir := t.TempDir()
	tb := makeTarball(testPrefix, sampleFrameworkContent())
	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("serveAndInstall: %v", err)
	}
	modifiedFile := ".editorconfig"
	if err := os.WriteFile(filepath.Join(dir, modifiedFile), []byte("# modified\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	result, err := serveAndCheckDrift(t, dir, tb)
	if err != nil {
		t.Fatalf("CheckDrift: %v", err)
	}
	if !containsStr(result.Modified, modifiedFile) {
		t.Errorf("expected %q in Modified, got %v", modifiedFile, result.Modified)
	}
	if len(result.Added) != 0 {
		t.Errorf("expected no Added, got %v", result.Added)
	}
}

// TestCheckDrift_DetectsRemovedFiles verifies files dropped from upstream appear in Removed.
func TestCheckDrift_DetectsRemovedFiles(t *testing.T) {
	dir := t.TempDir()
	baseFiles := sampleFrameworkContent()
	tb := makeTarball(testPrefix, baseFiles)
	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("serveAndInstall: %v", err)
	}
	removedFile := ".pre-commit-config.yaml"
	upstreamFiles := make(map[string]string, len(baseFiles)-1)
	for k, v := range baseFiles {
		if k != removedFile {
			upstreamFiles[k] = v
		}
	}
	upstreamTb := makeTarball(testPrefix, upstreamFiles)
	result, err := serveAndCheckDrift(t, dir, upstreamTb)
	if err != nil {
		t.Fatalf("CheckDrift: %v", err)
	}
	if !containsStr(result.Removed, removedFile) {
		t.Errorf("expected %q in Removed, got %v", removedFile, result.Removed)
	}
	if len(result.Added) != 0 {
		t.Errorf("expected no Added, got %v", result.Added)
	}
}

// TestCheckDrift_NoDiskModification verifies CheckDrift does not write to disk.
func TestCheckDrift_NoDiskModification(t *testing.T) {
	dir := t.TempDir()
	tb := makeTarball(testPrefix, sampleFrameworkContent())
	if err := serveAndInstall(t, dir, tb); err != nil {
		t.Fatalf("serveAndInstall: %v", err)
	}
	before := dirSnapshot(t, dir)
	upstreamFiles := sampleFrameworkContent()
	upstreamFiles[".github/agents/phantom.agent.md"] = "# Phantom\n"
	upstreamTb := makeTarball(testPrefix, upstreamFiles)
	if _, err := serveAndCheckDrift(t, dir, upstreamTb); err != nil {
		t.Fatalf("CheckDrift: %v", err)
	}
	after := dirSnapshot(t, dir)
	for path, content := range before {
		if after[path] != content {
			t.Errorf("file %q was modified by CheckDrift", path)
		}
	}
	for path := range after {
		if _, existed := before[path]; !existed {
			t.Errorf("CheckDrift created unexpected file %q", path)
		}
	}
}

// TestCheckDrift_HasDrift_TrueWhenDrift verifies HasDrift returns true when there are changes.
func TestCheckDrift_HasDrift_TrueWhenDrift(t *testing.T) {
	d := &DriftResult{Added: []string{"new-file.md"}}
	if !d.HasDrift() {
		t.Error("HasDrift() should return true when Added is non-empty")
	}
}

// TestCheckDrift_HasDrift_FalseWhenClean verifies HasDrift returns false when there are no changes.
func TestCheckDrift_HasDrift_FalseWhenClean(t *testing.T) {
	d := &DriftResult{}
	if d.HasDrift() {
		t.Error("HasDrift() should return false when no drift")
	}
}

// dirSnapshot returns a map of relative path to content for all regular files in dir.
func dirSnapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	snap := make(map[string]string)
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		snap[rel] = string(data)
		return nil
	})
	return snap
}

// containsStr reports whether needle is in haystack.
func containsStr(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
