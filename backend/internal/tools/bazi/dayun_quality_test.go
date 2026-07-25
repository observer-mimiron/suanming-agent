package bazi

import "testing"

func TestAssessDayunQuality_DowngradesFlatBaseWhenCorePillarsAreHit(t *testing.T) {
	chonghe := []map[string]string{
		{"type": "六冲", "pillars": "大运日柱亥", "description": "大运巳亥冲日柱亥"},
		{"type": "相害", "pillars": "大运月柱辰", "description": "大运卯辰害月柱辰"},
	}

	got := assessDayunQuality("平", chonghe)
	if got != "偏压" && got != "凶" {
		t.Fatalf("expected downgraded pressure label, got %s", got)
	}
}

func TestAssessDayunQuality_KeepsSupportiveLuckPositive(t *testing.T) {
	chonghe := []map[string]string{
		{"type": "三合", "pillars": "年柱大运月柱", "description": "大运参与寅午戌合火局"},
	}

	got := assessDayunQuality("大吉", chonghe)
	if got != "大吉" {
		t.Fatalf("expected supportive luck to stay positive, got %s", got)
	}
}

func TestQualitySummary_ExplainsBaseAndBranchAdjustment(t *testing.T) {
	chonghe := []map[string]string{
		{"type": "六冲", "pillars": "大运日柱亥", "description": "大运巳亥冲日柱亥"},
	}

	got := qualitySummary("平", "偏压", chonghe)
	if got == "" {
		t.Fatal("expected non-empty summary")
	}
	if !hasNegativeDayunRelation(chonghe) {
		t.Fatal("expected negative relation detection")
	}
	if !touchesCorePillars(chonghe) {
		t.Fatal("expected core pillar detection")
	}
}
