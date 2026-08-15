// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责八字静态和动态已验收综合结果的值对象；
// 不读取会话，不调用模型、检索、repair、追踪或输出传输。
package domain

// StaticSynthesis 是八字静态综合的已验收结果。
type StaticSynthesis struct {
	Source         string `json:"source,omitempty"`
	RecoveryReason string `json:"recovery_reason,omitempty"`
	// FieldAudit records runtime-only local wording repairs. It is neither
	// rendered nor returned to another model as part of the judgment.
	FieldAudit              []string            `json:"-"`
	RuleProfile             string              `json:"rule_profile"`
	MainAxis                string              `json:"main_axis"`
	ClaimStrength           string              `json:"claim_strength"`
	SupportLevel            string              `json:"support_level"`
	LimitationLevel         string              `json:"limitation_level"`
	WordingCap              string              `json:"wording_cap"`
	ConsistencyFlags        []string            `json:"consistency_flags"`
	AxisLevel               string              `json:"axis_level"`
	EffectOnTiaohou         string              `json:"effect_on_tiaohou"`
	EffectOnCoreDisease     string              `json:"effect_on_core_disease"`
	EffectOnJiShenDirection string              `json:"effect_on_jishen_direction"`
	AxisCeiling             string              `json:"axis_ceiling"`
	ConflictReasons         []string            `json:"conflict_reasons"`
	PatternBasis            string              `json:"pattern_basis"`
	PatternOutcome          string              `json:"pattern_outcome"`
	CounterEvidence         string              `json:"counter_evidence"`
	AxisConsistency         string              `json:"axis_consistency"`
	TiaohouConstraint       string              `json:"tiaohou_constraint"`
	TiaohouAnchor           string              `json:"tiaohou_anchor"`
	StrengthBalance         string              `json:"strength_balance"`
	Strength                StrengthJudgment    `json:"strength,omitempty"`
	Usage                   UsageLayers         `json:"usage,omitempty"`
	PatternAdjudication     PatternAdjudication `json:"pattern_adjudication"`
	PatternAndQingZhuo      string              `json:"pattern_and_qing_zhuo"`
	QiShiOrCongHua          string              `json:"qishi_or_conghua"`
	TierJudgment            string              `json:"tier_judgment"`
	TierBasis               string              `json:"tier_basis"`
	TierAssessment          TierAssessment      `json:"tier_assessment"`
	ReasoningSummary        string              `json:"reasoning_summary"`
	ReasoningSteps          []string            `json:"reasoning_steps"`
	TopicDirectAnswer       string              `json:"topic_direct_answer,omitempty"`
	TopicFocusAnswer        string              `json:"topic_focus_answer,omitempty"`
	Advantages              []string            `json:"advantages"`
	Risks                   []string            `json:"risks"`
	Citations               []Citation          `json:"citations"`
	Assertions              []Assertion         `json:"assertions,omitempty"`
	ContractAudit           ContractAudit       `json:"-"`
}

// DynamicSynthesis 是八字动态综合的已验收结果。
type DynamicSynthesis struct {
	Source         string `json:"source,omitempty"`
	RecoveryReason string `json:"recovery_reason,omitempty"`
	// FieldAudit records runtime-only local wording repairs.
	FieldAudit               []string        `json:"-"`
	CurrentTrend             string          `json:"current_trend"`
	CurrentPeriodRealization string          `json:"current_period_realization"`
	ClaimStrength            string          `json:"claim_strength"`
	SupportLevel             string          `json:"support_level"`
	LimitationLevel          string          `json:"limitation_level"`
	WordingCap               string          `json:"wording_cap"`
	ConsistencyFlags         []string        `json:"consistency_flags"`
	DayunPath                []string        `json:"dayun_path"`
	DayunJudgments           []DayunJudgment `json:"dayun_judgments,omitempty"`
	// CurrentDayunIndex identifies the current period in the chronologically
	// ordered DayunPath. It prevents validators from treating the first period
	// as the current period when the path includes the full life sequence.
	CurrentDayunIndex int           `json:"current_dayun_index"`
	LiunianFocus      string        `json:"liunian_focus"`
	WindowLevel       string        `json:"window_level"`
	TriggerSignals    []string      `json:"trigger_signals"`
	KeyWindows        []string      `json:"key_windows"`
	Risks             []string      `json:"risks"`
	ReasoningSummary  string        `json:"reasoning_summary"`
	ReasoningSteps    []string      `json:"reasoning_steps"`
	OutcomeDomains    []string      `json:"outcome_domains"`
	Assertions        []Assertion   `json:"assertions,omitempty"`
	ContractAudit     ContractAudit `json:"-"`
}
