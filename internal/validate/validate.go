// Package validate checks the structural integrity of a .teamwork/ directory.
package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Result captures the outcome of a single validation check.
type Result struct {
	Path    string `json:"path"`
	Check   string `json:"check"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// validStatuses enumerates the allowed workflow status values.
var validStatuses = map[string]bool{
	"active":    true,
	"blocked":   true,
	"completed": true,
	"failed":    true,
	"cancelled": true,
}

// Run validates the .teamwork/ directory under dir and returns individual check results.
// It returns an error only if the .teamwork/ directory cannot be accessed.
func Run(dir string) ([]Result, error) {
	twDir := filepath.Join(dir, ".teamwork")
	if _, err := os.Stat(twDir); err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", twDir, err)
	}

	var results []Result

	results = append(results, checkConfigExists(twDir)...)
	results = append(results, checkStateFiles(twDir)...)
	results = append(results, checkHandoffFiles(twDir)...)
	results = append(results, checkMemoryFiles(twDir)...)
	results = append(results, checkMCPServers(twDir)...)

	return results, nil
}

// checkConfigExists validates config.yaml existence, parse-ability, and required fields.
func checkConfigExists(twDir string) []Result {
	var results []Result
	cfgPath := filepath.Join(twDir, "config.yaml")
	relPath := ".teamwork/config.yaml"

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		results = append(results, Result{
			Path:    relPath,
			Check:   "exists",
			Passed:  false,
			Message: fmt.Sprintf("%s: not found or unreadable", relPath),
		})
		return results
	}
	results = append(results, Result{
		Path:    relPath,
		Check:   "exists",
		Passed:  true,
		Message: relPath + ": exists",
	})

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		results = append(results, Result{
			Path:    relPath,
			Check:   "valid_yaml",
			Passed:  false,
			Message: fmt.Sprintf("%s: invalid YAML: %v", relPath, err),
		})
		return results
	}
	results = append(results, Result{
		Path:    relPath,
		Check:   "valid_yaml",
		Passed:  true,
		Message: relPath + ": valid YAML",
	})

	// Check required fields: project.name, project.repo, roles.core
	passed := true
	var missing []string

	project, _ := raw["project"].(map[string]interface{})
	if project == nil {
		missing = append(missing, "project.name", "project.repo")
		passed = false
	} else {
		if s, _ := project["name"].(string); s == "" {
			missing = append(missing, "project.name")
			passed = false
		}
		if s, _ := project["repo"].(string); s == "" {
			missing = append(missing, "project.repo")
			passed = false
		}
	}

	roles, _ := raw["roles"].(map[string]interface{})
	if roles == nil {
		missing = append(missing, "roles.core")
		passed = false
	} else {
		core, _ := roles["core"].([]interface{})
		if len(core) == 0 {
			missing = append(missing, "roles.core")
			passed = false
		}
	}

	if passed {
		results = append(results, Result{
			Path:    relPath,
			Check:   "required_fields",
			Passed:  true,
			Message: relPath + ": has required fields",
		})
	} else {
		results = append(results, Result{
			Path:    relPath,
			Check:   "required_fields",
			Passed:  false,
			Message: fmt.Sprintf("%s: missing required fields: %s", relPath, strings.Join(missing, ", ")),
		})
	}

	return results
}

// checkStateFiles validates all *.yaml files under .teamwork/state/.
func checkStateFiles(twDir string) []Result {
	var results []Result
	stateDir := filepath.Join(twDir, "state")

	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		return results
	}

	_ = filepath.Walk(stateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		relPath := ".teamwork/" + strings.TrimPrefix(path, twDir+string(os.PathSeparator))

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			results = append(results, Result{
				Path:    relPath,
				Check:   "valid_yaml",
				Passed:  false,
				Message: fmt.Sprintf("%s: unreadable: %v", relPath, readErr),
			})
			return nil
		}

		var raw map[string]interface{}
		if parseErr := yaml.Unmarshal(data, &raw); parseErr != nil {
			results = append(results, Result{
				Path:    relPath,
				Check:   "valid_yaml",
				Passed:  false,
				Message: fmt.Sprintf("%s: invalid YAML: %v", relPath, parseErr),
			})
			return nil
		}
		results = append(results, Result{
			Path:    relPath,
			Check:   "valid_yaml",
			Passed:  true,
			Message: relPath + ": valid YAML",
		})

		// Validate required state fields
		results = append(results, validateStateFields(relPath, raw)...)

		return nil
	})

	return results
}

// validateStateFields checks that a state YAML has the required fields with valid values.
func validateStateFields(relPath string, raw map[string]interface{}) []Result {
	var results []Result
	passed := true
	var msgs []string

	if s, _ := raw["id"].(string); s == "" {
		msgs = append(msgs, "missing or invalid id (string)")
		passed = false
	}
	if s, _ := raw["type"].(string); s == "" {
		msgs = append(msgs, "missing or invalid type (string)")
		passed = false
	}

	status, _ := raw["status"].(string)
	if !validStatuses[status] {
		msgs = append(msgs, fmt.Sprintf("invalid status %q", status))
		passed = false
	}

	// current_step must be int >= 0
	switch v := raw["current_step"].(type) {
	case int:
		if v < 0 {
			msgs = append(msgs, fmt.Sprintf("current_step must be >= 0, got %d", v))
			passed = false
		}
	case float64:
		if v < 0 || v != float64(int(v)) {
			msgs = append(msgs, fmt.Sprintf("current_step must be int >= 0, got %v", v))
			passed = false
		}
	default:
		msgs = append(msgs, "missing or invalid current_step (int)")
		passed = false
	}

	if s, _ := raw["created_at"].(string); s == "" {
		msgs = append(msgs, "missing or invalid created_at (string)")
		passed = false
	}

	if passed {
		results = append(results, Result{
			Path:    relPath,
			Check:   "required_fields",
			Passed:  true,
			Message: relPath + ": has required fields",
		})
	} else {
		for _, msg := range msgs {
			results = append(results, Result{
				Path:    relPath,
				Check:   "required_fields",
				Passed:  false,
				Message: fmt.Sprintf("%s: %s", relPath, msg),
			})
		}
	}

	return results
}

// checkHandoffFiles validates that all *.md files under .teamwork/handoffs/ are non-empty.
func checkHandoffFiles(twDir string) []Result {
	var results []Result
	handoffsDir := filepath.Join(twDir, "handoffs")

	if _, err := os.Stat(handoffsDir); os.IsNotExist(err) {
		return results
	}

	_ = filepath.Walk(handoffsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		relPath := ".teamwork/" + strings.TrimPrefix(path, twDir+string(os.PathSeparator))

		if info.Size() == 0 {
			results = append(results, Result{
				Path:    relPath,
				Check:   "non_empty",
				Passed:  false,
				Message: fmt.Sprintf("%s: empty handoff file", relPath),
			})
		} else {
			results = append(results, Result{
				Path:    relPath,
				Check:   "non_empty",
				Passed:  true,
				Message: relPath + ": non-empty",
			})
		}

		return nil
	})

	return results
}

// knownRoles enumerates the recognized Teamwork role names.
var knownRoles = map[string]bool{
	"planner":            true,
	"architect":          true,
	"coder":              true,
	"tester":             true,
	"reviewer":           true,
	"security-auditor":   true,
	"documenter":         true,
	"orchestrator":       true,
	"triager":            true,
	"devops":             true,
	"dependency-manager": true,
	"refactorer":         true,
	"lint-agent":         true,
	"api-agent":          true,
	"dba-agent":          true,
	"product-owner":      true,
	"qa-lead":            true,
}

// checkMCPServers validates the mcp_servers section of config.yaml, if present.
func checkMCPServers(twDir string) []Result {
	var results []Result
	cfgPath := filepath.Join(twDir, "config.yaml")
	relPath := ".teamwork/config.yaml"

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		// config.yaml missing is handled by checkConfigExists; skip silently.
		return results
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		// Invalid YAML is handled by checkConfigExists; skip silently.
		return results
	}

	mcpRaw, ok := raw["mcp_servers"]
	if !ok {
		// MCP servers section absent — optional, pass silently.
		return results
	}

	servers, _ := mcpRaw.(map[string]interface{})
	if servers == nil {
		// Present but empty or null — nothing to validate.
		return results
	}

	serverCount := 0
	missingEnvCount := 0

	for name, entry := range servers {
		srv, _ := entry.(map[string]interface{})
		if srv == nil {
			results = append(results, Result{
				Path:    relPath,
				Check:   "mcp_servers",
				Passed:  false,
				Message: fmt.Sprintf("mcp_servers.%s: invalid server entry", name),
			})
			continue
		}

		// description required
		desc, _ := srv["description"].(string)
		if desc == "" {
			results = append(results, Result{
				Path:    relPath,
				Check:   "mcp_servers",
				Passed:  false,
				Message: fmt.Sprintf("mcp_servers.%s: missing required field 'description'", name),
			})
			continue
		}

		// url XOR command
		urlVal, _ := srv["url"].(string)
		cmdVal, _ := srv["command"].(string)
		hasURL := urlVal != ""
		hasCmd := cmdVal != ""

		if hasURL && hasCmd {
			results = append(results, Result{
				Path:    relPath,
				Check:   "mcp_servers",
				Passed:  false,
				Message: fmt.Sprintf("mcp_servers.%s: must have either 'url' or 'command', not both", name),
			})
			continue
		}
		if !hasURL && !hasCmd {
			results = append(results, Result{
				Path:    relPath,
				Check:   "mcp_servers",
				Passed:  false,
				Message: fmt.Sprintf("mcp_servers.%s: must have either 'url' or 'command'", name),
			})
			continue
		}

		// URL format
		if hasURL && !strings.HasPrefix(urlVal, "http://") && !strings.HasPrefix(urlVal, "https://") {
			results = append(results, Result{
				Path:    relPath,
				Check:   "mcp_servers",
				Passed:  false,
				Message: fmt.Sprintf("mcp_servers.%s: url must start with http:// or https://", name),
			})
			continue
		}

		// roles validation
		rolesRaw, _ := srv["roles"].([]interface{})
		invalidRole := false
		for _, r := range rolesRaw {
			roleName, _ := r.(string)
			if !knownRoles[roleName] {
				results = append(results, Result{
					Path:    relPath,
					Check:   "mcp_servers",
					Passed:  false,
					Message: fmt.Sprintf("mcp_servers.%s: invalid role %q", name, roleName),
				})
				invalidRole = true
				break
			}
		}
		if invalidRole {
			continue
		}

		// env var warnings
		envVarsRaw, _ := srv["env_vars"].([]interface{})
		for _, ev := range envVarsRaw {
			envName, _ := ev.(string)
			if envName != "" && os.Getenv(envName) == "" {
				results = append(results, Result{
					Path:    relPath,
					Check:   "mcp_servers",
					Passed:  true,
					Message: fmt.Sprintf("mcp_servers.%s: WARN env var %s is not set", name, envName),
				})
				missingEnvCount++
			}
		}

		serverCount++
	}

	// Summary result when all servers are valid
	if serverCount > 0 {
		msg := fmt.Sprintf("mcp_servers: %d servers configured", serverCount)
		if missingEnvCount > 0 {
			msg = fmt.Sprintf("mcp_servers: %d servers configured (%d env vars missing)", serverCount, missingEnvCount)
		}
		results = append(results, Result{
			Path:    relPath,
			Check:   "mcp_servers",
			Passed:  true,
			Message: msg,
		})
	}

	return results
}

// checkMemoryFiles validates that all *.yaml files under .teamwork/memory/ parse as valid YAML.
func checkMemoryFiles(twDir string) []Result {
	var results []Result
	memoryDir := filepath.Join(twDir, "memory")

	if _, err := os.Stat(memoryDir); os.IsNotExist(err) {
		return results
	}

	_ = filepath.Walk(memoryDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		relPath := ".teamwork/" + strings.TrimPrefix(path, twDir+string(os.PathSeparator))

		if info.Size() == 0 {
			results = append(results, Result{
				Path:    relPath,
				Check:   "valid_yaml",
				Passed:  true,
				Message: relPath + ": empty (skipped)",
			})
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			results = append(results, Result{
				Path:    relPath,
				Check:   "valid_yaml",
				Passed:  false,
				Message: fmt.Sprintf("%s: unreadable: %v", relPath, readErr),
			})
			return nil
		}

		var raw interface{}
		if parseErr := yaml.Unmarshal(data, &raw); parseErr != nil {
			results = append(results, Result{
				Path:    relPath,
				Check:   "valid_yaml",
				Passed:  false,
				Message: fmt.Sprintf("%s: invalid YAML: %v", relPath, parseErr),
			})
		} else {
			results = append(results, Result{
				Path:    relPath,
				Check:   "valid_yaml",
				Passed:  true,
				Message: relPath + ": valid YAML",
			})
		}

		return nil
	})

	return results
}
