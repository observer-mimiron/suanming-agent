package policy

import (
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/state"
)

const guidanceFallbackRetryLimit = 2

// GuidanceReducerInput 是 guidance reducer 的最小输入。
type GuidanceReducerInput struct {
	Current *state.GuidanceState
	Message string
	Profile map[string]any
}

// ReduceGuidance 根据当前 guidance、用户消息和资料补丁计算下一状态。
func ReduceGuidance(input GuidanceReducerInput) *state.GuidanceState {
	next := cloneGuidanceState(input.Current)
	if next == nil {
		return nil
	}

	if topic := detectChosenTopic(input.Message); topic != "" {
		next.ChosenTopic = topic
	}

	return reduceGuidanceMessageOnly(next, input)
}

func reduceGuidanceMessageOnly(next *state.GuidanceState, input GuidanceReducerInput) *state.GuidanceState {
	// Topic change is progress — don't count as retry
	prevTopic := ""
	if input.Current != nil {
		prevTopic = input.Current.ChosenTopic
	}
	var effective bool
	if prevTopic != "" && next.ChosenTopic != "" && prevTopic != next.ChosenTopic {
		effective = true
	}

	switch next.DirectiveKind {
	case "offer_consult":
		if isGuidanceAcceptance(input.Message) {
			next.DirectiveKind = "choose_topic"
			next.PendingSlot = ""
			next.RetryCount = 0
			effective = true
		}
	case "choose_topic":
		if next.ChosenTopic != "" {
			if pending := nextMissingGuidanceSlot(input.Profile); pending != "" {
				next.DirectiveKind = "collect_slot"
				next.PendingSlot = pending
				next.RetryCount = 0
				effective = true
			}
		}
	case "collect_slot":
		if slotFilled(next.PendingSlot, input.Profile) {
			oldSlot := next.PendingSlot
			next.PendingSlot = nextMissingGuidanceSlot(input.Profile)
			next.RetryCount = 0
			effective = true
			if next.PendingSlot == "" && oldSlot != "" {
				return nil
			}
		}
	case "guided_fallback":
		if isGuidanceAcceptance(input.Message) {
			return nil
		}
	}

	if !effective {
		next.RetryCount++
		if next.RetryCount >= guidanceFallbackRetryLimit {
			next.DirectiveKind = "guided_fallback"
			next.PendingSlot = ""
		}
	}

	return normalizeGuidanceState(next)
}

func nextMissingGuidanceSlot(profile map[string]any) string {
	if !hasProfileField(profile, "year") || !hasProfileField(profile, "month") || !hasProfileField(profile, "day") {
		return "birth_date"
	}
	if !hasProfileField(profile, "hour") {
		return "birth_time"
	}
	if !hasProfileField(profile, "gender") {
		return "gender"
	}
	if !hasProfileField(profile, "birthplace") {
		return "birthplace"
	}
	return ""
}

func slotFilled(slot string, profile map[string]any) bool {
	switch slot {
	case "birth_date":
		return hasProfileField(profile, "year") && hasProfileField(profile, "month") && hasProfileField(profile, "day")
	case "birth_time":
		return hasProfileField(profile, "hour")
	case "gender":
		return hasProfileField(profile, "gender")
	case "birthplace":
		return hasProfileField(profile, "birthplace")
	default:
		return false
	}
}

func hasProfileField(profile map[string]any, key string) bool {
	if len(profile) == 0 {
		return false
	}
	value, ok := profile[key]
	if !ok || value == nil {
		return false
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	return true
}

func detectChosenTopic(message string) string {
	trimmed := strings.TrimSpace(message)
	type topicCandidate struct {
		topic    string
		keywords []string
	}
	candidates := []topicCandidate{
		{topic: "事业", keywords: []string{"事业", "工作"}},
		{topic: "感情", keywords: []string{"感情", "爱情", "婚姻"}},
		{topic: "财运", keywords: []string{"财运", "财富", "钱"}},
		{topic: "健康", keywords: []string{"健康", "身体"}},
		{topic: "流年", keywords: []string{"流年", "今年"}},
	}

	bestTopic := ""
	bestIndex := -1
	for _, candidate := range candidates {
		for _, keyword := range candidate.keywords {
			if idx := strings.LastIndex(trimmed, keyword); idx > bestIndex {
				bestIndex = idx
				bestTopic = candidate.topic
			}
		}
	}
	return bestTopic
}

func isGuidanceAcceptance(message string) bool {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return false
	}
	accepts := []string{"行", "可以", "看看", "好", "好的", "行，那", "可以，那", "好，那"}
	for _, accept := range accepts {
		if trimmed == accept ||
			strings.HasPrefix(trimmed, accept+"，") ||
			strings.HasPrefix(trimmed, accept+"。") ||
			strings.Contains(trimmed, accept+"看看") {
			return true
		}
	}
	return strings.Contains(trimmed, "看看")
}

func cloneGuidanceState(current *state.GuidanceState) *state.GuidanceState {
	if current == nil {
		return nil
	}
	return &state.GuidanceState{
		DirectiveKind: current.DirectiveKind,
		ChosenTopic:   current.ChosenTopic,
		PendingSlot:   current.PendingSlot,
		RetryCount:    current.RetryCount,
	}
}

func normalizeGuidanceState(next *state.GuidanceState) *state.GuidanceState {
	if next == nil {
		return nil
	}
	if next.DirectiveKind == "" && next.ChosenTopic == "" && next.PendingSlot == "" && next.RetryCount == 0 {
		return nil
	}
	return next
}
