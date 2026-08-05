// Package runtime 分类八字综合合同失败，供 repair、恢复和 trace 投影使用。
//
// 独立审计拥有语义 finding，本文件只映射成确定性 runtime 动作；
// 不重写命理解读，也不引入命盘专项分支。
package runtime

import "strings"

const (
	baziContractFailureEvidenceOverclaim  = "evidence_overclaim"
	baziContractFailureDomainUnauthorized = "domain_unauthorized"
	baziContractFailureProjectionMismatch = "projection_mismatch"
	baziContractFailureFactConflict       = "fact_conflict"
	baziContractFailureMethodContract     = "method_contract"
	baziContractFailureUnknown            = "unknown"

	baziRecoveryPolicyRetryOnly        = "retry_only"
	baziRecoveryPolicyStaticFactsOnly  = "static_facts_only"
	baziRecoveryPolicyDynamicFactsOnly = "dynamic_facts_only"
	baziRecoveryPolicyFullFactsOnly    = "full_facts_only"
	baziRecoveryPolicyHardError        = "hard_error"
)

// baziContractFailure is the internal, machine-readable failure classification
// shared by feedback, recovery and trace projection.
type baziContractFailure struct {
	Class          string
	FindingCode    string
	Field          string
	DetectedDomain string
	Excerpt        string
	Reason         string
	RecoveryPolicy string
}

// baziContractFailureFromAuditFinding 把审计 finding 映射到闭合失败分类。
// recovery policy 只由 runtime 决定，不信任模型自报建议。
func baziContractFailureFromAuditFinding(stage string, finding baziContractAuditFinding) baziContractFailure {
	code := strings.TrimSpace(finding.Code)
	failure := baziContractFailure{
		Class:          baziContractFailureUnknown,
		FindingCode:    code,
		Field:          strings.TrimSpace(finding.Field),
		DetectedDomain: strings.TrimSpace(finding.DetectedDomain),
		Excerpt:        strings.TrimSpace(finding.Excerpt),
		Reason:         strings.TrimSpace(finding.Reason),
	}
	switch code {
	case "evidence_topic_overclaim":
		failure.Class = baziContractFailureEvidenceOverclaim
	case "outcome_domain_mismatch", "age_scope":
		failure.Class = baziContractFailureDomainUnauthorized
	case "static_projection_mismatch":
		failure.Class = baziContractFailureProjectionMismatch
	case "undeclared_relation", "branch_tengod_conflict":
		failure.Class = baziContractFailureFactConflict
	case "month_command_single_rejection", "hidden_axis_uncompared", "unauthorized_rule_claim":
		failure.Class = baziContractFailureMethodContract
	}
	failure.RecoveryPolicy = baziRecoveryPolicyForFailure(stage, failure.Class)
	return withBaziStaticFallback(stage, failure)
}

// baziContractFailureFromViolation 分类确定性 validator 错误。
// 若错误来自审计 finding，则保留 finding metadata 供 repair 和 trace 使用。
func baziContractFailureFromViolation(stage string, violation baziValidationViolation) baziContractFailure {
	failure := baziContractFailure{
		Class:          baziContractFailureUnknown,
		FindingCode:    strings.TrimSpace(violation.ContractFindingCode),
		Field:          strings.TrimSpace(violation.Field),
		DetectedDomain: strings.TrimSpace(violation.DetectedDomain),
		Excerpt:        strings.TrimSpace(violation.Excerpt),
		Reason:         strings.TrimSpace(violation.Message),
	}
	if failure.FindingCode != "" {
		fromFinding := baziContractFailureFromAuditFinding(stage, baziContractAuditFinding{
			Code:           failure.FindingCode,
			Field:          failure.Field,
			Excerpt:        failure.Excerpt,
			DetectedDomain: failure.DetectedDomain,
			Reason:         failure.Reason,
		})
		fromFinding.Reason = failure.Reason
		return fromFinding
	}
	switch violation.Code {
	case baziViolationEvidenceTopicMissing:
		failure.Class = baziContractFailureEvidenceOverclaim
	case baziViolationUnsupportedConcreteOutcome:
		failure.Class = baziContractFailureDomainUnauthorized
	case baziViolationFactConflict, baziViolationFactRefMissing:
		failure.Class = baziContractFailureFactConflict
	case baziViolationMethodContract, baziViolationClaimNotAuthorized:
		failure.Class = baziContractFailureMethodContract
	case baziViolationScopeEscalation, baziViolationDayunCoverageMissing, baziViolationSemanticContract:
		failure.Class = baziContractFailureProjectionMismatch
	}
	failure.RecoveryPolicy = baziRecoveryPolicyForFailure(stage, failure.Class)
	return withBaziStaticFallback(stage, failure)
}

// baziContractFailureFromError extracts the classification from validation
// errors; non-contract errors intentionally remain unclassified.
func baziContractFailureFromError(stage string, err error) (baziContractFailure, bool) {
	violation, ok := baziViolationFromError(err)
	if !ok {
		return baziContractFailure{}, false
	}
	return baziContractFailureFromViolation(stage, violation), true
}

// repairFailureFromBaziContract maps BaZi-only validation failures into the
// global Repair Harness contract without changing current control flow.
func repairFailureFromBaziContract(stage string, err error) (RepairFailure, bool) {
	failure, ok := baziContractFailureFromError(stage, err)
	if !ok {
		return RepairFailure{}, false
	}
	repairFailure := RepairFailure{
		Domain:   "bazi",
		Stage:    strings.TrimSpace(stage),
		Class:    repairClassFromBaziContract(failure.Class),
		Field:    strings.TrimSpace(failure.Field),
		Code:     strings.TrimSpace(failure.FindingCode),
		Message:  strings.TrimSpace(failure.Reason),
		Excerpt:  strings.TrimSpace(failure.Excerpt),
		Fallback: repairFallbackFromBaziRecoveryPolicy(failure.RecoveryPolicy),
		Cause:    err,
	}
	decision := DefaultRepairPolicy().Decide(repairFailure, RepairState{})
	repairFailure.Retryable = decision.Retryable
	repairFailure.Repairable = decision.Repairable
	return repairFailure, true
}

// baziRecoveryPolicyForFailure 是合同失败恢复策略的基础映射。
// retry-only 类默认不能静默 facts-only；字段级例外由后续 helper 显式收窄。
func baziRecoveryPolicyForFailure(stage, class string) string {
	stage = strings.TrimSpace(stage)
	switch class {
	case baziContractFailureEvidenceOverclaim:
		if strings.HasPrefix(stage, "static") {
			return baziRecoveryPolicyStaticFactsOnly
		}
		return baziRecoveryPolicyHardError
	case baziContractFailureDomainUnauthorized:
		if strings.HasPrefix(stage, "dynamic") {
			return baziRecoveryPolicyDynamicFactsOnly
		}
		return baziRecoveryPolicyHardError
	case baziContractFailureProjectionMismatch:
		return baziRecoveryPolicyRetryOnly
	case baziContractFailureFactConflict, baziContractFailureMethodContract:
		return baziRecoveryPolicyHardError
	default:
		return baziRecoveryPolicyHardError
	}
}

// withBaziStaticFallback 收窄允许 static facts-only 的静态投影失败字段。
// 普通 fact_conflict 和 method_contract 仍不放宽，避免模型或降级吞掉确定性错误。
func withBaziStaticFallback(stage string, failure baziContractFailure) baziContractFailure {
	if failure.Class == baziContractFailureFactConflict &&
		strings.HasPrefix(strings.TrimSpace(stage), "static") &&
		strings.TrimSpace(failure.Field) == "static.strength_balance" {
		failure.RecoveryPolicy = baziRecoveryPolicyStaticFactsOnly
		return failure
	}
	if failure.Class != baziContractFailureProjectionMismatch {
		return failure
	}
	if !strings.HasPrefix(strings.TrimSpace(stage), "static") {
		return failure
	}
	if strings.TrimSpace(failure.Field) != "static.tiaohou_anchor" {
		return failure
	}
	failure.RecoveryPolicy = baziRecoveryPolicyStaticFactsOnly
	return failure
}

// baziTraceAttrsForContractFailure projects compact failure details into traces
// without retaining full candidate text.
func baziTraceAttrsForContractFailure(stage string, err error) map[string]any {
	failure, ok := baziContractFailureFromError(stage, err)
	if !ok {
		return nil
	}
	attrs := map[string]any{
		"bazi.contract.failure_class":   failure.Class,
		"bazi.contract.recovery_policy": failure.RecoveryPolicy,
	}
	if failure.FindingCode != "" {
		attrs["bazi.contract.finding_code"] = failure.FindingCode
	}
	if failure.Field != "" {
		attrs["bazi.contract.finding_field"] = failure.Field
	}
	if failure.DetectedDomain != "" {
		attrs["bazi.contract.detected_domain"] = failure.DetectedDomain
	}
	if repairFailure, ok := repairFailureFromBaziContract(stage, err); ok {
		decision := DefaultRepairPolicy().Decide(repairFailure, RepairState{})
		for key, value := range RepairTraceAttrs(RepairTraceEvent{
			Failure:     repairFailure,
			Attempt:     0,
			MaxAttempts: decision.MaxAttempts,
			Action:      decision.Action,
			Exhausted:   decision.Exhausted,
			FinalAction: decision.Action,
		}) {
			attrs[key] = value
		}
	}
	return attrs
}

// repairClassFromBaziContract bridges the BaZi-local closed taxonomy into the
// global Repair Harness taxonomy.
func repairClassFromBaziContract(class string) RepairClass {
	switch class {
	case baziContractFailureEvidenceOverclaim:
		return RepairEvidenceOverclaim
	case baziContractFailureDomainUnauthorized:
		return RepairDomainUnauthorized
	case baziContractFailureProjectionMismatch:
		return RepairProjectionMismatch
	case baziContractFailureFactConflict:
		return RepairFactConflict
	case baziContractFailureMethodContract:
		return RepairMethodContract
	default:
		return RepairUnknown
	}
}

// repairFallbackFromBaziRecoveryPolicy preserves existing BaZi recovery policy
// as a global fallback marker without running fallback here.
func repairFallbackFromBaziRecoveryPolicy(policy string) string {
	switch policy {
	case baziRecoveryPolicyStaticFactsOnly, baziRecoveryPolicyDynamicFactsOnly, baziRecoveryPolicyFullFactsOnly:
		return "facts_only"
	default:
		return ""
	}
}

// isBaziInnerAgentParseError identifies malformed JSON transport from the
// bounded synthesis model. These errors are retryable once but not recoverable
// as facts-only because no trustworthy candidate object exists.
func isBaziInnerAgentParseError(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "parse inner agent") || strings.Contains(text, "unexpected end of JSON input")
}

// buildBaziJSONRetryFeedback gives the model a narrow correction after it
// emitted truncated or malformed JSON.
func buildBaziJSONRetryFeedback(stage string, err error) string {
	return strings.Join([]string{
		stage + "上次输出不是完整合法 JSON，本次只重写为一个完整 JSON 对象。",
		"不得输出 markdown 代码块、解释文字或截断字段；所有字符串必须闭合，数组和对象必须闭合。",
		"保持原任务合同不变，只修复 JSON 完整性。",
		"上次解析失败原因：" + err.Error(),
	}, "\n")
}
