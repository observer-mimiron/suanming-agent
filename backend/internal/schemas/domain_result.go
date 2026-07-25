// Package schemas 定义领域专家返回值和 supervisor 决策的数据结构。
// 这些类型在 LLM、编排器、前端之间传递，构成系统的契约边界。
package schemas

// DomainResult 是领域专家的类型化返回契约，包含分析摘要、结构化数据和后续追问。
type DomainResult struct {
	Domain            string         `json:"domain"`
	Summary           string         `json:"summary"`
	StructuredData    map[string]any `json:"structured_data,omitempty"`
	Evidence          []string       `json:"evidence,omitempty"`
	FollowupQuestions []string       `json:"followup_questions,omitempty"`
	Final             bool           `json:"final"`
}
