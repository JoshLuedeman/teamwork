package cmd

import "github.com/spf13/cobra"

// ExitError is returned by commands that need a specific exit code.
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string { return e.Message }

var rootCmd = &cobra.Command{
	Use:   "teamwork",
	Short: "Teamwork — agent workflow orchestration",
	Long:  "Teamwork coordinates AI agent workflows: dispatching roles, tracking state, validating handoffs, and providing human oversight.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringP("dir", "d", ".", "Project root directory")
}
