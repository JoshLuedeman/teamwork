package cmd

import (
	"fmt"
	"strings"

	"github.com/joshluedeman/teamwork/internal/memory"
	"github.com/spf13/cobra"
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback",
	Short: "Manage structured reviewer feedback entries",
}

var feedbackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List feedback entries",
	RunE:  runFeedbackList,
}

var feedbackResolveCmd = &cobra.Command{
	Use:   "resolve <id>",
	Short: "Mark a feedback entry as resolved",
	Args:  cobra.ExactArgs(1),
	RunE:  runFeedbackResolve,
}

func init() {
	feedbackListCmd.Flags().String("domain", "", "Filter by domain tag")
	feedbackListCmd.Flags().String("status", "", "Filter by status: open|resolved")
	feedbackCmd.AddCommand(feedbackListCmd)
	feedbackCmd.AddCommand(feedbackResolveCmd)
	rootCmd.AddCommand(feedbackCmd)
}

func runFeedbackList(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	domain, err := cmd.Flags().GetString("domain")
	if err != nil {
		return err
	}

	statusFilter, err := cmd.Flags().GetString("status")
	if err != nil {
		return err
	}

	ff, err := memory.LoadFeedback(dir)
	if err != nil {
		return fmt.Errorf("loading feedback: %w", err)
	}

	if len(ff.Entries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No feedback entries found.")
		return nil
	}

	// Count open entries per domain for recurring detection.
	domainOpenCount := make(map[string]int)
	for _, e := range ff.Entries {
		if e.Status == "open" {
			for _, d := range e.Domain {
				domainOpenCount[d]++
			}
		}
	}

	for _, e := range ff.Entries {
		// Apply domain filter.
		if domain != "" && !containsStr(e.Domain, domain) {
			continue
		}
		// Apply status filter.
		if statusFilter != "" && e.Status != statusFilter {
			continue
		}

		// Check for recurring (more than 2 open entries in same domain).
		recurring := false
		for _, d := range e.Domain {
			if domainOpenCount[d] > 2 {
				recurring = true
				break
			}
		}

		prefix := ""
		if recurring && e.Status == "open" {
			prefix = "⚠ recurring: "
		}

		domains := strings.Join(e.Domain, ", ")
		fmt.Fprintf(cmd.OutOrStdout(), "%s[%s] %s (%s) [%s]\n  %s\n\n",
			prefix, e.Status, e.ID, domains, e.Date, e.Feedback)
	}

	return nil
}

func runFeedbackResolve(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	id := args[0]

	ff, err := memory.LoadFeedback(dir)
	if err != nil {
		return fmt.Errorf("loading feedback: %w", err)
	}

	found := false
	for i, e := range ff.Entries {
		if e.ID == id {
			ff.Entries[i].Status = "resolved"
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("feedback entry %q not found", id)
	}

	if err := memory.SaveFeedback(dir, ff); err != nil {
		return fmt.Errorf("saving feedback: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Resolved: %s\n", id)
	return nil
}

// containsStr reports whether slice contains s.
func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
