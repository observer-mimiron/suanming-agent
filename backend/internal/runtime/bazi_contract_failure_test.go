// Package runtime 包含 Manager 拥有的八字合同测试。
//
// 本文件保护动态模型文本越界与真正方法合同错误的恢复边界。
package runtime

import (
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/repair"
)

func TestDynamicPresentationReferenceViolationFallsBackToFactsOnly(t *testing.T) {
	err := baziViolationError(
		baziViolationMethodContract,
		"dynamic.limitations[0]",
		"",
		"model prose must keep runtime identifiers in typed reference arrays",
		[]string{"chart.season"},
		[]string{"fact_refs", "relation_refs", "claim_refs"},
	)
	failure, ok := baziContractFailureFromError("dynamic_projection", err)
	if !ok {
		t.Fatal("expected contract failure")
	}
	if failure.Class != baziContractFailureProjectionMismatch || failure.RecoveryPolicy != baziRecoveryPolicyDynamicFactsOnly {
		t.Fatalf("failure = %+v", failure)
	}
	repairFailure, ok := repairFailureFromBaziContract("dynamic_projection", err)
	if !ok {
		t.Fatal("expected global repair failure")
	}
	if repairFailure.Class != repair.ProjectionMismatch || repairFailure.Fallback != "facts_only" || !repairFailure.Repairable || !repairFailure.Retryable {
		t.Fatalf("repair failure = %+v", repairFailure)
	}
	policy := repair.DefaultPolicy()
	decision := policy.Decide(repairFailure, repair.NewState())
	if decision.Action != repair.ActionRepairNode {
		t.Fatalf("initial decision = %+v, want repair_node", decision)
	}
	state := repair.RecordAttempt(repair.NewState(), repair.Attempt{
		Domain: repairFailure.Domain,
		Stage:  repairFailure.Stage,
		Class:  repairFailure.Class,
		Field:  repairFailure.Field,
		Action: repair.ActionRepairNode,
	})
	decision = policy.Decide(repairFailure, state)
	if decision.Action != repair.ActionFallback || !decision.Exhausted {
		t.Fatalf("exhausted decision = %+v, want facts-only fallback", decision)
	}
}

func TestDynamicPeriodBindingMethodContractRemainsHardError(t *testing.T) {
	err := baziViolationError(
		baziViolationMethodContract,
		"dynamic.current_period_ref",
		"",
		"runtime cannot bind a current dayun",
		nil,
		nil,
	)
	failure, ok := baziContractFailureFromError("dynamic_projection", err)
	if !ok {
		t.Fatal("expected contract failure")
	}
	if failure.Class != baziContractFailureMethodContract || failure.RecoveryPolicy != baziRecoveryPolicyHardError {
		t.Fatalf("failure = %+v", failure)
	}
	repairFailure, ok := repairFailureFromBaziContract("dynamic_projection", err)
	if !ok {
		t.Fatal("expected global repair failure")
	}
	if repairFailure.Class != repair.MethodContract || repairFailure.Fallback != "" || repairFailure.Repairable || repairFailure.Retryable {
		t.Fatalf("repair failure = %+v", repairFailure)
	}
	decision := repair.DefaultPolicy().Decide(repairFailure, repair.NewState())
	if decision.Action != repair.ActionHardError || decision.Exhausted {
		t.Fatalf("decision = %+v, want non-repairable hard_error", decision)
	}
}
