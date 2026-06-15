package orchestrator

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
)

func TestBridgeDecisionDirectBaziExtractsPillars(t *testing.T) {
	route := policy.ApprovedRoute{
		TaskIntent: "direct_bazi",
		Slots: schemas.DecisionSlots{
			QuestionText: "分析这个八字",
		},
	}

	action, patch, question, needsQimen, rawBazi := bridgeDecision(route, "乙巳 丁亥 甲申 甲子，分析这个八字")

	if action != "bazi_input" {
		t.Fatalf("action = %q, want bazi_input", action)
	}
	if len(patch) != 0 {
		t.Fatalf("patch = %#v, want empty patch", patch)
	}
	if question != "分析这个八字" {
		t.Fatalf("question = %q, want 分析这个八字", question)
	}
	if needsQimen {
		t.Fatal("needsQimen = true, want false")
	}
	want := []string{"乙巳", "丁亥", "甲申", "甲子"}
	if len(rawBazi) != len(want) {
		t.Fatalf("rawBazi length = %d, want %d", len(rawBazi), len(want))
	}
	for i := range want {
		if rawBazi[i] != want[i] {
			t.Fatalf("rawBazi[%d] = %q, want %q", i, rawBazi[i], want[i])
		}
	}
}

func TestBridgeDecisionNeedsClarificationShortCircuits(t *testing.T) {
	route := policy.ApprovedRoute{
		NeedsClarification: true,
		PolicyHints: schemas.PolicyHints{
			NeedsQimen: true,
		},
		Slots: schemas.DecisionSlots{
			Profile:      map[string]any{"gender": "女"},
			QuestionText: "今年怎么样",
		},
	}

	action, patch, question, needsQimen, rawBazi := bridgeDecision(route, "今年怎么样")

	if action != "incomplete" {
		t.Fatalf("action = %q, want incomplete", action)
	}
	if patch["gender"] != "女" {
		t.Fatalf("patch = %#v, want preserved profile patch", patch)
	}
	if question != "今年怎么样" {
		t.Fatalf("question = %q, want 今年怎么样", question)
	}
	if !needsQimen {
		t.Fatal("needsQimen = false, want true")
	}
	if rawBazi != nil {
		t.Fatalf("rawBazi = %#v, want nil", rawBazi)
	}
}
