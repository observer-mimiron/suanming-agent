package repair

import "sort"

// TraceEvent is the safe repair metadata projected into a trace.
type TraceEvent struct {
	Failure           Failure
	Attempt           int
	MaxAttempts       int
	Action            Action
	FeedbackKeys      []string
	LearningHintCount int
	Exhausted         bool
	FinalAction       Action
}

// TraceAttrs returns only compact repair metadata and never candidate or feedback values.
func TraceAttrs(event TraceEvent) map[string]any {
	attrs := map[string]any{
		"repair.attempt":             event.Attempt,
		"repair.max_attempts":        event.MaxAttempts,
		"repair.learning_hint_count": event.LearningHintCount,
		"repair.exhausted":           event.Exhausted,
	}
	if event.Failure.Domain != "" {
		attrs["repair.domain"] = event.Failure.Domain
	}
	if event.Failure.Stage != "" {
		attrs["repair.stage"] = event.Failure.Stage
	}
	if event.Failure.Class != "" {
		attrs["repair.class"] = string(event.Failure.Class)
	}
	if event.Failure.Field != "" {
		attrs["repair.field"] = event.Failure.Field
	}
	if event.Failure.Origin != "" {
		attrs["repair.failure_origin"] = string(event.Failure.Origin)
	}
	if event.Action != "" {
		attrs["repair.action"] = string(event.Action)
	}
	if len(event.FeedbackKeys) > 0 {
		attrs["repair.feedback_keys"] = append([]string(nil), event.FeedbackKeys...)
	}
	if event.FinalAction != "" {
		attrs["repair.final_action"] = string(event.FinalAction)
	}
	return attrs
}

// FeedbackKeys returns stable field names without retaining feedback values.
func FeedbackKeys(feedback map[string]any) []string {
	keys := make([]string, 0, len(feedback))
	for key := range feedback {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
