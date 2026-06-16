package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/szporwolik/cqops/internal/app"
)

var rigCmd = &cobra.Command{
	Use:   "rig",
	Short: "Rig control commands",
}

var rigStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show rig connection status",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		fmt.Println("Rig control is configured per rig preset (C-End → Rig).")
		fmt.Println("Use cqops logbook show <name> to see station details.")

		fmt.Println("Status:   unavailable")
		fmt.Println("")
		fmt.Println("Tip: Configure rig provider in config.yaml")
		fmt.Println("     Supported: flrig, rigctld")

		return nil
	},
}

func registerRigCommands() {
	rootCmd.AddCommand(rigCmd)
	rigCmd.AddCommand(rigStatusCmd)
}
