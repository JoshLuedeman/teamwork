package cmd

import (
	"fmt"

	"github.com/joshluedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var approveCmd = &cobra.Command{
	Use:   "approve <workflow-id>",
	Short: "Approve the current step of a workflow",
	Args:  cobra.ExactArgs(1),
	RunE:  runApprove,
}

func init() {
	rootCmd.AddCommand(approveCmd)
}

func runApprove(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	if err := engine.Approve(args[0]); err != nil {
		return err
	}

	fmt.Printf("Approved workflow %s.\n", args[0])
	return nil
}
