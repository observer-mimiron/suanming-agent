package policy

import (
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

func makeSession(profileComplete, hasChart bool) *state.SessionState {
	s := state.NewSession("test-session")
	if profileComplete {
		s.MergeProfile(map[string]any{
			"year": 1990.0, "month": 5.0, "day": 20.0,
			"hour": 8.0, "gender": "男", "birthplace": "北京",
		})
	}
	if hasChart {
		s.BaziResult = map[string]any{"dayGan": "甲", "dayZhi": "子"}
	}
	return s
}

func TestGate_IncompleteProfileForcesClarification(t *testing.T) {
	st := makeSession(false, false)
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		TaskIntent:         "interpret_chart",
		NeedsClarification: false,
		Confidence:         0.9,
	}
	d.Normalize()

	route := Apply(d, st)
	if !route.NeedsClarification {
		t.Fatal("expected needs_clarification for incomplete profile")
	}
}

func TestGate_CompleteProfileAllowsConsultation(t *testing.T) {
	st := makeSession(true, true)
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		TaskIntent:         "interpret_chart",
		NeedsClarification: false,
		Confidence:         0.9,
	}
	d.Normalize()

	route := Apply(d, st)
	if route.NeedsClarification {
		t.Fatal("complete profile should not force clarification")
	}
	if route.PrimaryDomain != "bazi" {
		t.Fatalf("PrimaryDomain: got %q, want %q", route.PrimaryDomain, "bazi")
	}
	if !route.Gate.Admitted {
		t.Fatal("Gate.Admitted = false, want true")
	}
	if route.Gate.ExecutionMode != "execute" {
		t.Fatalf("Gate.ExecutionMode = %q, want execute", route.Gate.ExecutionMode)
	}
}

func TestGate_UnsupportedSecondaryDomainsDropped(t *testing.T) {
	st := makeSession(true, true)
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		SecondaryDomains:   []string{"emotion", "career"},
		TaskIntent:         "interpret_chart",
		Confidence:         0.85,
	}
	d.Normalize()

	route := Apply(d, st)
	for _, dom := range route.SecondaryDomains {
		if dom == "emotion" || dom == "career" {
			t.Fatalf("non-mingli domain %q should have been dropped in phase 1", dom)
		}
	}
}

func TestGate_ParallelHardDisabled(t *testing.T) {
	st := makeSession(true, true)
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		SecondaryDomains:   []string{"qimen"},
		TaskIntent:         "cross_domain_consult",
		Parallelizable:     true,
		Confidence:         0.95,
	}
	d.Normalize()

	route := Apply(d, st)
	if route.ParallelAllowed {
		t.Fatal("parallel must be hard-disabled in phase 1")
	}
}

func TestGate_LowConfidenceForcesClarification(t *testing.T) {
	st := makeSession(true, true)
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		TaskIntent:         "interpret_chart",
		NeedsClarification: false,
		Confidence:         0.45,
	}
	d.Normalize()

	route := Apply(d, st)
	if !route.NeedsClarification {
		t.Fatal("low confidence should force clarification")
	}
	if route.Gate.Admitted {
		t.Fatal("Gate.Admitted = true, want false")
	}
	if route.Gate.ExecutionMode != "clarify" {
		t.Fatalf("Gate.ExecutionMode = %q, want clarify", route.Gate.ExecutionMode)
	}
}

func TestGate_QimenSurvivesAsSecondary(t *testing.T) {
	st := makeSession(true, true)
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		SecondaryDomains:   []string{"qimen"},
		TaskIntent:         "cross_domain_consult",
		Confidence:         0.85,
	}
	d.Normalize()

	route := Apply(d, st)
	found := false
	for _, dom := range route.SecondaryDomains {
		if dom == "qimen" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("qimen should survive as secondary domain in phase 1")
	}
}

func TestGate_QimenAsPrimaryForTiming(t *testing.T) {
	st := makeSession(true, true)
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "qimen",
		TaskIntent:         "timing_followup",
		Confidence:         0.85,
	}
	d.Normalize()

	route := Apply(d, st)
	// qimen as primary for timing is allowed in phase 1
	if route.PrimaryDomain != "qimen" {
		t.Fatalf("qimen should be allowed as primary for timing: got %q", route.PrimaryDomain)
	}
}

func TestGate_QimenPrimaryWithoutProfileAllowedWhenProfileRequirementNone(t *testing.T) {
	st := makeSession(false, false)
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "qimen",
		TaskIntent:         "fortune_followup",
		Confidence:         0.9,
		Slots: schemas.DecisionSlots{
			QuestionText: "今天运气怎么样",
			TimeScope:    "今天",
		},
		PolicyHints: schemas.PolicyHints{
			NeedsQimen:         true,
			QimenMode:          "primary",
			ProfileRequirement: "none",
		},
	}
	d.Normalize()

	route := Apply(d, st)
	if route.NeedsClarification {
		t.Fatal("qimen primary without profile_requirement should not force clarification")
	}
	if route.PrimaryDomain != "qimen" {
		t.Fatalf("PrimaryDomain: got %q, want qimen", route.PrimaryDomain)
	}
}

func TestGate_QimenPrimaryWithProfileRequirementFullStillClarifies(t *testing.T) {
	st := makeSession(false, false)
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "qimen",
		TaskIntent:         "fortune_followup",
		Confidence:         0.9,
		Slots: schemas.DecisionSlots{
			QuestionText: "结合我的命盘看今天运气怎么样",
			TimeScope:    "今天",
		},
		PolicyHints: schemas.PolicyHints{
			NeedsQimen:         true,
			QimenMode:          "primary",
			ProfileRequirement: "full",
		},
	}
	d.Normalize()

	route := Apply(d, st)
	if !route.NeedsClarification {
		t.Fatal("qimen primary with full profile requirement should still force clarification")
	}
}

func TestGate_QimenPrimaryForRecentLuckWithoutProfileAllowed(t *testing.T) {
	st := makeSession(false, false)
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "qimen",
		TaskIntent:         "fortune_followup",
		Confidence:         0.85,
		Slots: schemas.DecisionSlots{
			QuestionText: "最近运气怎么样",
			TimeScope:    "最近",
		},
		PolicyHints: schemas.PolicyHints{
			NeedsQimen:         true,
			QimenMode:          "primary",
			ProfileRequirement: "none",
		},
	}
	d.Normalize()

	route := Apply(d, st)
	if route.NeedsClarification {
		t.Fatal("recent luck qimen primary should not force clarification without profile")
	}
	if route.PrimaryDomain != "qimen" {
		t.Fatalf("PrimaryDomain: got %q, want qimen", route.PrimaryDomain)
	}
}

func TestGate_ZiweiPrimaryForChildrenQuestionStillNeedsProfileOrChart(t *testing.T) {
	st := makeSession(false, false)
	d := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "ziwei",
		TaskIntent:         "fortune_followup",
		Confidence:         0.85,
		Slots: schemas.DecisionSlots{
			QuestionText: "命里有几个孩子",
		},
	}
	d.Normalize()

	route := Apply(d, st)
	if !route.NeedsClarification {
		t.Fatal("ziwei primary without profile or chart should still force clarification")
	}
	if route.PrimaryDomain != "ziwei" {
		t.Fatalf("PrimaryDomain: got %q, want ziwei", route.PrimaryDomain)
	}
}
