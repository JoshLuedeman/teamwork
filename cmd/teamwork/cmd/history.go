package cmd

import (
	"fmt"

	"github.com/JoshLuedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history <workflow-id>",
	Short: "Show step-by-step history of a workflow",
	Args:  cobra.ExactArgs(1),
	RunE:  runHistory,
}

func init() {
	rootCmd.AddCommand(historyCmd)
}

func runHistory(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	ws, artifacts, err := engine.History(args[0])
	if err != nil {
		return err
	}

	if len(ws.Steps) == 0 {
		fmt.Println("No history for this workflow.")
		return nil
	}

	fmt.Printf("History for workflow %s:\n\n", args[0])
	for _, s := range ws.Steps {
		fmt.Printf("  Step %d — %s [%s]\n", s.Step, s.Action, s.Role)
		if s.Handoff != "" {
			fmt.Printf("    Handoff: %s\n", s.Handoff)
		}
		fmt.Printf("    %s\n\n", s.Started)
	}

	if len(artifacts) > 0 {
		fmt.Printf("Handoff artifacts: %d\n", len(artifacts))
		for _, a := range artifacts {
			fmt.Printf("  Step %d — %s → %s\n", a.Step, a.Role, a.NextRole)
		}
	}

	return nil
}
