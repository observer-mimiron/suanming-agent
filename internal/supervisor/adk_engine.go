// Package supervisor 暂与 client.go 共享包注释，本文件提供基于 Eino ADK 的路由引擎实现。

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
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

type adkRouteEngine struct {
	model      einomodel.ToolCallingChatModel
	outputTool einotool.InvokableTool
}

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
