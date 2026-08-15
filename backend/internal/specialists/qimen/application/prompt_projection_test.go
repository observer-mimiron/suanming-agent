package application

import "testing"

func TestBuildDataBlockPreservesQimenPromptContract(t *testing.T) {
	got := BuildDataBlock(map[string]any{
		"case_id":          "case-1",
		"purpose":          "event_question",
		"owner_ref":        map[string]any{"kind": "case", "id": "case-1"},
		"question_time":    "2026-08-05T14:30:00+08:00",
		"time_source":      "question_time",
		"symbol_system":    "eight_gate_eight_god",
		"ju_text":          "阳遁一局",
		"value_star":       "天蓬",
		"value_door":       "休门",
		"pan_schema":       "rotating_8",
		"duty_star_palace": "坎",
		"duty_door_palace": "艮",
		"cells": []map[string]any{{
			"palace": "坎", "star": "天蓬", "door": "休门", "god": "值符",
			"guest_gan": "壬", "host_gan": "癸",
		}},
	})

	want := "**Case**：case-1\n" +
		"**问事目的**：event_question\n" +
		"**资产归属**：case/case-1\n" +
		"**提问时间**：2026-08-05T14:30:00+08:00\n" +
		"**起局时间来源**：question_time\n" +
		"**符号体系**：eight_gate_eight_god\n" +
		"**局数**：阳遁一局\n" +
		"**值符星**：天蓬\n" +
		"**值使门**：休门\n" +
		"**盘式口径**：rotating_8\n" +
		"**值符宫**：坎\n" +
		"**值使宫**：艮\n" +
		"**九宫**： 坎(天蓬星/休门门/值符神/天壬地癸)\n\n" +
		"**⚠️ 奇门盘数据已就绪，直接引用解读，禁止调用 qimen_dunjia。**\n"
	if got != want {
		t.Fatalf("BuildDataBlock() = %q, want %q", got, want)
	}
}

func TestBuildDataBlockUsesJSONFallbackForSparsePayload(t *testing.T) {
	got := BuildDataBlock(map[string]any{"raw": "value"})
	want := "<!-- 完整奇门盘 JSON（供推理引用）\n{\"raw\":\"value\"}\n-->\n\n**⚠️ 奇门盘数据已就绪，直接引用解读，禁止调用 qimen_dunjia。**\n"
	if got != want {
		t.Fatalf("BuildDataBlock() sparse payload = %q, want %q", got, want)
	}
}

func TestBuildDataBlockReturnsEmptyForNilPayload(t *testing.T) {
	if got := BuildDataBlock(nil); got != "" {
		t.Fatalf("BuildDataBlock(nil) = %q, want empty", got)
	}
}
