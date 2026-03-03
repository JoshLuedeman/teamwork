package cmd

import (
	"fmt"

	"github.com/JoshLuedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Show pending actions across workflows",
	RunE:  runNext,
}

func init() {
	rootCmd.AddCommand(nextCmd)
}

func runNext(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	actions, err := engine.Next()
	if err != nil {
		return err
	}

	if len(actions) == 0 {
		fmt.Println("All workflows are up to date.")
		return nil
	}

	for _, a := range actions {
		fmt.Printf("Workflow: %s\n", a.WorkflowID)
		if a.Repo != "" {
			fmt.Printf("  Repo: %s\n", a.Repo)
		}
		fmt.Printf("  Step %d: [%s] %s\n", a.Step, a.Role, a.Action)
		fmt.Println()
	}

	return nil
}
