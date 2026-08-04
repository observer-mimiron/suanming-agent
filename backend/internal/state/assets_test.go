// This test file belongs to the session state layer.
// It verifies domain asset state behavior and protects the related contract from regressions.
// It stores session truth; routing and interpretation decisions stay outside state structs.
package state

import "testing"

func completeProfile(year float64) map[string]any {
	return map[string]any{
		"year": year, "month": 10.0, "day": 5.0, "hour": 12.0,
		"gender": "男", "birthplace": "北京",
	}
}

func TestSessionAssets_KeepChartsSeparateAcrossSubjects(t *testing.T) {
	st := NewSession("subjects")
	st.MergeProfile(completeProfile(1991))
	selfRef := st.StoreChart(AssetKindBaziChart, map[string]any{"pillars": "self"}, "test")

	st.SetActiveSubject("孩子")
	st.MergeProfile(completeProfile(2020))
	childRef := st.StoreChart(AssetKindBaziChart, map[string]any{"pillars": "child"}, "test")
	if childRef == selfRef {
		t.Fatal("child chart reused self asset")
	}
	if got := st.ActiveChart(AssetKindBaziChart)["pillars"]; got != "child" {
		t.Fatalf("active child chart = %v, want child", got)
	}

	st.SetActiveSubject("自己")
	if got := st.ActiveChart(AssetKindBaziChart)["pillars"]; got != "self" {
		t.Fatalf("active self chart = %v, want self", got)
	}
	if len(st.Assets) != 2 {
		t.Fatalf("asset count = %d, want 2", len(st.Assets))
	}
}

func TestSessionAssets_ProfileCorrectionCreatesRevisionWithoutOverwritingChart(t *testing.T) {
	st := NewSession("profile-revision")
	st.MergeProfile(completeProfile(1991))
	firstProfile := st.ActiveFocus.ProfileRevisionID
	firstChart := st.StoreChart(AssetKindBaziChart, map[string]any{"pillars": "old"}, "test")

	st.MergeProfile(map[string]any{"hour": 13.0})
	if st.ActiveFocus.ProfileRevisionID == firstProfile {
		t.Fatal("profile correction did not create a new revision")
	}
	if st.ActiveChart(AssetKindBaziChart) != nil {
		t.Fatal("corrected profile unexpectedly reused a chart bound to the old revision")
	}
	if len(st.ProfileRevisions) != 2 || len(st.Assets) != 1 {
		t.Fatalf("revisions/assets = %d/%d, want 2/1", len(st.ProfileRevisions), len(st.Assets))
	}
	asset, ok := st.assetByRef(firstChart)
	if !ok || asset.OwnerID != firstProfile {
		t.Fatalf("old chart owner = %+v, want profile %q", asset, firstProfile)
	}
}

func TestSessionAssets_MigratesAllLegacyChartsFromOneSnapshot(t *testing.T) {
	st := NewSession("legacy")
	st.Profile = completeProfile(1991)
	st.BaziResult = map[string]any{"pillars": "bazi"}
	st.ZiWeiResult = map[string]any{"ming_gong": "ziwei"}
	st.QimenResult = map[string]any{"ju": "qimen"}

	if !st.HasBaziResult() || !st.HasZiWeiResult() || !st.HasQimenResult() {
		t.Fatalf("legacy charts did not all migrate: bazi=%v ziwei=%v qimen=%v", st.HasBaziResult(), st.HasZiWeiResult(), st.HasQimenResult())
	}
	if len(st.Assets) != 3 {
		t.Fatalf("asset count = %d, want 3", len(st.Assets))
	}
}

func TestSessionAssets_QimenFreshCasesDoNotOverwriteEarlierChart(t *testing.T) {
	st := NewSession("qimen-cases")
	first := st.StartCase("qimen", "第一件事", true)
	st.StoreChart(AssetKindQimenChart, map[string]any{"ju": "first"}, "test")
	second := st.StartCase("qimen", "第二件事", true)
	st.StoreChart(AssetKindQimenChart, map[string]any{"ju": "second"}, "test")

	if first.ID == second.ID {
		t.Fatal("fresh qimen case reused prior case")
	}
	if got := st.ActiveChart(AssetKindQimenChart)["ju"]; got != "second" {
		t.Fatalf("active qimen chart = %v, want second", got)
	}
	if len(st.Cases) != 2 || len(st.Assets) != 2 {
		t.Fatalf("cases/assets = %d/%d, want 2/2", len(st.Cases), len(st.Assets))
	}
}

func TestSessionAssets_CloneDoesNotShareNestedPayload(t *testing.T) {
	st := NewSession("clone")
	st.MergeProfile(completeProfile(1991))
	st.StoreChart(AssetKindBaziChart, map[string]any{
		"nested": map[string]any{"value": "original"},
	}, "test")

	clone := st.Clone()
	clone.ActiveChart(AssetKindBaziChart)["nested"].(map[string]any)["value"] = "changed"
	if got := st.ActiveChart(AssetKindBaziChart)["nested"].(map[string]any)["value"]; got != "original" {
		t.Fatalf("original nested payload = %v, clone mutation leaked", got)
	}
}
