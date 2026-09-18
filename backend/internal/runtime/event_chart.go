// Package runtime 包含 Manager 所有的执行主链。
//
// 本文件负责把确定性工具返回的命盘 JSON 投影为前端 chart component 事件；
// 不负责工具调用、领域裁断或最终文本。
package runtime

import (
	"context"
	"encoding/json"
)

// emitChartFromToolResult detects chart payload in tool results and emits component events.
func emitChartFromToolResult(ctx context.Context, sink EventSink, toolName, resultJSON string) {
	chartType := chartComponentType(toolName)
	if chartType == "" {
		return
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(resultJSON), &payload); err != nil || payload == nil {
		return
	}
	_ = emitEventWithTrace(ctx, sink, Event{Type: "component", Data: map[string]any{
		"type": chartType, "payload": payload,
	}}, map[string]any{
		"component_type": chartType,
		"tool_name":      toolName,
	})
}

// chartComponentType maps a chart-producing tool to its stable frontend event type.
func chartComponentType(toolName string) string {
	switch toolName {
	case "bazi_calc":
		return "bazi-chart"
	case "qimen_dunjia":
		return "qimen-chart"
	case "ziwei_calc", "ziwei_liunian":
		return "ziwei-chart"
	default:
		return ""
	}
}
