package cmd

import (
	"fmt"

	"github.com/joshluedeman/teamwork/internal/gates"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Run secrets scan on the project",
	Long:  "Scan the project directory for secrets using gitleaks, detect-secrets, or trufflehog.\nExits 0 if clean, 1 if secrets are found.",
	RunE:  runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "Running secrets scan...")

	found, details, err := gates.RunSecretsGate(dir, gates.ShellRunner{})
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	if found {
		fmt.Fprintln(w, "⚠️  Secrets found:")
		if details != "" {
			fmt.Fprintln(w, details)
		}
		return &ExitError{Code: 1, Message: ""}
	}

	if details != "" {
		fmt.Fprintln(w, details)
	}
	fmt.Fprintln(w, "✅ No secrets found.")
	return nil
}
