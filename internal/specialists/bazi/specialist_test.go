package bazi

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

func makeSession(profileComplete, hasChart bool) *state.SessionState {
	s := state.NewSession("test-bazi")
	if profileComplete {
		s.MergeProfile(map[string]any{
			"year": 1990.0, "month": 5.0, "day": 20.0,
			"hour": 8.0, "gender": "男", "birthplace": "北京",
		})
	}
	if hasChart {
		s.BaziResult = map[string]any{
			"dayGan": "甲", "dayZhi": "子",
			"yearGan": "庚", "yearZhi": "午",
			"monthGan": "丙", "monthZhi": "辰",
			"hourGan": "戊", "hourZhi": "申",
		}
	}
	return s
}

func TestBaziSpecialist_IncompleteProfileReturnsClarification(t *testing.T) {
	sp := New(nil)
	st := makeSession(false, false)
	route := specialists.ApprovedRoute{
		ConversationIntent:    "consult",
		PrimaryDomain:         "bazi",
		TaskIntent:            "interpret_chart",
		NeedsClarification:    false,
		Slots:                 schemas.DecisionSlots{QuestionText: "我的财运如何"},
	}

	result, err := sp.Run(context.Background(), st, route, noopSink(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Final {
		t.Fatal("incomplete profile should produce a final clarification result")
	}
	if result.Summary == "" {
		t.Fatal("incomplete profile should return a clarification message")
	}
}

func TestBaziSpecialist_ReusableChartFollowup(t *testing.T) {
	sp := New(nil)
	st := makeSession(true, true)
	route := specialists.ApprovedRoute{
		ConversationIntent:    "consult",
		PrimaryDomain:         "bazi",
		TaskIntent:            "fortune_followup",
		NeedsClarification:    false,
		Slots:                 schemas.DecisionSlots{QuestionText: "今年运势如何"},
		PolicyHints:           schemas.PolicyHints{CanReuseCachedResult: true},
	}

	result, err := sp.Run(context.Background(), st, route, noopSink(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Domain != "bazi" {
		t.Fatalf("Domain: got %q, want %q", result.Domain, "bazi")
	}
}

func TestBaziSpecialist_NewProfileCompleteReading(t *testing.T) {
	sp := New(nil)
	st := makeSession(true, false)
	route := specialists.ApprovedRoute{
		ConversationIntent:    "consult",
		PrimaryDomain:         "bazi",
		TaskIntent:            "collect_profile",
		NeedsClarification:    false,
		Slots:                 schemas.DecisionSlots{QuestionText: "这是我的八字，帮我看看"},
	}

	result, err := sp.Run(context.Background(), st, route, noopSink(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Domain != "bazi" {
		t.Fatalf("Domain: got %q, want %q", result.Domain, "bazi")
	}
}
