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

		fmt.Printf("Provider: %s\n", a.Config.Rig.Provider)
		if a.Config.Rig.Provider == "" {
			fmt.Println("Status:   not configured")
			return nil
		}

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
