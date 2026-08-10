// Package runtime 包含八字合同 finding 的 runtime 投影辅助。
//
// 本文件负责把合同 finding 转为统一校验错误并生成 trace 摘要；
// 不调用模型，不决定恢复策略，也不改写用户可见文本。
package runtime

import (
	"fmt"
	"strings"
)

// baziContractAuditError preserves the finding code and field so retry,
// recovery and trace reporting do not infer failure classes from prose.
func baziContractAuditError(stage string, finding baziContractAuditFinding) error {
	field := firstNonEmptyTrim(finding.Field, strings.TrimSpace(stage))
	message := firstNonEmptyTrim(finding.Reason, finding.Code, "independent synthesis contract audit failed")
	return baziValidationError{Violation: baziValidationViolation{
		Code:                baziViolationSemanticContract,
		Field:               field,
		Message:             message,
		ContractFindingCode: strings.TrimSpace(finding.Code),
		DetectedDomain:      strings.TrimSpace(finding.DetectedDomain),
		Excerpt:             strings.TrimSpace(finding.Excerpt),
	}}
}

// baziContractAuditSummary returns a compact trace value without retaining the
// complete model-authored audit explanation.
func baziContractAuditSummary(audit baziContractAudit) string {
	if audit.Compliant && len(audit.Findings) == 0 {
		return "clean"
	}
	if len(audit.Findings) == 0 {
		return "not_run"
	}
	return fmt.Sprintf("failed:%s", strings.TrimSpace(audit.Findings[0].Code))
}
