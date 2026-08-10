// Package runtime 包含 Manager 拥有的八字证据阶段。
//
// 本文件负责证据规划、受控检索、引用归并和有限补证；
// 不负责图拓扑、静态/动态合同校验或最终答复渲染。
package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/observer-mimiron/suanming-agent/internal/mcp"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

// runBaziEvidenceStage 规划并执行一轮八字证据检索，再计算覆盖质量。
func (e *Executor) runBaziEvidenceStage(ctx context.Context, st *state.SessionState, question string, chartFacts baziCharterInput, analysisPlan baziAnalysisPlan) (baziEvidencePlan, baziEvidenceBundle, baziEvidenceQuality, error) {
	plan, err := e.runBaziEvidencePlanner(ctx, st, question, chartFacts, analysisPlan)
	if err != nil {
		plan = defaultBaziEvidencePlan(question, analysisPlan, chartFacts)
	} else {
		plan = normalizeBaziEvidencePlan(plan, chartFacts, analysisPlan)
	}
	bundle, err := e.runControlledBaziRetrieval(ctx, plan)
	if err != nil {
		return plan, baziEvidenceBundle{}, baziEvidenceQuality{}, err
	}
	quality := evaluateEvidenceBundleQuality(plan, bundle)
	return plan, bundle, quality, nil
}

// runBaziEvidencePlanner 让受限模型为当前分析阶段生成证据查询计划。
func (e *Executor) runBaziEvidencePlanner(ctx context.Context, st *state.SessionState, question string, chartFacts baziCharterInput, analysisPlan baziAnalysisPlan) (baziEvidencePlan, error) {
	stage := analysisPlan.RetrievalStage
	if strings.TrimSpace(stage) == "" {
		stage = "static"
	}
	payload := buildEvidencePlannerPayload(question, chartFacts, analysisPlan)
	return runBaziInnerAgentJSON[baziEvidencePlan](ctx, e.builder, baziEvidencePlannerConfig(), st, buildBaziCharterPrompt("证据规划", question, payload))
}

// defaultBaziEvidencePlan 在规划模型失败时生成固定的保守查询计划。
func defaultBaziEvidencePlan(question string, analysisPlan baziAnalysisPlan, chartFacts ...baziCharterInput) baziEvidencePlan {
	stage := analysisPlan.RetrievalStage
	if strings.TrimSpace(stage) == "" {
		stage = "static"
	}
	var input baziCharterInput
	if len(chartFacts) > 0 {
		input = chartFacts[0]
	}
	sources := stageAuthoritySources(stage)
	plan := baziEvidencePlan{
		NeedRetrieval:     true,
		Stage:             stage,
		RecommendedSource: append([]string{}, sources.Primary...),
		AllowReflection:   true,
	}
	if stage == "dynamic" {
		plan.EvidenceGaps = []string{"当前大运如何兑现静态主轴", "目标流年触发点补证"}
		plan.QueryPackets = []baziQueryPacket{
			{
				Topic:            "dayun",
				Query:            "三命通会 大运 行运 岁运并临",
				PreferredSources: []string{"三命通会", "滴天髓"},
				SourceTier:       "A",
			},
			{
				Topic:            "liunian",
				Query:            "三命通会 流年 应期 太岁",
				PreferredSources: []string{"三命通会", "子平真诠"},
				SourceTier:       "A",
			},
		}
		return plan
	}

	plan.EvidenceGaps = []string{"取格依据补证", "调候与格局交叉验证", "清浊依据补证", "病药依据补证", "救应依据补证", "破格风险补证", "何知章印证补证", "同结构命例校验补证"}
	plan.QueryPackets = []baziQueryPacket{
		{
			Topic:            "geju",
			Query:            "子平真诠 格局 月令 取格",
			PreferredSources: []string{"子平真诠", "渊海子平"},
			SourceTier:       "A",
		},
		{
			Topic:            "tiaohou",
			Query:            buildTiaohouEvidenceQuery(input),
			PreferredSources: []string{"穷通宝鉴", "滴天髓"},
			SourceTier:       "A",
		},
		{
			Topic:            "qingzhuo",
			Query:            "子平真诠 清浊 纯杂",
			PreferredSources: []string{"子平真诠", "渊海子平"},
			SourceTier:       "A",
		},
		{
			Topic:            "bingyao",
			Query:            "滴天髓 病药 制化",
			PreferredSources: []string{"滴天髓", "子平真诠"},
			SourceTier:       "A",
		},
		{
			Topic:            "jiuying",
			Query:            "滴天髓 救应 制化",
			PreferredSources: []string{"滴天髓", "子平真诠"},
			SourceTier:       "A",
		},
		{
			Topic:            "poge",
			Query:            "子平真诠 破格 败格",
			PreferredSources: []string{"子平真诠", "渊海子平"},
			SourceTier:       "A",
		},
		{
			Topic:            "hezhizhang",
			Query:            "滴天髓 何知章 富贵贫贱吉凶",
			PreferredSources: []string{"滴天髓", "子平真诠"},
			SourceTier:       "A",
		},
		{
			Topic:            "geju",
			Query:            "子平真诠 格局 命例 举例",
			PreferredSources: []string{"子平真诠", "格局论命"},
			SourceTier:       "B",
		},
	}
	return normalizeBaziEvidencePlan(plan, input, analysisPlan)
}

// normalizeBaziEvidencePlan keeps model-planned retrieval useful for the
// downstream quality gate. 九级层次需要独立的清浊、病药、救应、破格和
// 何知章主证，因此由 runtime 补齐不可省略的查询包，不能依赖 planner 恰好选中它们。
func normalizeBaziEvidencePlan(plan baziEvidencePlan, chartFacts baziCharterInput, analysisPlan baziAnalysisPlan) baziEvidencePlan {
	if strings.TrimSpace(plan.Stage) == "" {
		plan.Stage = firstNonEmptyTrim(analysisPlan.RetrievalStage, "static")
	}
	if plan.Stage != "static" {
		return plan
	}
	hasTiaohou := false
	for i := range plan.QueryPackets {
		packet := &plan.QueryPackets[i]
		if packet.Topic != "tiaohou" || !strings.EqualFold(strings.TrimSpace(packet.SourceTier), "A") {
			continue
		}
		hasTiaohou = true
		if !queryContainsTiaohouChartTerms(packet.Query, chartFacts) {
			packet.Query = buildTiaohouEvidenceQuery(chartFacts)
		}
		packet.PreferredSources = mergeStrings(packet.PreferredSources, "穷通宝鉴", "滴天髓")
	}
	if !hasTiaohou {
		plan.QueryPackets = append(plan.QueryPackets, baziQueryPacket{
			Topic:            "tiaohou",
			Query:            buildTiaohouEvidenceQuery(chartFacts),
			PreferredSources: []string{"穷通宝鉴", "滴天髓"},
			SourceTier:       "A",
		})
	}
	for _, packet := range []baziQueryPacket{
		{Topic: "qingzhuo", Query: "子平真诠 清浊 纯杂", PreferredSources: []string{"子平真诠", "渊海子平"}, SourceTier: "A"},
		{Topic: "bingyao", Query: "滴天髓 病药 制化", PreferredSources: []string{"滴天髓", "子平真诠"}, SourceTier: "A"},
		{Topic: "jiuying", Query: "滴天髓 救应 制化", PreferredSources: []string{"滴天髓", "子平真诠"}, SourceTier: "A"},
		{Topic: "poge", Query: "子平真诠 破格 败格", PreferredSources: []string{"子平真诠", "渊海子平"}, SourceTier: "A"},
		{Topic: "hezhizhang", Query: "滴天髓 何知章 富贵贫贱吉凶", PreferredSources: []string{"滴天髓", "子平真诠"}, SourceTier: "A"},
	} {
		if !hasATierEvidenceTopic(plan, packet.Topic) {
			plan.QueryPackets = append(plan.QueryPackets, packet)
		}
	}
	return plan
}

// hasATierEvidenceTopic reports whether a static plan already owns the topic's
// authority query. A B-tier counterexample cannot satisfy a tier qualification.
func hasATierEvidenceTopic(plan baziEvidencePlan, topic string) bool {
	for _, packet := range plan.QueryPackets {
		if packet.Topic == topic && strings.EqualFold(strings.TrimSpace(packet.SourceTier), "A") {
			return true
		}
	}
	return false
}

// buildTiaohouEvidenceQuery derives a concrete 穷通 query from the calculated
// day master and month branch. This makes evidence coverage deterministic across
// runs instead of depending on the planner to remember the exact chart terms.
func buildTiaohouEvidenceQuery(input baziCharterInput) string {
	terms := []string{"穷通宝鉴"}
	if day := dayMasterForEvidenceQuery(input); day != "" {
		terms = append(terms, day)
	}
	if month := monthBranchForEvidenceQuery(input); month != "" {
		terms = append(terms, month+"月")
		if label := baziMonthBranchLabel(month); label != "" {
			terms = append(terms, label)
			if day := dayMasterForEvidenceQuery(input); day != "" {
				terms = append(terms, label+day)
			}
		}
		if day := dayMasterForEvidenceQuery(input); day != "" {
			terms = append(terms, month+"月"+day)
		}
	}
	terms = append(terms, "调候")
	if len(terms) <= 2 {
		terms = append(terms, "月令", "寒暖燥湿")
	}
	return strings.Join(terms, " ")
}

// queryContainsTiaohouChartTerms 检查调候查询是否绑定当前命盘的日主和月令。
func queryContainsTiaohouChartTerms(query string, input baziCharterInput) bool {
	day := dayMasterForEvidenceQuery(input)
	month := monthBranchForEvidenceQuery(input)
	if day != "" && !strings.Contains(query, day) {
		return false
	}
	if month != "" {
		monthTerms := []string{month + "月"}
		if label := baziMonthBranchLabel(month); label != "" {
			monthTerms = append(monthTerms, label)
		}
		if !containsAnyText([]string{query}, monthTerms) {
			return false
		}
		if day != "" {
			specificMonthTerms := []string{month + "月" + day}
			if label := baziMonthBranchLabel(month); label != "" {
				specificMonthTerms = append(specificMonthTerms, label+day)
			}
			if !containsAnyText([]string{query}, specificMonthTerms) {
				return false
			}
		}
	}
	return day != "" || month != ""
}

// baziMonthBranchLabel adds the traditional month name used by classics such
// as 穷通宝鉴, where 亥月 is usually indexed as 十月.
func baziMonthBranchLabel(branch string) string {
	switch strings.TrimSpace(branch) {
	case "寅":
		return "正月"
	case "卯":
		return "二月"
	case "辰":
		return "三月"
	case "巳":
		return "四月"
	case "午":
		return "五月"
	case "未":
		return "六月"
	case "申":
		return "七月"
	case "酉":
		return "八月"
	case "戌":
		return "九月"
	case "亥":
		return "十月"
	case "子":
		return "十一月"
	case "丑":
		return "十二月"
	default:
		return ""
	}
}

// dayMasterForEvidenceQuery 将日主事实转换为古籍检索使用的五行标签。
func dayMasterForEvidenceQuery(input baziCharterInput) string {
	day := firstNonEmptyTrim(stringValue(input.BaziResult["dayGan"]), stringValue(input.Yongshen["day_master"]))
	switch day {
	case "甲", "乙":
		return day + "木"
	case "丙", "丁":
		return day + "火"
	case "戊", "己":
		return day + "土"
	case "庚", "辛":
		return day + "金"
	case "壬", "癸":
		return day + "水"
	default:
		return strings.TrimSpace(day)
	}
}

// monthBranchForEvidenceQuery 从月柱或扶抑事实中读取月令地支。
func monthBranchForEvidenceQuery(input baziCharterInput) string {
	if pillar := extractMonthPillar(input.BaziResult["pillars"]); len(pillar) > 0 {
		if branch := stringValue(pillar["branch"]); branch != "" {
			return branch
		}
	}
	return firstNonEmptyTrim(stringValue(input.Yongshen["month_branch"]), stringValue(input.Yongshen["month_zhi"]))
}

// runControlledBaziRetrieval 并发执行固定查询包并按主题归并检索结果。
func (e *Executor) runControlledBaziRetrieval(ctx context.Context, plan baziEvidencePlan) (baziEvidenceBundle, error) {
	bundle := baziEvidenceBundle{
		Stage:                plan.Stage,
		TopicBuckets:         map[string][]baziCitation{},
		CriticalTopicBuckets: map[string][]baziCitation{},
	}
	if !plan.NeedRetrieval || len(plan.QueryPackets) == 0 {
		return bundle, nil
	}
	if e == nil || e.builder == nil || e.builder.reg == nil {
		return bundle, fmt.Errorf("knowledge search registry unavailable")
	}

	searchTool, ok := e.builder.reg.Get("knowledge_search")
	if !ok {
		return bundle, fmt.Errorf("knowledge_search tool not registered")
	}

	type retrievalResult struct {
		index     int
		packet    baziQueryPacket
		citations []baziCitation
		degraded  bool
		err       error
	}
	results := make([]retrievalResult, len(plan.QueryPackets))
	var wg sync.WaitGroup
	for i, packet := range plan.QueryPackets {
		wg.Add(1)
		go func(index int, packet baziQueryPacket) {
			defer wg.Done()
			retrievalSpan := tracing.SpanFromContext(ctx, "knowledge_search", tracing.KindRetriever)
			retrievalSpan.SetAttribute("query", packet.Query)
			retrievalSpan.SetAttribute("topic", packet.Topic)
			retrievalSpan.SetAttribute("source_tier", packet.SourceTier)
			defer retrievalSpan.End()
			result, err := searchTool.Execute(ctx, map[string]any{
				"query": packet.Query,
				"top_k": float64(3),
			})
			if err != nil {
				retrievalSpan.RecordError(err)
				results[index] = retrievalResult{index: index, packet: packet, err: err}
				return
			}
			citations := citationsFromKnowledgeResult(result, packet)
			degraded := knowledgeResultDegraded(result)
			retrievalSpan.SetAttribute("hits", len(citations))
			retrievalSpan.SetAttribute("degraded", degraded)
			results[index] = retrievalResult{
				index:     index,
				packet:    packet,
				citations: citations,
				degraded:  degraded,
			}
		}(i, packet)
	}
	wg.Wait()
	for _, result := range results {
		if result.err != nil {
			return bundle, result.err
		}
		if len(result.citations) == 0 {
			if result.degraded {
				bundle.DegradedTopics = mergeStrings(bundle.DegradedTopics, result.packet.Topic)
			}
			continue
		}
		bundle.TopicBuckets[result.packet.Topic] = mergeCitations(bundle.TopicBuckets[result.packet.Topic], result.citations...)
		if strings.EqualFold(strings.TrimSpace(result.packet.SourceTier), "A") {
			bundle.CriticalTopicBuckets[result.packet.Topic] = mergeCitations(bundle.CriticalTopicBuckets[result.packet.Topic], result.citations...)
		}
		bundle.Citations = mergeCitations(bundle.Citations, result.citations...)
	}

	return bundle, nil
}

// knowledgeResultDegraded exposes a tool-level fallback so missing evidence is
// distinguishable from a successful empty semantic search.
func knowledgeResultDegraded(result any) bool {
	rm, ok := result.(map[string]any)
	if !ok {
		return false
	}
	degraded, _ := rm["fallback"].(bool)
	return degraded
}

// hasDynamicSystemFacts reports whether deterministic dayun and liunian facts are both ready.
func hasDynamicSystemFacts(input baziCharterInput) bool {
	return len(input.Dayun) > 0 && len(input.Liunian) > 0
}

// citationsFromKnowledgeResult 将知识库返回的 passages 转成引用结构。
func citationsFromKnowledgeResult(result any, packet baziQueryPacket) []baziCitation {
	rm, ok := result.(map[string]any)
	if !ok {
		return nil
	}
	rawPassages, ok := rm["passages"]
	if !ok || rawPassages == nil {
		return nil
	}

	var out []baziCitation
	switch passages := rawPassages.(type) {
	case []mcp.Passage:
		for _, passage := range passages {
			out = mergeCitations(out, citationFromPassage(passage.Source, passage.Content, packet))
		}
	case []any:
		for _, raw := range passages {
			pm, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			source, _ := pm["source"].(string)
			content, _ := pm["content"].(string)
			out = mergeCitations(out, citationFromPassage(source, content, packet))
		}
	}

	if preferred := filterPreferredCitations(out, packet.PreferredSources); len(preferred) > 0 {
		return preferred
	}
	return out
}

// filterPreferredCitations 优先保留查询计划声明的权威来源。
func filterPreferredCitations(items []baziCitation, preferredSources []string) []baziCitation {
	if len(items) == 0 || len(preferredSources) == 0 {
		return nil
	}
	var filtered []baziCitation
	for _, item := range items {
		if containsString(preferredSources, item.Classic) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// citationFromPassage 将单条检索片段映射为可合并的古籍引用。
func citationFromPassage(source, content string, packet baziQueryPacket) baziCitation {
	classic := extractAuthorityClassic(source)
	if classic == "" {
		classic = source
	}
	return baziCitation{
		Classic: classic,
		Quotes:  []string{strings.TrimSpace(content)},
	}
}

// extractAuthorityClassic 从来源标识中识别规范古籍名称。
func extractAuthorityClassic(source string) string {
	if classic := extractAuthorityClassicFromSlug(source); classic != "" {
		return classic
	}
	for _, classic := range allAuthorityClassicNames() {
		if strings.Contains(source, classic) {
			return classic
		}
	}
	return ""
}

// extractAuthorityClassicFromSlug maps local wiki slugs back to canonical
// classics. The retrieval API returns sources like
// knowledge://ref-bazi-qiongtong-s001 (五行总论), whose title alone does not name
// 穷通宝鉴; without this map real 调候 hits are misclassified as non-authority.
func extractAuthorityClassicFromSlug(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	slugMap := []struct {
		needle  string
		classic string
	}{
		{needle: "ref-bazi-qiongtong", classic: "穷通宝鉴"},
		{needle: "ref-bazi-ditiansui", classic: "滴天髓"},
		{needle: "ref-bazi-ziping", classic: "子平真诠"},
		{needle: "ref-bazi-yuanhai", classic: "渊海子平"},
		{needle: "ref-bazi-sanming", classic: "三命通会"},
		{needle: "ref-bazi-gelulunming", classic: "格局论命"},
	}
	for _, item := range slugMap {
		if strings.Contains(source, item.needle) {
			return item.classic
		}
	}
	return ""
}

// allAuthorityClassicNames returns the canonical source names accepted by both stages.
func allAuthorityClassicNames() []string {
	static := stageAuthoritySources("static")
	dynamic := stageAuthoritySources("dynamic")

	var names []string
	for _, bucket := range [][]string{static.Primary, static.Secondary, dynamic.Primary, dynamic.Secondary} {
		for _, name := range bucket {
			if !containsString(names, name) {
				names = append(names, name)
			}
		}
	}
	return names
}

// mergeCitations 按古籍去重并合并引用片段。
func mergeCitations(base []baziCitation, adds ...baziCitation) []baziCitation {
	merged := append([]baziCitation{}, base...)
	for _, add := range adds {
		if strings.TrimSpace(add.Classic) == "" {
			continue
		}
		found := false
		for i := range merged {
			if merged[i].Classic != add.Classic {
				continue
			}
			merged[i].Quotes = mergeStrings(merged[i].Quotes, add.Quotes...)
			found = true
			break
		}
		if !found {
			merged = append(merged, add)
		}
	}
	return merged
}

// mergeEvidenceBundles 合并首轮与补证检索的主题、引用和降级状态。
func mergeEvidenceBundles(base, add baziEvidenceBundle) baziEvidenceBundle {
	merged := baziEvidenceBundle{
		Stage:                base.Stage,
		TopicBuckets:         map[string][]baziCitation{},
		CriticalTopicBuckets: map[string][]baziCitation{},
		Citations:            mergeCitations(base.Citations, add.Citations...),
		Conflicts:            mergeStrings(base.Conflicts, add.Conflicts...),
		DegradedTopics:       mergeStrings(base.DegradedTopics, add.DegradedTopics...),
	}
	if strings.TrimSpace(merged.Stage) == "" {
		merged.Stage = add.Stage
	}
	for topic, citations := range base.TopicBuckets {
		merged.TopicBuckets[topic] = mergeCitations(merged.TopicBuckets[topic], citations...)
	}
	for topic, citations := range add.TopicBuckets {
		merged.TopicBuckets[topic] = mergeCitations(merged.TopicBuckets[topic], citations...)
	}
	for topic, citations := range base.CriticalTopicBuckets {
		merged.CriticalTopicBuckets[topic] = mergeCitations(merged.CriticalTopicBuckets[topic], citations...)
	}
	for topic, citations := range add.CriticalTopicBuckets {
		merged.CriticalTopicBuckets[topic] = mergeCitations(merged.CriticalTopicBuckets[topic], citations...)
	}
	return merged
}

// mergeStrings 对字符串切片做去空、去重和稳定追加。
func mergeStrings(base []string, adds ...string) []string {
	merged := append([]string{}, base...)
	for _, add := range adds {
		add = strings.TrimSpace(add)
		if add == "" || containsString(merged, add) {
			continue
		}
		merged = append(merged, add)
	}
	return merged
}

// maybeReflectOnBaziEvidence 仅对缺失或高冲突主题执行一次受控补证。
func (e *Executor) maybeReflectOnBaziEvidence(ctx context.Context, st *state.SessionState, chartState baziCharterState) (baziCharterState, error) {
	if !chartState.EvidencePlan.AllowReflection {
		return chartState, nil
	}
	if !shouldReflectOnEvidence(chartState.EvidenceQuality) {
		return chartState, nil
	}

	retryPlan := buildEvidenceRetryPlan(chartState.EvidencePlan, chartState.EvidenceQuality)
	if !retryPlan.NeedRetrieval || len(retryPlan.QueryPackets) == 0 {
		return chartState, nil
	}
	bundle, err := e.runControlledBaziRetrieval(ctx, retryPlan)
	if err != nil {
		return chartState, err
	}
	chartState.EvidenceBundle = mergeEvidenceBundles(chartState.EvidenceBundle, bundle)
	chartState.EvidenceQuality = evaluateEvidenceBundleQuality(chartState.EvidencePlan, chartState.EvidenceBundle)
	return chartState, nil
}

// buildEvidenceRetryPlan retries only missing A-tier topics with stable source
// and topic terms. Reissuing an unchanged broad plan cannot repair a coverage gap.
func buildEvidenceRetryPlan(plan baziEvidencePlan, quality baziEvidenceQuality) baziEvidencePlan {
	retryTopics := append([]string{}, quality.MissingTopics...)
	if len(retryTopics) == 0 && quality.ConflictScore == "high" {
		retryTopics = requiredEvidenceTopics(plan)
	}
	retry := baziEvidencePlan{
		NeedRetrieval:     len(retryTopics) > 0,
		Stage:             plan.Stage,
		EvidenceGaps:      append([]string{}, retryTopics...),
		RecommendedSource: append([]string{}, plan.RecommendedSource...),
	}
	for _, topic := range retryTopics {
		for _, packet := range plan.QueryPackets {
			if packet.Topic != topic || !strings.EqualFold(strings.TrimSpace(packet.SourceTier), "A") {
				continue
			}
			retryPacket := packet
			retryPacket.Query = strings.TrimSpace(strings.Join(append(append([]string{}, packet.PreferredSources...), packet.Topic), " "))
			retry.QueryPackets = append(retry.QueryPackets, retryPacket)
			break
		}
	}
	return retry
}
