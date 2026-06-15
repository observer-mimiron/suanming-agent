// Package supervisor 提供三层防御的路由决策系统，将 LLM 输出转换为结构化 SupervisorDecision。

package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// RouteEngine 是 ADK 路由引擎接口，提供基于 LLM 的结构化决策。
type RouteEngine interface {
	Decide(ctx context.Context, prompt, msg string) (schemas.SupervisorDecision, error)
}

// ClientOption 是 Client 构造函数的选项函数。
type ClientOption func(*Client)

func WithRouteEngine(engine RouteEngine) ClientOption {
	return func(c *Client) {
		c.routeEngine = engine
	}
}

var loadSupervisorPrompt = buildSupervisorPrompt

// Client 调用 LLM supervisor 并返回结构化路由决策。
type Client struct {
	flash       llm.Chat
	routeEngine RouteEngine
}

// NewClient 创建一个由 flash 模型驱动的 supervisor 客户端，可传入选项配置路由引擎。
func NewClient(flash llm.Chat, opts ...ClientOption) *Client {
	client := &Client{flash: flash}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// Decide 通过三层防御机制运行 supervisor 决策：
//
//	第 1 层 — structuredDecide（约束解码）：
//	  使用强制 tool_choice 的 GenerateWithTool。模型必须调用 "output" 工具，
//	  其 input_schema 为 SupervisorDecision 模式。这保证 JSON 结构精确匹配——是数学保证，而非概率。
//	  验证失败时有最多 2 次重试，附带错误反馈。
//
//	第 2 层 — textDecide（提示词 + 验证 + 重试）：
//	  如果第 1 层失败（API 错误或持续验证失败），回退到纯文本生成。
//	  提示模型输出 JSON，解析并验证响应，将具体的错误反馈注入重试提示词中。
//	  最多 3 次尝试。
//
//	第 3 层 — fallbackExtract / safeFallback：
//	  如果所有第 2 层尝试均失败，回退到简化的提取提示词，不进行复杂路由。
//	  如果模型完全不可用，通过 safeFallback 返回硬编码的保守默认值。
func (c *Client) Decide(ctx context.Context, msg string, st *state.SessionState) (schemas.SupervisorDecision, error) {
	prompt, err := loadSupervisorPrompt()
	if err != nil {
		return safeFallback(st), fmt.Errorf("supervisor prompt: %w", err)
	}

	// 注入会话上下文，使 supervisor 了解已有的数据。
	sessionCtx := buildSessionContext(st)
	if sessionCtx != "" {
		prompt += "\n\n## 当前会话状态\n\n" + sessionCtx
	}

	messages := []llm.Message{
		{Role: "user", Content: msg},
	}

	// 第 1 层：尝试通过强制工具使用的约束解码。
	// 这保证响应 JSON 符合模式定义。
	decision, err := c.structuredDecide(ctx, prompt, messages)
	if err == nil {
		return decision, nil
	}
	log.Printf("[supervisor] layer-1 structuredDecide failed: %v, falling back to text-based routing", err)

	// 第 2+3 层：回退到基于文本的生成，带验证和重试。
	decision, err = c.textDecide(ctx, prompt, messages, st, msg)
	if err != nil {
		log.Printf("[supervisor] layer-2/3 textDecide also failed: %v, using degraded fallback", err)
	}
	return decision, err
}

// structuredDecide（第 1 层）：通过强制 tool_choice 进行约束解码。
// 模型必须调用 "output" 工具，其 input_schema 为 SupervisorDecision 模式——这保证结构上有效的 JSON。
// 最多 2 次尝试，失败时附带领域级验证的错误反馈。
func (c *Client) structuredDecide(ctx context.Context, prompt string, messages []llm.Message) (schemas.SupervisorDecision, error) {
	if c.routeEngine != nil {
		if len(messages) == 0 {
			return schemas.SupervisorDecision{}, fmt.Errorf("structured: missing user message")
		}
		return c.routeEngine.Decide(ctx, prompt, messages[len(messages)-1].Content)
	}

	tool := buildDecisionTool()
	if llm.IsEinoChat(c.flash) {
		ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
			Name: "supervisor_model",
			Kind: tracing.KindLLM,
		})
	}

	for attempt := 0; attempt < 2; attempt++ {
		input, _, err := c.flash.GenerateWithTool(ctx, prompt, messages, tool)
		if err != nil {
			return schemas.SupervisorDecision{}, fmt.Errorf("structured: %w", err)
		}

		// 将工具输入转换回 JSON 字符串用于 parseAndValidate
		raw, _ := json.Marshal(input)
		decision, validationErr := parseAndValidate(string(raw))
		if validationErr == nil {
			return decision, nil
		}

		// 验证失败——将错误反馈回模型进行自我修正。
		messages = append(messages,
			llm.Message{Role: "assistant", Content: string(raw)},
			llm.Message{Role: "user", Content: decisionRetryPrompt(validationErr)},
		)
	}

	return schemas.SupervisorDecision{}, fmt.Errorf("structured: validation failed after retry")
}

// textDecide（第 2 层）：纯文本生成，带 JSON 解析、领域验证和最多 3 次重试。
// 每次验证失败时，将具体错误注入下一次提示词，使模型能够自我修正。
// 所有重试耗尽时回退到第 3 层（fallbackExtract）。
func (c *Client) textDecide(ctx context.Context, prompt string, messages []llm.Message, st *state.SessionState, msg string) (schemas.SupervisorDecision, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		callCtx := ctx
		if llm.IsEinoChat(c.flash) {
			callCtx = tracing.WithEinoCallbackSpan(callCtx, tracing.EinoCallbackSpanConfig{
				Name: "supervisor_model",
				Kind: tracing.KindLLM,
			})
		}
		resp, _, err := c.flash.Generate(callCtx, prompt, messages)
		if err != nil {
			lastErr = fmt.Errorf("supervisor call: %w", err)
			continue
		}

		decision, validationErr := parseAndValidate(resp)
		if validationErr == nil {
			return decision, nil
		}

		lastErr = validationErr
		// 附加错误反馈，使模型在下次尝试时能够自我修正。
		messages = append(messages,
			llm.Message{Role: "assistant", Content: resp},
			llm.Message{Role: "user", Content: decisionRetryPrompt(validationErr)},
		)
	}

	// 所有重试耗尽——回退到聚焦提取。
	return c.fallbackExtract(ctx, msg, st), lastErr
}

// parseDecision 将原始 JSON 解析为规范化的 SupervisorDecision。
func parseDecision(raw string) schemas.SupervisorDecision {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		d := schemas.SupervisorDecision{}
		d.Normalize()
		return d
	}

	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var d schemas.SupervisorDecision
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		d = schemas.SupervisorDecision{}
	}
	d.Normalize()
	return d
}

// parseAndValidate 解析 LLM 响应并运行领域特定的验证。
func parseAndValidate(raw string) (schemas.SupervisorDecision, error) {
	d := parseDecision(raw)

	if d.TaskIntent == "collect_profile" && len(d.Slots.Profile) == 0 {
		return d, fmt.Errorf("task_intent 为 collect_profile，但 slots.profile 为空。必须从用户原始消息中提取出生信息")
	}

	// 检测编造的资料：当 LLM 为缺失字段填入默认值时。
	// 如果资料有可疑的默认值（month=1, day=1, hour=0）且只有 1-2 个真实字段，
	// 很可能是模型编造了其余部分。拒绝并重试。
	if d.TaskIntent == "collect_profile" && len(d.Slots.Profile) >= 4 {
		if looksHallucinated(d.Slots.Profile) {
			return d, fmt.Errorf("slots.profile 疑似编造了默认值。只提取消息中明确出现的字段，缺字段就缺着不要补，让系统自动追问")
		}
	}

	if (d.TaskIntent == "timing_followup" || d.TaskIntent == "cross_domain_consult") && d.Slots.QuestionText == "" {
		return d, fmt.Errorf("task_intent 为 %s，但 slots.question_text 为空。必须提取用户的核心问题", d.TaskIntent)
	}

	return d, nil
}

// looksHallucinated 检查资料是否可能包含编造的默认值。
// 信号：month=1、day=1 和 hour=0 同时出现——LLM 为缺失出生信息设置的经典默认值。
func looksHallucinated(profile map[string]any) bool {
	month, hasMonth := profile["month"].(float64)
	day, hasDay := profile["day"].(float64)
	hour, hasHour := profile["hour"].(float64)
	if !hasMonth || !hasDay || !hasHour {
		return false
	}
	return month == 1 && day == 1 && hour == 0
}

// fallbackExtract（第 3a 层）：所有重试均失败时的最后手段 LLM 提取。
// 使用重点明确的单一用途提示词——不进行复杂路由，只提取已有内容。
// 即使此步骤也失败时返回硬编码默认值。
func (c *Client) fallbackExtract(ctx context.Context, msg string, st *state.SessionState) schemas.SupervisorDecision {
	fallbackPrompt := `从用户消息中提取信息，只返回一个 JSON 对象。

如果消息包含出生时间（年份+月份+日期），设置 task_intent="collect_profile"，并在 slots.profile 中提取：
- year: 数字 (1900-2100)
- month: 数字 (1-12)
- day: 数字 (1-31)
- hour: 数字 (0-23, 24小时制, 上午0-11, 下午12-23)
- gender: "男" 或 "女"
- birthplace: 城市名称（如提及）
- calendar_type: "solar" 或 "lunar"（默认 solar）

如果消息是纯问题（无出生信息），设置 task_intent="interpret_chart" 或 "fortune_followup"，slots.question_text 填问题内容。

其他必填字段使用默认值: conversation_intent="consult", primary_domain="bazi", confidence=0.5, policy_hints.needs_knowledge=true。

只返回 JSON，不要 markdown，不要额外说明。`

	messages := []llm.Message{{Role: "user", Content: msg}}
	resp, _, err := c.flash.Generate(ctx, fallbackPrompt, messages)
	if err != nil {
		return safeFallback(st)
	}

	d := parseDecision(resp)
	// 确保回退后关键字段有合理的默认值。
	if d.ConversationIntent == "" {
		d.ConversationIntent = "consult"
	}
	if d.PrimaryDomain == "" {
		d.PrimaryDomain = "bazi"
	}
	if d.TaskIntent == "" {
		d.TaskIntent = "collect_profile"
	}
	return d
}

// buildSessionContext 构建当前会话状态的简洁摘要，用于注入到 supervisor 提示词中。
func buildSessionContext(st *state.SessionState) string {
	hasProfile := len(st.Profile) > 0
	hasChart := st.HasBaziResult()
	isComplete := st.IsProfileComplete()

	if !hasProfile && !hasChart {
		return fmt.Sprintf("会话状态：新会话，尚无任何用户资料或命盘。\n当前日期：%s", time.Now().Format("2006-01-02"))
	}

	var parts []string
	if hasProfile {
		profileJSON, _ := json.Marshal(st.Profile)
		parts = append(parts, fmt.Sprintf("当前日期：%s\n已有资料：%s", time.Now().Format("2006-01-02"), string(profileJSON)))
		if isComplete {
			parts = append(parts, "资料完整度：完整")
		} else {
			parts = append(parts, "资料完整度：不完整（缺少必填字段）")
		}
	}
	if hasChart {
		parts = append(parts, "命盘状态：已有八字命盘（可复用）")
	}
	if st.LastUserQuestion != "" {
		parts = append(parts, fmt.Sprintf("上一轮问题：%s", st.LastUserQuestion))
	}

	return strings.Join(parts, "\n") + "\n\n根据以上会话状态：\n- 如果用户刚提供出生信息且资料刚完整、尚无命盘 → task_intent 应为 interpret_chart（首次完整解读）\n- 如果用户提供新的出生信息，且已有完整资料+命盘 → task_intent 应为 amend_profile\n- 如果用户追问且已有命盘 → task_intent 应为 fortune_followup，can_reuse_cached_result=true\n- 如果用户仅补充部分字段（如「我是女的」）且已有资料 → task_intent 应为 amend_profile，can_reuse_session_profile=true"
}

// safeFallback（第 3b 层）：LLM 不可用时的硬编码保守默认值。
// 不进行网络调用——纯基于当前会话状态的逻辑判断。
func safeFallback(st *state.SessionState) schemas.SupervisorDecision {
	needsClarification := !st.IsProfileComplete() && !st.HasBaziResult()
	taskIntent := "collect_profile"
	if st.IsProfileComplete() || st.HasBaziResult() {
		taskIntent = "interpret_chart"
		needsClarification = false
	}

	return schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		SecondaryDomains:   []string{},
		TaskIntent:         taskIntent,
		NeedsClarification: needsClarification,
		Confidence:         0.5,
		Slots: schemas.DecisionSlots{
			Profile:  map[string]any{},
			Language: "zh",
		},
		PolicyHints: schemas.PolicyHints{
			NeedsKnowledge:         true,
			CanReuseSessionProfile: st.IsProfileComplete() || st.HasBaziResult(),
			CanReuseCachedResult:   st.HasBaziResult(),
		},
	}
}

// buildSupervisorPrompt 加载统一的 supervisor 提示词。
func buildSupervisorPrompt() (string, error) {
	b, err := os.ReadFile("prompts/supervisor/unified_router.md")
	if err != nil {
		return "", fmt.Errorf("read unified_router.md: %w", err)
	}
	return string(b), nil
}

// buildDecisionTool 返回 SupervisorDecision 模式作为 Anthropic 工具定义。
// 这启用第 1 层约束解码：模型被强制调用此工具，保证输出 JSON 符合模式。
func buildDecisionTool() llm.ToolDef {
	return llm.ToolDef{
		Name:        decisionToolName,
		Description: decisionToolDescription,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"conversation_intent": map[string]any{
					"type":        "string",
					"enum":        []string{"chitchat", "consult"},
					"description": "L0 对话意图: chitchat=闲聊打招呼, consult=命理咨询",
				},
				"primary_domain": map[string]any{
					"type":        "string",
					"enum":        []string{"bazi", "qimen", "ziwei"},
					"description": "L1 主要命理领域",
				},
				"secondary_domains": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
						"enum": []string{"bazi", "qimen", "ziwei"},
					},
					"description": "L1 辅助领域列表",
				},
				"task_intent": map[string]any{
					"type": "string",
					"enum": []string{
						"collect_profile", "amend_profile", "interpret_chart",
						"fortune_followup", "timing_followup", "cross_domain_consult", "chitchat",
					},
					"description": "L2 任务意图: collect_profile=采集出生信息, amend_profile=补充修改资料, interpret_chart=首次排盘解读, fortune_followup=追问运势, timing_followup=择时择日, cross_domain_consult=跨领域咨询, chitchat=闲聊",
				},
				"needs_clarification": map[string]any{
					"type":        "boolean",
					"description": "是否需要反问用户以澄清信息",
				},
				"clarification_question": map[string]any{
					"type":        "string",
					"description": "反问问题文本，仅在 needs_clarification=true 时填写",
				},
				"parallelizable": map[string]any{
					"type":        "boolean",
					"description": "是否可并行调用多个领域同时处理",
				},
				"confidence": map[string]any{
					"type":        "number",
					"description": "决策置信度，0.0-1.0，信息充分时给高分",
				},
				"slots": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"profile": map[string]any{
							"type":        "object",
							"description": "提取的出生信息字典。可包含字段: year(数字,1900-2100), month(数字,1-12), day(数字,1-31), hour(数字,0-23,24小时制), gender(字符串,'男'/'女'), birthplace(字符串,城市名), calendar_type(字符串,'solar'/'lunar')。只填用户明确提到的字段，缺少的字段不要编造默认值，留给系统追问。",
						},
						"question_text": map[string]any{
							"type":        "string",
							"description": "用户核心问题原文，如'我的财运如何'",
						},
						"time_scope": map[string]any{
							"type":        "string",
							"description": "问题涉及的时间范围，如'今年'、'下个月'、'明年'",
						},
						"target_subject": map[string]any{
							"type":        "string",
							"description": "用户关注的主题，如'事业'、'财运'、'婚姻'、'健康'",
						},
						"language": map[string]any{
							"type":        "string",
							"description": "用户使用的语言代码，默认 zh",
						},
					},
					"required": []string{"profile", "question_text"},
				},
				"policy_hints": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"needs_knowledge": map[string]any{
							"type":        "boolean",
							"description": "是否需要检索知识库获取命理参考",
						},
						"needs_qimen": map[string]any{
							"type":        "boolean",
							"description": "是否需要奇门遁甲排盘",
						},
						"can_reuse_session_profile": map[string]any{
							"type":        "boolean",
							"description": "是否可复用会话中已有的用户资料",
						},
						"can_reuse_cached_result": map[string]any{
							"type":        "boolean",
							"description": "是否可复用已缓存的命盘计算结果",
						},
					},
					"required": []string{"needs_knowledge"},
				},
			},
			"required": []string{
				"conversation_intent", "primary_domain", "task_intent",
				"confidence", "slots", "policy_hints",
			},
		},
	}
}
