package runtime

type baziCharterInput struct {
	UserQuestion string          `json:"user_question"`
	BaziResult   map[string]any  `json:"bazi_result"`
	Yongshen     map[string]any  `json:"yongshen"`
	Dayun        map[string]any  `json:"dayun"`
	Liunian      map[string]any  `json:"liunian"`
	RuleProfile  baziRuleProfile `json:"selected_rule_profile"`
}

// baziRuleProfile describes the sole rule family allowed to issue verdicts for
// one chart. Facts remain reusable, while profile rules are explicit and versioned.
type baziRuleProfile struct {
	ID             string                   `json:"id"`
	Status         string                   `json:"status"`
	Scope          string                   `json:"scope"`
	NotImplemented []string                 `json:"not_implemented"`
	Overlays       []baziRuleProfileOverlay `json:"overlays,omitempty"`
	Verdicts       []baziProfileRuleVerdict `json:"verdicts,omitempty"`
	Claims         []baziProfileClaim       `json:"claims,omitempty"`
}

// baziProfileClaim is a profile-owned conclusion.  Unlike chart facts, it
// records the rule and evidence that authorize a user-visible interpretation.
// The renderer may explain a claim but must not strengthen it.
type baziProfileClaim struct {
	ID           string         `json:"id"`
	Category     string         `json:"category"`
	Verdict      string         `json:"verdict"`
	Confidence   string         `json:"confidence"`
	RuleID       string         `json:"rule_id"`
	SupportFacts []string       `json:"support_facts"`
	CounterFacts []string       `json:"counter_facts,omitempty"`
	Boundary     string         `json:"boundary"`
	Citations    []baziCitation `json:"citations,omitempty"`
}

// baziRuleProfileOverlay identifies an additional rule family that is active
// only for its declared subset. It keeps 调候 rules explicit instead of
// silently treating them as part of the default 子平 profile.
type baziRuleProfileOverlay struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Scope  string `json:"scope"`
}

// baziProfileRuleVerdict is a deterministic conclusion that a selected rule
// profile is explicitly allowed to make from chart facts. It remains separate
// from the LLM's whole-chart synthesis so a narrow rule cannot become a broad
// wealth, status, or event prediction.
type baziProfileRuleVerdict struct {
	RuleID    string         `json:"rule_id"`
	Status    string         `json:"status"`
	Summary   string         `json:"summary"`
	Facts     []string       `json:"facts"`
	Boundary  string         `json:"boundary"`
	Citations []baziCitation `json:"citations"`
}

type baziCitation struct {
	Classic string   `json:"classic"`
	Quotes  []string `json:"quotes"`
}

type baziAssertionKind string

const (
	baziAssertionMainAxis     baziAssertionKind = "main_axis"
	baziAssertionStrength     baziAssertionKind = "strength"
	baziAssertionTiaohou      baziAssertionKind = "tiaohou"
	baziAssertionPatternUsage baziAssertionKind = "pattern_usage"
	baziAssertionTier         baziAssertionKind = "tier"
	baziAssertionDayunPeriod  baziAssertionKind = "dayun_period"
	baziAssertionLiunian      baziAssertionKind = "liunian"
	baziAssertionTopicAnswer  baziAssertionKind = "topic_answer"
)

type baziFactRef string
type baziClaimRef string

// baziAssertion is the smallest runtime-verifiable reading unit. The legacy
// synthesis text remains for rendering, while assertions record which chart
// facts and rule-profile claims authorize each visible conclusion.
type baziAssertion struct {
	ID         string            `json:"id"`
	Kind       baziAssertionKind `json:"kind"`
	Subject    string            `json:"subject"`
	Verdict    string            `json:"verdict"`
	FactRefs   []baziFactRef     `json:"fact_refs,omitempty"`
	ClaimRefs  []baziClaimRef    `json:"claim_refs,omitempty"`
	Confidence string            `json:"confidence,omitempty"`
	Boundary   string            `json:"boundary,omitempty"`
}

type baziViolationCode string

const (
	baziViolationFactRefMissing             baziViolationCode = "fact_ref_missing"
	baziViolationFactConflict               baziViolationCode = "fact_conflict"
	baziViolationClaimNotAuthorized         baziViolationCode = "claim_not_authorized"
	baziViolationScopeEscalation            baziViolationCode = "scope_escalation"
	baziViolationDayunCoverageMissing       baziViolationCode = "dayun_coverage_missing"
	baziViolationUnsupportedConcreteOutcome baziViolationCode = "unsupported_concrete_outcome"
	baziViolationRendererContract           baziViolationCode = "renderer_contract_violation"
)

// baziValidationViolation gives recovery and feedback a machine-readable reason
// instead of forcing them to infer semantics from Chinese output phrases.
type baziValidationViolation struct {
	Code        baziViolationCode `json:"code"`
	Field       string            `json:"field,omitempty"`
	Message     string            `json:"message"`
	AssertionID string            `json:"assertion_id,omitempty"`
	MissingRefs []string          `json:"missing_refs,omitempty"`
	AllowedRefs []string          `json:"allowed_refs,omitempty"`
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

// baziStrengthJudgment keeps the model's whole-chart strength conclusion
// separate from the deterministic balance evidence that supports it.
type baziStrengthJudgment struct {
	Conclusion string `json:"conclusion,omitempty"`
	Reasoning  string `json:"reasoning,omitempty"`
	Boundary   string `json:"boundary,omitempty"`
}

// baziUsageLayers keeps 扶抑、格局与调候的取用 directions distinct. They can
// point in different directions without becoming contradictory global 喜忌.
type baziUsageLayers struct {
	Fuyi     string `json:"fuyi,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
	Tiaohou  string `json:"tiaohou,omitempty"`
	Priority string `json:"priority,omitempty"`
}

// baziDayunJudgment is one model-owned reading of a deterministic luck-period
// fact. The renderer turns it into Markdown but never derives a trend itself.
type baziDayunJudgment struct {
	GanZhi         string   `json:"gan_zhi"`
	Trend          string   `json:"trend"`
	Interpretation string   `json:"interpretation"`
	Evidence       []string `json:"evidence,omitempty"`
}

type baziStaticSynthesis struct {
	Source         string `json:"source,omitempty"`
	RecoveryReason string `json:"recovery_reason,omitempty"`
	// FieldAudit records runtime-only local wording repairs. It is neither
	// rendered nor returned to another model as part of the judgment.
	FieldAudit              []string             `json:"-"`
	RuleProfile             string               `json:"rule_profile"`
	MainAxis                string               `json:"main_axis"`
	ClaimStrength           string               `json:"claim_strength"`
	SupportLevel            string               `json:"support_level"`
	LimitationLevel         string               `json:"limitation_level"`
	WordingCap              string               `json:"wording_cap"`
	ConsistencyFlags        []string             `json:"consistency_flags"`
	AxisLevel               string               `json:"axis_level"`
	EffectOnTiaohou         string               `json:"effect_on_tiaohou"`
	EffectOnCoreDisease     string               `json:"effect_on_core_disease"`
	EffectOnJiShenDirection string               `json:"effect_on_jishen_direction"`
	AxisCeiling             string               `json:"axis_ceiling"`
	ConflictReasons         []string             `json:"conflict_reasons"`
	PatternBasis            string               `json:"pattern_basis"`
	PatternOutcome          string               `json:"pattern_outcome"`
	CounterEvidence         string               `json:"counter_evidence"`
	AxisConsistency         string               `json:"axis_consistency"`
	TiaohouConstraint       string               `json:"tiaohou_constraint"`
	TiaohouAnchor           string               `json:"tiaohou_anchor"`
	StrengthBalance         string               `json:"strength_balance"`
	Strength                baziStrengthJudgment `json:"strength,omitempty"`
	Usage                   baziUsageLayers      `json:"usage,omitempty"`
	PatternAndQingZhuo      string               `json:"pattern_and_qing_zhuo"`
	QiShiOrCongHua          string               `json:"qishi_or_conghua"`
	TierJudgment            string               `json:"tier_judgment"`
	TierBasis               string               `json:"tier_basis"`
	ReasoningSummary        string               `json:"reasoning_summary"`
	ReasoningSteps          []string             `json:"reasoning_steps"`
	TopicDirectAnswer       string               `json:"topic_direct_answer,omitempty"`
	TopicFocusAnswer        string               `json:"topic_focus_answer,omitempty"`
	Advantages              []string             `json:"advantages"`
	Risks                   []string             `json:"risks"`
	Citations               []baziCitation       `json:"citations"`
	Assertions              []baziAssertion      `json:"assertions,omitempty"`
}

type baziDynamicSynthesis struct {
	Source         string `json:"source,omitempty"`
	RecoveryReason string `json:"recovery_reason,omitempty"`
	// FieldAudit records runtime-only local wording repairs.
	FieldAudit       []string            `json:"-"`
	CurrentTrend     string              `json:"current_trend"`
	ClaimStrength    string              `json:"claim_strength"`
	SupportLevel     string              `json:"support_level"`
	LimitationLevel  string              `json:"limitation_level"`
	WordingCap       string              `json:"wording_cap"`
	ConsistencyFlags []string            `json:"consistency_flags"`
	DayunPath        []string            `json:"dayun_path"`
	DayunJudgments   []baziDayunJudgment `json:"dayun_judgments,omitempty"`
	// CurrentDayunIndex identifies the current period in the chronologically
	// ordered DayunPath. It prevents validators from treating the first period
	// as the current period when the path includes the full life sequence.
	CurrentDayunIndex int             `json:"current_dayun_index"`
	LiunianFocus      string          `json:"liunian_focus"`
	WindowLevel       string          `json:"window_level"`
	TriggerSignals    []string        `json:"trigger_signals"`
	KeyWindows        []string        `json:"key_windows"`
	Risks             []string        `json:"risks"`
	ReasoningSummary  string          `json:"reasoning_summary"`
	ReasoningSteps    []string        `json:"reasoning_steps"`
	Assertions        []baziAssertion `json:"assertions,omitempty"`
}

type baziCharterState struct {
	AnalysisPlan     baziAnalysisPlan     `json:"analysis_plan"`
	Input            baziCharterInput     `json:"input"`
	EvidencePlan     baziEvidencePlan     `json:"evidence_plan"`
	EvidenceBundle   baziEvidenceBundle   `json:"evidence_bundle"`
	EvidenceQuality  baziEvidenceQuality  `json:"evidence_quality"`
	StaticSynthesis  baziStaticSynthesis  `json:"static_synthesis"`
	DynamicSynthesis baziDynamicSynthesis `json:"dynamic_synthesis"`
	FieldAudit       []string             `json:"-"`
}
