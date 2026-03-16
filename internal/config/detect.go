package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DetectTestFramework reports whether the project rooted at dir appears to
// have a test framework configured.  It checks for Go, Node, Python, and
// generic test-directory markers.
func DetectTestFramework(dir string) bool {
	// Go: go.mod implies `go test` is available.
	if fileExists(filepath.Join(dir, "go.mod")) {
		return true
	}

	// Node: package.json with a meaningful test script.
	if hasNodeTestScript(filepath.Join(dir, "package.json")) {
		return true
	}

	// Python: common test-framework config files.
	pythonIndicators := []string{"pytest.ini", "setup.cfg", "pyproject.toml"}
	for _, f := range pythonIndicators {
		if fileExists(filepath.Join(dir, f)) {
			return true
		}
	}

	// Generic: test/ or tests/ directory.
	for _, d := range []string{"test", "tests"} {
		if dirExists(filepath.Join(dir, d)) {
			return true
		}
	}

	return false
}

// DetectLinter reports whether the project rooted at dir appears to have a
// linter configured.  It checks for Go, JavaScript/TypeScript, and Python
// linter configuration files.
func DetectLinter(dir string) bool {
	linterFiles := []string{
		// Go
		".golangci.yml",
		".golangci.yaml",
		// JavaScript / TypeScript — legacy config
		".eslintrc",
		".eslintrc.js",
		".eslintrc.json",
		".eslintrc.yml",
		".eslintrc.yaml",
		// JavaScript / TypeScript — flat config
		"eslint.config.js",
		"eslint.config.mjs",
		"eslint.config.cjs",
		// Biome
		"biome.json",
		// Python
		".flake8",
		".pylintrc",
		"ruff.toml",
		".ruff.toml",
	}

	for _, f := range linterFiles {
		if fileExists(filepath.Join(dir, f)) {
			return true
		}
	}

	return false
}

// fileExists reports whether path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// dirExists reports whether path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// hasNodeTestScript returns true when the given package.json file contains a
// "test" script that is not the default npm placeholder.
func hasNodeTestScript(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}

	testCmd, ok := pkg.Scripts["test"]
	if !ok || testCmd == "" {
		return false
	}

	// npm init generates this placeholder — it means no real test runner.
	if strings.Contains(testCmd, "no test specified") {
		return false
	}

	return true
}
