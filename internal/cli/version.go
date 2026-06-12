package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/szporwolik/cqops/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CQOps version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("CQOps version %s\n", version.Resolved())
	},
}

func registerVersionCommands() {
	rootCmd.AddCommand(versionCmd)
}
