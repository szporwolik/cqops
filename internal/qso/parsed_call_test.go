package qso

import (
	"testing"
)

func TestParseCallsign_Plain(t *testing.T) {
	pc := ParseCallsign("K1ABC")
	if pc.BaseCall != "K1ABC" {
		t.Errorf("BaseCall = %q, want K1ABC", pc.BaseCall)
	}
	if pc.OperatingPrefix != "" {
		t.Errorf("unexpected prefix: %q", pc.OperatingPrefix)
	}
}

func TestParseCallsign_Portable(t *testing.T) {
	pc := ParseCallsign("SP9SPM/P")
	if pc.BaseCall != "SP9SPM" {
		t.Errorf("BaseCall = %q, want SP9SPM", pc.BaseCall)
	}
	if !pc.IsPortable {
		t.Error("expected IsPortable")
	}
	if len(pc.OperatingSuffixes) != 1 || pc.OperatingSuffixes[0] != "P" {
		t.Errorf("suffixes = %v, want [P]", pc.OperatingSuffixes)
	}
}

func TestParseCallsign_ForeignPrefix(t *testing.T) {
	pc := ParseCallsign("9A/SP9SPM/P")
	if pc.BaseCall != "SP9SPM" {
		t.Errorf("BaseCall = %q, want SP9SPM", pc.BaseCall)
	}
	if pc.OperatingPrefix != "9A" {
		t.Errorf("OperatingPrefix = %q, want 9A", pc.OperatingPrefix)
	}
	if !pc.IsPortable {
		t.Error("expected IsPortable")
	}
}

func TestParseCallsign_ForeignPrefixOnly(t *testing.T) {
	pc := ParseCallsign("9A/SP9SPM")
	if pc.BaseCall != "SP9SPM" {
		t.Errorf("BaseCall = %q, want SP9SPM", pc.BaseCall)
	}
	if pc.OperatingPrefix != "9A" {
		t.Errorf("OperatingPrefix = %q, want 9A", pc.OperatingPrefix)
	}
	if pc.IsPortable {
		t.Error("unexpected IsPortable")
	}
}

func TestParseCallsign_CanaryIslands(t *testing.T) {
	pc := ParseCallsign("EA8/SP9SPM/P")
	if pc.BaseCall != "SP9SPM" {
		t.Errorf("BaseCall = %q, want SP9SPM", pc.BaseCall)
	}
	if pc.OperatingPrefix != "EA8" {
		t.Errorf("OperatingPrefix = %q, want EA8", pc.OperatingPrefix)
	}
	if !pc.IsPortable {
		t.Error("expected IsPortable")
	}
}

func TestParseCallsign_Germany(t *testing.T) {
	pc := ParseCallsign("DL/SP9SPM")
	if pc.BaseCall != "SP9SPM" {
		t.Errorf("BaseCall = %q, want SP9SPM", pc.BaseCall)
	}
	if pc.OperatingPrefix != "DL" {
		t.Errorf("OperatingPrefix = %q, want DL", pc.OperatingPrefix)
	}
}

func TestParseCallsign_France(t *testing.T) {
	pc := ParseCallsign("F/SP9SPM/P")
	if pc.BaseCall != "SP9SPM" {
		t.Errorf("BaseCall = %q, want SP9SPM", pc.BaseCall)
	}
	if pc.OperatingPrefix != "F" {
		t.Errorf("OperatingPrefix = %q, want F", pc.OperatingPrefix)
	}
	if !pc.IsPortable {
		t.Error("expected IsPortable")
	}
}

func TestParseCallsign_Mobile(t *testing.T) {
	pc := ParseCallsign("SP9SPM/M")
	if pc.BaseCall != "SP9SPM" {
		t.Errorf("BaseCall = %q, want SP9SPM", pc.BaseCall)
	}
	if !pc.IsMobile {
		t.Error("expected IsMobile")
	}
}

func TestParseCallsign_MaritimeMobile(t *testing.T) {
	pc := ParseCallsign("SP9SPM/MM")
	if pc.BaseCall != "SP9SPM" {
		t.Errorf("BaseCall = %q, want SP9SPM", pc.BaseCall)
	}
	if !pc.IsMaritimeMobile {
		t.Error("expected IsMaritimeMobile")
	}
}

func TestParseCallsign_QRP(t *testing.T) {
	pc := ParseCallsign("SP9SPM/QRP")
	if pc.BaseCall != "SP9SPM" {
		t.Errorf("BaseCall = %q, want SP9SPM", pc.BaseCall)
	}
	if !pc.IsQRP {
		t.Error("expected IsQRP")
	}
}

func TestParseCallsign_Hawaii(t *testing.T) {
	pc := ParseCallsign("KH6/K1ABC/P")
	if pc.BaseCall != "K1ABC" {
		t.Errorf("BaseCall = %q, want K1ABC", pc.BaseCall)
	}
	if pc.OperatingPrefix != "KH6" {
		t.Errorf("OperatingPrefix = %q, want KH6", pc.OperatingPrefix)
	}
	if !pc.IsPortable {
		t.Error("expected IsPortable")
	}
}

func TestParseCallsign_Lowercase(t *testing.T) {
	pc := ParseCallsign("9a/sp9spm/p")
	if pc.BaseCall != "SP9SPM" {
		t.Errorf("BaseCall = %q, want SP9SPM", pc.BaseCall)
	}
	if pc.OperatingPrefix != "9A" {
		t.Errorf("OperatingPrefix = %q, want 9A", pc.OperatingPrefix)
	}
}

func TestParseCallsign_Empty(t *testing.T) {
	pc := ParseCallsign("")
	if pc.Canonical != "" {
		t.Error("expected empty canonical")
	}
	if pc.BaseCall != "" {
		t.Error("expected empty base call")
	}
}

func TestParseCallsign_Whitespace(t *testing.T) {
	pc := ParseCallsign("  SP9SPM  ")
	if pc.BaseCall != "SP9SPM" {
		t.Errorf("BaseCall = %q, want SP9SPM", pc.BaseCall)
	}
}

func TestParseCallsign_HasForeignPrefix(t *testing.T) {
	pc := ParseCallsign("9A/SP9SPM/P")
	if !pc.HasForeignPrefix() {
		t.Error("expected HasForeignPrefix to be true for foreign-prefix callsign")
	}
}

func TestParseCallsign_KI6NAZ_P(t *testing.T) {
	pc := ParseCallsign("KI6NAZ/P")
	if pc.BaseCall != "KI6NAZ" {
		t.Errorf("BaseCall = %q, want KI6NAZ", pc.BaseCall)
	}
	if !pc.IsPortable {
		t.Error("expected IsPortable")
	}
}
