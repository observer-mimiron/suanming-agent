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
		{name: "transport transient", class: TransportTransient, wantAction: ActionRetry, wantRetryable: true},
		{name: "schema repair", class: SchemaError, wantAction: ActionRepairNode, wantRetryable: true},
		{name: "fact conflict", class: FactConflict, wantAction: ActionHardError},
		{name: "unauthorized fallback", class: DomainUnauthorized, fallback: "facts_only", wantAction: ActionFallback},
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
	state = RecordAttempt(state, Attempt{
		Domain: "bazi", Stage: "static", Field: "main_axis", Action: ActionRetry,
	})

	if got := AttemptsFor(state, failure); got != 1 {
		t.Fatalf("AttemptsFor() = %d, want 1", got)
	}
	if !BudgetExhausted(state, failure) {
		t.Fatal("same stage/field repair budget should be exhausted")
	}

	state = RecordAttempt(state, Attempt{
		Domain: "bazi", Stage: "dynamic", Field: "current_period", Action: ActionRepairNode,
	})
	if !BudgetExhausted(state, Failure{Domain: "bazi", Stage: "evidence", Field: "topics"}) {
		t.Fatal("whole-turn repair budget should be exhausted after two repairs")
	}
}

func TestHTTPStatusRetryableOnlyForTransientFailures(t *testing.T) {
	for _, test := range []struct {
		status int
		want   bool
	}{
		{status: 408, want: true},
		{status: 429, want: true},
		{status: 500, want: true},
		{status: 400, want: false},
		{status: 401, want: false},
		{status: 404, want: false},
	} {
		if got := HTTPStatusRetryable(test.status); got != test.want {
			t.Fatalf("HTTPStatusRetryable(%d) = %t, want %t", test.status, got, test.want)
		}
	}
}
