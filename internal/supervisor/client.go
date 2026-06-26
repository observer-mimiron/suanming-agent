// Package supervisor 提供三层防御的路由决策系统，将 LLM 输出转换为结构化 SupervisorDecision。
//
// # 架构概览
//
// 整个包围绕一个核心职责：将用户的自然语言消息转换为可执行的结构化路由决策。
// 这个转换过程通过三层防御机制实现：
//
//	第 1 层 — 结构化输出（structuredDecide / RouteEngine）：
//	  通过 tool calling 机制引导模型输出符合 SupervisorDecision JSON schema 的结构化数据。
//	  运行时将此 schema 绑定为强制工具调用，提供 schema 级约束。失败时附带校验错误反馈重试 1 次。
//
//	第 2 层 — 文本生成 + 校验重试（textDecide）：
//	  回退到纯文本生成，通过提示词引导模型输出 JSON，解析后在应用层校验。
//	  校验失败时注入具体错误反馈，最多重试 3 次。
//
//	第 3 层 — 安全降级（fallbackExtract / safeFallback）：
//	  放弃复杂路由，使用简化的单用途提取提示词。若模型完全不可用，
//	  通过 safeFallback 基于会话状态的纯逻辑判断返回保守默认值。
//
// # 调用入口
//
//   - Client.Approve：orchestrator 的主入口，决策 → 策略应用 → 规范化
//   - Client.Decide：仅执行 LLM 决策，用于需要原始 Decision 的场景
//   - Client.ExtractProfile：轻量级资料补抽，不参与主路由
//
// # 文件分工
//
//   - client.go：核心 Client、三层防御、决策解析与校验
//   - adk_engine.go：Eino ADK 引擎实现（RouteEngine 接口）
//   - approved_route.go：路由审批与规范化修正
//   - decision_contract.go：工具参数类型、重试消息契约

package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/wikiglobal/suanming-agent/internal/intent"
	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/prompts"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// RouteEngine 是结构化路由引擎接口，当前固定由 Eino ADK 实现。
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

// WithSemanticRouter 注入 semantic router，用于 applyExplicitMethodPreference。
// 传 nil 等于不启用（走 regex 兜底）。
func WithSemanticRouter(router intent.Router) ClientOption {
	return func(c *Client) {
		c.router = router
	}
}

// WithRouterMode 设置 semantic router 的运行模式：off | shadow | enforce。
func WithRouterMode(mode string) ClientOption {
	return func(c *Client) {
		c.routerMode = mode
	}
}

// loadSupervisorPrompt 在运行时指向 buildSupervisorPrompt，测试中可替换为返回固定提示词的函数。
var loadSupervisorPrompt = buildSupervisorPrompt

// Client 调用 LLM supervisor 并返回结构化路由决策。
type Client struct {
	flash       llm.Chat
	routeEngine RouteEngine
	router      intent.Router  // semantic router；nil 走 regex 兜底
	routerMode  string         // off | shadow | enforce
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
//	第 1 层 — structuredDecide（结构化输出）：
//	  通过 Eino ADK route engine 使用 tool calling 获取受约束的结构化结果。
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

// structuredDecide（第 1 层）：委托给 Eino ADK route engine 获取结构化输出。
func (c *Client) structuredDecide(ctx context.Context, prompt string, messages []llm.Message) (schemas.SupervisorDecision, error) {
	if c.routeEngine == nil {
		return schemas.SupervisorDecision{}, fmt.Errorf("structured: route engine not configured")
	}
	if len(messages) == 0 {
		return schemas.SupervisorDecision{}, fmt.Errorf("structured: missing user message")
	}
	return c.routeEngine.Decide(ctx, prompt, messages[len(messages)-1].Content)
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

// ExtractProfile 使用聚焦的简化提取链从消息中抽取出生资料和问题文本。
//
// 与 Decide 不同，本方法不参与主路由决策——它只做提取，固定返回 collect_profile 意图。
// 适用场景：normalizeApprovedRoute 检测到消息明显包含出生时间但 LLM 未在 slots.profile 中填充，
// 此时调用本方法做一次轻量级的补抽，避免让 orchestrator 带着空 profile 进入后续流程。
func (c *Client) ExtractProfile(ctx context.Context, msg string, st *state.SessionState) (map[string]any, string, error) {
	decision := c.fallbackExtract(ctx, msg, st)
	profile := decision.Slots.Profile
	if profile == nil {
		profile = map[string]any{}
	}
	question := strings.TrimSpace(decision.Slots.QuestionText)
	if question == "" {
		question = msg
	}
	return profile, question, nil
}

// parseDecision 将 LLM 原始输出解析为规范化的 SupervisorDecision。
//
// 处理三种常见格式：
//   - 纯 JSON：直接反序列化
//   - Markdown 代码块：剥离 ```json / ``` 包裹后反序列化
//   - 空字符串 / 非法 JSON：返回 normalize 后的零值（安全默认值，不 panic）
//
// 所有路径最终都调用 Normalize()，确保缺失字段被填充为业务安全的默认值。
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

	// collect_profile + 空 profile 是合法状态：
	// 用户表示想排盘但还没提供出生信息，系统应该追问。
	// 出生信息被遗漏的情况由 normalizeApprovedRoute 的 extractBirthSlots 路径兜底。

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
	if d.PolicyHints.QimenMode != "" && d.PolicyHints.QimenMode != "none" && d.PolicyHints.QimenMode != "supplement" && d.PolicyHints.QimenMode != "primary" {
		return d, fmt.Errorf("policy_hints.qimen_mode 非法：%s。必须为 none、supplement 或 primary", d.PolicyHints.QimenMode)
	}
	if d.PolicyHints.ProfileRequirement != "" && d.PolicyHints.ProfileRequirement != "none" && d.PolicyHints.ProfileRequirement != "full" {
		return d, fmt.Errorf("policy_hints.profile_requirement 非法：%s。必须为 none 或 full", d.PolicyHints.ProfileRequirement)
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
// fallbackExtractPrompt 返回 fallback 提取器的提示词。
// 它只负责资料/问题提取。
func fallbackExtractPrompt() string {
	return `从用户消息中提取信息，只返回一个 JSON 对象。

如果消息包含出生时间（年份+月份+日期），设置 task_intent="collect_profile"，并在 slots.profile 中只提取用户明确说出的字段：
- year: 数字 (1900-2100)
- month: 数字 (1-12)
- day: 数字 (1-31)
- hour: 数字 (0-23, 24小时制, 上午0-11, 下午12-23)
- gender: "男" 或 "女"
- birthplace: 城市名称（如提及）

如果消息没有出生信息，把用户问题放进 slots.question_text；不要猜缺失资料。

不要在这里扩展完整术数路由策略。其他缺省字段保持保守默认值即可。

只返回 JSON，不要 markdown，不要额外说明。`
}

func (c *Client) fallbackExtract(ctx context.Context, msg string, st *state.SessionState) schemas.SupervisorDecision {
	prompt := fallbackExtractPrompt()
	if sessionCtx := buildSessionContext(st); sessionCtx != "" {
		prompt += "\n\n## 当前会话状态\n\n" + sessionCtx
	}
	messages := []llm.Message{{Role: "user", Content: msg}}
	resp, _, err := c.flash.Generate(ctx, prompt, messages)
	if err != nil {
		return safeFallback(st)
	}

	d := parseDecision(resp)
	// 确保回退后关键字段有合理的默认值。
	if d.ConversationIntent == "" {
		d.ConversationIntent = "consult"
	}
	if d.PrimaryDomain == "" {
		if intent.ContainsTimingKeyword(msg) {
			d.PrimaryDomain = "qimen"
		} else {
			d.PrimaryDomain = "bazi"
		}
	}
	if d.TaskIntent == "" {
		d.TaskIntent = "collect_profile"
	}
	return d
}

// buildSessionContext 构建当前会话状态的摘要，注入到 supervisor 系统提示词中。
//
// 让 LLM 感知已有数据状态，从而做出更合理的路由决策。摘要包含：
//   - 已有资料（JSON 格式）及完整度评估
//   - 命盘状态（是否已有计算结果可复用）
//   - 上一轮用户问题（用于多轮对话的上下文连贯性）
//   - 当前日期（用于判断"今天"/"最近"等相对时间表述）
func buildSessionContext(st *state.SessionState) string {
	now := time.Now().Format("2006-01-02")
	if st == nil {
		return fmt.Sprintf("会话状态：未知。\n当前日期：%s", now)
	}

	hasProfile := len(st.Profile) > 0
	hasChart := st.HasBaziResult()
	isComplete := st.IsProfileComplete()
	hasGuidance := st.Guidance != nil

	if !hasProfile && !hasChart && !hasGuidance && st.LastUserQuestion == "" {
		return fmt.Sprintf("会话状态：新会话，尚无任何用户资料或命盘。\n当前日期：%s", now)
	}

	parts := []string{fmt.Sprintf("当前日期：%s", now)}
	if hasProfile {
		profileJSON, _ := json.Marshal(st.Profile)
		parts = append(parts, fmt.Sprintf("已有资料：%s", string(profileJSON)))
		parts = append(parts, fmt.Sprintf("可复用资料字段：%s", strings.Join(reusableProfileKeys(st.Profile), ", ")))
		if isComplete {
			parts = append(parts, "资料完整度：完整")
		} else {
			parts = append(parts, "资料完整度：不完整（缺少必填字段）")
		}
	} else {
		parts = append(parts, "已有资料：无")
	}
	if st.Subject != "" {
		parts = append(parts, fmt.Sprintf("当前命盘归属：%s", st.Subject))
	}
	parts = append(parts, fmt.Sprintf(
		"命盘状态：bazi=%t qimen=%t ziwei=%t",
		st.HasBaziResult(),
		st.HasQimenResult(),
		st.HasZiWeiResult(),
	))
	if hasGuidance {
		parts = append(parts, fmt.Sprintf("引导状态：active(kind=%s)", st.Guidance.DirectiveKind))
		if st.Guidance.ChosenTopic != "" {
			parts = append(parts, fmt.Sprintf("已选主题：%s", st.Guidance.ChosenTopic))
		}
		if st.Guidance.PendingSlot != "" {
			parts = append(parts, fmt.Sprintf("待补资料：%s", st.Guidance.PendingSlot))
		}
	} else {
		parts = append(parts, "引导状态：inactive")
	}
	if st.LastUserQuestion != "" {
		parts = append(parts, fmt.Sprintf("上一轮问题：%s", st.LastUserQuestion))
	}

	return strings.Join(parts, "\n")
}


func reusableProfileKeys(profile map[string]any) []string {
	if len(profile) == 0 {
		return nil
	}
	keys := make([]string, 0, len(profile))
	for key, value := range profile {
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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

// buildSupervisorPrompt 返回 embed 嵌入的统一 supervisor 路由提示词。
//
// 提示词文件位于 internal/prompts/supervisor/unified_router.md，定义了 L0-L2 的三层路由
// 分类体系（对话意图 → 命理领域 → 任务意图）及完整的输出格式规范。
// embed 在编译期把内容打进二进制，加载不会失败——本函数保留 (string, error) 签名是为了
// 兼容 loadSupervisorPrompt var 的测试替换模式。
func buildSupervisorPrompt() (string, error) {
	return prompts.SupervisorUnifiedRouter, nil
}
