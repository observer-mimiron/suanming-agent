// Package runtime 包含 Manager 拥有的八字模型适配。
//
// 本文件负责分析规划、提示构建和内层 agent 的 JSON/文本适配；
// 不负责 Graph 拓扑、合同判定、事实计算或最终答复渲染。
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// runBaziAnalysisPlanner 让内层模型选择本轮八字分析所需的阶段和输出模板。
func (e *Executor) runBaziAnalysisPlanner(ctx context.Context, st *state.SessionState, question string, chartFacts baziCharterInput) (baziAnalysisPlan, error) {
	payload := buildAnalysisPlannerPayload(question, chartFacts)
	return runBaziInnerAgentJSON[baziAnalysisPlan](ctx, e.builder, baziAnalysisPlannerConfig(), st, buildBaziCharterPrompt("分析模式判定", question, payload))
}

// defaultBaziAnalysisPlan 在规划模型不可用时提供保守的完整分析计划。
func defaultBaziAnalysisPlan(question string) baziAnalysisPlan {
	return baziAnalysisPlan{
		Mode:              "static_full",
		RetrievalStage:    "static",
		NeedDynamic:       true,
		NeedLifetimeDayun: true,
		FocusTopics:       []string{"命局主轴", "命格层次", "大运验证", "流年应期"},
		WriterTemplate:    "full",
		TopicMode:         "analysis",
		StageSummary:      "已判定本轮以命局主轴分析为主。",
	}
}

// normalizeBaziAnalysisPlan 将规划别名收敛为 Graph 可识别的固定枚举。
func normalizeBaziAnalysisPlan(plan baziAnalysisPlan) baziAnalysisPlan {
	plan.Mode = strings.TrimSpace(plan.Mode)
	plan.RetrievalStage = strings.TrimSpace(plan.RetrievalStage)
	plan.WriterTemplate = strings.TrimSpace(plan.WriterTemplate)
	plan.TopicMode = normalizeByAlias(plan.TopicMode, map[string]string{
		"":                    "",
		"analysis":            "analysis",
		"general_analysis":    "analysis",
		"普通分析":                "analysis",
		"explain_term":        "explain_term",
		"term_explain":        "explain_term",
		"解释术语":                "explain_term",
		"conservative_reason": "conservative_reason",
		"保守原因":                "conservative_reason",
		"timing_reason":       "timing_reason",
		"岁运原因":                "timing_reason",
	})
	if plan.WriterTemplate == "topic" && plan.TopicMode == "" {
		plan.TopicMode = "analysis"
	}
	if plan.WriterTemplate != "full" {
		plan.NeedLifetimeDayun = false
	}
	return plan
}

// buildBaziCharterPrompt 把阶段、问题和确定性输入编码为内层模型提示。
func buildBaziCharterPrompt(stage, question string, payload any) string {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		body = []byte("{}")
	}
	return strings.TrimSpace(fmt.Sprintf(
		"当前阶段：%s\n用户问题：%s\n\n请依据本阶段职责完成分析并输出。\n\n输入数据：\n%s",
		stage,
		question,
		string(body),
	))
}

// runBaziInnerAgentText 运行一次非流式内层 agent，并返回最后一条文本消息。
func runBaziInnerAgentText(ctx context.Context, builder *AgentBuilder, cfg specialists.Config, st *state.SessionState, userPrompt string) (string, error) {
	agent, err := builder.BuildEphemeralInnerAgent(ctx, cfg, st)
	if err != nil {
		tracing.SetTraceAttributes(ctx, map[string]any{
			"bazi.inner_agent.name":  cfg.Name,
			"bazi.inner_agent.stage": "build",
			"bazi.inner_agent.error": err.Error(),
		})
		return "", err
	}
	if agent == nil {
		err := fmt.Errorf("inner agent %s is not configured", cfg.Name)
		tracing.SetTraceAttributes(ctx, map[string]any{
			"bazi.inner_agent.name":  cfg.Name,
			"bazi.inner_agent.stage": "build",
			"bazi.inner_agent.error": err.Error(),
		})
		return "", err
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: false,
	})
	raw, err := collectInnerAgentMessage(runner.Query(ctx, userPrompt))
	if err != nil {
		wrapped := fmt.Errorf("run inner agent %s: %w", cfg.Name, err)
		tracing.SetTraceAttributes(ctx, map[string]any{
			"bazi.inner_agent.name":  cfg.Name,
			"bazi.inner_agent.stage": "run",
			"bazi.inner_agent.error": wrapped.Error(),
		})
		return "", wrapped
	}
	return strings.TrimSpace(raw), nil
}

// runBaziInnerAgentJSON 以 JSON Mode 运行内层 agent，并按绑定的 Schema 严格解析。
// JSON Mode 只保证 JSON 外形；Schema、未知字段、尾随数据与 DTO 合同均由 runtime 强制。
func runBaziInnerAgentJSON[T any](ctx context.Context, builder *AgentBuilder, cfg specialists.Config, st *state.SessionState, userPrompt string) (T, error) {
	var out T
	raw, err := runBaziInnerAgentText(ctx, builder, cfg, st, userPrompt)
	if err != nil {
		return out, err
	}
	if err := decodeStructuredOutput(cfg.StructuredSchema, raw, &out); err != nil {
		tracing.SetTraceAttributes(ctx, map[string]any{
			"bazi.inner_agent.name":           cfg.Name,
			"bazi.inner_agent.stage":          "parse_json",
			"bazi.inner_agent.error":          err.Error(),
			"bazi.inner_agent.output_preview": truncateTracePreview(raw, 1200),
		})
		return out, fmt.Errorf("parse inner agent %s output: %w", cfg.Name, err)
	}
	return out, nil
}

// collectInnerAgentMessage 从异步事件中提取最后一条有效模型消息。
func collectInnerAgentMessage(iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var last string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil || event.Output.MessageOutput.Message == nil {
			continue
		}
		last = event.Output.MessageOutput.Message.Content
	}
	if strings.TrimSpace(last) == "" {
		return "", fmt.Errorf("inner agent produced empty output")
	}
	return last, nil
}
