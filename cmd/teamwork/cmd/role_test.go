package cmd

import (
"bytes"
"os"
"path/filepath"
"strings"
"testing"

"github.com/spf13/pflag"
)

// executeRoleCmd runs a "role create" command with the given dir and extra args,
// returning the combined output and any error.
func executeRoleCmd(t *testing.T, dir string, args ...string) (string, error) {
t.Helper()

// Reset flag state to avoid pollution between tests.
roleCreateCmd.Flags().VisitAll(func(f *pflag.Flag) {
f.Value.Set(f.DefValue)
f.Changed = false
})

buf := new(bytes.Buffer)
rootCmd.SetOut(buf)
rootCmd.SetErr(buf)
rootCmd.SetArgs(append([]string{"role", "create", "--dir", dir}, args...))

err := rootCmd.Execute()
return buf.String(), err
}

func TestRoleCreate_ValidName(t *testing.T) {
dir := t.TempDir()

out, err := executeRoleCmd(t, dir, "data-engineer")
if err != nil {
t.Fatalf("unexpected error: %v", err)
}

// Verify success message.
if !strings.Contains(out, "Created custom agent role") {
t.Errorf("expected success message, got:\n%s", out)
}

// Verify file was created.
filePath := filepath.Join(dir, ".github", "agents", "data-engineer.agent.md")
content, err := os.ReadFile(filePath)
if err != nil {
t.Fatalf("expected file to exist at %s: %v", filePath, err)
}

// Verify template rendering.
s := string(content)
if !strings.Contains(s, "name: data-engineer") {
t.Error("expected file to contain 'name: data-engineer'")
}
if !strings.Contains(s, "# Role: Data Engineer") {
t.Error("expected file to contain title 'Data Engineer'")
}
if !strings.Contains(s, "You are the Data Engineer.") {
t.Error("expected file to contain identity section")
}
if !strings.Contains(s, "**Tier:** Standard") {
t.Error("expected default tier to be Standard")
}
if !strings.Contains(s, `description: "A custom agent role"`) {
t.Error("expected default description")
}
}

func TestRoleCreate_BuiltinRejected(t *testing.T) {
dir := t.TempDir()

for _, name := range []string{"coder", "planner", "security-auditor", "product-owner", "qa-lead"} {
_, err := executeRoleCmd(t, dir, name)
if err == nil {
t.Errorf("expected error for built-in role %q, got nil", name)
continue
}
if !strings.Contains(err.Error(), "conflicts with a built-in role") {
t.Errorf("expected conflict error for %q, got: %v", name, err)
}
}
}

func TestRoleCreate_InvalidNameRejected(t *testing.T) {
dir := t.TempDir()

cases := []struct {
name string
desc string
}{
{"DataEngineer", "uppercase letters"},
{"trailing-", "trailing hyphen"},
{"data--engineer", "double hyphen"},
{"123start", "starts with number"},
{"data_engineer", "underscore"},
{"data.engineer", "dot"},
}

for _, tc := range cases {
t.Run(tc.desc, func(t *testing.T) {
_, err := executeRoleCmd(t, dir, tc.name)
if err == nil {
t.Fatalf("expected error for name %q (%s), got nil", tc.name, tc.desc)
}
if !strings.Contains(err.Error(), "invalid role name") {
t.Errorf("expected 'invalid role name' error for %q, got: %v", tc.name, err)
}
})
}
}

func TestRoleCreate_DescriptionFlag(t *testing.T) {
dir := t.TempDir()

_, err := executeRoleCmd(t, dir, "--description", "Handles data pipelines", "data-engineer")
if err != nil {
t.Fatalf("unexpected error: %v", err)
}

filePath := filepath.Join(dir, ".github", "agents", "data-engineer.agent.md")
content, err := os.ReadFile(filePath)
if err != nil {
t.Fatalf("expected file to exist: %v", err)
}

if !strings.Contains(string(content), `description: "Handles data pipelines"`) {
t.Errorf("expected description in frontmatter, got:\n%s", string(content))
}
}

func TestRoleCreate_TierFlag(t *testing.T) {
cases := []struct {
tier     string
expected string
}{
{"premium", "Premium"},
{"standard", "Standard"},
{"fast", "Fast"},
}

for _, tc := range cases {
t.Run(tc.tier, func(t *testing.T) {
dir := t.TempDir()

_, err := executeRoleCmd(t, dir, "--tier", tc.tier, "test-role")
if err != nil {
t.Fatalf("unexpected error: %v", err)
}

filePath := filepath.Join(dir, ".github", "agents", "test-role.agent.md")
content, err := os.ReadFile(filePath)
if err != nil {
t.Fatalf("expected file to exist: %v", err)
}

expected := "**Tier:** " + tc.expected
if !strings.Contains(string(content), expected) {
t.Errorf("expected %q in content, got:\n%s", expected, string(content))
}
})
}
}

func TestRoleCreate_InvalidTierRejected(t *testing.T) {
dir := t.TempDir()

_, err := executeRoleCmd(t, dir, "--tier", "mega", "test-role")
if err == nil {
t.Fatal("expected error for invalid tier, got nil")
}
if !strings.Contains(err.Error(), "invalid tier") {
t.Errorf("expected 'invalid tier' error, got: %v", err)
}
}

func TestRoleCreate_FileAlreadyExists(t *testing.T) {
dir := t.TempDir()

// Pre-create the file.
agentsDir := filepath.Join(dir, ".github", "agents")
if err := os.MkdirAll(agentsDir, 0o755); err != nil {
t.Fatal(err)
}
if err := os.WriteFile(filepath.Join(agentsDir, "data-engineer.agent.md"), []byte("existing"), 0o644); err != nil {
t.Fatal(err)
}

_, err := executeRoleCmd(t, dir, "data-engineer")
if err == nil {
t.Fatal("expected error for existing file, got nil")
}
if !strings.Contains(err.Error(), "already exists") {
t.Errorf("expected 'already exists' error, got: %v", err)
}
}

func TestRoleCreate_CreatesAgentsDirectory(t *testing.T) {
dir := t.TempDir()

// Ensure .github/agents/ does not exist yet.
agentsDir := filepath.Join(dir, ".github", "agents")
if _, err := os.Stat(agentsDir); err == nil {
t.Fatal("expected agents dir to not exist before test")
}

_, err := executeRoleCmd(t, dir, "my-role")
if err != nil {
t.Fatalf("unexpected error: %v", err)
}

// Verify directory was created.
info, err := os.Stat(agentsDir)
if err != nil {
t.Fatalf("expected agents dir to be created: %v", err)
}
if !info.IsDir() {
t.Error("expected agents path to be a directory")
}

// Verify file exists inside.
if _, err := os.Stat(filepath.Join(agentsDir, "my-role.agent.md")); err != nil {
t.Errorf("expected role file to exist: %v", err)
}
}

func TestRoleCreate_SingleWordName(t *testing.T) {
dir := t.TempDir()

out, err := executeRoleCmd(t, dir, "analyst")
if err != nil {
t.Fatalf("unexpected error: %v", err)
}

if !strings.Contains(out, "Created custom agent role") {
t.Errorf("expected success message, got:\n%s", out)
}

filePath := filepath.Join(dir, ".github", "agents", "analyst.agent.md")
content, err := os.ReadFile(filePath)
if err != nil {
t.Fatalf("expected file to exist: %v", err)
}

if !strings.Contains(string(content), "# Role: Analyst") {
t.Error("expected title 'Analyst' for single-word name")
}
}

func TestRoleCreate_NextStepsOutput(t *testing.T) {
dir := t.TempDir()

out, err := executeRoleCmd(t, dir, "my-role")
if err != nil {
t.Fatalf("unexpected error: %v", err)
}

if !strings.Contains(out, "Next steps:") {
t.Error("expected 'Next steps:' in output")
}
if !strings.Contains(out, "CUSTOMIZE") {
t.Error("expected CUSTOMIZE mention in next steps")
}
if !strings.Contains(out, "TODO") {
t.Error("expected TODO mention in next steps")
}
}
