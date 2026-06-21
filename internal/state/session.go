// 本文件定义会话状态的数据结构和方法。包注释见 locker.go。

package state

import "time"

// Turn 表示一轮对话中的一条消息（用户或助手）。
type Turn struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// MaxRecentTurns 是 RecentTurns 保留窗口大小（约 4 轮问答）。
const MaxRecentTurns = 30

// RoutingSnapshot 记录写入会话状态的上一次 supervisor 路由决策。
type RoutingSnapshot struct {
	ConversationIntent    string   `json:"conversation_intent,omitempty"`
	PrimaryDomain         string   `json:"primary_domain,omitempty"`
	SecondaryDomains      []string `json:"secondary_domains,omitempty"`
	TaskIntent            string   `json:"task_intent,omitempty"`
	QimenMode             string   `json:"qimen_mode,omitempty"`
	AwaitingClarification bool     `json:"awaiting_clarification,omitempty"`
	Confidence            float64  `json:"confidence,omitempty"`
	TimeScope             string   `json:"time_scope,omitempty"`
	TargetSubject         string   `json:"target_subject,omitempty"`
}

// BaziState 存储八字领域的会话状态。
type BaziState struct {
	ProfileComplete bool `json:"profile_complete,omitempty"`
	ChartReady      bool `json:"chart_ready,omitempty"`
}

// QimenState 存储奇门遁甲领域的会话状态。
type QimenState struct {
	LastTimeScope string `json:"last_time_scope,omitempty"`
}

// ZiWeiState 存储紫微斗数领域的会话状态。
type ZiWeiState struct {
	ChartReady bool `json:"chart_ready,omitempty"`
}

// DomainStates 聚合会话中各个领域的独立状态。
type DomainStates struct {
	Bazi  BaziState  `json:"bazi,omitempty"`
	Qimen QimenState `json:"qimen,omitempty"`
	ZiWei ZiWeiState `json:"ziwei,omitempty"`
}

// GuidanceState 记录当前会话里的对话引导状态。
//
// 它只保存“引导到了哪一步”和少量复用信息，不负责决定如何迁移。
type GuidanceState struct {
	DirectiveKind string `json:"directive_kind,omitempty"`
	ChosenTopic   string `json:"chosen_topic,omitempty"`
	PendingSlot   string `json:"pending_slot,omitempty"`
	RetryCount    int    `json:"retry_count,omitempty"`
}

// SessionState 表示一个会话的完整状态，包含用户资料、命盘结果、路由快照、对话历史等。
type SessionState struct {
	SessionID           string
	Profile             map[string]any // {year,month,day,hour,gender,...}
	BaziResult          map[string]any // bazi_calc 输出
	QimenResult         map[string]any // qimen_dunjia 输出（首次后缓存，不再重复 emit UI 组件）
	ZiWeiResult         map[string]any // ziwei chart 输出
	ConversationStage   string         // "collecting" | "ready" | "completed"
	ConversationSummary string
	LastUserQuestion    string
	NeedsQimen          bool // set by the approved route / specialist dispatch, consumed by followup execution
	NeedsKnowledge      bool // legacy flag kept for compatibility with existing session snapshots

	// 上下文工程第一阶段：会话内最近多轮对话 + 滚动摘要
	RecentTurns    []Turn `json:"recent_turns,omitempty"`
	RunningSummary string `json:"running_summary,omitempty"`

	// Supervisor 架构：路由快照 + 领域状态
	Routing      RoutingSnapshot `json:"routing,omitempty"`
	DomainStates DomainStates    `json:"domain_states,omitempty"`

	// Guidance 只持久化对话引导状态，不承载迁移规则。
	Guidance *GuidanceState `json:"guidance,omitempty"`
}

// NewSession 创建一个新的会话状态实例，初始化空资料和收集阶段。
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

// HasBaziResult 判断会话是否已有可复用的命盘上下文。
func (s *SessionState) HasBaziResult() bool {
	return s != nil && len(s.BaziResult) > 0
}

// HasQimenResult 判断会话中是否已发出过奇门数据。
func (s *SessionState) HasQimenResult() bool {
	return s != nil && len(s.QimenResult) > 0
}

// HasZiWeiResult 判断会话中是否存在紫微斗数命盘结果。
func (s *SessionState) HasZiWeiResult() bool {
	return s != nil && s.ZiWeiResult != nil && len(s.ZiWeiResult) > 0
}

// MergeProfile 将 patch 中的字段合并到当前 Profile 中，返回是否有变更。
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

// RecordTurn 将一条新消息追加到最近对话历史中。
func (s *SessionState) RecordTurn(role, content string) {
	s.RecentTurns = append(s.RecentTurns, Turn{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// TrimTurns 移除超出 MaxRecentTurns 的旧轮次并返回被截断的内容。
// 调用方应将返回的溢出轮次汇总到 RunningSummary 中。
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

// ActivePrimaryDomain 返回当前活跃的主领域，默认为 "bazi"。
func (s *SessionState) ActivePrimaryDomain() string {
	if s.ConversationStage == "qimen" {
		return "qimen"
	}
	return "bazi"
}

// SetActivePrimaryDomain 记录当前活跃的主领域。
func (s *SessionState) SetActivePrimaryDomain(domain string) {
	if domain == "qimen" {
		s.ConversationStage = "qimen"
		return
	}
	s.ConversationStage = "ready"
}

// Clone 返回一个独立的会话状态副本，修改副本不影响原始会话。
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
	if s.ZiWeiResult != nil {
		clone.ZiWeiResult = make(map[string]any, len(s.ZiWeiResult))
		for k, v := range s.ZiWeiResult {
			clone.ZiWeiResult[k] = v
		}
	}
	if len(s.RecentTurns) > 0 {
		clone.RecentTurns = make([]Turn, len(s.RecentTurns))
		copy(clone.RecentTurns, s.RecentTurns)
	} else {
		clone.RecentTurns = make([]Turn, 0)
	}
	if s.Guidance != nil {
		clone.Guidance = &GuidanceState{
			DirectiveKind: s.Guidance.DirectiveKind,
			ChosenTopic:   s.Guidance.ChosenTopic,
			PendingSlot:   s.Guidance.PendingSlot,
			RetryCount:    s.Guidance.RetryCount,
		}
	}
	return clone
}
