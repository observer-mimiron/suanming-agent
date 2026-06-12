package schemas

// SupervisorDecision is the structured output from the LLM supervisor.
// It represents layered routing decisions: L0 intent, L1 domain, L2 task, L3 slots/hints.
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

// DecisionSlots holds structured slot values extracted by the supervisor.
type DecisionSlots struct {
	Profile       map[string]any `json:"profile"`
	QuestionText  string         `json:"question_text"`
	TimeScope     string         `json:"time_scope"`
	TargetSubject string         `json:"target_subject"`
	Language      string         `json:"language"`
}

// PolicyHints are flags that inform the policy gate about optional behaviors.
type PolicyHints struct {
	NeedsKnowledge         bool `json:"needs_knowledge"`
	NeedsQimen             bool `json:"needs_qimen"`
	CanReuseSessionProfile bool `json:"can_reuse_session_profile"`
	CanReuseCachedResult   bool `json:"can_reuse_cached_result"`
}

// Normalize applies safe defaults to a SupervisorDecision after parsing.
func (d *SupervisorDecision) Normalize() {
	if d.ConversationIntent == "" {
		d.ConversationIntent = "consult"
	}
	if d.PrimaryDomain == "" {
		d.PrimaryDomain = "bazi"
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
}
