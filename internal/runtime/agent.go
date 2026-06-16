package runtime

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/wikiglobal/suanming-agent/internal/state"
)

// NewRuntimeAgent creates the ADK ChatModelAgent for fortune telling.
//
// tools: list of Eino BaseTool adapters (from buildAdapters).
// systemPrompt: system instruction from Builder.BuildAgentInstruction().
// model: Eino ToolCallingChatModel (shared with supervisor).
func NewRuntimeAgent(ctx context.Context, model einomodel.ToolCallingChatModel, tools []einotool.BaseTool, systemPrompt string) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "fortune_teller",
		Description:   "命理大师，八字/奇门/紫微斗数全领域分析助手。可调用排盘、用神、大运、知识检索等工具。",
		Instruction:   systemPrompt,
		Model:         model,
		MaxIterations: 12,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries: 2,
			ShouldRetry: func(ctx context.Context, rc *adk.RetryContext) *adk.RetryDecision {
				if rc.Err != nil {
					return &adk.RetryDecision{Retry: true, Backoff: time.Second}
				}
				return &adk.RetryDecision{Retry: false}
			},
		},
	})
}

// sessionStateSync syncs tool execution results from ADK SessionValues back to SessionState.
// Currently a stub — tool results are returned inline via the agent's response messages.
func sessionStateSync(ctx context.Context, st *state.SessionState) {
	// Tool results are already stored in SessionState by the executor after runAgent
}
