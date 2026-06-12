package supervisor

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/schemas"
)

func TestParseDecision_ValidJSON(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "bazi",
		"secondary_domains": [],
		"task_intent": "interpret_chart",
		"needs_clarification": false,
		"clarification_question": "",
		"parallelizable": false,
		"confidence": 0.95,
		"slots": {
			"profile": {"name": "张三", "gender": "男"},
			"question_text": "我的财运如何",
			"time_scope": "今年",
			"target_subject": "",
			"language": "zh"
		},
		"policy_hints": {
			"needs_knowledge": true,
			"needs_qimen": false,
			"can_reuse_session_profile": false,
			"can_reuse_cached_result": false
		}
	}`

	got := parseDecision(raw)
	if got.ConversationIntent != "consult" {
		t.Fatalf("ConversationIntent: got %q, want %q", got.ConversationIntent, "consult")
	}
	if got.PrimaryDomain != "bazi" {
		t.Fatalf("PrimaryDomain: got %q, want %q", got.PrimaryDomain, "bazi")
	}
	if got.TaskIntent != "interpret_chart" {
		t.Fatalf("TaskIntent: got %q, want %q", got.TaskIntent, "interpret_chart")
	}
	if got.Confidence != 0.95 {
		t.Fatalf("Confidence: got %f, want %f", got.Confidence, 0.95)
	}
	if got.Slots.QuestionText != "我的财运如何" {
		t.Fatalf("Slots.QuestionText: got %q, want %q", got.Slots.QuestionText, "我的财运如何")
	}
	if got.Slots.TimeScope != "今年" {
		t.Fatalf("Slots.TimeScope: got %q, want %q", got.Slots.TimeScope, "今年")
	}
	if !got.PolicyHints.NeedsKnowledge {
		t.Fatal("PolicyHints.NeedsKnowledge: got false, want true")
	}
	if got.PolicyHints.NeedsQimen {
		t.Fatal("PolicyHints.NeedsQimen: got true, want false")
	}
}

func TestParseDecision_NormalizesDefaults(t *testing.T) {
	raw := `{"primary_domain":"bazi"}`
	got := parseDecision(raw)

	if got.ConversationIntent != "consult" {
		t.Fatalf("ConversationIntent: got %q, want %q", got.ConversationIntent, "consult")
	}
	if got.PrimaryDomain != "bazi" {
		t.Fatalf("PrimaryDomain: got %q, want %q", got.PrimaryDomain, "bazi")
	}
	if got.TaskIntent == "" {
		t.Fatal("TaskIntent should not be empty after normalization")
	}
	if got.SecondaryDomains == nil {
		t.Fatal("SecondaryDomains should not be nil after normalization")
	}
	if got.Slots.Profile == nil {
		t.Fatal("Slots.Profile should not be nil after normalization")
	}
}

func TestParseDecision_MalformedJSON(t *testing.T) {
	raw := `not valid json at all {{{`
	got := parseDecision(raw)

	// Malformed JSON must fall back to safe defaults, not panic.
	if got.ConversationIntent != "consult" {
		t.Fatalf("malformed JSON: ConversationIntent got %q, want %q", got.ConversationIntent, "consult")
	}
	if got.PrimaryDomain != "bazi" {
		t.Fatalf("malformed JSON: PrimaryDomain got %q, want %q", got.PrimaryDomain, "bazi")
	}
}

func TestParseDecision_EmptyOutput(t *testing.T) {
	raw := ""
	got := parseDecision(raw)

	// Empty output must not panic and must return safe defaults.
	if got.ConversationIntent != "consult" {
		t.Fatalf("empty output: ConversationIntent got %q, want %q", got.ConversationIntent, "consult")
	}
	if got.PrimaryDomain != "bazi" {
		t.Fatalf("empty output: PrimaryDomain got %q, want %q", got.PrimaryDomain, "bazi")
	}
}

func TestParseDecision_InvalidConfidence(t *testing.T) {
	raw := `{"conversation_intent":"consult","primary_domain":"bazi","confidence":-0.5}`
	got := parseDecision(raw)

	// Negative confidence must be clamped to 0 after normalization.
	if got.Confidence < 0 {
		t.Fatalf("Confidence: got %f, want >= 0", got.Confidence)
	}
}

func TestParseDecision_QimenSecondaryDomain(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "bazi",
		"secondary_domains": ["qimen"],
		"task_intent": "cross_domain_consult",
		"confidence": 0.85
	}`

	got := parseDecision(raw)
	if got.PrimaryDomain != "bazi" {
		t.Fatalf("PrimaryDomain: got %q, want %q", got.PrimaryDomain, "bazi")
	}
	if len(got.SecondaryDomains) == 0 || got.SecondaryDomains[0] != "qimen" {
		t.Fatalf("SecondaryDomains: got %v, want [qimen]", got.SecondaryDomains)
	}
}

// stub parseDecision for TDD — Task 5 will implement the real version.
func parseDecision(raw string) schemas.SupervisorDecision {
	// Stub returns zero value; tests expect this to fail.
	return schemas.SupervisorDecision{}
}
