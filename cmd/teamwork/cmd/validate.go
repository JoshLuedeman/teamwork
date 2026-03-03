package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/JoshLuedeman/teamwork/internal/validate"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate .teamwork/ directory structure and contents",
	RunE:  runValidate,
}

func init() {
	validateCmd.Flags().Bool("json", false, "Output results as JSON array")
	validateCmd.Flags().Bool("quiet", false, "Suppress passing checks")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	quiet, _ := cmd.Flags().GetBool("quiet")

	results, err := validate.Run(dir)
	if err != nil {
		// Exit code 2: cannot run validation
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	passed := 0
	failed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
	} else {
		for _, r := range results {
			if quiet && r.Passed {
				continue
			}
			if r.Passed {
				fmt.Printf("✓ %s: valid\n", r.Path)
			} else {
				fmt.Printf("✗ %s\n", r.Message)
			}
		}

		fmt.Printf("\n%d passed, %d failed\n", passed, failed)
	}

	if failed > 0 {
		os.Exit(1)
	}
	return nil
}
