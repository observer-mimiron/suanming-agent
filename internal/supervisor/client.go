package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Client calls the LLM supervisor and returns structured decisions.
type Client struct {
	flash llm.Chat
}

// NewClient creates a supervisor client backed by the given flash model.
func NewClient(flash llm.Chat) *Client {
	return &Client{flash: flash}
}

// Decide runs the supervisor through a three-layer defense:
//
//	Layer 1 — structuredDecide (constrained decoding):
//	  Uses GenerateWithTool with forced tool_choice. The model is required to
//	  call the "output" tool whose input_schema is the SupervisorDecision schema.
//	  This guarantees the JSON structure matches — not probabilistic, mathematical.
//	  Up to 2 attempts with error feedback on validation failure.
//
//	Layer 2 — textDecide (prompt + validate + retry):
//	  Falls back to plain text generation if layer 1 fails (API error or
//	  persistent validation failure). The model is prompted to output JSON,
//	  the response is parsed and validated, and specific error feedback is
//	  injected into retry prompts. Up to 3 attempts.
//
//	Layer 3 — fallbackExtract / safeFallback:
//	  If all layer-2 attempts fail, falls back to a simplified extraction prompt
//	  without complex routing. If the model is completely unavailable, returns
//	  hardcoded conservative defaults via safeFallback.
func (c *Client) Decide(ctx context.Context, msg string, st *state.SessionState) (schemas.SupervisorDecision, error) {
	prompt, err := buildSupervisorPrompt()
	if err != nil {
		return safeFallback(st), fmt.Errorf("supervisor prompt: %w", err)
	}

	// Inject session context so the supervisor knows what data already exists.
	sessionCtx := buildSessionContext(st)
	if sessionCtx != "" {
		prompt += "\n\n## 当前会话状态\n\n" + sessionCtx
	}

	messages := []llm.Message{
		{Role: "user", Content: msg},
	}

	// Layer 1: Try constrained decoding via forced tool use.
	// This guarantees the response JSON conforms to the schema.
	decision, err := c.structuredDecide(ctx, prompt, messages)
	if err == nil {
		return decision, nil
	}

	// Layer 2+3: Fall back to text-based generation with validation + retries.
	return c.textDecide(ctx, prompt, messages, st, msg)
}

// structuredDecide (layer 1): constrained decoding via forced tool_choice.
// The model must call the "output" tool, whose input_schema is the SupervisorDecision
// schema — this guarantees structurally valid JSON. Up to 2 attempts with error
// feedback on domain-level validation failures.
func (c *Client) structuredDecide(ctx context.Context, prompt string, messages []llm.Message) (schemas.SupervisorDecision, error) {
	tool := buildDecisionTool()

	for attempt := 0; attempt < 2; attempt++ {
		input, _, err := c.flash.GenerateWithTool(ctx, prompt, messages, tool)
		if err != nil {
			return schemas.SupervisorDecision{}, fmt.Errorf("structured: %w", err)
		}

		// Convert the tool input back to JSON string for parseAndValidate
		raw, _ := json.Marshal(input)
		decision, validationErr := parseAndValidate(string(raw))
		if validationErr == nil {
			return decision, nil
		}

		// Validation failed — feed error back for self-correction.
		messages = append(messages,
			llm.Message{Role: "assistant", Content: string(raw)},
			llm.Message{Role: "user", Content: fmt.Sprintf(
				"返回的 JSON 有误: %s。请重新返回完整的 JSON，特别注意 slots.profile 必须从用户原始消息中提取实际值，不要用示例值或空对象。", validationErr.Error(),
			)},
		)
	}

	return schemas.SupervisorDecision{}, fmt.Errorf("structured: validation failed after retry")
}

// textDecide (layer 2): plain text generation with JSON parsing, domain validation,
// and up to 3 retries. On each validation failure, the specific error is injected
// into the next prompt so the model can self-correct. Falls through to layer 3
// (fallbackExtract) if all retries are exhausted.
func (c *Client) textDecide(ctx context.Context, prompt string, messages []llm.Message, st *state.SessionState, msg string) (schemas.SupervisorDecision, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, _, err := c.flash.Generate(ctx, prompt, messages)
		if err != nil {
			lastErr = fmt.Errorf("supervisor call: %w", err)
			continue
		}

		decision, validationErr := parseAndValidate(resp)
		if validationErr == nil {
			return decision, nil
		}

		lastErr = validationErr
		// Append error feedback so the model can self-correct on the next attempt.
		messages = append(messages,
			llm.Message{Role: "assistant", Content: resp},
			llm.Message{Role: "user", Content: fmt.Sprintf(
				"返回的 JSON 有误: %s。请重新返回完整的 JSON，特别注意 slots.profile 必须从用户原始消息中提取实际值，不要用示例值或空对象。", validationErr.Error(),
			)},
		)
	}

	// All retries exhausted — fall back to focused extraction.
	return c.fallbackExtract(ctx, msg, st), lastErr
}

// parseDecision unmarshals raw JSON into a normalized SupervisorDecision.
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

// parseAndValidate parses the LLM response and runs domain-specific validation.
func parseAndValidate(raw string) (schemas.SupervisorDecision, error) {
	d := parseDecision(raw)

	if d.TaskIntent == "collect_profile" && len(d.Slots.Profile) == 0 {
		return d, fmt.Errorf("task_intent 为 collect_profile，但 slots.profile 为空。必须从用户原始消息中提取出生信息")
	}

	// Detect hallucinated profile: when the LLM fills in default values for missing fields.
	// If profile has suspicious defaults (month=1, day=1, hour=0) alongside only 1-2 real fields,
	// it's likely the model fabricated the rest. Reject and retry.
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

// looksHallucinated checks if a profile likely contains fabricated default values.
// Signal: month=1, day=1, and hour=0 all present — classic LLM defaults for missing birth info.
func looksHallucinated(profile map[string]any) bool {
	month, hasMonth := profile["month"].(float64)
	day, hasDay := profile["day"].(float64)
	hour, hasHour := profile["hour"].(float64)
	if !hasMonth || !hasDay || !hasHour {
		return false
	}
	return month == 1 && day == 1 && hour == 0
}

// fallbackExtract (layer 3a): last-resort LLM extraction when all retries fail.
// Uses a focused, single-purpose prompt — no complex routing, just extract what's there.
// Returns hardcoded defaults if even this fails.
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
	// Ensure critical fields have sane defaults after fallback.
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

// buildSessionContext builds a concise summary of the current session state
// for injection into the supervisor prompt.
func buildSessionContext(st *state.SessionState) string {
	hasProfile := len(st.Profile) > 0
	hasChart := st.HasBaziResult()
	isComplete := st.IsProfileComplete()

	if !hasProfile && !hasChart {
		return "会话状态：新会话，尚无任何用户资料或命盘。"
	}

	var parts []string
	if hasProfile {
		profileJSON, _ := json.Marshal(st.Profile)
		parts = append(parts, fmt.Sprintf("已有资料：%s", string(profileJSON)))
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

	return strings.Join(parts, "\n") + "\n\n根据以上会话状态：\n- 如果用户提供新的出生信息，且已有完整资料 → task_intent 应为 amend_profile\n- 如果用户追问且已有命盘 → task_intent 应为 fortune_followup，can_reuse_cached_result=true\n- 如果用户仅补充部分字段（如「我是女的」）且已有资料 → task_intent 应为 amend_profile，can_reuse_session_profile=true"
}

// safeFallback (layer 3b): hardcoded conservative defaults when the LLM is unavailable.
// No network calls — pure logic based on current session state.
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

// buildSupervisorPrompt loads the unified supervisor prompt.
func buildSupervisorPrompt() (string, error) {
	b, err := os.ReadFile("prompts/supervisor/unified_router.md")
	if err != nil {
		return "", fmt.Errorf("read unified_router.md: %w", err)
	}
	return string(b), nil
}

// buildDecisionTool returns the SupervisorDecision schema as an Anthropic tool definition.
// This enables layer-1 constrained decoding: the model is forced to call this tool,
// guaranteeing the output JSON conforms to the schema.
func buildDecisionTool() llm.ToolDef {
	return llm.ToolDef{
		Name:        "output",
		Description: "输出结构化的路由决策结果，包含对话意图分类、领域路由、任务意图和提取的槽位数据",
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
