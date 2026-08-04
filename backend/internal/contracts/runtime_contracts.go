// This file belongs to the runtime/frontend contract layer.
// It owns shared runtime contract shapes for this package.
// It defines shared DTOs; execution policy stays in runtime and orchestrator.
package contracts

// LastInputState records the user-facing input choices that are useful for
// restoring the next turn's UI affordances. It is not a runtime truth source.
type LastInputState struct {
	PreferredDomain       string   `json:"preferred_domain,omitempty"`
	SecondaryDomains      []string `json:"secondary_domains,omitempty"`
	ExplicitMethod        string   `json:"explicit_method,omitempty"`
	ConsultMode           string   `json:"consult_mode,omitempty"`
	TimeScope             string   `json:"time_scope,omitempty"`
	TargetSubject         string   `json:"target_subject,omitempty"`
	QuestionText          string   `json:"question_text,omitempty"`
	GuidanceActive        bool     `json:"guidance_active,omitempty"`
	GuidanceDirectiveKind string   `json:"guidance_directive_kind,omitempty"`
}

// GateContract captures the deterministic execution policy applied after the
// supervisor proposes a route and before runtime executes it.
type GateContract struct {
	Admitted            bool     `json:"admitted"`
	Reason              string   `json:"reason,omitempty"`
	AllowedDomains      []string `json:"allowed_domains,omitempty"`
	ProfileRequirement  string   `json:"profile_requirement,omitempty"`
	ReuseSessionProfile bool     `json:"reuse_session_profile,omitempty"`
	ReuseCachedResult   bool     `json:"reuse_cached_result,omitempty"`
	ExecutionMode       string   `json:"execution_mode,omitempty"`
	GuidancePolicy      string   `json:"guidance_policy,omitempty"`
	FollowupPolicy      string   `json:"followup_policy,omitempty"`
}

// HasSignal reports whether the contract contains any meaningful gate decision
// beyond the zero-value shape.
func (g GateContract) HasSignal() bool {
	return g.Admitted ||
		g.Reason != "" ||
		len(g.AllowedDomains) > 0 ||
		g.ProfileRequirement != "" ||
		g.ReuseSessionProfile ||
		g.ReuseCachedResult ||
		g.ExecutionMode != "" ||
		g.GuidancePolicy != "" ||
		g.FollowupPolicy != ""
}

// ExecutionSnapshot captures the real execution contract that the runtime used
// for the current turn. Unlike LastInputState, this is a runtime truth source.
type ExecutionSnapshot struct {
	PrimaryDomain      string       `json:"primary_domain,omitempty"`
	SecondaryDomains   []string     `json:"secondary_domains,omitempty"`
	Domains            []string     `json:"domains,omitempty"`
	TaskIntent         string       `json:"task_intent,omitempty"`
	ConversationIntent string       `json:"conversation_intent,omitempty"`
	RequiredArtifacts  []string     `json:"required_artifacts,omitempty"`
	FollowupMode       string       `json:"followup_mode,omitempty"`
	NeedsClarification bool         `json:"needs_clarification,omitempty"`
	QimenMode          string       `json:"qimen_mode,omitempty"`
	TargetSubject      string       `json:"target_subject,omitempty"`
	TimeScope          string       `json:"time_scope,omitempty"`
	Gate               GateContract `json:"gate,omitempty"`
}

// HasSignal reports whether the snapshot carries a real runtime contract.
func (s ExecutionSnapshot) HasSignal() bool {
	return s.PrimaryDomain != "" ||
		len(s.SecondaryDomains) > 0 ||
		len(s.Domains) > 0 ||
		s.TaskIntent != "" ||
		s.ConversationIntent != "" ||
		len(s.RequiredArtifacts) > 0 ||
		s.FollowupMode != "" ||
		s.NeedsClarification ||
		s.QimenMode != "" ||
		s.TargetSubject != "" ||
		s.TimeScope != "" ||
		s.Gate.HasSignal()
}
