package runtime

import (
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/repair"
)

func TestRepairPolicyEnforcesBusinessBudget(t *testing.T) {
	failure := repair.Failure{Domain: "bazi", Stage: "static_projection", Class: repair.ProjectionMismatch, Field: "static.tiaohou_anchor", Fallback: "facts_only"}
	policy := repair.DefaultPolicy()
	if decision := policy.Decide(failure, repair.NewState()); decision.Action != repair.ActionRepairNode {
		t.Fatalf("first action = %q, want repair_node", decision.Action)
	}
	state := repair.RecordAttempt(repair.NewState(), repair.Attempt{Domain: failure.Domain, Stage: failure.Stage, Class: failure.Class, Field: failure.Field, Action: repair.ActionRepairNode})
	if decision := policy.Decide(failure, state); decision.Action != repair.ActionFallback || !decision.Exhausted {
		t.Fatalf("exhausted decision = %+v, want fallback", decision)
	}
}

func TestRepairPolicyEnforcesWholeTurnBudget(t *testing.T) {
	state := repair.NewState()
	for _, attempt := range []repair.Attempt{
		{Domain: "bazi", Stage: "static", Field: "axis", Class: repair.ProjectionMismatch, Action: repair.ActionRepairNode},
		{Domain: "bazi", Stage: "dynamic", Field: "trend", Class: repair.ProjectionMismatch, Action: repair.ActionRepairNode},
	} {
		state = repair.RecordAttempt(state, attempt)
	}
	decision := repair.DefaultPolicy().Decide(repair.Failure{Domain: "bazi", Stage: "final_guard", Class: repair.ProjectionMismatch, Field: "answer", Fallback: "facts_only"}, state)
	if decision.Action != repair.ActionFallback || !decision.Exhausted {
		t.Fatalf("whole-turn decision = %+v, want exhausted fallback", decision)
	}
}

func TestRepairPolicyRejectsFactConflictAndMethodContract(t *testing.T) {
	for _, class := range []repair.Class{repair.FactConflict, repair.MethodContract} {
		decision := repair.DefaultPolicy().Decide(repair.Failure{Class: class}, repair.NewState())
		if decision.Action != repair.ActionHardError || decision.Repairable || decision.Retryable {
			t.Fatalf("%s decision = %+v, want hard error", class, decision)
		}
	}
}

func TestRepairHTTPStatusRetryable(t *testing.T) {
	for _, status := range []int{400, 401, 402} {
		if repair.HTTPStatusRetryable(status) || repair.ClassForHTTPStatus(status) != repair.TransportFatal {
			t.Fatalf("status %d should be fatal and non-retryable", status)
		}
	}
	for _, status := range []int{408, 429, 500, 503} {
		if !repair.HTTPStatusRetryable(status) || repair.ClassForHTTPStatus(status) != repair.TransportTransient {
			t.Fatalf("status %d should be transient and retryable", status)
		}
	}
}

func TestRepairTraceAttrsProjectsOnlySafeFields(t *testing.T) {
	attrs := RepairTraceAttrs(RepairTraceEvent{Failure: repair.Failure{Domain: "bazi", Stage: "static", Class: repair.ProjectionMismatch, Field: "axis"}, Feedback: map[string]any{"reason": "private", "allowed_fix": []string{"x"}}})
	if _, leaked := attrs["reason"]; leaked {
		t.Fatal("trace attrs leaked feedback value")
	}
	keys, ok := attrs["repair.feedback_keys"].([]string)
	if !ok || len(keys) != 2 || keys[0] != "allowed_fix" || keys[1] != "reason" {
		t.Fatalf("feedback keys = %#v", attrs["repair.feedback_keys"])
	}
}
