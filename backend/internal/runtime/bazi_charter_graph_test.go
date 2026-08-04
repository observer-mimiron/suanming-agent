// This test file belongs to the manager-owned runtime layer.
// It verifies BaZi charter graph execution and protects the related contract from regressions.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/prompts"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

func TestBaziCharterPrompts_AreEmbedded(t *testing.T) {
	cases := map[string]string{
		"constitution":      prompts.BaziConstitutionInstruction,
		"analysis_planner":  prompts.BaziAnalysisPlannerInstruction,
		"evidence_planner":  prompts.BaziEvidencePlannerInstruction,
		"static_synthesis":  prompts.BaziStaticSynthesisInstruction,
		"dynamic_synthesis": prompts.BaziDynamicSynthesisInstruction,
	}

	for name, content := range cases {
		if strings.TrimSpace(content) == "" {
			t.Fatalf("%s prompt should not be empty", name)
		}
	}
}

func TestBaziEvidencePlannerPrompt_RequiresCaseGuidance(t *testing.T) {
	content := prompts.BaziEvidencePlannerInstruction
	if !strings.Contains(content, "命例") && !strings.Contains(content, "举例") {
		t.Fatalf("evidence planner prompt must require case/example retrieval guidance")
	}
}

func TestBaziCharterState_SeparatesStaticAndDynamicStages(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:           "杀印相生为主轴",
			PatternBasis:       "月令取格以官杀为先，七杀透干，印星贴身承接。",
			PatternOutcome:     "主轴可立，但带病，需靠印星救应。",
			CounterEvidence:    "虽可立杀印主轴，但财星干扰承接，故不能拔高为纯贵格。",
			AxisConsistency:    "虽见杀重，但印路受财阻，故仍以杀印承接为主轴而非另起从杀之论。",
			TiaohouAnchor:      "乙木生申月，先按金旺木受制的月令口径审寒燥与润养需求。",
			PatternAndQingZhuo: "清中有制，主轴可立",
			TierJudgment:       "中上",
			TierBasis:          "主轴可立，清中有制，虽有病但仍有救应，故可定中上，不宜再拔高为上等。",
			ReasoningSummary:   "月令先定七杀主轴，再看印星贴身承接，故主轴可立；但仍需说明病药救应如何化解压力。",
			ReasoningSteps: []string{
				"先看月令司令与透干，确认七杀主气已经出头。",
				"再看印星是否贴身承接，判断杀印相生能否成立。",
			},
		},
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend: "前运多压，后运渐开",
			DayunPath: []string{
				"前一步运偏压，更多放大七杀压力。",
				"当前运开始转缓，印星承接力增强。",
				"后一步运更利主轴兑现，可视作发力窗口。",
			},
			LiunianFocus:     "当前流年重点应在事业压力与节奏调整，属于官杀触发而非主轴全破。",
			WindowLevel:      "窗口年",
			ReasoningSummary: "当前大运先压后托，流年触发更多落在事业压力与节奏调整上，因此属于阶段性受压而非主轴全破。",
			ReasoningSteps: []string{
				"先看当前大运对七杀主轴偏压。",
				"再看后续流年是否有印星承接，故判断为先压后托。",
			},
		},
	}

	if state.StaticSynthesis.MainAxis == "" {
		t.Fatalf("expected static final axis")
	}
	if state.StaticSynthesis.PatternBasis == "" {
		t.Fatalf("expected pattern basis")
	}
	if state.StaticSynthesis.CounterEvidence == "" {
		t.Fatalf("expected static counter evidence")
	}
	if state.StaticSynthesis.AxisConsistency == "" {
		t.Fatalf("expected static axis consistency")
	}
	if state.StaticSynthesis.TiaohouAnchor == "" {
		t.Fatalf("expected static tiaohou anchor")
	}
	if state.StaticSynthesis.ReasoningSummary == "" {
		t.Fatalf("expected static reasoning summary")
	}
	if state.StaticSynthesis.TierJudgment == "" || state.StaticSynthesis.TierBasis == "" {
		t.Fatalf("expected static tier judgment and basis")
	}
	if len(state.StaticSynthesis.ReasoningSteps) == 0 {
		t.Fatalf("expected static reasoning steps")
	}
	if state.DynamicSynthesis.CurrentTrend == "" {
		t.Fatalf("expected dynamic summary")
	}
	if state.DynamicSynthesis.ReasoningSummary == "" {
		t.Fatalf("expected dynamic reasoning summary")
	}
	if state.DynamicSynthesis.WindowLevel == "" {
		t.Fatalf("expected dynamic window level")
	}
	if len(state.DynamicSynthesis.DayunPath) == 0 || state.DynamicSynthesis.LiunianFocus == "" {
		t.Fatalf("expected dynamic dayun path and liunian focus")
	}
	if len(state.DynamicSynthesis.ReasoningSteps) == 0 {
		t.Fatalf("expected dynamic reasoning steps")
	}
}

func TestBaziCharterPrompts_ContainRoleBoundaries(t *testing.T) {
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "不得另起第二主轴") {
		t.Fatalf("static synthesis prompt must forbid second axis")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "必须单独交代“取格依据”") {
		t.Fatalf("static synthesis prompt must require pattern basis")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "## 任务类型") {
		t.Fatalf("static synthesis prompt must define task type")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "## 输入事实优先级") {
		t.Fatalf("static synthesis prompt must define input priority")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "## 成功标准") {
		t.Fatalf("static synthesis prompt must define success criteria")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "## DO / DON'T") {
		t.Fatalf("static synthesis prompt must define do/don't constraints")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "扶抑与调候两个维度") {
		t.Fatalf("static synthesis prompt must separate fuyi and tiaohou dimensions")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`counter_evidence`") {
		t.Fatalf("static synthesis prompt must require counter evidence")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`axis_consistency`") {
		t.Fatalf("static synthesis prompt must require axis consistency")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`tiaohou_anchor`") {
		t.Fatalf("static synthesis prompt must require tiaohou anchor")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`reasoning_summary`") {
		t.Fatalf("static synthesis prompt must require reasoning summary")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`reasoning_steps`") {
		t.Fatalf("static synthesis prompt must require reasoning steps")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`topic_direct_answer`") {
		t.Fatalf("static synthesis prompt must require topic direct answer for topic follow-up")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`topic_focus_answer`") {
		t.Fatalf("static synthesis prompt must require topic focus answer for topic follow-up")
	}
	if !strings.Contains(prompts.BaziConstitutionInstruction, "writer 无权改写上游综合结论") {
		t.Fatalf("constitution prompt must forbid writer from changing conclusions")
	}
	if !strings.Contains(prompts.BaziDynamicSynthesisInstruction, "不得回头推翻静态主轴") {
		t.Fatalf("dynamic synthesis prompt must separate static and dynamic layers")
	}
	if !strings.Contains(prompts.BaziDynamicSynthesisInstruction, "evidence_quality.missing_topics") || !strings.Contains(prompts.BaziDynamicSynthesisInstruction, "社会地位") {
		t.Fatalf("dynamic synthesis prompt must bind evidence gaps to authorized outcome domains")
	}
}

func TestBaziCharterPrompts_ContainMethodologyOrder(t *testing.T) {
	if !strings.Contains(prompts.BaziConstitutionInstruction, "以子平格局法为主轴") {
		t.Fatalf("constitution prompt must define the main methodology axis")
	}
	if !strings.Contains(prompts.BaziDynamicSynthesisInstruction, "先看当前大运是在承托、放大、压制还是扭转静态主轴") {
		t.Fatalf("dynamic synthesis prompt must define the dynamic evaluation order")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "先把 `chart_facts`") {
		t.Fatalf("static synthesis prompt must define the synthesis order")
	}
	if !strings.Contains(prompts.BaziDynamicSynthesisInstruction, "`reasoning_summary`") {
		t.Fatalf("dynamic synthesis prompt must require reasoning summary")
	}
	if !strings.Contains(prompts.BaziDynamicSynthesisInstruction, "`window_level`") {
		t.Fatalf("dynamic synthesis prompt must require window level")
	}
	if !strings.Contains(prompts.BaziDynamicSynthesisInstruction, "不得使用并不存在的术语") {
		t.Fatalf("dynamic synthesis prompt must forbid invalid terminology")
	}
	if !strings.Contains(prompts.BaziDynamicSynthesisInstruction, "`reasoning_steps`") {
		t.Fatalf("dynamic synthesis prompt must require reasoning steps")
	}
	if !strings.Contains(prompts.BaziAnalysisPlannerInstruction, "`topic_mode`") {
		t.Fatalf("analysis planner prompt must require topic mode")
	}
}

func TestBaziCharterPrompts_ContainAxisVerdictContract(t *testing.T) {
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`axis_level`") {
		t.Fatalf("static synthesis prompt must require axis level")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`effect_on_tiaohou`") {
		t.Fatalf("static synthesis prompt must require tiaohou effect")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`effect_on_core_disease`") {
		t.Fatalf("static synthesis prompt must require core disease effect")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`effect_on_jishen_direction`") {
		t.Fatalf("static synthesis prompt must require ji-shen direction effect")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`axis_ceiling`") {
		t.Fatalf("static synthesis prompt must require axis ceiling")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "`conflict_reasons`") {
		t.Fatalf("static synthesis prompt must require conflict reasons")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "结构存在 ≠ 主轴成立 ≠ 可以拔高") {
		t.Fatalf("static synthesis prompt must explain axis promotion ladder")
	}
	if !strings.Contains(prompts.BaziStaticSynthesisInstruction, "若候选路线继续放大忌神、病点或调候冲突") {
		t.Fatalf("static synthesis prompt must define conflict-based downgrade rule")
	}
	if !strings.Contains(prompts.BaziMethodologyCharterInstruction, "结构存在 ≠ 主轴成立 ≠ 可以拔高") {
		t.Fatalf("methodology charter must define axis promotion ladder")
	}
	if !strings.Contains(prompts.BaziMethodologyCharterInstruction, "若候选主轴继续放大忌神、核心病点或调候冲突") {
		t.Fatalf("methodology charter must define reusable downgrade rule")
	}
}

func TestBuildEphemeralInnerAgent_ReusesSessionContextInjection(t *testing.T) {
	builder := &AgentBuilder{}
	cfg := specialists.Config{
		Domain:      "bazi",
		Name:        "static_synthesis",
		Description: "静态综合器",
		Instruction: "{{SESSION_CONTEXT}}\n只输出 JSON",
		ToolNames:   []string{},
	}
	st := &state.SessionState{
		BaziResult: map[string]any{"dayGan": "甲"},
	}

	_, err := builder.BuildEphemeralInnerAgent(context.Background(), cfg, st)
	if err != nil {
		t.Fatalf("BuildEphemeralInnerAgent returned error: %v", err)
	}
}

func TestValidateStaticStage_RequiresStaticSynthesis(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis: "杀印相生",
		},
	}

	if err := validateStaticStage(state); err == nil {
		t.Fatalf("expected missing pattern basis/outcome and qingzhuo to fail validation")
	}
}

func TestDynamicLayer_MustWaitForStaticSynthesis(t *testing.T) {
	state := baziCharterState{}
	if err := validateDynamicPreconditions(state); err == nil {
		t.Fatalf("expected missing static synthesis to block dynamic evaluation")
	}
}

func TestValidateDynamicStage_RequiresReasoningSummary(t *testing.T) {
	state := baziCharterState{
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "当前岁运先压后托",
			DayunPath:        []string{"当前运先压后托。"},
			LiunianFocus:     "当前流年重点落在事业压力。",
			WindowLevel:      "承压年",
			ReasoningSummary: "先看大运受压，再看后续流年承接。",
		},
	}
	if err := validateDynamicStage(state); err == nil {
		t.Fatalf("expected missing dynamic reasoning steps to fail validation")
	}
}

func TestShouldUseBaziCharterGraph_PureBaziOnly(t *testing.T) {
	if !shouldUseBaziCharterGraph(ExecutionPlan{
		Route:        policy.ApprovedRoute{PrimaryDomain: "bazi"},
		Domains:      []string{"bazi"},
		FollowupMode: followupModeRerunSpecialist,
	}) {
		t.Fatalf("expected pure bazi route to use inner charter graph")
	}
	if shouldUseBaziCharterGraph(ExecutionPlan{
		Route:        policy.ApprovedRoute{PrimaryDomain: "bazi"},
		Domains:      []string{"bazi", "ziwei"},
		FollowupMode: followupModeRerunSpecialist,
	}) {
		t.Fatalf("mixed-domain route must stay on supervisor path")
	}
	if shouldUseBaziCharterGraph(ExecutionPlan{
		Route:        policy.ApprovedRoute{PrimaryDomain: "bazi"},
		Domains:      []string{"bazi"},
		FollowupMode: followupModeDirect,
	}) {
		t.Fatalf("direct follow-up must not re-enter inner charter graph")
	}
	if shouldUseBaziCharterGraph(ExecutionPlan{
		Route:        policy.ApprovedRoute{PrimaryDomain: "bazi"},
		Domains:      []string{"bazi"},
		FollowupMode: followupModeReuseArtifact,
	}) {
		t.Fatalf("reused-artifact follow-up must not re-enter inner charter graph")
	}
}

func TestBaziAuthorityGraph_StaticStageAllowsEmptyEvidenceBundle(t *testing.T) {
	state := baziCharterState{
		EvidencePlan: baziEvidencePlan{
			NeedRetrieval: true,
			QueryPackets: []baziQueryPacket{
				{Topic: "geju", Query: "子平真诠 格局 月令 取格"},
			},
		},
	}
	if err := validateEvidenceBundlePreconditions(state); err != nil {
		t.Fatalf("empty evidence bundle should degrade instead of blocking static lenses: %v", err)
	}
}

func TestBaziAuthorityGraph_RetrievalRequiredStateStillNeedsQueryPackets(t *testing.T) {
	state := baziCharterState{
		EvidencePlan: baziEvidencePlan{NeedRetrieval: true},
	}
	if err := validateEvidenceBundlePreconditions(state); err == nil {
		t.Fatalf("expected retrieval-required state without query packets to fail")
	}
}

func TestDefaultBaziEvidencePlan_StaticStageIncludesCaseValidationQuery(t *testing.T) {
	plan := defaultBaziEvidencePlan("", baziAnalysisPlan{RetrievalStage: "static"}, baziCharterInput{
		BaziResult: map[string]any{
			"dayGan":  "甲",
			"pillars": []map[string]any{{"name": "年柱", "stem": "癸", "branch": "酉"}, {"name": "月柱", "stem": "乙", "branch": "亥"}},
		},
	})

	if !containsString(plan.EvidenceGaps, "同结构命例校验补证") {
		t.Fatalf("static evidence gaps must include case validation gap, got %v", plan.EvidenceGaps)
	}

	hasTheoryGeju := false
	hasTheoryTiaohou := false
	hasTheoryBingyao := false
	hasCaseValidation := false

	for _, packet := range plan.QueryPackets {
		switch {
		case packet.Topic == "geju" && strings.Contains(packet.Query, "格局"):
			hasTheoryGeju = true
		case packet.Topic == "tiaohou" && strings.Contains(packet.Query, "调候"):
			if !strings.Contains(packet.Query, "甲木") || !strings.Contains(packet.Query, "亥月") {
				t.Fatalf("tiaohou query must include chart-specific day/month terms, got %q", packet.Query)
			}
			hasTheoryTiaohou = true
		case packet.Topic == "bingyao" && strings.Contains(packet.Query, "病药"):
			hasTheoryBingyao = true
		}

		if strings.Contains(packet.Query, "命例") || strings.Contains(packet.Query, "举例") {
			hasCaseValidation = true
			if packet.Topic != "geju" {
				t.Fatalf("case validation query topic = %q, want geju", packet.Topic)
			}
			if packet.SourceTier != "B" {
				t.Fatalf("case validation source tier = %q, want B", packet.SourceTier)
			}
			if !containsString(packet.PreferredSources, "子平真诠") {
				t.Fatalf("case validation query must prefer 子平真诠, got %v", packet.PreferredSources)
			}
		}
	}

	if !hasTheoryGeju {
		t.Fatalf("expected geju theory query in static evidence plan")
	}
	if !hasTheoryTiaohou {
		t.Fatalf("expected tiaohou theory query in static evidence plan")
	}
	if !hasTheoryBingyao {
		t.Fatalf("expected bingyao theory query in static evidence plan")
	}
	if !hasCaseValidation {
		t.Fatalf("expected case validation query in static evidence plan")
	}
}

func TestBaziCharterEvaluators_HaveNoRetrievalTools(t *testing.T) {
	configs := []specialists.Config{
		baziAnalysisPlannerConfig(),
		baziStaticSynthesisConfig(),
		baziDynamicSynthesisConfig(),
	}
	for _, cfg := range configs {
		if len(cfg.ToolNames) != 0 {
			t.Fatalf("%s must not have retrieval tools", cfg.Name)
		}
	}
}

func TestBaziCharterLightweightNodes_UseFastModel(t *testing.T) {
	if !baziAnalysisPlannerConfig().UseFastModel {
		t.Fatalf("analysis planner should use fast model")
	}
	if !baziEvidencePlannerConfig().UseFastModel {
		t.Fatalf("evidence planner should use fast model")
	}
	if baziStaticSynthesisConfig().UseFastModel {
		t.Fatalf("static synthesis should stay on main model")
	}
	if baziDynamicSynthesisConfig().UseFastModel {
		t.Fatalf("dynamic synthesis should stay on main model")
	}
}

func TestBaziCharterInnerNodes_DisableSessionContextInjection(t *testing.T) {
	configs := []specialists.Config{
		baziAnalysisPlannerConfig(),
		baziEvidencePlannerConfig(),
		baziStaticSynthesisConfig(),
		baziDynamicSynthesisConfig(),
	}

	for _, cfg := range configs {
		if cfg.InjectSessionContext {
			t.Fatalf("%s should disable automatic session context injection", cfg.Name)
		}
	}
}

// TestBuildBaziSubjectContext_InfantAtTargetYear locks the age-aware dynamic
// scope used to prevent adult life-domain projections for young children.
func TestBuildBaziSubjectContext_InfantAtTargetYear(t *testing.T) {
	context := buildBaziSubjectContext(baziCharterInput{
		BaziResult: map[string]any{"birthday": "2025-11-11 00:15"},
		Liunian:    map[string]any{"liunian_year": 2026},
	})
	if context.BirthYear != 2025 || context.TargetYear != 2026 || context.Age != 1 || context.AgeBand != "infant" {
		t.Fatalf("unexpected subject context: %+v", context)
	}
	if containsString(context.AllowedOutcomeDomains, "user_requested_authorized_domain") || !containsString(context.AllowedOutcomeDomains, "growth_environment") {
		t.Fatalf("unexpected infant outcome authorization: %+v", context.AllowedOutcomeDomains)
	}
}

// TestValidateDynamicOutcomeDomains_RejectsAdultDomainForInfant validates the
// structured age scope without relying on a growing prose blacklist.
func TestValidateDynamicOutcomeDomains_RejectsAdultDomainForInfant(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			BaziResult: map[string]any{"birthday": "2025-11-11 00:15"},
			Liunian:    map[string]any{"liunian_year": 2026},
		},
		DynamicSynthesis: baziDynamicSynthesis{OutcomeDomains: []string{"user_requested_authorized_domain"}},
	}
	if err := validateDynamicOutcomeDomains(state); err == nil {
		t.Fatal("infant dynamic synthesis must reject adult-authorized domains")
	}
}

// TestValidateDynamicOutcomeDomains_RejectsAdultDomainTextForInfant prevents a
// model from declaring an allowed minor domain while writing adult outcomes in
// the visible text.
func TestValidateDynamicOutcomeDomains_RejectsAdultDomainTextForInfant(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			BaziResult: map[string]any{"birthday": "2025-11-11 00:15"},
			Liunian:    map[string]any{"liunian_year": 2026},
		},
		DynamicSynthesis: baziDynamicSynthesis{
			OutcomeDomains: []string{"structure", "growth_environment"},
			DayunJudgments: []baziDayunJudgment{{
				GanZhi:         "丙戌",
				Trend:          "结构观察",
				Interpretation: "这步运会带来事业压力与财富增长。",
				OutcomeDomains: []string{"structure"},
			}},
		},
	}
	assertBaziViolationCode(t, validateDynamicOutcomeDomains(state), baziViolationUnsupportedConcreteOutcome)
}

func TestValidateDynamicOutcomeDomains_AllowsMinorGrowthDomainText(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			BaziResult: map[string]any{"birthday": "2025-11-11 00:15"},
			Liunian:    map[string]any{"liunian_year": 2026},
		},
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:   "只作成长节奏观察。",
			OutcomeDomains: []string{"structure", "growth_environment"},
			DayunJudgments: []baziDayunJudgment{{
				GanZhi:         "丙戌",
				Trend:          "结构观察",
				Interpretation: "更多体现在成长环境变化、照护节奏和可观察发展上。",
				OutcomeDomains: []string{"structure", "growth_environment"},
			}},
		},
	}
	if err := validateDynamicOutcomeDomains(state); err != nil {
		t.Fatalf("minor growth-domain wording should pass: %v", err)
	}
}

func TestValidateStaticOutcomeScope_RejectsAdultDomainTextForInfant(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			BaziResult: map[string]any{"birthday": "2025-11-11 00:15"},
			Liunian:    map[string]any{"liunian_year": 2026},
		},
		StaticSynthesis: func() baziStaticSynthesis {
			out := validStaticSynthesisForConsistencyTests()
			out.Risks = []string{"事业竞争压力较大，需注意健康。"}
			return out
		}(),
	}
	assertBaziViolationCode(t, validateStaticOutcomeScope(state), baziViolationUnsupportedConcreteOutcome)
}

func TestValidateStaticOutcomeScope_AllowsMinorGrowthText(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			BaziResult: map[string]any{"birthday": "2025-11-11 00:15"},
			Liunian:    map[string]any{"liunian_year": 2026},
		},
		StaticSynthesis: func() baziStaticSynthesis {
			out := validStaticSynthesisForConsistencyTests()
			out.Risks = []string{"成长环境变化较多，照护节奏宜保持稳定。"}
			out.Advantages = []string{"结构层面有承接，适合观察可见发展节奏。"}
			return out
		}(),
	}
	if err := validateStaticOutcomeScope(state); err != nil {
		t.Fatalf("minor growth-domain static wording should pass: %v", err)
	}
}

// TestBuildDynamicSynthesisFeedback_RepeatsAgeAndEvidenceBoundaries keeps the
// retry prompt aligned with the independent contract audit instead of relying on
// final rendering to hide unauthorized adult-domain projections.
func TestBuildDynamicSynthesisFeedback_RepeatsAgeAndEvidenceBoundaries(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			BaziResult: map[string]any{"birthday": "2025-11-11 00:15"},
			Liunian:    map[string]any{"liunian_year": 2026},
		},
		EvidenceQuality: baziEvidenceQuality{Enough: false, MissingTopics: []string{"bingyao"}},
	}

	feedback := buildDynamicSynthesisFeedback(state, baziDynamicSynthesis{OutcomeDomains: []string{"structure"}}, nil)
	for _, want := range []string{"allowed_outcome_domains", "社会地位", "权威", "missing_topics=bingyao", "structure 结构观察"} {
		if !strings.Contains(feedback, want) {
			t.Fatalf("dynamic feedback missing %q in %s", want, feedback)
		}
	}
}

// TestValidateStaticEvidenceCoverageBoundaryCapsMissingTopics prevents a
// partial retrieval bundle from authorizing a high-confidence axis tier.
func TestValidateStaticEvidenceCoverageBoundaryCapsMissingTopics(t *testing.T) {
	state := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{MissingTopics: []string{"bingyao"}},
		StaticSynthesis: baziStaticSynthesis{AxisLevel: "可以拔高"},
	}
	if err := validateStaticEvidenceCoverageBoundary(state); err == nil {
		t.Fatal("missing critical evidence must cap axis level")
	}
}

func TestValidateStaticTiaohouEvidenceWordingRejectsMissingClaimWhenCovered(t *testing.T) {
	state := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{CoveredTopics: []string{"tiaohou"}},
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
	}
	state.StaticSynthesis.TiaohouConstraint = "调候用神未定，需穷通宝鉴等证据。"
	if err := validateStaticTiaohouEvidenceWording(state); err == nil {
		t.Fatal("covered tiaohou evidence must reject missing-evidence wording")
	}
}

func TestValidateStaticTiaohouEvidenceWordingRequiresConcreteAnchorWhenCovered(t *testing.T) {
	state := baziCharterState{
		EvidenceQuality: baziEvidenceQuality{CoveredTopics: []string{"tiaohou"}},
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
	}
	state.StaticSynthesis.TiaohouAnchor = "调候为环境约束，不直接决定格局成败。"
	if err := validateStaticTiaohouEvidenceWording(state); err == nil {
		t.Fatal("covered tiaohou evidence must require a concrete anchor verdict")
	}
	state.StaticSynthesis.TiaohouAnchor = "甲木生亥月，寒湿重，调候火有但弱，年支巳火被亥冲而受损。"
	if err := validateStaticTiaohouEvidenceWording(state); err != nil {
		t.Fatalf("concrete tiaohou anchor should pass: %v", err)
	}
}

func TestBuildTiaohouConclusionUsesAnchorBeforeBoundary(t *testing.T) {
	state := baziCharterState{StaticSynthesis: baziStaticSynthesis{
		TiaohouAnchor:     "甲木生亥月，寒湿重，调候火有但弱。",
		TiaohouConstraint: "调候为环境约束，不直接决定格局成败。",
	}}
	if got := buildTiaohouConclusion(state); got != "甲木生亥月，寒湿重，调候火有但弱。" {
		t.Fatalf("tiaohou conclusion = %q", got)
	}
}

func TestBuildStaticSynthesisPayload_UsesSlimViews(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			UserQuestion: "看看整体命局",
			BaziResult: map[string]any{
				"pillars":            []map[string]any{{"stem": "甲", "branch": "子"}},
				"dayGan":             "甲",
				"geju":               "正印格",
				"dayun_analyzed":     map[string]any{"periods": []any{"raw"}},
				"liunian":            map[string]any{"liunian_year": 2026},
				"shi_shen_power":     map[string]any{"印": map[string]any{"weighted": 4.2}},
				"extra_large_block":  strings.Repeat("x", 128),
				"another_extra_fact": "should-not-leak",
			},
			Yongshen: map[string]any{
				"yong_shen": []any{"火"},
				"xi_shen":   []any{"土"},
				"ji_shen":   []any{"水"},
			},
			Dayun: map[string]any{
				"periods": []map[string]any{{"ganzhi": "辛卯"}},
			},
			Liunian: map[string]any{
				"liunian_year":   2026,
				"liunian_ganzhi": "丙午",
			},
		},
		AnalysisPlan: baziAnalysisPlan{
			Mode:           "static_full",
			RetrievalStage: "static",
			NeedDynamic:    true,
			WriterTemplate: "full",
		},
		EvidencePlan: baziEvidencePlan{
			Stage: "static",
		},
		EvidenceBundle: baziEvidenceBundle{
			Stage: "static",
			Citations: []baziCitation{
				{Classic: "子平真诠", Quotes: []string{"月令为提纲"}},
			},
		},
		EvidenceQuality: baziEvidenceQuality{
			Enough: true,
		},
	}

	payload := buildStaticSynthesisPayload(state)
	rawInput, ok := payload["input"].(map[string]any)
	if !ok {
		t.Fatalf("payload input type = %T, want map[string]any", payload["input"])
	}
	if _, exists := rawInput["bazi_result"]; exists {
		t.Fatalf("static payload should not carry full raw bazi_result")
	}
	if _, exists := rawInput["dayun"]; exists {
		t.Fatalf("static payload should not carry raw dayun block")
	}
	if _, exists := rawInput["liunian"]; exists {
		t.Fatalf("static payload should not carry raw liunian block")
	}
	if _, exists := rawInput["subject_context"]; !exists {
		t.Fatalf("static payload must carry age-aware subject_context")
	}

	core, ok := rawInput["core_chart"].(map[string]any)
	if !ok {
		t.Fatalf("payload input must contain core_chart view")
	}
	if core["day_master"] != "甲" {
		t.Fatalf("core_chart day_master = %v, want 甲", core["day_master"])
	}
	if _, exists := core["another_extra_fact"]; exists {
		t.Fatalf("core_chart should not leak unrelated raw facts")
	}
}

func TestBuildDynamicSynthesisPayload_UsesSlimViews(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			BaziResult: map[string]any{
				"pillars": []map[string]any{
					{"stem": "甲", "branch": "子"},
					{"stem": "戊", "branch": "子"},
				},
				"dayGan":            "甲",
				"extra_large_block": strings.Repeat("y", 128),
			},
			Yongshen: map[string]any{
				"geju":      "正印格",
				"tiao_hou":  "火为先",
				"yong_shen": []any{"火"},
			},
			Dayun: map[string]any{
				"dayun_analyzed": []map[string]any{
					{"ganZhi": "辛卯", "quality": "偏吉"},
				},
				"raw_dump": strings.Repeat("d", 64),
			},
			Liunian: map[string]any{
				"liunian_year":   2026,
				"liunian_ganzhi": "丙午",
				"raw_dump":       strings.Repeat("l", 64),
			},
		},
		AnalysisPlan: baziAnalysisPlan{
			Mode:           "static_full",
			RetrievalStage: "static",
			NeedDynamic:    true,
			WriterTemplate: "full",
		},
		EvidenceBundle: baziEvidenceBundle{
			Stage: "dynamic",
			Citations: []baziCitation{
				{Classic: "三命通会", Quotes: []string{"岁运并临"}},
			},
		},
		EvidenceQuality: baziEvidenceQuality{Enough: true},
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
	}

	payload := buildDynamicSynthesisPayload(state)
	rawInput, ok := payload["input"].(map[string]any)
	if !ok {
		t.Fatalf("payload input type = %T, want map[string]any", payload["input"])
	}
	if _, exists := rawInput["bazi_result"]; exists {
		t.Fatalf("dynamic payload should not carry full raw bazi_result")
	}
	if _, exists := rawInput["dayun"]; exists {
		t.Fatalf("dynamic payload should not carry raw dayun block")
	}
	if _, exists := rawInput["liunian"]; exists {
		t.Fatalf("dynamic payload should not carry raw liunian block")
	}

	dynamicFacts, ok := rawInput["dynamic_facts"].(map[string]any)
	if !ok {
		t.Fatalf("payload input must contain dynamic_facts view")
	}
	if _, ok := dynamicFacts["dayun"].(map[string]any); !ok {
		t.Fatalf("dynamic_facts must include dayun view")
	}
	if _, ok := dynamicFacts["liunian"].(map[string]any); !ok {
		t.Fatalf("dynamic_facts must include liunian view")
	}
}

func TestRunFinalWriter_RendersFullTemplateWithoutModel(t *testing.T) {
	e := &Executor{}
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{
			WriterTemplate: "full",
		},
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "当前运势属于吉中有阻，机会伴随强变动。",
			ClaimStrength:    "倾向成立",
			SupportLevel:     "有气",
			LimitationLevel:  "明显",
			WordingCap:       "中性",
			ConsistencyFlags: []string{"吉中有阻", "机会伴随强变动"},
			DayunPath: []string{
				"当前大运能承接主轴，但仍会放大病点。",
				"后续阶段更适合稳步发力，不宜激进。",
			},
			LiunianFocus:     "这一年更像窗口年，适合顺势争取，但不宜激进。",
			WindowLevel:      "窗口年",
			TriggerSignals:   []string{"流年点亮用神", "同时引动原局冲扰"},
			KeyWindows:       []string{"当前阶段先稳后进"},
			Risks:            []string{"同辈竞争带来压力"},
			ReasoningSummary: "机会伴随强变动，整体属于吉中有阻。",
			ReasoningSteps: []string{
				"先看当前大运承接主轴，但并未把限制完全化开。",
				"再看流年触发点亮用神，同时冲扰也被引动。",
			},
		},
	}

	output, err := e.runFinalWriter(context.Background(), nil, state, "看看整体命局")
	if err != nil {
		t.Fatalf("runFinalWriter returned error: %v", err)
	}
	if err := validateFinalWriterOutput(state.AnalysisPlan, state, output); err != nil {
		t.Fatalf("deterministic final writer output should satisfy validation: %v", err)
	}
	for _, want := range []string{
		"## 总览结论",
		"## 强弱视角",
		"## 调候视角",
		"## 格局视角",
		"## 大运验证",
		"## 综合判定",
		"◎ 主轴",
		"▲ 限制",
		"◇ 读法",
		"**规则口径**",
		"**依据**",
		"**解释**",
		"**岁运兑现**",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("full template output missing %q:\n%s", want, output)
		}
	}
}

func TestRunFinalWriter_RendersTopicTemplateWithoutModel(t *testing.T) {
	e := &Executor{}
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{
			WriterTemplate: "topic",
			TopicMode:      "analysis",
		},
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:    "当前属于吉中有阻。",
			ClaimStrength:   "倾向成立",
			SupportLevel:    "有气",
			LimitationLevel: "明显",
			WordingCap:      "中性",
			LiunianFocus:    "今年宜稳中求进。",
			WindowLevel:     "窗口年",
		},
	}

	output, err := e.runFinalWriter(context.Background(), nil, state, "我想问事业")
	if err != nil {
		t.Fatalf("runFinalWriter returned error: %v", err)
	}
	if err := validateFinalWriterOutput(state.AnalysisPlan, state, output); err != nil {
		t.Fatalf("topic final writer output should satisfy validation: %v", err)
	}
}

func TestRunFinalWriter_RendersTopicFollowUpWithFocusAnswer(t *testing.T) {
	e := &Executor{}
	static := validStaticSynthesisForConsistencyTests()
	static.CounterEvidence = "财星透干，印路受阻，所以财星坏印之感会被放大。"
	static.AxisConsistency = "关键不在另起别格，而在原局财印牵制已明。"
	static.TopicDirectAnswer = "财星坏印被放大，重点不是另起主轴，而是原局财印牵制本来就存在。"
	static.TopicFocusAnswer = "这轮追问真正要看的是限制为何被看重，而不是回头另立一条格局主轴。"
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{
			WriterTemplate: "topic",
			TopicMode:      "analysis",
		},
		StaticSynthesis: static,
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:    "当前属于吉中有阻。",
			ClaimStrength:   "倾向成立",
			SupportLevel:    "有气",
			LimitationLevel: "明显",
			WordingCap:      "中性",
			LiunianFocus:    "今年宜稳中求进。",
			WindowLevel:     "窗口年",
		},
	}

	output, err := e.runFinalWriter(context.Background(), nil, state, "为什么会放大财星坏印的影响？")
	if err != nil {
		t.Fatalf("runFinalWriter returned error: %v", err)
	}
	if !strings.Contains(output, "**这次追问的关键**") {
		t.Fatalf("topic follow-up output should expose focused answer, got:\n%s", output)
	}
	if !strings.Contains(output, "财星坏印") {
		t.Fatalf("topic follow-up output should answer the asked controversy, got:\n%s", output)
	}
}

func TestRunFinalWriter_RendersTopicExplainFollowUpWithoutGenericAdvice(t *testing.T) {
	e := &Executor{}
	static := validStaticSynthesisForConsistencyTests()
	static.MainAxis = "月劫格中取食神制杀为主轴"
	static.PatternOutcome = "这里的重点不是重新评命高低，而是说明主格与出路的关系：先按月劫格立局，再看食神制杀这条具体成局路线。"
	static.AxisConsistency = "月劫格是格局框架，食神制杀是这张命盘在这个框架里的主要出路，不是另起第二格。"
	static.CounterEvidence = "这句话本身是在解释结构，不是在重判层次。"
	static.TopicDirectAnswer = "这里是在解释结构层次：月劫格先定框架，食神制杀再说明这张盘靠什么成局。"
	static.TopicFocusAnswer = "这轮追问不是在并列两个格局，而是在区分格局框架和成局路线。"
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{
			WriterTemplate: "topic",
			TopicMode:      "explain_term",
		},
		StaticSynthesis: static,
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:    "当前属于吉中有阻。",
			ClaimStrength:   "倾向成立",
			SupportLevel:    "有气",
			LimitationLevel: "明显",
			WordingCap:      "中性",
			LiunianFocus:    "今年宜稳中求进。",
			WindowLevel:     "窗口年",
		},
	}

	output, err := e.runFinalWriter(context.Background(), nil, state, "月劫格中取食神制杀是啥意思？能解释一下吗")
	if err != nil {
		t.Fatalf("runFinalWriter returned error: %v", err)
	}
	if !strings.Contains(output, "月劫格") || !strings.Contains(output, "食神制杀") {
		t.Fatalf("topic explain output should explain both layers, got:\n%s", output)
	}
	if strings.Contains(output, "判断宜保守落稳") {
		t.Fatalf("topic explain output should not fall back to generic conservative verdict, got:\n%s", output)
	}
	if strings.Contains(output, "先做自己能掌控的决定") {
		t.Fatalf("topic explain output should not use generic practical advice for explanation questions, got:\n%s", output)
	}
}

func TestRunFinalWriter_TopicTemplateUsesStructuredTopicAnswersInsteadOfQuestionHeuristics(t *testing.T) {
	e := &Executor{}
	static := validStaticSynthesisForConsistencyTests()
	static.TopicDirectAnswer = "财星破印就是财来压印，印的护持与承接会被削弱。"
	static.TopicFocusAnswer = "这轮追问重点是在解释术语本身在当前命盘里对应哪一层限制。"
	static.CounterEvidence = "财印牵制已明，所以这句话主要是在解释限制，不是在另起一条格局主轴。"
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{
			WriterTemplate: "topic",
			TopicMode:      "explain_term",
		},
		StaticSynthesis: static,
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:    "当前属于吉中有阻。",
			ClaimStrength:   "倾向成立",
			SupportLevel:    "有气",
			LimitationLevel: "明显",
			WordingCap:      "中性",
			LiunianFocus:    "今年宜稳中求进。",
			WindowLevel:     "窗口年",
		},
	}

	output, err := e.runFinalWriter(context.Background(), nil, state, "这句怎么读我不重要，直接看结构化字段")
	if err != nil {
		t.Fatalf("runFinalWriter returned error: %v", err)
	}
	if !strings.Contains(output, static.TopicDirectAnswer) {
		t.Fatalf("topic renderer should use structured direct answer, got:\n%s", output)
	}
	if !strings.Contains(output, static.TopicFocusAnswer) {
		t.Fatalf("topic renderer should use structured focus answer, got:\n%s", output)
	}
	if strings.Contains(output, "先按月劫格立框架") {
		t.Fatalf("topic renderer should not fall back to question-text heuristic, got:\n%s", output)
	}
}

func TestSupplementDynamicEvidenceIfNeeded_SkipsWhenSystemFactsExist(t *testing.T) {
	e := &Executor{}
	state := baziCharterState{
		AnalysisPlan: baziAnalysisPlan{
			NeedDynamic:    true,
			RetrievalStage: "static",
		},
		Input: baziCharterInput{
			Dayun:   map[string]any{"ok": true},
			Liunian: map[string]any{"year": 2026},
		},
	}
	next, err := e.supplementDynamicEvidenceIfNeeded(context.Background(), nil, "test", state)
	if err != nil {
		t.Fatalf("supplementDynamicEvidenceIfNeeded returned error: %v", err)
	}
	if next.EvidencePlan.Stage != "" || len(next.EvidenceBundle.Citations) != 0 {
		t.Fatalf("expected dynamic supplement to be skipped when system facts already exist")
	}
}

func TestNormalizeBaziAnalysisPlan_DefaultsTopicModeForTopicWriter(t *testing.T) {
	plan := normalizeBaziAnalysisPlan(baziAnalysisPlan{
		WriterTemplate: "topic",
	})
	if plan.TopicMode != "analysis" {
		t.Fatalf("TopicMode = %q, want analysis", plan.TopicMode)
	}
}

func TestBaziDynamicSynthesisPrompt_RequiresTrendConsistency(t *testing.T) {
	if !strings.Contains(prompts.BaziDynamicSynthesisInstruction, "`current_trend` 与 `dayun_path` 必须同一线") {
		t.Fatalf("dynamic synthesis prompt must require current_trend and dayun_path consistency")
	}
	if !strings.Contains(prompts.BaziDynamicSynthesisInstruction, "若某步运是吉中有阻，总纲就不得写成从此后明显一路顺") {
		t.Fatalf("dynamic synthesis prompt must forbid over-positive trend summary when detailed dayun is mixed")
	}
}

func TestBaziDynamicSynthesisPrompt_ForbidsCasualInvalidRelations(t *testing.T) {
	if !strings.Contains(prompts.BaziDynamicSynthesisInstruction, "`合身`、`制劫`、`合杀`、`合官`") {
		t.Fatalf("dynamic synthesis prompt must forbid casual invalid relational jargon")
	}
}

func TestBaziDynamicSynthesisPrompt_UsesReadableTrendLabels(t *testing.T) {
	prompt := prompts.BaziDynamicSynthesisInstruction
	for _, forbidden := range []string{"压力面可见", "承托与压力并见", "结构承接可见"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("dynamic synthesis prompt must not teach mechanical trend label %q", forbidden)
		}
	}
	for _, required := range []string{"有助力但不纯顺", "阻力偏重，宜保守观察", "有转机，也有牵制"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("dynamic synthesis prompt missing readable trend label %q", required)
		}
	}
}

func TestBaziStaticSynthesisPrompt_CompletesTiaohouWithoutImplementationLeak(t *testing.T) {
	prompt := prompts.BaziStaticSynthesisInstruction
	for _, forbidden := range []string{"调候规则表尚未实现", "规则表未实现", "调候视角未完成"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("static synthesis prompt must not expose implementation state %q", forbidden)
		}
	}
	for _, required := range []string{"本轮只确认季节环境，具体调候先后需补足对应规则材料", "季节环境 + 寒暖燥湿约束 + 证据边界", "input.subject_context"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("static synthesis prompt missing tiaohou/age boundary %q", required)
		}
	}
}

func TestValidateStaticStage_RequiresDecisionStrengthFields(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:           "弃印就财",
			PatternBasis:       "先看月令印星成势，再看财星透干破印。",
			PatternOutcome:     "方向成立，但力度受限。",
			CounterEvidence:    "财星偏湿，仍属受限，不算强救。",
			AxisConsistency:    "当前仍以弃印就财为主轴，而非改取别格。",
			TiaohouAnchor:      "甲木生子月，先按寒冬木冷的月令场景审调候。",
			PatternAndQingZhuo: "浊中有清。",
			TierJudgment:       "中等",
			TierBasis:          "主轴可立，但调候不足压低层次。",
			ReasoningSummary:   "静态主轴可以成立，但限制明显。",
			ReasoningSteps:     []string{"先看月令子水当令，印星成势。"},
		},
	}

	if err := validateStaticStage(state); err == nil {
		t.Fatalf("expected missing decision-strength fields to fail static validation")
	}
}

func TestValidateDynamicStage_RequiresDecisionStrengthFields(t *testing.T) {
	state := baziCharterState{
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "当前属于吉中有阻。",
			DayunPath:        []string{"当前大运承托主轴，但同时放大病点。"},
			LiunianFocus:     "这一年更像窗口年，但冲扰较重。",
			WindowLevel:      "窗口年",
			ReasoningSummary: "先看大运承托，再看流年冲动原局。",
			ReasoningSteps:   []string{"先看当前大运承托主轴。"},
		},
	}

	if err := validateDynamicStage(state); err == nil {
		t.Fatalf("expected missing decision-strength fields to fail dynamic validation")
	}
}

func TestValidateDynamicStage_SoftAuditsRelationsButRejectsHighRiskPredictions(t *testing.T) {
	valid := baziDynamicSynthesis{
		ClaimStrength:   "保守判断",
		SupportLevel:    "出现",
		LimitationLevel: "明显",
		WordingCap:      "保守",
		WindowLevel:     "扰动年",
	}
	valid.LiunianFocus = "流年午与大运午、时柱午构成自刑，属于已声明关系。"
	if err := validateDynamicDecisionConsistency(valid); err != nil {
		t.Fatalf("expected declared 午午自刑 to pass, got %v", err)
	}

	for _, text := range []string{
		"流年午酉相破，变化明显。", "申午暗合，关系有变化。",
		"丙火合月干丁火，印星汇聚。", "羊刃逢刑，只作结构观察。",
	} {
		candidate := valid
		candidate.LiunianFocus = text
		candidate.CurrentTrend = "当前运势有扰动。"
		candidate.DayunPath = []string{"甲午运：关系有扰动。"}
		candidate.ReasoningSummary = "按已计算关系观察。"
		candidate.ReasoningSteps = []string{"逐项核对。"}
		state := baziCharterState{DynamicSynthesis: candidate}
		if err := validateDynamicStage(state); err != nil {
			t.Fatalf("interpretive relation language must be soft-audited, got %v for %s", err, text)
		}
	}

	for _, text := range []string{"注意高血压风险。", "易有官非。", "羊刃逢刑，需防血光。"} {
		invalid := valid
		invalid.LiunianFocus = text
		invalid.CurrentTrend = "当前运势有扰动。"
		invalid.DayunPath = []string{"甲午运：关系有扰动。"}
		invalid.ReasoningSummary = "按已计算关系观察。"
		invalid.ReasoningSteps = []string{"逐项核对。"}
		if err := validateDynamicStage(baziCharterState{DynamicSynthesis: invalid}); err == nil {
			t.Fatalf("expected unsupported dynamic claim to fail: %s", text)
		}
	}
}

func TestValidateDynamicStage_RejectsClaimsOutsideDefaultProfileAcrossAllFields(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{},
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "当前只陈述已计算关系。",
			ClaimStrength:    "保守判断",
			SupportLevel:     "出现",
			LimitationLevel:  "明显",
			WordingCap:       "保守",
			DayunPath:        []string{"当前大运的具体吉凶待模型综合。"},
			LiunianFocus:     "流年仅呈现已计算关系的触发。",
			WindowLevel:      "扰动年",
			Risks:            []string{"注意心血管风险。"},
			ReasoningSummary: "本轮不作具体应事。",
			ReasoningSteps:   []string{"先保留已计算关系。"},
		},
	}
	if err := validateDynamicStage(state); err == nil {
		t.Fatal("default profile must reject unsupported outcome in risks")
	}
}

func TestValidateCharterConsistency_AllowsMixedTrendWording(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "从此以后明显一路顺行。",
			ClaimStrength:    "倾向成立",
			SupportLevel:     "有气",
			LimitationLevel:  "明显",
			WordingCap:       "中性",
			ConsistencyFlags: []string{"吉中有阻"},
			DayunPath: []string{
				"当前大运承托主轴，但吉中有阻。",
				"后续阶段仍有放大病点之处。",
			},
			LiunianFocus:     "这一年仍有机会，但限制并存。",
			WindowLevel:      "窗口年",
			ReasoningSummary: "整体仍是吉中有阻，不算全面起飞。",
			ReasoningSteps:   []string{"先看当前大运承托主轴，但仍有阻力。"},
		},
	}

	if err := validateCharterConsistency(state); err != nil {
		t.Fatalf("mixed explanatory wording must remain a soft-audit concern, got %v", err)
	}
}

func TestValidateStaticDecisionConsistency_AcceptsEquivalentLimitationWording(t *testing.T) {
	static := validStaticSynthesisForConsistencyTests()
	static.PatternOutcome = "主轴方向可以成立，但药力不够强，层次受限。"
	static.CounterEvidence = "财星虽透，但湿土难任重药，难以拔高。"
	static.TierBasis = "整体路线可立，但关键限制未解，难以进入上等。"

	if err := validateStaticDecisionConsistency(static); err != nil {
		t.Fatalf("expected equivalent limitation wording to pass, got %v", err)
	}
}

func TestValidateCharterConsistency_DoesNotInferJiShenVerdictWithoutProfileRule(t *testing.T) {
	state := baziCharterState{
		Input: baziCharterInput{
			Yongshen: map[string]any{
				"strength":    "身强",
				"tiao_hou":    "需火调候暖局",
				"yong_shen":   []any{"火", "土", "金"},
				"ji_shen":     []any{"木", "水"},
				"geju":        "偏印格",
				"geju_status": "基本成立",
			},
		},
		StaticSynthesis: baziStaticSynthesis{
			MainAxis:           "杀印相生为主轴",
			ClaimStrength:      "明确成立",
			SupportLevel:       "得力",
			LimitationLevel:    "明显",
			WordingCap:         "明确",
			PatternBasis:       "申子半合，印星承杀。",
			PatternOutcome:     "可以直接按杀印相生贵格立主轴。",
			CounterEvidence:    "原局仍有寒湿，但不改主轴贵气。",
			AxisConsistency:    "虽然身强忌水，仍以杀印相生为第一主轴。",
			TiaohouConstraint:  "寒木逢冬，火为第一调候。",
			TiaohouAnchor:      "甲木生亥月，先按寒木待火的月令场景审调候。",
			StrengthBalance:    "印水偏旺，身强，忌水木再增。",
			PatternAndQingZhuo: "结构可见。",
			QiShiOrCongHua:     "不从印，仍按正格处理。",
			TierJudgment:       "中等偏上",
			TierBasis:          "杀印路线既成，足以拔高层次。",
			ReasoningSummary:   "虽有寒湿，但不影响杀印相生作为主要贵格。",
			ReasoningSteps: []string{
				"先看申子半合，再看印星承杀。",
				"再看调候不足，但不足以影响主轴贵气。",
			},
		},
	}

	if err := validateCharterConsistency(state); err != nil {
		t.Fatalf("legacy yongshen fields must not become an implicit profile verdict: %v", err)
	}
}

func TestValidateCharterConsistency_AcceptsEquivalentVolatilityWording(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "当前处在可主动求变、但局势并不算稳的阶段。",
			ClaimStrength:    "倾向成立",
			SupportLevel:     "有气",
			LimitationLevel:  "明显",
			WordingCap:       "明确",
			ConsistencyFlags: []string{"机会伴随强变动"},
			DayunPath: []string{
				"当前大运承托主轴，但外部扰动也会同步放大。",
			},
			LiunianFocus:     "这一年有切换跑道的机会，但波折反复，宜边走边看。",
			WindowLevel:      "窗口年",
			ReasoningSummary: "机会是真有，但过程起伏明显，贸然重仓反而容易失手。",
			ReasoningSteps:   []string{"先看流年点亮用神，再看冲扰并存，所以只能按强变动中的机会处理。"},
		},
	}

	if err := validateCharterConsistency(state); err != nil {
		t.Fatalf("expected equivalent volatility wording to pass, got %v", err)
	}
}

func TestValidateCharterConsistency_AllowsCurrentDayunDirectionSplitAsSoftAudit(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "当前甲午大运承托静态主轴，但同时放大病点，整体属于吉中有阻。",
			ClaimStrength:    "倾向成立",
			SupportLevel:     "有气",
			LimitationLevel:  "明显",
			WordingCap:       "明确",
			ConsistencyFlags: []string{"吉中有阻"},
			DayunPath: []string{
				"甲午运（30-39岁）：天干甲木七杀为用神，地支午火印星为忌神。此运为偏压，但并非全凶，是承压中求发展的阶段。",
				"癸巳运（40-49岁）：财星透出，整体更利于主轴兑现。",
			},
			LiunianFocus:     "这一年有推进窗口，但仍会同步触发内耗与反复。",
			WindowLevel:      "窗口年",
			ReasoningSummary: "先看甲午运对主轴有承托，再看午火把原局病点一并放大，所以只能按吉中有阻落判。",
			ReasoningSteps:   []string{"先看当前甲午大运承托主轴，但不能把放大病点这一层忽略。"},
		},
	}

	if err := validateCharterConsistency(state); err != nil {
		t.Fatalf("current-dayun wording split must remain a soft-audit concern, got %v", err)
	}
}

func TestRunStaticSynthesisWithFeedback_RetriesAfterRestrictedRouteEscalation(t *testing.T) {
	executor := &Executor{}
	chartState := baziCharterState{}
	first := baziStaticSynthesis{
		MainAxis:                "杀印相生这条路有其结构，但还不能直接拔高为贵格主轴",
		ClaimStrength:           "倾向成立",
		SupportLevel:            "有气",
		LimitationLevel:         "明显",
		WordingCap:              "中性",
		ConsistencyFlags:        []string{"方向成立但力度受限"},
		AxisLevel:               "方向成立",
		EffectOnTiaohou:         "冲突",
		EffectOnCoreDisease:     "放大",
		EffectOnJiShenDirection: "放大",
		AxisCeiling:             "受限路线",
		ConflictReasons:         []string{"该路线继续放大寒湿与忌神方向，不能拔高为纯主轴贵格"},
		PatternBasis:            "先看七杀与印星确有承接，再看这条路线仍受寒湿病点掣肘。",
		PatternOutcome:          "这条路线可以直接化杀为权，足以拔高为纯主轴贵格。",
		CounterEvidence:         "虽然看似有其路数，但寒湿病点未解，仍不宜直接拔高。",
		AxisConsistency:         "当前只能把它当作受限路线参考，不能按纯贵格主轴落判。",
		TiaohouConstraint:       "寒木待火，调候仍是第一硬约束。",
		TiaohouAnchor:           "甲木生亥月，先按寒木待火的月令场景审调候。",
		StrengthBalance:         "印水偏旺，忌神方向未退，继续放大则病点更重。",
		PatternAndQingZhuo:      "结构可见，但清浊未净。",
		QiShiOrCongHua:          "不从化，仍按正格中的受限路线处理。",
		TierJudgment:            "中等",
		TierBasis:               "若把它拔高为贵格主轴，就会越过受限路线的天花板。",
		ReasoningSummary:        "这条结构有其路数，但调候与病点冲突未解，因此不能拔高。",
		ReasoningSteps: []string{
			"先看杀印之间确有承接，所以只能说这条路线方向成立。",
			"再看寒湿病点与忌神方向仍被放大，所以它最多只能停在受限路线。",
		},
	}

	second := baziStaticSynthesis{
		MainAxis:                "杀印相生这条路可以保留为受限路线参考",
		ClaimStrength:           "倾向成立",
		SupportLevel:            "有气",
		LimitationLevel:         "明显",
		WordingCap:              "中性",
		ConsistencyFlags:        []string{"方向成立但力度受限"},
		AxisLevel:               "方向成立",
		EffectOnTiaohou:         "冲突",
		EffectOnCoreDisease:     "放大",
		EffectOnJiShenDirection: "放大",
		AxisCeiling:             "受限路线",
		ConflictReasons:         []string{"该路线继续放大寒湿与忌神方向，不能拔高为纯主轴贵格"},
		PatternBasis:            "先看七杀与印星确有承接，再看这条路线仍受寒湿病点掣肘。",
		PatternOutcome:          "这条路线方向成立，但力度受限，只能作受限路线参考。",
		CounterEvidence:         "虽然有其路数，但仍会放大病点，不宜拔高为纯主轴贵格。",
		AxisConsistency:         "当前只能保留它的结构参考价值，而不能改写成贵格主轴。",
		TiaohouConstraint:       "寒木待火，调候仍是第一硬约束。",
		TiaohouAnchor:           "甲木生亥月，先按寒木待火的月令场景审调候。",
		StrengthBalance:         "印水偏旺，忌神方向未退，继续放大则病点更重。",
		PatternAndQingZhuo:      "结构可见，但清浊未净。",
		QiShiOrCongHua:          "不从化，仍按正格中的受限路线处理。",
		TierJudgment:            "中等",
		TierBasis:               "主轴有路，但核心冲突未解，层次受限，难以拔高。",
		ReasoningSummary:        "这条结构只能保留为受限路线，不能越级写成贵格主轴。",
		ReasoningSteps: []string{
			"先看杀印之间确有承接，所以只能说这条路线方向成立。",
			"再看寒湿病点与忌神方向仍被放大，因此必须把结论收在受限路线。",
		},
	}

	calls := 0
	gotFeedback := ""
	runStatic := func(payload map[string]any) (baziStaticSynthesis, error) {
		calls++
		if calls == 2 {
			if feedback, _ := payload["static_feedback"].(string); strings.TrimSpace(feedback) == "" {
				t.Fatalf("expected second static synthesis attempt to include static_feedback")
			} else {
				gotFeedback = feedback
			}
			return second, nil
		}
		if _, exists := payload["static_feedback"]; exists {
			t.Fatalf("first static synthesis attempt should not include static_feedback")
		}
		return first, nil
	}

	out, err := executor.runStaticSynthesisWithFeedback(chartState, runStatic)
	if err != nil {
		t.Fatalf("expected retry to recover restricted-route escalation, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("static synthesis calls = %d, want 2", calls)
	}
	if strings.TrimSpace(gotFeedback) == "" {
		t.Fatalf("expected retry feedback to be non-empty")
	}
	if !strings.Contains(gotFeedback, "受限路线") {
		t.Fatalf("expected retry feedback to mention restricted route ceiling, got %q", gotFeedback)
	}
	if !strings.Contains(out.PatternOutcome, "受限") {
		t.Fatalf("expected recovered static synthesis to retain a local limitation, got %q", out.PatternOutcome)
	}
}

func TestValidateCharterConsistency_AllowsOverstatedWindowYearAsSoftAudit(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "当前是机会伴随强变动的阶段。",
			ClaimStrength:    "倾向成立",
			SupportLevel:     "有气",
			LimitationLevel:  "明显",
			WordingCap:       "明确",
			ConsistencyFlags: []string{"机会伴随强变动"},
			DayunPath:        []string{"当前大运承托主轴，但也放大波动。"},
			LiunianFocus:     "这是关键翻身年，足以一飞冲天。",
			WindowLevel:      "窗口年",
			ReasoningSummary: "这一年彻底起势。",
			ReasoningSteps:   []string{"先看流年点亮用神，但冲扰也同步出现。"},
		},
	}

	if err := validateCharterConsistency(state); err != nil {
		t.Fatalf("window-year wording must remain a soft-audit concern, got %v", err)
	}
}

func TestValidateFinalWriterOutput_RejectsMissingSummarySection(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
	}
	plan := baziAnalysisPlan{WriterTemplate: "full"}
	output := strings.Join([]string{
		"## 强弱视角\n**结论：身旺极。**",
		"## 调候视角\n**结论：调候不足。**",
		"## 格局视角\n**结论：弃印就财。**",
		"## 大运验证\n**结论：吉中有阻。**",
		"## 流年应期\n**结论：机会伴随强变动。**",
		"## 综合判定\n**结论：命格层次中等。**",
	}, "\n")

	if err := validateFinalWriterOutput(plan, state, output); err == nil {
		t.Fatalf("expected missing 总览结论 and 命格总结 sections to fail validation")
	}
}

func TestValidateFinalWriterOutput_AllowsFlourishClaimsForSoftAudit(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
	}
	plan := baziAnalysisPlan{WriterTemplate: "topic"}
	output := strings.Join([]string{
		"## 直接回答",
		"**结论：整体可期。**",
		"命局贵人众多，福泽深厚，但仍有受限处。",
		"## 命盘依据",
		"**结论：主轴明确。**",
		"## 建议",
		"**结论：稳中求进。**",
	}, "\n")

	if err := validateFinalWriterOutput(plan, state, output); err != nil {
		t.Fatalf("flourish wording must not fail the final structure contract, got %v", err)
	}
}

func TestValidateFinalWriterOutput_AllowsMeasuredFlourishWhenWordingCapExplicit(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: func() baziStaticSynthesis {
			s := validStaticSynthesisForConsistencyTests()
			s.WordingCap = "明确"
			return s
		}(),
	}
	plan := baziAnalysisPlan{WriterTemplate: "topic"}
	output := strings.Join([]string{
		"## 直接回答",
		"**结论：整体仍可期待。**",
		"命局助力不缺，福泽深厚，但仍有受限处，不宜拔高。",
		"## 命盘依据",
		"**结论：主轴明确。**",
		"## 建议",
		"**结论：稳中求进。**",
	}, "\n")

	if err := validateFinalWriterOutput(plan, state, output); err != nil {
		t.Fatalf("expected explicit wording cap to allow measured flourish, got %v", err)
	}
}

func TestValidateFinalWriterOutput_DoesNotOwnUnsupportedDynamicBoundaryTerms(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
	}
	plan := baziAnalysisPlan{WriterTemplate: "topic"}
	output := strings.Join([]string{
		"## 直接回答",
		"**结论：当前阶段宜稳步观察。**",
		"水火相战，据此直接断为高血压，但原局仍有受限处。",
		"## 命盘依据",
		"**结论：只保留已计算关系。**",
		"## 建议",
		"**结论：不据此作具体应事判断。**",
	}, "\n")

	if err := validateFinalWriterOutput(plan, state, output); err != nil {
		t.Fatalf("final writer should only validate structure and preserved boundaries, got %v", err)
	}
}

func TestValidateFinalWriterOutput_RejectsDroppedLimitationAndVolatilitySignals(t *testing.T) {
	state := baziCharterState{
		StaticSynthesis: validStaticSynthesisForConsistencyTests(),
		DynamicSynthesis: baziDynamicSynthesis{
			CurrentTrend:     "当前属于吉中有阻。",
			ClaimStrength:    "倾向成立",
			SupportLevel:     "有气",
			LimitationLevel:  "明显",
			WordingCap:       "中性",
			ConsistencyFlags: []string{"吉中有阻", "机会伴随强变动"},
			DayunPath:        []string{"当前大运承托主轴，但放大病点。"},
			LiunianFocus:     "这一年更像窗口年，不宜激进。",
			WindowLevel:      "窗口年",
			ReasoningSummary: "机会伴随强变动，吉中带险。",
			ReasoningSteps:   []string{"先看流年点亮用神，再看冲扰同步放大。"},
		},
	}
	plan := baziAnalysisPlan{WriterTemplate: "full"}
	output := strings.Join([]string{
		"## 强弱视角\n**结论：身旺极。**",
		"## 调候视角\n**结论：调候不足。**",
		"## 格局视角\n**结论：弃印就财。**",
		"## 大运验证\n**结论：当前运势偏吉。**",
		"## 流年应期\n**结论：这是关键翻身年。**",
		"## 综合判定\n**结论：命格层次中等偏上。**",
		"## 命格总结\n- 最大优点：主轴清晰。",
	}, "\n")

	if err := validateFinalWriterOutput(plan, state, output); err == nil {
		t.Fatalf("expected dropped limitation or volatility signals to fail validation")
	}
}

func validStaticSynthesisForConsistencyTests() baziStaticSynthesis {
	return baziStaticSynthesis{
		MainAxis:           "弃印就财",
		ClaimStrength:      "倾向成立",
		SupportLevel:       "有气",
		LimitationLevel:    "明显",
		WordingCap:         "中性",
		ConsistencyFlags:   []string{"方向成立但力度受限"},
		PatternBasis:       "先看印星成势，再看财星透出破印。",
		PatternOutcome:     "方向成立，但力度受限。",
		CounterEvidence:    "财星偏湿，仍属受限，不足以算强救。",
		AxisConsistency:    "当前仍以弃印就财为主轴，而非改取别格。",
		TiaohouConstraint:  "寒冬木冷，火为第一调候。",
		TiaohouAnchor:      "甲木生子月，先按寒冬木冷的月令场景审调候。",
		StrengthBalance:    "印比偏旺，喜火土金。",
		PatternAndQingZhuo: "浊中有清。",
		QiShiOrCongHua:     "不从强，不从印，仍按正格处理。",
		TierJudgment:       "中等",
		TierBasis:          "主轴可立，但受限明显，难以拔高。",
		ReasoningSummary:   "静态主轴成立，但调候与药力都有限。",
		ReasoningSteps: []string{
			"先看月令印星当令，命局根气偏厚。",
			"再看财星虽透，但力度受限，不算强救。",
		},
	}
}

func TestRenderBaziFinalReply_DeduplicatesExactLimitationFallback(t *testing.T) {
	state := baziCharterState{StaticSynthesis: validStaticSynthesisForConsistencyTests()}
	state.StaticSynthesis.CounterEvidence = "关系触发会增加过程反复，具体应事不作展开。"
	state.StaticSynthesis.TierBasis = "关系触发会增加过程反复，具体应事不作展开。"
	if got := buildLimitationText(state); strings.Count(got, "关系触发会增加过程反复，具体应事不作展开。") != 1 {
		t.Fatalf("expected exact duplicate limitation to be rendered once, got %q", got)
	}
}

func TestEmitBaziStageThinking_EmitsThinkingEvent(t *testing.T) {
	tracer := tracing.NewRealTracer(nil)
	ctx, trace := tracer.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	sink := &recordingSink{}
	emitBaziStageThinking(ctx, sink, "bazi_graph", "静态综合已完成，命局主轴收敛为：杀印相生")

	if len(sink.events) != 1 {
		t.Fatalf("events = %d, want 1", len(sink.events))
	}
	if sink.events[0].Type != "thinking" {
		t.Fatalf("event type = %q, want thinking", sink.events[0].Type)
	}
	data, ok := sink.events[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("event data type = %T, want map[string]any", sink.events[0].Data)
	}
	if data["agent"] != "bazi_graph" {
		t.Fatalf("agent = %v, want bazi_graph", data["agent"])
	}
	if data["text"] != "静态综合已完成，命局主轴收敛为：杀印相生" {
		t.Fatalf("text = %v", data["text"])
	}
}
