package supervisor

import "testing"

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

func TestParseAndValidate_CollectProfileWithoutProfileData(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "bazi",
		"task_intent": "collect_profile",
		"confidence": 0.9,
		"slots": {
			"profile": {},
			"question_text": ""
		}
	}`

	_, err := parseAndValidate(raw)
	if err == nil {
		t.Fatal("collect_profile with empty profile must return error")
	}
}

func TestParseAndValidate_CollectProfileWithProfileData(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "bazi",
		"task_intent": "collect_profile",
		"confidence": 0.9,
		"slots": {
			"profile": {"year": 1990, "month": 5, "day": 20, "hour": 8, "gender": "男", "birthplace": "北京"},
			"question_text": ""
		}
	}`

	d, err := parseAndValidate(raw)
	if err != nil {
		t.Fatalf("collect_profile with profile data should pass: %v", err)
	}
	if d.TaskIntent != "collect_profile" {
		t.Fatalf("TaskIntent: got %q, want collect_profile", d.TaskIntent)
	}
}

func TestParseAndValidate_TimingFollowupWithoutQuestionText(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "qimen",
		"task_intent": "timing_followup",
		"confidence": 0.9,
		"slots": {
			"profile": {},
			"question_text": ""
		}
	}`

	_, err := parseAndValidate(raw)
	if err == nil {
		t.Fatal("timing_followup with empty question_text must return error")
	}
}

func TestParseAndValidate_ValidNonCollectProfilePasses(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "bazi",
		"task_intent": "interpret_chart",
		"confidence": 0.95,
		"slots": {
			"profile": {},
			"question_text": "我的财运如何"
		}
	}`

	d, err := parseAndValidate(raw)
	if err != nil {
		t.Fatalf("interpret_chart with empty profile should pass (profile not needed): %v", err)
	}
	if d.TaskIntent != "interpret_chart" {
		t.Fatalf("TaskIntent: got %q, want interpret_chart", d.TaskIntent)
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

