// Package runtime 包含 Manager 拥有的结构化输出合同。
//
// 本文件负责 Schema registry、prompt 注入和原始输出严格解码；
// 不负责八字语义、恢复策略、渲染或 SSE 表示。
package runtime

import "github.com/observer-mimiron/suanming-agent/internal/structured"

// structuredOutputPromptContract 将校验器使用的同一份 Schema 注入结构化 prompt。
func structuredOutputPromptContract(name string) (string, error) {
	return structured.PromptContract(name)
}

// structuredSchemaHash 返回稳定指纹，供 prompt 与校验器同源回归测试使用。
func structuredSchemaHash(name string) (string, error) {
	return structured.Hash(name)
}

// decodeStructuredOutput 在 DTO 进入领域校验前拒绝空、fence 或不合规的原始输出。
