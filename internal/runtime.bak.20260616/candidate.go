package runtime

import (
	"context"
	"regexp"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// executeRoute 直接从 ApprovedRoute 字段调度执行。
func (e *Executor) ExecuteRoute(ctx context.Context, sink EventSink, st *state.SessionState, route policy.ApprovedRoute, message string) (string, string, error) {
	profilePatch := route.Slots.Profile
	if profilePatch == nil {
		profilePatch = map[string]any{}
	}
	userQuestion := strings.TrimSpace(route.Slots.QuestionText)
	if userQuestion == "" {
		userQuestion = message
	}
	rawBazi := []string(nil)
	if route.TaskIntent == "direct_bazi" {
		rawBazi = extractBaziPillars(message)
	}
	// 奇门补充只看当前这轮批准路由，避免上一轮择时把后续普通追问也带成奇门。
	needsQimen := routeNeedsQimen(route)

	// 当会话已有资料时，重新分类误判的 collect_profile
	taskIntent := route.TaskIntent
	if taskIntent == "collect_profile" && (st.HasBaziResult() || len(st.Profile) > 0) && !containsBirthTime(userQuestion) {
		taskIntent = "amend_profile"
	}

	if route.NeedsClarification {
		return e.executeClarificationRoute(ctx, sink, st, profilePatch, userQuestion, route.ClarificationQuestion)
	}

	// 奇门主领域路径：当 supervisor + 策略门批准奇门作为
	// 主分析链时，将 qimen_dunjia 作为主线步骤执行，
	// 不再强耦合到 timing_followup 这一个 task_intent。
	if routeWantsQimenPrimary(route) {
		return e.executeQimenPrimaryRoute(ctx, sink, st, profilePatch, userQuestion)
	}

	// 紫微主领域路径：当 supervisor + 策略门批准紫微作为
	// 主领域时，将 ziwei_calc 作为主线步骤执行。
	if route.PrimaryDomain == "ziwei" {
		return e.executeZiweiPrimaryRoute(ctx, sink, st, profilePatch, userQuestion)
	}

	switch taskIntent {
	case "direct_bazi":
		return e.executeDirectBaziRoute(ctx, sink, st, profilePatch, rawBazi)
	case "collect_profile":
		return e.executeCollectProfileRoute(ctx, sink, st, profilePatch, userQuestion)
	case "amend_profile":
		return e.executeAmendProfileRoute(ctx, sink, st, profilePatch, userQuestion)
	case "timing_followup", "cross_domain_consult":
		return e.executeFollowupRoute(ctx, sink, st, userQuestion, true)
	default:
		return e.executeFollowupRoute(ctx, sink, st, userQuestion, needsQimen)
	}
}

// executeClarificationRoute 在路由需要时询问澄清问题，否则回退到询问缺失的资料字段。
func (e *Executor) executeClarificationRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string, clarificationQuestion string) (string, string, error) {
	candidate := st.Clone()
	candidate.MergeProfile(profilePatch)
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}

	// 当路由明确要求澄清时，不直接从现有命盘回答
	// 而是询问已批准的澄清问题。
	if candidate.HasBaziResult() || candidate.IsProfileComplete() {
		if strings.TrimSpace(clarificationQuestion) == "" {
			clarificationQuestion = "请确认一下您的需求，我再为您详细分析。"
		}
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": clarificationQuestion}})
		*st = *candidate
		return "clarification", clarificationQuestion, nil
	}

	if strings.TrimSpace(clarificationQuestion) != "" {
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": clarificationQuestion}})
		*st = *candidate
		return "ask_missing_profile", clarificationQuestion, nil
	}

	text, err := e.handleAsk(ctx, sink, candidate)
	if err == nil {
		*st = *candidate
	}
	return "ask_missing_profile", text, err
}

// executeCollectProfileRoute 处理 collect_profile：重置候选会话的资料/命盘，询问缺失字段，或在资料完整时进行完整解读。
func (e *Executor) executeCollectProfileRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string) (string, string, error) {
	candidate := st.Clone()
	candidate.Profile = make(map[string]any)
	candidate.BaziResult = nil
	for k, v := range profilePatch {
		candidate.Profile[k] = v
	}
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}
	if !candidate.IsProfileComplete() {
		text, err := e.handleAsk(ctx, sink, candidate)
		if err == nil && !st.HasBaziResult() && len(st.Profile) == 0 {
			*st = *candidate
		}
		return "ask_missing_profile", text, err
	}
	text, err := e.handleFullReading(ctx, sink, candidate)
	if candidate.HasBaziResult() {
		*st = *candidate
	}
	return "full_reading", text, err
}

// executeAmendProfileRoute 处理 amend_profile：合并资料补丁而不清空命盘，除非影响命盘的字段发生了变化。
func (e *Executor) executeAmendProfileRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, userQuestion string) (string, string, error) {
	candidate := st.Clone()
	changed := candidate.MergeProfile(profilePatch)
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}
	if changed && profileChangesAffectChart(profilePatch) {
		candidate.BaziResult = nil
	}
	if !candidate.IsProfileComplete() && !candidate.HasBaziResult() {
		text, err := e.handleAsk(ctx, sink, candidate)
		if err == nil {
			*st = *candidate
		}
		return "ask_missing_profile", text, err
	}
	if candidate.BaziResult == nil {
		text, err := e.handleFullReading(ctx, sink, candidate)
		if candidate.HasBaziResult() {
			*st = *candidate
		}
		return "full_reading", text, err
	}
	text, err := e.handleFollowupReading(ctx, sink, candidate)
	if err == nil {
		*st = *candidate
	}
	return "followup_reading", text, err
}

// executeDirectBaziRoute 处理 direct_bazi：将性别合并到候选会话中，并运行直接八字输入分析。
func (e *Executor) executeDirectBaziRoute(ctx context.Context, sink EventSink, st *state.SessionState, profilePatch map[string]any, rawBazi []string) (string, string, error) {
	candidate := st.Clone()
	if g, ok := profilePatch["gender"]; ok {
		candidate.Profile["gender"] = g
	}
	text, err := e.handleBaziInput(ctx, sink, candidate, rawBazi)
	if candidate.HasBaziResult() {
		*st = *candidate
	}
	return "direct_bazi", text, err
}

// executeFollowupRoute 处理所有跟进任务意图：有命盘时复用，否则回退到资料收集/完整解读。
func (e *Executor) executeFollowupRoute(ctx context.Context, sink EventSink, st *state.SessionState, userQuestion string, needsQimen bool) (string, string, error) {
	candidate := st.Clone()
	if userQuestion != "" {
		candidate.LastUserQuestion = userQuestion
	}
	candidate.NeedsQimen = needsQimen

	if needsQimen {
		return e.executeParallelFortuneRoute(ctx, sink, st, candidate, userQuestion)
	}

	if candidate.HasBaziResult() {
		text, err := e.handleFollowupReading(ctx, sink, candidate)
		if err == nil {
			*st = *candidate
		}
		return "followup_reading", text, err
	}
	if !candidate.IsProfileComplete() {
		text, err := e.handleAsk(ctx, sink, candidate)
		if err == nil {
			*st = *candidate
		}
		return "ask_missing_profile", text, err
	}
	text, err := e.handleFullReading(ctx, sink, candidate)
	if candidate.HasBaziResult() {
		*st = *candidate
	}
	return "full_reading", text, err
}

var birthTimeSignalRe = regexp.MustCompile(`\d{4}\s*年.*\d{1,2}\s*月|\d{4}[-/]\d{1,2}|农历|阴历|正月|腊月`)

var baziPairRe = regexp.MustCompile(`([甲乙丙丁戊己庚辛壬癸][子丑寅卯辰巳午未申酉戌亥])`)

func containsBirthTime(msg string) bool {
	return birthTimeSignalRe.MatchString(msg)
}

func extractBaziPillars(msg string) []string {
	matches := baziPairRe.FindAllString(msg, -1)
	if len(matches) < 4 {
		return nil
	}
	return matches[:4]
}

func routeNeedsQimen(route policy.ApprovedRoute) bool {
	switch route.PolicyHints.QimenMode {
	case "primary", "supplement":
		return true
	case "none":
		return false
	}
	if route.PolicyHints.NeedsQimen {
		return true
	}
	switch route.TaskIntent {
	case "timing_followup", "cross_domain_consult":
		return true
	}
	return false
}

func routeWantsQimenPrimary(route policy.ApprovedRoute) bool {
	if route.PrimaryDomain != "qimen" {
		return false
	}
	if route.PolicyHints.QimenMode == "primary" {
		return true
	}
	switch route.TaskIntent {
	case "timing_followup", "cross_domain_consult":
		return true
	}
	return false
}

var chartFields = map[string]bool{"year": true, "month": true, "day": true, "hour": true}

func profileChangesAffectChart(patch map[string]any) bool {
	for k, v := range patch {
		if !chartFields[k] {
			continue
		}
		switch val := v.(type) {
		case float64:
			if val > 0 {
				return true
			}
		case int:
			if val > 0 {
				return true
			}
		case string:
			if val != "" {
				return true
			}
		default:
			if v != nil {
				return true
			}
		}
	}
	return false
}
