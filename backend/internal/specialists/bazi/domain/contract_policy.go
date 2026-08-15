// Package graph 包含八字领域拥有的有界执行图。
//
// 本文件负责合同失败后的确定性恢复策略；
// 不解析模型输出，不持有 runtime 服务或输出传输。
package domain

import "strings"

const (
	RecoveryStateClean                    = "clean"
	RecoveryStateRetryableModelFailure    = "retryable_model_failure"
	RecoveryStateEvidenceOverclaim        = "evidence_overclaim"
	RecoveryStateDomainUnauthorized       = "domain_unauthorized"
	RecoveryStateFactConflict             = "fact_conflict"
	RecoveryStateMethodContractViolation  = "method_contract_violation"
	RecoveryStateStaticFactsOnlyDegraded  = "static_facts_only_degraded"
	RecoveryStateDynamicFactsOnlyDegraded = "dynamic_facts_only_degraded"
	RecoveryStateHardError                = "hard_error"

	ContractFailureEvidenceOverclaim  = "evidence_overclaim"
	ContractFailureDomainUnauthorized = "domain_unauthorized"
	ContractFailureProjectionMismatch = "projection_mismatch"
	ContractFailureSchemaError        = "schema_error"
	ContractFailureFactConflict       = "fact_conflict"
	ContractFailureMethodContract     = "method_contract"
	RecoveryPolicyRetryOnly           = "retry_only"
	RecoveryPolicyStaticFactsOnly     = "static_facts_only"
	RecoveryPolicyDynamicFactsOnly    = "dynamic_facts_only"
	RecoveryPolicyFullFactsOnly       = "full_facts_only"
	RecoveryPolicyHardError           = "hard_error"
)

// RecoveryPolicyForFailure 根据失败阶段和分类选择固定恢复出口。
func RecoveryPolicyForFailure(stage, class string) string {
	stage = strings.TrimSpace(stage)
	switch class {
	case ContractFailureEvidenceOverclaim:
		if strings.HasPrefix(stage, "static") {
			return RecoveryPolicyStaticFactsOnly
		}
	case ContractFailureDomainUnauthorized:
		if strings.HasPrefix(stage, "dynamic") {
			return RecoveryPolicyDynamicFactsOnly
		}
	case ContractFailureProjectionMismatch:
		if strings.HasPrefix(stage, "dynamic") {
			return RecoveryPolicyDynamicFactsOnly
		}
		return RecoveryPolicyRetryOnly
	case ContractFailureSchemaError:
		return RecoveryPolicyRetryOnly
	}
	return RecoveryPolicyHardError
}

// WithStaticFallback 只为已定义的静态合同失败开放 facts-only 降级。
func WithStaticFallback(stage string, failure ContractFailure) ContractFailure {
	if failure.Class == ContractFailureFactConflict && strings.HasPrefix(strings.TrimSpace(stage), "static") && strings.TrimSpace(failure.Field) == "static.strength_balance" {
		failure.RecoveryPolicy = RecoveryPolicyStaticFactsOnly
		return failure
	}
	if failure.Class != ContractFailureProjectionMismatch || !strings.HasPrefix(strings.TrimSpace(stage), "static") {
		return failure
	}
	field := strings.TrimSpace(failure.Field)
	if field == "static.main_axis" && len(failure.MissingRefs) > 0 {
		failure.RecoveryPolicy = RecoveryPolicyStaticFactsOnly
		return failure
	}
	if field == "static.tiaohou_anchor" {
		failure.RecoveryPolicy = RecoveryPolicyStaticFactsOnly
	}
	return failure
}

// RecoveryStateForFailure 将合同失败分类映射为 Graph 状态机标签。
func RecoveryStateForFailure(failure ContractFailure) string {
	switch failure.Class {
	case ContractFailureEvidenceOverclaim:
		return RecoveryStateEvidenceOverclaim
	case ContractFailureDomainUnauthorized:
		return RecoveryStateDomainUnauthorized
	case ContractFailureFactConflict:
		return RecoveryStateFactConflict
	case ContractFailureMethodContract:
		return RecoveryStateMethodContractViolation
	default:
		return RecoveryStateRetryableModelFailure
	}
}
