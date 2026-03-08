// Package config parses and manages project-level orchestration settings
// stored in .teamwork/config.yaml.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level .teamwork/config.yaml structure.
type Config struct {
	Project      ProjectConfig        `yaml:"project"`
	Roles        RolesConfig          `yaml:"roles"`
	Workflows    WorkflowsConfig      `yaml:"workflows"`
	QualityGates QualityGatesConfig   `yaml:"quality_gates"`
	Memory       MemoryConfig         `yaml:"memory"`
	MCPServers   map[string]MCPServer `yaml:"mcp_servers"`
	Repos        []RepoConfig         `yaml:"repos,omitempty"`
}

// ProjectConfig identifies the project.
type ProjectConfig struct {
	Name string `yaml:"name"`
	Repo string `yaml:"repo"`
}

// RepoConfig describes a spoke repository in a multi-repo setup.
type RepoConfig struct {
	Name string `yaml:"name"` // Short identifier (e.g., "api", "frontend")
	Path string `yaml:"path"` // Local filesystem path (relative or absolute)
	Repo string `yaml:"repo"` // GitHub owner/repo slug
}

// MCPServer describes an MCP server that agents can use for tooling.
type MCPServer struct {
	Description string   `yaml:"description"`
	URL         string   `yaml:"url,omitempty"`
	Command     string   `yaml:"command,omitempty"`
	Roles       []string `yaml:"roles"`
	EnvVars     []string `yaml:"env_vars"`
	Install     string   `yaml:"install"`
}

// RolesConfig defines which roles are active in the project.
type RolesConfig struct {
	Core     []string `yaml:"core"`
	Optional []string `yaml:"optional"`
}

// WorkflowsConfig holds per-workflow-type customizations.
type WorkflowsConfig struct {
	SkipSteps  map[string][]string            `yaml:"skip_steps"`
	ExtraGates map[string]map[string][]string  `yaml:"extra_gates"`
}

// QualityGatesConfig controls default quality gate enforcement.
type QualityGatesConfig struct {
	HandoffComplete bool `yaml:"handoff_complete"`
	TestsPass       bool `yaml:"tests_pass"`
	LintPass        bool `yaml:"lint_pass"`
}

// MemoryConfig controls structured memory management.
type MemoryConfig struct {
	ArchiveThreshold int  `yaml:"archive_threshold"`
	SyncToMemoryMD   bool `yaml:"sync_to_memory_md"`
}

// configPath returns the path to config.yaml inside the given directory.
func configPath(dir string) string {
	return filepath.Join(dir, ".teamwork", "config.yaml")
}

// Load reads and parses the config from <dir>/.teamwork/config.yaml.
func Load(dir string) (*Config, error) {
	data, err := os.ReadFile(configPath(dir))
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", configPath(dir), err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", configPath(dir), err)
	}
	return &cfg, nil
}

// Default returns a Config populated with sensible defaults that match the
// template .teamwork/config.yaml shipped with the project.
func Default() *Config {
	return &Config{
		Project: ProjectConfig{
			Name: "my-project",
			Repo: "owner/repo",
		},
		Roles: RolesConfig{
			Core: []string{
				"planner",
				"architect",
				"coder",
				"tester",
				"reviewer",
				"security-auditor",
				"documenter",
				"orchestrator",
			},
			Optional: []string{},
		},
		Workflows: WorkflowsConfig{
			SkipSteps: map[string][]string{
				"documentation": {"security-auditor"},
				"spike":         {"tester", "security-auditor"},
			},
			ExtraGates: map[string]map[string][]string{},
		},
		QualityGates: QualityGatesConfig{
			HandoffComplete: true,
			TestsPass:       true,
			LintPass:        true,
		},
		Memory: MemoryConfig{
			ArchiveThreshold: 50,
			SyncToMemoryMD:   true,
		},
	}
}

// Save writes the config to <dir>/.teamwork/config.yaml, creating the
// directory if it does not exist.
func (c *Config) Save(dir string) error {
	p := configPath(dir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", filepath.Dir(p), err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}

	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("config: write %s: %w", p, err)
	}
	return nil
}

// IsRoleActive reports whether the given role is listed in core or optional.
func (c *Config) IsRoleActive(role string) bool {
	for _, r := range c.Roles.Core {
		if r == role {
			return true
		}
	}
	for _, r := range c.Roles.Optional {
		if r == role {
			return true
		}
	}
	return false
}

// ShouldSkipStep reports whether the given role should be skipped for the
// specified workflow type, according to the skip_steps configuration.
func (c *Config) ShouldSkipStep(workflowType, role string) bool {
	roles, ok := c.Workflows.SkipSteps[workflowType]
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// GetRepo returns the RepoConfig for the given name, or nil if not found.
func (c *Config) GetRepo(name string) *RepoConfig {
	for i := range c.Repos {
		if c.Repos[i].Name == name {
			return &c.Repos[i]
		}
	}
	return nil
}
