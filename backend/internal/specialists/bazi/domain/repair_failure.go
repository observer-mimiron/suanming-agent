// Package domain 包含八字领域的不变事实和合同。
//
// 本文件负责领域合同失败的可序列化分类；
// 不依赖共享 repair、runtime Executor、会话、模型、检索或 SSE sink。
package domain

// ContractFailure 是八字合同失败的可序列化分类状态。
type ContractFailure struct {
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
