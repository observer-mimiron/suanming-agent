// Package domain 定义八字领域校验错误及其字段信息。
//
// 本文件定义合同校验错误的统一包装，供合同审计、repair、恢复和 trace 复用；
// 不执行断言校验，也不决定具体恢复策略。
package domain

// baziValidationError 保留 runtime 的兼容名称；错误包装逻辑由 Bazi domain 所有。
type baziValidationError = ValidationError

// baziViolationError 创建带字段、引用和允许集合的机器可读合同错误。
func baziViolationError(code baziViolationCode, field, assertionID, message string, missing, allowed []string) error {
	return NewValidationError(code, field, assertionID, message, missing, allowed)
}

// baziViolationFromError 提取合同错误携带的结构化 violation。
func baziViolationFromError(err error) (baziValidationViolation, bool) {
	return ValidationViolationFromError(err)
}
