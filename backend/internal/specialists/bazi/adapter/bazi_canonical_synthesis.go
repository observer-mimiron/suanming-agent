// package adapter 包含 Manager 拥有的八字 canonical synthesis 合同。
//
// 本文件负责静态/动态综合模型调用、claim 归一和字段级 repair 输入；
// 不负责 renderer 兼容投影、最终答复权或确定性排盘事实。
package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/repair"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	baziapplication "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/application"
	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
)

// runStaticSynthesis 将 canonical 共同裁断扩展为静态 claim 单元。
// 模型只能写 Schema DTO，legacy 展示字段仍由本文件的投影函数生成。
func (e *Executor) runStaticSynthesis(ctx context.Context, view *specialists.SessionView, chartState baziCharterState, canonical baziCanonicalSynthesis, question string) (baziCanonicalSynthesis, error) {
	payload := baziapplication.BuildStaticSynthesisPayload(chartState)
	if strings.TrimSpace(canonical.MainAxis.Verdict) != "" {
		payload["canonical_synthesis"] = canonical
	}
	payload["rejected_candidate"] = canonical
	payload["runtime_catalog"] = baziStaticRuntimeCatalogView(chartState)
	out, err := runBaziInnerAgentJSON[baziStructuredStaticSynthesis](ctx, e.builder, baziStaticSynthesisConfig(), view, buildBaziCharterPrompt("静态综合", question, payload))
	if err != nil {
		return baziCanonicalSynthesis{}, err
	}
	out.Claims, err = bazidomain.NormalizeStaticClaims(out.Claims)
	if err != nil {
		return baziCanonicalSynthesis{}, err
	}
	assertions := structuredStaticClaims(chartState, out.Claims)
	if err := bazidomain.ValidateStaticReferenceCatalog(chartState, assertions); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := bazidomain.ValidateStaticJudgment(chartState, out); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	return applyStaticClaims(chartState, canonical, out)
}

// runStaticSynthesisRepair 仅重跑失败的静态 claim 节点，并附带字段级反馈。
// 它复用同一份 Schema 与 runtime catalog，不允许通过修改确定性事实绕过合同。
func (e *Executor) runStaticSynthesisRepair(ctx context.Context, view *specialists.SessionView, chartState baziCharterState, canonical baziCanonicalSynthesis, question string, feedback map[string]any) (baziCanonicalSynthesis, error) {
	payload := baziapplication.BuildStaticSynthesisPayload(chartState)
	if strings.TrimSpace(canonical.MainAxis.Verdict) != "" {
		payload["canonical_synthesis"] = canonical
	}
	payload["rejected_candidate"] = canonical
	payload["runtime_catalog"] = baziStaticRuntimeCatalogView(chartState)
	payload["validation_feedback"] = feedback
	out, err := runBaziInnerAgentJSON[baziStructuredStaticSynthesis](ctx, e.builder, baziStaticSynthesisConfig(), view, buildBaziCharterPrompt("静态综合修复", question, payload))
	if err != nil {
		return baziCanonicalSynthesis{}, err
	}
	out.Claims, err = bazidomain.NormalizeStaticClaims(out.Claims)
	if err != nil {
		return baziCanonicalSynthesis{}, err
	}
	assertions := structuredStaticClaims(chartState, out.Claims)
	if err := bazidomain.ValidateStaticReferenceCatalog(chartState, assertions); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := bazidomain.ValidateStaticJudgment(chartState, out); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	return applyStaticClaims(chartState, canonical, out)
}

// runDynamicSynthesis 将已通过静态校验的主轴扩展为逐运与流年 claim。
// 该节点不重判静态结构，也不能输出确定性大运事实或 renderer 字段。
func (e *Executor) runDynamicSynthesis(ctx context.Context, view *specialists.SessionView, chartState baziCharterState, canonical baziCanonicalSynthesis, question string) (baziCanonicalSynthesis, error) {
	if err := bazidomain.ValidateDynamicPreconditions(chartState); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	payload := baziapplication.BuildDynamicSynthesisPayload(chartState)
	payload["runtime_catalog"] = baziDynamicRuntimeCatalogView(chartState)
	out, err := runBaziInnerAgentJSON[baziStructuredDynamicSynthesis](ctx, e.builder, baziDynamicSynthesisConfig(), view, buildBaziCharterPrompt("动态综合", question, payload))
	if err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := validateDynamicPeriodClaims(chartState, out.PeriodClaims); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := bazidomain.ValidateDynamicJudgment(chartState, out); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	claims := structuredDynamicClaims(chartState, out)
	if err := bazidomain.ValidateDynamicReferenceCatalog(chartState, claims); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	return applyDynamicClaims(chartState, canonical, out)
}

// runDynamicSynthesisRepair 只重跑失败的动态节点，并复用同一份动态 Schema 与引用目录。
// 它不重判静态主轴，也不把动态引用错误交给 canonical 节点处理。
func (e *Executor) runDynamicSynthesisRepair(ctx context.Context, view *specialists.SessionView, chartState baziCharterState, canonical baziCanonicalSynthesis, question string, feedback map[string]any) (baziCanonicalSynthesis, error) {
	if err := bazidomain.ValidateDynamicPreconditions(chartState); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	payload := baziapplication.BuildDynamicSynthesisPayload(chartState)
	payload["runtime_catalog"] = baziDynamicRuntimeCatalogView(chartState)
	payload["rejected_candidate"] = canonical
	payload["validation_feedback"] = feedback
	out, err := runBaziInnerAgentJSON[baziStructuredDynamicSynthesis](ctx, e.builder, baziDynamicSynthesisConfig(), view, buildBaziCharterPrompt("动态综合修复", question, payload))
	if err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := validateDynamicPeriodClaims(chartState, out.PeriodClaims); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	if err := bazidomain.ValidateDynamicJudgment(chartState, out); err != nil {
		return baziCanonicalSynthesis{}, err
	}
	claims := structuredDynamicClaims(chartState, out)
	if err := bazidomain.ValidateDynamicReferenceCatalog(chartState, claims); err != nil {
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
	canonical.PatternName = strings.TrimSpace(output.PatternName)
	canonical.PatternRoute = strings.TrimSpace(output.PatternRoute)
	canonical.PatternEvaluation = strings.TrimSpace(output.PatternEvaluation)
	canonical.Limitations = staticRuntimeLimitations(state)
	canonical.StaticReasoningSummary = staticRuntimeReasoningSummary(state)
	canonical.ReasoningSteps = staticRuntimeReasoningSteps()
	canonical.AdviceBoundary = staticRuntimeAdviceBoundary()
	return canonical, nil
}

// applyDynamicClaims 将动态 DTO 按确定性大运顺序映射为 canonical 动态单元。
func applyDynamicClaims(state baziCharterState, canonical baziCanonicalSynthesis, output baziStructuredDynamicSynthesis) (baziCanonicalSynthesis, error) {
	periods := bazidomain.DayunPeriods(state.Input.Dayun)
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
	// 限制、推理步骤和年龄授权范围由 runtime 根据事实胶囊投影，
	// 不再要求动态模型重复填写这些高漂移字段。
	return canonical, nil
}

// dynamicRuntimeClaimBoundary owns dynamic display limits. The model no longer
// authors boundary text; runtime supplies it so limits cannot leak catalog terms.
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
		periodIndex, _ := dynamicPeriodIndex(claim.PeriodRef, bazidomain.DayunPeriods(state.Input.Dayun))
		out = append(out, structuredClaimAssertion(structuredClaimFromPeriodClaim(claim), baziAssertionDayunPeriod, fmt.Sprintf("dayun.%d", periodIndex), fmt.Sprintf("dynamic.dayun.%d", periodIndex)))
	}
	out = append(out, structuredClaimAssertion(output.LiunianClaim, baziAssertionLiunian, "liunian", "dynamic.liunian"))
	return out
}

// structuredClaimFromPeriodClaim removes the period selector before shared reference validation.
func structuredClaimFromPeriodClaim(claim baziStructuredPeriodClaim) baziStructuredClaim {
	return baziStructuredClaim{
		Verdict: claim.Verdict, FactRefs: claim.FactRefs, RelationRefs: claim.RelationRefs,
		ClaimRefs: claim.ClaimRefs, EvidenceTopics: claim.EvidenceTopics, Confidence: claim.Confidence,
	}
}

// validateDynamicPeriodClaims checks model-selected periods against the current deterministic catalog.
func validateDynamicPeriodClaims(state baziCharterState, claims []baziStructuredPeriodClaim) error {
	periods := bazidomain.DayunPeriods(state.Input.Dayun)
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
	return baziAssertion{ID: id, Kind: kind, Subject: subject, Verdict: claim.Verdict, FactRefs: claim.FactRefs, RelationRefs: claim.RelationRefs, ClaimRefs: claim.ClaimRefs, EvidenceTopics: claim.EvidenceTopics, Confidence: claim.Confidence, Boundary: dynamicRuntimeClaimBoundary(kind)}
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
		Kind: string(kind), Verdict: claim.Verdict, Boundary: dynamicRuntimeClaimBoundary(kind),
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
		return "调候按月令、全局寒暖燥湿与已覆盖材料判断；火透或有根只是条件，不能单独定效。"
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
		limits = append(limits, "调候力度按月令与已覆盖材料确认；火透或有根不能单独定效。")
	case !capsule.FireEffective:
		limits = append(limits, "已有材料显示火不足以单独作为调候依据。")
	}
	return baziapplication.CanonicalListOrDefault(limits, []string{"静态层只解释本命结构；岁运承接与条件风险由动态层单独处理。"})
}

// staticRuntimeReasoningSummary states the fixed static decision basis without
// repeating the model's main-axis wording in another renderer section.
func staticRuntimeReasoningSummary(state baziCharterState) string {
	return "静态裁断按月令、通根、透干、受力、官星透藏和调候资格投影。"
}

// staticRuntimeReasoningSteps returns the stable order used by the legacy
// renderer. These steps describe the contract and never add a chart conclusion.
func staticRuntimeReasoningSteps() []string {
	return []string{
		"先核对月令、通根、透干、印比与泄耗克身的已计算事实。",
		"再在事实与已覆盖规则材料范围内裁断主轴、强弱、调候和格局。",
		"最后收束格局取用；当前大运只解释承接，不改写本命结构。",
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
func buildBaziCanonicalRepairFeedback(failure repair.Failure, attempt int) map[string]any {
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
