package cmd

import (
	"fmt"

	"github.com/joshluedeman/teamwork/internal/search"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search memory, handoffs, ADRs, and state for matching content",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

func init() {
	searchCmd.Flags().String("domain", "", "Filter memory results by domain tag")
	searchCmd.Flags().String("type", "", "Filter by artifact type: memory|handoff|adr|state")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	query := args[0]
	if len(args) > 1 {
		// Join all args as the query (supports multi-word queries without quotes).
		for _, a := range args[1:] {
			query += " " + a
		}
	}

	domain, err := cmd.Flags().GetString("domain")
	if err != nil {
		return err
	}

	artifactType, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}

	results, err := search.Query(dir, query, search.QueryOptions{
		Domain: domain,
		Type:   artifactType,
	})
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No results for '%s'\n", query)
		return nil
	}

	for _, r := range results {
		fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s (score: %d)\n  %s\n\n",
			r.Type, r.Path, r.Score, r.Snippet)
	}

	return nil
}
