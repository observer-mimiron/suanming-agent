// Package runtime 包含 Manager 拥有的 Repair Harness trace 投影。
//
// 本文件只输出安全的 repair.* 短字段；不得写入完整 prompt、
// 完整 trace、候选文本、feedback value 或用户隐私。
package runtime

import "sort"

// RepairTraceEvent 是 repair trace 投影的安全输入。
type RepairTraceEvent struct {
	Failure           RepairFailure
	Attempt           int
	MaxAttempts       int
	Action            RepairAction
	Feedback          map[string]any
	LearningHintCount int
	Exhausted         bool
	FinalAction       RepairAction
}

// RepairTraceAttrs 只返回允许写入 trace 的 repair.* 短字段。
func RepairTraceAttrs(event RepairTraceEvent) map[string]any {
	failure := event.Failure
	attrs := map[string]any{}
	if failure.Domain != "" {
		attrs["repair.domain"] = failure.Domain
	}
	if failure.Stage != "" {
		attrs["repair.stage"] = failure.Stage
	}
	if failure.Class != "" {
		attrs["repair.class"] = string(failure.Class)
	}
	if failure.Field != "" {
		attrs["repair.field"] = failure.Field
	}
	attrs["repair.attempt"] = event.Attempt
	attrs["repair.max_attempts"] = event.MaxAttempts
	if event.Action != "" {
		attrs["repair.action"] = string(event.Action)
	}
	if keys := RepairFeedbackKeys(event.Feedback); len(keys) > 0 {
		attrs["repair.feedback_keys"] = keys
	}
	hintCount := event.LearningHintCount
	if hintCount == 0 {
		hintCount = RepairLearningHintCount(event.Feedback)
	}
	attrs["repair.learning_hint_count"] = hintCount
	attrs["repair.exhausted"] = event.Exhausted
	if event.FinalAction != "" {
		attrs["repair.final_action"] = string(event.FinalAction)
	}
	return attrs
}

// RepairFeedbackKeys 返回排序后的 feedback key，不保留任何 value。
func RepairFeedbackKeys(feedback map[string]any) []string {
	if len(feedback) == 0 {
		return nil
	}
	keys := make([]string, 0, len(feedback))
	for key := range feedback {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
