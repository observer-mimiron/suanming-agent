package qimen

import (
	"context"
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

func noopSink(t *testing.T) specialists.EventSink {
	return func(ctx context.Context, evt specialists.Event) error {
		t.Logf("event: type=%s", evt.Type)
		return nil
	}
}

func TestQimenSpecialist_TimingRouteInvokesQimen(t *testing.T) {
	sp := New()
	st := state.NewSession("test-qimen-timing")
	st.BaziResult = map[string]any{"dayGan": "甲"}
	route := specialists.ApprovedRoute{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		SecondaryDomains:   []string{"qimen"},
		TaskIntent:         "timing_followup",
		Slots:              schemas.DecisionSlots{QuestionText: "这个月适合签约吗"},
		PolicyHints:        schemas.PolicyHints{NeedsQimen: true},
	}

	result, err := sp.Run(context.Background(), st, route, noopSink(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Domain != "qimen" {
		t.Fatalf("Domain: got %q, want %q", result.Domain, "qimen")
	}
}

func TestQimenSpecialist_NonTimingRouteSkipsQimen(t *testing.T) {
	sp := New()
	st := state.NewSession("test-qimen-nontiming")
	route := specialists.ApprovedRoute{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		TaskIntent:         "interpret_chart",
		Slots:              schemas.DecisionSlots{QuestionText: "我的性格如何"},
	}

	result, err := sp.Run(context.Background(), st, route, noopSink(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Non-timing route should produce empty/skip result.
	if result.Domain == "qimen" {
		t.Fatal("non-timing route should not invoke qimen specialist")
	}
}

func TestQimenSpecialist_SupplementalNotReplacement(t *testing.T) {
	sp := New()
	st := state.NewSession("test-qimen-supplement")
	st.BaziResult = map[string]any{"dayGan": "甲"}
	route := specialists.ApprovedRoute{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		SecondaryDomains:   []string{"qimen"},
		TaskIntent:         "cross_domain_consult",
		Slots:              schemas.DecisionSlots{QuestionText: "今年什么时候适合跳槽"},
		PolicyHints:        schemas.PolicyHints{NeedsQimen: true},
	}

	result, err := sp.Run(context.Background(), st, route, noopSink(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Final {
		t.Fatal("phase 1 qimen result must be supplemental (Final=false), not a replacement for mainline")
	}
	if result.Domain != "qimen" {
		t.Fatalf("Domain: got %q, want %q", result.Domain, "qimen")
	}
}
