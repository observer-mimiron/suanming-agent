// Package presentation 包含八字已验收结果的用户可见投影。
//
// 本文件只定义最终渲染所需的窄输入 DTO；不读取 runtime、SessionState、模型、检索、trace 或 SSE。
package presentation

// FinalReplyInput 是最终 Markdown 渲染的最小输入，不携带 Graph 或外部依赖对象。
type FinalReplyInput struct {
	AnalysisPlan              AnalysisPlan
	Facts                     ChartFacts
	EvidenceBundle            EvidenceBundle
	StaticSynthesis           StaticSynthesis
	LifetimeSynthesis         LifetimeDayunSynthesis
	DynamicSynthesis          DynamicSynthesis
	FactsOnlyDynamicSynthesis DynamicSynthesis
}

// AnalysisPlan 只保留影响最终章节选择的计划字段。
type AnalysisPlan struct {
	NeedLifetimeDayun bool
	WriterTemplate    string
	TopicMode         string
}

// ChartFacts 是 runtime 已整理的确定性展示事实，不暴露原始工具载荷。
type ChartFacts struct {
	PillarsSummary         string
	DayMaster              string
	StrengthEvidence       string
	PatternSummary         string
	LiunianGanZhi          string
	LiunianTenGod          string
	CurrentDayunGanZhi     string
	LiunianRelations       []string
	DayunPeriods           []DayunPeriod
	RuleProfileID          string
	SubjectAgeBand         string
	MonthCommand           string
	OfficialVisible        bool
	OfficialHidden         bool
	FireEffectivenessKnown bool
}

// DayunPeriod 是最终文本需要的大运目录项；Label 只保留干支与起止年龄。
type DayunPeriod struct {
	Ref    string
	Label  string
	GanZhi string
	TenGod string
}

// EvidenceBundle 只保留可读的古籍短引文。
type EvidenceBundle struct {
	Citations []Citation
}

// Citation 是一条可展示的古籍引用。
type Citation struct {
	Classic string
	Quotes  []string
}

// StrengthJudgment 是已通过合同的强弱结论与依据。
type StrengthJudgment struct {
	Conclusion string
	Reasoning  string
}

// UsageLayers 分开扶抑和调候两个取用视角，避免展示层合并语义。
type UsageLayers struct {
	Fuyi    string
	Tiaohou string
}

// StaticSynthesis 是已通过静态合同的展示槽位。
type StaticSynthesis struct {
	FactsOnly         bool
	RecoveryReason    string
	RuleProfile       string
	MainAxis          string
	AxisConsistency   string
	PatternOutcome    string
	CounterEvidence   string
	TiaohouConstraint string
	TiaohouAnchor     string
	StrengthBalance   string
	Strength          StrengthJudgment
	Usage             UsageLayers
	TierStatus        string
	TierJudgment      string
	TierBasis         string
	ReasoningSummary  string
	TopicDirectAnswer string
	TopicFocusAnswer  string
	Advantages        []string
	Risks             []string
}

// DynamicSynthesis 是已通过动态合同或事实降级合同的展示槽位。
type DynamicSynthesis struct {
	FactsOnly                bool
	RecoveryReason           string
	CurrentTrend             string
	CurrentPeriodRealization string
	ConsistencyFlags         []string
	DayunPath                []string
	CurrentDayunIndex        int
	LiunianFocus             string
	WindowLevel              string
	TriggerSignals           []string
	Risks                    []string
	ReasoningSummary         string
	ReasoningSteps           []string
	OutcomeDomains           []string
}

// LifetimeDayunSynthesis 是全程大运章节的独立展示槽位。
type LifetimeDayunSynthesis struct {
	Status       string
	Trajectory   string
	PeriodClaims []LifetimeDayunClaim
	Summary      string
}

// LifetimeDayunClaim 是一条已接受的大运全程判断。
type LifetimeDayunClaim struct {
	PeriodRef    string
	PeriodEffect string
}

// DayunJudgment 保留旧 renderer 辅助函数所需的最小大运判断形状。
type DayunJudgment struct {
	GanZhi         string
	Trend          string
	Interpretation string
	Evidence       []string
	OutcomeDomains []string
}
