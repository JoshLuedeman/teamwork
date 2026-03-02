package cmd

import (
	"fmt"
	"strings"

	"github.com/JoshLuedeman/teamwork/internal/workflow"
	"github.com/spf13/cobra"
)

var knownWorkflowTypes = []string{
	"feature", "bugfix", "refactor", "hotfix",
	"security-response", "dependency-update",
	"documentation", "spike", "release", "rollback",
}

var startCmd = &cobra.Command{
	Use:   "start <type> <goal>",
	Short: "Start a new workflow",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runStart,
}

func init() {
	startCmd.Flags().IntP("issue", "i", 0, "GitHub issue number to link")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	wfType := args[0]
	goal := strings.Join(args[1:], " ")

	if !isKnownType(wfType) {
		return fmt.Errorf("unknown workflow type %q — must be one of: %s",
			wfType, strings.Join(knownWorkflowTypes, ", "))
	}

	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	issue, err := cmd.Flags().GetInt("issue")
	if err != nil {
		return err
	}

	engine, err := workflow.NewEngine(dir)
	if err != nil {
		return err
	}

	wf, err := engine.Start(wfType, goal, issue)
	if err != nil {
		return err
	}

	fmt.Printf("Started workflow %s\n", wf.ID)
	fmt.Printf("  Type:   %s\n", wf.Type)
	fmt.Printf("  Goal:   %s\n", goal)
	fmt.Printf("  Status: %s\n", wf.Status)
	if issue > 0 {
		fmt.Printf("  Issue:  #%d\n", issue)
	}

	return nil
}

func isKnownType(t string) bool {
	for _, known := range knownWorkflowTypes {
		if t == known {
			return true
		}
	}
	return false
}
