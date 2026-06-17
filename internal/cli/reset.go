package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"golang.org/x/term"
)

var resetForce bool

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Delete all data and reset to factory defaults",
	Long: `Reset removes all logbooks, QSO data, and configuration,
then recreates a fresh default config.

This action cannot be undone. Use --force to skip confirmation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !resetForce {
			tty := term.IsTerminal(int(os.Stdin.Fd()))

			if tty {
				fmt.Print("This will delete ALL logbooks, QSO data, and configuration.\nAre you sure? (y/N): ")
				os.Stdout.Sync()
			}

			answer := readLine(tty)

			if answer != "y" && answer != "Y" {
				if answer == "" && !tty {
					fmt.Println("Reset requires --force when no confirmation input is available.")
					return fmt.Errorf("use --force to skip confirmation")
				}
				fmt.Println("Reset cancelled.")
				return nil
			}
		}

		configDir, err := config.ConfigDir()
		if err != nil {
			return fmt.Errorf("config dir: %w", err)
		}

		dataDir, err := config.DataDir()
		if err != nil {
			return fmt.Errorf("data dir: %w", err)
		}

		logDir, err := config.LogDir()
		if err != nil {
			return fmt.Errorf("log dir: %w", err)
		}

		configPath, err := config.ConfigPath()
		if err != nil {
			return fmt.Errorf("config path: %w", err)
		}

		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			os.Remove(configPath)
			os.RemoveAll(logDir)
			if err := os.RemoveAll(dataDir); err == nil {
				lastErr = nil
				break
			} else {
				lastErr = err
				time.Sleep(200 * time.Millisecond)
			}
		}
		if lastErr != nil {
			return fmt.Errorf("remove data: %w", lastErr)
		}

		os.MkdirAll(configDir, 0755)
		os.MkdirAll(dataDir, 0755)
		os.MkdirAll(logDir, 0755)

		cfg := config.DefaultConfig()
		if err := config.Save(configPath, cfg); err != nil {
			return fmt.Errorf("save default config: %w", err)
		}

		fmt.Printf("Reset complete.\n")
		fmt.Printf("  Config:  %s\n", configPath)
		fmt.Printf("  Data:    %s\n", dataDir)
		fmt.Printf("  Logs:    %s\n", logDir)
		applog.Info("Settings reset to factory defaults")
		return nil
	},
}

func registerResetCommands() {
	rootCmd.AddCommand(resetCmd)
	resetCmd.Flags().BoolVarP(&resetForce, "force", "f", false, "Skip confirmation prompt")
}

func readLine(tty bool) string {
	if tty {
		return readLineTerminal()
	}
	return readLineRaw()
}

func readLineTerminal() string {
	rw := struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}
	t := term.NewTerminal(rw, "")
	line, err := t.ReadLine()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(line)
}

func readLineRaw() string {
	var buf [1]byte
	var result strings.Builder
	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil || n == 0 {
			break
		}
		if buf[0] == '\r' {
			continue
		}
		if buf[0] == '\n' {
			break
		}
		result.WriteByte(buf[0])
	}
	return strings.TrimSpace(result.String())
}
