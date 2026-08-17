package domain

import (
	"strings"
	"testing"
)

func TestBuildFactCapsuleKeepsDeterministicFactsSeparate(t *testing.T) {
	capsule := BuildFactCapsule(FactInput{
		BaziResult: map[string]any{
			"dayGan": "甲",
			"pillars": []map[string]any{
				{"name": "年柱", "stem": "辛", "branch": "未", "hideGan": []string{"己"}},
				{"name": "月柱", "stem": "乙", "branch": "亥", "hideGan": []string{"壬", "甲"}},
				{"name": "日柱", "stem": "甲", "branch": "子", "hideGan": []string{"癸"}},
				{"name": "时柱", "stem": "丙", "branch": "寅", "hideGan": []string{"甲", "丙", "戊"}},
			},
		},
		Yongshen: map[string]any{
			"day_master":     "甲",
			"month_score":    2,
			"root_count":     2,
			"same_element":   1,
			"generate_count": 1,
			"strength_evidence": map[string]any{
				"support_score":    3,
				"pressure_score":   4,
				"support_signals":  []string{"通根", "印星"},
				"pressure_signals": []string{"泄耗"},
			},
			"official_visibility": map[string]any{
				"visible": []map[string]any{{"stem": "辛"}},
			},
			"fire_status": map[string]any{
				"effective": true,
			},
		},
		MonthCommand:     "亥月",
		CurrentPeriodRef: "dayun[1]",
		CurrentPeriod: map[string]any{
			"ganZhi":        "甲午",
			"dayun_chonghe": []map[string]any{{"description": "甲午与原局有已计算关系"}},
		},
	})

	if !capsule.CoreFactsReady || !capsule.FireEffective || !capsule.FireEffectivenessKnown {
		t.Fatalf("core and fire facts = %+v", capsule)
	}
	if capsule.CurrentPeriodRef != "dayun[1]" || capsule.CurrentPeriodGanZhi != "甲午" {
		t.Fatalf("current period = %+v", capsule)
	}
	if !capsule.OfficialVisible || capsule.OfficialHidden {
		t.Fatalf("official visibility = %+v", capsule)
	}
	if len(capsule.RootPositions) != 2 || len(capsule.VisibleSameElementStems) != 1 {
		t.Fatalf("root/same-element facts = %+v", capsule)
	}
}

func TestBuildPromptViewUsesReadableLabelsOnly(t *testing.T) {
	view := BuildPromptView(FactInput{
		BaziResult:       map[string]any{"dayGan": "甲"},
		Yongshen:         map[string]any{"day_master": "甲"},
		MonthCommand:     "亥月",
		CurrentPeriod:    map[string]any{"ganZhi": "甲午"},
		CurrentPeriodRef: "dayun[1]",
	}, true)
	for _, key := range []string{"月令", "强弱受力", "官星透藏", "火与调候状态", "当前大运"} {
		if _, ok := view[key]; !ok {
			t.Fatalf("missing readable key %q: %#v", key, view)
		}
	}
	for _, key := range []string{"support_score", "fire_effective", "current_period_ref"} {
		if _, ok := view[key]; ok {
			t.Fatalf("leaked internal key %q: %#v", key, view)
		}
	}
	if !strings.Contains(view["当前大运"].(string), "甲午") {
		t.Fatalf("current period view = %#v", view["当前大运"])
	}
}
