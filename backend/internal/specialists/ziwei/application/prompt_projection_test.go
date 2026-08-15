package application

import "testing"

func TestBuildDataBlockPreservesZiweiPromptContract(t *testing.T) {
	got := BuildDataBlock(map[string]any{
		"palaces": []interface{}{
			map[string]interface{}{
				"name": "命宫",
				"major_stars": []interface{}{
					map[string]interface{}{"name": "紫微"},
					map[string]interface{}{"name": "天机"},
				},
			},
			map[string]interface{}{
				"name": "身宫",
				"major_stars": []interface{}{
					map[string]interface{}{"name": "天相"},
				},
			},
		},
		"four_pillars": map[string]interface{}{"年柱": "甲子"},
		"wuxing_ju":    "水二局",
		"liunian":      map[string]interface{}{"year": 2026},
	})

	want := "**命宫主星**：紫微、天机\n" +
		"**身宫主星**：天相\n" +
		"**生年年柱**：甲子\n" +
		"**五行局**：水二局\n" +
		"**流年数据**：{\"year\":2026}\n\n" +
		"**⚠️ 紫微命盘数据已就绪，直接引用解读，禁止调用 ziwei_calc/ziwei_liunian。**\n"
	if got != want {
		t.Fatalf("BuildDataBlock() = %q, want %q", got, want)
	}
}

func TestBuildDataBlockUsesJSONFallbackForSparsePayload(t *testing.T) {
	got := BuildDataBlock(map[string]any{"raw": "value"})
	want := "<!-- 完整紫微命盘 JSON（供推理引用）\n{\"raw\":\"value\"}\n-->\n\n" +
		"**⚠️ 紫微命盘数据已就绪，直接引用解读，禁止调用 ziwei_calc/ziwei_liunian。**\n"
	if got != want {
		t.Fatalf("BuildDataBlock() sparse payload = %q, want %q", got, want)
	}
}

func TestBuildDataBlockReturnsEmptyForNilPayload(t *testing.T) {
	if got := BuildDataBlock(nil); got != "" {
		t.Fatalf("BuildDataBlock(nil) = %q, want empty", got)
	}
}

func TestBuildDataBlockKeepsEmptyPayloadFallback(t *testing.T) {
	got := BuildDataBlock(map[string]any{})
	want := "<!-- 完整紫微命盘 JSON（供推理引用）\n{}\n-->\n\n" +
		"**⚠️ 紫微命盘数据已就绪，直接引用解读，禁止调用 ziwei_calc/ziwei_liunian。**\n"
	if got != want {
		t.Fatalf("BuildDataBlock() empty payload = %q, want %q", got, want)
	}
}
