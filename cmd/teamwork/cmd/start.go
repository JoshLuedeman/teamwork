package cmd

import (
	"fmt"
	"strings"

	"github.com/joshluedeman/teamwork/internal/config"
	"github.com/joshluedeman/teamwork/internal/workflow"
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
	startCmd.Flags().Bool("dry-run", false, "Preview workflow steps without creating state files")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	wfType := args[0]
	goal := strings.Join(args[1:], " ")

	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	// Check built-in types first; if not found, check custom workflows in config.
	if !isKnownType(wfType) {
		cfg, cfgErr := config.Load(dir)
		if cfgErr != nil || !cfg.HasCustomWorkflow(wfType) {
			return fmt.Errorf("unknown workflow type %q — must be one of: %s (or define a custom workflow in config)",
				wfType, strings.Join(knownWorkflowTypes, ", "))
		}
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	if dryRun {
		return runDryRun(cmd, dir, wfType, goal)
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

	fmt.Fprintf(cmd.OutOrStdout(), "Started workflow %s\n", wf.ID)
	fmt.Fprintf(cmd.OutOrStdout(), "  Type:   %s\n", wf.Type)
	fmt.Fprintf(cmd.OutOrStdout(), "  Goal:   %s\n", goal)
	fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", wf.Status)
	if issue > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Issue:  #%d\n", issue)
	}

	return nil
}

// runDryRun previews the workflow steps without creating state files.
func runDryRun(cmd *cobra.Command, dir, wfType, goal string) error {
	cfg, err := config.Load(dir)
	if err != nil {
		return err
	}

	steps, err := workflow.PreviewStepsWithConfig(cfg, wfType)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "\nWorkflow: %s\n", wfType)
	fmt.Fprintf(out, "Goal: %s\n", goal)
	fmt.Fprintf(out, "Steps:\n")

	var skipped []string
	agentSteps := 0

	for _, s := range steps {
		if cfg.ShouldSkipStep(wfType, s.Role) {
			skipped = append(skipped, s.Role)
			fmt.Fprintf(out, "  %d. [%-17s] <- SKIPPED (configured in skip_steps)\n",
				s.Number, titleCase(s.Role))
			continue
		}

		tier := workflow.RoleTier(s.Role)
		if tier != "" {
			fmt.Fprintf(out, "  %d. [%-17s] %-40s (%s)\n",
				s.Number, titleCase(s.Role), s.Action, tier)
			agentSteps++
		} else {
			fmt.Fprintf(out, "  %d. [%-17s] %s\n",
				s.Number, titleCase(s.Role), s.Action)
		}
	}

	// Quality gates.
	var gates []string
	if cfg.QualityGates.HandoffComplete {
		gates = append(gates, "handoff_complete")
	}
	if cfg.QualityGates.TestsPass {
		gates = append(gates, "tests_pass")
	}
	if cfg.QualityGates.LintPass {
		gates = append(gates, "lint_pass")
	}
	if len(gates) > 0 {
		fmt.Fprintf(out, "\nQuality gates: %s\n", strings.Join(gates, ", "))
	}

	// Skipped steps.
	if len(skipped) > 0 {
		fmt.Fprintf(out, "Skipped steps: %s\n", strings.Join(skipped, ", "))
	} else {
		fmt.Fprintf(out, "Skipped steps: none\n")
	}

	fmt.Fprintf(out, "Total agent steps: %d\n", agentSteps)

	return nil
}

// titleCase converts a kebab-case role name to Title Case for display.
// e.g. "security-auditor" -> "Security Auditor"
func titleCase(role string) string {
	parts := strings.Split(role, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func isKnownType(t string) bool {
	for _, known := range knownWorkflowTypes {
		if t == known {
			return true
		}
	}
	return false
}
