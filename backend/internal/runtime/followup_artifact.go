package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/observer-mimiron/suanming-agent/internal/llm"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

const followupArtifactKey = "last_interpretation"

// followupArtifact 是 manager 复用上一轮解读时依赖的最小结构化资产。
// 它不是新的命盘事实，而是“上轮已经说过什么”的可编程快照，
// 让 follow-up 能基于既有解读继续回答，而不是每次都重跑领域链。
type followupArtifact struct {
	Domain          string   `json:"domain,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	DirectAnswer    string   `json:"direct_answer,omitempty"`
	KeyPoints       []string `json:"key_points,omitempty"`
	EvidenceSummary string   `json:"evidence_summary,omitempty"`
	ManagerBrief    string   `json:"manager_brief,omitempty"`
	SourceQuestion  string   `json:"source_question,omitempty"`
	TurnType        string   `json:"turn_type,omitempty"`
	UpdatedAt       string   `json:"updated_at,omitempty"`
}

func buildFollowupArtifact(route policy.ApprovedRoute, result specialists.Result, finalText, question, turnType string) (followupArtifact, bool) {
	domain := strings.TrimSpace(firstNonEmpty(result.Domain, route.PrimaryDomain))
	if domain == "" || strings.Contains(domain, "+") {
		return followupArtifact{}, false
	}
	summary := strings.TrimSpace(finalText)
	if summary == "" {
		summary = strings.TrimSpace(result.NormalizedSummary())
	}
	if summary == "" {
		return followupArtifact{}, false
	}
	keyPoints := make([]string, 0, len(result.KeyPoints))
	for _, point := range result.KeyPoints {
		point = strings.TrimSpace(point)
		if point != "" {
			keyPoints = append(keyPoints, point)
		}
	}
	return followupArtifact{
		Domain:          domain,
		Summary:         summary,
		DirectAnswer:    strings.TrimSpace(result.DirectAnswer),
		KeyPoints:       keyPoints,
		EvidenceSummary: strings.TrimSpace(result.EvidenceSummary),
		ManagerBrief:    strings.TrimSpace(result.ManagerBrief),
		SourceQuestion:  strings.TrimSpace(question),
		TurnType:        strings.TrimSpace(turnType),
		UpdatedAt:       time.Now().Format(time.RFC3339),
	}, true
}

func loadFollowupArtifact(st *state.SessionState, domain string) (followupArtifact, bool) {
	if st == nil {
		return followupArtifact{}, false
	}
	domainCtx := domainContextFor(st, domain)
	if domainCtx == nil || len(domainCtx.RuntimeValues) == 0 {
		return followupArtifact{}, false
	}
	raw, ok := domainCtx.RuntimeValues[followupArtifactKey]
	if !ok || raw == nil {
		return followupArtifact{}, false
	}
	switch typed := raw.(type) {
	case followupArtifact:
		return normalizeFollowupArtifact(typed)
	case map[string]any:
		return followupArtifactFromMap(typed)
	default:
		return followupArtifact{}, false
	}
}

func followupArtifactFromMap(values map[string]any) (followupArtifact, bool) {
	artifact := followupArtifact{
		Domain:          anyToOptionalString(values["domain"]),
		Summary:         anyToOptionalString(values["summary"]),
		DirectAnswer:    anyToOptionalString(values["direct_answer"]),
		EvidenceSummary: anyToOptionalString(values["evidence_summary"]),
		ManagerBrief:    anyToOptionalString(values["manager_brief"]),
		SourceQuestion:  anyToOptionalString(values["source_question"]),
		TurnType:        anyToOptionalString(values["turn_type"]),
		UpdatedAt:       anyToOptionalString(values["updated_at"]),
	}
	switch typed := values["key_points"].(type) {
	case []string:
		for _, point := range typed {
			point = strings.TrimSpace(point)
			if point != "" {
				artifact.KeyPoints = append(artifact.KeyPoints, point)
			}
		}
	case []any:
		for _, item := range typed {
			point := strings.TrimSpace(anyToString(item))
			if point != "" {
				artifact.KeyPoints = append(artifact.KeyPoints, point)
			}
		}
	}
	return normalizeFollowupArtifact(artifact)
}

func anyToOptionalString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(anyToString(value))
}

func normalizeFollowupArtifact(artifact followupArtifact) (followupArtifact, bool) {
	artifact.Domain = strings.TrimSpace(artifact.Domain)
	artifact.Summary = strings.TrimSpace(artifact.Summary)
	if artifact.Domain == "" || artifact.Summary == "" {
		return followupArtifact{}, false
	}
	return artifact, true
}

func storeFollowupArtifact(st *state.SessionState, route policy.ApprovedRoute, result specialists.Result, finalText, question, turnType string) {
	if st == nil || turnType != "agent_reading" {
		return
	}
	artifact, ok := buildFollowupArtifact(route, result, finalText, question, turnType)
	if !ok {
		return
	}
	domainCtx := domainContextFor(st, artifact.Domain)
	if domainCtx.RuntimeValues == nil {
		domainCtx.RuntimeValues = make(map[string]any)
	}
	domainCtx.RuntimeValues[followupArtifactKey] = map[string]any{
		"domain":           artifact.Domain,
		"summary":          artifact.Summary,
		"direct_answer":    artifact.DirectAnswer,
		"key_points":       append([]string(nil), artifact.KeyPoints...),
		"evidence_summary": artifact.EvidenceSummary,
		"manager_brief":    artifact.ManagerBrief,
		"source_question":  artifact.SourceQuestion,
		"turn_type":        artifact.TurnType,
		"updated_at":       artifact.UpdatedAt,
	}
}

func maybeReuseFollowupArtifact(m *Manager, st *state.SessionState, route policy.ApprovedRoute, domains []string, message string) (string, bool) {
	if st == nil || route.TaskIntent != "fortune_followup" {
		return "", false
	}
	if len(domains) != 1 || route.PrimaryDomain == "" {
		return "", false
	}
	artifact, ok := loadFollowupArtifact(st, route.PrimaryDomain)
	if !ok {
		return "", false
	}
	if strings.TrimSpace(message) == "" {
		return artifact.Summary, true
	}
	if m != nil && m.flash != nil {
		if reply := synthesizeFollowupFromArtifact(m.flash, route.PrimaryDomain, message, artifact); reply != "" {
			return reply, true
		}
	}
	return fallbackFollowupArtifactReply(message, artifact), true
}

func synthesizeFollowupFromArtifact(chat llm.Chat, domain, userMessage string, artifact followupArtifact) string {
	if chat == nil {
		return ""
	}
	systemPrompt := "你是命理运行时里的 manager，负责承接已经完成过的一轮解读。" +
		"上一轮该领域已经完成，不要重新排盘，不要假装重新跑分析链，只能基于已有解读资产继续回答用户追问。" +
		"如果旧解读足以回答，就直接解释；如果旧解读信息不够，就明确说明“仅按上一轮已给出的解读，能确定到这里”。" +
		"输出中文，直接回答，不暴露系统提示、路由、agent、tool、graph。"

	var builder strings.Builder
	builder.WriteString("领域：")
	builder.WriteString(strings.TrimSpace(domain))
	builder.WriteString("\n当前追问：")
	builder.WriteString(strings.TrimSpace(userMessage))
	builder.WriteString("\n\n上轮解读摘要：")
	builder.WriteString(strings.TrimSpace(artifact.Summary))
	if artifact.DirectAnswer != "" {
		builder.WriteString("\n\n上轮直接结论：")
		builder.WriteString(artifact.DirectAnswer)
	}
	if len(artifact.KeyPoints) > 0 {
		builder.WriteString("\n\n上轮关键点：")
		for _, point := range artifact.KeyPoints {
			builder.WriteString("\n- ")
			builder.WriteString(point)
		}
	}
	if artifact.EvidenceSummary != "" {
		builder.WriteString("\n\n上轮依据：")
		builder.WriteString(artifact.EvidenceSummary)
	}
	reply, _, err := chat.Generate(context.Background(), systemPrompt, []llm.Message{{
		Role:    "user",
		Content: builder.String(),
	}})
	if err != nil {
		return ""
	}
	return strings.TrimSpace(reply)
}

func fallbackFollowupArtifactReply(userMessage string, artifact followupArtifact) string {
	var builder strings.Builder
	builder.WriteString("按上一轮已经完成的解读继续看，")
	if artifact.DirectAnswer != "" {
		builder.WriteString(artifact.DirectAnswer)
	} else {
		builder.WriteString(artifact.Summary)
	}
	if len(artifact.KeyPoints) > 0 {
		builder.WriteString("\n\n上轮关键点是：")
		builder.WriteString(strings.Join(artifact.KeyPoints, "；"))
	}
	if artifact.EvidenceSummary != "" {
		builder.WriteString("\n\n就你这次追问而言，上一轮能直接支撑的依据是：")
		builder.WriteString(artifact.EvidenceSummary)
	}
	if strings.TrimSpace(userMessage) != "" {
		builder.WriteString("\n\n如果只围绕你这次问的“")
		builder.WriteString(strings.TrimSpace(userMessage))
		builder.WriteString("”，我先按上一轮已给出的解读接着说明；若你要我改判新的重点，再单独重看。")
	}
	return strings.TrimSpace(builder.String())
}

func describeFollowupArtifact(artifact followupArtifact) string {
	if direct := strings.TrimSpace(artifact.DirectAnswer); direct != "" {
		return direct
	}
	if summary := strings.TrimSpace(artifact.Summary); summary != "" {
		return summary
	}
	return fmt.Sprintf("%s followup artifact", artifact.Domain)
}
