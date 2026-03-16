package cmd

import (
	"fmt"
	"sort"

	"github.com/joshluedeman/teamwork/internal/metrics"
	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "View workflow metrics and reports",
}

var metricsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show per-workflow metric summaries",
	RunE:  runMetricsSummary,
}

var metricsRolesCmd = &cobra.Command{
	Use:   "roles",
	Short: "Show per-role aggregate statistics",
	RunE:  runMetricsRoles,
}

func init() {
	metricsCmd.AddCommand(metricsSummaryCmd)
	metricsCmd.AddCommand(metricsRolesCmd)
	rootCmd.AddCommand(metricsCmd)
}

func runMetricsSummary(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	summaries, err := metrics.SummarizeAll(dir)
	if err != nil {
		return fmt.Errorf("loading metrics: %w", err)
	}

	if len(summaries) == 0 {
		fmt.Println("No metrics data found.")
		return nil
	}

	fmt.Printf("%-36s  %5s  %8s  %5s  %5s  %7s  %s\n",
		"Workflow", "Steps", "Duration", "Fails", "Escal", "Defects", "Cost")
	fmt.Println("------------------------------------  -----  --------  -----  -----  -------  ----------")

	for _, s := range summaries {
		cost := "-"
		if s.TotalCost != "" {
			cost = s.TotalCost
		}
		duration := formatDuration(s.TotalDuration)
		fmt.Printf("%-36s  %5d  %8s  %5d  %5d  %7d  %s\n",
			truncate(s.WorkflowID, 36), s.StepCount, duration, s.FailureCount, s.EscalationCount, s.DefectCount, cost)
	}

	return nil
}

func runMetricsRoles(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	summaries, err := metrics.SummarizeAll(dir)
	if err != nil {
		return fmt.Errorf("loading metrics: %w", err)
	}

	if len(summaries) == 0 {
		fmt.Println("No metrics data found.")
		return nil
	}

	// Aggregate across all workflows.
	roleDuration := make(map[string]int)
	roleWorkflows := make(map[string]int)
	for _, s := range summaries {
		for role, dur := range s.RoleDurations {
			roleDuration[role] += dur
			roleWorkflows[role]++
		}
	}

	// Sort roles for consistent output.
	var roles []string
	for r := range roleDuration {
		roles = append(roles, r)
	}
	sort.Strings(roles)

	fmt.Printf("%-18s  %8s  %10s  %s\n", "Role", "Duration", "Workflows", "Avg Duration")
	fmt.Println("------------------  --------  ----------  ------------")

	for _, role := range roles {
		dur := roleDuration[role]
		wf := roleWorkflows[role]
		avg := 0
		if wf > 0 {
			avg = dur / wf
		}
		fmt.Printf("%-18s  %8s  %10d  %s\n", role, formatDuration(dur), wf, formatDuration(avg))
	}

	return nil
}

func formatDuration(seconds int) string {
	if seconds == 0 {
		return "-"
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm%ds", seconds/60, seconds%60)
	}
	return fmt.Sprintf("%dh%dm", seconds/3600, (seconds%3600)/60)
}
