package cmd

import (
	"fmt"

	"github.com/joshluedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var rollbackStepCmd = &cobra.Command{
	Use:   "rollback-step <workflow-id>",
	Short: "Roll back a workflow to its previous step",
	Args:  cobra.ExactArgs(1),
	RunE:  runRollbackStep,
}

func init() {
	rootCmd.AddCommand(rollbackStepCmd)
}

func runRollbackStep(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	if err := engine.RollbackStep(args[0]); err != nil {
		return err
	}

	fmt.Printf("Rolled back workflow %s to previous step.\n", args[0])
	return nil
}
