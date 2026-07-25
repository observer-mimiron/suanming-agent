package runtime

import (
	"strings"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/guidance"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

func makeSession(profileComplete, hasBazi, hasZiwei bool) *state.SessionState {
	s := state.NewSession("test-session")
	if profileComplete {
		s.MergeProfile(map[string]any{
			"birthplace": "北京", "year": 1990.0, "month": 5.0, "day": 20.0,
			"hour": 8.0, "gender": "男",
		})
	}
	if hasBazi {
		s.BaziResult = map[string]any{"dayGan": "甲", "dayZhi": "子"}
	}
	if hasZiwei {
		s.ZiWeiResult = map[string]any{"mingGong": "天机"}
	}
	return s
}

func routeWithHints(primaryDomain, taskIntent, qimenMode, profileReq string) policy.ApprovedRoute {
	return policy.ApprovedRoute{
		PrimaryDomain: primaryDomain,
		TaskIntent:    taskIntent,
		PolicyHints: schemas.PolicyHints{
			QimenMode:          qimenMode,
			ProfileRequirement: profileReq,
		},
	}
}

// TestPreflight_ProfileRequiredRouteBlocksWithoutProfile 验证 profile_requirement=full 且
// profile 不完整时，preflight 返回 ask_missing_profile。
func TestPreflight_ProfileRequiredRouteBlocksWithoutProfile(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("bazi", "fortune_followup", "none", "full")

	result := preflight(st, route, "", nil)
	if !result.ShortCircuit {
		t.Fatal("expected ShortCircuit=true for profile-required route without profile")
	}
	if result.TurnType != "ask_missing_profile" {
		t.Fatalf("TurnType: got %q, want ask_missing_profile", result.TurnType)
	}
}

// TestPreflight_ProfileRequiredRouteAllowsWithProfile 验证 profile_requirement=full 且
// profile 完整时，preflight 放行。
func TestPreflight_ProfileRequiredRouteAllowsWithProfile(t *testing.T) {
	st := makeSession(true, true, false)
	route := routeWithHints("bazi", "fortune_followup", "none", "full")

	result := preflight(st, route, "", nil)
	if result.ShortCircuit {
		t.Fatalf("expected no short circuit for complete profile, got TurnType=%q", result.TurnType)
	}
}

// TestPreflight_QimenPrimaryWithoutProfile 验证奇门主链且 profile_requirement=none 时，
// 即使无资料也放行。
func TestPreflight_QimenPrimaryWithoutProfile(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("qimen", "fortune_followup", "primary", "none")

	result := preflight(st, route, "", nil)
	if result.ShortCircuit {
		t.Fatalf("qimen primary with profile_requirement=none should not short circuit, got TurnType=%q", result.TurnType)
	}
}

// TestPreflight_QimenPrimaryWithFullProfileRequirementBlocks 验证奇门主链但
// profile_requirement=full 且无资料时，preflight 拦截。
func TestPreflight_QimenPrimaryWithFullProfileRequirementBlocks(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("qimen", "fortune_followup", "primary", "full")

	result := preflight(st, route, "", nil)
	if !result.ShortCircuit {
		t.Fatal("qimen primary with profile_requirement=full and no profile should short circuit")
	}
}

// TestPreflight_ClarificationRouteShortCircuits 验证 NeedsClarification 路由被 preflight
// 直接返回。
func TestPreflight_ClarificationRouteShortCircuits(t *testing.T) {
	st := makeSession(true, true, false)
	route := policy.ApprovedRoute{
		NeedsClarification:    true,
		ClarificationQuestion: "  请问您想了解哪方面？\n",
	}

	result := preflight(st, route, "", nil)
	if !result.ShortCircuit {
		t.Fatal("NeedsClarification route should short circuit")
	}
	if result.TurnType != "clarification" {
		t.Fatalf("TurnType: got %q, want clarification", result.TurnType)
	}
	want := guidance.Render(guidance.Request{
		Boundary: guidance.BoundaryClarificationFallback,
		Context: guidance.Context{
			ClarificationQuestion: route.ClarificationQuestion,
		},
	})
	if result.Text != want {
		t.Fatalf("Text: got %q, want %q", result.Text, want)
	}
}

// TestPreflight_BaziWithoutProfileOrChartBlocks 验证八字主域无资料且无命盘时，preflight
// 返回 ask_missing_profile（collect_profile/amend_profile 除外）。
func TestPreflight_BaziWithoutProfileOrChartBlocks(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("bazi", "interpret_chart", "none", "none")

	result := preflight(st, route, "", nil)
	if !result.ShortCircuit {
		t.Fatal("bazi route without profile or chart should short circuit")
	}
	if result.TurnType != "ask_missing_profile" {
		t.Fatalf("TurnType: got %q, want ask_missing_profile", result.TurnType)
	}
}

// TestPreflight_CollectProfileIntentShortCircuits 验证 collect_profile + 空资料
// 短路返回领域追问模板。
func TestPreflight_CollectProfileIntentShortCircuits(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("bazi", "collect_profile", "none", "none")

	result := preflight(st, route, "", nil)
	if !result.ShortCircuit {
		t.Fatal("collect_profile with empty profile should short circuit with domain prompt")
	}
	if result.TurnType != "clarification" {
		t.Fatalf("TurnType: got %q, want clarification", result.TurnType)
	}
	want := guidance.Render(guidance.Request{Boundary: guidance.BoundaryAskBaziProfile})
	if result.Text != want {
		t.Fatalf("Text: got %q, want %q", result.Text, want)
	}
}

// TestPreflight_ZiweiWithoutProfileOrChartBlocks 验证紫微主域无资料且无命盘时拦截。
func TestPreflight_ZiweiWithoutProfileOrChartBlocks(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("ziwei", "fortune_followup", "none", "none")

	result := preflight(st, route, "", nil)
	if !result.ShortCircuit {
		t.Fatal("ziwei route without profile or chart should short circuit")
	}
}

func TestPreflight_CollectProfileMissingGenderUsesBoundaryRenderer(t *testing.T) {
	st := state.NewSession("test-session")
	st.MergeProfile(map[string]any{
		"year":  1990.0,
		"month": 5.0,
		"day":   20.0,
		"hour":  8.0,
	})
	route := routeWithHints("bazi", "collect_profile", "none", "none")

	result := preflight(st, route, "", nil)
	if !result.ShortCircuit {
		t.Fatal("collect_profile missing gender should short circuit")
	}
	if result.TurnType != "clarification" {
		t.Fatalf("TurnType: got %q, want clarification", result.TurnType)
	}
	want := guidance.Render(guidance.Request{Boundary: guidance.BoundaryCollectGenderFromBirthTime})
	if result.Text != want {
		t.Fatalf("Text: got %q, want %q", result.Text, want)
	}
}

func TestPreflight_CollectProfileUsesRouteSlotProfileForMissingGender(t *testing.T) {
	st := state.NewSession("test-session")
	route := routeWithHints("bazi", "collect_profile", "none", "none")
	route.Slots.Profile = map[string]any{
		"year":  2025.0,
		"month": 11.0,
		"day":   10.0,
		"hour":  23.0,
	}

	result := preflight(st, route, "2025年11月10日23点", nil)
	if !result.ShortCircuit {
		t.Fatal("collect_profile with route slot birth time should short circuit")
	}
	if result.TurnType != "clarification" {
		t.Fatalf("TurnType: got %q, want clarification", result.TurnType)
	}
	want := guidance.Render(guidance.Request{Boundary: guidance.BoundaryCollectGenderFromBirthTime})
	if result.Text != want {
		t.Fatalf("Text: got %q, want %q", result.Text, want)
	}
}

func TestPreflight_CollectProfileOnlyMissingBirthplaceUsesSpecificPrompt(t *testing.T) {
	st := state.NewSession("test-session")
	route := routeWithHints("bazi", "collect_profile", "none", "none")
	route.Slots.Profile = map[string]any{
		"year":   2025.0,
		"month":  11.0,
		"day":    10.0,
		"hour":   23.0,
		"gender": "男",
	}

	result := preflight(st, route, "2025年11月10日23点 男", nil)
	if !result.ShortCircuit {
		t.Fatal("collect_profile with only birthplace missing should short circuit")
	}
	if result.TurnType != "clarification" {
		t.Fatalf("TurnType: got %q, want clarification", result.TurnType)
	}
	want := guidance.Render(guidance.Request{Boundary: guidance.BoundaryCollectBirthplaceFromProfile})
	if result.Text != want {
		t.Fatalf("Text: got %q, want %q", result.Text, want)
	}
}

func TestPreflight_ZiweiMissingProfileUsesBoundaryRenderer(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("ziwei", "fortune_followup", "none", "none")

	result := preflight(st, route, "", nil)
	if !result.ShortCircuit {
		t.Fatal("ziwei route without profile or chart should short circuit")
	}
	if result.TurnType != "ask_missing_profile" {
		t.Fatalf("TurnType: got %q, want ask_missing_profile", result.TurnType)
	}
	want := guidance.Render(guidance.Request{Boundary: guidance.BoundaryAskZiweiProfile})
	if result.Text != want {
		t.Fatalf("Text: got %q, want %q", result.Text, want)
	}
}

// TestPreflight_CollectProfileWithFullRequirementShortCircuits 验证即使 supervisor
// 设置了 ProfileRequirement=full，collect_profile + 空资料也会短路回追问模板。
func TestPreflight_CollectProfileWithFullRequirementShortCircuits(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("bazi", "collect_profile", "none", "full")

	result := preflight(st, route, "", nil)
	if !result.ShortCircuit {
		t.Fatal("collect_profile with empty profile should short circuit")
	}
}

// TestPreflight_ZiweiCollectProfileShortCircuits 验证 ziwei 主域 + collect_profile
// + 空资料时短路返回 ziwei 追问模板。
func TestPreflight_ZiweiCollectProfileShortCircuits(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("ziwei", "collect_profile", "none", "none")

	result := preflight(st, route, "", nil)
	if !result.ShortCircuit {
		t.Fatal("ziwei collect_profile with empty profile should short circuit")
	}
	if result.TurnType != "clarification" {
		t.Fatalf("TurnType: got %q, want clarification", result.TurnType)
	}
	want := guidance.Render(guidance.Request{Boundary: guidance.BoundaryAskZiweiProfile})
	if result.Text != want {
		t.Fatalf("Text: got %q, want %q", result.Text, want)
	}
}

// TestPreflight_Purity_DoesNotMutateSession 验证 preflight 不直接写 session。
func TestPreflight_Purity_DoesNotMutateSession(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("bazi", "fortune_followup", "none", "none")
	snapshot := st.Guidance
	_ = preflight(st, route, "最近真是喝凉水都塞牙", nil)
	if st.Guidance != snapshot {
		t.Fatal("preflight mutated session.Guidance — purity violation")
	}
}

// TestPreflight_FateAdjacentReturnsGuidanceNext 验证命运感叹消息返回 GuidanceNext。
func TestPreflight_FateAdjacentReturnsGuidanceNext(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("bazi", "collect_profile", "none", "none")
	result := preflight(st, route, "最近真是喝凉水都塞牙", nil)
	if !result.ShortCircuit {
		t.Fatal("fate-adjacent should short circuit")
	}
	if result.GuidanceNext == nil {
		t.Fatal("GuidanceNext is nil, want offer_consult")
	}
	if result.GuidanceNext.DirectiveKind != "offer_consult" {
		t.Fatalf("GuidanceNext.DirectiveKind = %q, want offer_consult", result.GuidanceNext.DirectiveKind)
	}
}

// TestPreflight_BroadIntentReturnsGuidanceNext 验证 broad intent 消息返回 GuidanceNext。
func TestPreflight_BroadIntentReturnsGuidanceNext(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("bazi", "collect_profile", "none", "none")
	result := preflight(st, route, "看看事业", nil)
	if !result.ShortCircuit {
		t.Fatal("broad-intent should short circuit")
	}
	if result.GuidanceNext == nil {
		t.Fatal("GuidanceNext is nil, want choose_topic")
	}
	if result.GuidanceNext.DirectiveKind != "choose_topic" {
		t.Fatalf("GuidanceNext.DirectiveKind = %q, want choose_topic", result.GuidanceNext.DirectiveKind)
	}
}

// TestPreflight_ExplicitActionNotGuidance 验证 "帮我算一下八字" 不进入 guidance。
func TestPreflight_ExplicitActionNotGuidance(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("bazi", "collect_profile", "none", "none")
	result := preflight(st, route, "帮我算一下八字", nil)
	if result.GuidanceNext != nil {
		t.Fatalf("GuidanceNext = %+v, want nil for explicit action", result.GuidanceNext)
	}
	if !result.ShortCircuit {
		t.Fatal("collect_profile without data should short circuit")
	}
}

// TestPreflight_ActiveGuidanceProgresses 验证已有 guidance 时继续推进。
func TestPreflight_ActiveGuidanceProgresses(t *testing.T) {
	st := makeSession(false, false, false)
	st.Guidance = &state.GuidanceState{DirectiveKind: "offer_consult"}
	route := routeWithHints("bazi", "collect_profile", "none", "none")
	result := preflight(st, route, "行，那你看看", nil)
	if !result.ShortCircuit {
		t.Fatal("active guidance continuation should short circuit")
	}
	if result.GuidanceNext == nil {
		t.Fatal("GuidanceNext is nil, want choose_topic after acceptance")
	}
	if result.GuidanceNext.DirectiveKind != "choose_topic" {
		t.Fatalf("GuidanceNext.DirectiveKind = %q, want choose_topic", result.GuidanceNext.DirectiveKind)
	}
}

func TestPreflight_ActiveGuidanceUsesRouteSlotProfilePatch(t *testing.T) {
	st := state.NewSession("test-session")
	st.Guidance = &state.GuidanceState{
		DirectiveKind: "collect_slot",
		PendingSlot:   "birthplace",
	}
	st.MergeProfile(map[string]any{
		"year":   1990.0,
		"month":  5.0,
		"day":    20.0,
		"hour":   8.0,
		"gender": "男",
	})
	route := routeWithHints("bazi", "collect_profile", "none", "none")
	route.Slots.Profile = map[string]any{
		"birthplace": "北京",
	}

	result := preflight(st, route, "北京", nil)
	if result.ShortCircuit {
		t.Fatalf("ShortCircuit = true, want false after route slot patch completes guidance")
	}
	if result.GuidanceNext != nil {
		t.Fatalf("GuidanceNext = %+v, want nil after birthplace patch completes profile", result.GuidanceNext)
	}
}

// TestPreflight_StaleGuidanceDoesNotHijackFollowup 验证已有旧 guidance 时，
// 如果 supervisor 已明确判定为可执行的 follow-up 且命盘已存在，preflight 不应再短路回引导文案。
func TestPreflight_StaleGuidanceDoesNotHijackFollowup(t *testing.T) {
	st := makeSession(true, true, false)
	st.Guidance = &state.GuidanceState{DirectiveKind: "choose_topic", RetryCount: 1}
	route := routeWithHints("bazi", "fortune_followup", "none", "full")

	result := preflight(st, route, "前40年还有家庭危机吗", nil)
	if result.ShortCircuit {
		t.Fatalf("stale guidance should not hijack executable follow-up, got TurnType=%q Text=%q", result.TurnType, result.Text)
	}
	if result.GuidanceNext != nil {
		t.Fatalf("GuidanceNext = %+v, want nil when follow-up should enter execution path", result.GuidanceNext)
	}
}

// TestPreflight_FollowupWithApprovedCrossDomainRouteDoesNotRestartGuidance
// 验证已经批准为可执行的跨域追问，不会被新开 choose_topic guidance 抢走。
func TestPreflight_FollowupWithApprovedCrossDomainRouteDoesNotRestartGuidance(t *testing.T) {
	st := makeSession(true, true, true)
	route := routeWithHints("bazi", "fortune_followup", "none", "full")
	route.SecondaryDomains = []string{"ziwei"}

	result := preflight(st, route, "八字和紫微一起看下事业和感情", nil)
	if result.ShortCircuit {
		t.Fatalf("approved cross-domain follow-up should continue execution, got TurnType=%q Text=%q", result.TurnType, result.Text)
	}
	if result.GuidanceNext != nil {
		t.Fatalf("GuidanceNext = %+v, want nil for approved cross-domain follow-up", result.GuidanceNext)
	}
}

// TestPreflight_FollowupWithExplicitTopicsDoesNotOpenNewGuidance
// 验证已具备可复用命盘结果的明确追问，不会被新建 choose_topic guidance 误短路。
func TestPreflight_FollowupWithExplicitTopicsDoesNotOpenNewGuidance(t *testing.T) {
	st := makeSession(true, true, false)
	route := routeWithHints("bazi", "fortune_followup", "none", "full")

	result := preflight(st, route, "看看事业和感情", nil)
	if result.ShortCircuit {
		t.Fatalf("explicit follow-up topics should continue execution, got TurnType=%q Text=%q", result.TurnType, result.Text)
	}
	if result.GuidanceNext != nil {
		t.Fatalf("GuidanceNext = %+v, want nil for explicit follow-up topics", result.GuidanceNext)
	}
}

// TestPreflight_BaziGlossaryFollowupShortCircuitsDirectly 验证纯八字术语解释型追问
// 在已有命盘时由 manager-owned preflight 直接答复，不再重跑八字 graph。
func TestPreflight_BaziGlossaryFollowupShortCircuitsDirectly(t *testing.T) {
	st := makeSession(true, true, false)
	route := routeWithHints("bazi", "fortune_followup", "none", "full")
	plan := (&Manager{}).BuildExecutionPlan(st, route, "财星破印是啥意思")

	result := preflightWithPlan(st, plan, "财星破印是啥意思", nil)
	if !result.ShortCircuit {
		t.Fatal("glossary follow-up should short circuit")
	}
	if result.TurnType != "fortune_followup" {
		t.Fatalf("TurnType = %q, want fortune_followup", result.TurnType)
	}
	if !containsAnyText([]string{result.Text}, []string{"财来克印", "通用术语"}) {
		t.Fatalf("Text = %q, want glossary explanation", result.Text)
	}
}

// TestPreflight_ChartSpecificBaziQuestionStillRunsExecution 验证“我这盘为什么算 X”
// 这类依赖具体命盘结构的追问不会被常识 gate 误短路。
func TestPreflight_ChartSpecificBaziQuestionStillRunsExecution(t *testing.T) {
	st := makeSession(true, true, false)
	route := routeWithHints("bazi", "fortune_followup", "none", "full")
	plan := (&Manager{}).BuildExecutionPlan(st, route, "那我这盘里为什么算财星破印")

	result := preflightWithPlan(st, plan, "那我这盘里为什么算财星破印", nil)
	if result.ShortCircuit {
		t.Fatalf("chart-specific bazi follow-up should continue execution, got TurnType=%q Text=%q", result.TurnType, result.Text)
	}
}

// TestPreflight_BaziGlossaryFollowupDoesNotHijackCrossDomain 验证多域追问仍走执行链，
// 不会被八字术语常识 gate 提前截走。
func TestPreflight_BaziGlossaryFollowupDoesNotHijackCrossDomain(t *testing.T) {
	st := makeSession(true, true, true)
	route := routeWithHints("bazi", "fortune_followup", "none", "full")
	route.SecondaryDomains = []string{"ziwei"}
	plan := (&Manager{}).BuildExecutionPlan(st, route, "财星破印是啥意思")

	result := preflightWithPlan(st, plan, "财星破印是啥意思", nil)
	if result.ShortCircuit {
		t.Fatalf("cross-domain follow-up should continue execution, got TurnType=%q Text=%q", result.TurnType, result.Text)
	}
}

func TestPreflight_ReusedArtifactFollowupShortCircuitsDirectly(t *testing.T) {
	st := makeSession(true, true, false)
	st.DomainContexts.Bazi.RuntimeValues = map[string]any{
		followupArtifactKey: map[string]any{
			"domain":  "bazi",
			"summary": "上轮已经判断事业主线可走稳。",
		},
	}
	manager := &Manager{}
	route := routeWithHints("bazi", "fortune_followup", "none", "full")
	plan := manager.BuildExecutionPlan(st, route, "那事业具体怎么推进")

	result := preflightWithPlan(st, plan, "那事业具体怎么推进", nil)
	if !result.ShortCircuit {
		t.Fatal("reused interpretation follow-up should short circuit")
	}
	if result.TurnType != "fortune_followup" {
		t.Fatalf("TurnType = %q, want fortune_followup", result.TurnType)
	}
	if !strings.Contains(result.Text, "上轮已经判断事业主线可走稳") {
		t.Fatalf("Text = %q, want reused interpretation content", result.Text)
	}
}

// TestPreflight_GuidedFallbackAcceptanceReturnsForcedRoute 验证 guided_fallback 接受后返回 ForcedRoute。
func TestPreflight_GuidedFallbackAcceptanceReturnsForcedRoute(t *testing.T) {
	st := makeSession(false, false, false)
	st.Guidance = &state.GuidanceState{DirectiveKind: "guided_fallback"}
	route := routeWithHints("bazi", "collect_profile", "none", "none")
	result := preflight(st, route, "好，那你综合看看", nil)
	if result.ShortCircuit {
		t.Fatal("guided_fallback acceptance should NOT short circuit")
	}
	if result.ForcedRoute == nil {
		t.Fatal("ForcedRoute is nil, want qimen primary route")
	}
	if result.ForcedRoute.PrimaryDomain != "qimen" {
		t.Fatalf("ForcedRoute.PrimaryDomain = %q, want qimen", result.ForcedRoute.PrimaryDomain)
	}
	if result.ForcedRoute.PolicyHints.QimenMode != "primary" {
		t.Fatalf("ForcedRoute.QimenMode = %q, want primary", result.ForcedRoute.PolicyHints.QimenMode)
	}
	if result.ForcedRoute.PolicyHints.ProfileRequirement != "none" {
		t.Fatalf("ForcedRoute.ProfileRequirement = %q, want none", result.ForcedRoute.PolicyHints.ProfileRequirement)
	}
}
