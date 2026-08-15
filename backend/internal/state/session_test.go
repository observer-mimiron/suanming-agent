// This test file belongs to the session state layer.
// It verifies session API behavior and protects the related contract from regressions.
// It stores session truth; routing and interpretation decisions stay outside state structs.
package state

import "testing"

func TestSessionState(t *testing.T) {
	s := NewSession("test1")
	if s.IsProfileComplete() {
		t.Error("empty profile should not be complete")
	}
	s.MergeProfile(map[string]any{"year": 1990, "month": 5, "day": 20, "hour": 8, "gender": "男", "birthplace": "北京"})
	if !s.IsProfileComplete() {
		t.Error("should be complete")
	}
	if len(s.MissingFields()) != 0 {
		t.Error("should have no missing")
	}
}

func TestMergeProfileNormalizesGenderExpressions(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{input: "男命", want: "男"},
		{input: "男性", want: "男"},
		{input: "male", want: "男"},
		{input: "女命", want: "女"},
		{input: "female", want: "女"},
	} {
		s := NewSession("gender-" + tc.input)
		s.MergeProfile(map[string]any{"gender": tc.input})
		if got := s.ActiveProfile()["gender"]; got != tc.want {
			t.Fatalf("gender %q normalized to %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestClone_PreservesRoutingSnapshot(t *testing.T) {
	s := NewSession("test-clone")
	s.Routing = RoutingSnapshot{
		ConversationIntent:    "consult",
		PrimaryDomain:         "bazi",
		SecondaryDomains:      []string{"qimen"},
		TaskIntent:            "interpret_chart",
		AwaitingClarification: false,
		Confidence:            0.95,
	}
	s.LastInput.PreferredDomain = "bazi"
	s.Execution.PrimaryDomain = "bazi"
	s.Execution.RequiredArtifacts = []string{"bazi_chart"}

	clone := s.Clone()
	if clone.Routing.ConversationIntent != "consult" {
		t.Fatalf("clone.Routing.ConversationIntent: got %q, want %q", clone.Routing.ConversationIntent, "consult")
	}
	if clone.Routing.PrimaryDomain != "bazi" {
		t.Fatalf("clone.Routing.PrimaryDomain: got %q, want %q", clone.Routing.PrimaryDomain, "bazi")
	}
	if len(clone.Routing.SecondaryDomains) != 1 || clone.Routing.SecondaryDomains[0] != "qimen" {
		t.Fatalf("clone.Routing.SecondaryDomains: got %v, want [qimen]", clone.Routing.SecondaryDomains)
	}
	if clone.Routing.Confidence != 0.95 {
		t.Fatalf("clone.Routing.Confidence: got %f, want 0.95", clone.Routing.Confidence)
	}
	if clone.LastInput.PreferredDomain != "bazi" {
		t.Fatalf("clone.LastInput.PreferredDomain: got %q, want bazi", clone.LastInput.PreferredDomain)
	}
	if clone.Execution.PrimaryDomain != "bazi" {
		t.Fatalf("clone.Execution.PrimaryDomain: got %q, want bazi", clone.Execution.PrimaryDomain)
	}
	if len(clone.Execution.RequiredArtifacts) != 1 || clone.Execution.RequiredArtifacts[0] != "bazi_chart" {
		t.Fatalf("clone.Execution.RequiredArtifacts: got %v, want [bazi_chart]", clone.Execution.RequiredArtifacts)
	}
}

func TestClone_RoutingIndependent(t *testing.T) {
	s := NewSession("test-indep")
	s.Routing = RoutingSnapshot{PrimaryDomain: "bazi"}

	clone := s.Clone()
	clone.Routing.PrimaryDomain = "qimen"

	// Original must be unchanged.
	if s.Routing.PrimaryDomain != "bazi" {
		t.Fatalf("original Routing.PrimaryDomain must not be affected by clone mutation")
	}
}

func TestBackwardCompat_OldSessionLoads(t *testing.T) {
	// Simulate an old session that has no routing/domain fields.
	s := &SessionState{
		SessionID:         "old-session",
		Profile:           map[string]any{"year": 1990.0},
		BaziResult:        map[string]any{"dayGan": "甲"},
		ConversationStage: "ready",
		RecentTurns:       make([]Turn, 0),
	}

	// Old fields must still work.
	if !s.HasBaziResult() {
		t.Fatal("HasBaziResult should return true for old session")
	}
	// Routing should be zero-value for old sessions.
	if s.Routing.PrimaryDomain != "" {
		t.Fatal("old session Routing.PrimaryDomain should be empty")
	}
	// ActivePrimaryDomain should default safely.
	if s.ActivePrimaryDomain() != "bazi" {
		t.Fatalf("old session ActivePrimaryDomain: got %q, want %q", s.ActivePrimaryDomain(), "bazi")
	}
	// Clone should not panic on old sessions.
	c := s.Clone()
	if c == nil {
		t.Fatal("Clone of old session should not be nil")
	}
	if !c.HasBaziResult() {
		t.Fatal("Clone of old session should preserve BaziResult")
	}
}
