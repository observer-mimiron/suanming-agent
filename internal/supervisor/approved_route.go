package supervisor

import (
	"context"
	"log"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/intent"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// Approve 是 supervisor 的外部入口：决策 → 策略应用 → 规范化，返回可直接执行的路由。
//
// 这是 orchestrator 调用的主方法。内部依次执行：
//  1. Decide：三层防御的 LLM 决策（约束解码 → 文本生成 → 安全回退）
//  2. policy.Apply：将 SupervisorDecision 转换为 ApprovedRoute，注入策略默认值
//  3. normalizeApprovedRoute：基于当前会话状态做确定性修正（如已有资料时 collect_profile → amend_profile）
//
// 即使 Decide 返回 error（降级回退），也会返回一个保守的 ApprovedRoute 而非 nil，
// 确保 orchestrator 始终有可执行的路由。
func (c *Client) Approve(ctx context.Context, msg string, st *state.SessionState) (policy.ApprovedRoute, error) {
	decision, err := c.Decide(ctx, msg, st)
	route := policy.Apply(decision, st)
	route = c.normalizeApprovedRoute(ctx, msg, st, route)
	return route, err
}

// normalizeApprovedRoute 根据当前会话状态对 LLM 产出的路由做确定性修正。
//
// LLM 决策可能忽略已有的会话上下文（如已知用户出生信息却仍然判定 collect_profile），
// 本函数用硬规则修正这些情况，不依赖模型判断：
//
//   - 已有资料时 collect_profile → amend_profile（补充而非重新采集）
//   - 已有命盘时 collect_profile → fortune_followup（除非消息确实包含新出生时间）
//   - 消息包含出生时间但模型未识别 → 回填 profile 并强制 collect_profile
//
// 这些修正是纯确定性的，不涉及 LLM 调用，保证关键路由决策的稳定性。
func (c *Client) normalizeApprovedRoute(ctx context.Context, msg string, st *state.SessionState, route policy.ApprovedRoute) policy.ApprovedRoute {
	applyExplicitMethodPreference(msg, &route)

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

	if route.TaskIntent == "collect_profile" && len(st.Profile) > 0 {
		route.TaskIntent = "amend_profile"
		route.PolicyHints.CanReuseSessionProfile = true
		if st.HasBaziResult() {
			route.PolicyHints.CanReuseCachedResult = true
		}
	}

	if route.TaskIntent == "collect_profile" && st.HasBaziResult() && intent.ContainsBirthInfo(msg) {
		// keep collect_profile if user really gave new birth-time info
	} else if route.TaskIntent == "collect_profile" && st.HasBaziResult() {
		route.TaskIntent = "fortune_followup"
		route.PolicyHints.CanReuseCachedResult = true
		route.PolicyHints.CanReuseSessionProfile = true
	}

	// 已有命盘且用户未提供新出生信息时，interpret_chart → fortune_followup
	// 避免已有命盘时 specialist 重新跑完整首次解读
	if route.TaskIntent == "interpret_chart" && st.HasBaziResult() && !intent.ContainsBirthInfo(msg) {
		route.TaskIntent = "fortune_followup"
		route.PolicyHints.CanReuseCachedResult = true
		route.PolicyHints.CanReuseSessionProfile = true
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
func applyExplicitMethodPreference(msg string, route *policy.ApprovedRoute) {
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
