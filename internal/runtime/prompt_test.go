package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/state"
)

func TestCurrentQuestion(t *testing.T) {
	st := state.NewSession("s1")
	if got := CurrentQuestion(st); got == "" {
		t.Fatal("CurrentQuestion should return a default prompt when session question is empty")
	}

	st.LastUserQuestion = "看看事业"
	if got := CurrentQuestion(st); got != "看看事业" {
		t.Fatalf("CurrentQuestion() = %q, want 看看事业", got)
	}
}

func TestBuildKnowledgeQuery_QimenWithoutBaziUsesQimenTerms(t *testing.T) {
	st := state.NewSession("s2")
	st.LastUserQuestion = "现在适合换工作吗"
	st.QimenResult = map[string]any{
		"value_star": "天辅",
		"value_door": "开门",
		"ju_text":    "阳遁三局",
	}

	builder := NewBuilder("soft")
	got := builder.BuildKnowledgeQuery(nil, st, "qimen")
	if got == "" {
		t.Fatal("BuildKnowledgeQuery should return qimen terms")
	}
	if want := "奇门遁甲"; got[:len(want)] != want {
		t.Fatalf("BuildKnowledgeQuery() = %q, want prefix %q", got, want)
	}
}

func TestBuildZiweiKnowledgeQuery_WithChart(t *testing.T) {
	st := state.NewSession("test-zw-001")
	st.ZiWeiResult = map[string]any{
		"gender":                "男",
		"five_elements_class":   "火六局",
		"soul_palace_branch":    "子",
		"body_palace_branch":    "午",
		"soul_master":           "贪狼",
		"body_master":           "天同",
		"four_pillars":          map[string]string{"年柱": "庚午", "月柱": "辛巳", "日柱": "乙酉", "时柱": "庚辰"},
		"palaces": []any{
			map[string]any{
				"name":          "命宫",
				"is_body_palace": false,
				"major_stars": []any{
					map[string]any{"name": "紫微", "type": "主星", "brightness": "庙", "mutagen": "化科"},
				},
			},
		},
	}
	st.RecordTurn("user", "我的财运怎么样")

	b := NewBuilder("")
	query := b.BuildKnowledgeQuery(context.Background(), st, "ziwei")

	if query == "" {
		t.Error("query should not be empty")
	}
	if !strings.Contains(query, "紫微斗数") {
		t.Errorf("query should contain 紫微斗数, got: %s", query)
	}
	// Should extract 命宫 main star name
	if !strings.Contains(query, "紫微") {
		t.Errorf("query should contain main star 紫微, got: %s", query)
	}
}

func TestBuildAgentInstruction_ContainsXMLTagInstructions(t *testing.T) {
	st := state.NewSession("s_test")
	st.Profile = map[string]any{
		"year": float64(2000), "month": float64(1), "day": float64(1),
		"hour": float64(12), "gender": "男",
	}
	b := NewBuilder("default")
	result := b.BuildAgentInstruction(st, "bazi")
	if !strings.Contains(result, "<analysis>") || !strings.Contains(result, "<response>") {
		t.Fatal("BuildAgentInstruction should mention <analysis> and <response> tags")
	}
	if !strings.Contains(result, "严禁逐项") {
		t.Fatal("BuildAgentInstruction should contain anti-duplication instruction")
	}
}
