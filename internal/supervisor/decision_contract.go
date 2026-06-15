// Package supervisor 暂与 client.go 共享包注释，本文件定义决策合约的类型和工具函数。

package supervisor

import (
	"fmt"
	"strings"
)

const (
	decisionToolName        = "output"
	decisionToolDescription = "输出结构化的路由决策结果，必须调用该工具返回完整的 SupervisorDecision JSON 字段。"
	decisionRetryPrefix     = "返回的 JSON 有误:"
)

type decisionSlotsOutput struct {
	Profile       map[string]any `json:"profile,omitempty"`
	QuestionText  string         `json:"question_text,omitempty"`
	TimeScope     string         `json:"time_scope,omitempty"`
	TargetSubject string         `json:"target_subject,omitempty"`
	Language      string         `json:"language,omitempty"`
}

type decisionPolicyHintsOutput struct {
	NeedsKnowledge         bool `json:"needs_knowledge,omitempty"`
	NeedsQimen             bool `json:"needs_qimen,omitempty"`
	CanReuseSessionProfile bool `json:"can_reuse_session_profile,omitempty"`
	CanReuseCachedResult   bool `json:"can_reuse_cached_result,omitempty"`
}

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

func decisionRetryPrompt(err error) string {
	return fmt.Sprintf(
		"%s %s。请重新返回完整的 JSON，特别注意 slots.profile 必须从用户原始消息中提取实际值，不要用示例值或空对象。",
		decisionRetryPrefix,
		err.Error(),
	)
}

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

func decisionRetryMessage(msg, feedback string) string {
	return fmt.Sprintf(
		"%s\n\n系统纠错反馈（不是用户新问题，必须据此修正结构化输出，不要改变原始用户事实）:\n%s",
		msg,
		feedback,
	)
}
