// Package runtime 包含 Manager 所有的执行主链。
//
// 本文件负责 ADK 事件到 EventSink 的桥接和 chart component 事件；
// 不负责路由、Graph、领域裁断。
package runtime

import (
	"context"
	"encoding/json"
	"log"

	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// isSpecialistTool 判断工具名是否属于领域专家节点。
// 这类回复代表领域分析正文，应作为主文本输出，不走普通 tool_call 事件。
func isSpecialistTool(name string) bool {
	return strings.HasSuffix(name, "_specialist")
}

// agentEventBridge consumes the ADK AsyncIterator and bridges events to EventSink.
// bufferFinal=true 时缓存最终回答，供上层做 post-run contract gate 校验；
// 否则按 chunk 直接推送 text，恢复普通主链的流式体验。
// resultSaver 在工具返回结果时调用，用于将结构化命盘数据写回会话状态。
func agentEventBridge(ctx context.Context, sink EventSink, iter *adk.AsyncIterator[*adk.AgentEvent], resultSaver func(toolName, resultJSON string), labelFor func(toolName string) string, bufferFinal bool) (string, error) {
	var pendingText string    // 当前累积的 Assistant 文本，待分类
	var finalText string      // 用户可见的最终文本
	var searchAttempts int    // knowledge_search
	var thinkingBuf string    // 累积 thinking 文本以写入当前 LLM span 调用计数，本轮内递增
	var issuedToolName string // 最近一次上游发出的工具调用名，用于检测专家回复是否被内联
	var toolCallIssued bool
	var specialistRunning string              // specialist 运行中的工具名，独立于 toolCallIssued，不受 specialist 内部工具调用影响
	var specialistDone bool                   // 至少一个 specialist 已返回，supervisor 后续总结文本需丢弃
	processedSpecialists := map[string]bool{} // 已处理的 specialist 名称，防止同一 specialist 内容重复加入
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
			log.Printf("[bridge] event: role=%s tool=%s streaming=%v", role, toolName, mv.IsStreaming)
		}

		switch {
		case role == schema.Assistant && !mv.IsStreaming:
			msg, err := mv.GetMessage()
			if err != nil || msg == nil {
				continue
			}
			if isAssistantPlanningMessage(msg) {
				// 记录上游发出的工具调用，用于检测专家回复是否被内联
				if len(msg.ToolCalls) > 0 {
					toolCallIssued = true
					issuedToolName = msg.ToolCalls[0].Function.Name
					if isSpecialistTool(issuedToolName) {
						specialistRunning = issuedToolName
					}
					log.Printf("[bridge] supervisor issued tool: %s", issuedToolName)
				}
				if msg.Content != "" {
					thinkingBuf += msg.Content + "\n"
					tracing.AppendCurrentSpanAttribute(ctx, "thinking", thinkingBuf)
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
				// 专家回复可能被 Eino 内联到 Assistant 流中（非 planning message），
				// 此时需要补齐后续归类逻辑。
				if toolCallIssued {
					toolCallIssued = false
					log.Printf("[bridge] inlined specialist response (empty content) for %s", issuedToolName)
				}
				continue
			}
			// 非 planning 的 assistant 消息里，继续检测是否混入专家回复。
			if toolCallIssued && !isAssistantPlanningMessage(msg) {
				toolCallIssued = false
				if isSpecialistTool(issuedToolName) {
					// specialist 内联回复：按名称去重，加入 pendingText 作为主文本
					if !processedSpecialists[issuedToolName] {
						processedSpecialists[issuedToolName] = true
						specialistDone = true
						log.Printf("[bridge] specialist inline response for %s, adding to pendingText", issuedToolName)
						if bufferFinal {
							pendingText += msg.Content
						}
						emitChartFromToolResult(ctx, sink, issuedToolName, msg.Content)
						if resultSaver != nil {
							resultSaver(issuedToolName, msg.Content)
						}
					} else {
						log.Printf("[bridge] specialist %s inline response already processed, skipping", issuedToolName)
					}
					continue
				} else {
					log.Printf("[bridge] inlined specialist response for %s, emitting tool_call", issuedToolName)
					_ = emitEventWithTrace(ctx, sink, Event{Type: "tool_call", Data: map[string]any{
						"tool":   issuedToolName,
						"label":  labelFor(issuedToolName),
						"result": msg.Content,
					}}, map[string]any{
						"tool_name": issuedToolName,
					})
					emitChartFromToolResult(ctx, sink, issuedToolName, msg.Content)
					if resultSaver != nil {
						resultSaver(issuedToolName, msg.Content)
					}
				}
			}
			if specialistDone {
				// specialist 已输出最终内容，supervisor 后续文本作为 thinking 丢弃，不追加到 pendingText
				log.Printf("[bridge] dropping supervisor post-specialist text (%d chars), specialist content already final", len(msg.Content))
			} else if specialistRunning != "" {
				// specialist 正在运行，其内联 Assistant 文本将由 Tool message 路径统一处理，
				// 此处不加入 pendingText 以避免重复
				log.Printf("[bridge] dropping specialist running text (%d chars) for %s, waiting for tool result", len(msg.Content), specialistRunning)
			} else if bufferFinal {
				pendingText += msg.Content
			} else {
				finalText += msg.Content
				_ = emitEventWithTrace(ctx, sink, Event{Type: "text", Data: map[string]any{"content": msg.Content}}, map[string]any{
					"buffer_final": false,
				})
			}

		case role == schema.Assistant && mv.IsStreaming:
			// specialistRunning 非空时也视为 specialist 流式（内部工具调用可能已重置 toolCallIssued）。
			// specialist 内容由 Tool message 路径统一处理，流式 chunk 仅做去重标记。
			isSpecialistStream := (toolCallIssued && isSpecialistTool(issuedToolName)) || (specialistRunning != "" && !specialistDone)
			for {
				chunk, err := mv.MessageStream.Recv()
				if err != nil {
					break
				}
				if chunk != nil && chunk.Content != "" {
					if specialistDone || isSpecialistStream {
						// specialist 已输出或正在运行，流式文本不加入 pendingText（Tool message 会带正式内容）
						continue
					}
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
			// specialist 流式结束后标记 specialistDone，防止 Tool message 和 supervisor 文本重复加入
			if isSpecialistStream {
				name := issuedToolName
				if specialistRunning != "" {
					name = specialistRunning
				}
				processedSpecialists[name] = true
				specialistDone = true
				toolCallIssued = false
				log.Printf("[bridge] specialist %s streamed response captured, marking done", name)
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
			toolCallIssued = false // 正常的 Tool 事件重置标记

			// specialist 的回复是最终分析内容，按名称去重后加入 pendingText，不走 tool_call。
			// 多个 specialist 并行时各自的内容都保留，supervisor 后续总结由 specialistDone 拦截。
			if isSpecialistTool(toolName) {
				if !processedSpecialists[toolName] {
					processedSpecialists[toolName] = true
					specialistDone = true
					specialistRunning = ""

					// 当 EnableStreaming=true 时，上游转发可能会先把 specialist 的
					// 最终回复作为 streaming assistant event 转发给父级 iterator。这些 chunk
					// 在 specialistRunning 未设置的情况下被当作普通 supervisor 文本加入了
					// pendingText。Tool message 是 specialist 回复的权威副本。
					// forwarded streaming content 本质是 specialist 的思考过程，作为 thinking
					// 事件发给前端展示，不进入最终文本。
					if bufferFinal && pendingText != "" {
						log.Printf("[bridge] specialist tool %s: emitting forwarded streaming as thinking (%d chars), pendingText %d -> %d", toolName, len(pendingText), len(pendingText), len(msg.Content))
						_ = emitEventWithTrace(ctx, sink, Event{Type: "thinking", Data: map[string]any{
							"text":  pendingText,
							"agent": toolName,
						}}, map[string]any{"source": "forwarded_streaming"})
						pendingText = msg.Content
					} else if bufferFinal {
						pendingText = msg.Content
					}
				} else {
					specialistRunning = ""
					log.Printf("[bridge] specialist tool %s response already processed, skipping", toolName)
				}
				if resultSaver != nil {
					resultSaver(toolName, msg.Content)
				}
				emitChartFromToolResult(ctx, sink, toolName, msg.Content)
				continue
			}

			// 只有缓冲模式才把工具前的 assistant 文本归类为 thinking（内部独白）。
			if bufferFinal && pendingText != "" {
				tracing.AppendCurrentSpanAttribute(ctx, "thinking", pendingText)
				_ = emitEventWithTrace(ctx, sink, Event{Type: "thinking", Data: map[string]any{
					"text":  pendingText,
					"agent": "supervisor",
				}}, map[string]any{
					"buffer_final": true,
				})
				pendingText = ""
			}

			tracing.AppendCurrentSpanAttribute(ctx, "thinking", thinkingBuf)
			thinkingBuf = ""
			_ = emitEventWithTrace(ctx, sink, Event{Type: "tool_call", Data: map[string]any{
				"tool":   toolName,
				"label":  labelFor(toolName),
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
		analysisText, responseText, hasTags := parseXMLSections(pendingText)
		if hasTags && analysisText != "" {
			_ = emitEventWithTrace(ctx, sink, Event{Type: "thinking", Data: map[string]any{
				"text":  analysisText,
				"agent": "supervisor",
			}}, map[string]any{
				"buffer_final": true,
				"from_marker":  true,
			})
		}
		return sanitizeFinalOutput(responseText), nil
	}

	analysisText, responseText, hasTags := parseXMLSections(finalText)
	if hasTags {
		if analysisText != "" {
			_ = emitEventWithTrace(ctx, sink, Event{Type: "thinking", Data: map[string]any{
				"text":  analysisText,
				"agent": "supervisor",
			}}, map[string]any{
				"buffer_final": false,
				"from_marker":  true,
			})
		}
		finalText = responseText
	}
	return sanitizeFinalOutput(finalText), nil
}

// specialistEventBridge currently reuses the shared ADK event-to-SSE
// adaptation through a specialist-specific entry point so the manager-owned
// execution path can evolve independently.
func specialistEventBridge(ctx context.Context, sink EventSink, iter *adk.AsyncIterator[*adk.AgentEvent], resultSaver func(toolName, resultJSON string), labelFor func(toolName string) string, bufferFinal bool) (string, error) {
	return agentEventBridge(ctx, sink, iter, resultSaver, labelFor, bufferFinal)
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

// parseXMLSections 解析 LLM 输出中的 <analysis> 和 <response> XML 标记段。
//
// 返回 (analysisText, responseText, hasTags)。hasTags 为 false 时表示输入不含标记，
// 此时整个 input 视为 responseText（降级行为）。
func parseXMLSections(input string) (analysis, response string, hasTags bool) {
	analysisStart := strings.Index(input, "<analysis>")
	analysisEnd := strings.Index(input, "</analysis>")
	responseStart := strings.Index(input, "<response>")
	responseEnd := strings.Index(input, "</response>")

	if analysisStart == -1 || analysisEnd == -1 || responseStart == -1 || responseEnd == -1 {
		return "", strings.TrimSpace(input), false
	}

	analysis = strings.TrimSpace(input[analysisStart+len("<analysis>") : analysisEnd])
	response = strings.TrimSpace(input[responseStart+len("<response>") : responseEnd])
	return analysis, response, true
}
