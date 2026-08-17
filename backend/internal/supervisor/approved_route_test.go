// This test file belongs to the route approval layer.
// It verifies approved route contract behavior and protects the related contract from regressions.
// It approves routes; execution contracts are built later by Manager.
package supervisor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// stubRouter 用预设 MatchResult，避免依赖真 embedder
type stubRouter struct {
	result intent.MatchResult
	err    error
	called bool
}

func (s *stubRouter) Match(ctx context.Context, msg string) (intent.MatchResult, error) {
	s.called = true
	return s.result, s.err
}

func newRouteWithConfidence(primary string, conf float64) policy.ApprovedRoute {
	return policy.ApprovedRoute{
		PrimaryDomain: primary,
		Confidence:    conf,
		PolicyHints:   schemas.PolicyHints{},
	}
}

func TestNormalizeExplicitBirthClock_PreservesUserMinute(t *testing.T) {
	route := policy.ApprovedRoute{Slots: schemas.DecisionSlots{Profile: map[string]any{
		"year": float64(2025), "month": float64(11), "day": float64(10), "hour": float64(23),
	}}}

	normalizeExplicitBirthClock("2025年11月10日23点30分 男 北京", &route)

	if got := route.Slots.Profile["hour"]; got != float64(23) {
		t.Fatalf("hour = %#v, want 23", got)
	}
	if got := route.Slots.Profile["minute"]; got != float64(30) {
		t.Fatalf("minute = %#v, want 30", got)
	}
}

func TestApplyExplicitMethodPreference_EnforcePositiveOverridesLLM(t *testing.T) {
	router := &stubRouter{result: intent.MatchResult{Decision: intent.DecisionPositive, Method: "ziwei"}}
	c := &Client{router: router, routerMode: "enforce"}

	route := newRouteWithConfidence("bazi", 0.9) // LLM 高置信说 bazi
	c.applyExplicitMethodPreference(context.Background(), "排个紫微盘", &route)

	if route.PrimaryDomain != "ziwei" {
		t.Fatalf("PrimaryDomain = %q, want ziwei (router positive overrides LLM)", route.PrimaryDomain)
	}
	if !router.called {
		t.Fatal("router.Match not called in enforce mode")
	}
}

func TestApplyExplicitMethodPreference_EnforceNegativeDoesNotFallbackToRegex(t *testing.T) {
	router := &stubRouter{result: intent.MatchResult{Decision: intent.DecisionNegative}}
	c := &Client{router: router, routerMode: "enforce"}

	route := newRouteWithConfidence("bazi", 0.5)
	c.applyExplicitMethodPreference(context.Background(), "我不看紫微", &route)

	// negative 时不覆盖，不退回 regex——PrimaryDomain 应保持原值 bazi
	if route.PrimaryDomain != "bazi" {
		t.Fatalf("PrimaryDomain = %q, want bazi (negative should not override)", route.PrimaryDomain)
	}
}

func TestApplyExplicitMethodPreference_EnforceErrorFallbackGuardedByConfidence(t *testing.T) {
	router := &stubRouter{err: errors.New("network")}
	c := &Client{router: router, routerMode: "enforce"}

	// 高置信 + err → 不走 regex，信任 LLM
	high := newRouteWithConfidence("bazi", 0.9)
	c.applyExplicitMethodPreference(context.Background(), "排个紫微盘", &high)
	if high.PrimaryDomain != "bazi" {
		t.Fatalf("high confidence + err: PrimaryDomain = %q, want bazi", high.PrimaryDomain)
	}

	// 低置信 + err → 走 regex 兜底，"紫微" 命中 → ziwei
	router.err = errors.New("network")
	router.called = false
	low := newRouteWithConfidence("bazi", 0.5)
	c.applyExplicitMethodPreference(context.Background(), "排个紫微盘看紫微", &low)
	if low.PrimaryDomain != "ziwei" {
		t.Fatalf("low confidence + err: PrimaryDomain = %q, want ziwei (regex fallback)", low.PrimaryDomain)
	}
}

func TestApplyExplicitMethodPreference_ShadowModeDoesNotOverride(t *testing.T) {
	router := &stubRouter{result: intent.MatchResult{Decision: intent.DecisionPositive, Method: "ziwei"}}
	c := &Client{router: router, routerMode: "shadow"}

	route := newRouteWithConfidence("bazi", 0.5)
	c.applyExplicitMethodPreference(context.Background(), "排个紫微盘", &route)

	// shadow 模式 router 跑了但只 log，决策走 regex（"紫微" 命中 regex → ziwei）
	if !router.called {
		t.Fatal("router.Match should be called in shadow mode for logging")
	}
	if route.PrimaryDomain != "ziwei" {
		t.Fatalf("shadow: PrimaryDomain = %q, want ziwei (regex decision)", route.PrimaryDomain)
	}
}

func TestApplyExplicitMethodPreference_OffModeSkipsRouter(t *testing.T) {
	router := &stubRouter{}
	c := &Client{router: router, routerMode: "off"}

	route := newRouteWithConfidence("bazi", 0.5)
	c.applyExplicitMethodPreference(context.Background(), "排个紫微盘看紫微", &route)

	if router.called {
		t.Fatal("router.Match should NOT be called in off mode")
	}
	// off 模式 + 低置信 → 走 regex，"紫微" 命中 → ziwei
	if route.PrimaryDomain != "ziwei" {
		t.Fatalf("off: PrimaryDomain = %q, want ziwei (regex)", route.PrimaryDomain)
	}
}

func TestApplyConsultationContract_UsesFrozenKinds(t *testing.T) {
	cases := []struct {
		name       string
		message    string
		kind       contracts.ConsultationKind
		primary    string
		secondary  []string
		qimenMode  string
		profileReq string
	}{
		{name: "period", message: "本月运势如何", kind: contracts.ConsultationKindPeriodFortune, primary: "bazi", secondary: []string{"ziwei"}, qimenMode: "none", profileReq: "full"},
		{name: "event", message: "这次签约能否成功", kind: contracts.ConsultationKindEventQuestion, primary: "qimen", qimenMode: "primary", profileReq: "none"},
		{name: "instant event", message: "今天运气怎么样", kind: contracts.ConsultationKindEventQuestion, primary: "qimen", qimenMode: "primary", profileReq: "none"},
		{name: "health", message: "最近身体健康如何", kind: contracts.ConsultationKindHealthRisk, primary: "bazi", secondary: []string{"ziwei"}, qimenMode: "none", profileReq: "full"},
		{name: "health method remains health", message: "用八字看看最近身体", kind: contracts.ConsultationKindHealthRisk, primary: "bazi", secondary: []string{"ziwei"}, qimenMode: "none", profileReq: "full"},
		{name: "natal", message: "帮我排个八字命盘", kind: contracts.ConsultationKindNatalChart, primary: "bazi", qimenMode: "none", profileReq: "full"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			route := policy.ApprovedRoute{PrimaryDomain: "qimen", SecondaryDomains: []string{"ziwei"}}
			applyConsultationContract(tc.message, nil, &route)
			if route.ConsultationKind != tc.kind {
				t.Fatalf("ConsultationKind = %q, want %q", route.ConsultationKind, tc.kind)
			}
			if route.PrimaryDomain != tc.primary {
				t.Fatalf("PrimaryDomain = %q, want %q", route.PrimaryDomain, tc.primary)
			}
			if strings.Join(route.SecondaryDomains, ",") != strings.Join(tc.secondary, ",") {
				t.Fatalf("SecondaryDomains = %v, want %v", route.SecondaryDomains, tc.secondary)
			}
			if route.PolicyHints.QimenMode != tc.qimenMode {
				t.Fatalf("QimenMode = %q, want %q", route.PolicyHints.QimenMode, tc.qimenMode)
			}
			if route.PolicyHints.ProfileRequirement != tc.profileReq {
				t.Fatalf("ProfileRequirement = %q, want %q", route.PolicyHints.ProfileRequirement, tc.profileReq)
			}
		})
	}
}

func TestIsEventQuestionKeepsTopicOnlyCooperationOutOfQimen(t *testing.T) {
	if isEventQuestion("合作关系如何") {
		t.Fatal("topic-only cooperation should not become an event question")
	}
	if !isEventQuestion("这个合作项目能不能成") {
		t.Fatal("concrete cooperation event should become an event question")
	}
}

func TestApplyConsultationContractLeavesPreConsultationRoutesUnclassified(t *testing.T) {
	for _, task := range []string{"collect_profile", "amend_profile", "direct_bazi"} {
		route := policy.ApprovedRoute{
			TaskIntent:         task,
			PrimaryDomain:      "bazi",
			NeedsClarification: false,
		}
		applyConsultationContract("分析八字", nil, &route)
		if route.ConsultationKind != "" {
			t.Fatalf("task %q got consultation kind %q", task, route.ConsultationKind)
		}
	}

	route := policy.ApprovedRoute{
		TaskIntent:         "interpret_chart",
		PrimaryDomain:      "bazi",
		NeedsClarification: true,
	}
	applyConsultationContract("本月运势如何", nil, &route)
	if route.ConsultationKind != "" {
		t.Fatalf("clarification route got consultation kind %q", route.ConsultationKind)
	}
}

func TestApplyConsultationContractClearsKindWhenProfileRequiresClarification(t *testing.T) {
	st := state.NewSession("sess-profile-required")
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
	}

	applyConsultationContract("本月运势如何", st, &route)
	if route.ConsultationKind != "" {
		t.Fatalf("ConsultationKind = %q, want empty during clarification", route.ConsultationKind)
	}
	if !route.NeedsClarification || route.Gate.ExecutionMode != "clarify" {
		t.Fatalf("route = %+v, want profile clarification", route)
	}
}

func TestNormalizeBareNumericReplyKeepsExistingBaziDomain(t *testing.T) {
	st := state.NewSession("numeric-reply")
	seedBaziAsset(st)
	route := policy.ApprovedRoute{PrimaryDomain: "ziwei", SecondaryDomains: []string{"bazi"}, TaskIntent: "interpret_chart"}

	normalizeBareNumericReply("1", st, &route)

	if route.PrimaryDomain != "bazi" || route.TaskIntent != "fortune_followup" {
		t.Fatalf("route = %+v, want bazi fortune_followup", route)
	}
	if hasDomain(route.SecondaryDomains, "ziwei") || !route.PolicyHints.CanReuseCachedResult {
		t.Fatalf("route = %+v, numeric reply must not keep ziwei or lose cached reuse", route)
	}
}
