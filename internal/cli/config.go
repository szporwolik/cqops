package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/szporwolik/cqops/internal/app"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CQOps configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		fmt.Printf("Config path:    %s\n", a.ConfigPath)
		fmt.Printf("Active logbook: %s\n", a.Config.State.ActiveLogbook)
		fmt.Println()

		fmt.Println("Logbooks:")
		for name, lb := range a.Config.Logbooks {
			marker := " "
			if name == a.Config.State.ActiveLogbook {
				marker = "*"
			}
			fmt.Printf("  %s %-12s %s\n", marker, name, lb.Description)
		}

		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		fmt.Println(a.ConfigPath)
		return nil
	},
}

func registerConfigCommands() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
}
