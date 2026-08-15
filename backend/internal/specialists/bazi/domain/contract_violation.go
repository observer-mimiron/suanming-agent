// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责八字合同 violation 的稳定错误元数据；
// 不决定 retry、降级或 hard-error，不调用模型、repair、追踪或输出传输。
package domain

import (
	"errors"
	"fmt"
	"strings"
)

// ViolationCode 是八字合同校验失败的稳定机器码。
type ViolationCode string

const (
	// ViolationFactRefMissing 表示必要事实引用缺失。
	ViolationFactRefMissing ViolationCode = "fact_ref_missing"
	// ViolationUndeclaredFactClaim 表示模型使用了目录外引用。
	ViolationUndeclaredFactClaim ViolationCode = "undeclared_fact_claim"
	// ViolationFactConflict 表示裁断与确定性事实冲突。
	ViolationFactConflict ViolationCode = "fact_conflict"
	// ViolationClaimNotAuthorized 表示规则资料未授权该结论。
	ViolationClaimNotAuthorized ViolationCode = "claim_not_authorized"
	// ViolationScopeEscalation 表示结论越过年龄或主题范围。
	ViolationScopeEscalation ViolationCode = "scope_escalation"
	// ViolationDayunCoverageMissing 表示大运覆盖不足。
	ViolationDayunCoverageMissing ViolationCode = "dayun_coverage_missing"
	// ViolationMethodContract 表示固定方法合同未满足。
	ViolationMethodContract ViolationCode = "method_contract_violation"
	// ViolationEvidenceTopicMissing 表示必需证据主题缺失。
	ViolationEvidenceTopicMissing ViolationCode = "evidence_topic_missing"
	// ViolationSemanticContract 表示语义合同失败。
	ViolationSemanticContract ViolationCode = "semantic_contract_violation"
	// ViolationUnsupportedConcreteOutcome 表示出现未授权的具体应事。
	ViolationUnsupportedConcreteOutcome ViolationCode = "unsupported_concrete_outcome"
	// ViolationRendererContract 表示展示合同失败。
	ViolationRendererContract ViolationCode = "renderer_contract_violation"
)

// ValidationViolation 为恢复和反馈提供机器可读的合同失败原因。
type ValidationViolation struct {
	Code                ViolationCode `json:"code"`
	Field               string        `json:"field,omitempty"`
	Message             string        `json:"message"`
	AssertionID         string        `json:"assertion_id,omitempty"`
	MissingRefs         []string      `json:"missing_refs,omitempty"`
	AllowedRefs         []string      `json:"allowed_refs,omitempty"`
	ContractFindingCode string        `json:"contract_finding_code,omitempty"`
	DetectedDomain      string        `json:"detected_domain,omitempty"`
	Excerpt             string        `json:"excerpt,omitempty"`
}

// ValidationError 把 ValidationViolation 包装为 error，避免调用方从文案反推失败类别。
type ValidationError struct {
	Violation ValidationViolation
}

// Error 返回稳定错误码和简短说明。
func (e ValidationError) Error() string {
	if strings.TrimSpace(e.Violation.Message) == "" {
		return string(e.Violation.Code)
	}
	return fmt.Sprintf("%s: %s", e.Violation.Code, e.Violation.Message)
}

// NewValidationError 创建带字段和引用集合的机器可读合同错误。
func NewValidationError(code ViolationCode, field, assertionID, message string, missing, allowed []string) error {
	return ValidationError{Violation: ValidationViolation{
		Code:        code,
		Field:       field,
		Message:     message,
		AssertionID: assertionID,
		MissingRefs: NonEmptyStrings(missing),
		AllowedRefs: NonEmptyStrings(allowed),
	}}
}

// ValidationViolationFromError 提取 ValidationError 携带的机器可读 violation。
func ValidationViolationFromError(err error) (ValidationViolation, bool) {
	var validationErr ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Violation, true
	}
	return ValidationViolation{}, false
}
