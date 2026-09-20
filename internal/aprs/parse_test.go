package aprs

import (
	"testing"
	"time"
)

func TestParsePacketTimestamp(t *testing.T) {
	now := time.Date(2026, 8, 9, 14, 30, 0, 0, time.UTC)

	cases := []struct {
		name string
		body string
		want time.Time
	}{
		{
			name: "valid timestamp",
			body: "@091425z/5023.45N/02012.34E>",
			want: time.Date(2026, 8, 9, 14, 25, 0, 0, time.UTC),
		},
		{
			name: "day rollover to previous month",
			body: "@312300z/5023.45N/02012.34E>",
			want: time.Date(2026, 7, 31, 23, 0, 0, 0, time.UTC),
		},
		{
			name: "timestamp just in the future within tolerance",
			body: "@091431z/5023.45N/02012.34E>",
			want: time.Date(2026, 8, 9, 14, 31, 0, 0, time.UTC),
		},
		{
			name: "clock skew one hour ahead clamps to arrival",
			body: "@091530z/5023.45N/02012.34E>",
			want: now,
		},
		{
			name: "clock skew two hours ahead clamps to arrival",
			body: "@091630z/5023.45N/02012.34E>",
			want: now,
		},
		{
			name: "not a timestamped packet",
			body: "!5023.45N/02012.34E>",
			want: time.Time{},
		},
		{
			name: "local time suffix not supported",
			body: "@091425/5023.45N/02012.34E>",
			want: time.Time{},
		},
		{
			name: "unknown time all zeros",
			body: "@000000z5023.45N/02012.34E>",
			want: time.Time{},
		},
		{
			name: "invalid hour",
			body: "@092530z5023.45N/02012.34E>",
			want: time.Time{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parsePacketTimestamp(tc.body, now)
			if !got.Equal(tc.want) {
				t.Errorf("parsePacketTimestamp(%q) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

func TestParsePositionPacket_SetsLastHeardFromTimestamp(t *testing.T) {
	// Timestamped packet — LastHeard must come from the packet, not arrival.
	raw := "SP9ABC>APRS,TCPIP*:@091425z5023.45N/02012.34E>Test"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("parse failed")
	}
	if sr.LastHeard.IsZero() {
		t.Error("LastHeard should be set from the embedded timestamp")
	}
	// The minute should match the packet (25), regardless of current time.
	if sr.LastHeard.Minute() != 25 {
		t.Errorf("LastHeard minute = %d, want 25", sr.LastHeard.Minute())
	}

	// Non-timestamped packet — parser leaves LastHeard zero; callers set
	// the arrival time as fallback.
	raw2 := "SP9ABC>APRS,TCPIP*:!5023.45N/02012.34E>Test"
	sr2, ok := ParsePositionPacket(raw2)
	if !ok {
		t.Fatal("parse failed")
	}
	if !sr2.LastHeard.IsZero() {
		t.Errorf("LastHeard should be zero without timestamp, got %v", sr2.LastHeard)
	}
}

func TestParsePositionPacket_WeatherClockSkew(t *testing.T) {
	// Real-world WX station with a clock ~2h ahead: the embedded timestamp
	// must not be rolled back to the previous month (which would make the
	// packet look a month stale) — LastHeard falls back to arrival time.
	raw := "SP9EPI-13>APRS,TCPIP*,qAS,SP9EPI:@201540z5002.82N/02014.93E_236/022t075P000h79b10172 /Rad: 0.084 uSv/h"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("weather packet should parse")
	}
	if sr.Callsign != "SP9EPI-13" {
		t.Errorf("callsign = %q, want SP9EPI-13", sr.Callsign)
	}
	if sr.Symbol != "/_" {
		t.Errorf("symbol = %q, want /_", sr.Symbol)
	}
	if sr.LastHeard.IsZero() {
		t.Error("LastHeard should fall back to arrival time")
	}
	// The packet clock is ~2h ahead of arrival: depending on the actual time
	// of day, the embedded timestamp is either clamped to arrival (future
	// within 20 days) or kept as-is (already past). It must never be rolled
	// back to a previous month, so LastHeard must stay within ~31 days of
	// now and never be in the future.
	if d := time.Since(sr.LastHeard); d < -2*time.Minute || d > 31*24*time.Hour {
		t.Errorf("LastHeard = %v, want within the last 31 days", sr.LastHeard)
	}
}
