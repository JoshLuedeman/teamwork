package cmd

import (
	"github.com/JoshLuedeman/teamwork/internal/tui"
	"github.com/JoshLuedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Open interactive workflow dashboard",
	RunE:  runDashboard,
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}

func runDashboard(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	return tui.Run(engine)
}
