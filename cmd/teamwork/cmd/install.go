package cmd

import (
"fmt"

"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
Use:        "install",
Short:      "Deprecated: use 'teamwork init' instead",
Long:       "Install is deprecated. Use 'teamwork init' which now fetches framework files and creates config in one step.",
Deprecated: "use 'teamwork init' instead — it now fetches framework files and creates config in one step.",
RunE:       runInstall,
}

func init() {
rootCmd.AddCommand(installCmd)
installCmd.Flags().String("source", "joshluedeman/teamwork", "Source repository (owner/repo)")
installCmd.Flags().String("ref", "main", "Git ref to install from (branch, tag, or SHA)")
installCmd.Flags().Bool("force", false, "Overwrite existing installation")
}

func runInstall(cmd *cobra.Command, args []string) error {
fmt.Println("Note: 'teamwork install' is deprecated. Please use 'teamwork init' instead.")

// Forward flags to initCmd if they were explicitly set.
source, _ := cmd.Flags().GetString("source")
ref, _ := cmd.Flags().GetString("ref")
force, _ := cmd.Flags().GetBool("force")

initCmd.Flags().Set("source", source)
initCmd.Flags().Set("ref", ref)
if force {
initCmd.Flags().Set("force", "true")
}
initCmd.Flags().Set("non-interactive", "true")

return runInit(initCmd, args)
}