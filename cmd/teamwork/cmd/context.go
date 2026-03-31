package cmd

import (
	"fmt"

	agentcontext "github.com/joshluedeman/teamwork/internal/context"
	"github.com/joshluedeman/teamwork/internal/metrics"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context <workflow-id>",
	Short: "Assemble agent context for a workflow step",
	Args:  cobra.ExactArgs(1),
	RunE:  runContext,
}

func init() {
	contextCmd.Flags().Int("step", 0, "Step number (0 = current step)")
	rootCmd.AddCommand(contextCmd)
}

func runContext(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	workflowID := args[0]

	step, err := cmd.Flags().GetInt("step")
	if err != nil {
		return err
	}

	pkg, err := agentcontext.Assemble(dir, workflowID, step)
	if err != nil {
		return fmt.Errorf("assembling context: %w", err)
	}

	fmt.Fprint(cmd.OutOrStdout(), pkg.Render())

	// Log a metrics event for context assembly.
	_ = metrics.Log(dir, workflowID, metrics.Event{
		Step:   pkg.Step,
		Role:   pkg.Role,
		Action: "context_assembled",
		Detail: fmt.Sprintf("tokens~%d", pkg.TokenEstimate),
	})

	return nil
}
