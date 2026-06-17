package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
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

		activeDisplay := a.Config.State.ActiveLogbook
		if lb, ok := a.Config.Logbooks[activeDisplay]; ok {
			activeDisplay = config.LogbookDisplayName(&lb)
		}

		fmt.Printf("Config path:    %s\n", a.ConfigPath)
		fmt.Printf("Active logbook: %s\n", activeDisplay)
		fmt.Println()

		fmt.Println("Logbooks:")
		for _, id := range config.SortedLogbookIDs(a.Config) {
			lb := a.Config.Logbooks[id]
			marker := " "
			if id == a.Config.State.ActiveLogbook {
				marker = "*"
			}
			fmt.Printf("  %s %-12s %s\n", marker, config.LogbookDisplayName(&lb), lb.Description)
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
