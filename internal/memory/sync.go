package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	beginMarker = "<!-- BEGIN STRUCTURED MEMORY -->"
	endMarker   = "<!-- END STRUCTURED MEMORY -->"
)

// categoryTitle maps a Category to its human-readable heading for MEMORY.md.
var categoryTitle = map[Category]string{
	Patterns:     "Patterns That Work",
	Antipatterns: "Patterns to Avoid",
	Decisions:    "Key Decisions",
	Feedback:     "Reviewer Feedback",
}

// SyncToMemoryMD reads all memory entries from the four category files and
// writes a structured markdown section into MEMORY.md at the project root.
// Content outside the marker comments is preserved.
func SyncToMemoryMD(dir string) error {
	mdPath := filepath.Join(dir, "MEMORY.md")

	// Build the structured section from memory entries.
	section, err := buildStructuredSection(dir)
	if err != nil {
		return fmt.Errorf("building structured section: %w", err)
	}

	// Read existing MEMORY.md content (if any).
	existing, err := os.ReadFile(mdPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading MEMORY.md: %w", err)
	}

	var result string
	content := string(existing)

	if beginIdx := strings.Index(content, beginMarker); beginIdx != -1 {
		if endIdx := strings.Index(content, endMarker); endIdx != -1 {
			// Replace existing structured section.
			before := content[:beginIdx]
			after := content[endIdx+len(endMarker):]
			result = before + section + after
		} else {
			// Begin marker exists without end marker; append end marker.
			before := content[:beginIdx]
			after := content[beginIdx+len(beginMarker):]
			result = before + section + after
		}
	} else if len(content) > 0 {
		// No markers found; append section at the end.
		result = strings.TrimRight(content, "\n") + "\n\n" + section + "\n"
	} else {
		// No existing file; write section only.
		result = section + "\n"
	}

	return os.WriteFile(mdPath, []byte(result), 0o644)
}

// buildStructuredSection generates the markdown between (and including) the markers.
func buildStructuredSection(dir string) (string, error) {
	var sb strings.Builder

	sb.WriteString(beginMarker)
	sb.WriteString("\n")

	categories := []Category{Patterns, Antipatterns, Decisions, Feedback}
	for _, cat := range categories {
		mf, err := LoadCategory(dir, cat)
		if err != nil {
			return "", fmt.Errorf("loading category %s: %w", cat, err)
		}

		sb.WriteString("\n## ")
		sb.WriteString(categoryTitle[cat])
		sb.WriteString("\n\n")

		if len(mf.Entries) == 0 {
			sb.WriteString("*(No entries yet)*\n")
		} else {
			for _, e := range mf.Entries {
				sb.WriteString("- **")
				sb.WriteString(e.Content)
				sb.WriteString("**")
				if e.Context != "" {
					sb.WriteString(" — ")
					sb.WriteString(e.Context)
				}
				if e.Source != "" {
					sb.WriteString(" *(")
					sb.WriteString(e.Source)
					sb.WriteString(")*")
				}
				sb.WriteString("\n")
			}
		}
	}

	sb.WriteString("\n")
	sb.WriteString(endMarker)

	return sb.String(), nil
}
