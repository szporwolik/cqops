package rotor

import "testing"

func TestStatusZeroValue(t *testing.T) {
	var s Status
	if s.Connected {
		t.Error("zero-value Connected should be false")
	}
	if s.Azimuth != 0 {
		t.Error("zero-value Azimuth should be 0")
	}
	if s.Elevation != 0 {
		t.Error("zero-value Elevation should be 0")
	}
}
