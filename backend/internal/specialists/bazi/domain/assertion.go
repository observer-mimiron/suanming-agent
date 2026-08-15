// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责八字断言与引用 ID 的稳定值对象；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

// AssertionKind 是八字可校验结论槽位的闭合集合。
type AssertionKind string

const (
	// AssertionMainAxis 表示本命主轴断言。
	AssertionMainAxis AssertionKind = "main_axis"
	// AssertionStrength 表示强弱断言。
	AssertionStrength AssertionKind = "strength"
	// AssertionTiaohou 表示调候断言。
	AssertionTiaohou AssertionKind = "tiaohou"
	// AssertionPatternUsage 表示格局取用断言。
	AssertionPatternUsage AssertionKind = "pattern_usage"
	// AssertionTier 表示层次断言。
	AssertionTier AssertionKind = "tier"
	// AssertionDayunPeriod 表示单个大运断言。
	AssertionDayunPeriod AssertionKind = "dayun_period"
	// AssertionLiunian 表示流年断言。
	AssertionLiunian AssertionKind = "liunian"
	// AssertionTopicAnswer 表示专题回答断言。
	AssertionTopicAnswer AssertionKind = "topic_answer"
)

// FactRef 是模型可引用的确定性事实 ID。
type FactRef string

// ClaimRef 是规则资料已声明的结论 ID。
type ClaimRef string

// RelationRef 是模型可引用的已计算关系 ID；关系文字由外层目录投影提供。
type RelationRef string

// Assertion 是最小可验证的八字结论单元。
// 它只记录结论和授权依据，不生成解释，不选择恢复策略。
type Assertion struct {
	ID             string        `json:"id"`
	Kind           AssertionKind `json:"kind"`
	Subject        string        `json:"subject"`
	Verdict        string        `json:"verdict"`
	FactRefs       []FactRef     `json:"fact_refs,omitempty"`
	RelationRefs   []RelationRef `json:"relation_refs,omitempty"`
	ClaimRefs      []ClaimRef    `json:"claim_refs,omitempty"`
	EvidenceTopics []string      `json:"evidence_topics,omitempty"`
	EvidenceStatus string        `json:"evidence_status,omitempty"`
	Confidence     string        `json:"confidence,omitempty"`
	Boundary       string        `json:"boundary,omitempty"`
}
