package handoff

import "testing"

func TestTemplateForKnownTransition(t *testing.T) {
	tmpl := TemplateFor("feature", "coder", "tester")
	if tmpl == "" {
		t.Fatal("expected non-empty template for feature/coder->tester")
	}
	if want := "## Files Changed"; !contains2(tmpl, want) {
		t.Errorf("expected template to contain %q", want)
	}
	if want := "## How to Test"; !contains2(tmpl, want) {
		t.Errorf("expected template to contain %q", want)
	}
	if want := "## Edge Cases"; !contains2(tmpl, want) {
		t.Errorf("expected template to contain %q", want)
	}
	if want := "## Known Limitations"; !contains2(tmpl, want) {
		t.Errorf("expected template to contain %q", want)
	}
}

func TestTemplateForUnknownTransitionReturnsGeneric(t *testing.T) {
	tmpl := TemplateFor("unknownworkflow", "roleA", "roleB")
	if tmpl == "" {
		t.Fatal("expected non-empty generic template for unknown transition")
	}
	// Generic template should have at least a summary section
	if want := "## Summary"; !contains2(tmpl, want) {
		t.Errorf("expected generic template to contain %q", want)
	}
}

func TestTemplateForSecurityAuditorToReviewer(t *testing.T) {
	tmpl := TemplateFor("feature", "security-auditor", "reviewer")
	if want := "## Findings"; !contains2(tmpl, want) {
		t.Errorf("expected template to contain %q", want)
	}
	if want := "## Risk Level"; !contains2(tmpl, want) {
		t.Errorf("expected template to contain %q", want)
	}
	if want := "## Mitigations Applied"; !contains2(tmpl, want) {
		t.Errorf("expected template to contain %q", want)
	}
}

func TestTemplateForReviewerToHuman(t *testing.T) {
	tmpl := TemplateFor("feature", "reviewer", "human")
	if want := "## Changes Requested"; !contains2(tmpl, want) {
		t.Errorf("expected template to contain %q", want)
	}
}

func TestTemplateForTesterToSecurityAuditor(t *testing.T) {
	tmpl := TemplateFor("feature", "tester", "security-auditor")
	if want := "## Test Results"; !contains2(tmpl, want) {
		t.Errorf("expected template to contain %q", want)
	}
	if want := "## Coverage"; !contains2(tmpl, want) {
		t.Errorf("expected template to contain %q", want)
	}
}

// contains2 is a simple helper (avoids importing strings in test file by reusing package-level).
func contains2(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}
