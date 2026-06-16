package runtime

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
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

	result := preflight(st, route)
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

	result := preflight(st, route)
	if result.ShortCircuit {
		t.Fatalf("expected no short circuit for complete profile, got TurnType=%q", result.TurnType)
	}
}

// TestPreflight_QimenPrimaryWithoutProfile 验证奇门主链且 profile_requirement=none 时，
// 即使无资料也放行。
func TestPreflight_QimenPrimaryWithoutProfile(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("qimen", "fortune_followup", "primary", "none")

	result := preflight(st, route)
	if result.ShortCircuit {
		t.Fatalf("qimen primary with profile_requirement=none should not short circuit, got TurnType=%q", result.TurnType)
	}
}

// TestPreflight_QimenPrimaryWithFullProfileRequirementBlocks 验证奇门主链但
// profile_requirement=full 且无资料时，preflight 拦截。
func TestPreflight_QimenPrimaryWithFullProfileRequirementBlocks(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("qimen", "fortune_followup", "primary", "full")

	result := preflight(st, route)
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
		ClarificationQuestion: "请问您想了解哪方面？",
	}

	result := preflight(st, route)
	if !result.ShortCircuit {
		t.Fatal("NeedsClarification route should short circuit")
	}
	if result.TurnType != "clarification" {
		t.Fatalf("TurnType: got %q, want clarification", result.TurnType)
	}
}

// TestPreflight_BaziWithoutProfileOrChartBlocks 验证八字主域无资料且无命盘时，preflight
// 返回 ask_missing_profile（collect_profile/amend_profile 除外）。
func TestPreflight_BaziWithoutProfileOrChartBlocks(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("bazi", "interpret_chart", "none", "none")

	result := preflight(st, route)
	if !result.ShortCircuit {
		t.Fatal("bazi route without profile or chart should short circuit")
	}
	if result.TurnType != "ask_missing_profile" {
		t.Fatalf("TurnType: got %q, want ask_missing_profile", result.TurnType)
	}
}

// TestPreflight_CollectProfileIntentDoesNotBlock 验证 collect_profile 意图即使无资料
// 也放行。
func TestPreflight_CollectProfileIntentDoesNotBlock(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("bazi", "collect_profile", "none", "none")

	result := preflight(st, route)
	if result.ShortCircuit {
		t.Fatal("collect_profile should not short circuit even without profile")
	}
}

// TestPreflight_ZiweiWithoutProfileOrChartBlocks 验证紫微主域无资料且无命盘时拦截。
func TestPreflight_ZiweiWithoutProfileOrChartBlocks(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("ziwei", "fortune_followup", "none", "none")

	result := preflight(st, route)
	if !result.ShortCircuit {
		t.Fatal("ziwei route without profile or chart should short circuit")
	}
}
