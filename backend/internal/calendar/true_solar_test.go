package calendar

import "testing"

func TestTrueSolarOffsetMinutes_ChangesFourMinutesPerLongitudeDegree(t *testing.T) {
	base := TrueSolarOffsetMinutes(2025, 6, 21, 120)
	if got := TrueSolarOffsetMinutes(2025, 6, 21, 124) - base; got != 16 {
		t.Fatalf("longitude delta offset = %d, want 16", got)
	}
	if TrueSolarTimeVersion != "true_solar_v2" {
		t.Fatalf("TrueSolarTimeVersion = %q", TrueSolarTimeVersion)
	}
}
