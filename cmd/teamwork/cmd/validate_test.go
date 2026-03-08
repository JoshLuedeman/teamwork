package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func init() {
	// When running tests directly (bypassing cobra Execute), persistent flags
	// from rootCmd are not merged into child commands. Register "dir" locally
	// on validateCmd so tests can set it without going through Execute().
	if validateCmd.Flags().Lookup("dir") == nil {
		validateCmd.Flags().StringP("dir", "d", ".", "Project root directory")
	}
}

func setupValidDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	twDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(twDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "project:\n  name: test\n  repo: owner/repo\nroles:\n  core: [coder]\n"
	if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old
	return string(out)
}

func TestCIFlagRegistered(t *testing.T) {
	f := validateCmd.Flags().Lookup("ci")
	if f == nil {
		t.Fatal("--ci flag not registered on validate command")
	}
	if f.DefValue != "false" {
		t.Errorf("--ci default = %q, want %q", f.DefValue, "false")
	}
}

func TestRunValidate_CIOutput_PassingChecks(t *testing.T) {
	dir := setupValidDir(t)
	_ = validateCmd.Flags().Set("dir", dir)
	_ = validateCmd.Flags().Set("ci", "true")
	_ = validateCmd.Flags().Set("json", "false")
	_ = validateCmd.Flags().Set("quiet", "false")

	var runErr error
	output := captureStdout(t, func() {
		runErr = runValidate(validateCmd, nil)
	})

	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		t.Fatal("expected CI output lines, got none")
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "PASS") {
			t.Errorf("expected line to start with PASS, got: %q", line)
		}
	}
	if strings.Contains(output, "\u2713") || strings.Contains(output, "\u2717") {
		t.Error("CI output should not contain decorative Unicode characters")
	}
	if strings.Contains(output, " passed, ") {
		t.Error("CI output should not contain summary line")
	}
}

func TestRunValidate_CIOutput_FailingChecks(t *testing.T) {
	dir := t.TempDir()
	twDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(twDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = validateCmd.Flags().Set("dir", dir)
	_ = validateCmd.Flags().Set("ci", "true")
	_ = validateCmd.Flags().Set("json", "false")
	_ = validateCmd.Flags().Set("quiet", "false")

	var runErr error
	output := captureStdout(t, func() {
		runErr = runValidate(validateCmd, nil)
	})

	exitErr, ok := runErr.(*ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T: %v", runErr, runErr)
	}
	if exitErr.Code != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.Code)
	}
	if !strings.Contains(output, "FAIL") {
		t.Errorf("expected FAIL in CI output, got:\n%s", output)
	}
}

func TestRunValidate_CIOutput_Format(t *testing.T) {
	dir := setupValidDir(t)
	_ = validateCmd.Flags().Set("dir", dir)
	_ = validateCmd.Flags().Set("ci", "true")
	_ = validateCmd.Flags().Set("json", "false")
	_ = validateCmd.Flags().Set("quiet", "false")

	output := captureStdout(t, func() {
		_ = runValidate(validateCmd, nil)
	})
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			t.Errorf("CI output line missing colon separator: %q", line)
		}
	}
}
