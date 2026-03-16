// Package presets provides pre-built configuration templates for common
// technology stacks.  Each preset returns a *config.Config that pre-fills
// roles, quality gates, skip-step rules and MCP server suggestions so that
// `teamwork init --preset <name>` gives users a useful starting point.
package presets

import (
	"fmt"
	"sort"

	"github.com/joshluedeman/teamwork/internal/config"
)

// Names returns the sorted list of available preset identifiers.
func Names() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Get returns the Config for a named preset.  It returns an error if the
// name is not recognized.
func Get(name string) (*config.Config, error) {
	fn, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown preset %q (available: %v)", name, Names())
	}
	return fn(), nil
}

// registry maps preset names to builder functions.
var registry = map[string]func() *config.Config{
	"go-api":    goAPI,
	"react-ts":  reactTS,
	"python-ml": pythonML,
	"fullstack": fullstack,
}

func coreRoles() []string {
	return []string{
		"planner", "architect", "coder", "tester",
		"reviewer", "security-auditor", "documenter", "orchestrator",
	}
}

func defaultMemory() config.MemoryConfig {
	return config.MemoryConfig{
		ArchiveThreshold: 50,
		SyncToMemoryMD:   true,
	}
}

func mcpGitHub() config.MCPServer {
	return config.MCPServer{
		Description: "GitHub repos, PRs, issues, CI workflows, Dependabot alerts",
		URL:         "https://api.githubcopilot.com/mcp/",
		Roles:       []string{"planner", "architect", "coder", "tester", "reviewer", "security-auditor", "documenter", "orchestrator"},
		EnvVars:     []string{"GH_TOKEN"},
		Install:     "gh extension install github/gh-mcp",
	}
}

func mcpContext7() config.MCPServer {
	return config.MCPServer{
		Description: "Real-time library documentation",
		URL:         "https://mcp.context7.com/mcp",
		Roles:       []string{"architect", "coder", "documenter"},
		EnvVars:     []string{},
		Install:     "npx -y @upstash/context7-mcp",
	}
}

func mcpSemgrep() config.MCPServer {
	return config.MCPServer{
		Description: "SAST security scanning",
		Command:     "uvx semgrep-mcp",
		Roles:       []string{"security-auditor", "reviewer", "coder"},
		EnvVars:     []string{"SEMGREP_APP_TOKEN"},
		Install:     "pip install semgrep-mcp",
	}
}

func mcpE2B() config.MCPServer {
	return config.MCPServer{
		Description: "Cloud-sandboxed Python and JavaScript code execution",
		Command:     "uvx e2b-mcp",
		Roles:       []string{"coder", "tester"},
		EnvVars:     []string{"E2B_API_KEY"},
		Install:     "pip install e2b-mcp",
	}
}

func mcpOSV() config.MCPServer {
	return config.MCPServer{
		Description: "Open Source Vulnerability database",
		Command:     "uvx osv-mcp",
		Roles:       []string{"security-auditor", "reviewer"},
		EnvVars:     []string{},
		Install:     "pip install osv-mcp",
	}
}

func mcpCoverage() config.MCPServer {
	return config.MCPServer{
		Description: "Test coverage report analysis",
		Command:     "uvx teamwork-mcp-coverage",
		Roles:       []string{"tester", "reviewer", "orchestrator"},
		EnvVars:     []string{},
		Install:     "pip install teamwork-mcp-coverage",
	}
}

func mcpCommits() config.MCPServer {
	return config.MCPServer{
		Description: "Conventional commit message generation and validation from diffs",
		Command:     "uvx teamwork-mcp-commits",
		Roles:       []string{"coder", "reviewer", "orchestrator"},
		EnvVars:     []string{},
		Install:     "pip install teamwork-mcp-commits",
	}
}

func mcpADR() config.MCPServer {
	return config.MCPServer{
		Description: "Architecture Decision Record search, creation, and management",
		Command:     "uvx teamwork-mcp-adr",
		Roles:       []string{"architect", "orchestrator", "coder"},
		EnvVars:     []string{},
		Install:     "pip install teamwork-mcp-adr",
	}
}

func mcpChangelog() config.MCPServer {
	return config.MCPServer{
		Description: "Changelog generation and release notes using git-cliff",
		Command:     "uvx teamwork-mcp-changelog",
		Roles:       []string{"documenter", "orchestrator", "planner"},
		EnvVars:     []string{},
		Install:     "pip install teamwork-mcp-changelog",
	}
}

func mcpComplexity() config.MCPServer {
	return config.MCPServer{
		Description: "Code complexity analysis",
		Command:     "uvx teamwork-mcp-complexity",
		Roles:       []string{"reviewer", "tester", "architect"},
		EnvVars:     []string{},
		Install:     "pip install teamwork-mcp-complexity",
	}
}

// goAPI returns a config tuned for Go API/backend projects.
func goAPI() *config.Config {
	return &config.Config{
		Project: config.ProjectConfig{Name: "my-project", Repo: "owner/repo"},
		Roles: config.RolesConfig{
			Core:     coreRoles(),
			Optional: []string{"devops"},
		},
		Workflows: config.WorkflowsConfig{
			SkipSteps: map[string][]string{
				"documentation": {"security-auditor"},
				"spike":         {"tester", "security-auditor"},
			},
			ExtraGates: map[string]map[string][]string{},
		},
		QualityGates: config.QualityGatesConfig{HandoffComplete: true, TestsPass: true, LintPass: true},
		Memory:       defaultMemory(),
		MCPServers: map[string]config.MCPServer{
			"github":     mcpGitHub(),
			"context7":   mcpContext7(),
			"semgrep":    mcpSemgrep(),
			"osv":        mcpOSV(),
			"coverage":   mcpCoverage(),
			"commits":    mcpCommits(),
			"adr":        mcpADR(),
			"complexity": mcpComplexity(),
		},
	}
}

// reactTS returns a config tuned for React + TypeScript frontend projects.
func reactTS() *config.Config {
	return &config.Config{
		Project: config.ProjectConfig{Name: "my-project", Repo: "owner/repo"},
		Roles: config.RolesConfig{
			Core:     coreRoles(),
			Optional: []string{},
		},
		Workflows: config.WorkflowsConfig{
			SkipSteps: map[string][]string{
				"documentation": {"security-auditor"},
				"spike":         {"tester", "security-auditor"},
			},
			ExtraGates: map[string]map[string][]string{},
		},
		QualityGates: config.QualityGatesConfig{HandoffComplete: true, TestsPass: true, LintPass: true},
		Memory:       defaultMemory(),
		MCPServers: map[string]config.MCPServer{
			"github":     mcpGitHub(),
			"context7":   mcpContext7(),
			"semgrep":    mcpSemgrep(),
			"e2b":        mcpE2B(),
			"commits":    mcpCommits(),
			"complexity": mcpComplexity(),
		},
	}
}

// pythonML returns a config tuned for Python machine-learning projects.
func pythonML() *config.Config {
	return &config.Config{
		Project: config.ProjectConfig{Name: "my-project", Repo: "owner/repo"},
		Roles: config.RolesConfig{
			Core:     coreRoles(),
			Optional: []string{},
		},
		Workflows: config.WorkflowsConfig{
			SkipSteps: map[string][]string{
				"documentation": {"security-auditor"},
				"spike":         {"tester", "security-auditor"},
				"refactor":      {"security-auditor"},
			},
			ExtraGates: map[string]map[string][]string{},
		},
		QualityGates: config.QualityGatesConfig{HandoffComplete: true, TestsPass: true, LintPass: true},
		Memory:       defaultMemory(),
		MCPServers: map[string]config.MCPServer{
			"github":     mcpGitHub(),
			"context7":   mcpContext7(),
			"semgrep":    mcpSemgrep(),
			"e2b":        mcpE2B(),
			"osv":        mcpOSV(),
			"commits":    mcpCommits(),
			"complexity": mcpComplexity(),
		},
	}
}

// fullstack returns a config tuned for full-stack projects with both
// frontend and backend components.
func fullstack() *config.Config {
	return &config.Config{
		Project: config.ProjectConfig{Name: "my-project", Repo: "owner/repo"},
		Roles: config.RolesConfig{
			Core:     coreRoles(),
			Optional: []string{"devops", "dependency-manager"},
		},
		Workflows: config.WorkflowsConfig{
			SkipSteps: map[string][]string{
				"documentation": {"security-auditor"},
				"spike":         {"tester", "security-auditor"},
			},
			ExtraGates: map[string]map[string][]string{},
		},
		QualityGates: config.QualityGatesConfig{HandoffComplete: true, TestsPass: true, LintPass: true},
		Memory:       defaultMemory(),
		MCPServers: map[string]config.MCPServer{
			"github":     mcpGitHub(),
			"context7":   mcpContext7(),
			"semgrep":    mcpSemgrep(),
			"e2b":        mcpE2B(),
			"osv":        mcpOSV(),
			"coverage":   mcpCoverage(),
			"commits":    mcpCommits(),
			"adr":        mcpADR(),
			"changelog":  mcpChangelog(),
			"complexity": mcpComplexity(),
		},
	}
}
