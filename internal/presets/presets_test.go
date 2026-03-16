package presets

import (
	"testing"
)

func TestNames_ReturnsAllPresets(t *testing.T) {
	names := Names()
	expected := []string{"fullstack", "go-api", "python-ml", "react-ts"}
	if len(names) != len(expected) {
		t.Fatalf("Names() returned %d presets, want %d", len(names), len(expected))
	}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("Names()[%d] = %q, want %q", i, names[i], name)
		}
	}
}

func TestNames_IsSorted(t *testing.T) {
	names := Names()
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("Names() not sorted: %q comes after %q", names[i], names[i-1])
		}
	}
}

func TestGet_UnknownPreset(t *testing.T) {
	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("Get(nonexistent) should return an error")
	}
}

func TestGet_AllPresetsReturnValidConfig(t *testing.T) {
	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			cfg, err := Get(name)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", name, err)
			}
			if cfg == nil {
				t.Fatalf("Get(%q) returned nil config", name)
			}
			if cfg.Project.Name != "my-project" {
				t.Errorf("project.name = %q, want %q", cfg.Project.Name, "my-project")
			}
			if cfg.Project.Repo != "owner/repo" {
				t.Errorf("project.repo = %q, want %q", cfg.Project.Repo, "owner/repo")
			}
			if len(cfg.Roles.Core) == 0 {
				t.Error("core roles are empty")
			}
			if cfg.Roles.Optional == nil {
				t.Error("optional roles is nil, want non-nil slice")
			}
			if !cfg.QualityGates.HandoffComplete {
				t.Error("quality_gates.handoff_complete should be true")
			}
			if !cfg.QualityGates.TestsPass {
				t.Error("quality_gates.tests_pass should be true")
			}
			if !cfg.QualityGates.LintPass {
				t.Error("quality_gates.lint_pass should be true")
			}
			if cfg.Memory.ArchiveThreshold != 50 {
				t.Errorf("memory.archive_threshold = %d, want 50", cfg.Memory.ArchiveThreshold)
			}
			if _, ok := cfg.MCPServers["github"]; !ok {
				t.Error("mcp_servers missing github")
			}
			if cfg.Workflows.SkipSteps == nil {
				t.Error("workflows.skip_steps is nil")
			}
			if cfg.Workflows.ExtraGates == nil {
				t.Error("workflows.extra_gates is nil")
			}
		})
	}
}

func TestGoAPI_HasExpectedMCPServers(t *testing.T) {
	cfg, err := Get("go-api")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"github", "context7", "semgrep", "osv", "coverage", "commits", "adr", "complexity"}
	for _, name := range expected {
		if _, ok := cfg.MCPServers[name]; !ok {
			t.Errorf("go-api preset missing MCP server %q", name)
		}
	}
	if _, ok := cfg.MCPServers["e2b"]; ok {
		t.Error("go-api preset should not include e2b")
	}
}

func TestGoAPI_HasDevOpsOptionalRole(t *testing.T) {
	cfg, err := Get("go-api")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Roles.Optional) != 1 || cfg.Roles.Optional[0] != "devops" {
		t.Errorf("go-api optional roles = %v, want [devops]", cfg.Roles.Optional)
	}
}

func TestReactTS_HasExpectedMCPServers(t *testing.T) {
	cfg, err := Get("react-ts")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"github", "context7", "semgrep", "e2b", "commits", "complexity"}
	for _, name := range expected {
		if _, ok := cfg.MCPServers[name]; !ok {
			t.Errorf("react-ts preset missing MCP server %q", name)
		}
	}
	if _, ok := cfg.MCPServers["coverage"]; ok {
		t.Error("react-ts preset should not include coverage")
	}
}

func TestReactTS_NoOptionalRoles(t *testing.T) {
	cfg, err := Get("react-ts")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Roles.Optional) != 0 {
		t.Errorf("react-ts optional roles = %v, want empty", cfg.Roles.Optional)
	}
}

func TestPythonML_HasExpectedMCPServers(t *testing.T) {
	cfg, err := Get("python-ml")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"github", "context7", "semgrep", "e2b", "osv", "commits", "complexity"}
	for _, name := range expected {
		if _, ok := cfg.MCPServers[name]; !ok {
			t.Errorf("python-ml preset missing MCP server %q", name)
		}
	}
}

func TestPythonML_SkipsSecurityAuditorOnRefactor(t *testing.T) {
	cfg, err := Get("python-ml")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ShouldSkipStep("refactor", "security-auditor") {
		t.Error("python-ml should skip security-auditor for refactor workflows")
	}
}

func TestFullstack_HasExpectedMCPServers(t *testing.T) {
	cfg, err := Get("fullstack")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"github", "context7", "semgrep", "e2b", "osv", "coverage", "commits", "adr", "changelog", "complexity"}
	for _, name := range expected {
		if _, ok := cfg.MCPServers[name]; !ok {
			t.Errorf("fullstack preset missing MCP server %q", name)
		}
	}
}

func TestFullstack_HasExpectedOptionalRoles(t *testing.T) {
	cfg, err := Get("fullstack")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"devops": true, "dependency-manager": true}
	if len(cfg.Roles.Optional) != len(want) {
		t.Fatalf("fullstack optional roles = %v, want %v", cfg.Roles.Optional, want)
	}
	for _, role := range cfg.Roles.Optional {
		if !want[role] {
			t.Errorf("unexpected optional role %q", role)
		}
	}
}

func TestPresetsAreIndependent(t *testing.T) {
	cfg1, _ := Get("go-api")
	cfg2, _ := Get("go-api")
	cfg1.Project.Name = "mutated"
	if cfg2.Project.Name == "mutated" {
		t.Error("modifying one preset config affected another")
	}
}
