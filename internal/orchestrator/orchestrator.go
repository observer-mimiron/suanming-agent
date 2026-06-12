package orchestrator

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/mcp"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tools"
	qimenTools "github.com/wikiglobal/suanming-agent/internal/tools/qimen"
	"github.com/wikiglobal/suanming-agent/internal/tracing"
)

// Supervisor is the interface for LLM-driven routing decisions.
// The sole implementation is supervisor.Client; tests use a mock.
type Supervisor interface {
	Decide(ctx context.Context, msg string, st *state.SessionState) (schemas.SupervisorDecision, error)
}

// specialistEventSink adapts the orchestrator EventSink interface to the
// specialists.EventSink function type used by DomainHandler.
func specialistEventSink(sink EventSink) specialists.EventSink {
	return func(ctx context.Context, evt specialists.Event) error {
		return sink.Emit(ctx, Event{Type: evt.Type, Data: evt.Data})
	}
}

type Orchestrator struct {
	tools      *tools.Registry
	store      state.Store
	locker     state.Locker
	llm        llm.Chat
	flash      llm.Chat
	tracer     tracing.Tracer
	promptMode string
	llmModel   string
	supervisor Supervisor
	baziSp     specialists.DomainHandler
	qimenSp    specialists.DomainHandler
	ziweiSp    specialists.DomainHandler
}

func New(reg *tools.Registry, llmClient llm.Chat, flashClient llm.Chat, store state.Store, locker state.Locker, tracer tracing.Tracer, promptMode string) *Orchestrator {
	return &Orchestrator{tools: reg, llm: llmClient, flash: flashClient, store: store, locker: locker, tracer: tracer, promptMode: promptMode}
}

// SetLLMModel sets the model name used in LLM span metadata.
func (o *Orchestrator) SetLLMModel(model string) { o.llmModel = model }

// SetSupervisor injects the supervisor client for phase-1 routing.
// When nil, the orchestrator falls back to the legacy classifyAndExtract path.
func (o *Orchestrator) SetSupervisor(sv Supervisor) { o.supervisor = sv }

// SetSpecialists injects the bazi, qimen, and ziwei domain specialists.
func (o *Orchestrator) SetSpecialists(baziSp, qimenSp, ziweiSp specialists.DomainHandler) {
	o.baziSp = baziSp
	o.qimenSp = qimenSp
	o.ziweiSp = ziweiSp
}

func (o *Orchestrator) Run(ctx context.Context, sink EventSink, sessionID, message string) error {
	unlock := o.locker.Lock(sessionID)
	defer unlock()

	ctx, trace := o.tracer.StartTrace(ctx, "chat.turn")
	defer trace.End()

	st := o.store.LoadOrCreate(sessionID)
	defer o.store.Save(st)

	// Annotate root trace with metadata
	if t := tracing.TraceFromContext(ctx); t != nil {
		t.SessionID = sessionID
		t.UserMessage = message
	}

	var action string
	var profilePatch map[string]any
	var userQuestion string
	var needsQimen bool
	var rawBazi []string
	var approvedRoute policy.ApprovedRoute

	// Phase 1: use supervisor when available, fall back to legacy classify.
	if o.supervisor != nil {
		supSpan := tracing.SpanFromContext(ctx, "supervisor_decision", tracing.KindChain)
		decision, err := o.supervisor.Decide(ctx, message, st)
		if err != nil {
			supSpan.SetAttribute("error", err.Error())
		}
		supSpan.SetAttribute("primary_domain", decision.PrimaryDomain)
		supSpan.SetAttribute("task_intent", decision.TaskIntent)
		supSpan.SetAttribute("confidence", decision.Confidence)
		supSpan.SetAttribute("needs_clarification", decision.NeedsClarification)
		supSpan.End()

		gateSpan := tracing.SpanFromContext(ctx, "policy_gate", tracing.KindChain)
		route := policy.Apply(decision, st)
		gateSpan.SetAttribute("primary_domain", route.PrimaryDomain)
		gateSpan.SetAttribute("task_intent", route.TaskIntent)
		gateSpan.SetAttribute("needs_clarification", route.NeedsClarification)
		gateSpan.SetAttribute("parallel_allowed", route.ParallelAllowed)
		if len(route.SecondaryDomains) > 0 {
			gateSpan.SetAttribute("secondary_domains", strings.Join(route.SecondaryDomains, ","))
		}
		gateSpan.End()

		// Record routing snapshot into session state.
		st.Routing = state.RoutingSnapshot{
			ConversationIntent:    route.ConversationIntent,
			PrimaryDomain:         route.PrimaryDomain,
			SecondaryDomains:      route.SecondaryDomains,
			TaskIntent:            route.TaskIntent,
			AwaitingClarification: route.NeedsClarification,
			Confidence:            decision.Confidence,
		}

		// ─── 确定性状态修正：LLM 负责内容分类，Go 负责状态判定 ───
		//
		// 行业实践 (Pattern 1: Routing): LLM 回答 "这条消息包含什么内容"
		// (collect_profile / followup)，代码回答 "当前状态下应该走哪个分支"
		// (amend_profile / fortune_followup)。LLM 不擅长跨轮状态比对，
		// 反复 prompt 调优也修不稳——确定性代码一行就够了。
		//
		// 规则 A: 会话已有资料 + LLM 判为 collect_profile → 纠正为 amend_profile
		//   场景: T1 存了 year=1990，T2 用户说 "5月20日早上8点，男，北京"
		//   场景: T1 排了盘，T2 用户说 "不对，我是女的" 或 "改成1991年"
		if route.TaskIntent == "collect_profile" && len(st.Profile) > 0 {
			route.TaskIntent = "amend_profile"
			route.PolicyHints.CanReuseSessionProfile = true
			if st.HasBaziResult() {
				route.PolicyHints.CanReuseCachedResult = true
			}
		}

		// 规则 B: 会话已有命盘 + 用户纯追问（无新出生时间）→ fortune_followup
		//   场景: T1 排了盘，T2 用户说 "今年运势怎么样" / "那明年呢" / "我适合做什么工作"
		if route.TaskIntent == "collect_profile" && st.HasBaziResult() && !containsBirthTime(message) {
			route.TaskIntent = "fortune_followup"
			route.PolicyHints.CanReuseCachedResult = true
			route.PolicyHints.CanReuseSessionProfile = true
		}

		// Capture the approved route for specialist dispatch below.
		approvedRoute = route

		// Extract execution slots from the policy-approved route.
		profilePatch, userQuestion, needsQimen, rawBazi = bridgeDecision(route, message)
		st.NeedsQimen = needsQimen
	} else {
		classifySpan := tracing.SpanFromContext(ctx, "classify_and_extract", tracing.KindChain)
		action, profilePatch, userQuestion, needsQimen, rawBazi = o.classifyAndExtract(ctx, message, st)
		st.NeedsQimen = needsQimen
		classifySpan.SetAttribute("action", action)
		classifySpan.SetAttribute("needs_qimen", needsQimen)
		classifySpan.End()
	}

	var turnErr error
	var turnType string
	var assistantText string
	var recordState *state.SessionState

	// Phase 1 specialist dispatch: when supervisor path is active and specialists
	// are wired, validate the route through the primary domain specialist before execution.
	var specialistFinal bool
	if o.supervisor != nil && (o.baziSp != nil || o.qimenSp != nil) {
		dispSpan := tracing.SpanFromContext(ctx, "domain_dispatch", tracing.KindChain)
		spRoute := specialists.ApprovedRoute{
			ConversationIntent:    st.Routing.ConversationIntent,
			PrimaryDomain:         st.Routing.PrimaryDomain,
			SecondaryDomains:      st.Routing.SecondaryDomains,
			TaskIntent:            st.Routing.TaskIntent,
			NeedsClarification:    st.Routing.AwaitingClarification,
			ClarificationQuestion: approvedRoute.ClarificationQuestion,
			ParallelAllowed:       false,
			Slots:                 approvedRoute.Slots,
			PolicyHints:           approvedRoute.PolicyHints,
		}
		primarySp := o.baziSp
		switch approvedRoute.PrimaryDomain {
		case "qimen":
			if o.qimenSp != nil {
				primarySp = o.qimenSp
			}
		case "ziwei":
			if o.ziweiSp != nil {
				primarySp = o.ziweiSp
			}
		case "bazi":
			if o.baziSp == nil && o.qimenSp != nil {
				primarySp = o.qimenSp
			}
		default:
			if primarySp == nil {
				primarySp = o.qimenSp
			}
		}

		var spResult schemas.DomainResult
		var spErr error
		if primarySp != nil {
			spResult, spErr = primarySp.Run(ctx, st, spRoute, specialistEventSink(sink))
			dispSpan.SetAttribute("primary_domain", spResult.Domain)
			dispSpan.SetAttribute("final", spResult.Final)
			if spErr != nil {
				dispSpan.SetAttribute("error", spErr.Error())
			}
		}
		dispSpan.End()

		// If specialist returned a final answer (e.g., clarification), short-circuit.
		if spResult.Final {
			sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": spResult.Summary}})
			assistantText = spResult.Summary
			turnType = "ask_missing_profile"
			recordState = st
			specialistFinal = true
		}

		// Secondary domain dispatch: qimen / ziwei as supplemental context.
		if !specialistFinal && len(st.Routing.SecondaryDomains) > 0 {
			for _, d := range st.Routing.SecondaryDomains {
				switch d {
				case "qimen":
					if o.qimenSp != nil {
						qimenSpResult, _ := o.qimenSp.Run(ctx, st, spRoute, specialistEventSink(sink))
						if qimenSpResult.Domain == "qimen" && !qimenSpResult.Final {
							st.NeedsQimen = true
						}
					}
				case "ziwei":
					if o.ziweiSp != nil {
						ziweiSpResult, _ := o.ziweiSp.Run(ctx, st, spRoute, specialistEventSink(sink))
						if ziweiSpResult.Domain == "ziwei" && !ziweiSpResult.Final {
							st.DomainStates.ZiWei.ChartReady = true
						}
					}
				}
			}
		}
	}

	// Route-driven execution (phase 1.5): dispatch directly from ApprovedRoute.
	// The route has already been validated by policy gate and specialist.
	if o.supervisor != nil {
		if !specialistFinal {
			turnType, assistantText, turnErr = o.executeRoute(ctx, sink, st, approvedRoute, profilePatch, userQuestion, rawBazi)
			recordState = st
		}
	} else {
		// Legacy path: classify → action string → switch.
		// Reclassify spurious new_profile when session already has data.
		if action == "new_profile" && (st.HasBaziResult() || len(st.Profile) > 0) && !containsBirthTime(message) {
			action = "update_profile"
		}

		if !specialistFinal {
		switch action {
	case "bazi_input":
		candidate := st.Clone()
		// Merge any gender extracted from the same message (e.g. "乙巳 丁亥 甲申 甲子，女")
		if g, ok := profilePatch["gender"]; ok {
			candidate.Profile["gender"] = g
		}
		turnType = "direct_bazi"
		assistantText, turnErr = o.handleBaziInput(ctx, sink, candidate, rawBazi)
		if candidate.HasBaziResult() {
			*st = *candidate
			recordState = st
		}

	case "new_profile":
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
			turnType = "ask_missing_profile"
			assistantText, turnErr = o.handleAsk(ctx, sink, candidate)
			if !st.HasBaziResult() && len(st.Profile) == 0 {
				*st = *candidate
				recordState = st
			}
		} else {
			turnType = "full_reading"
			assistantText, turnErr = o.handleFullReading(ctx, sink, candidate)
			if candidate.HasBaziResult() {
				*st = *candidate
				recordState = st
			}
		}

	case "update_profile":
		candidate := st.Clone()
		changed := candidate.MergeProfile(profilePatch)
		if userQuestion != "" {
			candidate.LastUserQuestion = userQuestion
		}
		if changed && profileChangesAffectChart(profilePatch) {
			candidate.BaziResult = nil
		}
		if !candidate.IsProfileComplete() && !candidate.HasBaziResult() {
			turnType = "ask_missing_profile"
			assistantText, turnErr = o.handleAsk(ctx, sink, candidate)
			*st = *candidate
			recordState = st
		} else if candidate.BaziResult == nil {
			turnType = "full_reading"
			assistantText, turnErr = o.handleFullReading(ctx, sink, candidate)
			if candidate.HasBaziResult() {
				*st = *candidate
				recordState = st
			}
		} else {
			turnType = "followup_reading"
			assistantText, turnErr = o.handleFollowupReading(ctx, sink, candidate)
			*st = *candidate
			recordState = st
		}

	case "incomplete":
		candidate := st.Clone()
		candidate.MergeProfile(profilePatch)
		if userQuestion != "" {
			candidate.LastUserQuestion = userQuestion
		}
		if candidate.HasBaziResult() {
			turnType = "followup_reading"
			assistantText, turnErr = o.handleFollowupReading(ctx, sink, candidate)
		} else {
			turnType = "ask_missing_profile"
			assistantText, turnErr = o.handleAsk(ctx, sink, candidate)
		}
		*st = *candidate
		recordState = st

	default: // "followup"
		candidate := st.Clone()
		if userQuestion != "" {
			candidate.LastUserQuestion = userQuestion
		}
		if candidate.HasBaziResult() {
			turnType = "followup_reading"
			assistantText, turnErr = o.handleFollowupReading(ctx, sink, candidate)
		} else if !candidate.IsProfileComplete() {
			turnType = "ask_missing_profile"
			assistantText, turnErr = o.handleAsk(ctx, sink, candidate)
		} else if candidate.BaziResult == nil {
			turnType = "full_reading"
			assistantText, turnErr = o.handleFullReading(ctx, sink, candidate)
			if candidate.HasBaziResult() {
				*st = *candidate
			}
		} else {
			turnType = "followup_reading"
			assistantText, turnErr = o.handleFollowupReading(ctx, sink, candidate)
		}
		if turnType == "followup_reading" || turnType == "ask_missing_profile" {
			*st = *candidate
			recordState = st
		}
		if turnType == "full_reading" && candidate.HasBaziResult() {
			recordState = st
		}
	}
	} // end if !specialistFinal (legacy)
	} // end else (legacy path)

if recordState != nil {
		o.recordTurnAndMaintainContext(ctx, recordState, message, assistantText)
	}

	// Set turn type and status on trace
	if t := tracing.TraceFromContext(ctx); t != nil {
		t.TurnType = turnType
	}
	if turnErr != nil {
		trace.SetStatus("error")
	}

	// Emit trace digest before done
	o.emitTraceDigest(ctx, sink, turnType)
	sink.Emit(ctx, Event{Type: "done", Data: map[string]any{}})
	return turnErr
}

var (
	lunarHintRe = regexp.MustCompile(`农历|阴历|正月|腊月`)
	yearRe      = regexp.MustCompile(`(\d{4})\s*年`)
	monthRe     = regexp.MustCompile(`(\d{1,2})\s*月`)
	dayRe       = regexp.MustCompile(`(\d{1,2})\s*[日号]`)
	morningRe   = regexp.MustCompile(`(?:早上|上午|早晨)\s*(\d{1,2})\s*点`)
	noonRe      = regexp.MustCompile(`中午\s*(\d{1,2})\s*点`)
	pmRe        = regexp.MustCompile(`(?:下午|晚上)\s*(\d{1,2})\s*点`)
	clockRe     = regexp.MustCompile(`(\d{1,2})(?::00|\s*[点时])`)
	timeRe      = regexp.MustCompile(`(\d{1,2}):(\d{2})`)
	genderRe    = regexp.MustCompile(`(?:性别[:：]?\s*)?(男|女)`)
)

func extractProfileAndQuestion(msg string) (profilePatch map[string]any, question string) {
	normalized := strings.TrimSpace(msg)
	residual := normalized
	patch := map[string]any{"calendar_type": "solar"}
	if lunarHintRe.MatchString(normalized) {
		patch["calendar_type"] = "lunar"
	}

	extractInt := func(re *regexp.Regexp, key string, min, max int) {
		matches := re.FindStringSubmatch(normalized)
		if len(matches) != 2 {
			return
		}
		v, err := strconv.Atoi(matches[1])
		if err != nil || v < min || v > max {
			return
		}
		patch[key] = float64(v)
		residual = strings.Replace(residual, matches[0], "", 1)
	}

	extractInt(yearRe, "year", 1900, 2100)
	extractInt(monthRe, "month", 1, 12)
	extractInt(dayRe, "day", 1, 31)

	hourFound := false
	hourPatterns := []struct {
		re   *regexp.Regexp
		base int
	}{
		{morningRe, 0},
		{noonRe, 0},
		{pmRe, 12},
		{clockRe, 0},
	}
	for _, hp := range hourPatterns {
		matches := hp.re.FindStringSubmatch(normalized)
		if len(matches) != 2 {
			continue
		}
		h, err := strconv.Atoi(matches[1])
		if err != nil {
			break
		}
		val := h + hp.base
		if hp.base == 12 && h == 12 {
			val = 12
		}
		if val >= 0 && val <= 23 {
			patch["hour"] = float64(val)
			residual = strings.Replace(residual, matches[0], "", 1)
			hourFound = true
		}
		break
	}
	if !hourFound {
		if matches := timeRe.FindStringSubmatch(normalized); len(matches) == 3 {
			h, _ := strconv.Atoi(matches[1])
			m, _ := strconv.Atoi(matches[2])
			if h >= 0 && h <= 23 && m >= 0 && m <= 59 {
				patch["hour"] = float64(h)
				patch["minute"] = float64(m)
				residual = strings.Replace(residual, matches[0], "", 1)
			}
		}
	}

	if matches := genderRe.FindStringSubmatch(normalized); len(matches) == 2 {
		patch["gender"] = matches[1]
		residual = strings.Replace(residual, matches[0], "", 1)
	}

	residual = strings.NewReplacer(
		"，", " ", ",", " ", "。", " ", "；", " ", "！", " ", "？", " ",
	).Replace(residual)
	residual = strings.Join(strings.Fields(residual), " ")
	return patch, residual
}

var fieldNames = map[string]string{
	"year": "出生年份", "month": "出生月份", "day": "出生日期",
	"hour": "出生时辰", "gender": "性别", "birthplace": "出生地（城市）",
}

// chartFields are the profile keys that affect the bazi chart calculation.
// Changes to these fields invalidate any cached BaziResult.
var chartFields = map[string]bool{"year": true, "month": true, "day": true, "hour": true}

// containsBirthTime checks whether a message contains birth time information
// (year + month + day pattern), as opposed to just metadata like gender.
var birthTimeRe = regexp.MustCompile(`\d{4}\s*年.*\d{1,2}\s*月|\d{4}[-/]\d{1,2}|农历|阴历|正月|腊月`)

func containsBirthTime(msg string) bool {
	return birthTimeRe.MatchString(msg)
}

// profileChangesAffectChart reports whether any profile patch key would change
// the eight-character chart. Gender, birthplace, calendar_type changes alone
// should NOT invalidate an existing BaziResult.
// Only returns true when a chart-affecting field has a meaningful non-zero value.
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

func (o *Orchestrator) handleBaziInput(ctx context.Context, sink EventSink, st *state.SessionState, rawBazi []string) (string, error) {
	parseSpan := tracing.SpanFromContext(ctx, "parse_direct_bazi", tracing.KindChain)
	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "识别到直接输入的四柱八字，开始分析...",
	}})

	pillarNames := []string{"年柱", "月柱", "日柱", "时柱"}
	pillars := make([]map[string]any, 4)
	for i := 0; i < 4; i++ {
		s := rawBazi[i]
		pillars[i] = map[string]any{
			"name":   pillarNames[i],
			"stem":   string([]rune(s)[0:1]),
			"branch": string([]rune(s)[1:2]),
		}
	}
	data := map[string]any{
		"pillars": pillars,
		"dayGan":  string([]rune(rawBazi[2])[0:1]),
	}
	st.BaziResult = data
	parseSpan.End()

	sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
		"type": "bazi-chart", "payload": data,
	}})

	// Gender is not encoded in the eight characters — prompt if missing.
	if _, hasGender := st.Profile["gender"]; !hasGender {
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{
			"content": "⚠️ 八字本身不含性别信息。请问这个八字是男命还是女命？（男女命的大运顺逆、婚姻用神均不同）",
		}})
	}

	passages := o.runKnowledgeSearch(ctx, sink, st, nil)
	fullText, err := o.streamInterpretation(ctx, sink, st, passages, nil, false)
	if err != nil {
		sink.Emit(ctx, Event{Type: "error", Data: map[string]any{"message": "解读失败: " + err.Error()}})
		return fullText, err
	}
	return fullText, nil
}

func (o *Orchestrator) handleAsk(ctx context.Context, sink EventSink, st *state.SessionState) (string, error) {
	askSpan := tracing.SpanFromContext(ctx, "ask_missing_profile", tracing.KindChain)
	defer askSpan.End()

	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "正在核实出生信息...",
	}})
	missing := st.MissingFields()
	askSpan.SetAttribute("missing_fields", missing)

	names := make([]string, len(missing))
	for i, f := range missing {
		if n, ok := fieldNames[f]; ok {
			names[i] = n
		} else {
			names[i] = f
		}
	}
	var prompt string
	if len(names) == 0 {
		prompt = "请告诉我你的出生信息（至少需要年份、月份、日期、时辰和性别），例如：1990年5月20日早上8点，男"
	} else {
		prompt = fmt.Sprintf("请告诉我你的%s", strings.Join(names, "、"))
	}
	sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": prompt}})
	return prompt, nil
}

func (o *Orchestrator) runKnowledgeSearch(ctx context.Context, sink EventSink, st *state.SessionState, qimenData map[string]any) []mcp.Passage {
	ksSpan := tracing.SpanFromContext(ctx, "knowledge_search", tracing.KindRetriever)
	defer ksSpan.End()

	tool, ok := o.tools.Get("knowledge_search")
	if !ok {
		ksSpan.SetStatus("degraded")
		ksSpan.SetAttribute("degrade_reason", "tool_not_registered")
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "知识检索未注册，跳过引用检索。",
		}})
		return []mcp.Passage{}
	}

	query := o.buildKnowledgeQuery(ctx, st, qimenData)
	ksSpan.SetAttribute("query", query)
	sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{
		"tool":   "knowledge_search",
		"params": map[string]any{"query": query, "topK": 5},
	}})

	result, err := tool.Execute(ctx, map[string]any{"query": query, "topK": 5})
	if err != nil {
		ksSpan.SetStatus("degraded")
		ksSpan.SetAttribute("degrade_reason", "exec_error")
		ksSpan.RecordError(err)
		sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
			"agent": "orchestrator", "text": "知识检索失败，继续直接解读命盘。",
		}})
		return []mcp.Passage{}
	}

	payload, ok := result.(map[string]any)
	if !ok {
		ksSpan.SetStatus("degraded")
		ksSpan.SetAttribute("degrade_reason", "result_type_invalid")
		return []mcp.Passage{}
	}
	passages, _ := payload["passages"].([]mcp.Passage)
	ksSpan.SetAttribute("hits", len(passages))
	if len(passages) == 0 {
		ksSpan.SetStatus("degraded")
		ksSpan.SetAttribute("degrade_reason", "no_results")
	}
	if len(passages) > 0 {
		sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
			"type":    "knowledge-sources",
			"payload": passages,
		}})
	}
	return passages
}

func (o *Orchestrator) streamInterpretation(ctx context.Context, sink EventSink, st *state.SessionState, passages []mcp.Passage, extra map[string]any, qimenPrimary bool) (string, error) {
	llmSpan := tracing.SpanFromContext(ctx, "llm_generate", tracing.KindLLM)
	if o.llmModel != "" {
		llmSpan.SetAttribute("model", o.llmModel)
	}
	llmSpan.SetAttribute("output_tokens", nil) // unavailable in streaming mode

	systemPrompt := o.buildInterpretPrompt(st, passages, extra, qimenPrimary)
	messages := []llm.Message{
		{Role: "user", Content: currentQuestion(st)},
	}

	var tail strings.Builder
	var fullText strings.Builder
	blocked := false

	err := o.llm.Stream(ctx, systemPrompt, messages, func(chunk string) {
		if blocked {
			return
		}
		tail.WriteString(chunk)
		t := tail.String()
		if len(t) > 40 {
			t = t[len(t)-40:]
		}
		if strings.Contains(t, "仅供") || strings.Contains(t, "AI生成") || strings.Contains(t, "玄学算命") || strings.Contains(t, "以上内容由") {
			blocked = true
			return
		}
		fullText.WriteString(chunk)
		sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": chunk}})
	})

	if err != nil {
		llmSpan.RecordError(err)
	}
	llmSpan.End()
	return fullText.String(), err
}

func (o *Orchestrator) handleFullReading(ctx context.Context, sink EventSink, st *state.SessionState) (string, error) {
	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "信息齐全，开始排盘...",
	}})

	// bazi_calc
	baziSpan := tracing.SpanFromContext(ctx, "bazi_calc", tracing.KindTool)
	tool, ok := o.tools.Get("bazi_calc")
	if !ok {
		baziSpan.RecordError(fmt.Errorf("not registered"))
		baziSpan.End()
		sink.Emit(ctx, Event{Type: "error", Data: map[string]any{"message": "tool bazi_calc not registered"}})
		return "", fmt.Errorf("bazi_calc not registered")
	}
	sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{"tool": "bazi_calc", "params": st.Profile}})
	result, err := tool.Execute(ctx, st.Profile)
	if err != nil {
		baziSpan.RecordError(err)
		baziSpan.End()
		sink.Emit(ctx, Event{Type: "error", Data: map[string]any{"message": "排盘失败: " + err.Error()}})
		return "", err
	}
	data, ok := result.(map[string]any)
	if !ok {
		baziSpan.RecordError(fmt.Errorf("result type invalid"))
		baziSpan.End()
		sink.Emit(ctx, Event{Type: "error", Data: map[string]any{"message": "排盘结果格式错误"}})
		return "", fmt.Errorf("bazi_calc result type invalid")
	}
	st.BaziResult = data
	baziSpan.End()

	// yongshen (optional)
	if ysTool, ok := o.tools.Get("yongshen"); ok {
		func() {
			ysSpan := tracing.SpanFromContext(ctx, "yongshen", tracing.KindTool)
			defer ysSpan.End()
			ysResult, ysErr := ysTool.Execute(ctx, st.Profile)
			if ysErr != nil {
				ysSpan.RecordError(ysErr)
				return
			}
			if ysMap, ok2 := ysResult.(map[string]any); ok2 {
				st.BaziResult["yongshen"] = ysMap
				ysSpan.SetAttribute("day_master", ysMap["day_master"])
				ysSpan.SetAttribute("strength", ysMap["strength"])
				sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{"tool": "yongshen", "params": st.Profile}})
				sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
					"agent": "orchestrator",
					"text":  fmt.Sprintf("日主%s 强弱:%s 用神:%v 忌神:%v", ysMap["day_master"], ysMap["strength"], ysMap["yong_shen"], ysMap["ji_shen"]),
				}})
			}
		}()
	}

	// dayun_analyzer (optional)
	if daTool, ok2 := o.tools.Get("dayun_analyzer"); ok2 {
		func() {
			daSpan := tracing.SpanFromContext(ctx, "dayun_analyzer", tracing.KindTool)
			defer daSpan.End()
			daParams := map[string]any{
				"dayun":       data["dayun"],
				"bazi_result": st.BaziResult,
			}
			if daResult, daErr := daTool.Execute(ctx, daParams); daErr == nil {
				if daMap, ok3 := daResult.(map[string]any); ok3 {
					st.BaziResult["dayun_analyzed"] = daMap["dayun_analyzed"]
				}
			} else {
				daSpan.RecordError(daErr)
			}
		}()
	}

	sink.Emit(ctx, Event{Type: "component", Data: map[string]any{"type": "bazi-chart", "payload": data}})

	passages := o.runKnowledgeSearch(ctx, sink, st, nil)
	fullText, err := o.streamInterpretation(ctx, sink, st, passages, nil, false)
	if err != nil {
		sink.Emit(ctx, Event{Type: "error", Data: map[string]any{"message": "LLM 解读失败: " + err.Error()}})
		return fullText, err
	}
	return fullText, nil
}

func (o *Orchestrator) handleFollowupReading(ctx context.Context, sink EventSink, st *state.SessionState) (string, error) {
	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "复用已有命盘...",
	}})

	// reuse_bazi span
	reuseSpan := tracing.SpanFromContext(ctx, "reuse_bazi_result", tracing.KindChain)
	reuseSpan.SetAttribute("bazi_reused", true)
	reuseSpan.End()

	var extraPromptData map[string]any
	if st.NeedsQimen {
		if qimenTool, ok := o.tools.Get("qimen_dunjia"); ok {
			func() {
				qmSpan := tracing.SpanFromContext(ctx, "qimen_dunjia", tracing.KindTool)
				defer qmSpan.End()

				now := resolveQimenTime(time.Now())
				qimenParams := qimenTools.ResolveTime(now)
				sink.Emit(ctx, Event{Type: "tool_call", Data: map[string]any{
					"tool": "qimen_dunjia", "params": qimenParams,
				}})
				qimenResult, qimenErr := qimenTool.Execute(ctx, qimenParams)
				if qimenErr == nil {
					if qm, ok2 := qimenResult.(map[string]any); ok2 {
						sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
							"type": "qimen-chart", "payload": qm,
						}})
						extraPromptData = qm
					}
				} else {
					qmSpan.SetStatus("fallback")
					qmSpan.RecordError(qimenErr)
					sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
						"agent": "orchestrator", "text": "奇门排盘失败，改按八字继续分析。",
					}})
				}
			}()
		}
	}

	var passages []mcp.Passage
	// Always run knowledge search on followup — the query builder already
	// targets the chart + question, and even "simple" questions benefit from
	// domain references (marriage patterns, career profiles, year-event cases).
	passages = o.runKnowledgeSearch(ctx, sink, st, nil)
	fullText, err := o.streamInterpretation(ctx, sink, st, passages, extraPromptData, false)
	if err != nil {
		sink.Emit(ctx, Event{Type: "error", Data: map[string]any{"message": "LLM 解读失败: " + err.Error()}})
		return fullText, err
	}
	return fullText, nil
}

// emitTraceDigest builds a user-facing digest from the TurnTrace and sends it via component SSE.
func (o *Orchestrator) emitTraceDigest(ctx context.Context, sink EventSink, turnType string) {
	t := tracing.TraceFromContext(ctx)
	if t == nil {
		return
	}

	if t.TurnType == "" {
		t.TurnType = turnType
	}

	digest := t.BuildDigest()
	sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
		"type":    "trace-panel",
		"payload": digest,
	}})
}

// recordTurnAndMaintainContext records user and assistant turns, then trims
// the turn window and updates the running summary when the window overflows.
func (o *Orchestrator) recordTurnAndMaintainContext(ctx context.Context, st *state.SessionState, userMsg, assistantMsg string) {
	if userMsg != "" {
		st.RecordTurn("user", userMsg)
	}
	if assistantMsg != "" {
		st.RecordTurn("assistant", assistantMsg)
	}
	overflow := st.TrimTurns()
	if len(overflow) == 0 {
		return
	}
	if o.flash == nil {
		st.RecentTurns = append(overflow, st.RecentTurns...)
		return
	}
	summary, ok := o.summarizeTurns(ctx, st.RunningSummary, overflow)
	if !ok {
		st.RecentTurns = append(overflow, st.RecentTurns...)
		return
	}
	st.RunningSummary = summary
}
