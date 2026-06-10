package state

import "testing"

func TestSessionState(t *testing.T) {
	s := NewSession("test1")
	if s.IsProfileComplete() {
		t.Error("empty profile should not be complete")
	}
	s.MergeProfile(map[string]any{"year": 1990, "month": 5, "day": 20, "hour": 8, "gender": "男"})
	if !s.IsProfileComplete() {
		t.Error("should be complete")
	}
	if len(s.MissingFields()) != 0 {
		t.Error("should have no missing")
	}
}
