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
