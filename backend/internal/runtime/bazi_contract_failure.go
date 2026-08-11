// Package runtime 分类八字综合合同失败，供 repair、恢复和 trace 投影使用。
//
// 合同 finding 只提供语义元数据，本文件只映射成确定性 runtime 动作；
// 不重写命理解读，也不引入命盘专项分支。
package runtime

import (
	"errors"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/repair"
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
type baziContractFailure struct {
	Class          string
	FindingCode    string
	Field          string
	DetectedDomain string
	Excerpt        string
	Reason         string
	MissingRefs    []string
	AllowedRefs    []string
	RecoveryPolicy string
}

// baziContractFailureFromAuditFinding 把合同 finding 映射到闭合失败分类。
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
// 若错误携带 finding metadata，则保留它供 repair 和 trace 使用。
func baziContractFailureFromViolation(stage string, violation baziValidationViolation) baziContractFailure {
	failure := baziContractFailure{
		Class:          baziContractFailureUnknown,
		FindingCode:    strings.TrimSpace(violation.ContractFindingCode),
		Field:          strings.TrimSpace(violation.Field),
		DetectedDomain: strings.TrimSpace(violation.DetectedDomain),
		Excerpt:        strings.TrimSpace(violation.Excerpt),
		Reason:         strings.TrimSpace(violation.Message),
		MissingRefs:    append([]string(nil), violation.MissingRefs...),
		AllowedRefs:    append([]string(nil), violation.AllowedRefs...),
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
	case baziViolationUndeclaredFactClaim:
		// 未声明引用是输出合同错误：模型可根据同轮 catalog 重写一次；
		// 它不代表确定性事实本身冲突，不能与 fact_value_mismatch 共用硬失败路径。
		failure.Class = baziContractFailureSchemaError
	case baziViolationFactConflict, baziViolationFactRefMissing:
		failure.Class = baziContractFailureFactConflict
	case baziViolationMethodContract, baziViolationClaimNotAuthorized:
		// 动态模型把 runtime 引用路径写进用户可见文本时，候选文本可安全丢弃；
		// 保留静态结论并生成动态 facts-only。其它方法合同仍必须硬失败。
		if isDynamicPresentationReferenceViolation(stage, violation) {
			failure.Class = baziContractFailureProjectionMismatch
		} else {
			failure.Class = baziContractFailureMethodContract
		}
	case baziViolationScopeEscalation, baziViolationDayunCoverageMissing, baziViolationSemanticContract:
		failure.Class = baziContractFailureProjectionMismatch
	}
	failure.RecoveryPolicy = baziRecoveryPolicyForFailure(stage, failure.Class)
	return withBaziStaticFallback(stage, failure)
}

// isDynamicPresentationReferenceViolation 识别动态文本泄露内部引用路径的可降级错误。
// 只有带 invalid ref 且字段属于动态用户可见文本的合同错误才进入 facts-only。
func isDynamicPresentationReferenceViolation(stage string, violation baziValidationViolation) bool {
	if !strings.HasPrefix(strings.TrimSpace(stage), "dynamic") || len(violation.MissingRefs) == 0 {
		return false
	}
	field := strings.TrimSpace(violation.Field)
	return strings.HasPrefix(field, "dynamic.") &&
		(strings.Contains(field, "limitations") || strings.Contains(field, "reasoning") || strings.Contains(field, "period_claims") || strings.Contains(field, "liunian_claim"))
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
func repairFailureFromBaziContract(stage string, err error) (repair.Failure, bool) {
	var schemaErr *structured.Error
	if errors.As(err, &schemaErr) {
		failure := repair.Failure{
			Domain: "bazi", Stage: strings.TrimSpace(stage), Class: repair.SchemaError,
			Field: schemaErr.Schema, Code: "schema_error", Message: schemaErr.Detail,
			Fallback: baziStructuredFailureFallback(stage), Cause: err,
		}
		decision := repair.DefaultPolicy().Decide(failure, repair.State{})
		failure.Retryable, failure.Repairable = decision.Retryable, decision.Repairable
		return failure, true
	}
	if isBaziInnerAgentParseError(err) {
		failure := repair.Failure{
			Domain: "bazi", Stage: strings.TrimSpace(stage), Class: repair.ParseError,
			Field: "output", Code: "parse_error", Message: err.Error(),
			Fallback: baziStructuredFailureFallback(stage), Cause: err,
		}
		decision := repair.DefaultPolicy().Decide(failure, repair.State{})
		failure.Retryable, failure.Repairable = decision.Retryable, decision.Repairable
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
		Code:     strings.TrimSpace(failure.FindingCode),
		Message:  strings.TrimSpace(failure.Reason),
		Excerpt:  strings.TrimSpace(failure.Excerpt),
		Fallback: repairFallbackFromBaziRecoveryPolicy(failure.RecoveryPolicy),
		Cause:    err,
	}
	if violation, violationOK := baziViolationFromError(err); violationOK {
		repairFailure.MissingRefs = append([]string(nil), violation.MissingRefs...)
		repairFailure.AllowedRefs = append([]string(nil), violation.AllowedRefs...)
	}
	decision := repair.DefaultPolicy().Decide(repairFailure, repair.State{})
	repairFailure.Retryable = decision.Retryable
	repairFailure.Repairable = decision.Repairable
	return repairFailure, true
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
		if strings.HasPrefix(stage, "dynamic") {
			return baziRecoveryPolicyDynamicFactsOnly
		}
		return baziRecoveryPolicyRetryOnly
	case baziContractFailureSchemaError:
		return baziRecoveryPolicyRetryOnly
	case baziContractFailureFactConflict, baziContractFailureMethodContract:
		return baziRecoveryPolicyHardError
	default:
		return baziRecoveryPolicyHardError
	}
}

// withBaziStaticFallback allows only model-owned static claim slots to fall back
// after repair exhaustion. Facts-only projection errors, fact conflicts and
// method-contract violations remain hard errors because recovery cannot hide an
// invalid deterministic result.
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
	field := strings.TrimSpace(failure.Field)
	if field == "static.main_axis" && len(failure.MissingRefs) > 0 {
		failure.RecoveryPolicy = baziRecoveryPolicyStaticFactsOnly
		return failure
	}
	if field != "static.tiaohou_anchor" {
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
		decision := repair.DefaultPolicy().Decide(repairFailure, repair.State{})
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
