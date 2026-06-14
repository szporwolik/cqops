package cli

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/tui"
	"github.com/szporwolik/cqops/internal/version"
)

var logbookFlag string

var rootCmd = &cobra.Command{
	Use:   "cqops",
	Short: "CQOps - Ham Radio Logger",
	Long: `CQOps is a cross-platform amateur radio logging tool.

Run without arguments to start the interactive TUI.
Run with commands for CLI-based logging and management.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func RegisterCommands() {
	rootCmd.PersistentFlags().StringVarP(&logbookFlag, "logbook", "l", "", "Logbook name to use")

	registerConfigCommands()
	registerLogbookCommands()
	registerLogCommands()
	registerRigCommands()
	registerResetCommands()
	registerVersionCommands()
}

func Execute() error {
	RegisterCommands()
	applog.Init()
	applog.Info("══════════ CQOPS STARTED ══════════", "v", version.Resolved())

	if len(os.Args) <= 1 {
		return runTUI()
	}
	return rootCmd.Execute()
}

func runTUI() error {
	a, err := app.Init(logbookFlag)
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	defer a.Close()

	if config.IsFirstRun(a.Config) {
		w := tui.NewWizard(a)
		p := tea.NewProgram(w)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("wizard: %w", err)
		}

		if err := config.Save(a.ConfigPath, a.Config); err != nil {
			return fmt.Errorf("save after wizard: %w", err)
		}
	}

	if config.IsFirstRun(a.Config) {
		fmt.Println("No logbook configured. Run cqops again to complete setup, or use cqops logbook create.")
		return nil
	}

	qsos, err := store.ListQSOs(a.DB, 500)
	if err != nil {
		applog.Warn("Failed to load initial QSO list", "error", err.Error())
	}

	m := tui.New(a, qsos)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}

	applog.Info("TUI exited, cleaning up")
	return nil
}
