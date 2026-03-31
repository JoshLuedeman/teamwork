package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joshluedeman/teamwork/internal/handoff"
	"github.com/joshluedeman/teamwork/internal/state"
	"github.com/joshluedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var handoffInitCmd = &cobra.Command{
	Use:   "init <workflow-id>",
	Short: "Generate a role-specific handoff template for the current step",
	Args:  cobra.ExactArgs(1),
	RunE:  runHandoffInit,
}

func init() {
	handoffCmd.AddCommand(handoffInitCmd)
}

func runHandoffInit(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	workflowID := args[0]

	ws, err := state.Load(dir, workflowID)
	if err != nil {
		return fmt.Errorf("loading workflow state: %w", err)
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return fmt.Errorf("loading engine: %w", err)
	}

	// Determine the next role from the workflow definition.
	def, ok := workflow.DefinitionFor(engine.Config, ws.Type)
	nextRole := ""
	if ok {
		for _, s := range def.Steps {
			if s.Number == ws.CurrentStep+1 {
				nextRole = s.Role
				break
			}
		}
	}

	tmpl := handoff.TemplateFor(ws.Type, ws.CurrentRole, nextRole)

	// Construct the handoff path: .teamwork/handoffs/<workflow-id>/step-<N>-<role>.md
	filename := fmt.Sprintf("step-%02d-%s.md", ws.CurrentStep, ws.CurrentRole)
	p := filepath.Join(dir, ".teamwork", "handoffs", workflowID, filename)

	// Skip if the file already exists.
	if _, err := os.Stat(p); err == nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: handoff file already exists, skipping: %s\n", p)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("creating handoff directory: %w", err)
	}

	if err := os.WriteFile(p, []byte(tmpl), 0o644); err != nil {
		return fmt.Errorf("writing handoff template: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created: %s\n", p)
	return nil
}
