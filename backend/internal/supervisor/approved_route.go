package supervisor

import (
	"context"
	"log"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/contracts"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
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
//   - subject 切换时清空旧盘，强制重新采集
//   - 消息包含出生信息但模型漏提取时，回填 profile 并强制 collect_profile
//
// 依赖完整会话连续性的 task reinterpretation（如 collect_profile → amend_profile /
// fortune_followup）已经下沉到 manager 侧，由 runtime conversation owner 统一处理。
func (c *Client) normalizeApprovedRoute(ctx context.Context, msg string, st *state.SessionState, route policy.ApprovedRoute) policy.ApprovedRoute {
	c.applyExplicitMethodPreference(ctx, msg, &route)

	// subject 切换检测：TargetSubject 非空且与当前盘归属不同 → 清旧盘，强制重新采集
	if newSubject := route.Slots.TargetSubject; newSubject != "" && st.Subject != "" && newSubject != st.Subject {
		log.Printf("[supervisor] subject change: %q → %q, clearing old chart data", st.Subject, newSubject)
		st.BaziResult = nil
		st.ZiWeiResult = nil
		st.QimenResult = nil
		st.Profile = make(map[string]any)
		st.Subject = newSubject
		route.TaskIntent = "collect_profile"
		route.NeedsClarification = false
		route.ClarificationQuestion = ""
		route.PolicyHints.CanReuseCachedResult = false
		route.PolicyHints.CanReuseSessionProfile = false
	}

	profileReady := st.IsProfileComplete() || st.HasBaziResult()
	if !profileReady && intent.ContainsBirthInfo(msg) &&
		route.TaskIntent != "collect_profile" &&
		route.TaskIntent != "amend_profile" &&
		route.TaskIntent != "direct_bazi" {
		c.backfillRouteProfile(ctx, msg, st, &route)
		route.TaskIntent = "collect_profile"
		route.NeedsClarification = false
		route.ClarificationQuestion = ""
	}

	return route
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
	case intent.ContainsTimingKeyword(trimmed) && route.PolicyHints.QimenMode == "none":
		route.PolicyHints.QimenMode = "supplement"
		route.PolicyHints.NeedsQimen = true
	}
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

// backfillRouteProfile 当 LLM 漏提取出生信息但消息中明显包含时，用简化提取链补齐。
//
// 触发条件：normalizeApprovedRoute 检测到消息包含出生时间但模型返回的 route 中没有 profile 数据。
// 使用 fallbackExtract 的简化提示词做一次轻量提取，仅回填缺失字段，不覆盖已有值。
// 这是一个"补丁"操作——正常流程中 LLM 应在首次决策时完成提取。
func (c *Client) backfillRouteProfile(ctx context.Context, msg string, st *state.SessionState, route *policy.ApprovedRoute) {
	if route.Slots.Profile == nil {
		route.Slots.Profile = make(map[string]any)
	}
	if len(route.Slots.Profile) > 0 {
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
