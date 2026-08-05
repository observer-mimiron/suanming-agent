package qimen

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestToolExecute_UsesStandardRotatingEightGateSchema(t *testing.T) {
	questionTime := "1991-10-05T12:40:00+08:00"
	result, err := (&Tool{}).Execute(context.Background(), map[string]any{
		"question_time": questionTime,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	chart := result.(map[string]any)
	if got := chart["pan_schema"]; got != "rotating_8" {
		t.Fatalf("pan_schema = %v, want rotating_8", got)
	}
	if got := chart["symbol_system"]; got != "eight_gate_eight_god" {
		t.Fatalf("symbol_system = %v, want eight_gate_eight_god", got)
	}
	if got := chart["time_source"]; got != "question_time" {
		t.Fatalf("time_source = %v, want question_time", got)
	}
	if got := chart["question_time"]; got != questionTime {
		t.Fatalf("question_time = %v, want %s", got, questionTime)
	}
	if got := chart["ju_text"].(string); !strings.Contains(got, "下元 阴4局 甲寅遁癸") {
		t.Fatalf("ju_text = %q, want 下元 阴4局 甲寅遁癸", got)
	}
	if chart["duty_star_palace"] == "" || chart["duty_door_palace"] == "" {
		t.Fatalf("duty palaces missing: star=%v door=%v", chart["duty_star_palace"], chart["duty_door_palace"])
	}

	for _, cell := range chart["cells"].([]map[string]any) {
		if cell["door"] == "中" {
			t.Fatalf("rotating_8 must not emit 中门: %+v", cell)
		}
		switch cell["god"] {
		case "太常", "勾陈", "朱雀":
			t.Fatalf("rotating_8 must not emit 九神 item %v: %+v", cell["god"], cell)
		}
	}
}

func TestToolExecute_RejectsLegacyNumericParams(t *testing.T) {
	if _, err := (&Tool{}).Execute(context.Background(), map[string]any{
		"year": 1991.0, "month": 10.0, "day": 5.0, "hour": 12.0, "minute": 40.0,
	}); err == nil {
		t.Fatal("legacy numeric qimen params must be rejected")
	}
}

func TestToolExecute_RejectsInvalidQuestionTime(t *testing.T) {
	for _, params := range []map[string]any{
		{}, {"question_time": "2026-08-05 14:30:00"}, {"question_time": "2026-08-05T14:30:00+08:00:00"},
	} {
		if _, err := (&Tool{}).Execute(context.Background(), params); err == nil {
			t.Fatalf("Execute(%v) succeeded, want invalid question_time error", params)
		}
	}
}

func TestValidateRotatingSymbols_RejectsCrossSystemDoorAndGod(t *testing.T) {
	for name, cells := range map[string][]map[string]any{
		"middle door":         {{"palace": "坎", "door": "中门", "god": "值符"}},
		"middle abbreviation": {{"palace": "坎", "door": "中", "god": "值符"}},
		"太常":                  {{"palace": "坎", "door": "休", "god": "太常"}},
		"勾陈":                  {{"palace": "坎", "door": "休", "god": "勾陈"}},
		"朱雀":                  {{"palace": "坎", "door": "休", "god": "朱雀"}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateRotatingSymbols(cells); err == nil {
				t.Fatal("invalid rotating_8 symbol was accepted")
			}
		})
	}

	if err := validateRotatingSymbols([]map[string]any{{"palace": "中", "door": "休", "god": "值符"}}); err != nil {
		t.Fatalf("middle palace must remain valid: %v", err)
	}
}

func TestToolExecute_QuestionTimeIsRFC3339(t *testing.T) {
	const input = "2026-08-05T14:30:00+08:00"
	result, err := (&Tool{}).Execute(context.Background(), map[string]any{"question_time": input})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	got, ok := result.(map[string]any)["question_time"].(string)
	if !ok {
		t.Fatalf("question_time type = %T, want string", result.(map[string]any)["question_time"])
	}
	if _, err := time.Parse(time.RFC3339, got); err != nil {
		t.Fatalf("question_time %q is not RFC3339: %v", got, err)
	}
}
