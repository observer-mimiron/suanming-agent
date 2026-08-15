package application

import (
	"testing"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
)

func TestQuestionTimeParamsOnlyExposeQuestionTime(t *testing.T) {
	questionTime := time.Date(2026, time.August, 5, 14, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	params := QuestionTimeParams(questionTime)
	if len(params) != 1 || params["question_time"] != "2026-08-05T14:30:00+08:00" {
		t.Fatalf("QuestionTimeParams() = %#v, want only RFC3339 question_time", params)
	}
}

func TestMatchesStoredCaseChartRequiresExactTurnContract(t *testing.T) {
	turn := contracts.TurnContext{CaseID: "case-1", QuestionTime: "2026-08-05T14:30:00+08:00"}
	valid := map[string]any{
		"case_id":       "case-1",
		"purpose":       "event_question",
		"owner_ref":     map[string]any{"kind": "case", "id": "case-1"},
		"question_time": turn.QuestionTime,
		"time_source":   "question_time",
		"pan_schema":    "rotating_8",
		"symbol_system": "eight_gate_eight_god",
	}
	if !MatchesStoredCaseChart(valid, turn) {
		t.Fatal("valid case chart was rejected")
	}

	for name, mutate := range map[string]func(map[string]any){
		"wrong case":          func(chart map[string]any) { chart["case_id"] = "case-2" },
		"wrong owner":         func(chart map[string]any) { chart["owner_ref"] = map[string]any{"kind": "profile", "id": "case-1"} },
		"wrong purpose":       func(chart map[string]any) { chart["purpose"] = "natal_chart" },
		"wrong time source":   func(chart map[string]any) { chart["time_source"] = "system_clock" },
		"wrong question time": func(chart map[string]any) { chart["question_time"] = "2026-08-05T14:31:00+08:00" },
		"wrong schema":        func(chart map[string]any) { chart["pan_schema"] = "flying_9" },
		"wrong symbols":       func(chart map[string]any) { chart["symbol_system"] = "nine_gate_nine_god" },
	} {
		t.Run(name, func(t *testing.T) {
			chart := cloneMap(valid)
			mutate(chart)
			if MatchesStoredCaseChart(chart, turn) {
				t.Fatal("invalid case chart was accepted")
			}
		})
	}
}

func cloneMap(src map[string]any) map[string]any {
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}
