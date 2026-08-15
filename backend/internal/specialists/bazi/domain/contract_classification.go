// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责把领域合同 finding 和 violation 分类为恢复状态；
// 不解析模型输出，不持有 runtime 服务或输出传输。
package domain

import "strings"

// ClassifyContractFinding 将合同审计 finding 映射为固定失败分类。
func ClassifyContractFinding(stage string, finding ContractAuditFinding) ContractFailure {
	failure := ContractFailure{Class: "unknown", FindingCode: strings.TrimSpace(finding.Code), Field: strings.TrimSpace(finding.Field), DetectedDomain: strings.TrimSpace(finding.DetectedDomain), Excerpt: strings.TrimSpace(finding.Excerpt), Reason: strings.TrimSpace(finding.Reason)}
	switch failure.FindingCode {
	case "evidence_topic_overclaim":
		failure.Class = ContractFailureEvidenceOverclaim
	case "outcome_domain_mismatch", "age_scope":
		failure.Class = ContractFailureDomainUnauthorized
	case "static_projection_mismatch":
		failure.Class = ContractFailureProjectionMismatch
	case "undeclared_relation", "branch_tengod_conflict":
		failure.Class = ContractFailureFactConflict
	case "month_command_single_rejection", "hidden_axis_uncompared", "unauthorized_rule_claim":
		failure.Class = ContractFailureMethodContract
	}
	failure.RecoveryPolicy = RecoveryPolicyForFailure(stage, failure.Class)
	return WithStaticFallback(stage, failure)
}

// ClassifyViolation 将确定性领域校验错误映射为固定失败分类。
func ClassifyViolation(stage string, violation ValidationViolation) ContractFailure {
	failure := ContractFailure{Class: "unknown", FindingCode: strings.TrimSpace(violation.ContractFindingCode), Field: strings.TrimSpace(violation.Field), DetectedDomain: strings.TrimSpace(violation.DetectedDomain), Excerpt: strings.TrimSpace(violation.Excerpt), Reason: strings.TrimSpace(violation.Message), MissingRefs: append([]string(nil), violation.MissingRefs...), AllowedRefs: append([]string(nil), violation.AllowedRefs...)}
	if failure.FindingCode != "" {
		fromFinding := ClassifyContractFinding(stage, ContractAuditFinding{Code: failure.FindingCode, Field: failure.Field, Excerpt: failure.Excerpt, DetectedDomain: failure.DetectedDomain, Reason: failure.Reason})
		fromFinding.Reason = failure.Reason
		return fromFinding
	}
	switch violation.Code {
	case ViolationEvidenceTopicMissing:
		failure.Class = ContractFailureEvidenceOverclaim
	case ViolationUnsupportedConcreteOutcome:
		failure.Class = ContractFailureDomainUnauthorized
	case ViolationUndeclaredFactClaim:
		failure.Class = ContractFailureSchemaError
	case ViolationFactConflict, ViolationFactRefMissing:
		failure.Class = ContractFailureFactConflict
	case ViolationMethodContract, ViolationClaimNotAuthorized:
		if dynamicPresentationReferenceViolation(stage, violation) {
			failure.Class = ContractFailureProjectionMismatch
		} else {
			failure.Class = ContractFailureMethodContract
		}
	case ViolationScopeEscalation, ViolationDayunCoverageMissing, ViolationSemanticContract:
		failure.Class = ContractFailureProjectionMismatch
	}
	failure.RecoveryPolicy = RecoveryPolicyForFailure(stage, failure.Class)
	return WithStaticFallback(stage, failure)
}

// dynamicPresentationReferenceViolation 只允许动态用户可见文本的内部引用泄露降级。
func dynamicPresentationReferenceViolation(stage string, violation ValidationViolation) bool {
	if !strings.HasPrefix(strings.TrimSpace(stage), "dynamic") || (len(violation.MissingRefs) == 0 && len(violation.AllowedRefs) == 0) {
		return false
	}
	field := strings.TrimSpace(violation.Field)
	return strings.HasPrefix(field, "dynamic.") && (strings.Contains(field, "limitations") || strings.Contains(field, "reasoning") || strings.Contains(field, "period_claims") || strings.Contains(field, "liunian_claim"))
}
