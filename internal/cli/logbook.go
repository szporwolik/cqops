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
		for _, id := range config.SortedLogbookIDs(a.Config) {
			lb := a.Config.Logbooks[id]
			marker := " "
			if id == a.Config.State.ActiveLogbook {
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
			fmt.Printf("%s %-12s %s\n", marker, config.LogbookDisplayName(&lb), info)
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

		arg := args[0]
		id, lb, ok := resolveLogbookArg(a.Config, arg)
		if !ok {
			return fmt.Errorf("logbook %q not found", arg)
		}

		dbPath, _ := config.DBPath(id, lb)

		fmt.Printf("Name:        %s\n", config.LogbookDisplayName(lb))
		fmt.Printf("Description: %s\n", lb.Description)
		fmt.Printf("Database:    %s\n", dbPath)
		fmt.Printf("Callsign:    %s\n", lb.Station.Callsign)
		fmt.Printf("Operator:    %s\n", lb.Station.Operator)
		fmt.Printf("Grid:        %s\n", lb.Station.Grid)
		fmt.Printf("Rig:         %s\n", lb.Station.RigModel(a.Config.Rigs))
		fmt.Printf("Antenna:     %s\n", lb.Station.RigAntenna(a.Config.Rigs))
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

		arg := args[0]
		id, _, ok := resolveLogbookArg(a.Config, arg)
		if !ok {
			return fmt.Errorf("logbook %q does not exist", arg)
		}

		a.Config.State.ActiveLogbook = id
		if err := config.Save(a.ConfigPath, a.Config); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("Active logbook set to %q.\n", arg)
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
		if _, _, found := config.FindLogbookByCallsign(a.Config, name); found {
			return fmt.Errorf("logbook with callsign %q already exists", name)
		}

		lbID := config.NewID(name)
		rigID := config.NewID("default-rig")
		a.Config.Logbooks[lbID] = config.Logbook{
			ID:          lbID,
			Description: lbDescription,
			Station: config.Station{
				Callsign: lbCallsign,
				Operator: lbOperator,
				Grid:     lbGrid,
				RigName:  rigID,
			},
		}
		if lbRig != "" || lbAntenna != "" {
			a.Config.Rigs[rigID] = config.RigPreset{
				ID:      rigID,
				Model:   lbRig,
				Antenna: lbAntenna,
			}
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

		arg := args[0]
		id, lb, ok := resolveLogbookArg(a.Config, arg)
		if !ok {
			return fmt.Errorf("logbook %q not found", arg)
		}

		dbPath, _ := config.DBPath(id, lb)
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

// resolveLogbookArg tries to find a logbook by ID first, then by callsign.
func resolveLogbookArg(cfg *config.Config, arg string) (string, *config.Logbook, bool) {
	if lb, ok := cfg.Logbooks[arg]; ok {
		return arg, &lb, true
	}
	return config.FindLogbookByCallsign(cfg, arg)
}
