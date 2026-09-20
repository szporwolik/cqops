package aprs

import "testing"

func TestParseWeather_Full(t *testing.T) {
	w, ok := ParseWeather("236/022t075P000h79b10172 /Rad: 0.084 uSv/h")
	if !ok {
		t.Fatal("weather block should parse")
	}
	if !w.WindOk || w.WindDir != 236 || w.WindMph != 22 {
		t.Errorf("wind = %v/%v, want 236/22", w.WindDir, w.WindMph)
	}
	if !w.TempOk || w.TempF != 75 {
		t.Errorf("temp = %v, want 75", w.TempF)
	}
	if !w.HumOk || w.Humidity != 79 {
		t.Errorf("humidity = %v, want 79", w.Humidity)
	}
	if !w.PressOk || w.Pressure != 1017.2 {
		t.Errorf("pressure = %v, want 1017.2", w.Pressure)
	}
	if w.Rest != "/Rad: 0.084 uSv/h" {
		t.Errorf("rest = %q", w.Rest)
	}
}

func TestParseWeather_FormatMetric(t *testing.T) {
	w, _ := ParseWeather("236/022t075h79b10172")
	want := "236\u00b0 35 km/h \u00b7 24\u00b0C \u00b7 79% \u00b7 1017 hPa"
	if got := w.Format(true); got != want {
		t.Errorf("metric = %q, want %q", got, want)
	}
}

func TestParseWeather_FormatImperial(t *testing.T) {
	w, _ := ParseWeather("236/022t075h79b10172")
	want := "236\u00b0 22 mph \u00b7 75\u00b0F \u00b7 79% \u00b7 1017 hPa"
	if got := w.Format(false); got != want {
		t.Errorf("imperial = %q, want %q", got, want)
	}
}

func TestParseWeather_NoWind(t *testing.T) {
	w, ok := ParseWeather("001/000t075h42")
	if !ok {
		t.Fatal("should parse")
	}
	// No wind reported — the wind part must be omitted.
	if got := w.Format(true); got != "24\u00b0C \u00b7 42%" {
		t.Errorf("no-wind format = %q", got)
	}
}

func TestParseWeather_Humidity100(t *testing.T) {
	w, _ := ParseWeather("000/000h00")
	if !w.HumOk || w.Humidity != 100 {
		t.Errorf("humidity = %v, want 100", w.Humidity)
	}
}

func TestParseWeather_NegativeTemp(t *testing.T) {
	w, ok := ParseWeather("000/000t-12h80")
	if !ok || !w.TempOk || w.TempF != -12 {
		t.Fatalf("negative temp: ok=%v w=%+v", ok, w)
	}
	// -12°F → -24°C.
	if got := w.Format(true); got != "-24\u00b0C \u00b7 80%" {
		t.Errorf("negative temp format = %q", got)
	}
}

func TestParseWeather_NotWeather(t *testing.T) {
	for _, c := range []string{"", "Hello", "/Rad: 0.084 uSv/h", "23/abc"} {
		if _, ok := ParseWeather(c); ok {
			t.Errorf("ParseWeather(%q) should fail", c)
		}
	}
}
