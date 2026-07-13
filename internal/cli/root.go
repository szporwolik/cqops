package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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
var resetConfigFlag bool
var resetCacheFlag bool
var helpFlag bool

func init() {
	rootCmd.PersistentFlags().BoolVarP(&offlineFlag, "offline", "o", false, "Run in offline mode (skip all network checks)")
	rootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVarP(&versionFlag, "version", "v", false, "Print CQOps version and exit")
	rootCmd.PersistentFlags().BoolVar(&resetConfigFlag, "reset-config", false, "Reset all configuration, secrets, databases, cache, and lock file")
	rootCmd.PersistentFlags().BoolVar(&resetCacheFlag, "reset-cache", false, "Reset cached data only (solar, DXCC, REF, APRS)")
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CQOps version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("CQOps version %s\n", version.ResolvedFull())
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

	applog.Info("══════════ CQOps STARTED ══════════", "v", version.ResolvedFull(), "built", version.ResolvedDate(), "utc", time.Now().UTC().Format("2006-01-02 15:04:05"))
	if offlineFlag {
		applog.Info("Running in OFFLINE mode — all network connections skipped")
	}
	if debugFlag {
		applog.Info("Debug logging enabled")
	}

	// Only delegate to cobra when a subcommand is explicitly given.
	hasSubcommand := false
	for _, a := range os.Args[1:] {
		if a == "-h" || a == "--help" {
			helpFlag = true
		}
		if !strings.HasPrefix(a, "-") {
			hasSubcommand = true
		}
	}
	if versionFlag {
		fmt.Printf("CQOps version %s\n", version.ResolvedFull())
		return nil
	}
	if helpFlag {
		return rootCmd.Help()
	}
	if hasSubcommand {
		return rootCmd.Execute()
	}
	if resetConfigFlag {
		return resetConfig()
	}
	if resetCacheFlag {
		return resetCache()
	}
	return runTUI()
}

func runTUI() error {
	// The Linux console sets TERM=linux, which lacks proper colour and
	// key-sequence support.  Override to xterm-256color — the kernel's
	// console driver supports it natively and Bubble Tea needs it.
	if runtime.GOOS == "linux" && os.Getenv("TERM") == "linux" {
		os.Setenv("TERM", "xterm-256color")
	}

	a, err := app.Init()
	if err != nil {
		// Append a reset hint for configuration and database errors.
		errMsg := err.Error()
		if strings.Contains(errMsg, "config:") || strings.Contains(errMsg, "database:") || strings.Contains(errMsg, "logbook:") {
			return fmt.Errorf("%w\n\nRun 'cqops --reset-config' to reset all configuration, secrets, and databases, then start fresh.", err)
		}
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
			// Wizard may have changed the active logbook — reopen the
			// database so subsequent operations use the correct file.
			a.DB.Close()
			dbPath, err := config.DBPath(a.LogbookName, a.Logbook)
			if err != nil {
				return fmt.Errorf("db path after wizard: %w", err)
			}
			a.DB, err = store.InitDB(dbPath)
			if err != nil {
				return fmt.Errorf("reopen db after wizard: %w", err)
			}
			a.DBPath = dbPath
			applog.Info("Database reopened after wizard", "path", dbPath)
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

// resetConfig removes all CQOps configuration, secrets, databases, cache,
// and the lock file. It prompts for confirmation before deleting anything.
func resetConfig() error {
	configDir, err := config.ConfigDir()
	if err != nil {
		return fmt.Errorf("cannot determine config directory: %w", err)
	}
	dataDir, err := config.DataDir()
	if err != nil {
		return fmt.Errorf("cannot determine data directory: %w", err)
	}
	cacheDir, err := config.CacheDir()
	if err != nil {
		return fmt.Errorf("cannot determine cache directory: %w", err)
	}

	fmt.Println("CQOps Configuration Reset")
	fmt.Println("")
	fmt.Println("This will permanently delete:")
	fmt.Println("  - All configuration (config.yaml)")
	fmt.Println("  - All encrypted secrets (secrets.enc)")
	fmt.Println("  - All logbook databases (*.db)")
	fmt.Println("  - All cached data (solar, DXCC, REF, APRS)")
	fmt.Println("  - The instance lock file (cqops.lock)")
	fmt.Println("")
	fmt.Printf("Config dir: %s\n", configDir)
	fmt.Printf("Data dir:   %s\n", dataDir)
	fmt.Printf("Cache dir:  %s\n", cacheDir)
	fmt.Println("")

	if !promptYN("Proceed with reset?") {
		fmt.Println("Reset cancelled.")
		return nil
	}

	var deleted, failed int

	// Remove files in config dir: config.yaml, secrets.enc, cqops.lock
	for _, name := range []string{"config.yaml", "secrets.enc", "cqops.lock"} {
		p := filepath.Join(configDir, name)
		if err := os.Remove(p); err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", p, err)
				failed++
			}
		} else {
			fmt.Printf("Removed: %s\n", p)
			deleted++
		}
	}

	// Remove all .db files in data dir
	if entries, err := os.ReadDir(dataDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".db") {
				p := filepath.Join(dataDir, e.Name())
				if err := os.Remove(p); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", p, err)
					failed++
				} else {
					fmt.Printf("Removed: %s\n", p)
					deleted++
				}
			}
		}
	}

	// Remove all files in cache dir
	if entries, err := os.ReadDir(cacheDir); err == nil {
		for _, e := range entries {
			p := filepath.Join(cacheDir, e.Name())
			if err := os.RemoveAll(p); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", p, err)
				failed++
			} else {
				fmt.Printf("Removed: %s\n", p)
				deleted++
			}
		}
	}

	fmt.Printf("\nReset complete: %d file(s) removed", deleted)
	if failed > 0 {
		fmt.Printf(", %d warning(s)", failed)
	}
	fmt.Println(".\n\nRun cqops to start fresh with the setup wizard.")

	return nil
}

// resetCache removes cached data only (solar, DXCC, REF, APRS) and prompts
// for confirmation.
func resetCache() error {
	cacheDir, err := config.CacheDir()
	if err != nil {
		return fmt.Errorf("cannot determine cache directory: %w", err)
	}

	fmt.Println("CQOps Cache Reset")
	fmt.Println("")
	fmt.Println("This will delete all cached data (solar, DXCC, REF, APRS).")
	fmt.Println("Configuration, logbooks, and secrets are not affected.")
	fmt.Println("")
	fmt.Printf("Cache dir: %s\n", cacheDir)
	fmt.Println("")

	if !promptYN("Proceed with cache reset?") {
		fmt.Println("Reset cancelled.")
		return nil
	}

	var deleted, failed int

	if entries, err := os.ReadDir(cacheDir); err == nil {
		for _, e := range entries {
			p := filepath.Join(cacheDir, e.Name())
			if err := os.RemoveAll(p); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", p, err)
				failed++
			} else {
				fmt.Printf("Removed: %s\n", p)
				deleted++
			}
		}
	}

	fmt.Printf("\nCache reset complete: %d file(s) removed", deleted)
	if failed > 0 {
		fmt.Printf(", %d warning(s)", failed)
	}
	fmt.Println(".")

	return nil
}

func promptYN(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}
