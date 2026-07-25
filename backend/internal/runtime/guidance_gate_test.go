package runtime

import (
	"context"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/guidance"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

type stubGateRouter struct {
	result intent.MatchResult
	err    error
	called bool
}

func (s *stubGateRouter) Match(_ context.Context, msg string) (intent.MatchResult, error) {
	s.called = true
	return s.result, s.err
}

func newGateRoute(primary string, conf float64) policy.ApprovedRoute {
	return policy.ApprovedRoute{
		PrimaryDomain: primary,
		Confidence:    conf,
	}
}

func TestAnyHardNegative_RouterPositiveBreaksGuidance(t *testing.T) {
	r := &stubGateRouter{result: intent.MatchResult{Decision: intent.DecisionPositive, Method: "ziwei"}}
	route := newGateRoute("bazi", 0.5)
	signal := guidance.Signal{}

	got := anyHardNegative(r, "排个紫微盘", route, signal)
	if !got {
		t.Fatal("router positive should break guidance")
	}
}

func TestAnyHardNegative_RouterNegativeDoesNotBreak(t *testing.T) {
	r := &stubGateRouter{result: intent.MatchResult{Decision: intent.DecisionNegative}}
	route := newGateRoute("bazi", 0.5)
	signal := guidance.Signal{}

	got := anyHardNegative(r, "我不看紫微", route, signal)
	if got {
		t.Fatal("router negative should NOT break guidance (avoid regex hit)")
	}
}

func TestAnyHardNegative_NilRouterFallsBackToRegex(t *testing.T) {
	route := newGateRoute("bazi", 0.5) // 低置信，regex 兜底启用
	signal := guidance.Signal{}

	got := anyHardNegative(nil, "排个紫微盘", route, signal)
	if !got {
		t.Fatal("nil router + low confidence + 紫微 keyword should break guidance via regex")
	}
}

func TestAnyHardNegative_NilRouterHighConfidenceSkipsRegex(t *testing.T) {
	route := newGateRoute("bazi", 0.9) // 高置信，regex 兜底禁用
	signal := guidance.Signal{}

	got := anyHardNegative(nil, "排个紫微盘看紫微", route, signal)
	if got {
		t.Fatal("nil router + high confidence should skip regex, trust LLM")
	}
}

// Hard negative: 含出生信息的首轮消息不应进入 guidance
func TestShouldEnterGuidance_MessageWithBirthInfoReturnsFalse(t *testing.T) {
	st := state.NewSession("test")
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}
	result := ShouldEnterGuidance(nil, "我是1990年5月生的，男", route, st)
	if result {
		t.Fatal("ShouldEnterGuidance with birth info message should return false")
	}
}

// Hard negative: 显式指定术数方法不应进入 guidance
func TestShouldEnterGuidance_ExplicitMethodReturnsFalse(t *testing.T) {
	st := state.NewSession("test")
	route := policy.ApprovedRoute{
		PrimaryDomain: "ziwei",
		TaskIntent:    "collect_profile",
	}
	result := ShouldEnterGuidance(nil, "用紫微看看我的婚姻", route, st)
	if result {
		t.Fatal("ShouldEnterGuidance with explicit method should return false")
	}
}

// Hard negative: 显式要求执行动作不应进入 guidance
func TestShouldEnterGuidance_ExplicitActionReturnsFalse(t *testing.T) {
	st := state.NewSession("test")
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}
	for _, msg := range []string{
		"帮我算一下八字",
		"帮我看一下运势",
		"排盘分析一下",
	} {
		if ShouldEnterGuidance(nil, msg, route, st) {
			t.Fatalf("ShouldEnterGuidance(nil,%q) should return false", msg)
		}
	}
}

// qimen primary timing: 无 guidance 信号时不应进入（如"今天运气怎么样"）但 fate-adjacent/broad-intent 除外
func TestShouldEnterGuidance_QimenPrimaryTimingReturnsFalse(t *testing.T) {
	st := state.NewSession("test")
	route := policy.ApprovedRoute{
		PrimaryDomain: "qimen",
		TaskIntent:    "fortune_followup",
		PolicyHints: schemas.PolicyHints{
			QimenMode:          "primary",
			ProfileRequirement: "none",
		},
	}
	// 纯 timing 消息：无 guidance 信号 → 不进入
	cases := []string{
		"今天运气怎么样",
		"最近运势如何",
		"什么时候能转运",
	}
	for _, msg := range cases {
		if ShouldEnterGuidance(nil, msg, route, st) {
			t.Fatalf("ShouldEnterGuidance(nil,%q) should return false (pure timing)", msg)
		}
	}
}

// qimen primary 不阻止 fate-adjacent 入场（sniff 信号优先于模型路由）
func TestShouldEnterGuidance_QimenPrimaryDoesNotBlockFateAdjacent(t *testing.T) {
	st := state.NewSession("test")
	route := policy.ApprovedRoute{
		PrimaryDomain: "qimen",
		TaskIntent:    "collect_profile",
		PolicyHints: schemas.PolicyHints{
			QimenMode:          "primary",
			ProfileRequirement: "none",
		},
	}
	if !ShouldEnterGuidance(nil, "最近真是喝凉水都塞牙", route, st) {
		t.Fatal("fate-adjacent should enter guidance even if model routes to qimen primary")
	}
}

// Hard negative: collect_profile / amend_profile 进行中不应回退到 guidance
func TestShouldEnterGuidance_CollectProfileInProgressReturnsFalse(t *testing.T) {
	st := state.NewSession("test")
	for _, intent := range []string{"collect_profile", "amend_profile"} {
		route := policy.ApprovedRoute{
			PrimaryDomain: "bazi",
			TaskIntent:    intent,
		}
		if ShouldEnterGuidance(nil, "我1990年5月20日生的", route, st) {
			t.Fatalf("ShouldEnterGuidance with %s should return false", intent)
		}
	}
}

// Hard positive: fate-adjacent 首轮消息应进入 guidance
func TestShouldEnterGuidance_FateAdjacentReturnsTrue(t *testing.T) {
	st := state.NewSession("test")
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}
	result := ShouldEnterGuidance(nil, "最近真是喝凉水都塞牙", route, st)
	if !result {
		t.Fatal("ShouldEnterGuidance with fate-adjacent message should return true")
	}
}

// Hard positive: broad intent 首轮消息应进入 guidance
func TestShouldEnterGuidance_BroadIntentReturnsTrue(t *testing.T) {
	st := state.NewSession("test")
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}
	result := ShouldEnterGuidance(nil, "看看事业", route, st)
	if !result {
		t.Fatal("ShouldEnterGuidance with broad intent message should return true")
	}
}

// Continuation: active guidance + normal continuation 应允许
func TestShouldEnterGuidance_ActiveGuidanceContinuationReturnsTrue(t *testing.T) {
	st := state.NewSession("test")
	st.Guidance = &state.GuidanceState{DirectiveKind: "offer_consult"}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}
	result := ShouldEnterGuidance(nil, "行，那你看看", route, st)
	if !result {
		t.Fatal("ShouldEnterGuidance with active guidance continuation should return true")
	}
}

// Break guidance: active guidance 中途改口显式执行动作 → false
func TestShouldEnterGuidance_ActiveGuidanceBreaksOnExplicitMethod(t *testing.T) {
	st := state.NewSession("test")
	st.Guidance = &state.GuidanceState{DirectiveKind: "offer_consult"}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}
	result := ShouldEnterGuidance(nil, "用奇门看看", route, st)
	if result {
		t.Fatal("active guidance should break on explicit method")
	}
}

// Break guidance: active guidance 中途给完整出生资料 → false
func TestShouldEnterGuidance_ActiveGuidanceBreaksOnFullBirthInfo(t *testing.T) {
	st := state.NewSession("test")
	st.Guidance = &state.GuidanceState{DirectiveKind: "offer_consult"}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}
	result := ShouldEnterGuidance(nil, "我1990年5月20日早上8点生的，男，北京", route, st)
	if result {
		t.Fatal("active guidance should break on full birth info")
	}
}

// Break guidance: active guidance + qimen primary → break
func TestShouldEnterGuidance_ActiveGuidanceBreaksOnQimenTiming(t *testing.T) {
	st := state.NewSession("test")
	st.Guidance = &state.GuidanceState{DirectiveKind: "choose_topic"}
	route := policy.ApprovedRoute{
		PrimaryDomain: "qimen",
		TaskIntent:    "fortune_followup",
		PolicyHints: schemas.PolicyHints{
			QimenMode:          "primary",
			ProfileRequirement: "none",
		},
	}
	if ShouldEnterGuidance(nil, "今天适合出门吗", route, st) {
		t.Fatal("active guidance should break on qimen timing")
	}
}

// Break guidance: active guidance + sniff timing focus → break
func TestShouldEnterGuidance_ActiveGuidanceBreaksOnSniffTimingFocus(t *testing.T) {
	st := state.NewSession("test")
	st.Guidance = &state.GuidanceState{DirectiveKind: "offer_consult"}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}
	if ShouldEnterGuidance(nil, "最近运势怎么样", route, st) {
		t.Fatal("active guidance should break on sniff timing focus")
	}
}

// Break guidance 续：active guidance 遇到显式"算/看"请求 → false
func TestShouldEnterGuidance_ActiveGuidanceBreaksOnExplicitAction(t *testing.T) {
	st := state.NewSession("test")
	st.Guidance = &state.GuidanceState{DirectiveKind: "choose_topic"}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}
	msgs := []string{"帮我算算八字", "帮我看看财运"}
	for _, msg := range msgs {
		if ShouldEnterGuidance(nil, msg, route, st) {
			t.Fatalf("ShouldEnterGuidance(nil,%q) with active guidance should be false (break guidance)", msg)
		}
	}
}

// 首轮消息含"算/看/排盘"关键词 → false
func TestShouldEnterGuidance_FirstTurnExplicitActionReturnsFalse(t *testing.T) {
	st := state.NewSession("test")
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}
	if ShouldEnterGuidance(nil, "帮我算一下八字", route, st) {
		t.Fatal("'帮我算一下八字' should NOT enter guidance")
	}
	if ShouldEnterGuidance(nil, "帮我看看运势", route, st) {
		t.Fatal("'帮我看看运势' should NOT enter guidance")
	}
}
