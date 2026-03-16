package cmd

import (
	"fmt"

	"github.com/joshluedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var retryCmd = &cobra.Command{
	Use:   "retry <workflow-id>",
	Short: "Retry the current failed or blocked step of a workflow",
	Args:  cobra.ExactArgs(1),
	RunE:  runRetry,
}

func init() {
	rootCmd.AddCommand(retryCmd)
}

func runRetry(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	if err := engine.Retry(args[0]); err != nil {
		return err
	}

	fmt.Printf("Retrying workflow %s.\n", args[0])
	return nil
}
