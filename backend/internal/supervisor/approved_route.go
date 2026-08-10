// Package supervisor 的本文件属于路由准入层。
//
// 本文件负责把路由模型结果收敛为可执行的 ApprovedRoute，并补全用户明确写出的出生资料；
// 不负责 Manager 的会话语义重解释，也不负责命盘计算或渲染。
package supervisor

import (
	"context"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

var (
	explicitBirthClockPattern = regexp.MustCompile(`(?:^|[^0-9])([01]?\d|2[0-3])\s*(?:点|时|:|：)\s*([0-5]?\d)\s*分?`)
	explicitBirthDatePattern  = regexp.MustCompile(`(\d{4})\s*年\s*(\d{1,2})\s*月\s*(\d{1,2})\s*[日号]?`)
	explicitBirthplacePattern = regexp.MustCompile(`[男女]\s*([一-龥]{2,8})\s*$`)
)

// Approve 是 supervisor 的外部入口：决策 → 策略应用 → 规范化，返回可直接执行的路由。
//
// 这是 orchestrator 调用的主方法。内部依次执行：
//  1. Decide：三层防御的 LLM 决策（约束解码 → 文本生成 → 安全回退）
//  2. policy.Apply：将 SupervisorDecision 转换为 ApprovedRoute，注入策略默认值
//  3. normalizeApprovedRoute：基于当前会话状态做 supervisor-owned 的确定性修正
//
// 即使 Decide 返回 error（降级回退），也会返回一个保守的 ApprovedRoute 而非 nil，
// 确保 orchestrator 始终有可执行的路由。
func (c *Client) Approve(ctx context.Context, msg string, st *state.SessionState) (policy.ApprovedRoute, error) {
	if route, ok := c.tryCheapFollowupRoute(msg, st); ok {
		sp := tracing.SpanFromContext(ctx, "supervisor_decision", tracing.KindChain)
		sp.SetAttribute("decision_source", "cheap_followup_reuse")
		sp.SetAttribute("reuse_reason", route.Gate.Reason)
		sp.SetAttribute("reuse_cached_result", route.Gate.ReuseCachedResult)
		sp.SetAttribute("reuse_session_profile", route.Gate.ReuseSessionProfile)
		sp.SetAttribute("primary_domain", route.PrimaryDomain)
		sp.SetAttribute("task_intent", route.TaskIntent)
		sp.SetStatus("ok")
		sp.End()
		if c.reporter != nil {
			_ = c.reporter.Record(ctx, msg, contracts.ExecutionSnapshot{
				ConsultationKind:   route.ConsultationKind,
				PrimaryDomain:      route.PrimaryDomain,
				SecondaryDomains:   append([]string(nil), route.SecondaryDomains...),
				TaskIntent:         route.TaskIntent,
				ConversationIntent: route.ConversationIntent,
				QimenMode:          route.PolicyHints.QimenMode,
				TargetSubject:      route.Slots.TargetSubject,
				TimeScope:          route.Slots.TimeScope,
				Gate:               route.Gate,
			})
		}
		return route, nil
	}
	decision, err := c.Decide(ctx, msg, st)
	route := policy.Apply(decision, st)
	route = c.normalizeApprovedRoute(ctx, msg, st, route)
	return route, err
}

// normalizeApprovedRoute 根据当前会话状态对 LLM 产出的路由做确定性修正。
//
// supervisor 只保留前置准入层应该拥有的硬规则：
//   - 显式术数方法偏好纠偏
//   - 不在审批层处理对象切换；对象、资料版本和资产由 runtime resolver 绑定
//   - 消息包含出生信息但模型漏提取时，回填 profile 并强制 collect_profile
//
// 依赖完整会话连续性的 task reinterpretation（如 collect_profile → amend_profile /
// fortune_followup）已经下沉到 manager 侧，由 runtime conversation owner 统一处理。
func (c *Client) normalizeApprovedRoute(ctx context.Context, msg string, st *state.SessionState, route policy.ApprovedRoute) policy.ApprovedRoute {
	c.applyExplicitMethodPreference(ctx, msg, &route)
	applyConsultationContract(msg, st, &route)
	if intent.ContainsBirthInfo(msg) {
		normalizeExplicitBirthClock(msg, &route)
	}

	profileReady := st.IsProfileComplete() || st.HasBaziResult()
	if !profileReady && intent.ContainsBirthInfo(msg) {
		c.backfillRouteProfile(ctx, msg, st, &route)
		if routeProfileComplete(st, route.Slots.Profile) {
			route.TaskIntent = "collect_profile"
			route.NeedsClarification = false
			route.ClarificationQuestion = ""
		}
	}

	return route
}

// normalizeExplicitBirthClock 用用户原文补齐模型遗漏的出生时分。
// 分钟会影响接近换日边界的命盘，不能因模型漏填而丢失。
func normalizeExplicitBirthClock(msg string, route *policy.ApprovedRoute) {
	if route == nil {
		return
	}
	matches := explicitBirthClockPattern.FindStringSubmatch(msg)
	if len(matches) != 3 {
		return
	}
	hour, hourErr := strconv.Atoi(matches[1])
	minute, minuteErr := strconv.Atoi(matches[2])
	if hourErr != nil || minuteErr != nil {
		return
	}
	if route.Slots.Profile == nil {
		route.Slots.Profile = map[string]any{}
	}
	// 用户原文是出生事实，优先于只保留小时的路由模型输出。
	route.Slots.Profile["hour"] = float64(hour)
	route.Slots.Profile["minute"] = float64(minute)
}

// applyExplicitMethodPreference 在用户明确指定术数方法时做硬性纠偏。
// 这里只 obey 显式方法意图，不把一般语义问题扩展成 case 规则库。
func applyRegexMethodPreference(msg string, route *policy.ApprovedRoute) {
	trimmed := strings.TrimSpace(msg)
	if trimmed == "" || route == nil {
		return
	}
	switch {
	case intent.MentionsZiweiMethod(trimmed):
		route.PrimaryDomain = "ziwei"
		route.SecondaryDomains = removeDomain(route.SecondaryDomains, "ziwei")
		// 铁律：紫微必须结合八字
		if !hasDomain(route.SecondaryDomains, "bazi") {
			route.SecondaryDomains = append(route.SecondaryDomains, "bazi")
		}
		route.PolicyHints.QimenMode = "none"
		route.PolicyHints.NeedsQimen = false
		if route.PolicyHints.ProfileRequirement == "" {
			route.PolicyHints.ProfileRequirement = "full"
		}
	case intent.MentionsQimenMethod(trimmed):
		route.PrimaryDomain = "qimen"
		route.SecondaryDomains = removeDomain(route.SecondaryDomains, "qimen")
		route.PolicyHints.QimenMode = "primary"
		route.PolicyHints.NeedsQimen = true
		if route.PolicyHints.ProfileRequirement == "" {
			route.PolicyHints.ProfileRequirement = "none"
		}
	case intent.MentionsBaziMethod(trimmed):
		route.PrimaryDomain = "bazi"
		route.SecondaryDomains = removeDomain(route.SecondaryDomains, "bazi")
		if route.PolicyHints.QimenMode == "" {
			route.PolicyHints.QimenMode = "none"
		}
	}
}

// applyConsultationContract maps user-visible semantics to the frozen route contract.
// Timing words alone stay on the period-fortune path and never add qimen.
func applyConsultationContract(msg string, st *state.SessionState, route *policy.ApprovedRoute) {
	if route == nil {
		return
	}
	// 四值分类只属于已准入的咨询执行轮次；资料收集、修订、直答和澄清
	// 都会在 preflight 短路，保留分类会让 snapshot 虚报 specialist 已执行。
	if route.NeedsClarification || route.TaskIntent == "collect_profile" ||
		route.TaskIntent == "amend_profile" || route.TaskIntent == "direct_bazi" {
		route.ConsultationKind = ""
		return
	}
	if isUnsupportedConsultationCombination(msg) {
		markConsultationClarification(route)
		return
	}

	kind := consultationKindForMessage(msg, *route)
	route.ConsultationKind = kind
	switch kind {
	case contracts.ConsultationKindEventQuestion:
		route.PrimaryDomain = "qimen"
		route.SecondaryDomains = nil
		route.PolicyHints.QimenMode = "primary"
		route.PolicyHints.NeedsQimen = true
		route.PolicyHints.ProfileRequirement = "none"
		if route.Gate.Reason == "profile_incomplete" {
			route.NeedsClarification = false
			route.ClarificationQuestion = ""
			route.Gate.Admitted = true
			route.Gate.Reason = ""
			route.Gate.ExecutionMode = "execute"
		}
	case contracts.ConsultationKindHealthRisk:
		route.PrimaryDomain = "bazi"
		route.SecondaryDomains = []string{"ziwei"}
		route.PolicyHints.QimenMode = "none"
		route.PolicyHints.NeedsQimen = false
		route.PolicyHints.ProfileRequirement = "full"
	case contracts.ConsultationKindPeriodFortune:
		route.PrimaryDomain = "bazi"
		route.SecondaryDomains = []string{"ziwei"}
		route.PolicyHints.QimenMode = "none"
		route.PolicyHints.NeedsQimen = false
		route.PolicyHints.ProfileRequirement = "full"
	case contracts.ConsultationKindNatalChart:
		if intent.MentionsZiweiMethod(msg) {
			route.PrimaryDomain = "ziwei"
		} else if intent.MentionsBaziMethod(msg) {
			route.PrimaryDomain = "bazi"
		}
		route.SecondaryDomains = nil
		route.PolicyHints.QimenMode = "none"
		route.PolicyHints.NeedsQimen = false
		route.PolicyHints.ProfileRequirement = "full"
	}

	if kind != contracts.ConsultationKindEventQuestion && st != nil &&
		!st.IsProfileComplete() && !st.HasBaziResult() &&
		route.TaskIntent != "collect_profile" && route.TaskIntent != "amend_profile" &&
		route.TaskIntent != "direct_bazi" {
		// 分类合同只描述已准入执行轮次；进入资料澄清后必须清空，避免
		// trace 和后续重建把等待状态误认为已经执行 specialist。
		route.ConsultationKind = ""
		route.NeedsClarification = true
		route.Gate.Admitted = false
		route.Gate.Reason = "profile_incomplete"
		route.Gate.ExecutionMode = "clarify"
	}
}

// consultationKindForMessage applies the frozen classification priority to one message.
func consultationKindForMessage(msg string, route policy.ApprovedRoute) contracts.ConsultationKind {
	trimmed := strings.TrimSpace(msg)
	if isUnsupportedConsultationCombination(trimmed) {
		return ""
	}
	switch {
	case isEventQuestion(trimmed):
		return contracts.ConsultationKindEventQuestion
	case mentionsHealthRisk(trimmed):
		return contracts.ConsultationKindHealthRisk
	case isNatalChartRequest(trimmed):
		return contracts.ConsultationKindNatalChart
	case intent.ContainsTimingKeyword(trimmed):
		return contracts.ConsultationKindPeriodFortune
	case policy.ValidConsultationKind(route.ConsultationKind):
		return route.ConsultationKind
	case route.PrimaryDomain == "qimen":
		return contracts.ConsultationKindEventQuestion
	case route.PrimaryDomain == "ziwei" || route.TaskIntent == "interpret_chart":
		return contracts.ConsultationKindNatalChart
	case route.TaskIntent == "fortune_followup":
		return contracts.ConsultationKindPeriodFortune
	default:
		return ""
	}
}

// isUnsupportedConsultationCombination rejects health actions and explicit natal
// methods attached to concrete events before any route method preference applies.
func isUnsupportedConsultationCombination(msg string) bool {
	trimmed := strings.TrimSpace(msg)
	if isEventQuestion(trimmed) && mentionsHealthRisk(trimmed) {
		return true
	}
	return isEventQuestion(trimmed) &&
		(intent.MentionsBaziMethod(trimmed) || intent.MentionsZiweiMethod(trimmed)) &&
		!intent.MentionsQimenMethod(trimmed)
}

// markConsultationClarification routes an unsupported combination to the existing
// clarification short circuit without inventing another consultation kind.
func markConsultationClarification(route *policy.ApprovedRoute) {
	if route == nil {
		return
	}
	route.ConsultationKind = ""
	route.NeedsClarification = true
	route.ClarificationQuestion = "这个问题同时包含具体事件和方法选择，请确认是按奇门问事，还是按出生盘方法分析。"
	route.Gate.Admitted = false
	route.Gate.Reason = "consultation_method_conflict"
	route.Gate.ExecutionMode = "clarify"
}

// isNatalChartRequest recognizes an explicit birth-chart method request.
func isNatalChartRequest(msg string) bool {
	return (intent.ContainsExplicitDivinationAction(msg) || strings.Contains(msg, "命盘") || strings.Contains(msg, "排盘")) &&
		(intent.MentionsBaziMethod(msg) || intent.MentionsZiweiMethod(msg))
}

// isEventQuestion recognizes a concrete event or qimen question.
func isEventQuestion(msg string) bool {
	for _, marker := range []string{"面试", "签约", "谈合作", "出行", "择时", "择日", "起局", "问事", "这件事", "能否成", "要不要"} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	if strings.Contains(msg, "合作") {
		for _, marker := range []string{"这次", "此次", "项目", "谈", "能否", "能不能", "是否", "成不成"} {
			if strings.Contains(msg, marker) {
				return true
			}
		}
	}
	if intent.MentionsQimenMethod(msg) && !intent.ContainsTimingKeyword(msg) {
		return true
	}
	// 没有出生资料时，“今天/此刻运气如何”是本时刻问事，不应先走
	// 需要完整命盘的阶段运势路径；本月/今年/最近仍由 period_fortune 处理。
	if !intent.HasTimingFocus(msg) {
		return false
	}
	for _, marker := range []string{"今天", "今日", "此刻", "现在", "当前"} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

// mentionsHealthRisk recognizes the broad health-risk category for the safety profile.
func mentionsHealthRisk(msg string) bool {
	for _, marker := range []string{"健康", "身体", "生病", "疾病", "就医", "检查"} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

// applyExplicitMethodPreference 在用户显式指定术数方法时做主领域纠偏。
// 路由模式由 c.routerMode 控制：
//   - off: 不调 router，走旧 regex MentionsXxxMethod（受 Confidence 守卫）
//   - shadow: 调 router 只 log，决策仍走 regex（受 Confidence 守卫）
//   - enforce: router positive 命中即覆盖 LLM（不看 Confidence）；
//     router err 才退回 regex（受 Confidence 守卫）；negative/none 不覆盖不退回
func (c *Client) applyExplicitMethodPreference(ctx context.Context, msg string, route *policy.ApprovedRoute) {
	trimmed := strings.TrimSpace(msg)
	if trimmed == "" || route == nil {
		return
	}

	// router 路径（enforce/shadow 模式且 router 已注入）
	if c.router != nil && (c.routerMode == "enforce" || c.routerMode == "shadow") {
		result, err := c.router.Match(ctx, trimmed)

		if c.routerMode == "shadow" {
			// 旁路：只 log，决策走 regex（落入下面的 regex 分支）
			log.Printf("[router.shadow] msg=%q result=%+v err=%v", trimmed, result, err)
		} else if err == nil {
			// enforce 模式且 Match 成功
			switch result.Decision {
			case intent.DecisionPositive:
				// router 可信，positive 命中即覆盖，不看 Confidence
				route.PrimaryDomain = result.Method
				route.SecondaryDomains = removeDomain(route.SecondaryDomains, result.Method)
				applyMethodPolicyHints(result.Method, route)
				return
			case intent.DecisionNegative, intent.DecisionNone:
				// 不覆盖，**不退回 regex**——避免 negative 被 regex 击穿
				return
			}
		}
		// err != nil 落到下面的 regex 兜底分支
	}

	// regex 兜底分支（off 模式 / shadow 模式 / enforce+err）
	// Confidence 守卫：高置信时禁用 regex，信任 LLM
	if route.Confidence >= 0.7 {
		return
	}
	applyRegexMethodPreference(trimmed, route)
}

// applyMethodPolicyHints 在 router positive 命中时设置对应方法的策略提示。
// 逻辑与 applyRegexMethodPreference 一致，只是数据源从 regex 变成 router。
func applyMethodPolicyHints(method string, route *policy.ApprovedRoute) {
	switch method {
	case "ziwei":
		if !hasDomain(route.SecondaryDomains, "bazi") {
			route.SecondaryDomains = append(route.SecondaryDomains, "bazi")
		}
		route.PolicyHints.QimenMode = "none"
		route.PolicyHints.NeedsQimen = false
		if route.PolicyHints.ProfileRequirement == "" {
			route.PolicyHints.ProfileRequirement = "full"
		}
	case "qimen":
		route.PolicyHints.QimenMode = "primary"
		route.PolicyHints.NeedsQimen = true
		if route.PolicyHints.ProfileRequirement == "" {
			route.PolicyHints.ProfileRequirement = "none"
		}
	case "bazi":
		if route.PolicyHints.QimenMode == "" {
			route.PolicyHints.QimenMode = "none"
		}
	}
}

// backfillRouteProfile 当 LLM 漏提取出生信息但消息中明显包含时，用原文和简化提取链补齐。
//
// 原文中格式明确的日期、时分、性别和紧邻末尾地点先确定性提取，避免降级模型再次遗漏；
// 仍不完整时才调用 fallbackExtract，仅回填缺失字段，不覆盖已有值。
func (c *Client) backfillRouteProfile(ctx context.Context, msg string, st *state.SessionState, route *policy.ApprovedRoute) {
	if route.Slots.Profile == nil {
		route.Slots.Profile = make(map[string]any)
	}
	mergeExplicitBirthProfile(msg, route.Slots.Profile)
	if routeProfileComplete(st, route.Slots.Profile) {
		return
	}

	patch, question, err := c.ExtractProfile(ctx, msg, st)
	if err != nil {
		log.Printf("[supervisor] profile backfill failed: %v", err)
		return
	}
	for k, v := range patch {
		if _, exists := route.Slots.Profile[k]; !exists {
			route.Slots.Profile[k] = v
		}
	}
	if route.Slots.QuestionText == "" || route.Slots.QuestionText == msg {
		route.Slots.QuestionText = question
	}
}

// mergeExplicitBirthProfile 只从固定格式的用户原文提取出生字段，不猜测缺失资料。
func mergeExplicitBirthProfile(msg string, profile map[string]any) {
	if profile == nil {
		return
	}
	if matches := explicitBirthDatePattern.FindStringSubmatch(msg); len(matches) == 4 {
		for index, field := range []string{"year", "month", "day"} {
			if _, exists := profile[field]; exists {
				continue
			}
			if value, err := strconv.Atoi(matches[index+1]); err == nil {
				profile[field] = float64(value)
			}
		}
	}
	for _, gender := range []string{"男", "女"} {
		if _, exists := profile["gender"]; !exists && strings.Contains(msg, gender) {
			profile["gender"] = gender
		}
	}
	if _, exists := profile["birthplace"]; !exists {
		if matches := explicitBirthplacePattern.FindStringSubmatch(strings.TrimSpace(msg)); len(matches) == 2 {
			profile["birthplace"] = matches[1]
		}
	}
}

// routeProfileComplete 按 SessionState 的同一资料完整度合同检查路由 slots。
func routeProfileComplete(st *state.SessionState, profile map[string]any) bool {
	if st == nil {
		st = state.NewSession("")
	} else {
		st = st.Clone()
	}
	st.MergeProfile(profile)
	return st.IsProfileComplete()
}

func hasDomain(domains []string, target string) bool {
	for _, d := range domains {
		if d == target {
			return true
		}
	}
	return false
}

func removeDomain(domains []string, target string) []string {
	if len(domains) == 0 {
		return domains
	}
	filtered := domains[:0]
	for _, d := range domains {
		if d != target {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
