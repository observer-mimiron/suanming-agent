// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责八字结构化模型输出的纯数据合同；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

// StructuredClaim 是模型输出的最小可验证结论。
type StructuredClaim struct {
	Verdict        string        `json:"verdict"`
	FactRefs       []FactRef     `json:"fact_refs"`
	RelationRefs   []RelationRef `json:"relation_refs"`
	ClaimRefs      []ClaimRef    `json:"claim_refs"`
	EvidenceTopics []string      `json:"evidence_topics"`
	Confidence     string        `json:"confidence"`
	Boundary       string        `json:"boundary"`
}

// StructuredStaticClaim 是静态节点的受限裁断槽位。
type StructuredStaticClaim struct {
	Slot           string     `json:"slot"`
	Verdict        string     `json:"verdict"`
	Status         string     `json:"status"`
	FactRefs       []FactRef  `json:"fact_refs"`
	ClaimRefs      []ClaimRef `json:"claim_refs"`
	EvidenceTopics []string   `json:"evidence_topics"`
}

// TierDimension 是层次裁断的一个固定观察面。
type TierDimension struct {
	State          string     `json:"state"`
	FactRefs       []FactRef  `json:"fact_refs"`
	ClaimRefs      []ClaimRef `json:"claim_refs"`
	EvidenceTopics []string   `json:"evidence_topics"`
}

// TierDimensions 固定九个传统层次观察面，避免模型增删维度。
type TierDimensions struct {
	MainAxis   TierDimension `json:"main_axis"`
	YouQing    TierDimension `json:"youqing"`
	YouLi      TierDimension `json:"youli"`
	QingZhuo   TierDimension `json:"qingzhuo"`
	Disease    TierDimension `json:"disease"`
	Remedy     TierDimension `json:"remedy"`
	Rescue     TierDimension `json:"rescue"`
	Tiaohou    TierDimension `json:"tiaohou"`
	HeZhiZhang TierDimension `json:"hezhizhang"`
}

// TierAssessment 是静态命局的基础层次槽位，不表示财富或人格价值。
type TierAssessment struct {
	Status     string         `json:"status"`
	Level      int            `json:"level"`
	Confidence string         `json:"confidence"`
	Dimensions TierDimensions `json:"dimensions"`
}

// StructuredStaticSynthesis 是静态节点的原始模型输出。
type StructuredStaticSynthesis struct {
	Claims          []StructuredStaticClaim `json:"claims"`
	AxisStatus      string                  `json:"axis_status"`
	TierAssessment  TierAssessment          `json:"tier_assessment"`
	NatalRiskStatus string                  `json:"natal_risk_status"`
}

// StructuredPeriodClaim 是模型选择的重点大运结论。
type StructuredPeriodClaim struct {
	PeriodRef      string        `json:"period_ref"`
	Verdict        string        `json:"verdict"`
	FactRefs       []FactRef     `json:"fact_refs"`
	RelationRefs   []RelationRef `json:"relation_refs"`
	ClaimRefs      []ClaimRef    `json:"claim_refs"`
	EvidenceTopics []string      `json:"evidence_topics"`
	Confidence     string        `json:"confidence"`
	Boundary       string        `json:"boundary"`
}

// StructuredDynamicSynthesis 是动态节点的原始模型输出。
type StructuredDynamicSynthesis struct {
	CurrentPeriodRef         string                  `json:"current_period_ref"`
	CurrentPeriodRealization string                  `json:"current_period_realization"`
	PeriodClaims             []StructuredPeriodClaim `json:"period_claims"`
	LiunianClaim             StructuredClaim         `json:"liunian_claim"`
	Limitations              []string                `json:"limitations"`
	ReasoningSummary         string                  `json:"reasoning_summary"`
	ReasoningSteps           []string                `json:"reasoning_steps"`
	OutcomeDomains           []string                `json:"outcome_domains"`
}

// PatternCandidate 是一条待比较的静态格局路线。
type PatternCandidate struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Origin               string    `json:"origin"`
	Visibility           string    `json:"visibility"`
	Role                 string    `json:"role"`
	FactRefs             []FactRef `json:"fact_refs,omitempty"`
	EvidenceTopics       []string  `json:"evidence_topics,omitempty"`
	ComparisonDimensions any       `json:"comparison_dimensions,omitempty"`
	RejectionReasons     []string  `json:"rejection_reasons,omitempty"`
}

// PatternAdjudication 是模型对候选路线比较后的选择记录。
type PatternAdjudication struct {
	MonthCommandCandidateID  string             `json:"month_command_candidate_id"`
	SelectedAxisCandidateIDs []string           `json:"selected_axis_candidate_ids"`
	Candidates               []PatternCandidate `json:"candidates"`
}
