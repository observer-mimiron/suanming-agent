// Package supervisor 暂与 client.go 共享包注释。
//
// 本文件提供基于 Eino ADK (Agent Development Kit) 的路由引擎实现。
//
// 当前项目固定采用这套 ADK 路由引擎承载 layer-1 结构化决策，
// Go 侧继续保留 textDecide / fallbackExtract / safeFallback 作为外层业务降级。

package supervisor

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// adkRouteEngine 通过 Eino ADK ChatModelAgent 执行结构化路由决策。
//
// 模型调用被包装在 ADK Agent 中运行，利用框架的 Runner 管理事件流、
// ModelRetryConfig 处理模型级重试、ReturnDirectly 确保工具调用后不进入多余的推理循环。
type adkRouteEngine struct {
	model      einomodel.ToolCallingChatModel
	outputTool einotool.InvokableTool
}

// NewADKRouteEngine 创建一个基于 Eino ADK 的路由引擎。
//
// 引擎内部构建一个名为 "output" 的 InvokableTool，其输入类型为 decisionOutput——
// 与 SupervisorDecision 的 JSON schema 完全对应。该工具既是模型输出的约束框架
// （模型必须产出符合该 schema 的 JSON），也是验证关卡（工具执行时会调用 parseAndValidate）。
//
// 返回的 RouteEngine 可注入到 Client 中，作为固定的 layer-1 structured route engine。
func NewADKRouteEngine(ctx context.Context, model einomodel.ToolCallingChatModel) (RouteEngine, error) {
	outputTool, err := utils.InferTool(decisionToolName,
		decisionToolDescription,
		func(ctx context.Context, input decisionOutput) (string, error) {
			raw, err := sonic.MarshalString(input)
			if err != nil {
				return "", err
			}
			if _, err := parseAndValidate(raw); err != nil {
				return "", errors.New(decisionRetryPrompt(err))
			}
			return raw, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("build adk output tool: %w", err)
	}

	return &adkRouteEngine{
		model:      model,
		outputTool: outputTool,
	}, nil
}

// Decide 执行一次带自我修正的结构化路由决策。
//
// 最多 2 次尝试（首次 + 1 次修正重试）。每次尝试调用 runOnce 通过 ADK Agent 触发模型，
// 模型的 tool_use 输出经 parseAndValidate 校验。如果校验失败，从错误信息中提取纠错反馈
// 注入到下一条用户消息中，引导模型自我修正。
//
// 与 client.go 中的两层重试（structured → text）不同，本方法只有一层结构化重试——
// 因为 ADK 框架本身通过 ModelRetryConfig 已经处理了网络层面的重试，
// 这里的重试仅针对业务校验失败。
func (e *adkRouteEngine) Decide(ctx context.Context, prompt, msg string) (schemas.SupervisorDecision, error) {
	attemptMsg := msg
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := e.runOnce(ctx, prompt, attemptMsg)
		if err == nil {
			decision, parseErr := parseAndValidate(raw)
			if parseErr != nil {
				return schemas.SupervisorDecision{}, fmt.Errorf("adk parse decision: %w", parseErr)
			}
			return decision, nil
		}

		lastErr = err
		feedback, ok := decisionRetryFeedbackFromError(err)
		if !ok || attempt == 1 {
			break
		}
		attemptMsg = decisionRetryMessage(msg, feedback)
	}

	return schemas.SupervisorDecision{}, lastErr
}

// runOnce 执行单次 ADK Agent 调用并返回模型的工具输出原始 JSON 字符串。
//
// 内部流程：
//  1. 创建 ChatModelAgent，配置系统指令(prompt)、output 工具、ReturnDirectly 策略。
//     ReturnDirectly 确保模型调用 output 工具后立即终止，不会进入多余的推理循环。
//  2. 通过 Runner.Query 发起请求，Runner 负责事件循环管理。
//  3. 从 AgentEvent 流中收集最终消息体——这应该是 output 工具的返回内容，即 decisionOutput JSON。
//
// MaxIterations=4 是防御性上限：正常情况下 1 次迭代（模型直接调用 output 工具）即可完成，
// 但如果模型先输出思考文本再调用工具，最多给 4 轮。
func (e *adkRouteEngine) runOnce(ctx context.Context, prompt, msg string) (string, error) {
	ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
		Name: "supervisor_model",
		Kind: tracing.KindLLM,
	})
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "supervisor_router",
		Description:   "用于 SupervisorDecision 提取的结构化路由代理。",
		Instruction:   prompt,
		Model:         e.model,
		MaxIterations: 4,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []einotool.BaseTool{e.outputTool},
			},
			ReturnDirectly: map[string]bool{decisionToolName: true},
		},
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries: 2,
			ShouldRetry: func(ctx context.Context, retryCtx *adk.RetryContext) *adk.RetryDecision {
				if retryCtx.Err != nil {
					return &adk.RetryDecision{Retry: true}
				}
				return &adk.RetryDecision{Retry: false}
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("new chat model agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: false,
	})

	iter := runner.Query(ctx, msg)
	raw, err := collectFinalMessage(iter)
	if err != nil {
		return "", err
	}

	return raw, nil
}

// collectFinalMessage 从 ADK Agent 事件流中提取最终的文本输出。
//
// 遍历 AsyncIterator，跳过无消息体的事件（如思考步骤、中间状态更新），
// 持续更新 last 直到流结束——最后一条有效消息即为 output 工具的返回值。
// 流结束后若未收集到任何内容则报错，防止空结果穿透到上层解析逻辑。
func collectFinalMessage(iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
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

	if last == "" {
		return "", fmt.Errorf("adk route engine produced empty output")
	}

	return last, nil
}
