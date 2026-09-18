package runtime

import (
	"context"
	"testing"
)

func TestEmitChartFromToolResult_MapsOnlyChartTools(t *testing.T) {
	tests := []struct {
		name       string
		toolName   string
		result     string
		wantType   string
		wantEvents int
	}{
		{name: "bazi", toolName: "bazi_calc", result: `{"year":2026}`, wantType: "bazi-chart", wantEvents: 1},
		{name: "qimen", toolName: "qimen_dunjia", result: `{"case_id":"c1"}`, wantType: "qimen-chart", wantEvents: 1},
		{name: "ziwei", toolName: "ziwei_calc", result: `{"命宫":"午"}`, wantType: "ziwei-chart", wantEvents: 1},
		{name: "ziwei liunian", toolName: "ziwei_liunian", result: `{"year":2026}`, wantType: "ziwei-chart", wantEvents: 1},
		{name: "invalid json", toolName: "bazi_calc", result: "not-json", wantEvents: 0},
		{name: "non chart tool", toolName: "knowledge_search", result: `{"content":"x"}`, wantEvents: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sink := &captureSink{}
			emitChartFromToolResult(context.Background(), sink, tc.toolName, tc.result)
			if len(sink.events) != tc.wantEvents {
				t.Fatalf("events = %d, want %d", len(sink.events), tc.wantEvents)
			}
			if tc.wantEvents == 0 {
				return
			}
			data, ok := sink.events[0].Data.(map[string]any)
			if !ok {
				t.Fatalf("event data type = %T, want map[string]any", sink.events[0].Data)
			}
			if data["type"] != tc.wantType {
				t.Fatalf("component type = %v, want %q", data["type"], tc.wantType)
			}
		})
	}
}
