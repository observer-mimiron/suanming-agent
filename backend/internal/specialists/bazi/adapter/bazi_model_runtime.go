// package adapter 包含 Manager 拥有的八字模型适配。
//
// 本文件负责确定性分析范围、提示构建和内层 agent 的 JSON/文本适配；
// 不负责 Graph 拓扑、合同判定、事实计算或最终答复渲染。
package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	baziapplication "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/application"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// deterministicBaziAnalysisPlan 将有限的写作模板选择收束到代码，避免
// 模型把“是否需要动态层”判断成不稳定的自由文本任务。
func deterministicBaziAnalysisPlan(question string) baziAnalysisPlan {
	q := strings.TrimSpace(question)
	plan := defaultBaziAnalysisPlan(q)
	if containsBaziKeyword(q, "今年", "本年", "流年", "大运", "岁运", "近期", "最近", "哪年", "何时") {
		plan.Mode = "dynamic_focus"
		plan.RetrievalStage = "dynamic"
		plan.NeedDynamic = true
		plan.NeedLifetimeDayun = false
		plan.FocusTopics = []string{"当前大运", "流年应期"}
		plan.WriterTemplate = "year"
		plan.TopicMode = "timing_reason"
		plan.StageSummary = "已按时间窗口进入岁运分析。"
		return plan
	}
	if containsBaziKeyword(q, "财运", "事业", "婚姻", "感情", "健康", "子女", "用神", "格局", "调候", "什么意思", "什么是", "怎么") &&
		!containsBaziKeyword(q, "分析八字", "完整", "全面", "整体") {
		plan.Mode = "topic_focus"
		plan.RetrievalStage = "static"
		plan.NeedDynamic = false
		plan.NeedLifetimeDayun = false
		plan.FocusTopics = []string{"命局主轴", "专题依据"}
		plan.WriterTemplate = "topic"
		plan.TopicMode = "analysis"
		plan.StageSummary = "已按专题问题进入命局分析。"
	}
	return plan
}

// containsBaziKeyword 只做模板路由，不参与命理裁断。
func containsBaziKeyword(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

// defaultBaziAnalysisPlan 在规划模型不可用时提供保守的完整分析计划。
func defaultBaziAnalysisPlan(question string) baziAnalysisPlan {
	return baziAnalysisPlan{
		Mode:              "static_full",
		RetrievalStage:    "static",
		NeedDynamic:       true,
		NeedLifetimeDayun: true,
		FocusTopics:       []string{"命局主轴", "格局评价", "大运验证", "流年应期"},
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
	plan.TopicMode = baziapplication.NormalizeByAlias(plan.TopicMode, map[string]string{
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
func runBaziInnerAgentText(ctx context.Context, builder AgentBuilder, cfg specialists.Config, view *specialists.SessionView, userPrompt string) (string, error) {
	// 内层阶段使用配置名建立独立 LLM span，避免所有八字模型调用都被归到外层路由，
	// 否则无法判断是证据、静态、动态还是 repair 在消耗时间和 token。
	ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
		Name: cfg.Name,
		Kind: tracing.KindLLM,
		Attributes: map[string]any{
			"bazi.inner_agent.name":   cfg.Name,
			"bazi.inner_agent.schema": cfg.StructuredSchema,
		},
	})
	agent, err := builder.BuildEphemeralInnerAgent(ctx, cfg, view)
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
func runBaziInnerAgentJSON[T any](ctx context.Context, builder AgentBuilder, cfg specialists.Config, view *specialists.SessionView, userPrompt string) (T, error) {
	var out T
	raw, err := runBaziInnerAgentText(ctx, builder, cfg, view, userPrompt)
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
