package cmd

import (
	"fmt"

	"github.com/joshluedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var failCmd = &cobra.Command{
	Use:   "fail <workflow-id>",
	Short: "Mark a workflow as failed with a reason",
	Args:  cobra.ExactArgs(1),
	RunE:  runFail,
}

func init() {
	failCmd.Flags().StringP("reason", "r", "", "Reason for failure (required)")
	_ = failCmd.MarkFlagRequired("reason")
	rootCmd.AddCommand(failCmd)
}

func runFail(cmd *cobra.Command, args []string) error {
	reason, err := cmd.Flags().GetString("reason")
	if err != nil {
		return err
	}

	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	if err := engine.Fail(args[0], reason); err != nil {
		return err
	}

	fmt.Printf("Failed workflow %s: %s\n", args[0], reason)
	return nil
}
