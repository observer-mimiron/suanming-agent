package domain

import "testing"

func TestValidateStaticJudgmentAcceptsDeterministicPatternCandidates(t *testing.T) {
	state := CharterState{Input: CharterInput{Yongshen: map[string]any{
		"geju_candidate":   "伤官格",
		"geju_combination": "伤官佩印候选（伤官在年干、正印在月干）；伤官生财候选（伤官在年干、正财在时干）",
	}}}
	judgment := validStaticJudgment("伤官格", "伤官佩印")
	if err := ValidateStaticJudgment(state, judgment); err != nil {
		t.Fatalf("ValidateStaticJudgment() error = %v", err)
	}
}

func TestValidateStaticJudgmentRejectsRouteOutsideCandidates(t *testing.T) {
	state := CharterState{Input: CharterInput{Yongshen: map[string]any{
		"geju_candidate":   "伤官格",
		"geju_combination": "伤官佩印候选（伤官在年干、正印在月干）",
	}}}
	if err := ValidateStaticJudgment(state, validStaticJudgment("伤官格", "伤官生财")); err == nil {
		t.Fatal("ValidateStaticJudgment() accepted a route outside chart candidates")
	}
}

func validStaticJudgment(name, route string) StructuredStaticSynthesis {
	return StructuredStaticSynthesis{
		AxisStatus:        "established",
		PatternName:       name,
		PatternRoute:      route,
		PatternEvaluation: "成格受限",
		Claims: []StructuredStaticClaim{
			{Slot: "main_axis", Verdict: "伤官格为主轴", Status: "established"},
			{Slot: "strength", Verdict: "日主偏强", Status: "established"},
			{Slot: "tiaohou", Verdict: "调候待确认", Status: "limited"},
			{Slot: "pattern_usage", Verdict: "伤官佩印为取用路线", Status: "limited"},
		},
	}
}
