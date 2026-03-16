package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ProjectInfo holds detected project metadata used to pre-populate agent files.
type ProjectInfo struct {
// TechStack is a short description of the primary tech stack (e.g. "Go 1.24").
TechStack string
// Languages lists the primary programming language(s) (e.g. "Go", "TypeScript").
Languages string
// PackageManager is the detected dependency manager (e.g. "go mod", "npm").
PackageManager string
// DependencyManifest is the primary manifest file (e.g. "go.mod", "package.json").
DependencyManifest string
// Lockfile is the dependency lockfile (e.g. "go.sum", "package-lock.json").
Lockfile string
// TestFramework is the detected testing framework (e.g. "go test", "Jest").
TestFramework string
// BuildCommand is the command used to build the project.
BuildCommand string
// TestCommand is the command used to run tests.
TestCommand string
// LintCommand is the command used to run linters.
LintCommand string
// AuditCommand is the command used to audit dependencies.
AuditCommand string
}

// Detect inspects dir and returns a best-effort ProjectInfo.
// Fields that cannot be determined are left empty.
func Detect(dir string) *ProjectInfo {
if fileExists(filepath.Join(dir, "go.mod")) {
return detectGo(dir)
}
if fileExists(filepath.Join(dir, "package.json")) {
return detectNodeProject(dir)
}
if fileExists(filepath.Join(dir, "Cargo.toml")) {
return detectRust()
}
if fileExists(filepath.Join(dir, "pyproject.toml")) ||
fileExists(filepath.Join(dir, "requirements.txt")) ||
fileExists(filepath.Join(dir, "setup.py")) {
return detectPython(dir)
}
return &ProjectInfo{}
}

func detectGo(dir string) *ProjectInfo {
lang := goLangVersion(dir)
return &ProjectInfo{
TechStack:          lang,
Languages:          lang,
PackageManager:     "go mod",
DependencyManifest: "go.mod",
Lockfile:           "go.sum",
TestFramework:      "go test",
BuildCommand:       "go build ./...",
TestCommand:        "go test ./...",
LintCommand:        "golangci-lint run",
AuditCommand:       "govulncheck ./...",
}
}

func detectNodeProject(dir string) *ProjectInfo {
info := &ProjectInfo{}
switch {
case fileExists(filepath.Join(dir, "pnpm-lock.yaml")):
info.PackageManager = "pnpm"
info.Lockfile = "pnpm-lock.yaml"
case fileExists(filepath.Join(dir, "yarn.lock")):
info.PackageManager = "yarn"
info.Lockfile = "yarn.lock"
default:
info.PackageManager = "npm"
info.Lockfile = "package-lock.json"
}
info.DependencyManifest = "package.json"
info.AuditCommand = info.PackageManager + " audit"
if fileExists(filepath.Join(dir, "tsconfig.json")) {
info.Languages = "TypeScript"
} else {
info.Languages = "JavaScript"
}
data, _ := os.ReadFile(filepath.Join(dir, "package.json"))
pkg := string(data)
switch {
case strings.Contains(pkg, `"vitest"`):
info.TestFramework = "Vitest"
case strings.Contains(pkg, `"jest"`):
info.TestFramework = "Jest"
case strings.Contains(pkg, `"mocha"`):
info.TestFramework = "Mocha"
}
pm := info.PackageManager
if strings.Contains(pkg, `"build"`) {
info.BuildCommand = pm + " run build"
}
if strings.Contains(pkg, `"test"`) {
if pm == "npm" {
info.TestCommand = "npm test"
} else {
info.TestCommand = pm + " test"
}
}
if strings.Contains(pkg, `"lint"`) {
info.LintCommand = pm + " run lint"
}
info.TechStack = info.Languages
return info
}

func detectRust() *ProjectInfo {
return &ProjectInfo{
TechStack:          "Rust",
Languages:          "Rust",
PackageManager:     "cargo",
DependencyManifest: "Cargo.toml",
Lockfile:           "Cargo.lock",
TestFramework:      "cargo test",
BuildCommand:       "cargo build",
TestCommand:        "cargo test",
LintCommand:        "cargo clippy",
AuditCommand:       "cargo audit",
}
}

func detectPython(dir string) *ProjectInfo {
info := &ProjectInfo{
TechStack:     "Python",
Languages:     "Python",
TestFramework: "pytest",
TestCommand:   "pytest",
LintCommand:   "ruff check .",
AuditCommand:  "pip-audit",
}
content, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
if strings.Contains(strings.ToLower(string(content)), "poetry") {
info.PackageManager = "poetry"
info.DependencyManifest = "pyproject.toml"
info.Lockfile = "poetry.lock"
info.BuildCommand = "poetry build"
} else if fileExists(filepath.Join(dir, "pyproject.toml")) {
info.PackageManager = "pip"
info.DependencyManifest = "pyproject.toml"
info.Lockfile = "requirements.txt"
} else {
info.PackageManager = "pip"
info.DependencyManifest = "requirements.txt"
info.Lockfile = "requirements.txt"
}
return info
}

// goLangVersion reads the Go version from go.mod and returns a string like "Go 1.24".
func goLangVersion(dir string) string {
data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
if err != nil {
return "Go"
}
for _, line := range strings.Split(string(data), "\n") {
line = strings.TrimSpace(line)
if strings.HasPrefix(line, "go ") {
parts := strings.Fields(line)
if len(parts) >= 2 {
return "Go " + parts[1]
}
}
}
return "Go"
}

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
