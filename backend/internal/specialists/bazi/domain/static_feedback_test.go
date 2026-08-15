package domain

import "testing"

func TestValidateStaticStrengthAgainstEvidenceRejectsAxisConflict(t *testing.T) {
	for _, synthesis := range []StaticSynthesis{
		{MainAxis: "日主强旺，以七杀制刃为用。"},
		{PatternOutcome: "日主强旺，食神制杀有力。"},
	} {
		err := ValidateStaticStrengthAgainstEvidence(map[string]any{"strength": "偏弱"}, synthesis)
		violation, ok := ValidationViolationFromError(err)
		if !ok || violation.Code != ViolationFactConflict {
			t.Fatalf("err = %v, violation = %#v, want strength conflict", err, violation)
		}
	}
}
