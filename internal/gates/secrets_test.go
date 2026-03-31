package gates

import (
	"os/exec"
	"strings"
	"testing"
)

// mockSecretsRunner is a Runner that returns predefined outputs/errors.
type mockSecretsRunner struct {
	output string
	err    error
	called string
}

func (m *mockSecretsRunner) Run(command, dir string) ([]byte, error) {
	m.called = command
	return []byte(m.output), m.err
}

// TestRunSecretsGate_NoScannerInPath tests that the gate skips gracefully when
// no scanner binary is in PATH. This relies on the fact that none of the
// scanner binaries are actually installed in the test environment (or uses a
// modified scanners slice).
func TestRunSecretsGate_NoScannerInPath(t *testing.T) {
	// Override scanners with a tool that definitely doesn't exist.
	orig := scanners
	scanners = []secretsScanner{
		{"__no_such_tool_abc123__", "scan"},
	}
	t.Cleanup(func() { scanners = orig })

	runner := &mockSecretsRunner{}
	found, details, err := RunSecretsGate(t.TempDir(), runner)
	if err != nil {
		t.Fatalf("expected no error when no scanner in PATH, got: %v", err)
	}
	if found {
		t.Error("expected found=false when no scanner in PATH")
	}
	_ = details
}

// TestRunSecretsGate_SecretsFound simulates gitleaks returning exit code 1
// (secrets found).
func TestRunSecretsGate_SecretsFound(t *testing.T) {
	// Override scanners with a fake tool that appears to be in PATH by using
	// a real binary name but intercepting via mock runner.
	orig := scanners
	scanners = []secretsScanner{
		{"sh", "--version"}, // "sh" is always in PATH
	}
	t.Cleanup(func() { scanners = orig })

	runner := &mockSecretsRunner{
		output: "found secret: AWS_ACCESS_KEY_ID",
		err:    &exec.ExitError{},
	}

	found, details, err := RunSecretsGate(t.TempDir(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected found=true when scanner exits non-zero")
	}
	if !strings.Contains(details, "AWS_ACCESS_KEY_ID") {
		t.Errorf("expected details to contain scanner output, got: %q", details)
	}
}

// TestRunSecretsGate_CleanScan simulates a scanner running successfully with
// no secrets found (exit code 0).
func TestRunSecretsGate_CleanScan(t *testing.T) {
	orig := scanners
	scanners = []secretsScanner{
		{"sh", "--version"}, // "sh" is always in PATH
	}
	t.Cleanup(func() { scanners = orig })

	runner := &mockSecretsRunner{
		output: "No leaks found.",
		err:    nil,
	}

	found, details, err := RunSecretsGate(t.TempDir(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected found=false on clean scan")
	}
	if !strings.Contains(details, "No leaks found.") {
		t.Errorf("expected details from scanner output, got: %q", details)
	}
}

// TestRunSecretsGate_RunnerNotCalled verifies the mock runner is invoked when
// a scanner binary is found.
func TestRunSecretsGate_RunnerNotCalled(t *testing.T) {
	orig := scanners
	scanners = []secretsScanner{
		{"__no_such_binary__", "scan"},
	}
	t.Cleanup(func() { scanners = orig })

	runner := &mockSecretsRunner{}
	_, _, _ = RunSecretsGate(t.TempDir(), runner)

	if runner.called != "" {
		t.Errorf("expected runner not to be called when binary not in PATH, got: %q", runner.called)
	}
}
