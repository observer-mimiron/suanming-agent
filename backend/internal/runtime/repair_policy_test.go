package runtime

import (
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/llm"
	"github.com/observer-mimiron/suanming-agent/internal/repair"
)

func TestRepairPolicyEnforcesBusinessBudget(t *testing.T) {
	failure := repair.Failure{Domain: "bazi", Stage: "static_projection", Class: repair.ProjectionMismatch, Field: "static.tiaohou_anchor", Fallback: "facts_only"}
	policy := repair.DefaultPolicy()
	if decision := policy.Decide(failure, repair.NewState()); decision.Action != repair.ActionRepairNode {
		t.Fatalf("first action = %q, want repair_node", decision.Action)
	}
	state := repair.RecordAttempt(repair.NewState(), repair.Attempt{Domain: failure.Domain, Stage: failure.Stage, Class: failure.Class, Field: failure.Field, Action: repair.ActionRepairNode})
	if decision := policy.Decide(failure, state); decision.Action != repair.ActionRepairNode || decision.Exhausted {
		t.Fatalf("second attempt = %+v, want another repair", decision)
	}
	state = repair.RecordAttempt(state, repair.Attempt{Domain: failure.Domain, Stage: failure.Stage, Class: failure.Class, Field: failure.Field, Action: repair.ActionRepairNode})
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

func TestRepairHTTPStatusRetryable(t *testing.T) {
	for _, status := range []int{400, 401, 402} {
		if llm.HTTPStatusRetryable(status) {
			t.Fatalf("status %d should be fatal and non-retryable", status)
		}
	}
	for _, status := range []int{408, 429, 500, 503} {
		if !llm.HTTPStatusRetryable(status) {
			t.Fatalf("status %d should be transient and retryable", status)
		}
	}
}
