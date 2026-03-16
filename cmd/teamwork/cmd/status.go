package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/joshluedeman/teamwork/internal/state"
	"github.com/joshluedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active workflows",
	Long:  "Show workflow status with optional filtering by status, type, and output format.",
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().String("status", "", "Filter by workflow status (active, blocked, completed, failed, cancelled)")
	statusCmd.Flags().String("type", "", "Filter by workflow type (e.g., feature, bugfix, hotfix)")
	statusCmd.Flags().StringP("format", "f", "table", "Output format: table, json, yaml")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	statusFilter, _ := cmd.Flags().GetString("status")
	typeFilter, _ := cmd.Flags().GetString("type")
	format, _ := cmd.Flags().GetString("format")

	if err := validateFormat(format); err != nil {
		return err
	}
	if statusFilter != "" {
		if err := validateStatusFilter(statusFilter); err != nil {
			return err
		}
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	workflows, err := engine.Status()
	if err != nil {
		return err
	}

	workflows = state.Filter(workflows, statusFilter, typeFilter)

	w := cmd.OutOrStdout()

	switch format {
	case "json":
		return renderJSON(w, workflows)
	case "yaml":
		return renderYAML(w, workflows)
	default:
		return renderTable(w, workflows)
	}
}

// validateFormat checks that the format flag is one of the allowed values.
func validateFormat(format string) error {
	switch format {
	case "table", "json", "yaml":
		return nil
	default:
		return fmt.Errorf("unknown format %q: expected table, json, or yaml", format)
	}
}

// validateStatusFilter checks that the status flag matches a known status constant.
func validateStatusFilter(s string) error {
	switch s {
	case state.StatusActive, state.StatusBlocked, state.StatusCompleted,
		state.StatusFailed, state.StatusCancelled:
		return nil
	default:
		return fmt.Errorf("unknown status %q: expected active, blocked, completed, failed, or cancelled", s)
	}
}

// renderJSON writes workflows as a JSON array to w.
func renderJSON(w interface{ Write([]byte) (int, error) }, workflows []*state.WorkflowState) error {
	// Emit empty array instead of null when no results.
	if workflows == nil {
		workflows = []*state.WorkflowState{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(workflows)
}

// renderYAML writes workflows as a YAML document to w.
func renderYAML(w interface{ Write([]byte) (int, error) }, workflows []*state.WorkflowState) error {
	if workflows == nil {
		workflows = []*state.WorkflowState{}
	}
	data, err := yaml.Marshal(workflows)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	_, err = w.Write(data)
	return err
}

// renderTable writes workflows as a human-readable table to w.
func renderTable(w interface{ Write([]byte) (int, error) }, workflows []*state.WorkflowState) error {
	if len(workflows) == 0 {
		fmt.Fprintln(w, "No active workflows.")
		return nil
	}

	// Check if any workflow has repo info on its current step.
	hasRepo := false
	for _, wf := range workflows {
		for _, s := range wf.Steps {
			if s.Step == wf.CurrentStep && s.Repo != "" {
				hasRepo = true
				break
			}
		}
		if hasRepo {
			break
		}
	}

	if hasRepo {
		fmt.Fprintf(w, "%-36s  %-10s  %-12s  %-14s  %-14s  %-12s  %s\n",
			"ID", "Type", "Status", "Current Step", "Current Role", "Repo", "Updated")
		fmt.Fprintln(w, "------------------------------------  ----------  ------------  --------------  --------------  ------------  --------------------")
	} else {
		fmt.Fprintf(w, "%-36s  %-10s  %-12s  %-14s  %-14s  %s\n",
			"ID", "Type", "Status", "Current Step", "Current Role", "Updated")
		fmt.Fprintln(w, "------------------------------------  ----------  ------------  --------------  --------------  --------------------")
	}

	for _, wf := range workflows {
		repo := ""
		for _, s := range wf.Steps {
			if s.Step == wf.CurrentStep && s.Repo != "" {
				repo = s.Repo
				break
			}
		}

		if hasRepo {
			fmt.Fprintf(w, "%-36s  %-10s  %-12s  %-14d  %-14s  %-12s  %s\n",
				wf.ID, wf.Type, wf.Status, wf.CurrentStep, wf.CurrentRole, repo, wf.UpdatedAt)
		} else {
			fmt.Fprintf(w, "%-36s  %-10s  %-12s  %-14d  %-14s  %s\n",
				wf.ID, wf.Type, wf.Status, wf.CurrentStep, wf.CurrentRole, wf.UpdatedAt)
		}
	}

	return nil
}
