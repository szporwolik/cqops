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
