// Package schemas 暂与 domain_result.go 共享包注释，本文件定义 supervisor 的层次化路由决策结构。

package schemas

// SupervisorDecision 是 LLM supervisor 的结构化路由输出。
// 包含三层决策：L0 对话意图、L1 领域、L2 任务、L3 槽位/提示。
// 注：引导动作（offer_consult / choose_topic / collect_slot / guided_fallback）已移至 code-side guidance 状态机。
type SupervisorDecision struct {
	ConversationIntent    string        `json:"conversation_intent"`
	PrimaryDomain         string        `json:"primary_domain"`
	SecondaryDomains      []string      `json:"secondary_domains"`
	TaskIntent            string        `json:"task_intent"`
	NeedsClarification    bool          `json:"needs_clarification"`
	ClarificationQuestion string        `json:"clarification_question"`
	Parallelizable        bool          `json:"parallelizable"`
	Confidence            float64       `json:"confidence"`
	Slots                 DecisionSlots `json:"slots"`
	PolicyHints           PolicyHints   `json:"policy_hints"`
}

// DecisionSlots 存储 supervisor 从用户消息中提取的结构化槽位值。
type DecisionSlots struct {
	Profile       map[string]any `json:"profile"`
	QuestionText  string         `json:"question_text"`
	TimeScope     string         `json:"time_scope"`
	TargetSubject string         `json:"target_subject"`
	Language      string         `json:"language"`
}

// PolicyHints 是通知策略门控的可选行为标志。
type PolicyHints struct {
	NeedsKnowledge         bool   `json:"needs_knowledge"`
	NeedsQimen             bool   `json:"needs_qimen"`
	QimenMode              string `json:"qimen_mode,omitempty"`
	ProfileRequirement     string `json:"profile_requirement,omitempty"`
	CanReuseSessionProfile bool   `json:"can_reuse_session_profile"`
	CanReuseCachedResult   bool   `json:"can_reuse_cached_result"`
}

// Normalize 为解析后的 SupervisorDecision 应用安全默认值。
func (d *SupervisorDecision) Normalize() {
	if d.ConversationIntent == "" {
		d.ConversationIntent = "consult"
	}
	if d.PrimaryDomain == "" {
		d.PrimaryDomain = "bazi"
	}
	if d.TaskIntent == "" {
		d.TaskIntent = "collect_profile"
	}
	if d.Confidence < 0 {
		d.Confidence = 0
	}
	if d.SecondaryDomains == nil {
		d.SecondaryDomains = []string{}
	}
	if d.Slots.Profile == nil {
		d.Slots.Profile = map[string]any{}
	}
	if d.PolicyHints.QimenMode == "" {
		switch {
		case d.PrimaryDomain == "qimen":
			d.PolicyHints.QimenMode = "primary"
		case d.PolicyHints.NeedsQimen:
			d.PolicyHints.QimenMode = "supplement"
		default:
			d.PolicyHints.QimenMode = "none"
		}
	}
	if d.PolicyHints.ProfileRequirement == "" && d.PolicyHints.QimenMode == "primary" && d.PrimaryDomain == "qimen" {
		d.PolicyHints.ProfileRequirement = "none"
	}
}
