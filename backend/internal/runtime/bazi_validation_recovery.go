package runtime

import "strings"

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
		"受限":  "受限路线",
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
		"吉中带阻":     "吉中有阻",
		"吉中有阻":     "吉中有阻",
		"机会伴随变动":   "机会伴随强变动",
		"机会伴随大波动":  "机会伴随强变动",
		"机会伴随强变动": "机会伴随强变动",
	})
	return in
}

func recoverStaticSynthesis(chartState baziCharterState, candidate baziStaticSynthesis, cause error) baziStaticSynthesis {
	candidate = normalizeStaticSynthesis(candidate)
	if strings.TrimSpace(candidate.MainAxis) == "" {
		candidate.MainAxis = "当前主轴有可参考的结构依据，但限制仍在，宜保守落判。"
	}
	if strings.TrimSpace(candidate.ClaimStrength) == "" {
		candidate.ClaimStrength = "保守判断"
	}
	if strings.TrimSpace(candidate.SupportLevel) == "" {
		candidate.SupportLevel = "有气"
	}
	candidate.LimitationLevel = "明显"
	candidate.WordingCap = "保守"
	candidate.AxisLevel = "方向成立"
	candidate.EffectOnTiaohou = fallbackNormalizedValue(candidate.EffectOnTiaohou, "中性", []string{"支持", "中性", "冲突"})
	candidate.EffectOnCoreDisease = fallbackNormalizedValue(candidate.EffectOnCoreDisease, "中性", []string{"缓解", "中性", "放大"})
	candidate.EffectOnJiShenDirection = fallbackNormalizedValue(candidate.EffectOnJiShenDirection, "中性", []string{"缓解", "中性", "放大"})
	candidate.AxisCeiling = "受限路线"
	candidate.ConsistencyFlags = []string{"方向成立但力度受限"}
	if len(candidate.ConflictReasons) == 0 {
		candidate.ConflictReasons = []string{"现有结构与限制并存，本轮已按保守口径收束为受限路线。"}
	}
	candidate.PatternBasis = firstNonEmptyTrim(
		candidate.PatternBasis,
		"现有命盘与证据已经能支撑基本判断，但限制面同样明显，因此保留主轴依据、不再拔高。",
	)
	candidate.PatternOutcome = "方向可以参考，但力度受限，不宜拔高。"
	candidate.CounterEvidence = firstNonEmptyTrim(
		candidate.CounterEvidence,
		"本轮结论按保守口径处理，重点是把限制与风险一并保留下来。",
	)
	candidate.AxisConsistency = "当前结论已自动收束为受限路线，只保留可成立面，不再越级拔高。"
	candidate.TiaohouConstraint = firstNonEmptyTrim(candidate.TiaohouConstraint, "调候与限制仍需一并考虑。")
	candidate.TiaohouAnchor = firstNonEmptyTrim(candidate.TiaohouAnchor, "本轮仍按既有时令与结构事实保守落判。")
	candidate.StrengthBalance = firstNonEmptyTrim(candidate.StrengthBalance, "整体受力仍需保守判断。")
	candidate.PatternAndQingZhuo = firstNonEmptyTrim(candidate.PatternAndQingZhuo, "结构可见，但清浊与限制仍需并看。")
	candidate.QiShiOrCongHua = firstNonEmptyTrim(candidate.QiShiOrCongHua, "本轮仍按常规格局理解，不另起极端从化判断。")
	candidate.TierJudgment = firstNonEmptyTrim(candidate.TierJudgment, "中等")
	candidate.TierBasis = "本轮结果已按保守合同收束，限制仍在，因此层次不宜拔高。"
	candidate.ReasoningSummary = "现有结构有可成立面，但限制仍在，因此本轮按保守口径收束为受限路线。"
	candidate.ReasoningSteps = ensureRecoveredStaticSteps(candidate.ReasoningSteps, cause)
	return candidate
}

func recoverDynamicSynthesis(state baziCharterState, candidate baziDynamicSynthesis, cause error) baziDynamicSynthesis {
	candidate = normalizeDynamicSynthesis(candidate)
	if strings.TrimSpace(candidate.CurrentTrend) == "" || containsAnyText([]string{candidate.CurrentTrend}, []string{"一路顺", "明显起飞", "全面起飞"}) {
		candidate.CurrentTrend = "当前更像机会与波动并存，宜边走边看。"
	}
	if strings.TrimSpace(candidate.ClaimStrength) == "" {
		candidate.ClaimStrength = "倾向成立"
	}
	if strings.TrimSpace(candidate.SupportLevel) == "" {
		candidate.SupportLevel = "有气"
	}
	candidate.LimitationLevel = "明显"
	candidate.WordingCap = "中性"
	if len(candidate.ConsistencyFlags) == 0 {
		candidate.ConsistencyFlags = []string{"机会伴随强变动"}
	}
	candidate.WindowLevel = fallbackNormalizedValue(candidate.WindowLevel, "窗口年", []string{"窗口年", "扰动年", "转折年", "承压年"})
	if len(candidate.DayunPath) == 0 {
		candidate.DayunPath = []string{"当前阶段仍有限制，宜按吉中有阻来理解。"}
	}
	if strings.TrimSpace(candidate.LiunianFocus) == "" || containsAnyText([]string{candidate.LiunianFocus}, []string{"一飞冲天", "关键翻身", "彻底起势"}) {
		candidate.LiunianFocus = "这一年有可把握的窗口，但起伏并存，不宜激进。"
	}
	candidate.ReasoningSummary = "岁运机会确实存在，但波动与限制并存，因此本轮按保守口径理解。"
	candidate.ReasoningSteps = ensureRecoveredDynamicSteps(candidate.ReasoningSteps, cause)

	if containsString(candidate.ConsistencyFlags, "吉中有阻") && !containsAnyText(candidate.DayunPath, []string{"吉中有阻", "有阻", "限制"}) {
		candidate.DayunPath[0] = "当前阶段仍属吉中有阻，能推进，但不能按纯顺路理解。"
	}
	if containsString(candidate.ConsistencyFlags, "机会伴随强变动") &&
		!containsAnyText([]string{candidate.CurrentTrend, candidate.LiunianFocus, candidate.ReasoningSummary}, []string{"机会伴随强变动", "吉中带险", "变动", "起伏", "不宜激进"}) {
		candidate.LiunianFocus = "这一年有机会，但变动明显，宜边走边看，不宜激进。"
	}
	if isDynamicDirectionConflict(cause) {
		candidate.ConsistencyFlags = appendUniqueFlag(candidate.ConsistencyFlags, "吉中有阻")
		candidate.CurrentTrend = "当前大运并非纯顺，更像承托与限制并存，宜按吉中有阻理解。"
		candidate.DayunPath[0] = "当前大运有承托面，但限制同步存在，仍需按吉中有阻保守落判。"
		candidate.LiunianFocus = "这一年仍有可推进窗口，但起伏和内耗会并存，不宜按纯顺路押注。"
		candidate.ReasoningSummary = "当前大运有承托面，但限制没有退出，因此本轮只按吉中有阻理解，不把动态窗口写成纯顺兑现。"
	}
	if strings.TrimSpace(state.StaticSynthesis.AxisCeiling) == "结构信号" || strings.TrimSpace(state.StaticSynthesis.AxisCeiling) == "受限路线" {
		candidate.CurrentTrend = "动态只在既有受限主轴内承托，不代表结构层级已经改写。"
	}
	return candidate
}

func isDynamicDirectionConflict(cause error) bool {
	if cause == nil {
		return false
	}
	msg := strings.TrimSpace(cause.Error())
	return strings.Contains(msg, "current dayun path contradicts current trend direction") ||
		strings.Contains(msg, "dynamic current trend conflicts with mixed dayun path")
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

func fallbackNormalizedValue(value, fallback string, allowed []string) string {
	value = strings.TrimSpace(value)
	if containsString(allowed, value) {
		return value
	}
	return fallback
}

func ensureRecoveredStaticSteps(steps []string, cause error) []string {
	steps = filterNonEmpty(steps)
	if len(steps) >= 2 {
		return steps
	}
	_ = cause
	return []string{
		"先保留已经成立的结构依据，再把结论收束到受限路线，避免越级拔高。",
		"再把调候、病点和忌神方向带回限制说明中，确保结论不把风险写丢。",
	}
}

func ensureRecoveredDynamicSteps(steps []string, cause error) []string {
	steps = filterNonEmpty(steps)
	if len(steps) >= 1 {
		return steps
	}
	_ = cause
	return []string{
		"先保留可成立的动态机会，再把波动与限制一并写回，避免把窗口年写成纯顺年。",
	}
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
