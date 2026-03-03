package cmd

import (
	"fmt"

	"github.com/JoshLuedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel <workflow-id>",
	Short: "Cancel an active or blocked workflow",
	Args:  cobra.ExactArgs(1),
	RunE:  runCancel,
}

func init() {
	cancelCmd.Flags().StringP("reason", "r", "", "Reason for cancellation")
	rootCmd.AddCommand(cancelCmd)
}

func runCancel(cmd *cobra.Command, args []string) error {
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

	if err := engine.Cancel(args[0], reason); err != nil {
		return err
	}

	if reason != "" {
		fmt.Printf("Cancelled workflow %s: %s\n", args[0], reason)
	} else {
		fmt.Printf("Cancelled workflow %s.\n", args[0])
	}
	return nil
}
