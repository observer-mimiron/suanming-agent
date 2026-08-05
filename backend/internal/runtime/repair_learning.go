// Package runtime 包含 Manager 拥有的 Repair Harness 学习提示。
//
// 本文件只提供经代码审查固化的短提示选择；不读取线上 trace，
// 不持久化用户输入，也不把历史失败自动写回模型 prompt。
package runtime

import "strings"

const maxRepairLearningHintsPerField = 3

// RepairLearningHint 是 repair feedback 可注入的短提示单元。
// 它按 domain/stage/class/field 精确匹配，避免无关经验污染当前修复。
type RepairLearningHint struct {
	Domain      string
	Stage       string
	Class       RepairClass
	Field       string
	Pattern     string
	Instruction string
	BadExample  string
	GoodExample string
	AppliesWhen []string
}

var repairLearningHints = []RepairLearningHint{
	{
		Domain:      "bazi",
		Stage:       "static_projection",
		Class:       RepairProjectionMismatch,
		Field:       "static.tiaohou_anchor",
		Pattern:     "调候证据已覆盖，但 canonical.tiaohou.verdict 没有明确裁断词。",
		Instruction: "调候证据覆盖时必须给出明确裁断词，如调候不足、调候受限、调候得力、调候受损。",
		BadExample:  "调候上喜水润局，但水星不透，调候力度受限。",
		GoodExample: "调候受限：秋月戊土需火暖、水润；原局水不透，调候以水润为要但力度不足。",
		AppliesWhen: []string{"covered_topics 包含 tiaohou", "static.tiaohou_anchor 缺少明确裁断"},
	},
	{
		Domain:      "bazi",
		Stage:       "static_projection",
		Class:       RepairProjectionMismatch,
		Field:       "static.tiaohou_anchor",
		Pattern:     "只描述季节环境，没有说明调候力度或成败边界。",
		Instruction: "把季节事实收束成一个短 verdict，再把限制放入 boundary。",
		BadExample:  "本轮只确认季节环境与调候边界。",
		GoodExample: "调候不足：秋燥土重，需水润局；原局水不透，调候只能作受限判断。",
		AppliesWhen: []string{"verdict 只有环境描述", "boundary 可承接限制"},
	},
	{
		Domain:      "bazi",
		Stage:       "static_projection",
		Class:       RepairProjectionMismatch,
		Field:       "static.tiaohou_anchor",
		Pattern:     "调候 verdict 回避裁断，导致 legacy 投影不能形成锚点。",
		Instruction: "不得改四柱或主轴，只补 canonical.tiaohou.verdict 的裁断短句。",
		BadExample:  "调候需结合后文再看。",
		GoodExample: "调候受损：燥土当令而水不透，润燥力量不足。",
		AppliesWhen: []string{"只允许修 canonical.tiaohou", "不能新增现实应事"},
	},
}

// RepairLearningHintsFor 返回匹配失败字段的固化短提示，单字段最多三条。
func RepairLearningHintsFor(failure RepairFailure) []RepairLearningHint {
	hints := make([]RepairLearningHint, 0, maxRepairLearningHintsPerField)
	for _, hint := range repairLearningHints {
		if !repairLearningHintMatches(hint, failure) {
			continue
		}
		hints = append(hints, hint)
		if len(hints) >= maxRepairLearningHintsPerField {
			break
		}
	}
	return hints
}

// RepairLearningHintCount 返回 feedback 中实际注入的 hint 数量。
func RepairLearningHintCount(feedback map[string]any) int {
	if len(feedback) == 0 {
		return 0
	}
	switch hints := feedback["learning_hints"].(type) {
	case []map[string]string:
		return len(hints)
	case []RepairLearningHint:
		return len(hints)
	case []string:
		return len(hints)
	case []any:
		return len(hints)
	default:
		return 0
	}
}

// repairLearningHintMatches 执行 domain/stage/class/field 精确匹配。
func repairLearningHintMatches(hint RepairLearningHint, failure RepairFailure) bool {
	return strings.TrimSpace(hint.Domain) == strings.TrimSpace(failure.Domain) &&
		strings.TrimSpace(hint.Stage) == strings.TrimSpace(failure.Stage) &&
		hint.Class == failure.Class &&
		strings.TrimSpace(hint.Field) == strings.TrimSpace(failure.Field)
}

// repairLearningHintFeedback 把内部 hint 收束成可给模型看的短字段。
func repairLearningHintFeedback(hints []RepairLearningHint) []map[string]string {
	out := make([]map[string]string, 0, len(hints))
	for _, hint := range hints {
		item := map[string]string{}
		if hint.Pattern != "" {
			item["pattern"] = hint.Pattern
		}
		if hint.Instruction != "" {
			item["instruction"] = hint.Instruction
		}
		if hint.BadExample != "" {
			item["bad_example"] = hint.BadExample
		}
		if hint.GoodExample != "" {
			item["good_example"] = hint.GoodExample
		}
		if len(item) > 0 {
			out = append(out, item)
		}
	}
	return out
}
