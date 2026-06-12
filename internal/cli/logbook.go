package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
)

var logbookCmd = &cobra.Command{
	Use:   "logbook",
	Short: "Manage logbooks",
}

var logbookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured logbooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		fmt.Println("Logbooks:")
		for name, lb := range a.Config.Logbooks {
			marker := " "
			if name == a.Config.ActiveLogbook {
				marker = "*"
			}
			info := lb.Station.Callsign
			if lb.Station.Grid != "" {
				if info != "" {
					info += "  "
				}
				info += lb.Station.Grid
			}
			if info == "" {
				info = lb.Description
			}
			fmt.Printf("%s %-12s %s\n", marker, name, info)
		}
		return nil
	},
}

var logbookShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a specific logbook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		name := args[0]
		lb, ok := a.Config.Logbooks[name]
		if !ok {
			return fmt.Errorf("logbook %q not found", name)
		}

		dbPath, _ := config.DBPath(name, &lb)

		fmt.Printf("Name:        %s\n", name)
		fmt.Printf("Description: %s\n", lb.Description)
		fmt.Printf("Database:    %s\n", dbPath)
		fmt.Printf("Callsign:    %s\n", lb.Station.Callsign)
		fmt.Printf("Operator:    %s\n", lb.Station.Operator)
		fmt.Printf("Grid:        %s\n", lb.Station.Grid)
		fmt.Printf("Rig:         %s\n", lb.Station.Rig)
		fmt.Printf("Antenna:     %s\n", lb.Station.Antenna)
		fmt.Printf("ADIF export: %s\n", lb.ADIF.DefaultExportPath)
		return nil
	},
}

var logbookUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch active logbook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		name := args[0]
		if _, ok := a.Config.Logbooks[name]; !ok {
			return fmt.Errorf("logbook %q does not exist", name)
		}

		a.Config.ActiveLogbook = name
		if err := config.Save(a.ConfigPath, a.Config); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("Active logbook set to %q.\n", name)
		return nil
	},
}

var (
	lbDescription string
	lbCallsign    string
	lbOperator    string
	lbGrid        string
	lbRig         string
	lbAntenna     string
)

var logbookCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new logbook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		name := args[0]
		if _, ok := a.Config.Logbooks[name]; ok {
			return fmt.Errorf("logbook %q already exists", name)
		}

		a.Config.Logbooks[name] = config.Logbook{
			Description: lbDescription,
			Station: config.Station{
				Callsign: lbCallsign,
				Operator: lbOperator,
				Grid:     lbGrid,
				Rig:      lbRig,
				Antenna:  lbAntenna,
			},
		}

		if err := config.Save(a.ConfigPath, a.Config); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("Logbook %q created.\n", name)
		return nil
	},
}

var logbookPathCmd = &cobra.Command{
	Use:   "path <name>",
	Short: "Show the database path for a logbook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		name := args[0]
		lb, ok := a.Config.Logbooks[name]
		if !ok {
			return fmt.Errorf("logbook %q not found", name)
		}

		dbPath, _ := config.DBPath(name, &lb)
		fmt.Println(dbPath)
		return nil
	},
}

func registerLogbookCommands() {
	rootCmd.AddCommand(logbookCmd)
	logbookCmd.AddCommand(logbookListCmd)
	logbookCmd.AddCommand(logbookShowCmd)
	logbookCmd.AddCommand(logbookUseCmd)
	logbookCmd.AddCommand(logbookCreateCmd)
	logbookCmd.AddCommand(logbookPathCmd)

	logbookCreateCmd.Flags().StringVar(&lbDescription, "description", "", "Logbook description")
	logbookCreateCmd.Flags().StringVar(&lbCallsign, "callsign", "", "Station callsign")
	logbookCreateCmd.Flags().StringVar(&lbOperator, "operator", "", "Operator callsign")
	logbookCreateCmd.Flags().StringVar(&lbGrid, "grid", "", "Grid square / locator")
	logbookCreateCmd.Flags().StringVar(&lbRig, "rig", "", "Rig / transceiver")
	logbookCreateCmd.Flags().StringVar(&lbAntenna, "antenna", "", "Antenna")
}
