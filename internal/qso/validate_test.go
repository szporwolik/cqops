package qso

import (
	"strings"
	"testing"
)

func TestValidateBandFromFreq(t *testing.T) {
	tests := []struct {
		name    string
		qso     QSO
		wantErr string
	}{
		{
			name: "valid band set manually",
			qso: QSO{
				Call:    "SP9MOA",
				Band:    "20m",
				Mode:    "SSB",
				RSTSent: "59",
				RSTRcvd: "59",
				QSODate: "20250615",
				TimeOn:  "120000",
			},
			wantErr: "",
		},
		{
			name: "band empty, valid freq derives band",
			qso: QSO{
				Call:    "SP9MOA",
				Band:    "",
				Freq:    14.074,
				Mode:    "SSB",
				RSTSent: "59",
				RSTRcvd: "59",
				QSODate: "20250615",
				TimeOn:  "120000",
			},
			wantErr: "",
		},
		{
			name: "band empty, invalid freq",
			qso: QSO{
				Call:    "SP9MOA",
				Band:    "",
				Freq:    27.555,
				Mode:    "SSB",
				RSTSent: "59",
				RSTRcvd: "59",
				QSODate: "20250615",
				TimeOn:  "120000",
			},
			wantErr: "frequency does not match any band",
		},
		{
			name: "band empty, freq zero",
			qso: QSO{
				Call:    "SP9MOA",
				Band:    "",
				Freq:    0,
				Mode:    "SSB",
				RSTSent: "59",
				RSTRcvd: "59",
				QSODate: "20250615",
				TimeOn:  "120000",
			},
			wantErr: "band is required",
		},
		{
			name: "invalid band name",
			qso: QSO{
				Call:    "SP9MOA",
				Band:    "CB",
				Freq:    27.185,
				Mode:    "SSB",
				RSTSent: "59",
				RSTRcvd: "59",
				QSODate: "20250615",
				TimeOn:  "120000",
			},
			wantErr: "unknown band",
		},
		{
			name: "band empty, freq 160m lower edge",
			qso: QSO{
				Call:    "SP9MOA",
				Band:    "",
				Freq:    1.9,
				Mode:    "SSB",
				RSTSent: "59",
				RSTRcvd: "59",
				QSODate: "20250615",
				TimeOn:  "120000",
			},
			wantErr: "",
		},
		{
			name: "band empty, freq 70cm",
			qso: QSO{
				Call:    "SP9MOA",
				Band:    "",
				Freq:    432.0,
				Mode:    "FM",
				RSTSent: "59",
				RSTRcvd: "59",
				QSODate: "20250615",
				TimeOn:  "120000",
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(&tt.qso)
			if tt.wantErr == "" {
				if len(errs) > 0 {
					t.Errorf("expected no error, got: %v", errs)
				}
			} else {
				found := false
				for _, e := range errs {
					if strings.Contains(e, tt.wantErr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, errs)
				}
			}
		})
	}
}

func TestValidateForSave(t *testing.T) {
	// Valid QSO.
	q := &QSO{
		Call:    "SP9MOA",
		Band:    "20m",
		Freq:    14.074,
		Mode:    "SSB",
		RSTSent: "59",
		RSTRcvd: "59",
		QSODate: "20250615",
		TimeOn:  "120000",
	}
	if err := ValidateForSave(q); err != nil {
		t.Errorf("valid QSO should pass: %v", err)
	}

	// Missing call.
	q2 := &QSO{
		Band:    "20m",
		Mode:    "SSB",
		RSTSent: "59",
		RSTRcvd: "59",
		QSODate: "20250615",
		TimeOn:  "120000",
	}
	if err := ValidateForSave(q2); err == nil {
		t.Error("QSO without call should fail")
	}

	// Missing mode.
	q3 := &QSO{
		Call:    "SP9MOA",
		Band:    "20m",
		RSTSent: "59",
		RSTRcvd: "59",
		QSODate: "20250615",
		TimeOn:  "120000",
	}
	if err := ValidateForSave(q3); err == nil {
		t.Error("QSO without mode should fail")
	}

	// Missing band and freq.
	q4 := &QSO{
		Call:    "SP9MOA",
		Mode:    "SSB",
		RSTSent: "59",
		RSTRcvd: "59",
		QSODate: "20250615",
		TimeOn:  "120000",
	}
	if err := ValidateForSave(q4); err == nil {
		t.Error("QSO without band and freq should fail")
	}

	// Empty RST.
	q5 := &QSO{
		Call:    "SP9MOA",
		Band:    "20m",
		Mode:    "SSB",
		RSTRcvd: "59",
		QSODate: "20250615",
		TimeOn:  "120000",
	}
	if err := ValidateForSave(q5); err == nil {
		t.Error("QSO without RST sent should fail")
	}
}

// TestValidateForImportKeepsAWeakSignalContactWithoutAReport pins the rule that
// an FT8/FT4 contact delivered by WSJT-X without a received report is still a
// contact. WSJT-X writes <rst_rcvd:0> whenever the report never arrived (the
// operator logged before it was decoded, or the partner closed with RR73), and
// ADIF does not require the field. The manual form keeps requiring it.
func TestValidateForImportKeepsAWeakSignalContactWithoutAReport(t *testing.T) {
	base := QSO{
		Call:    "IZ3XAN",
		Band:    "40m",
		Freq:    7.076153,
		RSTSent: "-16",
		QSODate: "20261031",
		TimeOn:  "180415",
	}

	cases := []struct {
		name        string
		mode        string
		submode     string
		wantImport  bool // ValidateForImport accepts the record
		wantManual  bool // Validate (the form) accepts it
		wantMessage string
	}{
		{name: "FT8 without a received report", mode: "FT8", wantImport: true, wantMessage: "rst_rcvd"},
		{name: "MFSK+FT4 without a received report", mode: "MFSK", submode: "FT4", wantImport: true, wantMessage: "rst_rcvd"},
		{name: "MFSK+FT2 without a received report", mode: "MFSK", submode: "FT2", wantImport: true, wantMessage: "rst_rcvd"},
		{name: "JT65 without a received report", mode: "JT65", wantImport: true, wantMessage: "rst_rcvd"},
		// A spoken report is typed in: an empty one is a mistake, not a lost frame.
		{name: "SSB without a received report", mode: "SSB", wantMessage: "rst_rcvd"},
		{name: "CW without a received report", mode: "CW", wantMessage: "rst_rcvd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := base
			q.Mode, q.Submode = tc.mode, tc.submode
			errs := ValidateForImport(&q)
			gotImport := len(errs) > 0
			if gotImport != !tc.wantImport {
				t.Errorf("ValidateForImport = %v, want accepted=%v", errs, tc.wantImport)
			}
			if !tc.wantImport && tc.wantMessage != "" && !strings.Contains(strings.Join(errs, "; "), tc.wantMessage) {
				t.Errorf("ValidateForImport = %v, want a %q error", errs, tc.wantMessage)
			}
			if manual := Validate(&q); len(manual) == 0 {
				t.Errorf("the manual form accepted %s without a received report, want it refused", tc.mode)
			}
		})
	}

	// A missing sent report stays an error on both paths: WSJT-X always
	// transmits one, so its absence means a broken record.
	q := base
	q.Mode = "FT8"
	q.RSTSent = ""
	if errs := ValidateForImport(&q); len(errs) == 0 {
		t.Error("ValidateForImport accepted a contact with no sent report")
	}
}
