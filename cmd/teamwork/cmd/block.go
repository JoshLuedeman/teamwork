package cmd

import (
	"fmt"

	"github.com/JoshLuedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var blockCmd = &cobra.Command{
	Use:   "block <workflow-id>",
	Short: "Block a workflow with a reason",
	Args:  cobra.ExactArgs(1),
	RunE:  runBlock,
}

func init() {
	blockCmd.Flags().StringP("reason", "r", "", "Reason for blocking (required)")
	_ = blockCmd.MarkFlagRequired("reason")
	rootCmd.AddCommand(blockCmd)
}

func runBlock(cmd *cobra.Command, args []string) error {
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

	if err := engine.Block(args[0], reason, "human"); err != nil {
		return err
	}

	fmt.Printf("Blocked workflow %s: %s\n", args[0], reason)
	return nil
}
