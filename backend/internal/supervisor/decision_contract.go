// Package supervisor 暂与 client.go 共享包注释。
//
// 本文件定义决策重试合约的类型和辅助函数。
//
// # 类型体系
//
// 这里定义了三个内部类型——decisionOutput、decisionSlotsOutput、decisionPolicyHintsOutput——
// 它们与 schemas.SupervisorDecision 结构相似但语义不同：
//   - schemas.SupervisorDecision 是系统的公共"真理"类型，带 Normalize() 方法和完整的业务语义。
//   - 这些 Output 类型仅作为 tool_use / ADK 工具的参数契约，字段全部 omitempty，
//     允许模型只填充它能确定的字段，由后续的 parseAndValidate + Normalize 补齐默认值。
//
// 这种双层设计避免了循环依赖（schemas 不应知道 supervisor 的重试机制），
// 同时保持 JSON schema 与 Anthropic tool_use / Eino InferTool 的对齐。

package supervisor

import (
	"fmt"
	"strings"
)

const (
	// decisionToolName 是 Anthropic tool_use 和 Eino ADK 工具的统一名称。
	// 模型必须调用名为 "output" 的工具来返回决策结果。
	decisionToolName = "output"
	// decisionToolDescription 告知模型该工具的用途——输出结构化的路由决策。
	decisionToolDescription = "输出结构化的路由决策结果，必须调用该工具返回完整的 SupervisorDecision JSON 字段。"
	// decisionRetryPrefix 是重试提示中标记校验错误的前缀，同时也是 decisionRetryFeedbackFromError
	// 从错误链中提取反馈的锚点字符串。修改此处需同步更新提取逻辑。
	decisionRetryPrefix = "返回的 JSON 有误:"
)

// decisionSlotsOutput 是工具参数中的 slots 子对象，对应 schemas.DecisionSlots。
// 所有字段 omitempty，模型只需填充它从消息中实际提取到的字段。
type decisionSlotsOutput struct {
	Profile       map[string]any `json:"profile,omitempty"`
	QuestionText  string         `json:"question_text,omitempty"`
	TimeScope     string         `json:"time_scope,omitempty"`
	TargetSubject string         `json:"target_subject,omitempty"`
	Language      string         `json:"language,omitempty"`
}

// decisionPolicyHintsOutput 是工具参数中的 policy_hints 子对象，对应 schemas.PolicyHints。
type decisionPolicyHintsOutput struct {
	NeedsKnowledge         bool   `json:"needs_knowledge,omitempty"`
	NeedsQimen             bool   `json:"needs_qimen,omitempty"`
	QimenMode              string `json:"qimen_mode,omitempty"`
	ProfileRequirement     string `json:"profile_requirement,omitempty"`
	CanReuseSessionProfile bool   `json:"can_reuse_session_profile,omitempty"`
	CanReuseCachedResult   bool   `json:"can_reuse_cached_result,omitempty"`
}

// decisionOutput 是工具参数的顶层结构，与 SupervisorDecision 的 JSON schema 保持一致。
// 用于 Eino InferTool 的类型推断——InferTool 通过反射此结构体生成工具的 input_schema。
type decisionOutput struct {
	ConversationIntent    string                    `json:"conversation_intent,omitempty"`
	PrimaryDomain         string                    `json:"primary_domain,omitempty"`
	SecondaryDomains      []string                  `json:"secondary_domains,omitempty"`
	TaskIntent            string                    `json:"task_intent,omitempty"`
	NeedsClarification    bool                      `json:"needs_clarification,omitempty"`
	ClarificationQuestion string                    `json:"clarification_question,omitempty"`
	Parallelizable        bool                      `json:"parallelizable,omitempty"`
	Confidence            float64                   `json:"confidence,omitempty"`
	Slots                 decisionSlotsOutput       `json:"slots,omitempty"`
	PolicyHints           decisionPolicyHintsOutput `json:"policy_hints,omitempty"`
}

// decisionRetryPrompt 构建注入到重试消息中的校验错误反馈。
//
// 将 parseAndValidate 返回的业务校验错误包装为模型可理解的修正指令。
// 提示词中特别强调了 slots.profile 的来源要求——这是最常见的模型犯错点
// （模型倾向于填入示例值而非用户消息中的实际值）。
func decisionRetryPrompt(err error) string {
	return fmt.Sprintf(
		"%s %s。请重新返回完整的 JSON，特别注意 slots.profile 必须从用户原始消息中提取实际值，不要用示例值或空对象。",
		decisionRetryPrefix,
		err.Error(),
	)
}

// decisionRetryFeedbackFromError 从错误信息中提取可供模型自我修正的反馈文本。
//
// 在 ADK 引擎的 Decide 方法中，工具执行失败时错误信息会包含 decisionRetryPrefix 前缀。
// 本函数检测该前缀并提取完整的反馈内容，用于构造修正重试消息。
// 返回 false 表示错误不包含可识别的校验反馈（如网络错误），不应触发业务重试。
func decisionRetryFeedbackFromError(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	msg := err.Error()
	idx := strings.Index(msg, decisionRetryPrefix)
	if idx < 0 {
		return "", false
	}
	return msg[idx:], true
}

// decisionRetryMessage 将原始用户消息与系统纠错反馈拼接，构成修正重试的输入。
//
// 关键设计：使用"系统纠错反馈"标记明确告知模型这不是新用户问题，避免模型
// 将反馈文本误解为用户的补充信息而改变原始事实（如出生日期等）。
func decisionRetryMessage(msg, feedback string) string {
	return fmt.Sprintf(
		"%s\n\n系统纠错反馈（不是用户新问题，必须据此修正结构化输出，不要改变原始用户事实）:\n%s",
		msg,
		feedback,
	)
}
