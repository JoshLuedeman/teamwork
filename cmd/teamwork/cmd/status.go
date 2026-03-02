package cmd

import (
	"fmt"

	"github.com/JoshLuedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active workflows",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	workflows, err := engine.Status()
	if err != nil {
		return err
	}

	if len(workflows) == 0 {
		fmt.Println("No active workflows.")
		return nil
	}

	fmt.Printf("%-36s  %-10s  %-12s  %-14s  %-14s  %s\n",
		"ID", "Type", "Status", "Current Step", "Current Role", "Updated")
	fmt.Println("------------------------------------  ----------  ------------  --------------  --------------  --------------------")

	for _, w := range workflows {
		fmt.Printf("%-36s  %-10s  %-12s  %-14d  %-14s  %s\n",
			w.ID, w.Type, w.Status, w.CurrentStep, w.CurrentRole, w.UpdatedAt)
	}

	return nil
}
