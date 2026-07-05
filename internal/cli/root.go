package cli

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/tui"
	"github.com/szporwolik/cqops/internal/version"
)

var offlineFlag bool
var debugFlag bool
var versionFlag bool

func init() {
	rootCmd.PersistentFlags().BoolVarP(&offlineFlag, "offline", "o", false, "Run in offline mode (skip all network checks)")
	rootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVarP(&versionFlag, "version", "v", false, "Print CQOps version and exit")
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CQOps version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("CQOps version %s\n", version.Resolved())
	},
}

var rootCmd = &cobra.Command{
	Use:   "cqops",
	Short: "CQOps - Ham Radio Logger (TUI)",
	Long: `CQOps is a cross-platform amateur radio logging tool.

Run without arguments to start the interactive TUI.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() error {
	// Always let cobra parse persistent flags (--offline, --logbook, --debug)
	// even when launching the TUI without a subcommand. Must happen before
	// applog.Init so debug mode can be set early.
	if err := rootCmd.ParseFlags(os.Args[1:]); err != nil {
		// Ignore "unknown flag" errors — cobra will handle validation.
	}

	applog.SetDebugMode(debugFlag)
	applog.Init()

	applog.Info("══════════ CQOps STARTED ══════════", "v", version.Resolved(), "built", version.ResolvedDate(), "utc", time.Now().UTC().Format("2006-01-02 15:04:05"))
	if offlineFlag {
		applog.Info("Running in OFFLINE mode — all network connections skipped")
	}
	if debugFlag {
		applog.Info("Debug logging enabled")
	}

	// Only delegate to cobra when a subcommand is explicitly given.
	hasSubcommand := false
	for _, a := range os.Args[1:] {
		if !strings.HasPrefix(a, "-") {
			hasSubcommand = true
			break
		}
	}
	if versionFlag {
		fmt.Printf("CQOps version %s\n", version.Resolved())
		return nil
	}
	if hasSubcommand {
		return rootCmd.Execute()
	}
	return runTUI()
}

func runTUI() error {
	// The Linux console (TERM=linux) sends incompatible function-key
	// escape sequences that Bubble Tea cannot parse.  tmux provides a
	// proper xterm-compatible terminal layer — refuse to start without it.
	if runtime.GOOS == "linux" && os.Getenv("TERM") == "linux" {
		fmt.Fprintln(os.Stderr, "CQOps: The Linux console (TERM=linux) does not support function keys.")
		fmt.Fprintln(os.Stderr, "CQOps: Please run CQOps inside tmux:")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "    tmux")
		fmt.Fprintln(os.Stderr, "    cqops")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "CQOps: If tmux is not installed:  sudo apt install tmux")
		return nil
	}

	a, err := app.Init()
	if err != nil {
		return err
	}
	defer a.Close()

	if config.IsFirstRun(a.Config) {
		w := tui.NewWizard(a)
		w.Offline = offlineFlag
		p := tea.NewProgram(w)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("wizard: %w", err)
		}

		if w.Completed {
			if err := config.Save(a.ConfigPath, a.Config); err != nil {
				return fmt.Errorf("save after wizard: %w", err)
			}
		} else {
			applog.Info("Wizard not completed — config not saved")
			return nil
		}
	}

	if config.IsFirstRun(a.Config) {
		fmt.Println("No logbook configured. Run cqops again to complete setup.")
		return nil
	}

	qsos, err := store.ListQSOs(a.DB, 500, a.Logbook.ActiveContest)
	if err != nil {
		applog.Warn("Failed to load initial QSO list", "error", err.Error())
	}

	m := tui.New(a, qsos)
	m.Offline = offlineFlag
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}

	applog.Info("TUI exited, cleaning up")
	return nil
}
