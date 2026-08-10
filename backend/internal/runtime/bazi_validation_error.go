// Package runtime 包含 Manager 拥有的八字运行时合同。
//
// 本文件定义合同校验错误的统一包装，供合同审计、repair、恢复和 trace 复用；
// 不执行断言校验，也不决定具体恢复策略。
package runtime

import (
	"errors"
	"fmt"
	"strings"
)

// baziValidationError 携带机器可读的合同 violation，避免下游从文案猜测失败原因。
type baziValidationError struct {
	Violation baziValidationViolation
}

// Error 返回稳定的合同错误码和简短说明。
func (e baziValidationError) Error() string {
	if strings.TrimSpace(e.Violation.Message) == "" {
		return string(e.Violation.Code)
	}
	return fmt.Sprintf("%s: %s", e.Violation.Code, e.Violation.Message)
}

// baziViolationError 创建带字段、引用和允许集合的机器可读合同错误。
func baziViolationError(code baziViolationCode, field, assertionID, message string, missing, allowed []string) error {
	return baziValidationError{Violation: baziValidationViolation{
		Code:        code,
		Field:       field,
		Message:     message,
		AssertionID: assertionID,
		MissingRefs: filterNonEmpty(missing),
		AllowedRefs: filterNonEmpty(allowed),
	}}
}

// baziViolationFromError 提取合同错误携带的结构化 violation。
func baziViolationFromError(err error) (baziValidationViolation, bool) {
	var validationErr baziValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Violation, true
	}
	return baziValidationViolation{}, false
}
