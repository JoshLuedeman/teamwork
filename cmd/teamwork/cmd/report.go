package cmd

import (
	"fmt"
	"os"

	"github.com/joshluedeman/teamwork/internal/report"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report <workflow-id>",
	Short: "Generate an exportable workflow report",
	Long: `Generate a consolidated report of a workflow including steps, handoffs, gate
results, and cost estimates.

Output format is controlled by --format:
  md   — Markdown document (default; suitable for PR comments)
  json — JSON object with workflow_id and all report fields
  html — Self-contained HTML document with inline CSS`,
	Args: cobra.ExactArgs(1),
	RunE: runReport,
}

func init() {
	reportCmd.Flags().String("format", "md", "Output format: md, json, or html")
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	workflowID := args[0]

	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}

	r, err := report.Build(dir, workflowID)
	if err != nil {
		return fmt.Errorf("building report: %w", err)
	}

	switch format {
	case "md", "markdown":
		fmt.Print(report.RenderMarkdown(r))
	case "json":
		data, err := report.RenderJSON(r)
		if err != nil {
			return fmt.Errorf("rendering JSON: %w", err)
		}
		fmt.Printf("%s\n", data)
	case "html":
		fmt.Print(report.RenderHTML(r))
	default:
		fmt.Fprintf(os.Stderr, "unknown format %q — expected md, json, or html\n", format)
		os.Exit(1)
	}

	return nil
}
