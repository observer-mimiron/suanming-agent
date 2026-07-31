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

	if target != "" {
		st.SetActiveSubject(target)
	}

	if route.PrimaryDomain == "qimen" && route.TaskIntent != "fortune_followup" {
		st.StartCase("qimen", firstNonEmpty(strings.TrimSpace(route.Slots.QuestionText), strings.TrimSpace(message)), true)
	}
	return route
}

func isAmbiguousSubjectReference(value string) bool {
	switch strings.TrimSpace(value) {
	case "他", "她", "它", "对方", "那个人", "那个":
		return true
	default:
		return false
	}
}
