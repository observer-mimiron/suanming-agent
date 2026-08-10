// Package runtime 包含 Manager 拥有的八字 canonical synthesis 合同。
//
// 本文件负责静态/动态综合模型调用、字段级 repair 输入和 runtime projection；
// 不负责最终答复权，也不改写确定性排盘事实。
package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/state"
)

// runStaticSynthesis 将 canonical 共同裁断扩展为静态 claim 单元。
// 模型只能写 Schema DTO，legacy 展示字段仍由本文件的投影函数生成。
func (e *Executor) runStaticSynthesis(ctx context.Context, st *state.SessionState, chartState baziCharterState, canonical baziCanonicalSynthesis, question string) (baziCanonicalSynthesis, error) {
	payload := buildStaticSynthesisPayload(chartState)
	if strings.TrimSpace(canonical.MainAxis.Verdict) != "" {
		payload["canonical_synthesis"] = canonical
	}
	payload["runtime_catalog"] = baziStaticRuntimeCatalogView(chartState)
	out, err := runBaziInnerAgentJSON[baziStructuredStaticSynthesis](ctx, e.builder, baziStaticSynthesisConfig(), st, buildBaziCharterPrompt("静态综合", question, payload))
	if err != nil {
		return baziCanonicalSynthesis{}, err
	}
	out = normalizeBaziStaticJudgment(chartState, out)
	assertions := append(structuredStaticClaims(chartState, out.Claims), tierDimensionAssertions(out.TierAssessment)...)
	if err := validateStaticBaziReferenceCatalog(chartState, assertions); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := validateBaziStaticJudgmentPolicy(chartState, out); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	return applyStaticClaims(chartState, canonical, out)
}

// runStaticSynthesisRepair 仅重跑失败的静态 claim 节点，并附带字段级反馈。
// 它复用同一份 Schema 与 runtime catalog，不允许通过修改确定性事实绕过合同。
func (e *Executor) runStaticSynthesisRepair(ctx context.Context, st *state.SessionState, chartState baziCharterState, canonical baziCanonicalSynthesis, question string, feedback map[string]any) (baziCanonicalSynthesis, error) {
	payload := buildStaticSynthesisPayload(chartState)
	if strings.TrimSpace(canonical.MainAxis.Verdict) != "" {
		payload["canonical_synthesis"] = canonical
	}
	payload["runtime_catalog"] = baziStaticRuntimeCatalogView(chartState)
	payload["validation_feedback"] = feedback
	out, err := runBaziInnerAgentJSON[baziStructuredStaticSynthesis](ctx, e.builder, baziStaticSynthesisConfig(), st, buildBaziCharterPrompt("静态综合修复", question, payload))
	if err != nil {
		return baziCanonicalSynthesis{}, err
	}
	out = normalizeBaziStaticJudgment(chartState, out)
	assertions := append(structuredStaticClaims(chartState, out.Claims), tierDimensionAssertions(out.TierAssessment)...)
	if err := validateStaticBaziReferenceCatalog(chartState, assertions); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := validateBaziStaticJudgmentPolicy(chartState, out); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	return applyStaticClaims(chartState, canonical, out)
}

// normalizeBaziStaticJudgment 把原局风险状态投影到确定性官星透藏事实。
// 模型不能把官星不可见的命局写成既成风险；只修正这一项事实派生字段，其他合同冲突仍按原策略处理。
func normalizeBaziStaticJudgment(state baziCharterState, judgment baziStructuredStaticSynthesis) baziStructuredStaticSynthesis {
	if !buildBaziFactCapsule(state).OfficialVisible {
		judgment.NatalRiskStatus = "withheld"
	}
	return judgment
}

// runDynamicSynthesis 将已通过静态校验的主轴扩展为逐运与流年 claim。
// 该节点不重判静态结构，也不能输出确定性大运事实或 renderer 字段。
func (e *Executor) runDynamicSynthesis(ctx context.Context, st *state.SessionState, chartState baziCharterState, canonical baziCanonicalSynthesis, question string) (baziCanonicalSynthesis, error) {
	if err := validateDynamicPreconditions(chartState); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	payload := buildDynamicSynthesisPayload(chartState)
	payload["runtime_catalog"] = baziDynamicRuntimeCatalogView(chartState)
	out, err := runBaziInnerAgentJSON[baziStructuredDynamicSynthesis](ctx, e.builder, baziDynamicSynthesisConfig(), st, buildBaziCharterPrompt("动态综合", question, payload))
	if err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := validateDynamicPeriodClaims(chartState, out.PeriodClaims); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := validateBaziDynamicJudgmentPolicy(chartState, out); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	claims := structuredDynamicClaims(chartState, out)
	if err := validateDynamicBaziReferenceCatalog(chartState, claims); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	return applyDynamicClaims(chartState, canonical, out)
}

// runDynamicSynthesisRepair 只重跑失败的动态节点，并复用同一份动态 Schema 与引用目录。
// 它不重判静态主轴，也不把动态引用错误交给 canonical 节点处理。
func (e *Executor) runDynamicSynthesisRepair(ctx context.Context, st *state.SessionState, chartState baziCharterState, canonical baziCanonicalSynthesis, question string, feedback map[string]any) (baziCanonicalSynthesis, error) {
	if err := validateDynamicPreconditions(chartState); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	payload := buildDynamicSynthesisPayload(chartState)
	payload["runtime_catalog"] = baziDynamicRuntimeCatalogView(chartState)
	payload["validation_feedback"] = feedback
	out, err := runBaziInnerAgentJSON[baziStructuredDynamicSynthesis](ctx, e.builder, baziDynamicSynthesisConfig(), st, buildBaziCharterPrompt("动态综合修复", question, payload))
	if err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := validateDynamicPeriodClaims(chartState, out.PeriodClaims); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := validateBaziDynamicJudgmentPolicy(chartState, out); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	claims := structuredDynamicClaims(chartState, out)
	if err := validateDynamicBaziReferenceCatalog(chartState, claims); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	return applyDynamicClaims(chartState, canonical, out)
}

// applyStaticClaims 将静态 DTO 的固定 claim 集映射为 canonical 单元。
// 模型不拥有边界、限制或推理文本，它们必须从同轮事实胶囊单向投影。
func applyStaticClaims(state baziCharterState, canonical baziCanonicalSynthesis, output baziStructuredStaticSynthesis) (baziCanonicalSynthesis, error) {
	units := map[baziAssertionKind]*baziCanonicalUnit{
		baziAssertionMainAxis:     &canonical.MainAxis,
		baziAssertionStrength:     &canonical.Strength,
		baziAssertionTiaohou:      &canonical.Tiaohou,
		baziAssertionPatternUsage: &canonical.Pattern,
	}
	seen := map[baziAssertionKind]bool{}
	for index, claim := range output.Claims {
		kind, ok := staticClaimKindAt(index)
		if !ok {
			return baziCanonicalSynthesis{}, baziViolationError(baziViolationMethodContract, "static.claims", "", "static synthesis has unexpected claim count", nil, nil)
		}
		target, ok := units[kind]
		if !ok || seen[kind] {
			return baziCanonicalSynthesis{}, baziViolationError(baziViolationMethodContract, "static.claims", "", "static synthesis has invalid fixed claim slot", nil, nil)
		}
		*target = canonicalUnitFromStructuredStaticClaim(state, claim, kind)
		seen[kind] = true
	}
	for kind := range units {
		if !seen[kind] {
			return baziCanonicalSynthesis{}, baziViolationError(baziViolationScopeEscalation, "static.claims", "", "static synthesis misses required claim kind "+string(kind), nil, nil)
		}
	}
	canonical.TierAssessment = output.TierAssessment
	canonical.Tier = canonicalTierUnitFromAssessment(output.TierAssessment)
	canonical.Limitations = staticRuntimeLimitations(state)
	canonical.StaticReasoningSummary = staticRuntimeReasoningSummary(state)
	canonical.ReasoningSteps = staticRuntimeReasoningSteps()
	canonical.AdviceBoundary = staticRuntimeAdviceBoundary()
	return canonical, nil
}

// canonicalTierUnitFromAssessment keeps a nine-level result structured until
// runtime projects it into the legacy renderer fields. The model never writes
// the user-facing rank sentence itself.
func canonicalTierUnitFromAssessment(assessment baziTierAssessment) baziCanonicalUnit {
	factRefs, relationRefs, claimRefs, topics := tierAssessmentReferences(assessment)
	return baziCanonicalUnit{
		Kind:           string(baziAssertionTier),
		Verdict:        tierAssessmentJudgment(assessment),
		Boundary:       tierAssessmentBasis(assessment),
		FactRefs:       factRefs,
		RelationRefs:   relationRefs,
		ClaimRefs:      claimRefs,
		EvidenceTopics: topics,
		Confidence:     assessment.Confidence,
	}
}

// tierAssessmentReferences merges the nine structured dimensions into the
// one legacy tier assertion without adding any model-generated text.
func tierAssessmentReferences(assessment baziTierAssessment) ([]string, []baziRelationRef, []string, []string) {
	factRefs := []string{}
	claimRefs := []string{}
	topics := []string{}
	seenFacts, seenClaims, seenTopics := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, dimension := range baziTierDimensionEntries(assessment.Dimensions) {
		for _, ref := range dimension.Value.FactRefs {
			if value := strings.TrimSpace(string(ref)); value != "" && !seenFacts[value] {
				seenFacts[value] = true
				factRefs = append(factRefs, value)
			}
		}
		for _, ref := range dimension.Value.ClaimRefs {
			if value := strings.TrimSpace(string(ref)); value != "" && !seenClaims[value] {
				seenClaims[value] = true
				claimRefs = append(claimRefs, value)
			}
		}
		for _, topic := range dimension.Value.EvidenceTopics {
			if topic = strings.TrimSpace(topic); topic != "" && !seenTopics[topic] {
				seenTopics[topic] = true
				topics = append(topics, topic)
			}
		}
	}
	return factRefs, nil, claimRefs, topics
}

// tierAssessmentJudgment maps the selected level to the stable user-facing
// nine-level scale. A withheld state is reserved for absent core facts.
func tierAssessmentJudgment(assessment baziTierAssessment) string {
	if assessment.Status == "withheld" {
		return "命格基础层次暂缓判定"
	}
	labels := []string{
		"",
		"破格重，核心问题无救",
		"破格受阻，救应很弱",
		"有结构但病重待救",
		"结构受限，难以拔高",
		"中格，有路但利弊并见",
		"中上，可成但仍有短板",
		"上格，主轴清成且有力",
		"上上，清纯、病药得所、救应有效",
		"极上，主轴、清浊、病药、救应高度闭合",
	}
	if assessment.Level < 1 || assessment.Level >= len(labels) {
		return "命格基础层次暂缓判定"
	}
	prefix := "命格基础层次：第" + fmt.Sprintf("%d", assessment.Level) + "级（" + labels[assessment.Level] + "）"
	if assessment.Status == "provisional" {
		return prefix + "，暂定"
	}
	return prefix
}

// tierAssessmentBasis projects only the named dimension states so the renderer
// can explain a grade without consuming a second free-text tier conclusion.
func tierAssessmentBasis(assessment baziTierAssessment) string {
	if assessment.Status == "withheld" {
		return "静态主轴或基础命盘事实尚未建立，因此本轮不输出九级层次。"
	}
	parts := make([]string, 0, 9)
	for _, dimension := range baziTierDimensionEntries(assessment.Dimensions) {
		parts = append(parts, tierDimensionLabel(dimension.Name)+"："+tierDimensionStateLabel(dimension))
	}
	basis := "层次按主轴、有情、有力、清浊、病药、救应、调候与何知章交叉观察；" + strings.Join(parts, "；") + "。"
	if assessment.Status == "provisional" {
		return basis + " 相关主证尚未全部闭合，本级只作暂定定位，不上推高阶。"
	}
	return basis
}

// tierDimensionLabel keeps renderer text in domain vocabulary while model DTOs
// remain stable ASCII keys.
func tierDimensionLabel(name string) string {
	return map[string]string{
		"main_axis": "主轴", "youqing": "有情", "youli": "有力", "qingzhuo": "清浊",
		"disease": "病", "remedy": "药", "rescue": "救应", "tiaohou": "调候", "hezhizhang": "何知章印证",
	}[name]
}

// tierDimensionStateLabel translates state enums only; it does not re-evaluate them.
func tierDimensionStateLabel(dimension baziNamedTierDimension) string {
	if dimension.Disease {
		return map[string]string{
			"unresolved": "病势未明", "light": "病轻", "moderate": "病中", "heavy": "病重", "critical": "病重难解",
		}[dimension.Value.State]
	}
	return map[string]string{
		"missing": "缺位", "limited": "受限", "mixed": "并见", "usable": "可用", "strong": "得力",
	}[dimension.Value.State]
}

// applyDynamicClaims 将动态 DTO 按确定性大运顺序映射为 canonical 动态单元。
func applyDynamicClaims(state baziCharterState, canonical baziCanonicalSynthesis, output baziStructuredDynamicSynthesis) (baziCanonicalSynthesis, error) {
	periods := dayunPeriods(state.Input.Dayun)
	if err := validateDynamicPeriodClaims(state, output.PeriodClaims); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	canonical.DayunPeriods = make([]baziCanonicalDayunUnit, len(periods))
	for index, period := range periods {
		periodIndex := index
		canonical.DayunPeriods[index] = baziCanonicalDayunUnit{
			Index:  &periodIndex,
			GanZhi: strings.TrimSpace(stringValue(period["ganZhi"])),
		}
	}
	for _, claim := range output.PeriodClaims {
		index, _ := dynamicPeriodIndex(claim.PeriodRef, periods)
		canonical.DayunPeriods[index].Verdict = claim.Verdict
		canonical.DayunPeriods[index].Boundary = dynamicRuntimeClaimBoundary(baziAssertionDayunPeriod)
		canonical.DayunPeriods[index].FactRefs = factRefsToStrings(claim.FactRefs)
		canonical.DayunPeriods[index].RelationRefs = append([]baziRelationRef{}, claim.RelationRefs...)
		canonical.DayunPeriods[index].ClaimRefs = claimRefsToStrings(claim.ClaimRefs)
		canonical.DayunPeriods[index].EvidenceTopics = append([]string{}, claim.EvidenceTopics...)
		canonical.DayunPeriods[index].Confidence = claim.Confidence
	}
	canonical.Liunian = canonicalUnitFromStructuredClaim(output.LiunianClaim, baziAssertionLiunian)
	canonical.Liunian.Boundary = dynamicRuntimeClaimBoundary(baziAssertionLiunian)
	canonical.CurrentPeriodRealization = output.CurrentPeriodRealization
	for _, period := range canonical.DayunPeriods {
		if strings.TrimSpace(period.Verdict) == "" {
			continue
		}
		canonical.DayunOverview = baziCanonicalUnit{
			Kind: string(baziAssertionDayunPeriod), Verdict: period.Verdict, Boundary: period.Boundary,
			FactRefs: append([]string{}, period.FactRefs...), RelationRefs: append([]baziRelationRef{}, period.RelationRefs...),
			ClaimRefs: append([]string{}, period.ClaimRefs...), EvidenceTopics: append([]string{}, period.EvidenceTopics...), Confidence: period.Confidence,
		}
		break
	}
	canonical.Risks = append([]string{}, output.Limitations...)
	canonical.DynamicReasoningSummary = output.ReasoningSummary
	canonical.DynamicOutcomeDomains = append([]string{}, output.OutcomeDomains...)
	canonical.ReasoningSteps = append([]string{}, output.ReasoningSteps...)
	return canonical, nil
}

// dynamicRuntimeClaimBoundary owns dynamic display limits. The schema keeps a
// boundary field for transport compatibility, but model-authored limits are not
// trusted because they can contradict the static tier or leaked catalog terms.
func dynamicRuntimeClaimBoundary(kind baziAssertionKind) string {
	switch kind {
	case baziAssertionDayunPeriod:
		return "只按当前已绑定大运的日期边界和已声明关系观察，不改写本命基础判断。"
	case baziAssertionLiunian:
		return "只按当前大运与目标流年的已声明关系观察，不扩展为具体应事。"
	default:
		return "只解释本轮已声明的结构事实，不扩展为具体应事。"
	}
}

// staticClaimKindAt 返回静态 claims 数组的固定槽位语义，避免模型重复输出位置已知字段。
func staticClaimKindAt(index int) (baziAssertionKind, bool) {
	kinds := []baziAssertionKind{baziAssertionMainAxis, baziAssertionStrength, baziAssertionTiaohou, baziAssertionPatternUsage}
	if index < 0 || index >= len(kinds) {
		return "", false
	}
	return kinds[index], true
}

// structuredStaticClaims 把固定槽位 DTO 适配到共享 catalog/语义校验所需的内部 assertion。
// 静态关系不属于模型输入合同，边界由同轮确定性事实生成。
func structuredStaticClaims(state baziCharterState, claims []baziStructuredStaticClaim) []baziAssertion {
	out := make([]baziAssertion, len(claims))
	for index, claim := range claims {
		kind, _ := staticClaimKindAt(index)
		out[index] = structuredStaticClaimAssertion(state, claim, kind, staticClaimSubject(kind), fmt.Sprintf("static.%s", kind))
	}
	return out
}

// structuredDynamicClaims 把按大运序号和流年槽位返回的 DTO 适配到共享引用校验。
func structuredDynamicClaims(state baziCharterState, output baziStructuredDynamicSynthesis) []baziAssertion {
	out := make([]baziAssertion, 0, len(output.PeriodClaims)+1)
	for _, claim := range output.PeriodClaims {
		periodIndex, _ := dynamicPeriodIndex(claim.PeriodRef, dayunPeriods(state.Input.Dayun))
		out = append(out, structuredClaimAssertion(structuredClaimFromPeriodClaim(claim), baziAssertionDayunPeriod, fmt.Sprintf("dayun.%d", periodIndex), fmt.Sprintf("dynamic.dayun.%d", periodIndex)))
	}
	out = append(out, structuredClaimAssertion(output.LiunianClaim, baziAssertionLiunian, "liunian", "dynamic.liunian"))
	return out
}

// structuredClaimFromPeriodClaim removes the period selector before shared reference validation.
func structuredClaimFromPeriodClaim(claim baziStructuredPeriodClaim) baziStructuredClaim {
	return baziStructuredClaim{
		Verdict: claim.Verdict, FactRefs: claim.FactRefs, RelationRefs: claim.RelationRefs,
		ClaimRefs: claim.ClaimRefs, EvidenceTopics: claim.EvidenceTopics, Confidence: claim.Confidence, Boundary: claim.Boundary,
	}
}

// validateDynamicPeriodClaims checks model-selected periods against the current deterministic catalog.
func validateDynamicPeriodClaims(state baziCharterState, claims []baziStructuredPeriodClaim) error {
	periods := dayunPeriods(state.Input.Dayun)
	seen := map[int]bool{}
	currentIndex := currentDayunIndexForInput(state.Input)
	for index, claim := range claims {
		periodIndex, ok := dynamicPeriodIndex(claim.PeriodRef, periods)
		field := fmt.Sprintf("dynamic.period_claims[%d].period_ref", index)
		if !ok {
			return baziViolationError(baziViolationUndeclaredFactClaim, field, "", "period_ref is not declared in this runtime catalog", []string{claim.PeriodRef}, baziDynamicPeriodRefs(state))
		}
		if seen[periodIndex] {
			return baziViolationError(baziViolationMethodContract, field, "", "dynamic period claim repeats the same period", nil, nil)
		}
		if currentIndex >= 0 && periodIndex != currentIndex {
			return baziViolationError(baziViolationMethodContract, field, "", "dynamic period claim must use the runtime-selected current period", []string{fmt.Sprintf("dayun[%d]", currentIndex)}, []string{fmt.Sprintf("dayun[%d]", currentIndex)})
		}
		seen[periodIndex] = true
	}
	return nil
}

// dynamicPeriodIndex resolves the exact period selector accepted by the dynamic model contract.
func dynamicPeriodIndex(ref string, periods []map[string]any) (int, bool) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, "dayun[") || !strings.HasSuffix(ref, "]") {
		return 0, false
	}
	var index int
	if _, err := fmt.Sscanf(ref, "dayun[%d]", &index); err != nil || fmt.Sprintf("dayun[%d]", index) != ref {
		return 0, false
	}
	return index, index >= 0 && index < len(periods)
}

// structuredClaimAssertion 只在 runtime 内部补齐固定定位字段，模型输出仍保持最小合同。
func structuredClaimAssertion(claim baziStructuredClaim, kind baziAssertionKind, subject, id string) baziAssertion {
	return baziAssertion{ID: id, Kind: kind, Subject: subject, Verdict: claim.Verdict, FactRefs: claim.FactRefs, RelationRefs: claim.RelationRefs, ClaimRefs: claim.ClaimRefs, EvidenceTopics: claim.EvidenceTopics, Confidence: claim.Confidence, Boundary: claim.Boundary}
}

// structuredStaticClaimAssertion adapts the bounded static verdict while
// keeping boundaries and original-chart relations runtime-owned.
func structuredStaticClaimAssertion(state baziCharterState, claim baziStructuredStaticClaim, kind baziAssertionKind, subject, id string) baziAssertion {
	return baziAssertion{
		ID:             id,
		Kind:           kind,
		Subject:        subject,
		Verdict:        claim.Verdict,
		FactRefs:       claim.FactRefs,
		ClaimRefs:      claim.ClaimRefs,
		EvidenceTopics: claim.EvidenceTopics,
		Confidence:     staticClaimConfidence(claim.Status),
		Boundary:       staticRuntimeClaimBoundary(state, kind),
	}
}

// staticClaimSubject 返回静态 claim 的固定主体，避免模型重复生成展示定位字段。
func staticClaimSubject(kind baziAssertionKind) string {
	if kind == baziAssertionStrength {
		return "day_master"
	}
	return "chart"
}

// canonicalUnitFromStructuredClaim 将最小模型 claim 补成内部 canonical 单元。
func canonicalUnitFromStructuredClaim(claim baziStructuredClaim, kind baziAssertionKind) baziCanonicalUnit {
	return baziCanonicalUnit{
		Kind: string(kind), Verdict: claim.Verdict, Boundary: claim.Boundary,
		FactRefs: factRefsToStrings(claim.FactRefs), RelationRefs: append([]baziRelationRef{}, claim.RelationRefs...),
		ClaimRefs: claimRefsToStrings(claim.ClaimRefs), EvidenceTopics: append([]string{}, claim.EvidenceTopics...), Confidence: claim.Confidence,
	}
}

// canonicalUnitFromStructuredStaticClaim keeps the bounded model verdict and
// attaches the deterministic boundary that constrains its presentation.
func canonicalUnitFromStructuredStaticClaim(state baziCharterState, claim baziStructuredStaticClaim, kind baziAssertionKind) baziCanonicalUnit {
	return baziCanonicalUnit{
		Kind:           string(kind),
		Verdict:        claim.Verdict,
		Boundary:       staticRuntimeClaimBoundary(state, kind),
		FactRefs:       factRefsToStrings(claim.FactRefs),
		ClaimRefs:      claimRefsToStrings(claim.ClaimRefs),
		EvidenceTopics: append([]string{}, claim.EvidenceTopics...),
		Confidence:     staticClaimConfidence(claim.Status),
	}
}

// staticClaimConfidence derives display certainty from the same status used by the DTO.
func staticClaimConfidence(status string) string {
	return map[string]string{"established": "明确成立", "candidate": "倾向成立", "limited": "保守判断", "withheld": "保守判断"}[status]
}

// staticRuntimeClaimBoundary generates the static explanation boundary from
// deterministic facts. It intentionally does not consume any model prose.
func staticRuntimeClaimBoundary(state baziCharterState, kind baziAssertionKind) string {
	capsule := buildBaziFactCapsule(state)
	switch kind {
	case baziAssertionMainAxis:
		parts := []string{"主轴只按月令、透干、通根与已覆盖规则材料裁断。"}
		if !capsule.OfficialVisible && capsule.OfficialHidden {
			parts = append(parts, "官星藏支未透，原局官星风险不作既成限制；是否受岁运引动由动态层判断。")
		}
		return strings.Join(parts, "")
	case baziAssertionStrength:
		return "强弱依据月令、通根位置和层级、同类透干、印星生扶、食伤泄身、财官耗克及已计算受力；" + strengthEvidenceSummary(state.Input.Yongshen) + "扶抑结论不自动等同于格局取用或调候用神。"
	case baziAssertionTiaohou:
		return "调候先看月令的寒暖燥湿需求，再核对火的出现、透出和有效性；" + capsuleTiaohouDisplay(capsule) + "。火存在、火根或午的地势均不自动等同于调候有效。"
	case baziAssertionPatternUsage:
		return firstNonEmptyTrim(staticPatternFactSummary(state.Input), "工具未返回完整月令取格事实。") + "；格局取用只在已覆盖证据范围内比较，不把未闭合结构拔高为成格或贵格。"
	default:
		return "只解释已计算命盘事实和已覆盖证据，不展开具体应事。"
	}
}

// staticRuntimeLimitations projects only fact-capsule limits, so a model cannot
// inject a second risk conclusion through a generic limitations list.
func staticRuntimeLimitations(state baziCharterState) []string {
	capsule := buildBaziFactCapsule(state)
	limits := []string{}
	if !capsule.OfficialVisible && capsule.OfficialHidden {
		limits = append(limits, "官星藏支未透，原局官星风险不作既成限制；岁运条件另由动态层说明。")
	}
	switch {
	case !capsule.FireEffectivenessKnown:
		limits = append(limits, "调候有效性尚待明确材料确认，不以火存在或火根替代有效性判断。")
	case !capsule.FireEffective:
		limits = append(limits, "已有材料显示火不足以单独作为调候依据。")
	}
	if !capsule.TierEvidenceComplete {
		limits = append(limits, capsuleTierEvidenceDisplay(capsule)+"，层次仅作暂定定位。")
	}
	return canonicalListOrDefault(limits, []string{"静态层只解释本命结构；岁运承接与条件风险由动态层单独处理。"})
}

// staticRuntimeReasoningSummary states the fixed static decision basis without
// repeating the model's main-axis wording in another renderer section.
func staticRuntimeReasoningSummary(state baziCharterState) string {
	capsule := buildBaziFactCapsule(state)
	return "静态裁断按月令、通根、透干、受力、官星透藏、调候资格与层次独立证据状态投影；" + capsuleTierEvidenceDisplay(capsule) + "。"
}

// staticRuntimeReasoningSteps returns the stable order used by the legacy
// renderer. These steps describe the contract and never add a chart conclusion.
func staticRuntimeReasoningSteps() []string {
	return []string{
		"先核对月令、通根、透干、印比与泄耗克身的已计算事实。",
		"再在事实与已覆盖规则材料范围内裁断主轴、强弱、调候和格局。",
		"最后单列九级本命层次；当前大运只解释承接，不改写本命等级。",
	}
}

// staticRuntimeAdviceBoundary keeps static and dynamic responsibilities apart
// in the projected legacy field.
func staticRuntimeAdviceBoundary() string {
	return "静态层只解释本命结构；岁运承接、流年触发和条件风险由动态层单独处理。"
}

// canonicalUnitFromAssertion 仅适配同一份 Schema 已验证的引用型 claim。
func canonicalUnitFromAssertion(claim baziAssertion) baziCanonicalUnit {
	return baziCanonicalUnit{
		Kind:           string(claim.Kind),
		Verdict:        claim.Verdict,
		Boundary:       claim.Boundary,
		FactRefs:       factRefsToStrings(claim.FactRefs),
		RelationRefs:   append([]baziRelationRef{}, claim.RelationRefs...),
		ClaimRefs:      claimRefsToStrings(claim.ClaimRefs),
		EvidenceTopics: append([]string{}, claim.EvidenceTopics...),
		Confidence:     claim.Confidence,
	}
}

// claimRefsToStrings 保持 canonical DTO 的 JSON 兼容引用表示。
func claimRefsToStrings(refs []baziClaimRef) []string {
	out := make([]string, len(refs))
	for index, ref := range refs {
		out[index] = string(ref)
	}
	return out
}

// buildBaziCanonicalRepairFeedback 生成字段级 repair 反馈。
// learning_hints 只来自代码固化短提示，不读取线上 trace 或候选全文。
func buildBaziCanonicalRepairFeedback(failure RepairFailure, attempt int) map[string]any {
	feedback := map[string]any{
		"retry_attempt":  attempt,
		"failed_stage":   failure.Stage,
		"failure_class":  string(failure.Class),
		"field":          failure.Field,
		"reason":         firstNonEmptyTrim(failure.Message, failure.Code, "字段投影未通过合同校验。"),
		"allowed_fix":    baziCanonicalRepairAllowedFix(failure.Field),
		"must_preserve":  baziCanonicalRepairMustPreserve(),
		"forbidden":      baziCanonicalRepairForbidden(),
		"learning_hints": repairLearningHintFeedback(RepairLearningHintsFor(failure)),
	}
	if len(failure.MissingRefs) > 0 || len(failure.AllowedRefs) > 0 {
		feedback["reference_feedback"] = map[string]any{
			"invalid_refs": append([]string(nil), failure.MissingRefs...),
			"allowed_refs": append([]string(nil), failure.AllowedRefs...),
		}
	}
	return feedback
}

// baziCanonicalRepairAllowedFix 把 legacy 投影字段收束到可改的 canonical 单元。
func baziCanonicalRepairAllowedFix(field string) []string {
	switch strings.TrimSpace(field) {
	case "static.tiaohou_anchor", "static.tiaohou_constraint", "static.usage.tiaohou", "static.tiaohou":
		return []string{
			"只修改 canonical.tiaohou.verdict",
			"必要时修改 canonical.tiaohou.boundary",
		}
	default:
		return []string{
			"只修改导致本字段校验失败的 canonical 单元",
			"保持未失败 canonical 单元的裁断语义不变",
		}
	}
}

// baziCanonicalRepairMustPreserve 列出 repair 不能改动的事实和主轴。
func baziCanonicalRepairMustPreserve() []string {
	return []string{
		"四柱",
		"日主",
		"月令",
		"藏干与透干",
		"大运",
		"流年",
		"main_axis",
		"strength",
		"pattern",
	}
}

// baziCanonicalRepairForbidden 列出 repair 禁止越界的内容。
func baziCanonicalRepairForbidden() []string {
	return []string{
		"不得改排盘事实",
		"不得新增具体现实应事",
		"不得把证据不足写成强裁断",
		"不得输出 markdown、解释文字或非 JSON 内容",
	}
}

// projectCanonicalSynthesis maps the model's minimal judgment into the existing
// renderer structs. The projection is one-way; legacy fields are no longer model
// source-of-truth.
func projectCanonicalSynthesis(state baziCharterState, canonical baziCanonicalSynthesis) (baziStaticSynthesis, baziDynamicSynthesis) {
	static := projectCanonicalStaticSynthesis(state, canonical)
	dynamic := projectCanonicalDynamicSynthesis(state, canonical, static)
	return static, dynamic
}

func projectCanonicalStaticSynthesis(state baziCharterState, c baziCanonicalSynthesis) baziStaticSynthesis {
	tierVerdict, tierBoundary, tierWithheld := canonicalTierText(state, c.Tier, c.TierAssessment)
	reasoningSteps := c.ReasoningSteps
	if len(reasoningSteps) == 0 {
		reasoningSteps = []string{
			"先核对排盘、月令与日主强弱的可复算事实。",
			"再结合格局、调候和证据缺口收束主轴边界。",
			"最后只在证据覆盖范围内说明层次与岁运兑现。",
		}
	}
	patternBasis := firstNonEmptyTrim(c.Pattern.Boundary, staticPatternFactSummary(state.Input), "本轮只保留可复算结构事实。")
	patternOutcome := firstNonEmptyTrim(c.Pattern.Verdict, c.MainAxis.Verdict, "本轮未形成格局成败裁断。")
	counterEvidence := firstNonEmptyTrim(strings.Join(c.Limitations, "；"), c.MainAxis.Boundary, c.Pattern.Boundary, "本轮未形成额外反证。")
	axisConsistency := firstNonEmptyTrim(c.MainAxis.Boundary, c.Pattern.Boundary, "主轴边界以已覆盖证据为准。")
	strengthBalance := firstNonEmptyTrim(c.Strength.Verdict, strengthEvidenceSummary(state.Input.Yongshen), "本轮未形成强弱裁断。")
	tiaohouAnchor := firstNonEmptyTrim(c.Tiaohou.Verdict, "本轮只确认季节环境与调候边界。")
	tiaohouConstraint := firstNonEmptyTrim(c.Tiaohou.Boundary, "调候先后需以已覆盖规则材料为准。")
	claimStrength := canonicalConfidence(c.MainAxis.Confidence)
	static := baziStaticSynthesis{
		Source:                  firstNonEmptyTrim(c.Source, "model"),
		RuleProfile:             strings.TrimSpace(state.Input.RuleProfile.ID),
		MainAxis:                firstNonEmptyTrim(c.MainAxis.Verdict, "本轮未形成主轴裁断。"),
		ClaimStrength:           claimStrength,
		SupportLevel:            canonicalSupportLevel(claimStrength),
		LimitationLevel:         canonicalLimitationLevel(counterEvidence),
		WordingCap:              canonicalWordingCap(claimStrength),
		ConsistencyFlags:        []string{"仅作结构观察"},
		AxisLevel:               "方向成立",
		EffectOnTiaohou:         "中性",
		EffectOnCoreDisease:     "中性",
		EffectOnJiShenDirection: "中性",
		AxisCeiling:             "受限路线",
		ConflictReasons:         filterNonEmpty([]string{counterEvidence}),
		PatternBasis:            patternBasis,
		PatternOutcome:          patternOutcome,
		CounterEvidence:         counterEvidence,
		AxisConsistency:         axisConsistency,
		TiaohouConstraint:       tiaohouConstraint,
		TiaohouAnchor:           tiaohouAnchor,
		StrengthBalance:         strengthBalance,
		Strength: baziStrengthJudgment{
			Conclusion: canonicalStrengthConclusion(c.Strength.Verdict),
			Reasoning:  firstNonEmptyTrim(c.Strength.Boundary, strengthBalance),
			Boundary:   firstNonEmptyTrim(c.Strength.Boundary, "扶抑结论不自动等于格局取用或调候用神。"),
		},
		Usage: baziUsageLayers{
			Fuyi:     firstNonEmptyTrim(c.Strength.Boundary, "扶抑取用以强弱证据边界为准。"),
			Pattern:  patternOutcome,
			Tiaohou:  tiaohouConstraint,
			Priority: "以主轴、调候和证据覆盖共同收束，不作单一维度硬推。",
		},
		PatternAdjudication: buildProjectedPatternAdjudication(state),
		PatternAndQingZhuo:  firstNonEmptyTrim(c.Pattern.Boundary, c.Pattern.Verdict, "本轮仅作结构观察。"),
		QiShiOrCongHua:      "本轮不以气势从化另立主轴。",
		TierJudgment:        tierVerdict,
		TierBasis:           tierBoundary,
		TierAssessment:      c.TierAssessment,
		ReasoningSummary:    firstNonEmptyTrim(c.StaticReasoningSummary, c.MainAxis.Verdict+"；"+c.Pattern.Verdict+"；"+tierBoundary, c.MainAxis.Verdict),
		ReasoningSteps:      reasoningSteps,
		TopicDirectAnswer:   "",
		TopicFocusAnswer:    "",
		Advantages:          canonicalListOrDefault(c.Advantages, []string{firstNonEmptyTrim(c.MainAxis.Verdict, "结构主轴可观察。")}),
		Risks:               canonicalListOrDefault(c.Risks, []string{counterEvidence}),
		Citations:           mergeCitations(c.Citations, state.EvidenceBundle.Citations...),
		ContractAudit:       c.ContractAudit,
		FieldAudit:          append([]string{}, c.FieldAudit...),
	}
	if tierWithheld {
		static.FieldAudit = append(static.FieldAudit, "canonical_tier_withheld_by_runtime")
	}
	static = sanitizeMinorStaticProjection(state, static)
	static.Assertions = buildProjectedStaticAssertions(state, static, c, tierWithheld)
	return static
}

// sanitizeMinorStaticProjection keeps the code-owned legacy projection inside
// the minor age-domain contract. The canonical model may mention adult outcome
// domains while explaining risk; static renderer fields must drop those concrete
// outcomes and retain only structure, growth, care routine or observable change.
func sanitizeMinorStaticProjection(state baziCharterState, static baziStaticSynthesis) baziStaticSynthesis {
	context := buildBaziSubjectContext(state.Input)
	if context.AgeBand != "infant" && context.AgeBand != "child" && context.AgeBand != "adolescent" {
		return static
	}
	fallback := "本轮仅按结构、成长环境、照护节奏和可观察发展观察，不展开成人现实落点。"
	static.MainAxis = minorOutcomeSafeText(static.MainAxis, "本轮仅作命局结构观察。")
	static.PatternBasis = minorOutcomeSafeText(static.PatternBasis, fallback)
	static.PatternOutcome = minorOutcomeSafeText(static.PatternOutcome, fallback)
	static.CounterEvidence = minorOutcomeSafeText(static.CounterEvidence, fallback)
	static.AxisConsistency = minorOutcomeSafeText(static.AxisConsistency, fallback)
	static.TiaohouConstraint = minorOutcomeSafeText(static.TiaohouConstraint, fallback)
	static.TiaohouAnchor = minorOutcomeSafeText(static.TiaohouAnchor, fallback)
	static.TierJudgment = minorOutcomeSafeText(static.TierJudgment, "命格层次中等（保守定位）")
	static.TierBasis = minorOutcomeSafeText(static.TierBasis, fallback)
	static.ReasoningSummary = minorOutcomeSafeText(static.ReasoningSummary, fallback)
	static.ReasoningSteps = minorOutcomeSafeList(static.ReasoningSteps, []string{
		"先看命局结构、季节环境和已覆盖证据。",
		"再把结论限制在成长环境、照护节奏和可观察发展内。",
	})
	static.Advantages = minorOutcomeSafeList(static.Advantages, []string{"结构主轴可观察。"})
	static.Risks = minorOutcomeSafeList(static.Risks, []string{fallback})
	static.Strength.Conclusion = minorOutcomeSafeText(static.Strength.Conclusion, "强弱仅作受力估计。")
	static.Strength.Reasoning = minorOutcomeSafeText(static.Strength.Reasoning, "强弱为受力估计，不直接推出具体应事。")
	static.Strength.Boundary = minorOutcomeSafeText(static.Strength.Boundary, "扶抑结论不自动等于格局取用或现实落点。")
	static.Usage.Fuyi = minorOutcomeSafeText(static.Usage.Fuyi, "扶抑取用以强弱证据边界为准。")
	static.Usage.Pattern = minorOutcomeSafeText(static.Usage.Pattern, fallback)
	static.Usage.Tiaohou = minorOutcomeSafeText(static.Usage.Tiaohou, fallback)
	static.Usage.Priority = minorOutcomeSafeText(static.Usage.Priority, "以结构和证据覆盖共同收束。")
	return static
}

func minorOutcomeSafeText(text, fallback string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	if domain, _ := firstUnauthorizedMinorOutcomeSignal(text); domain != "" {
		return fallback
	}
	return text
}

func minorOutcomeSafeList(items, fallback []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if domain, _ := firstUnauthorizedMinorOutcomeSignal(item); domain != "" {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return append([]string{}, fallback...)
	}
	return out
}

func projectCanonicalDynamicSynthesis(state baziCharterState, c baziCanonicalSynthesis, static baziStaticSynthesis) baziDynamicSynthesis {
	if !state.AnalysisPlan.NeedDynamic {
		return baziDynamicSynthesis{}
	}
	facts := buildFactsOnlyDynamicSynthesis(state.Input, static, "")
	judgments := projectCanonicalDayunJudgments(state, c)
	dayunPath := facts.DayunPath
	if len(judgments) > 0 {
		dayunPath = mergeCanonicalDayunPath(dayunPath, judgments)
	}
	outcomeDomains := append([]string{}, c.DynamicOutcomeDomains...)
	if len(outcomeDomains) == 0 {
		outcomeDomains = canonicalOutcomeDomains(state)
	}
	claimStrength := canonicalConfidence(firstNonEmptyTrim(c.DayunOverview.Confidence, c.Liunian.Confidence))
	out := baziDynamicSynthesis{
		Source:                   firstNonEmptyTrim(c.Source, "model"),
		CurrentTrend:             firstNonEmptyTrim(c.DayunOverview.Verdict, facts.CurrentTrend),
		CurrentPeriodRealization: c.CurrentPeriodRealization,
		ClaimStrength:            claimStrength,
		SupportLevel:             canonicalSupportLevel(claimStrength),
		LimitationLevel:          canonicalLimitationLevel(strings.Join(append(c.Limitations, c.Risks...), "；")),
		WordingCap:               canonicalWordingCap(claimStrength),
		ConsistencyFlags:         []string{dynamicFlagStructureOnly},
		DayunPath:                dayunPath,
		DayunJudgments:           judgments,
		CurrentDayunIndex:        facts.CurrentDayunIndex,
		LiunianFocus:             firstNonEmptyTrim(c.Liunian.Verdict, facts.LiunianFocus),
		WindowLevel:              canonicalWindowLevel(c.Liunian.Verdict),
		TriggerSignals:           canonicalTriggerSignals(state, c),
		KeyWindows:               []string{firstNonEmptyTrim(c.DayunOverview.Verdict, "本轮只作结构观察。")},
		Risks:                    canonicalListOrDefault(c.Risks, facts.Risks),
		ReasoningSummary:         firstNonEmptyTrim(c.DynamicReasoningSummary, c.DayunOverview.Verdict+"；"+c.Liunian.Verdict, facts.ReasoningSummary),
		ReasoningSteps:           canonicalListOrDefault(c.ReasoningSteps, facts.ReasoningSteps),
		OutcomeDomains:           outcomeDomains,
		ContractAudit:            c.ContractAudit,
		FieldAudit:               append([]string{}, c.FieldAudit...),
	}
	out.Assertions = buildProjectedDynamicAssertions(state, out)
	return out
}

func canonicalTierText(state baziCharterState, tier baziCanonicalUnit, assessment baziTierAssessment) (string, string, bool) {
	if assessment.Status != "" {
		return tierAssessmentJudgment(assessment), tierAssessmentBasis(assessment), assessment.Status == "withheld"
	}
	missing := tierMissingTopics(state, tier)
	if len(missing) > 0 {
		return boundedTierJudgment(tier), boundedTierBasis(missing), true
	}
	return firstNonEmptyTrim(tier.Verdict, "仅作结构观察"), firstNonEmptyTrim(tier.Boundary, "层次判断以已覆盖证据为边界。"), false
}

// boundedTierJudgment keeps an incomplete tier claim within its evidence boundary.
func boundedTierJudgment(tier baziCanonicalUnit) string {
	return "命格层次暂不定级（仅作结构观察）"
}

// boundedTierBasis explains the rank cap in domain language instead of
// exposing an internal no-tier state to the user.
func boundedTierBasis(missing []string) string {
	labels := evidenceTopicLabels(missing)
	if len(labels) == 0 {
		labels = []string{"关键主证"}
	}
	return "层次需由清浊、病药、救应与破格风险等独立依据共同支撑；当前" +
		strings.Join(labels, "、") + "链条尚未闭合，因此不作高低定级。"
}

// evidenceTopicLabels converts internal evidence topic keys into product copy
// so renderer fields do not leak fixture or contract identifiers.
func evidenceTopicLabels(topics []string) []string {
	labels := []string{}
	for _, topic := range topics {
		switch strings.TrimSpace(topic) {
		case "geju":
			labels = append(labels, "格局成败")
		case "tiaohou":
			labels = append(labels, "调候")
		case "bingyao":
			labels = append(labels, "病药救应")
		case "jiuying":
			labels = append(labels, "救应")
		case "poge":
			labels = append(labels, "破格风险")
		case "qingzhuo":
			labels = append(labels, "清浊")
		case "hezhizhang":
			labels = append(labels, "何知章印证")
		case "dayun":
			labels = append(labels, "大运兑现")
		default:
			if strings.TrimSpace(topic) != "" {
				labels = append(labels, strings.TrimSpace(topic))
			}
		}
	}
	return labels
}

func tierMissingTopics(state baziCharterState, tier baziCanonicalUnit) []string {
	if len(state.EvidenceQuality.MissingTopics) == 0 {
		return nil
	}
	required := state.EvidenceQuality.RequiredTopics
	if len(required) == 0 {
		required = tier.EvidenceTopics
	}
	missing := []string{}
	for _, topic := range required {
		if containsString(state.EvidenceQuality.MissingTopics, topic) {
			missing = append(missing, topic)
		}
	}
	return missing
}

func canonicalEvidenceStatus(state baziCharterState, unit baziCanonicalUnit) string {
	if len(unit.EvidenceTopics) == 0 {
		return baziEvidenceSupported
	}
	for _, topic := range unit.EvidenceTopics {
		if containsString(state.EvidenceQuality.MissingTopics, topic) {
			return baziEvidenceWithheld
		}
	}
	return baziEvidenceSupported
}

func buildProjectedStaticAssertions(state baziCharterState, static baziStaticSynthesis, c baziCanonicalSynthesis, tierWithheld bool) []baziAssertion {
	main := canonicalAssertion(state, "static.main_axis", baziAssertionMainAxis, "chart", c.MainAxis)
	strength := canonicalAssertion(state, "static.strength", baziAssertionStrength, "day_master", c.Strength)
	tiaohou := canonicalAssertion(state, "static.tiaohou", baziAssertionTiaohou, "chart", c.Tiaohou)
	pattern := canonicalAssertion(state, "static.pattern", baziAssertionPatternUsage, "chart", c.Pattern)
	tier := canonicalAssertion(state, "static.tier", baziAssertionTier, "chart", c.Tier)
	if tierWithheld {
		tier.Verdict = static.TierJudgment
		tier.Boundary = static.TierBasis
		tier.EvidenceStatus = baziEvidenceWithheld
	}
	return []baziAssertion{main, strength, tiaohou, pattern, tier}
}

func buildProjectedDynamicAssertions(state baziCharterState, dynamic baziDynamicSynthesis) []baziAssertion {
	return ensureDynamicAssertions(state, dynamic).Assertions
}

func canonicalAssertion(state baziCharterState, id string, kind baziAssertionKind, subject string, unit baziCanonicalUnit) baziAssertion {
	return baziAssertion{
		ID:             id,
		Kind:           kind,
		Subject:        subject,
		Verdict:        firstNonEmptyTrim(unit.Verdict, "仅作结构观察"),
		FactRefs:       stringsToFactRefs(unit.FactRefs),
		RelationRefs:   append([]baziRelationRef{}, unit.RelationRefs...),
		ClaimRefs:      stringsToClaimRefs(unit.ClaimRefs),
		EvidenceTopics: append([]string{}, unit.EvidenceTopics...),
		EvidenceStatus: canonicalEvidenceStatus(state, unit),
		Confidence:     canonicalConfidence(unit.Confidence),
		Boundary:       firstNonEmptyTrim(unit.Boundary, "仅解释已覆盖证据内的结构，不展开具体应事。"),
	}
}

func stringsToFactRefs(items []string) []baziFactRef {
	out := make([]baziFactRef, 0, len(items))
	for _, item := range filterNonEmpty(items) {
		out = append(out, baziFactRef(item))
	}
	return out
}

func stringsToClaimRefs(items []string) []baziClaimRef {
	out := make([]baziClaimRef, 0, len(items))
	for _, item := range filterNonEmpty(items) {
		out = append(out, baziClaimRef(item))
	}
	return out
}

func buildProjectedPatternAdjudication(state baziCharterState) baziPatternAdjudication {
	candidateName := firstNonEmptyTrim(stringValue(state.Input.Yongshen["geju_candidate"]), "月令候选")
	return baziPatternAdjudication{
		MonthCommandCandidateID:  "month_command",
		SelectedAxisCandidateIDs: []string{"month_command"},
		Candidates: []baziPatternCandidate{{
			ID:             "month_command",
			Name:           candidateName,
			Origin:         "month_command",
			Visibility:     "以工具返回透藏事实为准",
			Role:           "selected_axis",
			FactRefs:       []baziFactRef{"yongshen.geju_candidate", "chart.month_branch"},
			EvidenceTopics: []string{"geju"},
			ComparisonDimensions: []string{
				"visibility",
				"hidden_stem_tier",
				"root_support",
				"season_support",
				"structural_closure",
				"counter_evidence",
			},
		}},
	}
}

func projectCanonicalDayunJudgments(state baziCharterState, c baziCanonicalSynthesis) []baziDayunJudgment {
	periods := dayunPeriods(state.Input.Dayun)
	if len(periods) == 0 {
		return nil
	}
	periodIndexByGanZhi := make(map[string]int, len(periods))
	for i, period := range periods {
		if ganZhi := strings.TrimSpace(stringValue(period["ganZhi"])); ganZhi != "" {
			periodIndexByGanZhi[ganZhi] = i
		}
	}
	byIndex := map[int]baziCanonicalDayunUnit{}
	for _, unit := range c.DayunPeriods {
		if strings.TrimSpace(unit.Verdict) == "" {
			continue
		}
		index := -1
		if ganZhi := strings.TrimSpace(unit.GanZhi); ganZhi != "" {
			if matched, ok := periodIndexByGanZhi[ganZhi]; ok {
				index = matched
			}
		}
		if index < 0 && unit.Index != nil {
			index = *unit.Index
		}
		if index >= 0 && index < len(periods) {
			byIndex[index] = unit
		}
	}
	if len(byIndex) == 0 {
		return nil
	}
	out := make([]baziDayunJudgment, 0, len(byIndex))
	for i, period := range periods {
		ganZhi := strings.TrimSpace(stringValue(period["ganZhi"]))
		unit, ok := byIndex[i]
		if !ok {
			continue
		}
		out = append(out, baziDayunJudgment{
			GanZhi:         firstNonEmptyTrim(unit.GanZhi, ganZhi),
			Trend:          firstNonEmptyTrim(unit.Verdict, "仅作结构观察"),
			Interpretation: firstNonEmptyTrim(unit.Boundary, "只解释结构触发，不展开具体生活事件。"),
			Evidence:       canonicalDayunEvidence(period),
			OutcomeDomains: canonicalOutcomeDomains(state),
		})
	}
	return out
}

// canonicalDayunEvidence projects provenance paths into calculated, user-readable period facts.
func canonicalDayunEvidence(period map[string]any) []string {
	facts := []string{}
	if ganZhi := strings.TrimSpace(stringValue(period["ganZhi"])); ganZhi != "" {
		facts = append(facts, "本运干支为"+ganZhi)
	}
	if age := strings.TrimSpace(stringValue(period["ageRange"])); age != "" {
		facts = append(facts, "适用年龄约为"+age)
	}
	facts = append(facts, relationTextList(period["dayun_chonghe"])...)
	if len(facts) > 0 {
		return uniqueText(facts)
	}
	return []string{"本运仅展示已计算结构事实。"}
}

func mergeCanonicalDayunPath(facts []string, judgments []baziDayunJudgment) []string {
	if len(facts) == 0 {
		return renderDayunJudgmentLines(judgments)
	}
	out := append([]string{}, facts...)
	for _, judgment := range judgments {
		index := -1
		for candidateIndex, line := range out {
			if strings.HasPrefix(periodHeadline(line), strings.TrimSpace(judgment.GanZhi)+"运") {
				index = candidateIndex
				break
			}
		}
		if index < 0 || strings.TrimSpace(judgment.Interpretation) == "" {
			continue
		}
		out[index] = strings.TrimSpace(out[index]) + "\n- **当前承接**：" + judgment.Trend + "；" + judgment.Interpretation
	}
	return out
}

func canonicalConfidence(value string) string {
	switch strings.TrimSpace(value) {
	case "明确成立", "倾向成立", "封顶判断":
		if strings.TrimSpace(value) == "封顶判断" {
			return "明确成立"
		}
		return strings.TrimSpace(value)
	default:
		return "保守判断"
	}
}

func canonicalSupportLevel(confidence string) string {
	switch canonicalConfidence(confidence) {
	case "明确成立":
		return "有气"
	case "倾向成立":
		return "有根"
	default:
		return "出现"
	}
}

func canonicalLimitationLevel(text string) string {
	if containsAnyText([]string{text}, []string{"核心", "明显", "不足", "受限", "缺失", "暂不"}) {
		return "明显"
	}
	return "轻微"
}

func canonicalWordingCap(confidence string) string {
	switch canonicalConfidence(confidence) {
	case "明确成立":
		return "明确"
	case "倾向成立":
		return "中性"
	default:
		return "保守"
	}
}

func canonicalStrengthConclusion(text string) string {
	for _, allowed := range []string{"中和偏强", "中和偏弱", "中和附近", "偏强", "偏弱"} {
		if strings.Contains(text, allowed) {
			return allowed
		}
	}
	return "中和附近"
}

func canonicalWindowLevel(text string) string {
	switch {
	case strings.Contains(text, "转折"):
		return "转折年"
	case strings.Contains(text, "承压"):
		return "承压年"
	case strings.Contains(text, "窗口"):
		return "窗口年"
	default:
		return "扰动年"
	}
}

func canonicalOutcomeDomains(state baziCharterState) []string {
	ctx := buildBaziSubjectContext(state.Input)
	if containsString(ctx.AllowedOutcomeDomains, "structure") {
		return []string{"structure"}
	}
	return []string{"structure"}
}

func canonicalTriggerSignals(state baziCharterState, c baziCanonicalSynthesis) []string {
	items := []string{}
	if index := currentDayunIndexForInput(state.Input); index >= 0 {
		periods := dayunPeriods(state.Input.Dayun)
		if index < len(periods) {
			items = append(items, relationTextList(periods[index]["dayun_chonghe"])...)
		}
	}
	items = append(items, relationTextList(state.Input.Liunian["liunian_chonghe"])...)
	return canonicalListOrDefault(items, []string{"本轮仅按已计算大运、流年和关系事实观察。"})
}

func canonicalListOrDefault(items, fallback []string) []string {
	items = filterNonEmpty(items)
	if len(items) > 0 {
		return items
	}
	return filterNonEmpty(fallback)
}

func canonicalFailureFactsOnly(state baziCharterState, err error, auditCode, fallback string) (baziStaticSynthesis, baziDynamicSynthesis) {
	reason := recoveryReasonText(err, fallback)
	static := buildFactsOnlyStaticSynthesis(state.Input, reason)
	dynamic := buildFactsOnlyDynamicSynthesis(state.Input, static, reason)
	// Facts-only output is runtime-generated after model text is discarded, so
	// the display contract is clean even though the recovery reason remains
	// visible in FieldAudit and trace attributes.
	static.ContractAudit = baziContractAudit{Compliant: true}
	dynamic.ContractAudit = baziContractAudit{Compliant: true}
	static.FieldAudit = append(static.FieldAudit, auditCode)
	dynamic.FieldAudit = append(dynamic.FieldAudit, auditCode)
	return static, dynamic
}

func canonicalDynamicFailureFactsOnly(state baziCharterState, static baziStaticSynthesis, err error) baziDynamicSynthesis {
	reason := recoveryReasonText(err, "最小裁断动态投影越过授权领域，已降级展示可复算大运与流年事实。")
	dynamic := buildFactsOnlyDynamicSynthesis(state.Input, static, reason)
	dynamic.ContractAudit = baziContractAudit{Compliant: true}
	dynamic.FieldAudit = append(dynamic.FieldAudit, "canonical_dynamic_projection_facts_only")
	if failure, ok := baziContractFailureFromError("dynamic_projection", err); ok {
		dynamic.FieldAudit = append(dynamic.FieldAudit,
			"contract_failure_class:"+failure.Class,
			"recovery_policy:"+failure.RecoveryPolicy,
		)
	}
	return dynamic
}
