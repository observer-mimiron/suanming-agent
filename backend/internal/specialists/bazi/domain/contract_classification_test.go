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
			violation: ValidationViolation{Code: ViolationMethodContract, Field: "dynamic.limitations[1]", AllowedRefs: []string{"dynamic.current_period_ref"}},
			class:     ContractFailureProjectionMismatch,
			policy:    RecoveryPolicyDynamicFactsOnly,
		},
		{
			name:      "missing reference in dynamic reasoning degrades",
			stage:     "dynamic_synthesis",
			violation: ValidationViolation{Code: ViolationClaimNotAuthorized, Field: "dynamic.reasoning", MissingRefs: []string{"dynamic.current_period_ref"}},
			class:     ContractFailureProjectionMismatch,
			policy:    RecoveryPolicyDynamicFactsOnly,
		},
		{
			name:      "dynamic method failure without reference evidence retries",
			stage:     "dynamic_synthesis",
			violation: ValidationViolation{Code: ViolationMethodContract, Field: "dynamic.limitations[1]"},
			class:     ContractFailureMethodContract,
			policy:    RecoveryPolicyRetryOnly,
		},
		{
			name:      "static presentation reference violation retries",
			stage:     "static_synthesis",
			violation: ValidationViolation{Code: ViolationMethodContract, Field: "static.limitations[1]", AllowedRefs: []string{"static.main_axis"}},
			class:     ContractFailureMethodContract,
			policy:    RecoveryPolicyRetryOnly,
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

func TestClassifyViolationUsesStableViolationCodeWhenFindingIsMissing(t *testing.T) {
	failure := ClassifyViolation("dynamic_projection", ValidationViolation{
		Code:    ViolationUnsupportedConcreteOutcome,
		Field:   "dynamic",
		Message: "dynamic synthesis includes a concrete outcome",
	})
	if failure.Class != ContractFailureDomainUnauthorized {
		t.Fatalf("failure class = %q, want %q", failure.Class, ContractFailureDomainUnauthorized)
	}
	if failure.FindingCode != string(ViolationUnsupportedConcreteOutcome) {
		t.Fatalf("finding code = %q, want %q", failure.FindingCode, ViolationUnsupportedConcreteOutcome)
	}
}
