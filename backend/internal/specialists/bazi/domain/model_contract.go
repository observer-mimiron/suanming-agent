// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责八字模型节点之间传递的纯数据合同；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

// AnalysisPlan 是模型选择的八字分析范围，不携带模型调用或会话状态。
type AnalysisPlan struct {
	Mode              string   `json:"mode"`
	RetrievalStage    string   `json:"retrieval_stage"`
	NeedDynamic       bool     `json:"need_dynamic"`
	NeedLifetimeDayun bool     `json:"need_lifetime_dayun"`
	FocusTopics       []string `json:"focus_topics"`
	WriterTemplate    string   `json:"writer_template"`
	TopicMode         string   `json:"topic_mode,omitempty"`
	StageSummary      string   `json:"stage_summary"`
}

// LifetimeDayunClaim 是单个全程大运的受限判断，不混入本命或当前岁运结论。
type LifetimeDayunClaim struct {
	PeriodRef      string        `json:"period_ref"`
	PeriodEffect   string        `json:"period_effect"`
	Verdict        string        `json:"verdict"`
	FactRefs       []FactRef     `json:"fact_refs"`
	RelationRefs   []RelationRef `json:"relation_refs"`
	ClaimRefs      []ClaimRef    `json:"claim_refs"`
	EvidenceTopics []string      `json:"evidence_topics"`
	Confidence     string        `json:"confidence"`
}

// LifetimeDayunSynthesis 是全程大运的模型综合结果。
// 它必须覆盖既有确定性大运目录，且不携带运行时恢复或渲染状态。
type LifetimeDayunSynthesis struct {
	Status       string               `json:"status"`
	Trajectory   string               `json:"trajectory"`
	PeriodClaims []LifetimeDayunClaim `json:"period_claims"`
	Summary      string               `json:"summary"`
}
