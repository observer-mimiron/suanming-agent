package runtime

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// agentEventBridge consumes the ADK AsyncIterator and bridges events to EventSink.
// bufferFinal=true 时缓存最终回答，供上层做 post-run contract gate 校验；
// 否则按 chunk 直接推送 text，恢复普通主链的流式体验。
// resultSaver 在工具返回结果时调用，用于将结构化命盘数据写回会话状态。
func agentEventBridge(ctx context.Context, sink EventSink, iter *adk.AsyncIterator[*adk.AgentEvent], resultSaver func(toolName, resultJSON string), bufferFinal bool) (string, error) {
	var pendingText string // 当前累积的 Assistant 文本，待分类
	var finalText string   // 用户可见的最终文本
	var searchAttempts int // knowledge_search 调用计数，本轮内递增
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			if bufferFinal {
				return pendingText, event.Err
			}
			return finalText, event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		role := mv.Role
		toolName := mv.ToolName
		if !mv.IsStreaming && mv.Message != nil {
			if role == "" {
				role = mv.Message.Role
			}
			if toolName == "" {
				toolName = mv.Message.ToolName
			}
		}

		switch {
		case role == schema.Assistant && !mv.IsStreaming:
			msg, err := mv.GetMessage()
			if err != nil || msg == nil {
				continue
			}
			if isAssistantPlanningMessage(msg) {
				if msg.Content != "" {
					_ = emitEventWithTrace(ctx, sink, Event{Type: "thinking", Data: map[string]any{
						"text":  msg.Content,
						"agent": "supervisor",
					}}, map[string]any{
						"reason":       "assistant_tool_call",
						"buffer_final": bufferFinal,
					})
				}
				continue
			}
			if msg.Content == "" {
				continue
			}
			if bufferFinal {
				pendingText += msg.Content
			} else {
				finalText += msg.Content
				_ = emitEventWithTrace(ctx, sink, Event{Type: "text", Data: map[string]any{"content": msg.Content}}, map[string]any{
					"buffer_final": false,
				})
			}

		case role == schema.Assistant && mv.IsStreaming:
			for {
				chunk, err := mv.MessageStream.Recv()
				if err != nil {
					break
				}
				if chunk != nil && chunk.Content != "" {
					if bufferFinal {
						pendingText += chunk.Content
					} else {
						finalText += chunk.Content
						_ = emitEventWithTrace(ctx, sink, Event{Type: "text", Data: map[string]any{"content": chunk.Content}}, map[string]any{
							"buffer_final": false,
						})
					}
				}
			}

		case role == schema.Tool:
			msg, err := mv.GetMessage()
			if err != nil || msg == nil {
				continue
			}

			// 记录 knowledge_search 调用次数（仅记 trace，不发 SSE）。
			if toolName == "knowledge_search" {
				searchAttempts++
			}

			// 只有缓冲模式才把工具前的 assistant 文本归类为 thinking（内部独白）。
			if bufferFinal && pendingText != "" {
				_ = emitEventWithTrace(ctx, sink, Event{Type: "thinking", Data: map[string]any{
					"text":  pendingText,
					"agent": "supervisor",
				}}, map[string]any{
					"buffer_final": true,
				})
				pendingText = ""
			}

			_ = emitEventWithTrace(ctx, sink, Event{Type: "tool_call", Data: map[string]any{
				"tool":   toolName,
				"result": msg.Content,
			}}, map[string]any{
				"tool_name": toolName,
			})

			// 保存工具结果到会话状态（供后续轮次复用）
			if resultSaver != nil {
				resultSaver(toolName, msg.Content)
			}

			emitChartFromToolResult(ctx, sink, toolName, msg.Content)
		}
	}

	if bufferFinal {
		// 循环结束后：pendingText 是最后一轮 Assistant 文本（最终回答）。
		// 不在此处直接发 SSE，交由上层做 post-run guardrail 校验后再决定是否输出。
		return pendingText, nil
	}
	return finalText, nil
}

func isAssistantPlanningMessage(msg *schema.Message) bool {
	return msg != nil && len(msg.ToolCalls) > 0
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
	case "ziwei_liunian":
		chartType = "ziwei-chart"
	default:
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
