// Package runtime 包含 Manager 拥有的八字 Graph 入口。
//
// 本文件负责选择八字内图、归一化领域失败并保留外层调用合同；
// 不负责节点实现、证据检索、合同细节、trace 审计或最终文本渲染。
package runtime

import (
	"context"

	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// shouldUseBaziCharterGraph reports whether the BaZi primary step must use the
// structured inner graph. Support domains do not change this decision: letting
// a mixed turn fall back to free-form BaZi output would bypass the same
// validated renderer used by a BaZi-only turn.
func shouldUseBaziCharterGraph(plan ExecutionPlan) bool {
	if plan.Route.PrimaryDomain != "bazi" {
		return false
	}
	switch plan.FollowupMode {
	case "", followupModeRerunSpecialist:
		// 非 follow-up，或 manager 已明确要求继续走 specialist 主链，才允许进入八字 inner graph。
	default:
		return false
	}
	return true
}

// shouldUseBaziAuthorityGraph keeps mixed-domain dispatch on the structured
// BaZi path only when the registered runner explicitly owns that capability.
// Test doubles and legacy runners must retain their own dispatch contract.
func (e *Executor) shouldUseBaziAuthorityGraph(plan ExecutionPlan) bool {
	if shouldUseBaziCharterGraph(plan) && len(plan.Domains) == 1 {
		return true
	}
	if e == nil || e.specialistRegistry == nil || plan.Route.PrimaryDomain != "bazi" {
		return false
	}
	runner, ok := e.specialistRegistry.RunnerFor("bazi")
	if !ok {
		return false
	}
	authorityRunner, ok := runner.(*ADKSpecialistRunner)
	return ok && authorityRunner.UseAuthorityGraph
}

// runBaziAuthorityFirstGraph keeps the outer orchestration contract unchanged
// while delegating BaZi-specific control flow to the internal graph.
func (e *Executor) runBaziAuthorityFirstGraph(ctx context.Context, sink EventSink, st *state.SessionState, question string) (string, error) {
	return e.runBaziInternalGraph(ctx, sink, st, question)
}

// baziSynthesisRuntimeFailure 将领域合同失败转换为外层 RuntimeFailure。
func baziSynthesisRuntimeFailure(stage, code string, cause error) error {
	return &RuntimeFailure{
		Class:       failureClassModelContractViolation,
		Stage:       stage,
		Domain:      "bazi",
		Code:        code,
		Retryable:   true,
		Degraded:    false,
		UserVisible: true,
		Message:     baziSynthesisFailureMessage(stage, cause),
		Cause:       cause,
	}
}

// baziSynthesisFailureMessage keeps user-visible errors aligned with the actual
// contract failure class without leaking candidate text.
func baziSynthesisFailureMessage(stage string, cause error) string {
	if failure, ok := baziContractFailureFromError(stage, cause); ok {
		switch failure.Class {
		case baziContractFailureEvidenceOverclaim:
			return "证据主题不足，已停止展示过度裁断。请稍后重试。"
		case baziContractFailureDomainUnauthorized:
			return "岁运综合越过授权领域，已停止展示不稳定内容。请稍后重试。"
		case baziContractFailureFactConflict, baziContractFailureMethodContract:
			return "候选推演与事实或方法合同冲突，已停止展示不稳定内容。请稍后重试。"
		}
	}
	return "本轮八字综合未通过结构化合同校验，已停止展示不稳定内容。请稍后重试。"
}
