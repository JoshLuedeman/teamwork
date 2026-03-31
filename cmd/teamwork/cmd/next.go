package cmd

import (
	"fmt"
	"strings"

	"github.com/joshluedeman/teamwork/internal/handoff"
	"github.com/joshluedeman/teamwork/internal/memory"
	"github.com/joshluedeman/teamwork/internal/state"
	"github.com/joshluedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Show pending actions across workflows",
	RunE:  runNext,
}

func init() {
	rootCmd.AddCommand(nextCmd)
}

func runNext(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	actions, err := engine.Next()
	if err != nil {
		return err
	}

	if len(actions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "All workflows are up to date.")
		return nil
	}

	// Load feedback once for all actions.
	ff, _ := memory.LoadFeedback(dir)

	for _, a := range actions {
		// Load workflow state to get type for template and feedback lookup.
		ws, _ := state.Load(dir, a.WorkflowID)

		// Show open feedback for Coder steps.
		if strings.EqualFold(a.Role, "coder") && ff != nil && ws != nil {
			var openFeedback []memory.FeedbackEntry
			for _, e := range ff.Entries {
				if e.Status == "open" && containsStr(e.Domain, ws.Type) {
					openFeedback = append(openFeedback, e)
				}
			}
			if len(openFeedback) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "📋 Open feedback:")
				for _, e := range openFeedback {
					fmt.Fprintf(cmd.OutOrStdout(), "  [%s] %s: %s\n", e.Date, e.Source, e.Feedback)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Workflow: %s\n", a.WorkflowID)
		if a.Repo != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Repo: %s\n", a.Repo)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  Step %d: [%s] %s\n", a.Step, a.Role, a.Action)

		// Show template hint if a template exists for this transition.
		if ws != nil {
			def, ok := workflow.DefinitionFor(engine.Config, ws.Type)
			if ok {
				nextRole := ""
				for _, s := range def.Steps {
					if s.Number == a.Step+1 {
						nextRole = s.Role
						break
					}
				}
				tmpl := handoff.TemplateFor(ws.Type, a.Role, nextRole)
				// Only show hint if it's a specific (non-generic) template.
				if tmpl != handoff.GenericTemplate() {
					fmt.Fprintf(cmd.OutOrStdout(),
						"  Template: .teamwork/handoffs/%s/step-%02d-%s.md (use 'teamwork handoff init %s' to generate)\n",
						a.WorkflowID, a.Step, a.Role, a.WorkflowID)
				}
			}
		}

		fmt.Fprintln(cmd.OutOrStdout())
	}

	return nil
}

