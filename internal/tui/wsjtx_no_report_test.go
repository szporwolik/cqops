package tui

import (
	"testing"

	"github.com/szporwolik/cqops/internal/store"
)

// TestWSJTXLogsAContactWithoutAReceivedReport reproduces the contact lost on
// 2026-10-31 at 18:04:15 UTC: WSJT-X delivered an FT8 record with
// <rst_rcvd:0> (the report never arrived), the validation demanded a received
// report and the contact was dropped — while WSJT-X's own log kept it. The
// record below is verbatim from wsjtx_log.adi.
//
// ADIF does not require RST (the suggested minimum is date, time, band/freq,
// call and mode) and LoTW matches a QSO on callsigns, start time, band and mode
// only, so an empty report neither invalidates the contact nor blocks a
// confirmation. It is stored empty and the operator is told.
func TestWSJTXLogsAContactWithoutAReceivedReport(t *testing.T) {
	m := newLifecycleTestModel(t)

	adif := "<call:6>IZ3XAN <gridsquare:6>JN55TC <mode:3>FT8 <rst_sent:3>-16 <rst_rcvd:0> " +
		"<qso_date:8>20261031 <time_on:6>180415 <qso_date_off:8>20261031 <time_off:6>180515 " +
		"<band:3>40m <freq:8>7.076153 <station_callsign:6>HF50HSP <my_gridsquare:6>KO00CA " +
		"<operator:6>SP9SPM <comment:25>FT8  Sent: -16  Rcvd:    <eor>"

	cmd, retry := m.logQSOFromADIF(adif)
	if retry {
		t.Fatal("a contact without a received report must not be retried forever")
	}
	if cmd != nil {
		_ = cmd() // the follow-up (refresh, enrichment) must not depend on the report
	}

	qsos, err := store.ListQSOs(m.App.DB, 10, "")
	if err != nil {
		t.Fatalf("ListQSOs: %v", err)
	}
	if len(qsos) != 1 {
		t.Fatalf("logged %d contacts, want 1 — the contact was dropped because its received report was empty", len(qsos))
	}
	got := qsos[0]
	if got.Call != "IZ3XAN" || got.Band != "40m" || got.Mode != "FT8" {
		t.Errorf("logged %s %s %s, want IZ3XAN 40m FT8", got.Call, got.Band, got.Mode)
	}
	if got.RSTRcvd != "" {
		t.Errorf("RSTRcvd = %q, want empty — the report never arrived and must not be invented", got.RSTRcvd)
	}
	if got.RSTSent != "-16" {
		t.Errorf("RSTSent = %q, want -16", got.RSTSent)
	}
	if !hasToast(m, "without a received report") {
		t.Errorf("toasts = %v, want the operator told the exchange was one-sided", toastMessages(m.toasts))
	}
}
