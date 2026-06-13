package state

import "time"

// Turn 表示一轮对话中的一条消息（用户或助手）。
type Turn struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// MaxRecentTurns 是 RecentTurns 保留窗口大小（约 4 轮问答）。
const MaxRecentTurns = 8

// RoutingSnapshot captures the last supervisor routing decision written into session state.
type RoutingSnapshot struct {
	ConversationIntent    string   `json:"conversation_intent,omitempty"`
	PrimaryDomain         string   `json:"primary_domain,omitempty"`
	SecondaryDomains      []string `json:"secondary_domains,omitempty"`
	TaskIntent            string   `json:"task_intent,omitempty"`
	AwaitingClarification bool     `json:"awaiting_clarification,omitempty"`
	Confidence            float64  `json:"confidence,omitempty"`
	TimeScope             string   `json:"time_scope,omitempty"`
	TargetSubject         string   `json:"target_subject,omitempty"`
}

// BaziState holds bazi-specific domain state.
type BaziState struct {
	ProfileComplete bool `json:"profile_complete,omitempty"`
	ChartReady      bool `json:"chart_ready,omitempty"`
}

// QimenState holds qimen-specific domain state.
type QimenState struct {
	LastTimeScope string `json:"last_time_scope,omitempty"`
}

// ZiWeiState holds ziwei-specific domain state.
type ZiWeiState struct {
	ChartReady bool `json:"chart_ready,omitempty"`
}

// DomainStates groups per-domain state for the session.
type DomainStates struct {
	Bazi  BaziState  `json:"bazi,omitempty"`
	Qimen QimenState `json:"qimen,omitempty"`
	ZiWei ZiWeiState `json:"ziwei,omitempty"`
}

type SessionState struct {
	SessionID           string
	Profile             map[string]any // {year,month,day,hour,gender,...}
	BaziResult          map[string]any // bazi_calc 输出
	QimenResult         map[string]any // qimen_dunjia 输出（首次后缓存，不再重复 emit UI 组件）
	ZiWeiResult         map[string]any // ziwei chart 输出
	ConversationStage   string         // "collecting" | "ready" | "completed"
	ConversationSummary string
	LastUserQuestion    string
	NeedsQimen     bool // set by classifyAndExtract, consumed by handleFollowupReading
	NeedsKnowledge bool // set by classifyAndExtract: whether to run knowledge search

	// 上下文工程第一阶段：会话内最近多轮对话 + 滚动摘要
	RecentTurns    []Turn `json:"recent_turns,omitempty"`
	RunningSummary string `json:"running_summary,omitempty"`

	// Supervisor 架构：路由快照 + 领域状态
	Routing      RoutingSnapshot `json:"routing,omitempty"`
	DomainStates DomainStates    `json:"domain_states,omitempty"`
}

func NewSession(id string) *SessionState {
	return &SessionState{
		SessionID:         id,
		Profile:           make(map[string]any),
		ConversationStage: "collecting",
		RecentTurns:       make([]Turn, 0),
	}
}

var requiredFields = []string{"year", "month", "day", "hour", "gender", "birthplace"}

func (s *SessionState) MissingFields() []string {
	var missing []string
	for _, f := range requiredFields {
		if _, ok := s.Profile[f]; !ok {
			missing = append(missing, f)
		}
	}
	return missing
}

func (s *SessionState) IsProfileComplete() bool {
	return len(s.MissingFields()) == 0
}

// HasBaziResult reports whether the session already has a reusable chart context.
func (s *SessionState) HasBaziResult() bool {
	return s != nil && len(s.BaziResult) > 0
}

// HasQimenResult reports whether qimen data was already emitted in this session.
func (s *SessionState) HasQimenResult() bool {
	return s != nil && len(s.QimenResult) > 0
}

// ZiWeiResult holds the ziwei chart output.
func (s *SessionState) HasZiWeiResult() bool {
	return s != nil && s.ZiWeiResult != nil && len(s.ZiWeiResult) > 0
}

func (s *SessionState) MergeProfile(patch map[string]any) bool {
	changed := false
	for k, v := range patch {
		if old, ok := s.Profile[k]; !ok || old != v {
			s.Profile[k] = v
			changed = true
		}
	}
	return changed
}

// RecordTurn appends a new turn to the recent history.
func (s *SessionState) RecordTurn(role, content string) {
	s.RecentTurns = append(s.RecentTurns, Turn{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// TrimTurns removes turns beyond MaxRecentTurns and returns the overflow.
// Caller is responsible for summarizing the returned turns into RunningSummary.
func (s *SessionState) TrimTurns() []Turn {
	if len(s.RecentTurns) <= MaxRecentTurns {
		return nil
	}
	excess := len(s.RecentTurns) - MaxRecentTurns
	overflow := make([]Turn, excess)
	copy(overflow, s.RecentTurns[:excess])
	s.RecentTurns = append([]Turn{}, s.RecentTurns[excess:]...)
	return overflow
}

// ActivePrimaryDomain returns the currently active primary domain.
// Defaults to "bazi" if not explicitly set.
func (s *SessionState) ActivePrimaryDomain() string {
	if s.ConversationStage == "qimen" {
		return "qimen"
	}
	return "bazi"
}

// SetActivePrimaryDomain records the active primary domain for the session.
func (s *SessionState) SetActivePrimaryDomain(domain string) {
	if domain == "qimen" {
		s.ConversationStage = "qimen"
		return
	}
	s.ConversationStage = "ready"
}

// Clone returns a detached copy that can be mutated without affecting the session in store.
func (s *SessionState) Clone() *SessionState {
	if s == nil {
		return nil
	}
	clone := &SessionState{
		SessionID:           s.SessionID,
		ConversationStage:   s.ConversationStage,
		ConversationSummary: s.ConversationSummary,
		LastUserQuestion:    s.LastUserQuestion,
		NeedsQimen:          s.NeedsQimen,
		NeedsKnowledge:      s.NeedsKnowledge,
			RunningSummary:      s.RunningSummary,
			Routing:             s.Routing,
			DomainStates:        s.DomainStates,
	}
	if s.Profile != nil {
		clone.Profile = make(map[string]any, len(s.Profile))
		for k, v := range s.Profile {
			clone.Profile[k] = v
		}
	} else {
		clone.Profile = make(map[string]any)
	}
	if s.BaziResult != nil {
		clone.BaziResult = make(map[string]any, len(s.BaziResult))
		for k, v := range s.BaziResult {
			clone.BaziResult[k] = v
		}
	}
	if s.QimenResult != nil {
		clone.QimenResult = make(map[string]any, len(s.QimenResult))
		for k, v := range s.QimenResult {
			clone.QimenResult[k] = v
		}
	}
	if len(s.RecentTurns) > 0 {
		clone.RecentTurns = make([]Turn, len(s.RecentTurns))
		copy(clone.RecentTurns, s.RecentTurns)
	} else {
		clone.RecentTurns = make([]Turn, 0)
	}
	return clone
}
