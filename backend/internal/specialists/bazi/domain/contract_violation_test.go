package domain

import "testing"

func TestValidationErrorNormalizesReferences(t *testing.T) {
	err := NewValidationError(ViolationFactConflict, "static.axis", "static.main_axis", "conflict", []string{" ref ", ""}, []string{" allowed "})
	violation, ok := ValidationViolationFromError(err)
	if !ok || violation.Code != ViolationFactConflict || len(violation.MissingRefs) != 1 || violation.MissingRefs[0] != "ref" || len(violation.AllowedRefs) != 1 || violation.AllowedRefs[0] != "allowed" {
		t.Fatalf("violation = %#v, ok=%v", violation, ok)
	}
}
