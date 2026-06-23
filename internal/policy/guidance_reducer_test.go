package policy

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/state"
)

func TestReduceGuidance_AcceptedOfferTransitionsToChooseTopic(t *testing.T) {
	got := ReduceGuidance(GuidanceReducerInput{
		Current: &state.GuidanceState{
			DirectiveKind: "offer_consult",
		},
		Message: "行，那你看看",
	})

	if got == nil {
		t.Fatal("ReduceGuidance() = nil, want guidance state")
	}
	if got.DirectiveKind != "choose_topic" {
		t.Fatalf("DirectiveKind = %q, want choose_topic", got.DirectiveKind)
	}
	if got.PendingSlot != "" {
		t.Fatalf("PendingSlot = %q, want empty", got.PendingSlot)
	}
}


func TestReduceGuidance_OfferConsultAmbiguousReplyAccumulatesRetry(t *testing.T) {
	got := ReduceGuidance(GuidanceReducerInput{
		Current: &state.GuidanceState{DirectiveKind: "offer_consult"},
		Message: "嗯",
	})
	if got == nil {
		t.Fatal("ReduceGuidance() = nil, want guidance with retry incremented")
	}
	if got.DirectiveKind != "offer_consult" {
		t.Fatalf("DirectiveKind = %q, want offer_consult", got.DirectiveKind)
	}
	if got.RetryCount != 1 {
		t.Fatalf("RetryCount = %d, want 1", got.RetryCount)
	}
}

func TestReduceGuidance_ChooseTopicAmbiguousReplyAccumulatesRetry(t *testing.T) {
	got := ReduceGuidance(GuidanceReducerInput{
		Current: &state.GuidanceState{DirectiveKind: "choose_topic"},
		Message: "都行",
	})
	if got == nil {
		t.Fatal("ReduceGuidance() = nil, want guidance with retry incremented")
	}
	if got.DirectiveKind != "choose_topic" {
		t.Fatalf("DirectiveKind = %q, want choose_topic", got.DirectiveKind)
	}
	if got.RetryCount != 1 {
		t.Fatalf("RetryCount = %d, want 1", got.RetryCount)
	}
}

func TestReduceGuidance_CollectSlotAmbiguousReplyAccumulatesRetry(t *testing.T) {
	got := ReduceGuidance(GuidanceReducerInput{
		Current: &state.GuidanceState{
			DirectiveKind: "collect_slot",
			PendingSlot:   "birth_date",
		},
		Message: "不知道",
		Profile: map[string]any{},
	})
	if got == nil {
		t.Fatal("ReduceGuidance() = nil, want guidance with retry incremented")
	}
	if got.DirectiveKind != "collect_slot" {
		t.Fatalf("DirectiveKind = %q, want collect_slot", got.DirectiveKind)
	}
	if got.RetryCount != 1 {
		t.Fatalf("RetryCount = %d, want 1", got.RetryCount)
	}
}

func TestReduceGuidance_RetryThresholdEscalatesToGuidedFallback(t *testing.T) {
	got := ReduceGuidance(GuidanceReducerInput{
		Current: &state.GuidanceState{
			DirectiveKind: "choose_topic",
			RetryCount:    1,
		},
		Message: "都行",
	})
	if got == nil {
		t.Fatal("ReduceGuidance() = nil, want guided_fallback")
	}
	if got.DirectiveKind != "guided_fallback" {
		t.Fatalf("DirectiveKind = %q, want guided_fallback", got.DirectiveKind)
	}
	if got.RetryCount != 2 {
		t.Fatalf("RetryCount = %d, want 2", got.RetryCount)
	}
}

func TestReduceGuidance_TopicSwitchResetsRetry(t *testing.T) {
	got := ReduceGuidance(GuidanceReducerInput{
		Current: &state.GuidanceState{
			DirectiveKind: "choose_topic",
			RetryCount:    1,
		},
		Message: "看看感情",
	})
	if got == nil {
		t.Fatal("ReduceGuidance() = nil, want collect_slot with reset retry")
	}
	if got.RetryCount != 0 {
		t.Fatalf("RetryCount = %d, want 0 (reset on topic switch)", got.RetryCount)
	}
	if got.ChosenTopic != "感情" {
		t.Fatalf("ChosenTopic = %q, want 感情", got.ChosenTopic)
	}
}

func TestReduceGuidance_SlotFilledAutoAdvances(t *testing.T) {
	got := ReduceGuidance(GuidanceReducerInput{
		Current: &state.GuidanceState{
			DirectiveKind: "collect_slot",
			PendingSlot:   "birth_date",
		},
		Message: "1990年5月20日",
		Profile: map[string]any{
			"year":  1990.0,
			"month": 5.0,
			"day":   20.0,
		},
	})
	if got == nil {
		t.Fatal("ReduceGuidance() = nil, want auto-advance to next slot")
	}
	if got.PendingSlot != "birth_time" {
		t.Fatalf("PendingSlot = %q, want birth_time", got.PendingSlot)
	}
	if got.RetryCount != 0 {
		t.Fatalf("RetryCount = %d, want 0 (reset on slot advance)", got.RetryCount)
	}
}

func TestReduceGuidance_AllSlotsFilledReturnsNil(t *testing.T) {
	got := ReduceGuidance(GuidanceReducerInput{
		Current: &state.GuidanceState{
			DirectiveKind: "collect_slot",
			PendingSlot:   "birthplace",
		},
		Message: "北京",
		Profile: map[string]any{
			"year":       1990.0,
			"month":      5.0,
			"day":        20.0,
			"hour":       8.0,
			"gender":     "男",
			"birthplace": "北京",
		},
	})
	if got != nil {
		t.Fatalf("ReduceGuidance() = %#v, want nil (all slots done)", got)
	}
}

func TestReduceGuidance_TopicSwitchReusesState(t *testing.T) {
	got := ReduceGuidance(GuidanceReducerInput{
		Current: &state.GuidanceState{
			DirectiveKind: "collect_slot",
			ChosenTopic:   "事业",
			PendingSlot:   "birth_time",
			RetryCount:    1,
		},
		Message: "先不看事业了，看看感情",
		Profile: map[string]any{
			"year":  1990.0,
			"month": 5.0,
			"day":   20.0,
		},
	})

	if got == nil {
		t.Fatal("ReduceGuidance() = nil, want guidance state")
	}
	if got.ChosenTopic != "感情" {
		t.Fatalf("ChosenTopic = %q, want 感情", got.ChosenTopic)
	}
	if got.PendingSlot != "birth_time" {
		t.Fatalf("PendingSlot = %q, want birth_time", got.PendingSlot)
	}
	if got.RetryCount != 1 {
		t.Fatalf("RetryCount = %d, want 1", got.RetryCount)
	}
}

func TestReduceGuidance_OfferConsultAcceptWithTopicSkipsChooseTopic(t *testing.T) {
	got := ReduceGuidance(GuidanceReducerInput{
		Current: &state.GuidanceState{
			DirectiveKind: "offer_consult",
		},
		Message: "可以，看看事业",
		Profile: map[string]any{},
	})
	if got == nil {
		t.Fatal("ReduceGuidance() = nil, want collect_slot")
	}
	if got.DirectiveKind != "collect_slot" {
		t.Fatalf("DirectiveKind = %q, want collect_slot (should skip choose_topic)", got.DirectiveKind)
	}
	if got.ChosenTopic != "事业" {
		t.Fatalf("ChosenTopic = %q, want 事业", got.ChosenTopic)
	}
	if got.PendingSlot != "birth_date" {
		t.Fatalf("PendingSlot = %q, want birth_date", got.PendingSlot)
	}
}

func TestReduceGuidance_OfferConsultAcceptWithTopicAndFullProfileReturnsNil(t *testing.T) {
	got := ReduceGuidance(GuidanceReducerInput{
		Current: &state.GuidanceState{
			DirectiveKind: "offer_consult",
		},
		Message: "可以，看看事业",
		Profile: map[string]any{
			"year":       1990.0,
			"month":      5.0,
			"day":        20.0,
			"hour":       8.0,
			"gender":     "男",
			"birthplace": "北京",
		},
	})
	if got != nil {
		t.Fatalf("ReduceGuidance() = %#v, want nil (profile complete, guidance done)", got)
	}
}
