package runtime

import "testing"

func TestRepairPolicyEnforcesBusinessBudget(t *testing.T) {
	policy := DefaultRepairPolicy()
	failure := RepairFailure{
		Domain:   "bazi",
		Stage:    "static_projection",
		Class:    RepairProjectionMismatch,
		Field:    "static.tiaohou_anchor",
		Fallback: "facts_only",
	}

	decision := policy.Decide(failure, NewRepairState())
	if decision.Action != RepairActionRepairNode {
		t.Fatalf("Action = %q, want repair_node", decision.Action)
	}
	if !decision.Repairable || !decision.Retryable || decision.Exhausted {
		t.Fatalf("unexpected decision: %+v", decision)
	}

	state := RecordRepairAttempt(NewRepairState(), RepairAttempt{
		Domain: failure.Domain,
		Stage:  failure.Stage,
		Class:  failure.Class,
		Field:  failure.Field,
		Action: RepairActionRepairNode,
	})
	decision = policy.Decide(failure, state)
	if decision.Action != RepairActionFallback || !decision.Exhausted {
		t.Fatalf("decision after same stage/field attempt = %+v, want exhausted fallback", decision)
	}
}

func TestRepairPolicyEnforcesWholeTurnBudget(t *testing.T) {
	policy := DefaultRepairPolicy()
	state := NewRepairState()
	for _, attempt := range []RepairAttempt{
		{Domain: "bazi", Stage: "static_projection", Field: "static.tiaohou_anchor", Action: RepairActionRepairNode},
		{Domain: "bazi", Stage: "dynamic_projection", Field: "dynamic.dayun", Action: RepairActionRepairNode},
	} {
		state = RecordRepairAttempt(state, attempt)
	}

	decision := policy.Decide(RepairFailure{
		Domain:   "bazi",
		Stage:    "final_guard",
		Class:    RepairProjectionMismatch,
		Field:    "answer",
		Fallback: "facts_only",
	}, state)
	if decision.Action != RepairActionFallback || !decision.Exhausted {
		t.Fatalf("decision after whole-turn budget = %+v, want exhausted fallback", decision)
	}
}

func TestRepairPolicyRejectsFactConflictAndMethodContract(t *testing.T) {
	for _, class := range []RepairClass{RepairFactConflict, RepairMethodContract} {
		decision := DefaultRepairPolicy().Decide(RepairFailure{Class: class}, NewRepairState())
		if decision.Action != RepairActionHardError {
			t.Fatalf("%s action = %q, want hard_error", class, decision.Action)
		}
		if decision.Repairable || decision.Retryable {
			t.Fatalf("%s should not be repairable or retryable: %+v", class, decision)
		}
	}
}

func TestRepairHTTPStatusRetryable(t *testing.T) {
	for _, status := range []int{400, 401, 402} {
		if RepairHTTPStatusRetryable(status) {
			t.Fatalf("status %d retryable = true, want false", status)
		}
		if got := RepairClassForHTTPStatus(status); got != RepairTransportFatal {
			t.Fatalf("status %d class = %q, want transport_fatal", status, got)
		}
	}
	for _, status := range []int{408, 429, 500, 503} {
		if !RepairHTTPStatusRetryable(status) {
			t.Fatalf("status %d retryable = false, want true", status)
		}
		if got := RepairClassForHTTPStatus(status); got != RepairTransportTransient {
			t.Fatalf("status %d class = %q, want transport_transient", status, got)
		}
	}
}

func TestRepairTraceAttrsProjectsOnlySafeFields(t *testing.T) {
	attrs := RepairTraceAttrs(RepairTraceEvent{
		Failure: RepairFailure{
			Domain: "bazi",
			Stage:  "static_projection",
			Class:  RepairProjectionMismatch,
			Field:  "static.tiaohou_anchor",
		},
		Attempt:           1,
		MaxAttempts:       1,
		Action:            RepairActionRepairNode,
		Feedback:          map[string]any{"reason": "private detail", "allowed_fix": []string{"x"}},
		LearningHintCount: 0,
		Exhausted:         false,
		FinalAction:       RepairActionRepairNode,
	})

	if _, ok := attrs["reason"]; ok {
		t.Fatal("trace attrs leaked feedback value")
	}
	keys, ok := attrs["repair.feedback_keys"].([]string)
	if !ok {
		t.Fatalf("feedback keys type = %T, want []string", attrs["repair.feedback_keys"])
	}
	if len(keys) != 2 || keys[0] != "allowed_fix" || keys[1] != "reason" {
		t.Fatalf("feedback keys = %v", keys)
	}
	if got := attrs["repair.class"]; got != "projection_mismatch" {
		t.Fatalf("repair.class = %v", got)
	}
}

func TestRepairFailureFromBaziContract(t *testing.T) {
	err := baziContractAuditError("static_projection", baziContractAuditFinding{
		Code:   "static_projection_mismatch",
		Field:  "static.tiaohou_anchor",
		Reason: "调候锚点不完整",
	})

	failure, ok := repairFailureFromBaziContract("static_projection", err)
	if !ok {
		t.Fatal("repairFailureFromBaziContract returned false")
	}
	if failure.Domain != "bazi" || failure.Class != RepairProjectionMismatch || failure.Field != "static.tiaohou_anchor" {
		t.Fatalf("unexpected repair failure: %+v", failure)
	}
	if !failure.Repairable || !failure.Retryable {
		t.Fatalf("failure should be repairable and retryable: %+v", failure)
	}
}

func TestRepairFailureFromBaziContractStaticStrengthBalanceFactsOnly(t *testing.T) {
	err := baziViolationError(
		baziViolationFactConflict,
		"static.strength_balance",
		"",
		"static strength reverses balance evidence: 偏强",
		nil,
		[]string{"偏强"},
	)

	failure, ok := repairFailureFromBaziContract("static_projection", err)
	if !ok {
		t.Fatal("repairFailureFromBaziContract returned false")
	}
	if failure.Class != RepairFactConflict || failure.Field != "static.strength_balance" || failure.Fallback != "facts_only" {
		t.Fatalf("unexpected repair failure: %+v", failure)
	}
	decision := DefaultRepairPolicy().Decide(failure, NewRepairState())
	if decision.Action != RepairActionHardError || decision.Repairable || decision.Retryable {
		t.Fatalf("decision = %+v, want hard non-repairable before recovery fallback", decision)
	}
}

func TestRepairFailureFromBaziContractStaticProjectionValidators(t *testing.T) {
	tests := []struct {
		name         string
		err          func() error
		wantClass    RepairClass
		wantField    string
		wantFallback string
	}{
		{
			name: "missing static field",
			err: func() error {
				static := validStaticSynthesisForConsistencyTests()
				static.MainAxis = ""
				return validateStaticStage(baziCharterState{StaticSynthesis: static})
			},
			wantClass: RepairProjectionMismatch,
			wantField: "static.main_axis",
		},
		{
			name: "invalid enum",
			err: func() error {
				return validateAllowedValue("static axis level", "乱填", []string{"结构可见", "方向成立"})
			},
			wantClass: RepairProjectionMismatch,
			wantField: "static.axis.level",
		},
		{
			name: "axis conflict ceiling",
			err: func() error {
				static := validStaticSynthesisForConsistencyTests()
				static.AxisLevel = "主轴成立"
				static.EffectOnTiaohou = "冲突"
				static.EffectOnCoreDisease = "中性"
				static.EffectOnJiShenDirection = "缓解"
				static.AxisCeiling = "可作主轴"
				static.ConflictReasons = []string{"调候冲突，不能拔高。"}
				return validateStaticAxisVerdictConsistency(static)
			},
			wantClass: RepairProjectionMismatch,
			wantField: "static.axis_ceiling",
		},
		{
			name: "wording beyond cap",
			err: func() error {
				static := validStaticSynthesisForConsistencyTests()
				static.WordingCap = "保守"
				static.MainAxis = "一飞冲天之象"
				return validateStaticDecisionConsistency(static)
			},
			wantClass: RepairProjectionMismatch,
			wantField: "static.wording_cap",
		},
		{
			name: "evidence boundary",
			err: func() error {
				static := validStaticSynthesisForConsistencyTests()
				static.AxisLevel = "可以拔高"
				return validateStaticEvidenceCoverageBoundary(baziCharterState{
					EvidenceQuality: baziEvidenceQuality{MissingTopics: []string{"bingyao"}},
					StaticSynthesis: static,
				})
			},
			wantClass:    RepairEvidenceOverclaim,
			wantField:    "static.axis_level",
			wantFallback: "facts_only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.err()
			if err == nil {
				t.Fatal("validator returned nil error")
			}
			failure, ok := repairFailureFromBaziContract("static_projection", err)
			if !ok {
				t.Fatalf("repairFailureFromBaziContract returned false for %v", err)
			}
			if failure.Class != tt.wantClass || failure.Field != tt.wantField || failure.Fallback != tt.wantFallback {
				t.Fatalf("failure = %+v; want class=%s field=%s fallback=%q", failure, tt.wantClass, tt.wantField, tt.wantFallback)
			}
		})
	}
}

func TestRepairFailureFromBaziContractFactsOnlyStaticValidators(t *testing.T) {
	tests := []struct {
		name      string
		state     baziCharterState
		wantField string
	}{
		{
			name: "rule profile mismatch",
			state: baziCharterState{
				Input:           baziCharterInput{RuleProfile: baziRuleProfile{ID: "runtime-profile"}},
				StaticSynthesis: baziStaticSynthesis{Source: baziSynthesisSourceFactsOnlyDegraded, RuleProfile: "model-profile", MainAxis: "只列可复算命盘事实。"},
			},
			wantField: "static.facts_only.rule_profile",
		},
		{
			name: "missing degraded message",
			state: baziCharterState{
				StaticSynthesis: baziStaticSynthesis{Source: baziSynthesisSourceFactsOnlyDegraded},
			},
			wantField: "static.facts_only.main_axis",
		},
		{
			name: "missing chart facts",
			state: baziCharterState{
				StaticSynthesis: baziStaticSynthesis{Source: baziSynthesisSourceFactsOnlyDegraded, MainAxis: "只列可复算命盘事实。"},
			},
			wantField: "static.facts_only.chart_facts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFactsOnlyStaticSynthesis(tt.state)
			if err == nil {
				t.Fatal("facts-only validator returned nil error")
			}
			failure, ok := repairFailureFromBaziContract("static_projection", err)
			if !ok {
				t.Fatalf("repairFailureFromBaziContract returned false for %v", err)
			}
			if failure.Class != RepairProjectionMismatch || failure.Field != tt.wantField || failure.Fallback != "" {
				t.Fatalf("failure = %+v; want projection mismatch field=%s without fallback", failure, tt.wantField)
			}
		})
	}
}

func TestRepairFailureFromBaziContractRejectsFactConflictAndMethodContract(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		wantClass RepairClass
	}{
		{name: "fact conflict", code: "branch_tengod_conflict", wantClass: RepairFactConflict},
		{name: "method contract", code: "hidden_axis_uncompared", wantClass: RepairMethodContract},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := baziContractAuditError("static_projection", baziContractAuditFinding{Code: tt.code})
			failure, ok := repairFailureFromBaziContract("static_projection", err)
			if !ok {
				t.Fatal("repairFailureFromBaziContract returned false")
			}
			if failure.Class != tt.wantClass {
				t.Fatalf("failure.Class = %q, want %q", failure.Class, tt.wantClass)
			}
			decision := DefaultRepairPolicy().Decide(failure, NewRepairState())
			if decision.Action != RepairActionHardError || decision.Retryable || decision.Repairable {
				t.Fatalf("decision = %+v, want hard non-repairable", decision)
			}
		})
	}
}
