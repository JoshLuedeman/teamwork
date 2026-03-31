package cmd

import (
	"fmt"
	"strings"

	"github.com/joshluedeman/teamwork/internal/state"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume <workflow-id>",
	Short: "Resume a workflow from a saved checkpoint",
	Long:  "Load and display a saved checkpoint for a workflow, showing where work was interrupted.\nUse --clear to delete the checkpoint without resuming.",
	Args:  cobra.ExactArgs(1),
	RunE:  runResume,
}

func init() {
	resumeCmd.Flags().Bool("clear", false, "Delete the checkpoint without resuming")
	rootCmd.AddCommand(resumeCmd)
}

func runResume(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	workflowID := args[0]
	clear, _ := cmd.Flags().GetBool("clear")

	w := cmd.OutOrStdout()

	if clear {
		if err := state.ClearCheckpoint(dir, workflowID); err != nil {
			return fmt.Errorf("resume: clear checkpoint: %w", err)
		}
		fmt.Fprintf(w, "Checkpoint cleared for workflow %q.\n", workflowID)
		return nil
	}

	cp, err := state.LoadCheckpoint(dir, workflowID)
	if err != nil {
		return fmt.Errorf("resume: load checkpoint: %w", err)
	}
	if cp == nil {
		fmt.Fprintf(w, "No checkpoint found for workflow %q.\n", workflowID)
		return nil
	}

	fmt.Fprintf(w, "Checkpoint found for workflow %q\n", workflowID)
	fmt.Fprintf(w, "  Step:    %d\n", cp.Step)
	fmt.Fprintf(w, "  Role:    %s\n", cp.Role)
	fmt.Fprintf(w, "  Saved:   %s\n", cp.SavedAt)
	if cp.PartialHandoff != "" {
		fmt.Fprintf(w, "  Partial handoff: %s\n", cp.PartialHandoff)
	}
	if len(cp.FilesModified) > 0 {
		fmt.Fprintf(w, "  Files modified: %s\n", strings.Join(cp.FilesModified, ", "))
	}
	if cp.Notes != "" {
		fmt.Fprintf(w, "  Notes: %s\n", cp.Notes)
	}
	return nil
}
