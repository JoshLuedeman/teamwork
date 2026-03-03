package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithRepos(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `
project:
  name: "test"
  repo: "owner/test"
roles:
  core: [coder]
  optional: []
workflows:
  skip_steps: {}
  extra_gates: {}
quality_gates:
  handoff_complete: true
  tests_pass: true
  lint_pass: true
memory:
  archive_threshold: 50
  sync_to_memory_md: true
repos:
  - name: "api"
    path: "../api"
    repo: "owner/api"
  - name: "frontend"
    path: "../frontend"
    repo: "owner/frontend"
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Name != "api" {
		t.Errorf("repos[0].name = %q, want %q", cfg.Repos[0].Name, "api")
	}
	if cfg.Repos[1].Repo != "owner/frontend" {
		t.Errorf("repos[1].repo = %q, want %q", cfg.Repos[1].Repo, "owner/frontend")
	}
}

func TestLoadWithoutRepos(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `
project:
  name: "test"
  repo: "owner/test"
roles:
  core: [coder]
  optional: []
workflows:
  skip_steps: {}
  extra_gates: {}
quality_gates:
  handoff_complete: true
  tests_pass: true
  lint_pass: true
memory:
  archive_threshold: 50
  sync_to_memory_md: true
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(cfg.Repos))
	}
}

func TestGetRepo(t *testing.T) {
	cfg := &Config{
		Repos: []RepoConfig{
			{Name: "api", Path: "../api", Repo: "owner/api"},
			{Name: "web", Path: "../web", Repo: "owner/web"},
		},
	}

	if r := cfg.GetRepo("api"); r == nil {
		t.Error("GetRepo(api) returned nil")
	} else if r.Repo != "owner/api" {
		t.Errorf("GetRepo(api).Repo = %q, want %q", r.Repo, "owner/api")
	}

	if r := cfg.GetRepo("missing"); r != nil {
		t.Errorf("GetRepo(missing) = %v, want nil", r)
	}
}
