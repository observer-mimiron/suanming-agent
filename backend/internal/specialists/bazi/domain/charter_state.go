// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责单轮八字分析链路共享的领域状态；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

// CharterState 汇集一次八字分析的输入、证据和已验收综合结果。
type CharterState struct {
	AnalysisPlan      AnalysisPlan           `json:"analysis_plan"`
	Input             CharterInput           `json:"input"`
	EvidencePlan      EvidencePlan           `json:"evidence_plan"`
	EvidenceBundle    EvidenceBundle         `json:"evidence_bundle"`
	EvidenceQuality   EvidenceQuality        `json:"evidence_quality"`
	StaticSynthesis   StaticSynthesis        `json:"static_synthesis"`
	LifetimeSynthesis LifetimeDayunSynthesis `json:"lifetime_synthesis"`
	DynamicSynthesis  DynamicSynthesis       `json:"dynamic_synthesis"`
	FieldAudit        []string               `json:"-"`
}
