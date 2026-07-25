package runtime

type baziCharterInput struct {
	UserQuestion string         `json:"user_question"`
	BaziResult   map[string]any `json:"bazi_result"`
	Yongshen     map[string]any `json:"yongshen"`
	Dayun        map[string]any `json:"dayun"`
	Liunian      map[string]any `json:"liunian"`
}

type baziCitation struct {
	Classic string   `json:"classic"`
	Quotes  []string `json:"quotes"`
}

type baziAnalysisPlan struct {
	Mode           string   `json:"mode"`
	RetrievalStage string   `json:"retrieval_stage"`
	NeedDynamic    bool     `json:"need_dynamic"`
	FocusTopics    []string `json:"focus_topics"`
	WriterTemplate string   `json:"writer_template"`
	TopicMode      string   `json:"topic_mode,omitempty"`
	StageSummary   string   `json:"stage_summary"`
}

type baziStaticSynthesis struct {
	MainAxis                string         `json:"main_axis"`
	ClaimStrength           string         `json:"claim_strength"`
	SupportLevel            string         `json:"support_level"`
	LimitationLevel         string         `json:"limitation_level"`
	WordingCap              string         `json:"wording_cap"`
	ConsistencyFlags        []string       `json:"consistency_flags"`
	AxisLevel               string         `json:"axis_level"`
	EffectOnTiaohou         string         `json:"effect_on_tiaohou"`
	EffectOnCoreDisease     string         `json:"effect_on_core_disease"`
	EffectOnJiShenDirection string         `json:"effect_on_jishen_direction"`
	AxisCeiling             string         `json:"axis_ceiling"`
	ConflictReasons         []string       `json:"conflict_reasons"`
	PatternBasis            string         `json:"pattern_basis"`
	PatternOutcome          string         `json:"pattern_outcome"`
	CounterEvidence         string         `json:"counter_evidence"`
	AxisConsistency         string         `json:"axis_consistency"`
	TiaohouConstraint       string         `json:"tiaohou_constraint"`
	TiaohouAnchor           string         `json:"tiaohou_anchor"`
	StrengthBalance         string         `json:"strength_balance"`
	PatternAndQingZhuo      string         `json:"pattern_and_qing_zhuo"`
	QiShiOrCongHua          string         `json:"qishi_or_conghua"`
	TierJudgment            string         `json:"tier_judgment"`
	TierBasis               string         `json:"tier_basis"`
	ReasoningSummary        string         `json:"reasoning_summary"`
	ReasoningSteps          []string       `json:"reasoning_steps"`
	TopicDirectAnswer       string         `json:"topic_direct_answer,omitempty"`
	TopicFocusAnswer        string         `json:"topic_focus_answer,omitempty"`
	Advantages              []string       `json:"advantages"`
	Risks                   []string       `json:"risks"`
	Citations               []baziCitation `json:"citations"`
}

type baziDynamicSynthesis struct {
	CurrentTrend     string   `json:"current_trend"`
	ClaimStrength    string   `json:"claim_strength"`
	SupportLevel     string   `json:"support_level"`
	LimitationLevel  string   `json:"limitation_level"`
	WordingCap       string   `json:"wording_cap"`
	ConsistencyFlags []string `json:"consistency_flags"`
	DayunPath        []string `json:"dayun_path"`
	LiunianFocus     string   `json:"liunian_focus"`
	WindowLevel      string   `json:"window_level"`
	TriggerSignals   []string `json:"trigger_signals"`
	KeyWindows       []string `json:"key_windows"`
	Risks            []string `json:"risks"`
	ReasoningSummary string   `json:"reasoning_summary"`
	ReasoningSteps   []string `json:"reasoning_steps"`
}

type baziCharterState struct {
	AnalysisPlan     baziAnalysisPlan     `json:"analysis_plan"`
	Input            baziCharterInput     `json:"input"`
	EvidencePlan     baziEvidencePlan     `json:"evidence_plan"`
	EvidenceBundle   baziEvidenceBundle   `json:"evidence_bundle"`
	EvidenceQuality  baziEvidenceQuality  `json:"evidence_quality"`
	StaticSynthesis  baziStaticSynthesis  `json:"static_synthesis"`
	DynamicSynthesis baziDynamicSynthesis `json:"dynamic_synthesis"`
}
