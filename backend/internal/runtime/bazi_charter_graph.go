// Package runtime 包含 Manager 拥有的八字 Graph 运行支持。
//
// 本文件负责补证、审计、阶段事件和最终 writer 适配；
// 不负责 Graph 拓扑、入口选择、确定性输入视图或合同校验。
package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// annotateBaziSynthesisSources writes compact provenance for trace inspection.
// It records accepted typed states only and never promotes them into a new judgment.
func annotateBaziSynthesisSources(ctx context.Context, state baziCharterState) {
	capsule := buildBaziFactCapsule(state)
	tier := state.StaticSynthesis.TierAssessment
	outputMode := "model_full"
	switch {
	case isFactsOnlyStaticSynthesis(state.StaticSynthesis):
		outputMode = baziSynthesisSourceFactsOnlyDegraded
	case isFactsOnlyDynamicSynthesis(state.DynamicSynthesis):
		outputMode = "model_static_dynamic_degraded"
	}
	attrs := map[string]any{
		"bazi.facts.source":                       "deterministic_tools",
		"bazi.static.source":                      firstNonEmptyTrim(state.StaticSynthesis.Source, "unknown"),
		"bazi.static.error":                       state.StaticSynthesis.RecoveryReason,
		"bazi.static.degraded_reason":             degradedReasonForSource(state.StaticSynthesis.Source, state.StaticSynthesis.RecoveryReason),
		"bazi.static.recovery_reason":             state.StaticSynthesis.RecoveryReason,
		"bazi.static.assertion_count":             len(state.StaticSynthesis.Assertions),
		"bazi.static.contract_audit":              baziContractAuditSummary(state.StaticSynthesis.ContractAudit),
		"bazi.tier.source":                        firstNonEmptyTrim(state.StaticSynthesis.Source, "unknown"),
		"bazi.tier.status":                        firstNonEmptyTrim(tier.Status, "legacy"),
		"bazi.tier.level":                         tier.Level,
		"bazi.tier.confidence":                    tier.Confidence,
		"bazi.tier.evidence_complete":             capsule.TierEvidenceComplete,
		"bazi.tier.evidence_missing":              strings.Join(capsule.TierEvidenceMissing, ","),
		"bazi.tiaohou.coverage":                   baziTiaohouCoverage(state.EvidenceQuality),
		"bazi.dynamic.source":                     firstNonEmptyTrim(state.DynamicSynthesis.Source, "unknown"),
		"bazi.dynamic.error":                      state.DynamicSynthesis.RecoveryReason,
		"bazi.dynamic.degraded_reason":            degradedReasonForSource(state.DynamicSynthesis.Source, state.DynamicSynthesis.RecoveryReason),
		"bazi.dynamic.recovery_reason":            state.DynamicSynthesis.RecoveryReason,
		"bazi.dynamic.assertion_count":            len(state.DynamicSynthesis.Assertions),
		"bazi.dynamic.contract_audit":             baziContractAuditSummary(state.DynamicSynthesis.ContractAudit),
		"bazi.dynamic.current_period_ref":         capsule.CurrentPeriodRef,
		"bazi.dynamic.current_period_ganzhi":      capsule.CurrentPeriodGanZhi,
		"bazi.dynamic.current_period_realization": state.DynamicSynthesis.CurrentPeriodRealization,
		"bazi.dayun.count":                        len(state.DynamicSynthesis.DayunPath),
		"bazi.final.output_mode":                  outputMode,
		"bazi.final.audit_result":                 baziFieldAuditResult(state.FieldAudit),
	}
	if class := firstFieldAuditValue(state.FieldAudit, "contract_failure_class:"); class != "" {
		attrs["bazi.contract.failure_class"] = class
	}
	if policy := firstFieldAuditValue(state.FieldAudit, "recovery_policy:"); policy != "" {
		attrs["bazi.contract.recovery_policy"] = policy
	}
	tracing.SetTraceAttributes(ctx, attrs)
}

// firstFieldAuditValue returns the first compact key-value note stored by a
// recovery path for trace projection.
func firstFieldAuditValue(notes []string, prefix string) string {
	for _, note := range notes {
		if strings.HasPrefix(note, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(note, prefix))
		}
	}
	return ""
}

// degradedReasonForSource 仅为 facts-only 来源保留可观测的降级原因。
func degradedReasonForSource(source, reason string) string {
	if strings.TrimSpace(source) != baziSynthesisSourceFactsOnlyDegraded {
		return ""
	}
	return reason
}

// baziFieldAuditResult 将恢复路径写入的审计备注压缩为 trace 状态。
func baziFieldAuditResult(notes []string) string {
	repairs := make([]string, 0, len(notes))
	seen := make(map[string]struct{}, len(notes))
	for _, note := range notes {
		note = strings.TrimSpace(note)
		if note == "" || note == "canonical_tier_withheld_by_runtime" || note == "canonical_dynamic_projection_facts_only" {
			continue
		}
		if _, exists := seen[note]; exists {
			continue
		}
		seen[note] = struct{}{}
		repairs = append(repairs, note)
	}
	if len(repairs) == 0 {
		return "clean"
	}
	return "repaired: " + strings.Join(repairs, ", ")
}

// annotateBaziSoftAudit records reviewable wording risks without mutating the
// user-facing report. These concerns depend on rule-profile scope and human
// calibration, so they are unsuitable as a runtime hard gate.
func annotateBaziSoftAudit(ctx context.Context, state baziCharterState) {
	warnings := collectBaziSoftAuditWarnings(state)
	if len(warnings) == 0 {
		return
	}
	tracing.SetTraceAttributes(ctx, map[string]any{
		"bazi.graph.soft_audit_warning_count": len(warnings),
		"bazi.graph.soft_audit_warnings":      strings.Join(warnings, " | "),
	})
}

// collectBaziSoftAuditWarnings gathers non-blocking wording and reference warnings for trace review.
func collectBaziSoftAuditWarnings(state baziCharterState) []string {
	staticText := strings.Join([]string{
		state.StaticSynthesis.MainAxis,
		state.StaticSynthesis.PatternOutcome,
		state.StaticSynthesis.TierJudgment,
		state.StaticSynthesis.TierBasis,
	}, "\n")
	dynamicText := strings.Join([]string{
		state.DynamicSynthesis.CurrentTrend,
		strings.Join(state.DynamicSynthesis.DayunPath, "\n"),
		state.DynamicSynthesis.LiunianFocus,
	}, "\n")
	warnings := []string{}
	knownFacts := knownBaziFactRefs(state)
	knownClaims := knownBaziClaimRefs(state.Input.RuleProfile)
	unknownFactRefs := []string{}
	unknownClaimRefs := []string{}
	for _, assertion := range append(append([]baziAssertion{}, state.StaticSynthesis.Assertions...), state.DynamicSynthesis.Assertions...) {
		for _, ref := range assertion.FactRefs {
			if !isKnownBaziFactRef(ref, knownFacts) {
				unknownFactRefs = append(unknownFactRefs, string(ref))
			}
		}
		for _, ref := range assertion.ClaimRefs {
			if _, ok := knownClaims[string(ref)]; !ok {
				unknownClaimRefs = append(unknownClaimRefs, string(ref))
			} else if !claimRefAllowsAssertionKind(state.Input.RuleProfile, string(ref), assertion.Kind) {
				warnings = append(warnings, "assertion uses a known claim outside its suggested kind: "+assertion.ID+" -> "+string(ref))
			}
		}
	}
	if len(unknownFactRefs) > 0 {
		warnings = append(warnings, "assertions use unknown fact-ref aliases: "+strings.Join(uniqueText(unknownFactRefs), ", "))
	}
	if len(unknownClaimRefs) > 0 {
		warnings = append(warnings, "assertions use unknown claim refs: "+strings.Join(uniqueText(unknownClaimRefs), ", "))
	}
	if containsAnyText([]string{staticText}, []string{"贵格", "富格", "伤官佩印格成", "伤官佩印成立"}) {
		warnings = append(warnings, "static wording uses a strong pattern or status conclusion; review profile evidence")
	}
	if containsAnyText([]string{dynamicText}, []string{"大吉", "大凶", "黄金窗口", "职位提升", "贵人赏识", "婚姻", "财运", "健康"}) {
		warnings = append(warnings, "dynamic wording includes a broad tendency or concrete domain outcome; review evidence before expanding the rule profile")
	}
	return warnings
}

// annotateBaziGraphError 把节点错误映射为可检索的阶段和合同 trace 属性。
func annotateBaziGraphError(ctx context.Context, stage string, err error) {
	if err == nil {
		return
	}
	attrs := map[string]any{
		"bazi.graph.error_stage": stage,
		"bazi.graph.error":       err.Error(),
	}
	switch {
	case strings.HasPrefix(stage, "static"):
		attrs["bazi.static.error_stage"] = stage
		attrs["bazi.static.error"] = err.Error()
	case strings.HasPrefix(stage, "dynamic"):
		attrs["bazi.dynamic.error_stage"] = stage
		attrs["bazi.dynamic.error"] = err.Error()
	case strings.HasPrefix(stage, "evidence"):
		attrs["bazi.evidence.error_stage"] = stage
		attrs["bazi.evidence.error"] = err.Error()
	}
	for key, value := range baziTraceAttrsForContractFailure(stage, err) {
		attrs[key] = value
	}
	tracing.SetTraceAttributes(ctx, attrs)
}

// supplementDynamicEvidenceIfNeeded 在首轮完整看盘这类“静态主判 + 动态补证”场景中，
// 追加一次动态证据规划与检索，避免 dynamic_synthesis 只拿到静态格局证据。
func (e *Executor) supplementDynamicEvidenceIfNeeded(ctx context.Context, st *state.SessionState, question string, chartState baziCharterState) (baziCharterState, error) {
	if !chartState.AnalysisPlan.NeedDynamic {
		return chartState, nil
	}
	if chartState.AnalysisPlan.RetrievalStage == "dynamic" {
		return chartState, nil
	}
	// 首轮完整看盘优先消费系统已就绪的 dayun_analyzed / liunian 字段，
	// 不再默认补第二轮动态检索。只有动态基础事实缺失时才补证。
	if hasDynamicSystemFacts(chartState.Input) {
		return chartState, nil
	}

	dynamicPlan := chartState.AnalysisPlan
	dynamicPlan.RetrievalStage = "dynamic"
	dynamicPlan.StageSummary = "已为大运验证与流年应期补充古籍依据。"

	plan, bundle, _, err := e.runBaziEvidenceStage(ctx, st, question, chartState.Input, dynamicPlan)
	if err != nil {
		return chartState, err
	}
	chartState.EvidencePlan = plan
	chartState.EvidenceBundle = mergeEvidenceBundles(chartState.EvidenceBundle, bundle)
	chartState.EvidenceQuality = evaluateEvidenceBundleQuality(plan, bundle)
	return chartState, nil
}

// emitBaziStageThinking sends one user-visible stage summary without exposing raw model reasoning.
func emitBaziStageThinking(ctx context.Context, sink EventSink, agent, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	_ = emitEventWithTrace(ctx, sink, Event{
		Type: "thinking",
		Data: map[string]any{
			"text":  text,
			"agent": agent,
		},
	}, map[string]any{
		"phase": "bazi_graph",
	})
}

// emitBaziReasoningSteps 只把产品化后的推演步骤发给前端，不暴露原始自由思维。
// 这些步骤来自上游结构化 synthesis 字段，属于可展示的“分析过程摘要”。
func emitBaziReasoningSteps(ctx context.Context, sink EventSink, label string, steps []string) {
	limit := 4
	count := 0
	for _, step := range steps {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}
		emitBaziStageThinking(ctx, sink, "bazi_graph", fmt.Sprintf("%s：%s", label, step))
		count++
		if count >= limit {
			return
		}
	}
}

// runFinalWriter renders accepted synthesis and rejects output that violates the final contract.
func (e *Executor) runFinalWriter(ctx context.Context, st *state.SessionState, chartState baziCharterState, question string) (string, error) {
	output := renderBaziFinalReply(chartState.AnalysisPlan, chartState, question)
	if err := validateFinalWriterOutput(chartState.AnalysisPlan, chartState, output); err != nil {
		tracing.SetTraceAttributes(ctx, map[string]any{
			"bazi.final_writer.template":       chartState.AnalysisPlan.WriterTemplate,
			"bazi.final_writer.validation_err": err.Error(),
			"bazi.final_writer.output_preview": truncateTracePreview(output, 1200),
		})
		return "", &RuntimeFailure{
			Class:       failureClassModelContractViolation,
			Stage:       failureStageFinalWriter,
			Domain:      "bazi",
			Code:        "FINAL_CONTRACT_VIOLATION",
			Retryable:   false,
			Degraded:    false,
			UserVisible: true,
			Message:     "本轮输出未通过最终合同校验，已停止展示不稳定内容。请稍后重试。",
			Cause:       err,
		}
	}
	return output, nil
}

// truncateTracePreview bounds trace previews without splitting UTF-8 characters.
func truncateTracePreview(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "...(truncated)"
}
