// Package adapter 包含 Manager 拥有的八字证据阶段。
//
// 本文件负责受限多查询、一次补充检索、引用质量过滤和本轮证据归并；
// 不负责图拓扑、静态/动态合同校验或最终答复渲染。
package adapter

import (
	"context"
	"regexp"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/mcp"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	baziapplication "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/application"
	bazidomain "github.com/observer-mimiron/suanming-agent/internal/specialists/bazi/domain"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

const (
	baziEvidenceInitialQueryBudget    = 2
	baziEvidenceSupplementQueryBudget = 1
	baziEvidenceQueryBudget           = baziEvidenceInitialQueryBudget + baziEvidenceSupplementQueryBudget
	baziEvidenceCitationBudget        = 2
)

var baziChapterSourcePattern = regexp.MustCompile(`knowledge://ref-bazi-[a-z]+-s\d{3}\b`)

// runBaziEvidenceStage 让独立规划器改写最多两条初检查询，并在可用检索无原文时允许一次补充查询。
// 古籍是可选证据，所有外部失败都归一为空 bundle。
func (e *Executor) runBaziEvidenceStage(ctx context.Context, view *specialists.SessionView, question string, chartFacts baziCharterInput, analysisPlan baziAnalysisPlan) (baziEvidencePlan, baziEvidenceBundle, baziEvidenceQuality) {
	plan, err := e.runBaziEvidencePlanner(ctx, view, question, chartFacts, analysisPlan)
	if err != nil {
		plan = defaultBaziEvidencePlan(question, analysisPlan, chartFacts)
	} else {
		plan = normalizeBaziEvidencePlan(plan, chartFacts, analysisPlan)
	}
	bundle := e.runControlledBaziRetrieval(ctx, plan, baziEvidenceInitialQueryBudget, baziEvidenceCitationBudget)
	quality := bazidomain.EvaluateEvidenceBundleQuality(plan, bundle)
	if shouldReflectOnBaziEvidence(plan, bundle) {
		supplement := buildEvidenceSupplementPlan(plan, bundle)
		remainingCitations := baziEvidenceCitationBudget - len(bundle.Citations)
		if remainingCitations > 0 {
			bundle = mergeEvidenceBundles(bundle, e.runControlledBaziRetrieval(ctx, supplement, baziEvidenceSupplementQueryBudget, remainingCitations))
			quality = bazidomain.EvaluateEvidenceBundleQuality(plan, bundle)
		}
	}
	return plan, bundle, quality
}

// runBaziEvidencePlanner 让快速模型根据当前问题和确定性命盘事实选择紧凑、按典籍限定的证据查询。
func (e *Executor) runBaziEvidencePlanner(ctx context.Context, view *specialists.SessionView, question string, chartFacts baziCharterInput, analysisPlan baziAnalysisPlan) (baziEvidencePlan, error) {
	payload := baziapplication.BuildEvidencePlannerPayload(question, chartFacts, analysisPlan)
	return runBaziInnerAgentJSON[baziEvidencePlan](ctx, e.builder, baziEvidencePlannerConfig(), view, buildBaziCharterPrompt("证据规划", question, payload))
}

// defaultBaziEvidencePlan 在规划器不可用时保留可用检索，并维持相同的两条初检预算。
func defaultBaziEvidencePlan(question string, analysisPlan baziAnalysisPlan, chartFacts ...baziCharterInput) baziEvidencePlan {
	_ = question
	stage := analysisPlan.RetrievalStage
	if strings.TrimSpace(stage) == "" {
		stage = "static"
	}
	var input baziCharterInput
	if len(chartFacts) > 0 {
		input = chartFacts[0]
	}
	sources := bazidomain.StageAuthoritySources(stage)
	plan := baziEvidencePlan{
		NeedRetrieval:     true,
		AllowReflection:   true,
		Stage:             stage,
		RecommendedSource: append([]string{}, sources.Primary...),
	}
	if stage == "dynamic" {
		plan.EvidenceGaps = []string{"大运解释参考", "流年解释参考"}
		plan.QueryPackets = []baziQueryPacket{{Topic: "dayun", Query: "三命通会 大运 行运 岁运", PreferredSources: []string{"三命通会"}, SourceTier: "A"}, {Topic: "liunian", Query: "三命通会 流年 应期 太岁", PreferredSources: []string{"三命通会"}, SourceTier: "A"}}
		return plan
	}
	if dayMasterForEvidenceQuery(input) != "" && bazidomain.MonthBranchForEvidenceQuery(input) != "" {
		plan.EvidenceGaps = []string{"格局解释参考", "调候解释参考"}
		plan.QueryPackets = []baziQueryPacket{{Topic: "geju", Query: "子平真诠 格局 月令 取格", PreferredSources: []string{"子平真诠"}, SourceTier: "A"}, {Topic: "tiaohou", Query: buildTiaohouEvidenceQuery(input), PreferredSources: []string{"穷通宝鉴"}, SourceTier: "A"}}
		return plan
	}
	plan.EvidenceGaps = []string{"格局解释参考", "扶抑解释参考"}
	plan.QueryPackets = []baziQueryPacket{{Topic: "geju", Query: "子平真诠 格局 月令 取格", PreferredSources: []string{"子平真诠"}, SourceTier: "A"}, {Topic: "fuyi", Query: "滴天髓 扶抑 病药 制化", PreferredSources: []string{"滴天髓"}, SourceTier: "A"}}
	return plan
}

// normalizeBaziEvidencePlan 只接受规划器的受限初检查询；调用方对不合法计划回退为确定性查询。
func normalizeBaziEvidencePlan(plan baziEvidencePlan, chartFacts baziCharterInput, analysisPlan baziAnalysisPlan) baziEvidencePlan {
	if strings.TrimSpace(plan.Stage) == "" {
		plan.Stage = firstNonEmptyTrim(analysisPlan.RetrievalStage, "static")
	}
	if !plan.NeedRetrieval {
		plan.QueryPackets = nil
		return plan
	}
	packets := make([]baziQueryPacket, 0, baziEvidenceInitialQueryBudget)
	for _, packet := range plan.QueryPackets {
		packet.Topic = canonicalBaziEvidenceTopic(packet.Topic)
		packet.Query = strings.TrimSpace(packet.Query)
		if packet.Topic == "" || packet.Query == "" || len([]rune(packet.Query)) > 120 {
			continue
		}
		packets = append(packets, packet)
		if len(packets) == baziEvidenceInitialQueryBudget {
			break
		}
	}
	if len(packets) == 0 {
		return defaultBaziEvidencePlan("", analysisPlan, chartFacts)
	}
	plan.QueryPackets = packets
	plan.NeedRetrieval = true
	return plan
}

// canonicalBaziEvidenceTopic 统一规划器可能输出的中文主题名，保证检索 trace
// 与证据合同使用同一稳定键；主题仍只用于引文审计，不能成为裁断前置条件。
func canonicalBaziEvidenceTopic(topic string) string {
	topic = strings.TrimSpace(topic)
	for _, alias := range []struct {
		keyword   string
		canonical string
	}{
		{keyword: "格局", canonical: "geju"},
		{keyword: "调候", canonical: "tiaohou"},
		{keyword: "扶抑", canonical: "fuyi"},
		{keyword: "大运", canonical: "dayun"},
		{keyword: "流年", canonical: "liunian"},
	} {
		if strings.Contains(topic, alias.keyword) {
			return alias.canonical
		}
	}
	return topic
}

// buildTiaohouEvidenceQuery derives a concrete 穷通 query from the calculated
// day master and month branch. This makes evidence coverage deterministic across
// runs instead of depending on the planner to remember the exact chart terms.
func buildTiaohouEvidenceQuery(input baziCharterInput) string {
	terms := []string{"穷通宝鉴"}
	if day := dayMasterForEvidenceQuery(input); day != "" {
		terms = append(terms, day)
	}
	if month := bazidomain.MonthBranchForEvidenceQuery(input); month != "" {
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
func MonthBranchForEvidenceQuery(input baziCharterInput) string {
	if pillar := bazidomain.ExtractMonthPillar(input.BaziResult["pillars"]); len(pillar) > 0 {
		if branch := stringValue(pillar["branch"]); branch != "" {
			return branch
		}
	}
	return firstNonEmptyTrim(stringValue(input.Yongshen["month_branch"]), stringValue(input.Yongshen["month_zhi"]))
}

// runControlledBaziRetrieval 至多执行 budget 条查询并限制累计注入材料。工具错误只记录，
// 不作为领域失败返回，因为最终模型可以没有古籍材料时继续回答。
func (e *Executor) runControlledBaziRetrieval(ctx context.Context, plan baziEvidencePlan, budget, citationBudget int) baziEvidenceBundle {
	bundle := baziEvidenceBundle{
		Stage:                plan.Stage,
		TopicBuckets:         map[string][]baziCitation{},
		CriticalTopicBuckets: map[string][]baziCitation{},
	}
	if !plan.NeedRetrieval || len(plan.QueryPackets) == 0 || budget <= 0 || citationBudget <= 0 {
		return bundle
	}
	if e == nil || e.builder == nil || e.reg == nil {
		tracing.SetTraceAttributes(ctx, map[string]any{"bazi.evidence.degrade_reason": "registry_unavailable"})
		return bundle
	}

	if e.toolRunner == nil {
		e.toolRunner = tools.NewToolRunner(e.reg)
	}
	for index, packet := range plan.QueryPackets {
		if index == budget {
			break
		}
		retrievalSpan := tracing.SpanFromContext(ctx, "knowledge_search", tracing.KindRetriever)
		retrievalSpan.SetAttribute("query", packet.Query)
		retrievalSpan.SetAttribute("topic", packet.Topic)
		run := e.toolRunner.Run(ctx, tools.ToolRunRequest{
			ToolName:       "knowledge_search",
			Params:         map[string]any{"query": packet.Query, "top_k": float64(2)},
			DecisionSource: "bazi_evidence",
		})
		if run.Status != tools.ToolRunStatusOK && run.Status != tools.ToolRunStatusFallback {
			if run.Error != nil {
				retrievalSpan.RecordError(run.Error)
			}
			retrievalSpan.SetAttribute("degrade_reason", "tool_error")
			retrievalSpan.End()
			bundle.DegradedTopics = mergeStrings(bundle.DegradedTopics, packet.Topic)
			continue
		}
		result := run.Data
		citations := citationsFromKnowledgeResult(result, packet)
		degradeReason := knowledgeResultDegradeReason(result)
		if len(citations) == 0 && degradeReason == "" {
			degradeReason = "no_results"
		}
		retrievalSpan.SetAttribute("hits", len(citations))
		retrievalSpan.SetAttribute("degraded", knowledgeResultDegraded(result))
		retrievalSpan.SetAttribute("degrade_reason", degradeReason)
		retrievalSpan.End()
		if len(citations) == 0 {
			if knowledgeResultDegraded(result) {
				bundle.DegradedTopics = mergeStrings(bundle.DegradedTopics, packet.Topic)
			}
			continue
		}
		remaining := citationBudget - len(bundle.Citations)
		if remaining <= 0 {
			continue
		}
		citations = limitCitationQuotes(citations, remaining)
		bundle.TopicBuckets[packet.Topic] = mergeCitations(bundle.TopicBuckets[packet.Topic], citations...)
		if strings.EqualFold(strings.TrimSpace(packet.SourceTier), "A") {
			bundle.CriticalTopicBuckets[packet.Topic] = mergeCitations(bundle.CriticalTopicBuckets[packet.Topic], citations...)
		}
		bundle.Citations = mergeCitations(bundle.Citations, citations...)
	}
	return bundle
}

// shouldReflectOnBaziEvidence 只在初检可用但没有原文或存在声明冲突时允许补充查询。
// 传输失败已有安全降级，不能消耗补充预算进行重试。
func shouldReflectOnBaziEvidence(plan baziEvidencePlan, bundle baziEvidenceBundle) bool {
	return plan.AllowReflection && len(bundle.DegradedTopics) == 0 && (len(bundle.Citations) == 0 || len(bundle.Conflicts) >= 2)
}

// buildEvidenceSupplementPlan 不重跑规划器而扩展一个已规划主题；固定的一条查询是反思阶段的硬边界。
func buildEvidenceSupplementPlan(plan baziEvidencePlan, bundle baziEvidenceBundle) baziEvidencePlan {
	if !plan.AllowReflection || len(plan.QueryPackets) == 0 || len(bundle.DegradedTopics) > 0 {
		return baziEvidencePlan{}
	}
	packet := plan.QueryPackets[0]
	query := strings.TrimSpace(strings.Join(append(append([]string{}, packet.PreferredSources...), packet.Topic, "原文"), " "))
	if query == "" {
		return baziEvidencePlan{}
	}
	packet.Query = query
	return baziEvidencePlan{NeedRetrieval: true, Stage: plan.Stage, QueryPackets: []baziQueryPacket{packet}}
}

// knowledgeResultDegraded 判断工具是否已走服务故障降级。
func knowledgeResultDegraded(result any) bool {
	rm, ok := result.(map[string]any)
	if !ok {
		return false
	}
	degraded, _ := rm["fallback"].(bool)
	return degraded
}

// knowledgeResultDegradeReason 提取工具返回的安全失败分类。
func knowledgeResultDegradeReason(result any) string {
	rm, ok := result.(map[string]any)
	if !ok {
		return ""
	}
	reason, _ := rm["degrade_reason"].(string)
	return strings.TrimSpace(reason)
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
			out = mergeCitations(out, citationFromPassage(passage.Source, passage.Content, passage.Quote))
		}
	case []any:
		for _, raw := range passages {
			pm, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			source, _ := pm["source"].(string)
			content, _ := pm["content"].(string)
			quote, _ := pm["quote"].(string)
			out = mergeCitations(out, citationFromPassage(source, content, quote))
		}
	}

	if preferred := filterPreferredCitations(out, packet.PreferredSources); len(preferred) > 0 {
		return limitCitationQuotes(preferred, 2)
	}
	return limitCitationQuotes(out, 2)
}

// limitCitationQuotes keeps optional evidence compact so retrieved material cannot
// crowd out deterministic chart facts in the synthesis prompt.
func limitCitationQuotes(items []baziCitation, limit int) []baziCitation {
	if limit <= 0 {
		return nil
	}
	var out []baziCitation
	for _, item := range items {
		if len(out) == limit {
			break
		}
		for _, quote := range item.Quotes {
			if len(out) == limit {
				break
			}
			out = append(out, baziCitation{Classic: item.Classic, Quotes: []string{quote}})
		}
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

// citationFromPassage 将单条检索片段映射为可合并的古籍引用，优先使用知识库提供的完整短引文。
func citationFromPassage(source, content, quote string) baziCitation {
	if !isSubstantiveBaziPassage(source, content) {
		return baziCitation{}
	}
	classic := extractAuthorityClassic(source)
	if classic == "" {
		classic = source
	}
	quote = firstNonEmptyTrim(quote, content)
	return baziCitation{
		Classic: classic,
		Quotes:  []string{quote},
	}
}

// isSubstantiveBaziPassage 拒绝书籍目录、索引和元数据摘要，即使旧知识库服务没有
// 做章节筛选也不会污染模型上下文或最终引用。
func isSubstantiveBaziPassage(source, content string) bool {
	content = strings.TrimSpace(content)
	return baziChapterSourcePattern.MatchString(source) && len([]rune(content)) >= 24 && !strings.Contains(content, "权威级别")
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
	static := bazidomain.StageAuthoritySources("static")
	dynamic := bazidomain.StageAuthoritySources("dynamic")

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

// mergeEvidenceBundles 合并受限初检与补充查询材料。
func mergeEvidenceBundles(base, add baziEvidenceBundle) baziEvidenceBundle {
	merged := baziEvidenceBundle{
		Stage:                firstNonEmptyTrim(base.Stage, add.Stage),
		TopicBuckets:         map[string][]baziCitation{},
		CriticalTopicBuckets: map[string][]baziCitation{},
		Citations:            mergeCitations(base.Citations, add.Citations...),
		Conflicts:            mergeStrings(base.Conflicts, add.Conflicts...),
		DegradedTopics:       mergeStrings(base.DegradedTopics, add.DegradedTopics...),
	}
	for _, bundle := range []baziEvidenceBundle{base, add} {
		for topic, citations := range bundle.TopicBuckets {
			merged.TopicBuckets[topic] = mergeCitations(merged.TopicBuckets[topic], citations...)
		}
		for topic, citations := range bundle.CriticalTopicBuckets {
			merged.CriticalTopicBuckets[topic] = mergeCitations(merged.CriticalTopicBuckets[topic], citations...)
		}
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
