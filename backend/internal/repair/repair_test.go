// Package repair 的测试保护共享 repair 合同的分类与预算边界。
package repair

import "testing"

func TestPolicyClassifiesFailureByAction(t *testing.T) {
	tests := []struct {
		name          string
		class         Class
		fallback      string
		wantAction    Action
		wantRetryable bool
	}{
		{name: "schema repair", class: SchemaError, wantAction: ActionRepairNode, wantRetryable: true},
		{name: "fact conflict", class: FactConflict, wantAction: ActionRepairNode, wantRetryable: true},
		{name: "method contract", class: MethodContract, wantAction: ActionRepairNode, wantRetryable: true},
		{name: "deterministic conflict", class: DeterministicConflict, wantAction: ActionHardError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision := DefaultPolicy().Decide(Failure{Class: test.class, Fallback: test.fallback}, NewState())
			if decision.Action != test.wantAction || decision.Retryable != test.wantRetryable {
				t.Fatalf("decision = %+v, want action=%q retryable=%t", decision, test.wantAction, test.wantRetryable)
			}
		})
	}
}

func TestBudgetCountsRepairAttemptsByTurnAndStage(t *testing.T) {
	failure := Failure{Domain: "bazi", Stage: "static", Field: "main_axis"}
	state := RecordAttempt(NewState(), Attempt{
		Domain: "bazi", Stage: "static", Field: "main_axis", Action: ActionRepairNode,
	})
	if got := AttemptsFor(state, failure); got != 1 {
		t.Fatalf("AttemptsFor() = %d, want 1", got)
	}
	if BudgetExhausted(state, failure) {
		t.Fatal("same stage/field repair budget should allow a second repair")
	}
	state = RecordAttempt(state, Attempt{
		Domain: "bazi", Stage: "static", Field: "main_axis", Action: ActionRepairNode,
	})
	if !BudgetExhausted(state, failure) {
		t.Fatal("same stage/field repair budget should be exhausted after two repairs")
	}

	state = RecordAttempt(state, Attempt{
		Domain: "bazi", Stage: "dynamic", Field: "current_period", Action: ActionRepairNode,
	})
	if !BudgetExhausted(state, Failure{Domain: "bazi", Stage: "evidence", Field: "topics"}) {
		t.Fatal("whole-turn repair budget should be exhausted after two repairs")
	}
}

func TestRecordFailurePreservesInitialAndUpdatesLast(t *testing.T) {
	state := RecordFailure(NewState(), Failure{Domain: "bazi", Stage: "static", Class: SchemaError, Field: "axis"})
	state = RecordFailure(state, Failure{Domain: "bazi", Stage: "dynamic", Class: FactConflict, Field: "period"})
	if state.InitialFailure.Stage != "static" || state.LastFailure.Stage != "dynamic" {
		t.Fatalf("snapshots = %#v / %#v", state.InitialFailure, state.LastFailure)
	}
}

func TestFailureSnapshotRebuildsOnlyStateSafeFields(t *testing.T) {
	failure := Failure{
		Domain: "bazi", Stage: "static", Class: FactConflict, Field: "axis", Code: "AXIS_CONFLICT",
		Origin: OriginModelCandidate, Fallback: "static_facts_only", Message: "rejected text", Excerpt: "candidate excerpt",
		MissingRefs: []string{"unexpected"}, AllowedRefs: []string{"known"},
	}

	rebuilt := failure.Snapshot().Failure()
	if rebuilt.Domain != failure.Domain || rebuilt.Class != failure.Class || rebuilt.Fallback != failure.Fallback {
		t.Fatalf("rebuilt failure = %#v", rebuilt)
	}
	if rebuilt.Message != "" || rebuilt.Excerpt != "" || len(rebuilt.MissingRefs) != 0 || len(rebuilt.AllowedRefs) != 0 || rebuilt.Cause != nil {
		t.Fatalf("snapshot leaked non-state fields: %#v", rebuilt)
	}
}

func TestTraceAttrsDoesNotExposeFeedbackValues(t *testing.T) {
	attrs := TraceAttrs(TraceEvent{
		Failure:      Failure{Domain: "bazi", Stage: "static", Class: ProjectionMismatch, Field: "axis", Origin: OriginModelCandidate},
		FeedbackKeys: []string{"allowed_fix", "reason"},
	})
	if _, leaked := attrs["reason"]; leaked {
		t.Fatal("trace attrs leaked feedback value")
	}
	if got := attrs["repair.failure_origin"]; got != string(OriginModelCandidate) {
		t.Fatalf("repair.failure_origin=%v", got)
	}
}
