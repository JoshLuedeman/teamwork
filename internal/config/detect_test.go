package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectTestFramework(t *testing.T) {
	tests := []struct {
		name  string
		setup func(dir string)
		want  bool
	}{
		{
			name:  "empty directory",
			setup: func(dir string) {},
			want:  false,
		},
		{
			name: "go.mod present",
			setup: func(dir string) {
				writeFile(t, dir, "go.mod", "module example\n")
			},
			want: true,
		},
		{
			name: "package.json with jest test script",
			setup: func(dir string) {
				writeFile(t, dir, "package.json", `{"scripts":{"test":"jest"}}`)
			},
			want: true,
		},
		{
			name: "package.json with vitest test script",
			setup: func(dir string) {
				writeFile(t, dir, "package.json", `{"scripts":{"test":"vitest run"}}`)
			},
			want: true,
		},
		{
			name: "package.json with default npm placeholder",
			setup: func(dir string) {
				writeFile(t, dir, "package.json",
					`{"scripts":{"test":"echo \"Error: no test specified\" && exit 1"}}`)
			},
			want: false,
		},
		{
			name: "package.json without scripts",
			setup: func(dir string) {
				writeFile(t, dir, "package.json", `{"name":"example"}`)
			},
			want: false,
		},
		{
			name: "package.json with empty test script",
			setup: func(dir string) {
				writeFile(t, dir, "package.json", `{"scripts":{"test":""}}`)
			},
			want: false,
		},
		{
			name: "pytest.ini present",
			setup: func(dir string) {
				writeFile(t, dir, "pytest.ini", "[pytest]\n")
			},
			want: true,
		},
		{
			name: "setup.cfg present",
			setup: func(dir string) {
				writeFile(t, dir, "setup.cfg", "[metadata]\n")
			},
			want: true,
		},
		{
			name: "pyproject.toml present",
			setup: func(dir string) {
				writeFile(t, dir, "pyproject.toml", "[project]\nname = \"example\"\n")
			},
			want: true,
		},
		{
			name: "tests directory exists",
			setup: func(dir string) {
				mkDir(t, dir, "tests")
			},
			want: true,
		},
		{
			name: "test directory exists",
			setup: func(dir string) {
				mkDir(t, dir, "test")
			},
			want: true,
		},
		{
			name: "test is a regular file not a directory",
			setup: func(dir string) {
				writeFile(t, dir, "test", "not a directory")
			},
			want: false,
		},
		{
			name: "unrelated files only",
			setup: func(dir string) {
				writeFile(t, dir, "README.md", "# Hello")
				writeFile(t, dir, "main.go", "package main")
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			if got := DetectTestFramework(dir); got != tt.want {
				t.Errorf("DetectTestFramework() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectLinter(t *testing.T) {
	tests := []struct {
		name  string
		setup func(dir string)
		want  bool
	}{
		{
			name:  "empty directory",
			setup: func(dir string) {},
			want:  false,
		},
		{
			name: "golangci-lint yml config",
			setup: func(dir string) {
				writeFile(t, dir, ".golangci.yml", "linters:\n")
			},
			want: true,
		},
		{
			name: "golangci-lint yaml config",
			setup: func(dir string) {
				writeFile(t, dir, ".golangci.yaml", "linters:\n")
			},
			want: true,
		},
		{
			name: "eslintrc json",
			setup: func(dir string) {
				writeFile(t, dir, ".eslintrc.json", "{}")
			},
			want: true,
		},
		{
			name: "eslintrc (no extension)",
			setup: func(dir string) {
				writeFile(t, dir, ".eslintrc", "{}")
			},
			want: true,
		},
		{
			name: "eslint flat config js",
			setup: func(dir string) {
				writeFile(t, dir, "eslint.config.js", "module.exports = {}")
			},
			want: true,
		},
		{
			name: "eslint flat config mjs",
			setup: func(dir string) {
				writeFile(t, dir, "eslint.config.mjs", "export default {}")
			},
			want: true,
		},
		{
			name: "biome.json",
			setup: func(dir string) {
				writeFile(t, dir, "biome.json", "{}")
			},
			want: true,
		},
		{
			name: "flake8 config",
			setup: func(dir string) {
				writeFile(t, dir, ".flake8", "[flake8]\n")
			},
			want: true,
		},
		{
			name: "pylintrc config",
			setup: func(dir string) {
				writeFile(t, dir, ".pylintrc", "[MASTER]\n")
			},
			want: true,
		},
		{
			name: "ruff.toml",
			setup: func(dir string) {
				writeFile(t, dir, "ruff.toml", "[lint]\n")
			},
			want: true,
		},
		{
			name: "dot ruff.toml",
			setup: func(dir string) {
				writeFile(t, dir, ".ruff.toml", "[lint]\n")
			},
			want: true,
		},
		{
			name: "unrelated files only",
			setup: func(dir string) {
				writeFile(t, dir, "README.md", "# Hello")
				writeFile(t, dir, "go.mod", "module example")
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			if got := DetectLinter(dir); got != tt.want {
				t.Errorf("DetectLinter() = %v, want %v", got, tt.want)
			}
		})
	}
}

// writeFile is a test helper that creates a file with the given content.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mkDir is a test helper that creates a subdirectory.
func mkDir(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestDetect_Go(t *testing.T) {
dir := t.TempDir()
writeFile(t, dir, "go.mod", "module example\n\ngo 1.24\n")
info := Detect(dir)
if info.Languages != "Go 1.24" {
t.Errorf("Languages = %q, want %q", info.Languages, "Go 1.24")
}
if info.PackageManager != "go mod" {
t.Errorf("PackageManager = %q, want %q", info.PackageManager, "go mod")
}
if info.TestCommand != "go test ./..." {
t.Errorf("TestCommand = %q, want %q", info.TestCommand, "go test ./...")
}
if info.BuildCommand != "go build ./..." {
t.Errorf("BuildCommand = %q, want %q", info.BuildCommand, "go build ./...")
}
if info.LintCommand != "golangci-lint run" {
t.Errorf("LintCommand = %q, want %q", info.LintCommand, "golangci-lint run")
}
if info.TestFramework != "go test" {
t.Errorf("TestFramework = %q, want %q", info.TestFramework, "go test")
}
}

func TestDetect_GoNoVersion(t *testing.T) {
dir := t.TempDir()
writeFile(t, dir, "go.mod", "module example\n")
info := Detect(dir)
if info.Languages != "Go" {
t.Errorf("Languages = %q, want %q", info.Languages, "Go")
}
}

func TestDetect_NodeNpm(t *testing.T) {
dir := t.TempDir()
writeFile(t, dir, "package.json",
`{"scripts":{"test":"jest","build":"webpack","lint":"eslint ."},"devDependencies":{"jest":"^29"}}`)
info := Detect(dir)
if info.Languages != "JavaScript" {
t.Errorf("Languages = %q, want %q", info.Languages, "JavaScript")
}
if info.PackageManager != "npm" {
t.Errorf("PackageManager = %q, want %q", info.PackageManager, "npm")
}
if info.TestFramework != "Jest" {
t.Errorf("TestFramework = %q, want %q", info.TestFramework, "Jest")
}
if info.TestCommand != "npm test" {
t.Errorf("TestCommand = %q, want %q", info.TestCommand, "npm test")
}
}

func TestDetect_NodeTypeScriptPnpm(t *testing.T) {
dir := t.TempDir()
writeFile(t, dir, "package.json",
`{"scripts":{"test":"vitest","build":"tsc"},"devDependencies":{"vitest":"^1","typescript":"^5"}}`)
writeFile(t, dir, "tsconfig.json", "{}")
writeFile(t, dir, "pnpm-lock.yaml", "lockfileVersion: 6\n")
info := Detect(dir)
if info.Languages != "TypeScript" {
t.Errorf("Languages = %q, want %q", info.Languages, "TypeScript")
}
if info.PackageManager != "pnpm" {
t.Errorf("PackageManager = %q, want %q", info.PackageManager, "pnpm")
}
if info.TestFramework != "Vitest" {
t.Errorf("TestFramework = %q, want %q", info.TestFramework, "Vitest")
}
}

func TestDetect_Rust(t *testing.T) {
dir := t.TempDir()
writeFile(t, dir, "Cargo.toml", "[package]\nname = \"myapp\"\n")
info := Detect(dir)
if info.Languages != "Rust" {
t.Errorf("Languages = %q, want %q", info.Languages, "Rust")
}
if info.PackageManager != "cargo" {
t.Errorf("PackageManager = %q, want %q", info.PackageManager, "cargo")
}
if info.TestCommand != "cargo test" {
t.Errorf("TestCommand = %q, want %q", info.TestCommand, "cargo test")
}
}

func TestDetect_PythonRequirements(t *testing.T) {
dir := t.TempDir()
writeFile(t, dir, "requirements.txt", "requests\n")
info := Detect(dir)
if info.Languages != "Python" {
t.Errorf("Languages = %q, want %q", info.Languages, "Python")
}
if info.TestFramework != "pytest" {
t.Errorf("TestFramework = %q, want %q", info.TestFramework, "pytest")
}
if info.PackageManager != "pip" {
t.Errorf("PackageManager = %q, want %q", info.PackageManager, "pip")
}
}

func TestDetect_PythonPoetry(t *testing.T) {
dir := t.TempDir()
writeFile(t, dir, "pyproject.toml",
"[tool.poetry]\nname = \"myapp\"\n[tool.poetry.dependencies]\npython = \"^3.11\"\n")
info := Detect(dir)
if info.PackageManager != "poetry" {
t.Errorf("PackageManager = %q, want %q", info.PackageManager, "poetry")
}
if info.Lockfile != "poetry.lock" {
t.Errorf("Lockfile = %q, want %q", info.Lockfile, "poetry.lock")
}
}

func TestDetect_UnknownProject(t *testing.T) {
dir := t.TempDir()
info := Detect(dir)
if info.Languages != "" {
t.Errorf("Languages = %q, want empty for unknown project", info.Languages)
}
if info.TestCommand != "" {
t.Errorf("TestCommand = %q, want empty for unknown project", info.TestCommand)
}
}

func TestDetect_GoTakesPrecedence(t *testing.T) {
dir := t.TempDir()
writeFile(t, dir, "go.mod", "module example\n\ngo 1.21\n")
writeFile(t, dir, "package.json", `{"scripts":{"test":"jest"}}`)
info := Detect(dir)
if info.PackageManager != "go mod" {
t.Errorf("PackageManager = %q, want %q", info.PackageManager, "go mod")
}
}
