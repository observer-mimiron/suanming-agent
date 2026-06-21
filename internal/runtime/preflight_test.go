package runtime

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/guidance"
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

	result := preflight(st, &route, "")
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

	result := preflight(st, &route, "")
	if result.ShortCircuit {
		t.Fatalf("expected no short circuit for complete profile, got TurnType=%q", result.TurnType)
	}
}

// TestPreflight_QimenPrimaryWithoutProfile 验证奇门主链且 profile_requirement=none 时，
// 即使无资料也放行。
func TestPreflight_QimenPrimaryWithoutProfile(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("qimen", "fortune_followup", "primary", "none")

	result := preflight(st, &route, "")
	if result.ShortCircuit {
		t.Fatalf("qimen primary with profile_requirement=none should not short circuit, got TurnType=%q", result.TurnType)
	}
}

// TestPreflight_QimenPrimaryWithFullProfileRequirementBlocks 验证奇门主链但
// profile_requirement=full 且无资料时，preflight 拦截。
func TestPreflight_QimenPrimaryWithFullProfileRequirementBlocks(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("qimen", "fortune_followup", "primary", "full")

	result := preflight(st, &route, "")
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

	result := preflight(st, &route, "")
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

func TestPreflight_DirectiveShortCircuitsThroughRenderer(t *testing.T) {
	st := makeSession(true, true, false)
	route := policy.ApprovedRoute{
		NeedsClarification:    true,
		ClarificationQuestion: "请确认一下您的需求。",
		Directive: &schemas.ConversationDirective{
			Kind:      "choose_topic",
			Reason:    "broad_intent",
			OptionSet: "top_topics",
		},
	}

	result := preflight(st, &route, "我想了解财运")
	if !result.ShortCircuit {
		t.Fatal("directive route should short circuit")
	}
	if result.TurnType != "clarification" {
		t.Fatalf("TurnType: got %q, want clarification", result.TurnType)
	}
	want := guidance.Render(guidance.Request{Directive: route.Directive})
	if result.Text != want {
		t.Fatalf("Text: got %q, want %q", result.Text, want)
	}
}

// TestPreflight_BaziWithoutProfileOrChartBlocks 验证八字主域无资料且无命盘时，preflight
// 返回 ask_missing_profile（collect_profile/amend_profile 除外）。
func TestPreflight_BaziWithoutProfileOrChartBlocks(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("bazi", "interpret_chart", "none", "none")

	result := preflight(st, &route, "")
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

	result := preflight(st, &route, "")
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

	result := preflight(st, &route, "")
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

	result := preflight(st, &route, "")
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

func TestPreflight_ZiweiMissingProfileUsesBoundaryRenderer(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("ziwei", "fortune_followup", "none", "none")

	result := preflight(st, &route, "")
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

	result := preflight(st, &route, "")
	if !result.ShortCircuit {
		t.Fatal("collect_profile with empty profile should short circuit")
	}
}

// TestPreflight_ZiweiCollectProfileShortCircuits 验证 ziwei 主域 + collect_profile
// + 空资料时短路返回 ziwei 追问模板。
func TestPreflight_ZiweiCollectProfileShortCircuits(t *testing.T) {
	st := makeSession(false, false, false)
	route := routeWithHints("ziwei", "collect_profile", "none", "none")

	result := preflight(st, &route, "")
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
