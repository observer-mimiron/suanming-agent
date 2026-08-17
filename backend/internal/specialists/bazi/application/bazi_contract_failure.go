// Package application 分类八字综合合同失败，供 adapter 的 repair 和恢复使用。
//
// 合同 finding 只映射为共享 repair envelope；不投影 trace、重写命理解读，
// 也不引入命盘专项分支。
package application

import (
	"errors"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/repair"
	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
	"github.com/observer-mimiron/suanming-agent/internal/structured"
)

const (
	baziContractFailureEvidenceOverclaim  = "evidence_overclaim"
	baziContractFailureDomainUnauthorized = "domain_unauthorized"
	baziContractFailureProjectionMismatch = "projection_mismatch"
	baziContractFailureSchemaError        = "schema_error"
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
type baziContractFailure = bazidomain.ContractFailure

// baziContractFailureFromViolation 分类确定性 validator 错误。
// 若错误携带 finding metadata，则保留它供 repair 和 trace 使用。
func baziContractFailureFromViolation(stage string, violation baziValidationViolation) baziContractFailure {
	return bazidomain.ClassifyViolation(stage, violation)
}

// baziContractFailureFromError extracts the classification from validation
// errors; non-contract errors intentionally remain unclassified.
func baziContractFailureFromError(stage string, err error) (baziContractFailure, bool) {
	violation, ok := bazidomain.ViolationFromError(err)
	if !ok {
		return baziContractFailure{}, false
	}
	return baziContractFailureFromViolation(stage, violation), true
}

// ContractFailureFromError classifies a Bazi contract error for a runtime adapter.
func ContractFailureFromError(stage string, err error) (bazidomain.ContractFailure, bool) {
	return baziContractFailureFromError(stage, err)
}

// repairFailureFromBaziContract maps BaZi-only validation failures into the
// global Repair Harness contract without changing current control flow.
func repairFailureFromBaziContract(stage string, err error) (repair.Failure, bool) {
	var schemaErr *structured.Error
	if errors.As(err, &schemaErr) {
		failure := repair.Failure{
			Domain: "bazi", Stage: strings.TrimSpace(stage), Class: repair.SchemaError,
			Field: schemaErr.Schema, Origin: repair.OriginModelCandidate, Code: "schema_error", Message: schemaErr.Detail,
			Fallback: baziStructuredFailureFallback(stage), Cause: err,
		}
		return failure, true
	}
	if isBaziInnerAgentParseError(err) {
		failure := repair.Failure{
			Domain: "bazi", Stage: strings.TrimSpace(stage), Class: repair.ParseError,
			Field: "output", Origin: repair.OriginModelCandidate, Code: "parse_error", Message: err.Error(),
			Fallback: baziStructuredFailureFallback(stage), Cause: err,
		}
		return failure, true
	}
	failure, ok := baziContractFailureFromError(stage, err)
	if !ok {
		return repair.Failure{}, false
	}
	repairFailure := repair.Failure{
		Domain:   "bazi",
		Stage:    strings.TrimSpace(stage),
		Class:    repairClassFromBaziContract(failure.Class),
		Field:    strings.TrimSpace(failure.Field),
		Origin:   repair.OriginModelCandidate,
		Code:     strings.TrimSpace(failure.FindingCode),
		Message:  strings.TrimSpace(failure.Reason),
		Excerpt:  strings.TrimSpace(failure.Excerpt),
		Fallback: repairFallbackFromBaziRecoveryPolicy(failure.RecoveryPolicy),
		Cause:    err,
	}
	if violation, violationOK := bazidomain.ViolationFromError(err); violationOK {
		repairFailure.MissingRefs = append([]string(nil), violation.MissingRefs...)
		repairFailure.AllowedRefs = append([]string(nil), violation.AllowedRefs...)
	}
	return repairFailure, true
}

// RepairFailureFromError maps a Bazi contract error to the shared repair contract.
func RepairFailureFromError(stage string, err error) (repair.Failure, bool) {
	return repairFailureFromBaziContract(stage, err)
}

// baziStructuredFailureFallback 允许动态结构化输出在一次 repair 失败后保留静态结果。
// static/canonical 仍沿用各自恢复策略，避免把所有节点错误都静默吞掉。
func baziStructuredFailureFallback(stage string) string {
	if strings.HasPrefix(strings.TrimSpace(stage), "dynamic") {
		return "facts_only"
	}
	return ""
}

// baziRecoveryPolicyForFailure 是合同失败恢复策略的基础映射。
// 结构投影失败先统一尝试一次 repair；动态层修复耗尽后可以保留已验收的
// 静态结果并降级为事实展示，事实和方法冲突仍无降级出口。
func baziRecoveryPolicyForFailure(stage, class string) string {
	return bazidomain.RecoveryPolicyForFailure(stage, class)
}

// withBaziStaticFallback allows only model-owned static claim slots to fall back
// after repair exhaustion. Facts-only projection errors, fact conflicts and
// method-contract violations remain hard errors because recovery cannot hide an
// invalid deterministic result.
func withBaziStaticFallback(stage string, failure baziContractFailure) baziContractFailure {
	return bazidomain.WithStaticFallback(stage, failure)
}

// repairClassFromBaziContract bridges the BaZi-local closed taxonomy into the
// global Repair Harness taxonomy.
func repairClassFromBaziContract(class string) repair.Class {
	switch class {
	case baziContractFailureEvidenceOverclaim:
		return repair.EvidenceOverclaim
	case baziContractFailureDomainUnauthorized:
		return repair.DomainUnauthorized
	case baziContractFailureProjectionMismatch:
		return repair.ProjectionMismatch
	case baziContractFailureSchemaError:
		return repair.SchemaError
	case baziContractFailureFactConflict:
		return repair.FactConflict
	case baziContractFailureMethodContract:
		return repair.MethodContract
	default:
		return repair.Unknown
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
