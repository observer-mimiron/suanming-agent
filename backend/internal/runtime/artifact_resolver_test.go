package runtime

import (
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

func TestResolveArtifactFocus_ExplicitSubjectSelectsExistingProfile(t *testing.T) {
	st := state.NewSession("resolver-subject")
	st.MergeProfile(completeProfileForResolver(1991))
	st.SetActiveSubject("孩子")
	st.MergeProfile(completeProfileForResolver(2020))
	st.SetActiveSubject("自己")

	plan := (&Manager{}).BuildExecutionPlan(st, policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "interpret_chart",
		Slots: schemas.DecisionSlots{
			TargetSubject: "孩子",
		},
	}, "看看孩子的命盘")

	if got := st.ActiveSubject().Display; got != "孩子" {
		t.Fatalf("active subject = %q, want 孩子", got)
	}
	if got := st.ActiveProfile()["year"]; got != 2020.0 {
		t.Fatalf("active profile year = %v, want 2020", got)
	}
	if len(plan.Requirements) != 1 || len(plan.Requirements[0].SubjectIDs) != 1 || plan.Requirements[0].SubjectIDs[0] == "" {
		t.Fatalf("requirements = %+v, want one exact subject reference", plan.Requirements)
	}
}

func TestResolveArtifactFocus_AmbiguousPronounClarifiesOnlyWithMultipleSubjects(t *testing.T) {
	st := state.NewSession("resolver-ambiguous")
	st.MergeProfile(completeProfileForResolver(1991))
	st.SetActiveSubject("孩子")
	st.MergeProfile(completeProfileForResolver(2020))

	plan := (&Manager{}).BuildExecutionPlan(st, policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
		Slots:         schemas.DecisionSlots{TargetSubject: "他"},
	}, "他今年怎么样")

	if !plan.Route.NeedsClarification {
		t.Fatal("ambiguous pronoun with multiple subjects did not request clarification")
	}
	if plan.Route.ClarificationQuestion == "" {
		t.Fatal("clarification question is empty")
	}
}

func TestResolveArtifactFocus_SingleSubjectPronounKeepsActiveFocus(t *testing.T) {
	st := state.NewSession("resolver-single")
	st.MergeProfile(completeProfileForResolver(1991))

	plan := (&Manager{}).BuildExecutionPlan(st, policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
		Slots:         schemas.DecisionSlots{TargetSubject: "他"},
	}, "他今年怎么样")

	if plan.Route.NeedsClarification {
		t.Fatal("single active subject should not require clarification")
	}
	if got := st.ActiveSubject().Display; got != "自己" {
		t.Fatalf("active subject = %q, want 自己", got)
	}
}

func TestResolveArtifactFocus_QimenPrimaryCreatesFreshCase(t *testing.T) {
	st := state.NewSession("resolver-qimen")
	manager := &Manager{}
	route := policy.ApprovedRoute{
		PrimaryDomain: "qimen",
		TaskIntent:    "interpret_chart",
		PolicyHints:   schemas.PolicyHints{QimenMode: "primary"},
	}

	first := manager.BuildExecutionPlan(st, route, "第一件事是否顺利")
	firstCase := st.ActiveFocus.CaseID
	second := manager.BuildExecutionPlan(st, route, "第二件事是否顺利")
	if firstCase == "" || st.ActiveFocus.CaseID == firstCase {
		t.Fatalf("qimen case ids = %q/%q, want distinct", firstCase, st.ActiveFocus.CaseID)
	}
	if len(st.Cases) != 2 {
		t.Fatalf("case count = %d, want 2", len(st.Cases))
	}
	if len(first.Requirements) != 1 || len(second.Requirements) != 1 || first.Requirements[0].OwnerRef.ID == second.Requirements[0].OwnerRef.ID {
		t.Fatalf("qimen requirements = %+v / %+v, want distinct exact case owners", first.Requirements, second.Requirements)
	}
}

func TestValidatePlanArtifacts_RejectsChartOwnedByAnotherSubject(t *testing.T) {
	st := state.NewSession("resolver-owner")
	st.MergeProfile(completeProfileForResolver(1991))
	selfRef := st.StoreChart(state.AssetKindBaziChart, map[string]any{
		"calendar_rule_version": "zi_zheng_v1",
	}, "test")
	st.SetActiveSubject("孩子")
	st.MergeProfile(completeProfileForResolver(2020))

	plan := (&Manager{}).BuildExecutionPlan(st, policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "interpret_chart",
		Slots:         schemas.DecisionSlots{TargetSubject: "孩子"},
	}, "看孩子的八字")
	// Simulate a stale selection pointer. Presence of a bazi chart alone must not
	// satisfy the child's exact profile/subject requirement.
	st.ActiveFocus.PrimaryAssetRefs = []state.AssetRef{selfRef}
	if err := validatePlanArtifacts(st, plan); err == nil {
		t.Fatal("chart owned by another subject satisfied the exact requirement")
	}

	st.StoreChart(state.AssetKindBaziChart, map[string]any{
		"calendar_rule_version": "zi_zheng_v1",
	}, "test")
	if err := validatePlanArtifacts(st, plan); err != nil {
		t.Fatalf("child chart should satisfy exact requirement: %v", err)
	}
}

func TestFollowupInterpretation_IsBoundToActiveSubjectChart(t *testing.T) {
	st := state.NewSession("followup-subject")
	st.MergeProfile(completeProfileForResolver(1991))
	st.StoreChart(state.AssetKindBaziChart, map[string]any{"calendar_rule_version": "zi_zheng_v1"}, "test")
	storeFollowupArtifact(st, policy.ApprovedRoute{PrimaryDomain: "bazi"}, specialists.Result{
		Domain: "bazi", Summary: "自己的旧解读",
	}, "自己的旧解读", "自己事业如何", "agent_reading")
	if _, ok := loadFollowupArtifact(st, "bazi"); !ok {
		t.Fatal("self interpretation was not stored against self chart")
	}

	st.SetActiveSubject("孩子")
	st.MergeProfile(completeProfileForResolver(2020))
	st.StoreChart(state.AssetKindBaziChart, map[string]any{"calendar_rule_version": "zi_zheng_v1"}, "test")
	if _, ok := loadFollowupArtifact(st, "bazi"); ok {
		t.Fatal("child chart reused self interpretation")
	}

	st.SetActiveSubject("自己")
	if artifact, ok := loadFollowupArtifact(st, "bazi"); !ok || artifact.Summary != "自己的旧解读" {
		t.Fatalf("self interpretation after switch = %+v, want original asset", artifact)
	}
}

// TestExecutionPlan_SubjectAssetConversationRegression preserves the real
// conversation failure sequence: one session can switch people, correct a
// birth time and ask separate QiMen questions without any chart or reading
// leaking across those identity and input boundaries.
func TestExecutionPlan_SubjectAssetConversationRegression(t *testing.T) {
	st := state.NewSession("subject-asset-conversation")
	manager := &Manager{}

	// "我" has a chart and an interpretation that later follow-ups may reuse.
	st.MergeProfile(completeProfileForResolver(1991))
	selfChart := st.StoreChart(state.AssetKindBaziChart, map[string]any{
		"calendar_rule_version": "zi_zheng_v1", "pillars": "self-v1",
	}, "test")
	storeFollowupArtifact(st, policy.ApprovedRoute{PrimaryDomain: "bazi"}, specialists.Result{
		Domain: "bazi", Summary: "自己的事业解读",
	}, "自己的事业解读", "我事业如何", "agent_reading")

	// "孩子" is a distinct Subject. Its first chart must not satisfy the
	// corrected profile that follows.
	st.SetActiveSubject("孩子")
	st.MergeProfile(completeProfileForResolver(2020))
	childV1 := st.StoreChart(state.AssetKindBaziChart, map[string]any{
		"calendar_rule_version": "zi_zheng_v1", "pillars": "child-v1",
	}, "test")
	childPlan := manager.BuildExecutionPlan(st, policy.ApprovedRoute{
		PrimaryDomain: "bazi", TaskIntent: "fortune_followup",
		Slots: schemas.DecisionSlots{TargetSubject: "孩子"},
	}, "看看孩子今年怎么样")
	if len(childPlan.Requirements) != 1 || childPlan.Requirements[0].OwnerRef.ID == "" {
		t.Fatalf("child requirement = %+v, want exact profile owner", childPlan.Requirements)
	}
	if err := validatePlanArtifacts(st, childPlan); err != nil {
		t.Fatalf("child v1 chart should satisfy child v1 plan: %v", err)
	}
	if _, ok := loadFollowupArtifact(st, "bazi"); ok {
		t.Fatal("child follow-up reused the self interpretation")
	}

	oldChildProfile := st.ActiveFocus.ProfileRevisionID
	st.MergeProfile(map[string]any{"hour": 13.0})
	correctedPlan := manager.BuildExecutionPlan(st, policy.ApprovedRoute{
		PrimaryDomain: "bazi", TaskIntent: "interpret_chart",
		Slots: schemas.DecisionSlots{TargetSubject: "孩子"},
	}, "更正一下，孩子是未时")
	if st.ActiveFocus.ProfileRevisionID == oldChildProfile {
		t.Fatal("birth correction did not create a child profile revision")
	}
	if err := validatePlanArtifacts(st, correctedPlan); err == nil {
		t.Fatal("corrected child profile reused its old chart")
	}
	childV2 := st.StoreChart(state.AssetKindBaziChart, map[string]any{
		"calendar_rule_version": "zi_zheng_v1", "pillars": "child-v2",
	}, "test")
	if childV1 == childV2 {
		t.Fatal("corrected child profile overwrote its original chart")
	}
	if err := validatePlanArtifacts(st, correctedPlan); err != nil {
		t.Fatalf("child v2 chart should satisfy corrected plan: %v", err)
	}

	// Returning to "自己" restores only the self chart and its matching reading.
	manager.BuildExecutionPlan(st, policy.ApprovedRoute{
		PrimaryDomain: "bazi", TaskIntent: "fortune_followup",
		Slots: schemas.DecisionSlots{TargetSubject: "自己"},
	}, "回来看我自己的事业")
	if st.ActiveChart(state.AssetKindBaziChart)["pillars"] != "self-v1" {
		t.Fatalf("active self chart = %v, want self-v1", st.ActiveChart(state.AssetKindBaziChart))
	}
	if refs := st.ActiveFocus.PrimaryAssetRefs; len(refs) != 1 || refs[0] != selfChart {
		t.Fatalf("active assets = %+v, want %v", refs, selfChart)
	}
	if reading, ok := loadFollowupArtifact(st, "bazi"); !ok || reading.Summary != "自己的事业解读" {
		t.Fatalf("self interpretation = %+v, want original self reading", reading)
	}

	// Each new primary QiMen question creates a Case, not a replacement chart.
	qimenRoute := policy.ApprovedRoute{
		PrimaryDomain: "qimen", TaskIntent: "interpret_chart",
		PolicyHints: schemas.PolicyHints{QimenMode: "primary"},
	}
	firstQimen := manager.BuildExecutionPlan(st, qimenRoute, "第一件事是否顺利")
	firstCaseID := firstQimen.Requirements[0].OwnerRef.ID
	st.StoreChart(state.AssetKindQimenChart, map[string]any{"ju": "first"}, "test")
	secondQimen := manager.BuildExecutionPlan(st, qimenRoute, "第二件事是否顺利")
	secondCaseID := secondQimen.Requirements[0].OwnerRef.ID
	st.StoreChart(state.AssetKindQimenChart, map[string]any{"ju": "second"}, "test")
	if firstCaseID == secondCaseID || len(st.Cases) != 2 {
		t.Fatalf("qimen cases = %q/%q (%d), want two distinct cases", firstCaseID, secondCaseID, len(st.Cases))
	}
	if len(st.Assets) != 6 { // three natal charts, one reading, two QiMen charts
		t.Fatalf("asset count = %d, want 6", len(st.Assets))
	}
}

func completeProfileForResolver(year float64) map[string]any {
	return map[string]any{
		"year": year, "month": 10.0, "day": 5.0, "hour": 12.0,
		"gender": "男", "birthplace": "北京",
	}
}
