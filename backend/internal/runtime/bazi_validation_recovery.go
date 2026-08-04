// This file belongs to the manager-owned runtime layer.
// It owns BaZi validation recovery for this package.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"fmt"
	"strings"
)

func normalizeStaticSynthesis(in baziStaticSynthesis) baziStaticSynthesis {
	in.ClaimStrength = normalizeByAlias(in.ClaimStrength, map[string]string{
		"保守":   "保守判断",
		"保守判断": "保守判断",
		"倾向":   "倾向成立",
		"倾向成立": "倾向成立",
		"明确":   "明确成立",
		"明确成立": "明确成立",
		"封顶":   "封顶判断",
		"封顶判断": "封顶判断",
	})
	in.SupportLevel = normalizeByAlias(in.SupportLevel, map[string]string{
		"有":   "出现",
		"出现":  "出现",
		"有根气": "有根",
		"有根":  "有根",
		"有气":  "有气",
		"有力":  "得力",
		"得力":  "得力",
		"成局":  "成势",
		"成势":  "成势",
	})
	in.LimitationLevel = normalizeByAlias(in.LimitationLevel, map[string]string{
		"轻":    "轻微",
		"轻微":   "轻微",
		"明显":   "明显",
		"较明显":  "明显",
		"核心问题": "核心硬伤",
		"核心硬伤": "核心硬伤",
	})
	in.WordingCap = normalizeByAlias(in.WordingCap, map[string]string{
		"克制": "保守",
		"保守": "保守",
		"中性": "中性",
		"明确": "明确",
		"封顶": "封顶",
	})
	in.AxisLevel = normalizeByAlias(in.AxisLevel, map[string]string{
		"结构存在": "结构可见",
		"结构可见": "结构可见",
		"方向可立": "方向成立",
		"方向成立": "方向成立",
		"主轴可立": "主轴成立",
		"主轴成立": "主轴成立",
		"可抬升":  "可以拔高",
		"可以拔高": "可以拔高",
	})
	in.EffectOnTiaohou = normalizeByAlias(in.EffectOnTiaohou, map[string]string{
		"有利": "支持",
		"支持": "支持",
		"中性": "中性",
		"不利": "冲突",
		"冲突": "冲突",
	})
	in.EffectOnCoreDisease = normalizeByAlias(in.EffectOnCoreDisease, map[string]string{
		"减轻": "缓解",
		"缓和": "缓解",
		"缓解": "缓解",
		"中性": "中性",
		"加重": "放大",
		"放大": "放大",
	})
	in.EffectOnJiShenDirection = normalizeByAlias(in.EffectOnJiShenDirection, map[string]string{
		"减轻": "缓解",
		"缓和": "缓解",
		"削弱": "缓解",
		"抑制": "缓解",
		"缓解": "缓解",
		"中性": "中性",
		"加重": "放大",
		"放大": "放大",
	})
	in.AxisCeiling = normalizeByAlias(in.AxisCeiling, map[string]string{
		"结构存在": "结构信号",
		"结构信号": "结构信号",
		"受限":   "受限路线",
		"受限路线": "受限路线",
		"主轴可立": "可作主轴",
		"可作主轴": "可作主轴",
		"可抬升":  "可以拔高",
		"可以拔高": "可以拔高",
	})
	in.ConsistencyFlags = normalizeFlags(in.ConsistencyFlags, map[string]string{
		"方向成立但受限":   "方向成立但力度受限",
		"方向成立但力度受限": "方向成立但力度受限",
	})
	if strings.TrimSpace(in.Strength.Conclusion) == "" {
		in.Strength.Conclusion = strengthConclusionFromText(in.StrengthBalance)
	}
	if strings.TrimSpace(in.Strength.Reasoning) == "" {
		in.Strength.Reasoning = strings.TrimSpace(in.StrengthBalance)
	}
	if strings.TrimSpace(in.Strength.Boundary) == "" {
		in.Strength.Boundary = "扶抑受力只说明日主承受能力；格局取用与调候另行判断。"
	}
	return in
}

func normalizeDynamicSynthesis(in baziDynamicSynthesis) baziDynamicSynthesis {
	in.ClaimStrength = normalizeByAlias(in.ClaimStrength, map[string]string{
		"保守":   "保守判断",
		"保守判断": "保守判断",
		"倾向":   "倾向成立",
		"倾向成立": "倾向成立",
		"明确":   "明确成立",
		"明确成立": "明确成立",
		"封顶":   "封顶判断",
		"封顶判断": "封顶判断",
	})
	in.SupportLevel = normalizeByAlias(in.SupportLevel, map[string]string{
		"有":   "出现",
		"出现":  "出现",
		"有根气": "有根",
		"有根":  "有根",
		"有气":  "有气",
		"有力":  "得力",
		"得力":  "得力",
		"成局":  "成势",
		"成势":  "成势",
		"强":   "得力",
	})
	in.LimitationLevel = normalizeByAlias(in.LimitationLevel, map[string]string{
		"轻":    "轻微",
		"轻微":   "轻微",
		"明显":   "明显",
		"较明显":  "明显",
		"核心问题": "核心硬伤",
		"核心硬伤": "核心硬伤",
	})
	in.WordingCap = normalizeByAlias(in.WordingCap, map[string]string{
		"克制": "保守",
		"保守": "保守",
		"中性": "中性",
		"明确": "明确",
		"封顶": "封顶",
	})
	in.WindowLevel = normalizeByAlias(in.WindowLevel, map[string]string{
		"机会年": "窗口年",
		"窗口年": "窗口年",
		"波动年": "扰动年",
		"扰动年": "扰动年",
		"转机年": "转折年",
		"转折年": "转折年",
		"压力年": "承压年",
		"承压年": "承压年",
	})
	in.ConsistencyFlags = normalizeFlags(in.ConsistencyFlags, map[string]string{
		"吉中带阻":        dynamicFlagMixedConstraint,
		"吉中有阻":        dynamicFlagMixedConstraint,
		"机会伴随变动":      dynamicFlagVolatileOpportunity,
		"机会伴随大波动":     dynamicFlagVolatileOpportunity,
		"机会伴随强变动":     dynamicFlagVolatileOpportunity,
		"限制仍在":        dynamicFlagLimitationRemains,
		"仍有限制":        dynamicFlagLimitationRemains,
		"仅作结构观察":      dynamicFlagStructureOnly,
		"承接与扰动并存":     dynamicFlagStructureOnly,
		"关系触发会增加过程反复": dynamicFlagStructureOnly,
	})
	if len(in.DayunPath) == 0 && len(in.DayunJudgments) > 0 {
		in.DayunPath = renderDayunJudgmentLines(in.DayunJudgments)
	}
	return in
}

const (
	dynamicFlagMixedConstraint     = "吉中有阻"
	dynamicFlagVolatileOpportunity = "机会伴随强变动"
	dynamicFlagLimitationRemains   = "限制仍在"
	dynamicFlagStructureOnly       = "仅作结构观察"
)

var allowedDynamicConsistencyFlags = []string{
	dynamicFlagMixedConstraint,
	dynamicFlagVolatileOpportunity,
	dynamicFlagLimitationRemains,
	dynamicFlagStructureOnly,
}

func strengthConclusionFromText(text string) string {
	text = strings.TrimSpace(text)
	for _, conclusion := range []string{"中和偏强", "中和偏弱", "中和附近", "日主偏强", "日主偏弱", "身强", "身弱"} {
		if strings.Contains(text, conclusion) {
			return strings.TrimPrefix(conclusion, "日主")
		}
	}
	return ""
}

func recoverStaticSynthesis(chartState baziCharterState, candidate baziStaticSynthesis, cause error) baziStaticSynthesis {
	_ = normalizeStaticSynthesis(candidate)
	recovered := buildFactsOnlyStaticSynthesis(chartState.Input, recoveryReasonText(cause, "静态综合未通过；本轮只展示排盘和工具事实。"))
	recovered.FieldAudit = append(recovered.FieldAudit, "static_facts_only_degraded")
	return recovered
}

func strengthEvidenceSummary(yongshen map[string]any) string {
	strength := strings.TrimSpace(stringValue(yongshen["strength"]))
	evidence, _ := yongshen["strength_evidence"].(map[string]any)
	support := intValue(evidence["support_score"])
	pressure := intValue(evidence["pressure_score"])
	if strength == "" {
		return "整体受力仍需保守判断。"
	}
	return fmt.Sprintf("日主%s；扶身证据 %d、泄耗克证据 %d。", strength, support, pressure)
}

func recoverDynamicSynthesis(state baziCharterState, candidate baziDynamicSynthesis, cause error) baziDynamicSynthesis {
	_ = normalizeDynamicSynthesis(candidate)
	recovered := buildFactsOnlyDynamicSynthesis(state.Input, state.StaticSynthesis, recoveryReasonText(cause, "动态裁断受授权边界限制；本轮只展示大运和流年事实。"))
	recovered.FieldAudit = append(recovered.FieldAudit, "dynamic_facts_only_degraded")
	return recovered
}

func recoveryReasonText(cause error, fallback string) string {
	if cause == nil || strings.TrimSpace(cause.Error()) == "" {
		return fallback
	}
	return cause.Error()
}

// sanitizeDynamicPresentationBoundaries only normalizes enum aliases during the
// migration. Unsupported outcomes are validation violations, not text rewrites.
func sanitizeDynamicPresentationBoundaries(candidate baziDynamicSynthesis) baziDynamicSynthesis {
	if flags, flagsChanged := sanitizeDynamicConsistencyFlags(candidate.ConsistencyFlags); flagsChanged {
		candidate.ConsistencyFlags = flags
		candidate.FieldAudit = append(candidate.FieldAudit, "dynamic_consistency_flags")
	}
	return candidate
}

func sanitizeDynamicConsistencyFlags(flags []string) ([]string, bool) {
	if len(flags) == 0 {
		return nil, false
	}
	changed := false
	out := make([]string, 0, len(flags))
	for _, flag := range normalizeDynamicSynthesis(baziDynamicSynthesis{ConsistencyFlags: flags}).ConsistencyFlags {
		if flag == "" {
			continue
		}
		if !containsString(allowedDynamicConsistencyFlags, flag) || hasDynamicHardBoundary(flag) {
			flag = dynamicFlagStructureOnly
			changed = true
		}
		if !containsString(out, flag) {
			out = append(out, flag)
		}
	}
	if len(out) != len(filterNonEmpty(flags)) {
		changed = true
	}
	return out, changed
}

func hasDynamicHardBoundary(text string) bool {
	return containsUnsupportedConcreteOutcome(text) || containsAnyText([]string{text}, []string{"投资", "投资建议"})
}

func appendUniqueFlag(flags []string, flag string) []string {
	flag = strings.TrimSpace(flag)
	if flag == "" || containsString(flags, flag) {
		return flags
	}
	return append(flags, flag)
}

func normalizeByAlias(value string, aliases map[string]string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if canonical, ok := aliases[value]; ok {
		return canonical
	}
	return value
}

func normalizeFlags(flags []string, aliases map[string]string) []string {
	if len(flags) == 0 {
		return nil
	}
	out := make([]string, 0, len(flags))
	for _, flag := range flags {
		flag = normalizeByAlias(flag, aliases)
		if flag == "" || containsString(out, flag) {
			continue
		}
		out = append(out, flag)
	}
	return out
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
