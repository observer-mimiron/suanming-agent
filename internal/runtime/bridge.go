package runtime

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// agentEventBridge consumes the ADK AsyncIterator, bridges events to EventSink.
// Returns the final assistant output text.
// resultSaver 在工具返回结果时调用，用于将结构化命盘数据写回会话状态。
func agentEventBridge(ctx context.Context, sink EventSink, iter *adk.AsyncIterator[*adk.AgentEvent], resultSaver func(toolName, resultJSON string)) (string, error) {
	var finalText string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return finalText, event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		role := mv.Role
		toolName := mv.ToolName

		switch {
		case role == schema.Assistant && !mv.IsStreaming:
			msg, err := mv.GetMessage()
			if err != nil || msg == nil || msg.Content == "" {
				continue
			}
			finalText = msg.Content
			sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": msg.Content}})

		case role == schema.Assistant && mv.IsStreaming:
			for {
				chunk, err := mv.MessageStream.Recv()
				if err != nil {
					break
				}
				if chunk != nil && chunk.Content != "" {
					finalText += chunk.Content
					sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": chunk.Content}})
				}
			}

		case role == schema.Tool:
			msg, err := mv.GetMessage()
			if err != nil || msg == nil {
				continue
			}
			sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{
				"tool":   toolName,
				"result": msg.Content,
			}})

			// 保存工具结果到会话状态（供后续轮次复用）
			if resultSaver != nil {
				resultSaver(toolName, msg.Content)
			}

			emitChartFromToolResult(ctx, sink, toolName, msg.Content)
		}
	}
	return finalText, nil
}

// emitChartFromToolResult detects chart payload in tool results and emits component events.
func emitChartFromToolResult(ctx context.Context, sink EventSink, toolName, resultJSON string) {
	var chartType string
	switch toolName {
	case "bazi_calc":
		chartType = "bazi-chart"
	case "qimen_dunjia":
		chartType = "qimen-chart"
	case "ziwei_calc":
		chartType = "ziwei-chart"
	default:
		return
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(resultJSON), &payload); err != nil || payload == nil {
		return
	}
	sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
		"type": chartType, "payload": payload,
	}})
}
