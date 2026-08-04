// This test file belongs to the route approval layer.
// It verifies approved route contract behavior and protects the related contract from regressions.
// It approves routes; execution contracts are built later by Manager.
package supervisor

import (
	"context"
	"errors"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
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
