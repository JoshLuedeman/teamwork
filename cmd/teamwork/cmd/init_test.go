package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/joshluedeman/teamwork/internal/config"
	"github.com/joshluedeman/teamwork/internal/memory"
)

func TestRunInit_NonInteractive_CreatesDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if cfg.Project.Name != "my-project" {
		t.Errorf("project name = %q, want %q", cfg.Project.Name, "my-project")
	}
	if cfg.Project.Repo != "owner/repo" {
		t.Errorf("project repo = %q, want %q", cfg.Project.Repo, "owner/repo")
	}
}

func TestRunInit_NonInteractiveFlag_SkipsWizard(t *testing.T) {
	dir := t.TempDir()
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--dir", dir, "--non-interactive"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if cfg.Project.Name != "my-project" {
		t.Errorf("project name = %q, want %q", cfg.Project.Name, "my-project")
	}
}

func TestRunInit_AlreadyExists_Skips(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".teamwork"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}
	cfgPath := filepath.Join(dir, ".teamwork", "config.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		t.Error("config.yaml should not exist when .teamwork/ was pre-existing")
	}
}

func TestRunInit_CreatesSubdirectories(t *testing.T) {
	dir := t.TempDir()
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}
	expected := []string{"state", "handoffs", "memory", "metrics"}
	for _, sub := range expected {
		path := filepath.Join(dir, ".teamwork", sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("subdirectory %q not created: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q exists but is not a directory", sub)
		}
	}
}

func TestRunInit_CreatesMemoryFiles(t *testing.T) {
	dir := t.TempDir()
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}
	memoryFiles := []string{"patterns.yaml", "antipatterns.yaml", "decisions.yaml", "feedback.yaml", "index.yaml"}
	for _, name := range memoryFiles {
		path := filepath.Join(dir, ".teamwork", "memory", name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("memory file %q not created: %v", name, err)
		}
	}
}

func TestRunInit_MemoryFilesSeededWithExamples(t *testing.T) {
	dir := t.TempDir()
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify each category file is non-empty and contains a parseable example entry.
	categories := map[string]string{
		"patterns.yaml":     "pattern-001",
		"antipatterns.yaml": "antipattern-001",
		"decisions.yaml":    "decision-001",
		"feedback.yaml":     "feedback-001",
	}
	for name, expectedID := range categories {
		path := filepath.Join(dir, ".teamwork", "memory", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}

		content := string(data)
		if len(data) == 0 {
			t.Errorf("%s is empty, expected seed data", name)
			continue
		}

		// Verify the example entry ID is present.
		if !strings.Contains(content, expectedID) {
			t.Errorf("%s missing expected example entry ID %q", name, expectedID)
		}

		// Verify entries are marked as examples.
		if !strings.Contains(content, `source: "example"`) {
			t.Errorf("%s missing example source marker", name)
		}
		if !strings.Contains(content, "example") {
			t.Errorf("%s missing example domain tag", name)
		}

		// Verify all field names are documented in comments.
		for _, field := range []string{"id:", "date:", "source:", "domain:", "content:", "context:"} {
			if !strings.Contains(content, field) {
				t.Errorf("%s missing field %q", name, field)
			}
		}
	}
}

func TestRunInit_MemoryFilesParseableByMemoryPackage(t *testing.T) {
	dir := t.TempDir()
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	categories := []memory.Category{memory.Patterns, memory.Antipatterns, memory.Decisions, memory.Feedback}
	for _, cat := range categories {
		mf, err := memory.LoadCategory(dir, cat)
		if err != nil {
			t.Errorf("LoadCategory(%s) failed: %v", cat, err)
			continue
		}
		if len(mf.Entries) != 1 {
			t.Errorf("LoadCategory(%s): got %d entries, want 1", cat, len(mf.Entries))
			continue
		}
		entry := mf.Entries[0]
		if entry.Source != "example" {
			t.Errorf("LoadCategory(%s): entry source = %q, want %q", cat, entry.Source, "example")
		}
		if len(entry.Domain) != 1 || entry.Domain[0] != "example" {
			t.Errorf("LoadCategory(%s): entry domain = %v, want [example]", cat, entry.Domain)
		}
	}
}

func TestRunInit_IndexSeededWithExampleDomain(t *testing.T) {
	dir := t.TempDir()
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	idx, err := memory.LoadIndex(dir)
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	exampleIDs, ok := idx.Domains["example"]
	if !ok {
		t.Fatal("index missing 'example' domain")
	}

	expectedIDs := []string{"antipattern-001", "decision-001", "feedback-001", "pattern-001"}
	if len(exampleIDs) != len(expectedIDs) {
		t.Fatalf("example domain has %d entries, want %d", len(exampleIDs), len(expectedIDs))
	}

	// Sort for deterministic comparison.
	sorted := make([]string, len(exampleIDs))
	copy(sorted, exampleIDs)
	sort.Strings(sorted)
	for i, id := range expectedIDs {
		if sorted[i] != id {
			t.Errorf("example domain[%d] = %q, want %q", i, sorted[i], id)
		}
	}
}

func TestRunInit_ConfigFileStructure(t *testing.T) {
	dir := t.TempDir()
	cmd := rootCmd
	cmd.SetArgs([]string{"init", "--dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".teamwork", "config.yaml"))
	if err != nil {
		t.Fatalf("reading config.yaml: %v", err)
	}
	content := string(data)
	for _, key := range []string{"project:", "roles:", "quality_gates:", "memory:"} {
		if !strings.Contains(content, key) {
			t.Errorf("config.yaml missing expected key %q", key)
		}
	}
}

func TestRunWizard_AcceptsDefaults(t *testing.T) {
	input := strings.NewReader("\n\n\n")
	cfg := config.Default()
	result := runWizard(cfg, input)
	if result.Project.Name != "my-project" {
		t.Errorf("project name = %q, want %q", result.Project.Name, "my-project")
	}
	if result.Project.Repo != "owner/repo" {
		t.Errorf("project repo = %q, want %q", result.Project.Repo, "owner/repo")
	}
	if len(result.Roles.Optional) != 0 {
		t.Errorf("optional roles = %v, want empty", result.Roles.Optional)
	}
}

func TestRunWizard_CustomValues(t *testing.T) {
	input := strings.NewReader("cool-app\njosh/cool-app\ny\n")
	cfg := config.Default()
	result := runWizard(cfg, input)
	if result.Project.Name != "cool-app" {
		t.Errorf("project name = %q, want %q", result.Project.Name, "cool-app")
	}
	if result.Project.Repo != "josh/cool-app" {
		t.Errorf("project repo = %q, want %q", result.Project.Repo, "josh/cool-app")
	}
	expectedRoles := []string{"triager", "devops", "dependency-manager", "refactorer"}
	if len(result.Roles.Optional) != len(expectedRoles) {
		t.Fatalf("optional roles count = %d, want %d", len(result.Roles.Optional), len(expectedRoles))
	}
	for i, role := range expectedRoles {
		if result.Roles.Optional[i] != role {
			t.Errorf("optional role[%d] = %q, want %q", i, result.Roles.Optional[i], role)
		}
	}
}

func TestRunWizard_OptionalRolesDeclined(t *testing.T) {
	input := strings.NewReader("\n\nN\n")
	cfg := config.Default()
	result := runWizard(cfg, input)
	if len(result.Roles.Optional) != 0 {
		t.Errorf("optional roles = %v, want empty when declined", result.Roles.Optional)
	}
}

func TestRunWizard_PartialCustomValues(t *testing.T) {
	input := strings.NewReader("my-app\n\nn\n")
	cfg := config.Default()
	result := runWizard(cfg, input)
	if result.Project.Name != "my-app" {
		t.Errorf("project name = %q, want %q", result.Project.Name, "my-app")
	}
	if result.Project.Repo != "owner/repo" {
		t.Errorf("project repo = %q, want default %q", result.Project.Repo, "owner/repo")
	}
}
