// Package handoff manages handoff artifacts in .teamwork/handoffs/.
//
// Handoff artifacts are markdown files that capture the output of one workflow
// step, structured so the next role can start without re-reading the entire repo.
package handoff

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Artifact represents a handoff between two roles in a workflow step.
type Artifact struct {
	WorkflowID string
	Step       int
	Role       string
	NextRole   string
	Date       string
	Summary    string
	Artifacts  []string // Files created/modified
	Context    string   // Context for next role
	Criteria   []CriterionStatus
	Questions  []string
	GatePassed bool
}

// CriterionStatus tracks whether an acceptance criterion has been met.
type CriterionStatus struct {
	Description string
	Met         bool
}

// New creates a new Artifact with the current timestamp.
func New(workflowID string, step int, role, nextRole string) *Artifact {
	return &Artifact{
		WorkflowID: workflowID,
		Step:       step,
		Role:       role,
		NextRole:   nextRole,
		Date:       time.Now().UTC().Format(time.RFC3339),
	}
}

// Render renders the artifact as a markdown handoff document following the
// template defined in protocols.md.
func (a *Artifact) Render() string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Handoff: %s → %s\n\n", a.Role, a.NextRole)
	fmt.Fprintf(&b, "**Workflow:** %s\n", a.WorkflowID)
	fmt.Fprintf(&b, "**Step:** %d\n", a.Step)
	fmt.Fprintf(&b, "**Date:** %s\n\n", a.Date)

	b.WriteString("## Summary\n\n")
	b.WriteString(sectionText(a.Summary))
	b.WriteString("\n\n")

	b.WriteString("## Artifacts Produced\n\n")
	if len(a.Artifacts) == 0 {
		b.WriteString("None\n")
	} else {
		for _, art := range a.Artifacts {
			fmt.Fprintf(&b, "- %s\n", art)
		}
	}
	b.WriteString("\n")

	b.WriteString("## Context for Next Role\n\n")
	b.WriteString(sectionText(a.Context))
	b.WriteString("\n\n")

	b.WriteString("## Acceptance Criteria Status\n\n")
	if len(a.Criteria) == 0 {
		b.WriteString("None\n")
	} else {
		for _, c := range a.Criteria {
			mark := " "
			if c.Met {
				mark = "x"
			}
			fmt.Fprintf(&b, "- [%s] %s\n", mark, c.Description)
		}
	}
	b.WriteString("\n")

	b.WriteString("## Open Questions or Risks\n\n")
	if len(a.Questions) == 0 {
		b.WriteString("None\n")
	} else {
		for _, q := range a.Questions {
			fmt.Fprintf(&b, "- %s\n", q)
		}
	}
	b.WriteString("\n")

	b.WriteString("## Quality Gate\n\n")
	gate := " "
	if a.GatePassed {
		gate = "x"
	}
	fmt.Fprintf(&b, "- [%s] Handoff reviewed by orchestrator\n", gate)
	fmt.Fprintf(&b, "- [%s] All required fields populated\n", gate)
	fmt.Fprintf(&b, "- [%s] Artifacts exist at referenced paths\n", gate)

	return b.String()
}

// Path returns the file path for a handoff artifact.
func Path(dir, workflowID string, step int, role string) string {
	filename := fmt.Sprintf("%02d-%s.md", step, role)
	return filepath.Join(dir, ".teamwork", "handoffs", workflowID, filename)
}

// validateWorkflowID checks that a workflow ID does not contain path traversal.
func validateWorkflowID(id string) error {
	cleaned := filepath.Clean(id)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return fmt.Errorf("handoff: invalid workflow ID %q: contains path traversal", id)
	}
	return nil
}

// Save writes the artifact to .teamwork/handoffs/<workflow-id>/<step>-<role>.md.
func (a *Artifact) Save(dir string) error {
	if err := validateWorkflowID(a.WorkflowID); err != nil {
		return err
	}
	p := Path(dir, a.WorkflowID, a.Step, a.Role)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("creating handoff directory: %w", err)
	}
	return os.WriteFile(p, []byte(a.Render()), 0o644)
}

// Load reads and parses a handoff artifact from disk.
func Load(dir, workflowID string, step int, role string) (*Artifact, error) {
	p := Path(dir, workflowID, step, role)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("reading handoff artifact: %w", err)
	}
	return parse(string(data))
}

// LoadAll loads all handoff artifacts for a workflow, sorted by step number.
func LoadAll(dir, workflowID string) ([]*Artifact, error) {
	handoffDir := filepath.Join(dir, ".teamwork", "handoffs", workflowID)
	entries, err := os.ReadDir(handoffDir)
	if err != nil {
		return nil, fmt.Errorf("reading handoff directory: %w", err)
	}

	var artifacts []*Artifact
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		step, role, ok := parseFilename(e.Name())
		if !ok {
			continue
		}
		a, err := Load(dir, workflowID, step, role)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", e.Name(), err)
		}
		artifacts = append(artifacts, a)
	}

	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Step < artifacts[j].Step
	})
	return artifacts, nil
}

// Validate checks an artifact for completeness and returns a list of
// validation errors. Returns nil if the artifact is valid.
func Validate(a *Artifact) []string {
	var errs []string
	if a.WorkflowID == "" {
		errs = append(errs, "workflow ID is required")
	}
	if a.Step < 1 {
		errs = append(errs, "step must be >= 1")
	}
	if a.Role == "" {
		errs = append(errs, "role is required")
	}
	if a.NextRole == "" {
		errs = append(errs, "next role is required")
	}
	if a.Date == "" {
		errs = append(errs, "date is required")
	}
	if strings.TrimSpace(a.Summary) == "" {
		errs = append(errs, "summary is required")
	}
	if strings.TrimSpace(a.Context) == "" {
		errs = append(errs, "context for next role is required")
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// sectionText returns the text for a markdown section, defaulting to "None".
func sectionText(s string) string {
	if strings.TrimSpace(s) == "" {
		return "None"
	}
	return s
}

// parseFilename extracts step number and role from a handoff filename like "01-planner.md".
func parseFilename(name string) (int, string, bool) {
	name = strings.TrimSuffix(name, ".md")
	parts := strings.SplitN(name, "-", 2)
	if len(parts) != 2 {
		return 0, "", false
	}
	step, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", false
	}
	return step, parts[1], true
}

// parse converts raw markdown into an Artifact.
func parse(content string) (*Artifact, error) {
	a := &Artifact{}

	// Parse header: # Handoff: Role → NextRole
	headerRe := regexp.MustCompile(`(?m)^# Handoff:\s*(.+?)\s*→\s*(.+?)\s*$`)
	if m := headerRe.FindStringSubmatch(content); m != nil {
		a.Role = m[1]
		a.NextRole = m[2]
	}

	// Parse metadata fields
	a.WorkflowID = parseField(content, "Workflow")
	a.Date = parseField(content, "Date")
	if s := parseField(content, "Step"); s != "" {
		step, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid step number %q: %w", s, err)
		}
		a.Step = step
	}

	// Parse sections
	sections := parseSections(content)

	a.Summary = cleanSection(sections["Summary"])
	a.Context = cleanSection(sections["Context for Next Role"])

	// Parse artifacts list
	a.Artifacts = parseBulletList(sections["Artifacts Produced"])

	// Parse acceptance criteria
	a.Criteria = parseCriteria(sections["Acceptance Criteria Status"])

	// Parse questions
	a.Questions = parseBulletList(sections["Open Questions or Risks"])

	// Parse quality gate
	gateSection := sections["Quality Gate"]
	a.GatePassed = strings.Contains(gateSection, "[x]") && !strings.Contains(gateSection, "[ ]")

	return a, nil
}

// parseField extracts a **Field:** value from markdown.
func parseField(content, field string) string {
	re := regexp.MustCompile(`(?m)\*\*` + regexp.QuoteMeta(field) + `:\*\*\s*(.+)$`)
	if m := re.FindStringSubmatch(content); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// parseSections splits markdown into a map of section heading → body text.
func parseSections(content string) map[string]string {
	sections := make(map[string]string)
	re := regexp.MustCompile(`(?m)^## (.+)$`)
	matches := re.FindAllStringSubmatchIndex(content, -1)

	for i, m := range matches {
		heading := content[m[2]:m[3]]
		bodyStart := m[1]
		var bodyEnd int
		if i+1 < len(matches) {
			bodyEnd = matches[i+1][0]
		} else {
			bodyEnd = len(content)
		}
		sections[heading] = content[bodyStart:bodyEnd]
	}
	return sections
}

// cleanSection trims whitespace and returns empty string for "None" sections.
func cleanSection(s string) string {
	s = strings.TrimSpace(s)
	if s == "None" || s == "" {
		return ""
	}
	return s
}

// parseBulletList extracts items from a markdown bullet list.
func parseBulletList(section string) []string {
	if cleanSection(section) == "" {
		return nil
	}
	var items []string
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			item := strings.TrimPrefix(line, "- ")
			// Strip checkbox markers if present
			item = strings.TrimPrefix(item, "[x] ")
			item = strings.TrimPrefix(item, "[ ] ")
			if item != "" {
				items = append(items, item)
			}
		}
	}
	return items
}

// parseCriteria extracts acceptance criteria with their met/unmet status.
func parseCriteria(section string) []CriterionStatus {
	if cleanSection(section) == "" {
		return nil
	}
	var criteria []CriterionStatus
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- [x] ") {
			criteria = append(criteria, CriterionStatus{
				Description: strings.TrimPrefix(line, "- [x] "),
				Met:         true,
			})
		} else if strings.HasPrefix(line, "- [ ] ") {
			criteria = append(criteria, CriterionStatus{
				Description: strings.TrimPrefix(line, "- [ ] "),
				Met:         false,
			})
		}
	}
	return criteria
}
