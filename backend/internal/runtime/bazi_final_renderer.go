package runtime

import (
	"fmt"
	"strings"
)

// renderBaziFinalReply 改为由程序直接消费上游结构化结论并渲染最终 markdown。
// 这样最终成文不再依赖自由文本 writer 自行排版，从根上消除标题、加粗结论、
// 编号步骤等展示合同漂移导致的整轮失败。
func renderBaziFinalReply(plan baziAnalysisPlan, state baziCharterState, question string) string {
	switch strings.TrimSpace(plan.WriterTemplate) {
	case "topic":
		return renderTopicTemplate(plan, state, question)
	case "year":
		return renderYearTemplate(state, question)
	default:
		return renderFullTemplate(state)
	}
}

func renderFullTemplate(state baziCharterState) string {
	var b strings.Builder
	writeHeading(&b, "总览结论")
	writeConclusion(&b, buildOverviewConclusion(state))
	writeParagraphs(&b, []string{
		buildOverviewTagline(state),
	})
	writeHighlightBlock(&b,
		"◎ 主轴",
		buildOverviewAxisSummary(state),
	)
	writeHighlightBlock(&b,
		"▲ 限制",
		buildOverviewLimitationSummary(state),
	)
	writeHighlightBlock(&b,
		"◇ 读法",
		"静态先定路，岁运再看应期。",
	)

	writeHeading(&b, "强弱视角")
	writeConclusion(&b, conclusionOrDefault(
		"命局强弱以"+fallbackText(state.StaticSynthesis.StrengthBalance, "扶抑受力尚需保守判断")+"为主。",
		buildStrengthConclusion(state),
	))
	writeBullets(&b, []string{
		"**扶抑受力**：" + fallbackText(state.StaticSynthesis.StrengthBalance, "当前仍以整体受力平衡为主。"),
		"**主轴关系**：" + fallbackText(state.StaticSynthesis.MainAxis, "主轴仍需回到格局层面综合判断。"),
		"**提示**：扶抑与调候是两个维度，不矛盾。",
	})

	writeHeading(&b, "调候视角")
	writeConclusion(&b, conclusionOrDefault(
		"这盘第一先看调候，核心约束在"+fallbackText(state.StaticSynthesis.TiaohouConstraint, "寒暖燥湿的平衡")+"。",
		buildTiaohouConclusion(state),
	))
	writeBullets(&b, []string{
		"**调候锚点**：" + fallbackText(state.StaticSynthesis.TiaohouAnchor, "本轮未给出更细的时令锚点，只能按现有结构保守落判。"),
		"**调候约束**：" + fallbackText(state.StaticSynthesis.TiaohouConstraint, "调候仍是本轮的重要边界。"),
		"**与扶抑关系**：扶抑与调候是两个维度，不矛盾。",
	})

	writeHeading(&b, "格局视角")
	writeConclusion(&b, conclusionOrDefault(
		"命局主轴为"+fallbackText(state.StaticSynthesis.MainAxis, "当前主轴仍需保守判断")+"。",
		buildPatternConclusion(state),
	))
	writeBullets(&b, []string{
		"**本轮总纲**：先以格局定主轴，再以调候定边界，再看扶抑与病药，最后才定命格层次。",
		"**方法提醒**：何知章只作验盘入口；层次仍回到主轴路线、清浊情势、病药救应与福寿风险综合判断。",
	})
	writeSteps(&b, ensureSteps(state.StaticSynthesis.ReasoningSteps, []string{
		"先看月令与透干，判断命局主气落点。",
		"再看关键藏干层级与承接关系，确认主轴是否真能成立。",
		"最后回到病药与限制，判断这条路线能否拔高。",
	}))
	writeBullets(&b, []string{
		"**主证依据**：" + fallbackText(state.StaticSynthesis.PatternBasis, "当前主轴依据以系统上游结构化裁定为准。"),
		"**为何成立**：" + fallbackText(state.StaticSynthesis.PatternOutcome, "方向能够成立，但需要连同限制一起看。"),
		"**为什么不取别路**：" + fallbackText(state.StaticSynthesis.AxisConsistency, "本轮仍以当前主轴为准，不改判为另一条路线。"),
		"**限制在哪里**：" + buildLimitationText(state),
	})
	writeSubheading(&b, "证据矩阵")
	writeMatrixEntry(&b, "格局主轴", []string{
		"**关键信号**：" + fallbackText(state.StaticSynthesis.PatternBasis, "当前主轴依据以上游结构化裁定为准。"),
		"**对主轴的作用**：用来判断主轴是否能立、为何取此不取彼。",
		"**对层次的影响**：" + fallbackText(state.StaticSynthesis.PatternOutcome, "方向成立，但仍需连同限制一起看。"),
	})
	writeMatrixEntry(&b, "调候边界", []string{
		"**关键信号**：" + joinInlineParts(
			fallbackText(state.StaticSynthesis.TiaohouAnchor, "本轮未给出更细的时令锚点。"),
			fallbackText(state.StaticSynthesis.TiaohouConstraint, "调候仍是本轮的重要边界。"),
		),
		"**对主轴的作用**：不另起第二主轴，只负责给主轴划定环境边界。",
		"**对层次的影响**：调候越硬，越会压住层次上限。",
	})
	writeMatrixEntry(&b, "扶抑受力", []string{
		"**关键信号**：" + fallbackText(state.StaticSynthesis.StrengthBalance, "当前仍以整体受力平衡为主。"),
		"**对主轴的作用**：说明这条主轴由谁承接、由谁泄耗。",
		"**对层次的影响**：帮助判断病药是否真能接得住主轴。",
	})
	writeMatrixEntry(&b, "反证与限制", []string{
		"**关键信号**：" + fallbackText(state.StaticSynthesis.CounterEvidence, "当前仍有限制，不宜拔高。"),
		"**对主轴的作用**：提醒为什么不能改取别路，也不能硬拔高。",
		"**对层次的影响**：" + fallbackText(state.StaticSynthesis.TierBasis, "层次判断要服从前面已经裁定的主轴与限制。"),
	})
	writeSubheading(&b, "古法验盘")
	writeParagraphs(&b, []string{
		"这一段是把前面已经裁定的主轴、限制与岁运兑现，投影到古法断语上做验盘，不另起第二套判断。",
	})
	writeBullets(&b, buildClassicalAuditBullets(state))

	writeHeading(&b, "大运验证")
	writeConclusion(&b, conclusionOrDefault(
		"大运节奏以"+fallbackText(state.DynamicSynthesis.CurrentTrend, "静态主轴为主、动态只作背景参考")+"来理解。",
		buildDayunConclusion(state),
	))
	writeBullets(&b, ensureBullets(state.DynamicSynthesis.DayunPath, []string{
		"当前暂无更细的大运展开，本轮以静态主轴为主。",
	}))

	writeHeading(&b, "流年应期")
	writeConclusion(&b, conclusionOrDefault(
		"这一年更像"+fallbackText(renderWindowLevel(state.DynamicSynthesis.WindowLevel), "需要结合现实节奏观察的一年")+"。",
		buildLiunianConclusion(state),
	))
	writeBullets(&b, []string{
		"**年性**：" + fallbackText(renderWindowLevel(state.DynamicSynthesis.WindowLevel), "本轮未指定更细的流年等级。"),
		"**触发点**：" + joinOrDefault(state.DynamicSynthesis.TriggerSignals, "当前重点仍看流年如何触发主轴与限制。"),
		"**应事领域**：" + fallbackText(state.DynamicSynthesis.LiunianFocus, "应事重心仍以用户当前问题与静态主轴交叉理解。"),
		"**限制**：" + buildDynamicConstraintText(state),
	})

	writeHeading(&b, "综合判定")
	writeConclusion(&b, "命格层次以"+fallbackText(state.StaticSynthesis.TierJudgment, "当前结构保守落判")+"为宜。")
	writeBullets(&b, []string{
		"**层次依据**：" + fallbackText(state.StaticSynthesis.TierBasis, "层次判断要服从前面已经裁定的主轴、调候与限制。"),
		"**原局定级**：" + buildOriginalTierText(state),
		"**岁运兑现**：" + buildTierRealizationText(state),
		"**读法提醒**：主轴负责定路，限制负责压上限，岁运负责看兑现，不宜混作一轮机械加减分。",
	})

	writeHeading(&b, "命格总结")
	writeBullets(&b, []string{
		"**最大优点**：" + joinOrDefault(state.StaticSynthesis.Advantages, fallbackText(state.StaticSynthesis.MainAxis, "主轴判断已有可依之处。")),
		"**最大风险**：" + joinOrDefault(state.StaticSynthesis.Risks, buildLimitationText(state)),
		"**用力方向**：顺着用神方向做取舍，先做泄秀暖局之事，再谈放大格局。",
		"**务实建议**：先做确定性更高、节奏更稳的安排，不把一时机会当成整盘改命。",
	})
	return strings.TrimSpace(b.String())
}

func buildOverviewConclusion(state baziCharterState) string {
	return "此命" +
		buildOverviewAxisSummary(state) +
		"，局能立，但" +
		buildOverviewLimitationSummary(state) +
		"，层次以" +
		fallbackText(state.StaticSynthesis.TierJudgment, "当前结构保守落判") +
		"为宜。"
}

func buildOverviewTagline(state baziCharterState) string {
	switch strings.TrimSpace(state.StaticSynthesis.LimitationLevel) {
	case "核心硬伤":
		return "古法提要：先看局能否立，再看病能否救；此局见路，而病重于药。"
	case "明显":
		return "古法提要：先看局能否立，再看病能否救；此局有路，而药力未厚。"
	default:
		return "古法提要：先看局能否立，再看病能否救；此局有路，仍须待运。"
	}
}

func buildOverviewAxisSummary(state baziCharterState) string {
	text := strings.Join(filterNonEmpty([]string{
		state.StaticSynthesis.MainAxis,
		state.StaticSynthesis.PatternBasis,
		state.StaticSynthesis.PatternOutcome,
	}), " ")
	switch {
	case containsAnyText([]string{text}, []string{"月劫", "劫财"}) && containsAnyText([]string{text}, []string{"食神制杀"}):
		return "以月劫格中取食神制杀为主轴"
	case containsAnyText([]string{text}, []string{"食神制杀"}):
		return "以食神制杀为主轴"
	case containsAnyText([]string{text}, []string{"杀印相生", "印化杀"}):
		return "以杀印相生为主轴"
	case containsAnyText([]string{text}, []string{"伤官配印"}):
		return "以伤官配印为主轴"
	case containsAnyText([]string{text}, []string{"弃印就财"}):
		return "以弃印就财为主轴"
	case containsAnyText([]string{text}, []string{"月劫格", "劫财格"}):
		return "以月劫格立局"
	default:
		return "主轴仍按" + shortenClause(state.StaticSynthesis.MainAxis, "当前主轴保守落判") + "来读"
	}
}

func buildOverviewLimitationSummary(state baziCharterState) string {
	text := strings.Join(filterNonEmpty([]string{
		state.StaticSynthesis.TiaohouConstraint,
		state.StaticSynthesis.CounterEvidence,
		state.StaticSynthesis.TierBasis,
	}), " ")
	switch {
	case strings.TrimSpace(state.StaticSynthesis.LimitationLevel) == "核心硬伤":
		return "硬伤仍在"
	case containsAnyText([]string{text}, []string{"调候", "寒", "冷", "暖局", "火暖"}):
		return "火候未足"
	case containsAnyText([]string{text}, []string{"受限", "不足", "难以", "不宜拔高", "不能拔高"}):
		return "条件未足"
	default:
		return "尚有牵制"
	}
}

func renderTopicTemplate(plan baziAnalysisPlan, state baziCharterState, question string) string {
	var b strings.Builder
	topicMode := normalizedTopicMode(plan.TopicMode)
	writeHeading(&b, "直接回答")
	writeConclusion(&b, buildTopicDirectConclusion(plan, state, question))
	if focus := buildTopicFocusAnswer(plan, state); focus != "" {
		writeBullets(&b, []string{
			"**这次追问的关键**：" + focus,
		})
	} else {
		writeParagraphs(&b, []string{
			buildTopicDirectParagraph(state),
		})
	}

	writeHeading(&b, "命盘依据")
	switch topicMode {
	case "explain_term":
		writeConclusion(&b, "这轮追问重点是在解释术语或句子在命盘结构里承担什么作用，不是在重判整盘高低。")
		writeBullets(&b, []string{
			"**结构落点**：" + buildTopicExplainPosition(state),
			"**命盘依据**：" + fallbackText(state.StaticSynthesis.AxisConsistency, fallbackText(state.StaticSynthesis.PatternOutcome, "先看它是在支撑主轴，还是在提醒限制。")),
			"**边界**：" + buildTopicExplainBoundary(state),
		})
	case "timing_reason":
		writeConclusion(&b, "这轮追问关键要看岁运是在承托主轴，还是把原有限制一并放大。")
		writeBullets(&b, []string{
			"**原局底盘**：" + fallbackText(state.StaticSynthesis.MainAxis, "先按既有主轴理解，不把动态波动提前改写成原局升格。"),
			"**岁运机制**：" + buildTierRealizationText(state),
			"**限制**：" + buildDynamicConstraintText(state),
		})
	case "conservative_reason":
		writeConclusion(&b, "这次口径保守，不是因为主轴全无成立面，而是因为限制没有退出。")
		writeBullets(&b, []string{
			"**成立面**：" + fallbackText(state.StaticSynthesis.PatternOutcome, "这盘并非全无成立面，但成立力度仍需保守判断。"),
			"**限制面**：" + buildLimitationText(state),
			"**为何不拔高**：" + fallbackText(state.StaticSynthesis.TierBasis, "层次判断要服从已裁定的主轴与限制。"),
		})
	default:
		writeConclusion(&b, "命盘主轴可以支撑这次判断，但限制不能写丢。")
		writeBullets(&b, []string{
			"**主轴**：" + fallbackText(state.StaticSynthesis.MainAxis, "当前仍以静态主轴为主要依据。"),
			"**为何这样看**：" + fallbackText(state.StaticSynthesis.PatternOutcome, "方向成立，但力度受限。"),
			"**限制**：" + buildTopicConstraintText(state),
		})
	}

	writeHeading(&b, "建议")
	switch topicMode {
	case "explain_term":
		writeConclusion(&b, "建议先确认这句话在命盘里解释的是主轴、限制还是承接，再决定它该占多大分量。")
		writeBullets(&b, []string{
			"不要把一句术语解释直接当成整盘结论；先放回当前主轴与限制里，再看它是在加分还是在提醒边界。",
		})
	case "timing_reason":
		writeConclusion(&b, "建议把动态起伏当成对原局的承托或扰动来看，不要把一时波动直接当成整盘改判。")
		writeBullets(&b, []string{
			"先看原局底盘，再看这步运是在放大机会还是放大限制，避免只盯当下单点起伏。",
		})
	case "conservative_reason":
		writeConclusion(&b, "建议把这次保守口径理解为“成立面和限制面并写”，而不是简单否定。")
		writeBullets(&b, []string{
			"先确认这盘能立在哪里，再确认压上限的限制在哪里，这样更接近真实结构。",
		})
	default:
		writeConclusion(&b, buildTopicAdviceConclusion(state))
		writeBullets(&b, []string{
			"先做自己能掌控的决定，别把节奏拉得过急。",
		})
	}
	return strings.TrimSpace(b.String())
}

func renderYearTemplate(state baziCharterState, question string) string {
	var b strings.Builder
	writeHeading(&b, "年度判断")
	writeConclusion(&b, conclusionOrDefault(
		"这一年更像"+fallbackText(renderWindowLevel(state.DynamicSynthesis.WindowLevel), "需要保守把握的一年")+"。",
		buildLiunianConclusion(state),
	))
	writeParagraphs(&b, []string{
		buildTopicDirectParagraph(state),
	})

	writeHeading(&b, "作用机制")
	writeConclusion(&b, "这一年的起伏，关键看主轴是否被承接，以及限制是否被放大。")
	writeSteps(&b, ensureSteps(state.DynamicSynthesis.ReasoningSteps, []string{
		"先看当前大运对静态主轴是承接还是压制。",
		"再看流年触发点是在放大机会，还是同时把限制引动。",
	}))

	writeHeading(&b, "重点应期")
	writeConclusion(&b, "重点应期要围绕机会与限制并存来把握。")
	writeBullets(&b, []string{
		"**年性**：" + fallbackText(renderWindowLevel(state.DynamicSynthesis.WindowLevel), "当前年性仍需保守落判。"),
		"**触发点**：" + joinOrDefault(state.DynamicSynthesis.TriggerSignals, "本轮未给出更细触发点。"),
		"**应事领域**：" + fallbackText(state.DynamicSynthesis.LiunianFocus, "本轮重点仍按现有动态裁定理解。"),
		"**限制**：" + buildDynamicConstraintText(state),
	})

	writeHeading(&b, "建议")
	writeConclusion(&b, buildTopicAdviceConclusion(state))
	writeBullets(&b, []string{
		"先求稳，再放大动作，避免把窗口年误当成毫无波动的纯顺年。",
	})
	return strings.TrimSpace(b.String())
}

func buildStrengthConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.StaticSynthesis.StrengthBalance); text != "" {
		return "命局强弱先按" + text + "来定。"
	}
	return ""
}

func buildTiaohouConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.StaticSynthesis.TiaohouConstraint); text != "" {
		return "这盘第一先看调候，核心约束在" + text + "。"
	}
	return ""
}

func buildPatternConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.StaticSynthesis.MainAxis); text != "" {
		return "命局主轴为" + text + "。"
	}
	return ""
}

func buildDayunConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.DynamicSynthesis.CurrentTrend); text != "" {
		return text
	}
	if len(state.DynamicSynthesis.DayunPath) > 0 {
		return state.DynamicSynthesis.DayunPath[0]
	}
	return ""
}

func buildLiunianConclusion(state baziCharterState) string {
	if text := strings.TrimSpace(state.DynamicSynthesis.LiunianFocus); text != "" {
		return text
	}
	if level := strings.TrimSpace(renderWindowLevel(state.DynamicSynthesis.WindowLevel)); level != "" {
		return "这一年更像" + level + "。"
	}
	return ""
}

func buildTopicDirectConclusion(plan baziAnalysisPlan, state baziCharterState, question string) string {
	if text := strings.TrimSpace(state.StaticSynthesis.TopicDirectAnswer); text != "" {
		return text
	}
	switch normalizedTopicMode(plan.TopicMode) {
	case "explain_term":
		return "这轮追问是在解释命盘里的一个结构说法，要放回主轴、限制和承接关系里一起看。"
	case "timing_reason":
		return "这轮追问重点不在重判原局，而在说明岁运为什么会形成当前这条兑现路径。"
	case "conservative_reason":
		return "这次口径偏保守，是因为成立面存在，但限制与反证也没有退出。"
	}
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "机会伴随强变动") {
		return "围绕您这次关心的问题看，整体属于机会伴随强变动，不适合按纯顺路来理解。"
	}
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "吉中有阻") {
		return "围绕您这次关心的问题看，整体更像吉中有阻，能推进，但不会完全无阻。"
	}
	if text := strings.TrimSpace(state.DynamicSynthesis.CurrentTrend); text != "" {
		return text
	}
	if strings.TrimSpace(question) != "" {
		return "围绕您这次关心的“" + strings.TrimSpace(question) + "”来看，判断宜保守落稳。"
	}
	return "围绕您这次关心的问题看，整体宜保守落稳。"
}

func buildTopicDirectParagraph(state baziCharterState) string {
	parts := make([]string, 0, 3)
	if text := strings.TrimSpace(state.StaticSynthesis.TopicFocusAnswer); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(state.StaticSynthesis.PatternOutcome); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(state.DynamicSynthesis.LiunianFocus); text != "" {
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return "本轮判断以现有结构化结论为准，先看成立面，再看限制面。"
	}
	return strings.Join(parts, " ")
}

func buildTopicFocusAnswer(plan baziAnalysisPlan, state baziCharterState) string {
	if text := strings.TrimSpace(state.StaticSynthesis.TopicFocusAnswer); text != "" {
		return text
	}
	switch normalizedTopicMode(plan.TopicMode) {
	case "explain_term":
		return "这次追问重点不在重判整盘，而在确认这句说法在当前命盘里到底是在解释主轴、限制，还是承接关系。"
	case "conservative_reason":
		return "这次判断偏保守，关键不在于主轴全无成立面，而在于" +
			buildOverviewLimitationSummary(state) +
			"，所以系统会把成局感与限制面一起写出来，不直接拔高。"
	case "timing_reason":
		return buildTierRealizationText(state)
	default:
		return ""
	}
}

func normalizedTopicMode(mode string) string {
	mode = strings.TrimSpace(mode)
	switch mode {
	case "explain_term", "timing_reason", "conservative_reason":
		return mode
	default:
		return "analysis"
	}
}

func buildTopicFrameText(state baziCharterState) string {
	if containsAnyText([]string{state.StaticSynthesis.MainAxis}, []string{"月劫", "劫财"}) {
		return "先按月令本气与透干关系，把这盘放在月劫格的框架里看。"
	}
	return fallbackText(state.StaticSynthesis.MainAxis, "先看这盘按什么格局框架立局。")
}

func buildTopicRouteText(state baziCharterState) string {
	if containsAnyText([]string{state.StaticSynthesis.MainAxis, state.StaticSynthesis.PatternOutcome}, []string{"食神制杀"}) {
		return "再看食神制杀这条路，意思是用食神去制七杀，把压力化成可用之力。"
	}
	return fallbackText(state.StaticSynthesis.PatternOutcome, "再看这盘在当前框架里靠什么路线成局。")
}

func buildTopicExplainPosition(state baziCharterState) string {
	parts := filterNonEmpty([]string{
		buildTopicFrameText(state),
		buildTopicRouteText(state),
	})
	if len(parts) == 0 {
		return "先确认这句话是在解释当前主轴、限制还是承接关系。"
	}
	return strings.Join(parts, " ")
}

func buildTopicExplainBoundary(state baziCharterState) string {
	return firstNonEmptyTrim(
		state.StaticSynthesis.CounterEvidence,
		"解释一句术语，不等于重判整盘高低；它该占多大权重，仍要回到前面已裁定的主轴与限制。",
	)
}

func buildTopicAdviceConclusion(state baziCharterState) string {
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "机会伴随强变动") {
		return "建议稳住节奏，先做确定性更高的安排，不宜激进。"
	}
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "吉中有阻") {
		return "建议按吉中有阻来应对，能进则进，但别忽略现实阻力。"
	}
	return "建议先稳住节奏，再逐步发力。"
}

func buildTierRealizationText(state baziCharterState) string {
	trend := strings.TrimSpace(state.DynamicSynthesis.CurrentTrend)
	switch {
	case containsAnyText([]string{trend}, []string{"吉中有阻", "机会伴随强变动"}):
		return "原局层次先按底盘看，岁运只决定兑现得更高还是被阻得更重；当前更像有机会但伴随波动，不宜把短期起色直接当成整盘升格。"
	case trend != "":
		return "原局先定底盘，岁运再看兑现；当前运势对主轴有承托，但是否真正放大层次，还要连同限制一起看。"
	default:
		return "原局先定底盘，岁运再看兑现；遇顺运可把成立面托得更高，遇逆运则多显有路而受制。"
	}
}

func buildOriginalTierText(state baziCharterState) string {
	tier := strings.TrimSpace(state.StaticSynthesis.TierJudgment)
	limit := buildOverviewLimitationSummary(state)
	switch {
	case tier != "" && limit != "":
		return "只看原局底盘，先按" + tier + "落判；此处之所以不先拔高，是因为" + limit + "仍在。"
	case tier != "":
		return "只看原局底盘，先按" + tier + "落判；岁运顺逆放到后面单看，不提前并入原局定级。"
	default:
		return "原局定级只看底盘，不把岁运顺逆提前并入；能否再上调，还要服从前面已经裁定的限制。"
	}
}

func buildClassicalAuditBullets(state baziCharterState) []string {
	hasAxis := containsAnyText([]string{state.StaticSynthesis.PatternOutcome, state.StaticSynthesis.MainAxis}, []string{"成立", "有路", "主轴"})
	hasLuckLift := containsAnyText([]string{state.DynamicSynthesis.CurrentTrend, state.DynamicSynthesis.LiunianFocus}, []string{"机会", "窗口", "顺势"})
	hasHardRisk := strings.TrimSpace(state.StaticSynthesis.LimitationLevel) == "核心硬伤"
	hasClearLine := strings.TrimSpace(state.StaticSynthesis.MainAxis) != "" && strings.TrimSpace(state.StaticSynthesis.AxisConsistency) != ""
	hasTurbidity := containsAnyText([]string{
		state.StaticSynthesis.CounterEvidence,
		state.StaticSynthesis.TiaohouConstraint,
		state.StaticSynthesis.TierBasis,
	}, []string{"受限", "不足", "难以", "调候", "寒", "病"})
	hasShouRisk := containsAnyText([]string{
		strings.Join(state.StaticSynthesis.Risks, " "),
		strings.Join(state.DynamicSynthesis.Risks, " "),
		state.StaticSynthesis.TiaohouConstraint,
	}, []string{"健康", "寒湿", "伤身", "损"})

	out := []string{
		"**贵**：" + classicalDimensionVerdict(
			hasAxis,
			"有其路数。主轴能立，故可言有贵气之理。",
			"不宜写满。主轴虽见其意，但仍受限制牵制。",
		),
		"**富**：" + classicalDimensionVerdict(
			hasLuckLift,
			"可随运见财。财富感更多看岁运如何承托。",
			"不作重断。原局能见财路，但不宜直接写成厚富。",
		),
		"**吉**：" + classicalDimensionVerdict(
			containsAnyText([]string{state.DynamicSynthesis.CurrentTrend}, []string{"吉中有阻", "机会伴随强变动"}) || strings.TrimSpace(state.StaticSynthesis.MainAxis) != "",
			"可言有吉。用神有路，但吉不离限制。",
			"不作纯吉。主轴未必全失，但现实阻力仍重。",
		),
		"**凶**：" + classicalDimensionVerdict(
			hasHardRisk,
			"须防硬伤发作，凶象不宜轻忽。",
			"不成纯凶。虽有限制，但未到全盘失守。",
		),
		"**寿**：" + classicalDimensionVerdict(
			!hasShouRisk,
			"可作平稳看。原局虽有病点，但不见重折之象。",
			"须防耗损。病点若遇逆运放大，宜守不宜躁。",
		),
		"**夭**：" + classicalDimensionVerdict(
			hasHardRisk && hasShouRisk,
			"不宜轻言，但若逆势叠加，须重视身心折损之象。",
			"不成立。原局有限而未至气竭神枯。",
		),
		"**清**：" + classicalDimensionVerdict(
			hasClearLine,
			"主线较清。主轴、限制、取舍关系都能交代清楚。",
			"清气不足。论断仍需防止多路并提。",
		),
		"**浊**：" + classicalDimensionVerdict(
			hasTurbidity,
			"浊处仍在。病药与限制未净，所以不能按纯清格写。",
			"不以浊局论。虽有限制，但主线未乱。",
		),
		"**成**：" + classicalDimensionVerdict(
			hasAxis,
			"可以立局。主轴不是空意，而是有证据支撑。",
			"未可言成。只能保留结构参考，不宜重断。",
		),
		"**败**：" + classicalDimensionVerdict(
			hasHardRisk,
			"有败笔。限制若被逆运放大，容易显得有路难行。",
			"不至全败。虽不能拔高，但也不是全盘失路。",
		),
	}
	return out
}

func classicalDimensionVerdict(condition bool, whenTrue, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func buildTopicConstraintText(state baziCharterState) string {
	parts := []string{buildLimitationText(state)}
	if text := strings.TrimSpace(state.DynamicSynthesis.CurrentTrend); text != "" {
		parts = append(parts, text)
	}
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "机会伴随强变动") {
		parts = append(parts, "机会伴随强变动。")
	}
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "吉中有阻") {
		parts = append(parts, "当前属于吉中有阻。")
	}
	return strings.Join(filterNonEmpty(parts), " ")
}

func buildDynamicConstraintText(state baziCharterState) string {
	parts := make([]string, 0, 4)
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "机会伴随强变动") {
		parts = append(parts, "机会伴随强变动，不宜激进。")
	}
	if containsString(state.DynamicSynthesis.ConsistencyFlags, "吉中有阻") {
		parts = append(parts, "整体属于吉中有阻。")
	}
	if len(state.DynamicSynthesis.Risks) > 0 {
		parts = append(parts, joinOrDefault(state.DynamicSynthesis.Risks, "风险仍需保守看待。"))
	}
	if len(parts) == 0 {
		return buildLimitationText(state)
	}
	return strings.Join(parts, " ")
}

func buildLimitationText(state baziCharterState) string {
	parts := make([]string, 0, 4)
	if text := strings.TrimSpace(state.StaticSynthesis.CounterEvidence); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(state.StaticSynthesis.TierBasis); text != "" {
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return "当前判断仍有明显限制，不宜拔高。"
	}
	return strings.Join(parts, " ")
}

func renderWindowLevel(level string) string {
	switch strings.TrimSpace(level) {
	case "窗口年":
		return "窗口年"
	case "扰动年":
		return "扰动年"
	case "转折年":
		return "转折年"
	case "承压年":
		return "承压年"
	default:
		return strings.TrimSpace(level)
	}
}

func ensureBullets(src []string, fallback []string) []string {
	if len(filterNonEmpty(src)) > 0 {
		return filterNonEmpty(src)
	}
	return filterNonEmpty(fallback)
}

func ensureSteps(src []string, fallback []string) []string {
	if len(filterNonEmpty(src)) > 0 {
		return filterNonEmpty(src)
	}
	return filterNonEmpty(fallback)
}

func filterNonEmpty(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func joinOrDefault(items []string, fallback string) string {
	items = filterNonEmpty(items)
	if len(items) == 0 {
		return sanitizeUnsupportedFlourish(fallback)
	}
	return sanitizeUnsupportedFlourish(strings.Join(items, "；"))
}

func joinInlineParts(parts ...string) string {
	items := filterNonEmpty(parts)
	if len(items) == 0 {
		return "当前仍需结合已有结构化裁定保守理解。"
	}
	return sanitizeUnsupportedFlourish(strings.Join(items, "；"))
}

func shortenClause(text, fallback string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return sanitizeUnsupportedFlourish(fallback)
	}
	parts := strings.FieldsFunc(text, func(r rune) bool {
		switch r {
		case '，', '。', '；', ',', ';', '：', ':':
			return true
		default:
			return false
		}
	})
	parts = filterNonEmpty(parts)
	if len(parts) == 0 {
		return sanitizeUnsupportedFlourish(fallback)
	}
	return sanitizeUnsupportedFlourish(parts[0])
}

func fallbackText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return sanitizeUnsupportedFlourish(fallback)
	}
	return sanitizeUnsupportedFlourish(strings.TrimSpace(value))
}

func conclusionOrDefault(fallback string, values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return fallback
}

func writeHeading(b *strings.Builder, heading string) {
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString("## ")
	b.WriteString(heading)
	b.WriteString("\n")
}

func writeSubheading(b *strings.Builder, heading string) {
	b.WriteString("\n")
	b.WriteString("### ")
	b.WriteString(heading)
	b.WriteString("\n")
}

func writeMinorHeading(b *strings.Builder, heading string) {
	b.WriteString("\n")
	b.WriteString("#### ")
	b.WriteString(heading)
	b.WriteString("\n")
}

func writeConclusion(b *strings.Builder, text string) {
	b.WriteString("**结论：")
	b.WriteString(strings.TrimSpace(text))
	b.WriteString("**\n")
}

func writeParagraphs(b *strings.Builder, paragraphs []string) {
	for _, paragraph := range filterNonEmpty(paragraphs) {
		b.WriteString(paragraph)
		b.WriteString("\n")
	}
}

func writeBullets(b *strings.Builder, bullets []string) {
	for _, bullet := range filterNonEmpty(bullets) {
		b.WriteString("- ")
		b.WriteString(strings.TrimSpace(bullet))
		b.WriteString("\n")
	}
}

func writeSteps(b *strings.Builder, steps []string) {
	for i, step := range filterNonEmpty(steps) {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.TrimSpace(step)))
	}
}

func writeHighlightBlock(b *strings.Builder, title, summary string, details ...string) {
	b.WriteString("> ")
	b.WriteString(strings.TrimSpace(title))
	b.WriteString("：")
	b.WriteString(strings.TrimSpace(summary))
	b.WriteString("\n")
	for _, detail := range filterNonEmpty(details) {
		b.WriteString("> ")
		b.WriteString(strings.TrimSpace(detail))
		b.WriteString("\n")
	}
}

func writeMatrixEntry(b *strings.Builder, title string, bullets []string) {
	writeMinorHeading(b, title)
	writeBullets(b, bullets)
}

func sanitizeUnsupportedFlourish(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"贵人众多", "助力较多",
		"福泽深厚", "福分较厚",
		"可享清福", "后程较稳",
	)
	return replacer.Replace(strings.TrimSpace(text))
}
