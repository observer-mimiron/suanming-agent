// Package runtime classifies BaZi synthesis contract failures for retry,
// recovery and trace reporting.
//
// The independent audit owns semantic findings, while this file maps those
// findings into deterministic runtime actions. It must not rewrite readings or
// introduce chart-specific case handling.
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

// baziContractFailureFromAuditFinding maps one audit finding into the runtime's
// closed failure taxonomy without trusting model-authored recovery suggestions.
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
	return failure
}

// baziContractFailureFromViolation classifies deterministic validation errors
// and preserves contract-audit metadata when the error originated in the judge.
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
	case baziViolationScopeEscalation, baziViolationSemanticContract:
		failure.Class = baziContractFailureProjectionMismatch
	}
	failure.RecoveryPolicy = baziRecoveryPolicyForFailure(stage, failure.Class)
	return failure
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

// baziRecoveryPolicyForFailure is the single source for model-text recovery
// decisions after retry. Retry-only classes must not silently become facts-only.
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
	return attrs
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
