package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joshluedeman/teamwork/internal/config"
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
