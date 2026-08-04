// Package runtime contains the manager-owned execution flow.
//
// This file binds natural-language subject references to deterministic session
// subjects and cases before the Manager builds artifact requirements.
package runtime

import (
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// resolveArtifactFocus binds natural-language object references to deterministic
// session identities before the Manager builds an execution plan.
func resolveArtifactFocus(st *state.SessionState, route policy.ApprovedRoute, message string) policy.ApprovedRoute {
	if st == nil {
		return route
	}
	target := strings.TrimSpace(route.Slots.TargetSubject)
	if isAmbiguousSubjectReference(target) {
		if len(st.Subjects) > 1 {
			route.NeedsClarification = true
			route.ClarificationQuestion = "你说的是哪一位？请说明是自己、孩子、父亲等。"
			return route
		}
		target = ""
	}

	if target != "" && shouldSelectSubject(st, target) {
		st.SetActiveSubject(target)
	} else if target != "" {
		if strings.TrimSpace(route.Slots.QuestionText) == "" {
			route.Slots.QuestionText = target
		}
		route.Slots.TargetSubject = ""
		resetTopicOnlyActiveSubject(st)
	}

	if route.PrimaryDomain == "qimen" && route.TaskIntent != "fortune_followup" {
		st.StartCase("qimen", firstNonEmpty(strings.TrimSpace(route.Slots.QuestionText), strings.TrimSpace(message)), true)
	}
	return route
}

// shouldSelectSubject allows only explicit people or existing non-topic labels to
// change ActiveFocus. This keeps topics like 婚姻 from becoming asset owners.
func shouldSelectSubject(st *state.SessionState, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" || isConsultationTopic(target) {
		return false
	}
	for _, subject := range st.Subjects {
		if strings.TrimSpace(subject.Display) == target {
			return true
		}
	}
	return isExplicitSubjectLabel(target)
}

// resetTopicOnlyActiveSubject repairs the active focus when an earlier turn
// polluted the subject list with a topic label such as 婚姻.
func resetTopicOnlyActiveSubject(st *state.SessionState) {
	if st == nil {
		return
	}
	current := st.ActiveSubject()
	if !isConsultationTopic(current.Display) {
		return
	}
	for _, subject := range st.Subjects {
		if strings.TrimSpace(subject.Display) == "自己" {
			st.SetActiveSubject("自己")
			return
		}
	}
}

// isExplicitSubjectLabel recognizes labels that denote consultation objects
// rather than question topics.
func isExplicitSubjectLabel(value string) bool {
	switch strings.TrimSpace(value) {
	case "自己", "我", "本人", "命主", "孩子", "儿子", "女儿", "父亲", "母亲", "爸爸", "妈妈", "丈夫", "妻子", "老公", "老婆", "伴侣":
		return true
	default:
		return false
	}
}

// isConsultationTopic recognizes common reading topics that must remain
// question scope, never asset ownership.
func isConsultationTopic(value string) bool {
	switch strings.TrimSpace(value) {
	case "婚姻", "感情", "姻缘", "恋爱", "事业", "工作", "财运", "财富", "健康", "学业", "考试", "子女", "父母", "本月", "今年", "流年", "大运", "运势":
		return true
	default:
		return false
	}
}

// isAmbiguousSubjectReference recognizes pronouns that require clarification when multiple subjects exist.
func isAmbiguousSubjectReference(value string) bool {
	switch strings.TrimSpace(value) {
	case "他", "她", "它", "对方", "那个人", "那个":
		return true
	default:
		return false
	}
}
