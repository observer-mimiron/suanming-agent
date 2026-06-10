package state

type SessionState struct {
	SessionID           string
	Profile             map[string]any // {year,month,day,hour,gender,...}
	BaziResult          map[string]any // bazi_calc 输出
	ConversationStage   string         // "collecting" | "ready" | "completed"
	ConversationSummary string
	LastUserQuestion    string
}

func NewSession(id string) *SessionState {
	return &SessionState{
		SessionID:         id,
		Profile:           make(map[string]any),
		ConversationStage: "collecting",
	}
}

var requiredFields = []string{"year", "month", "day", "hour", "gender"}

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
