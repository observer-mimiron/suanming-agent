// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责八字已接受综合结果的纯领域值对象；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

type CanonicalUnit struct {
	Kind           string        `json:"kind"`
	Verdict        string        `json:"verdict"`
	Boundary       string        `json:"boundary,omitempty"`
	FactRefs       []string      `json:"fact_refs,omitempty"`
	RelationRefs   []RelationRef `json:"relation_refs,omitempty"`
	ClaimRefs      []string      `json:"claim_refs,omitempty"`
	EvidenceTopics []string      `json:"evidence_topics,omitempty"`
	Confidence     string        `json:"confidence,omitempty"`
}

// CanonicalDayunUnit 是与确定性大运事实分离的模型大运判断。
type CanonicalDayunUnit struct {
	Index          *int          `json:"index,omitempty"`
	GanZhi         string        `json:"gan_zhi,omitempty"`
	Verdict        string        `json:"verdict"`
	Boundary       string        `json:"boundary,omitempty"`
	FactRefs       []string      `json:"fact_refs,omitempty"`
	RelationRefs   []RelationRef `json:"relation_refs,omitempty"`
	ClaimRefs      []string      `json:"claim_refs,omitempty"`
	EvidenceTopics []string      `json:"evidence_topics,omitempty"`
	Confidence     string        `json:"confidence,omitempty"`
}

// CanonicalSynthesis 是八字 Graph 接受的整盘综合结果。
type CanonicalSynthesis struct {
	Source                   string               `json:"source,omitempty"`
	RecoveryReason           string               `json:"recovery_reason,omitempty"`
	FieldAudit               []string             `json:"-"`
	StaticReasoningSummary   string               `json:"-"`
	DynamicReasoningSummary  string               `json:"-"`
	DynamicOutcomeDomains    []string             `json:"-"`
	MainAxis                 CanonicalUnit        `json:"main_axis"`
	Strength                 CanonicalUnit        `json:"strength"`
	Tiaohou                  CanonicalUnit        `json:"tiaohou"`
	Pattern                  CanonicalUnit        `json:"pattern"`
	Tier                     CanonicalUnit        `json:"tier"`
	TierAssessment           TierAssessment       `json:"-"`
	DayunOverview            CanonicalUnit        `json:"dayun_overview"`
	DayunPeriods             []CanonicalDayunUnit `json:"dayun_periods,omitempty"`
	Liunian                  CanonicalUnit        `json:"liunian"`
	CurrentPeriodRealization string               `json:"-"`
	Limitations              []string             `json:"limitations,omitempty"`
	Advantages               []string             `json:"advantages,omitempty"`
	Risks                    []string             `json:"risks,omitempty"`
	ReasoningSteps           []string             `json:"reasoning_steps,omitempty"`
	AdviceBoundary           string               `json:"advice_boundary,omitempty"`
	Citations                []Citation           `json:"citations,omitempty"`
	ContractAudit            ContractAudit        `json:"-"`
}

// StrengthJudgment 是整盘强弱的结论和确定性依据说明。
type StrengthJudgment struct {
	Conclusion string `json:"conclusion,omitempty"`
	Reasoning  string `json:"reasoning,omitempty"`
	Boundary   string `json:"boundary,omitempty"`
}

// UsageLayers 将扶抑、格局和调候的取用方向保持独立。
type UsageLayers struct {
	Fuyi     string `json:"fuyi,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
	Tiaohou  string `json:"tiaohou,omitempty"`
	Priority string `json:"priority,omitempty"`
}
