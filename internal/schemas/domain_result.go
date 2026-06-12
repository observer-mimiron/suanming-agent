package schemas

// DomainResult is the typed return contract from a domain specialist.
type DomainResult struct {
	Domain            string         `json:"domain"`
	Summary           string         `json:"summary"`
	StructuredData    map[string]any `json:"structured_data,omitempty"`
	Evidence          []string       `json:"evidence,omitempty"`
	FollowupQuestions []string       `json:"followup_questions,omitempty"`
	Final             bool           `json:"final"`
}
