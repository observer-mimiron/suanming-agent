// Package runtime 包含 Manager 拥有的八字合同测试。
//
// 本文件保护动态模型文本越界与真正方法合同错误的恢复边界。
package runtime

import "testing"

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
}
