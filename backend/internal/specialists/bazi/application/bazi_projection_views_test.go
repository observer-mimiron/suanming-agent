package application

import (
	"testing"

	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
)

func TestBuildDynamicFactsViewKeepsOnlyModelFields(t *testing.T) {
	view := BuildDynamicFactsView(bazidomain.CharterInput{
		Dayun: map[string]any{
			"periods": []any{"甲子"}, "current_dayun": "甲子", "private": "drop",
		},
		Liunian: map[string]any{
			"liunian_year": 2026, "liunian_ganzhi": "丙午", "private": "drop",
		},
	})
	dayun, _ := view["dayun"].(map[string]any)
	liunian, _ := view["liunian"].(map[string]any)
	if dayun["private"] != nil || liunian["private"] != nil {
		t.Fatalf("dynamic view leaked non-model fields: %#v", view)
	}
	if dayun["current_dayun"] != "甲子" || liunian["liunian_year"] != 2026 {
		t.Fatalf("dynamic view lost calculated facts: %#v", view)
	}
}

func TestNormalizeByAliasCanonicalizesPlannerMode(t *testing.T) {
	if got := NormalizeByAlias("普通分析", map[string]string{"普通分析": "analysis"}); got != "analysis" {
		t.Fatalf("NormalizeByAlias() = %q, want analysis", got)
	}
}

func TestProjectCanonicalStaticSynthesisKeepsVerifiedTiaohouVerdictWhenEffectivenessUnknown(t *testing.T) {
	static := projectCanonicalStaticSynthesis(
		baziCharterState{Input: baziCharterInput{
			BaziResult: map[string]any{
				"dayGan": "戊",
				"pillars": []any{
					map[string]any{"name": "年柱", "stem": "辛", "branch": "未", "hideGan": []any{"己", "丁", "乙"}},
					map[string]any{"name": "月柱", "stem": "丁", "branch": "酉", "hideGan": []any{"辛"}},
					map[string]any{"name": "日柱", "stem": "戊", "branch": "申", "hideGan": []any{"庚", "壬", "戊"}},
					map[string]any{"name": "时柱", "stem": "己", "branch": "未", "hideGan": []any{"己", "丁", "乙"}},
				},
			},
			Yongshen: map[string]any{"day_master": "戊"},
		}},
		baziCanonicalSynthesis{Tiaohou: baziCanonicalUnit{
			Verdict:  "时干透火但根气不足，调候之力有限，层次受此制约。",
			Boundary: "调候先看月令与火的有效性。",
		}},
	)
	want := "时干透火但根气不足，调候之力有限，层次受此制约。"
	if static.TiaohouAnchor != want {
		t.Fatalf("tiaohou anchor = %q, want %q", static.TiaohouAnchor, want)
	}
}

func TestProjectCanonicalStaticSynthesisKeepsVerdictWithVerifiedTiaohouFact(t *testing.T) {
	static := projectCanonicalStaticSynthesis(
		baziCharterState{Input: baziCharterInput{Yongshen: map[string]any{
			"tiaohou_fire": map[string]any{"effective": true},
		}}},
		baziCanonicalSynthesis{Tiaohou: baziCanonicalUnit{
			Verdict:  "冬令火透，可参与温养调候。",
			Boundary: "只确认火可参与调候，不替代完整取用裁断。",
		}},
	)
	if got, want := static.TiaohouAnchor, "冬令火透，可参与温养调候。"; got != want {
		t.Fatalf("tiaohou anchor = %q, want %q", got, want)
	}
}

func TestProjectCanonicalDynamicSynthesisUsesFactsOnlyForOmittedVerdicts(t *testing.T) {
	state := baziCharterState{AnalysisPlan: baziAnalysisPlan{NeedDynamic: true}}
	static := baziStaticSynthesis{}

	dynamic := projectCanonicalDynamicSynthesis(state, baziCanonicalSynthesis{Source: "model"}, static)

	if dynamic.Source != bazidomain.FactsOnlySource {
		t.Fatalf("dynamic source = %q, want %q", dynamic.Source, bazidomain.FactsOnlySource)
	}
}
