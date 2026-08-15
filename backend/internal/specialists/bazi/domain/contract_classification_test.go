package domain

import "testing"

func TestClassifyViolationDynamicPresentationReferenceViolation(t *testing.T) {
	tests := []struct {
		name      string
		stage     string
		violation ValidationViolation
		class     string
		policy    string
	}{
		{
			name:      "allowed reference catalog in dynamic limitation degrades",
			stage:     "dynamic_synthesis",
			violation: ValidationViolation{Code: ViolationMethodContract, Field: "dynamic.limitations[1]", AllowedRefs: []string{"tier_assessment"}},
			class:     ContractFailureProjectionMismatch,
			policy:    RecoveryPolicyDynamicFactsOnly,
		},
		{
			name:      "missing reference in dynamic reasoning degrades",
			stage:     "dynamic_synthesis",
			violation: ValidationViolation{Code: ViolationClaimNotAuthorized, Field: "dynamic.reasoning", MissingRefs: []string{"tier_assessment"}},
			class:     ContractFailureProjectionMismatch,
			policy:    RecoveryPolicyDynamicFactsOnly,
		},
		{
			name:      "dynamic method failure without reference evidence stays hard error",
			stage:     "dynamic_synthesis",
			violation: ValidationViolation{Code: ViolationMethodContract, Field: "dynamic.limitations[1]"},
			class:     ContractFailureMethodContract,
			policy:    RecoveryPolicyHardError,
		},
		{
			name:      "static presentation reference violation stays hard error",
			stage:     "static_synthesis",
			violation: ValidationViolation{Code: ViolationMethodContract, Field: "static.limitations[1]", AllowedRefs: []string{"tier_assessment"}},
			class:     ContractFailureMethodContract,
			policy:    RecoveryPolicyHardError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failure := ClassifyViolation(tt.stage, tt.violation)
			if failure.Class != tt.class || failure.RecoveryPolicy != tt.policy {
				t.Fatalf("failure = %#v, want class=%q policy=%q", failure, tt.class, tt.policy)
			}
		})
	}
}
