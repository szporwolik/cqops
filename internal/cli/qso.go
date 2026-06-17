package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

var (
	logCall        string
	logBand        string
	logFreq        float64
	logMode        string
	logSubmode     string
	logRSTSent     string
	logRSTRcvd     string
	logGrid        string
	logName        string
	logQTH         string
	logComment     string
	logStationCall string
	logOperator    string
	logMyGrid      string
	logMyRig       string
	logMyAntenna   string
	logDate        string
	logTime        string
	logPower       string
	logNotes       string
	logLimit       int
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Manage QSO log entries",
}

var logAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new QSO to the log",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		qs := qso.NewQSO()
		qs.Call = strings.ToUpper(logCall)
		qs.Band = strings.ToUpper(logBand)
		qs.Freq = logFreq
		qs.Mode = strings.ToUpper(logMode)
		qs.Submode = strings.ToUpper(logSubmode)

		// Auto-fill RST if not provided (same logic as TUI)
		if logRSTSent != "" {
			qs.RSTSent = logRSTSent
		} else if qs.Mode == "CW" {
			qs.RSTSent = "599"
		} else if qs.Mode != "" {
			qs.RSTSent = "59"
		}
		if logRSTRcvd != "" {
			qs.RSTRcvd = logRSTRcvd
		} else if qs.RSTSent != "" {
			qs.RSTRcvd = qs.RSTSent
		}
		qs.GridSquare = strings.ToUpper(logGrid)
		qs.Name = logName
		qs.QTH = logQTH
		qs.Comment = logComment
		qs.Notes = logNotes
		if logPower != "" {
			qs.TXPower = logPower
		}

		if logDate != "" {
			qs.QSODate = logDate
		}
		if logTime != "" {
			qs.TimeOn = logTime
		}
		if logStationCall != "" {
			qs.StationCallsign = strings.ToUpper(logStationCall)
		}
		if logOperator != "" {
			qs.Operator = strings.ToUpper(logOperator)
		}
		if logMyGrid != "" {
			qs.MyGridSquare = strings.ToUpper(logMyGrid)
		}
		if logMyRig != "" {
			qs.MyRig = logMyRig
		}
		if logMyAntenna != "" {
			qs.MyAntenna = logMyAntenna
		}

		qso.ApplyStationDefaults(qs, qso.StationInfo{
			StationCallsign: a.Logbook.Station.Callsign,
			Operator:        a.Logbook.Station.Operator,
			MyGridSquare:    a.Logbook.Station.Grid,
			MyRig:           a.Logbook.Station.RigModel(a.Config.Rigs),
			MyAntenna:       a.Logbook.Station.RigAntenna(a.Config.Rigs),
			TXPower:         a.Logbook.Station.RigPower(a.Config.Rigs),
			MySOTARef:       a.Logbook.Station.SOTARef,
			MyPOTARef:       a.Logbook.Station.POTARef,
			MyWWFFRef:       a.Logbook.Station.WWFFRef,
		})

		if err := qso.ValidateForSave(qs); err != nil {
			return fmt.Errorf("validation: %w", err)
		}

		id, err := store.InsertQSO(a.DB, qs)
		if err != nil {
			return fmt.Errorf("save: %w", err)
		}

		bandStr := qs.Band
		if bandStr == "" && qs.Freq > 0 {
			bandStr = fmt.Sprintf("%.3f MHz", qs.Freq)
		}

		fmt.Printf("QSO saved [%s]: %s %s %s %s UTC (id: %d)\n",
			config.LogbookDisplayName(a.Logbook), qs.Call, bandStr, qs.Mode, qs.QSODate, id)

		// Upload to Wavelog if configured on this logbook
		wl := a.Logbook.Wavelog
		if wl != nil && wl.Enabled && wl.StationProfileID != "" {
			adifStr := qs.ToADIFWithStation(a.Logbook.Station.Callsign)
			result, wlErr := wavelog.PostQSOWithResult(
				wl.URL,
				wl.APIKey,
				wl.StationProfileID,
				adifStr,
			)
			if wlErr != nil {
				store.UpdateWavelogStatus(a.DB, id, "no")
				applog.Warn("Wavelog: CLI upload failed", "qso_id", id, "call", qs.Call, "error", wlErr)
				fmt.Printf("Wavelog upload failed: %v\n", wlErr)
			} else if result != nil && result.AllDuplicates {
				store.UpdateWavelogStatus(a.DB, id, "yes")
				fmt.Println("Wavelog: QSO already present (duplicate)")
			} else {
				store.UpdateWavelogStatus(a.DB, id, "yes")
				fmt.Println("Wavelog: QSO uploaded")
			}
		}

		return nil
	},
}

var logListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent QSOs from the active logbook",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		qsos, err := store.ListQSOs(a.DB, logLimit)
		if err != nil {
			return err
		}

		if len(qsos) == 0 {
			fmt.Printf("No QSOs in logbook [%s].\n", config.LogbookDisplayName(a.Logbook))
			return nil
		}

		fmt.Printf("Logbook: %s  (%d QSOs)\n", config.LogbookDisplayName(a.Logbook), len(qsos))
		fmt.Println(strings.Repeat("-", 90))

		for _, q := range qsos {
			band := q.Band
			if band == "" {
				band = fmt.Sprintf("%.3f", q.Freq)
			}
			fmt.Printf("%4d %s %s %-8s %-6s %-5s %-8s %s\n",
				q.ID, formatCLIDate(q.QSODate), q.TimeOn, q.Call, band, q.Mode, q.RSTSent, q.Comment)
		}

		return nil
	},
}

var logShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a specific QSO by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ID: %s", args[0])
		}

		q, err := store.GetQSOByID(a.DB, id)
		if err != nil {
			return fmt.Errorf("qso %d not found", id)
		}

		fmt.Printf("QSO #%d\n", q.ID)
		fmt.Printf("  Call:         %s\n", q.Call)
		fmt.Printf("  Date:         %s\n", q.QSODate)
		fmt.Printf("  Time:         %s\n", q.TimeOn)
		if q.TimeOff != "" {
			fmt.Printf("  Time Off:     %s\n", q.TimeOff)
		}
		fmt.Printf("  Band:         %s\n", q.Band)
		if q.Freq > 0 {
			fmt.Printf("  Frequency:    %.4f MHz\n", q.Freq)
		}
		fmt.Printf("  Mode:         %s\n", q.Mode)
		if q.Submode != "" {
			fmt.Printf("  Submode:      %s\n", q.Submode)
		}
		fmt.Printf("  RST Sent:     %s\n", q.RSTSent)
		fmt.Printf("  RST Rcvd:     %s\n", q.RSTRcvd)
		if q.GridSquare != "" {
			fmt.Printf("  Grid:         %s\n", q.GridSquare)
		}
		if q.Name != "" {
			fmt.Printf("  Name:         %s\n", q.Name)
		}
		if q.QTH != "" {
			fmt.Printf("  QTH:          %s\n", q.QTH)
		}
		if q.Comment != "" {
			fmt.Printf("  Comment:      %s\n", q.Comment)
		}
		if q.Notes != "" {
			fmt.Printf("  Notes:        %s\n", q.Notes)
		}
		if q.TXPower != "" {
			fmt.Printf("  TX Power:     %s W\n", q.TXPower)
		}
		if q.Distance > 0 {
			fmt.Printf("  Distance:     %.0f km\n", q.Distance)
		}
		if q.Bearing > 0 {
			fmt.Printf("  Bearing:      %.0f°\n", q.Bearing)
		}
		fmt.Printf("  My Callsign:  %s\n", q.StationCallsign)
		if q.Operator != "" {
			fmt.Printf("  Operator:     %s\n", q.Operator)
		}
		if q.MyGridSquare != "" {
			fmt.Printf("  My Grid:      %s\n", q.MyGridSquare)
		}
		if q.MyRig != "" {
			fmt.Printf("  My Rig:       %s\n", q.MyRig)
		}
		if q.MyAntenna != "" {
			fmt.Printf("  My Antenna:   %s\n", q.MyAntenna)
		}
		fmt.Printf("  Source:       %s\n", q.Source)
		fmt.Printf("  Created:      %s\n", q.CreatedAt.Format("2006-01-02 15:04:05"))
		return nil
	},
}

var logDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a QSO by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Init(logbookFlag)
		if err != nil {
			return err
		}
		defer a.Close()

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ID: %s", args[0])
		}

		if err := store.DeleteQSO(a.DB, id); err != nil {
			return err
		}

		fmt.Printf("QSO %d deleted from [%s].\n", id, config.LogbookDisplayName(a.Logbook))
		return nil
	},
}

func registerLogCommands() {
	rootCmd.AddCommand(logCmd)
	logCmd.AddCommand(logAddCmd)
	logCmd.AddCommand(logListCmd)
	logCmd.AddCommand(logShowCmd)
	logCmd.AddCommand(logDeleteCmd)

	logAddCmd.Flags().StringVar(&logCall, "call", "", "Station callsign (required)")
	logAddCmd.Flags().StringVar(&logBand, "band", "", "Band (e.g., 20m)")
	logAddCmd.Flags().Float64Var(&logFreq, "freq", 0, "Frequency in MHz")
	logAddCmd.Flags().StringVar(&logMode, "mode", "", "Mode (e.g., FT8, SSB)")
	logAddCmd.Flags().StringVar(&logSubmode, "submode", "", "Submode")
	logAddCmd.Flags().StringVar(&logRSTSent, "rst-sent", "", "RST sent")
	logAddCmd.Flags().StringVar(&logRSTRcvd, "rst-rcvd", "", "RST received")
	logAddCmd.Flags().StringVar(&logGrid, "grid", "", "Gridsquare")
	logAddCmd.Flags().StringVar(&logName, "name", "", "Operator name")
	logAddCmd.Flags().StringVar(&logQTH, "qth", "", "QTH / location")
	logAddCmd.Flags().StringVar(&logComment, "comment", "", "Comment")
	logAddCmd.Flags().StringVar(&logStationCall, "station-callsign", "", "Station callsign")
	logAddCmd.Flags().StringVar(&logOperator, "operator", "", "Operator")
	logAddCmd.Flags().StringVar(&logMyGrid, "my-grid", "", "My gridsquare")
	logAddCmd.Flags().StringVar(&logMyRig, "my-rig", "", "My rig")
	logAddCmd.Flags().StringVar(&logMyAntenna, "my-antenna", "", "My antenna")
	logAddCmd.Flags().StringVar(&logDate, "date", "", "QSO date YYYYMMDD (default: today UTC)")
	logAddCmd.Flags().StringVar(&logTime, "time", "", "QSO time HHMMSS (default: now UTC)")
	logAddCmd.Flags().StringVar(&logPower, "power", "", "TX power in watts")
	logAddCmd.Flags().StringVar(&logNotes, "notes", "", "Private notes")

	logAddCmd.MarkFlagRequired("call")

	logListCmd.Flags().IntVarP(&logLimit, "limit", "n", 50, "Number of QSOs to show")
}

func formatCLIDate(adif string) string {
	if len(adif) < 8 {
		return "--------"
	}
	return adif[0:4] + "-" + adif[4:6] + "-" + adif[6:8]
}
