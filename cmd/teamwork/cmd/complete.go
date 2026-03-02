package cmd

import (
	"fmt"

	"github.com/JoshLuedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var completeCmd = &cobra.Command{
	Use:   "complete <workflow-id>",
	Short: "Mark a workflow as complete",
	Args:  cobra.ExactArgs(1),
	RunE:  runComplete,
}

func init() {
	rootCmd.AddCommand(completeCmd)
}

func runComplete(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	if err := engine.Complete(args[0]); err != nil {
		return err
	}

	fmt.Printf("Completed workflow %s.\n", args[0])
	return nil
}
