package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tools"
)

// specialistsConfig 是 specialists.Config 的类型别名，用于 route-bound 选择。
type specialistsConfig = specialists.Config

// AgentBuilder 负责从 specialist 配置构建 ADK ChatModelAgent 和 AgentTool。
// 持有共享的 ToolCallingChatModel 和工具 Registry。
type AgentBuilder struct {
	model    einomodel.ToolCallingChatModel
	reg      *tools.Registry
	llmModel string
}

// NewAgentBuilder 创建 AgentBuilder。
func NewAgentBuilder(model einomodel.ToolCallingChatModel, reg *tools.Registry) *AgentBuilder {
	return &AgentBuilder{model: model, reg: reg}
}

// SetLLMModel 设置用于追踪 span 的模型名称。
func (b *AgentBuilder) SetLLMModel(model string) { b.llmModel = model }

// BuildSpecialist 从 Config 构建一个领域专家 ChatModelAgent。
// 如果会话状态中已有出生资料或命盘结果，会被注入到 instruction 中。
func (b *AgentBuilder) BuildSpecialist(ctx context.Context, cfg specialists.Config, st *state.SessionState) (adk.Agent, error) {
	adapters, err := BuildAdaptersFor(b.reg, cfg.ToolNames)
	if err != nil {
		return nil, err
	}
	instruction := cfg.Instruction
	if st != nil && (len(st.Profile) > 0 || st.HasBaziResult()) {
		instruction += "\n\n## 会话已有上下文\n\n以下资料已在当前会话中提供，**直接使用，无需再次索要或调用工具获取**：\n"
		if len(st.Profile) > 0 {
			pb := NewBuilder("") // 只用 buildProfileSection
			instruction += "\n### 出生资料\n" + pb.buildProfileSection(st) + "\n"
		}
		if st.HasBaziResult() {
			instruction += "\n### 命盘结果\n命盘已排好，直接从会话上下文引用即可，**禁止重新调用 bazi_calc**。\n"
		}
	}
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Instruction:   instruction,
		Model:         b.model,
		MaxIterations: 12,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: adapters,
			},
		},
		ModelRetryConfig: defaultRetryConfig(),
	})
}

// Allowed returns a copy of the selected configs.
func (b *AgentBuilder) Allowed(route policy.ApprovedRoute, all []specialists.Config) []specialists.Config {
	return allowedSpecialists(route, all)
}

// BuildSupervisor 根据本轮批准路由构建 Supervisor Agent，只挂载允许调用的 AgentTool。
func (b *AgentBuilder) BuildSupervisor(ctx context.Context, route policy.ApprovedRoute, st *state.SessionState, allowedSpecialists []specialists.Config) (adk.Agent, error) {
	var agentTools []einotool.BaseTool
	for _, cfg := range allowedSpecialists {
		child, err := b.BuildSpecialist(ctx, cfg, st)
		if err != nil {
			return nil, fmt.Errorf("build specialist %s: %w", cfg.Name, err)
		}
		agt := adk.NewAgentTool(ctx, child)
		agentTools = append(agentTools, agt)
	}

	instruction := b.buildSupervisorInstruction(route, allowedSpecialists)

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "supervisor",
		Description: "命理咨询执行主管，负责调度领域专家 Agent 完成分析。",
		Instruction: instruction,
		Model:       b.model,
		ToolsConfig: adk.ToolsConfig{
			EmitInternalEvents: true,
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: agentTools,
			},
		},
		MaxIterations: 10,
		ModelRetryConfig: defaultRetryConfig(),
	})
}

// buildSupervisorInstruction 构建每轮动态 supervisor instruction。
func (b *AgentBuilder) buildSupervisorInstruction(route policy.ApprovedRoute, allowed []specialists.Config) string {
	toolDesc := formatAllowedTools(allowed)
	return fmt.Sprintf(`你是命理咨询执行主管。

## 身份
你不是权威路由器——权威路由已经由系统决策层完成。本轮批准的主领域是 %s，你只能调用下方可见的领域专家。

## 可见的领域专家（本轮允许调用）
%s

## 调用规则
1. 如果只有一个专家可见 → 直接调用它
2. 如果多个专家可见 → 先调主领域专家，再根据用户是否明确问了辅领域决定是否调第二个
3. 如果用户问题涉及多个领域但只有一个专家可见 → 只调可见的，不要抱怨缺少工具

## 禁止
- 不要回答命理分析问题（这由领域专家负责），你只做执行调度
- 不要请求更多工具或抱怨缺少工具
- 如果运行时 preflight 已放行 qimen-primary 且无 profile，不要追问出生信息`,
		route.PrimaryDomain,
		toolDesc,
	)
}

// formatAllowedTools 格式化可见 AgentTool 列表为 instruction 文本。
func formatAllowedTools(cfgs []specialists.Config) string {
	if len(cfgs) == 0 {
		return "（无可见专家）"
	}
	var b strings.Builder
	for i, cfg := range cfgs {
		b.WriteString(fmt.Sprintf("%d. **%s** - %s", i+1, cfg.Name, cfg.Description))
		if i < len(cfgs)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// allowedSpecialists 根据 ApprovedRoute 过滤可用的 specialist 配置。
//
// 规则：
//   - 始终包含主域 specialist。
//   - qimen 仅在 QimenMode=primary 或 supplement 时包含。
//   - 其他辅域仅在 route.SecondaryDomains 明确包含时加入。
//   - fortune_followup + QimenMode=none 不包含 qimen。
//   - 未注册的域降级为 bazi。
func allowedSpecialists(route policy.ApprovedRoute, configs []specialists.Config) []specialists.Config {
	byDomain := make(map[string]specialists.Config)
	for _, cfg := range configs {
		byDomain[cfg.Domain] = cfg
	}

	// Determine primary domain, fallback to bazi
	primaryDomain := route.PrimaryDomain
	if _, ok := byDomain[primaryDomain]; !ok {
		primaryDomain = "bazi"
	}

	allowed := make(map[string]bool)
	allowed[primaryDomain] = true

	// Qimen visible only when QimenMode is primary or supplement
	if route.PolicyHints.QimenMode == "primary" || route.PolicyHints.QimenMode == "supplement" {
		allowed["qimen"] = true
	}

	// Other secondary domains explicit in the route
	for _, d := range route.SecondaryDomains {
		allowed[d] = true
	}

	var result []specialists.Config
	for _, cfg := range configs {
		if allowed[cfg.Domain] {
			result = append(result, cfg)
		}
	}
	// Always include primary domain even if not explicitly in allowed map
	found := false
	for _, cfg := range result {
		if cfg.Domain == primaryDomain {
			found = true
			break
		}
	}
	if !found {
		if cfg, ok := byDomain[primaryDomain]; ok {
			result = append(result, cfg)
		}
	}
	return result
}

// defaultRetryConfig 返回共享的 ModelRetryConfig，所有 Agent 统一使用。
func defaultRetryConfig() *adk.ModelRetryConfig {
	return &adk.ModelRetryConfig{
		MaxRetries: 2,
		ShouldRetry: func(ctx context.Context, rc *adk.RetryContext) *adk.RetryDecision {
			if rc.Err != nil {
				return &adk.RetryDecision{Retry: true, Backoff: time.Second}
			}
			return &adk.RetryDecision{Retry: false}
		},
	}
}

