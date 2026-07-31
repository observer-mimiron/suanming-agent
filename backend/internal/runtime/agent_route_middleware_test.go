package runtime

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cloudwego/eino/adk/middlewares/reduction"
	"github.com/cloudwego/eino/schema"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

func TestBuildSummarizationMiddleware_NilModel(t *testing.T) {
	mw, err := buildSummarizationMiddleware(nil)
	if err != nil {
		t.Fatalf("nil model should not error: %v", err)
	}
	if mw != nil {
		t.Fatalf("nil model should return nil middleware, got %T", mw)
	}
}

func TestBuildSpecialistHandlers_WithNilSummarizer(t *testing.T) {
	b := &AgentBuilder{}
	handlers := b.buildSpecialistHandlers()
	// summarizerModel is nil → summarization skipped; reduction still present.
	if len(handlers) != 1 {
		t.Fatalf("expected 1 handler (reduction only), got %d", len(handlers))
	}
}

func TestBuildToolReductionMiddleware(t *testing.T) {
	mw, err := buildToolReductionMiddleware()
	if err != nil {
		t.Fatalf("buildToolReductionMiddleware: %v", err)
	}
	if mw == nil {
		t.Fatal("expected non-nil middleware")
	}
}

func TestKnowledgeSearchTruncHandler_UnderLimit(t *testing.T) {
	short := strings.Repeat("甲", 100)
	detail := &reduction.ToolDetail{
		ToolResult: &schema.ToolResult{
			Parts: []schema.ToolOutputPart{
				{Type: schema.ToolPartTypeText, Text: short},
			},
		},
	}
	res, err := knowledgeSearchTruncHandler(context.Background(), detail)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.NeedTrunc {
		t.Fatal("short result should not trigger truncation")
	}
}

func TestKnowledgeSearchTruncHandler_OverLimit(t *testing.T) {
	long := strings.Repeat("乙", knowledgeSearchMaxRunes+500)
	detail := &reduction.ToolDetail{
		ToolResult: &schema.ToolResult{
			Parts: []schema.ToolOutputPart{
				{Type: schema.ToolPartTypeText, Text: long},
			},
		},
	}
	res, err := knowledgeSearchTruncHandler(context.Background(), detail)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !res.NeedTrunc {
		t.Fatal("long result should trigger truncation")
	}
	if res.NeedOffload {
		t.Fatal("should not offload (no Backend)")
	}
	if res.ToolResult == nil {
		t.Fatal("truncated ToolResult must be set")
	}
	truncText := res.ToolResult.Parts[0].Text
	if !strings.Contains(truncText, "已截断") {
		t.Fatalf("truncated text should contain notice")
	}
	// truncated rune count (excluding notice) should not exceed limit
	// notice is appended after the cut, so total may slightly exceed — check the cut itself
	if utf8.RuneCountInString(truncText) > knowledgeSearchMaxRunes+50 {
		t.Fatalf("truncated text too long: %d runes", utf8.RuneCountInString(truncText))
	}
}

func TestKnowledgeSearchTruncHandler_MixedParts(t *testing.T) {
	textPart := strings.Repeat("丙", knowledgeSearchMaxRunes+100)
	detail := &reduction.ToolDetail{
		ToolResult: &schema.ToolResult{
			Parts: []schema.ToolOutputPart{
				{Type: schema.ToolPartTypeImage, Image: &schema.ToolOutputImage{}},
				{Type: schema.ToolPartTypeText, Text: textPart},
			},
		},
	}
	res, err := knowledgeSearchTruncHandler(context.Background(), detail)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !res.NeedTrunc {
		t.Fatal("should trigger truncation")
	}
	// image part preserved unchanged
	if res.ToolResult.Parts[0].Type != schema.ToolPartTypeImage {
		t.Fatal("image part should be preserved")
	}
	if res.ToolResult.Parts[0].Image == nil {
		t.Fatal("image content should be preserved")
	}
	// text part truncated
	if !strings.Contains(res.ToolResult.Parts[1].Text, "已截断") {
		t.Fatal("text part should be truncated with notice")
	}
}

func TestKnowledgeSearchTruncHandler_NilDetail(t *testing.T) {
	res, err := knowledgeSearchTruncHandler(context.Background(), nil)
	if err != nil {
		t.Fatalf("nil detail should not error: %v", err)
	}
	if res.NeedTrunc {
		t.Fatal("nil detail should not trigger truncation")
	}
}

func TestBuildBaziDataBlock_InjectsStructuredShenshaContext(t *testing.T) {
	b := &AgentBuilder{}
	st := &state.SessionState{
		BaziResult: map[string]any{
			"pillars": []map[string]any{
				{"name": "年柱", "stem": "甲", "branch": "子", "shiShen": "偏印"},
				{"name": "月柱", "stem": "丙", "branch": "寅", "shiShen": "食神"},
				{"name": "日柱", "stem": "戊", "branch": "午", "shiShen": "日主"},
				{"name": "时柱", "stem": "庚", "branch": "申", "shiShen": "偏财"},
			},
			"dayGan": "戊",
			"wuxing": map[string]int{"木": 1, "火": 2, "土": 2, "金": 2, "水": 1},
			"shensha_summary": map[string]any{
				"all": []string{"桃花", "将星"},
				"by_pillar": map[string][]map[string]any{
					"年柱": {{"name": "桃花", "tone": "neutral", "basis": "年支", "description": "主人缘、魅力、感情机会"}},
					"月柱": {},
					"日柱": {{"name": "将星", "tone": "neutral", "basis": "日支", "description": "主领导力、掌权"}},
					"时柱": {},
				},
			},
		},
	}

	block := b.buildBaziDataBlock(st)
	if !strings.Contains(block, "主要神煞") {
		t.Fatalf("expected 主要神煞 section, got: %s", block)
	}
	if !strings.Contains(block, "按柱神煞") {
		t.Fatalf("expected 按柱神煞 section, got: %s", block)
	}
	if !strings.Contains(block, "年柱：桃花[平/年支]") {
		t.Fatalf("expected 年柱 structured shensha line, got: %s", block)
	}
	if !strings.Contains(block, "神煞作为辅助佐证") {
		t.Fatalf("expected shensha guidance note, got: %s", block)
	}
	if !strings.Contains(block, "先看月令、旺衰、格局、用神") {
		t.Fatalf("expected shensha methodology ordering, got: %s", block)
	}
	if !strings.Contains(block, "只有当神煞与原局结构、大运、流年同向印证时") {
		t.Fatalf("expected shensha corroboration rule, got: %s", block)
	}
	if !strings.Contains(block, "禁止仅凭单一神煞直接断必然吉凶") {
		t.Fatalf("expected shensha output constraint, got: %s", block)
	}
}
