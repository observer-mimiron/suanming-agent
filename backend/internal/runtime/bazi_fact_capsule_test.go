package runtime

import (
	"strings"
	"testing"
)

func TestTierEvidenceComplete_RequiresEveryIndependentGround(t *testing.T) {
	all := []string{"qingzhuo", "bingyao", "jiuying", "poge", "hezhizhang"}
	if !tierEvidenceComplete(baziCharterState{EvidenceQuality: baziEvidenceQuality{CoveredTopics: all}}) {
		t.Fatal("complete independent tier evidence should permit tier qualification")
	}
	for _, missing := range all {
		covered := make([]string, 0, len(all)-1)
		for _, topic := range all {
			if topic != missing {
				covered = append(covered, topic)
			}
		}
		if tierEvidenceComplete(baziCharterState{EvidenceQuality: baziEvidenceQuality{CoveredTopics: covered}}) {
			t.Fatalf("missing %q must withhold tier qualification", missing)
		}
	}
}

func TestBaziFactCapsulePromptViewUsesReadableFacts(t *testing.T) {
	state := assertionTestState()
	state.EvidenceQuality.CoveredTopics = []string{"qingzhuo"}
	view := buildBaziFactCapsulePromptView(state, false)
	for _, required := range []string{"月令", "强弱受力", "官星透藏", "火与调候状态", "层次独立证据状态"} {
		if _, ok := view[required]; !ok {
			t.Fatalf("prompt view missing %q: %#v", required, view)
		}
	}
	for _, forbidden := range []string{"support_score", "pressure_score", "fire_effective", "fire_effectiveness_known"} {
		if _, ok := view[forbidden]; ok {
			t.Fatalf("prompt view leaked runtime key %q: %#v", forbidden, view)
		}
	}
	if got := view["火与调候状态"]; got == "" {
		t.Fatalf("fire state must be displayable, got %#v", got)
	}
}

func TestCapsuleTiaohouDisplayExplainsFireWithinSeasonalBoundary(t *testing.T) {
	state := baziCharterState{Input: baziCharterInput{
		BaziResult: map[string]any{
			"dayGan": "甲",
			"pillars": []map[string]any{
				{"name": "年柱", "stem": "辛", "branch": "未", "hideGan": []string{"己"}},
				{"name": "月柱", "stem": "乙", "branch": "亥", "hideGan": []string{"壬", "甲"}},
				{"name": "日柱", "stem": "甲", "branch": "子", "hideGan": []string{"癸"}},
				{"name": "时柱", "stem": "丙", "branch": "寅", "hideGan": []string{"甲", "丙", "戊"}},
			},
		},
	}}

	text := capsuleTiaohouDisplay(buildBaziFactCapsule(state))
	for _, required := range []string{"亥月", "调候作用是否足够"} {
		if !strings.Contains(text, required) {
			t.Fatalf("tiaohou display missing %q: %s", required, text)
		}
	}
	if strings.Contains(text, "有火") {
		t.Fatalf("tiaohou display must not use a bare fire-presence conclusion: %s", text)
	}
}
