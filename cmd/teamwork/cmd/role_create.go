package cmd

import (
"fmt"
"os"
"path/filepath"
"regexp"
"strings"
"text/template"

"github.com/spf13/cobra"
)

// builtinRoles lists role names reserved for built-in and planned agents.
var builtinRoles = map[string]bool{
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

var kebabCaseRe = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

type roleTemplateData struct {
Name        string
Title       string
Description string
Tier        string
}

const roleTemplate = `---
name: {{.Name}}
description: "{{.Description}}"
tools: ["read", "search", "edit"]
---

# Role: {{.Title}}

## Identity

You are the {{.Title}}. <!-- TODO: Define this role's identity and purpose -->

## Project Knowledge
<!-- CUSTOMIZE: Replace the placeholders below with your project's details -->
- **Tech Stack:** [e.g., React 18, TypeScript, Node.js 20, PostgreSQL 16]

## Model Requirements

- **Tier:** {{.Tier}}
- **Why:** <!-- TODO: Explain why this tier is appropriate -->
- **Key capabilities needed:** <!-- TODO: List key capabilities -->

## MCP Tools
- **GitHub MCP** — ` + "`" + `list_issues` + "`" + `, ` + "`" + `search_code` + "`" + ` — search and track work

## Responsibilities

- <!-- TODO: Define primary responsibilities -->

## Boundaries

### ✅ Always
- <!-- TODO: Define mandatory behaviors -->

### ⚠️ Ask First
- <!-- TODO: Define behaviors requiring human approval -->

### 🚫 Never
- <!-- TODO: Define prohibited behaviors -->

## Quality Bar

A {{.Title}} handoff is complete when:
- <!-- TODO: Define completion criteria -->
`

var roleCreateCmd = &cobra.Command{
Use:   "create <name>",
Short: "Scaffold a new custom agent role",
Args:  cobra.ExactArgs(1),
RunE:  runRoleCreate,
}

func init() {
roleCreateCmd.Flags().String("description", "", "Short description for the agent")
roleCreateCmd.Flags().String("tier", "standard", "Model tier (premium, standard, fast)")
roleCmd.AddCommand(roleCreateCmd)
}

// toTitleCase converts a kebab-case string to Title Case.
// e.g. "data-engineer" → "Data Engineer"
func toTitleCase(kebab string) string {
parts := strings.Split(kebab, "-")
for i, p := range parts {
if len(p) > 0 {
parts[i] = strings.ToUpper(p[:1]) + p[1:]
}
}
return strings.Join(parts, " ")
}

func runRoleCreate(cmd *cobra.Command, args []string) error {
name := args[0]

// Validate name format.
if !kebabCaseRe.MatchString(name) {
return fmt.Errorf("invalid role name %q: must be lowercase kebab-case (letters, numbers, hyphens)", name)
}

// Check for conflicts with built-in roles.
if builtinRoles[name] {
return fmt.Errorf("role name %q conflicts with a built-in role", name)
}

dir, err := cmd.Flags().GetString("dir")
if err != nil {
return err
}

description, err := cmd.Flags().GetString("description")
if err != nil {
return err
}
if description == "" {
description = "A custom agent role"
}

tier, err := cmd.Flags().GetString("tier")
if err != nil {
return err
}
switch tier {
case "premium", "standard", "fast":
// valid
default:
return fmt.Errorf("invalid tier %q: must be premium, standard, or fast", tier)
}
// Capitalize tier for display (e.g. "standard" → "Standard").
tier = strings.ToUpper(tier[:1]) + tier[1:]

// Build file path.
agentsDir := filepath.Join(dir, ".github", "agents")
filePath := filepath.Join(agentsDir, name+".agent.md")

// Check the file doesn't already exist.
if _, err := os.Stat(filePath); err == nil {
return fmt.Errorf("file already exists: %s", filePath)
}

// Create agents directory if it doesn't exist.
if err := os.MkdirAll(agentsDir, 0o755); err != nil {
return fmt.Errorf("failed to create directory %s: %w", agentsDir, err)
}

// Render template.
data := roleTemplateData{
Name:        name,
Title:       toTitleCase(name),
Description: description,
Tier:        tier,
}

tmpl, err := template.New("role").Parse(roleTemplate)
if err != nil {
return fmt.Errorf("failed to parse template: %w", err)
}

f, err := os.Create(filePath)
if err != nil {
return fmt.Errorf("failed to create file %s: %w", filePath, err)
}
defer f.Close()

if err := tmpl.Execute(f, data); err != nil {
return fmt.Errorf("failed to render template: %w", err)
}

out := cmd.OutOrStdout()
fmt.Fprintf(out, "Created custom agent role: %s\n", filePath)
fmt.Fprintf(out, "\nNext steps:\n")
fmt.Fprintf(out, "  1. Edit %s to define the role's identity and responsibilities\n", filePath)
fmt.Fprintf(out, "  2. Replace CUSTOMIZE and TODO placeholders with your project details\n")
fmt.Fprintf(out, "  3. Run 'teamwork validate' to check your configuration\n")

return nil
}
