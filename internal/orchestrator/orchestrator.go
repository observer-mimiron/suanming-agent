// Package orchestrator 实现命理咨询对话的核心编排逻辑。
//
// 管理用户对话的完整生命周期：加载会话状态、路由意图（通过 supervisor 或降级 classify）、
// 调度领域专家、执行工具、流式 LLM 解读、推送 SSE 事件、持久化更新后的状态。
// 所有公开方法在单会话内串行安全（由会话级锁保证），不同 sessionID 间可并发调用。
package orchestrator

import (
	"context"
	"fmt"
	"log"
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

// Supervisor 是 LLM 驱动的路由决策接口。
// 唯一实现是 supervisor.Client；测试使用 mock。
type Supervisor interface {
	Decide(ctx context.Context, msg string, st *state.SessionState) (schemas.SupervisorDecision, error)
}

// specialistEventSink 将编排器的 EventSink 接口适配为 DomainHandler 使用的 specialists.EventSink 函数类型。
func specialistEventSink(sink EventSink) specialists.EventSink {
	return func(ctx context.Context, evt specialists.Event) error {
		return sink.Emit(ctx, Event{Type: evt.Type, Data: evt.Data})
	}
}

// Orchestrator 是单个用户会话的中央协调器。
// 它拥有工具注册表、LLM 客户端、状态存储/锁、以及可选的 supervisor 和领域专家。
// 每次 Run 调用处理一条用户消息，并通过提供的 EventSink 产生 SSE 事件。
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

// New 使用给定的依赖创建 Orchestrator。
// flashClient 是用于轻量级分类和摘要任务的快速 LLM。
// llmClient 和 flashClient 都必须非空。
// Supervisor 和专家通过 SetSupervisor 和 SetSpecialists 分别注入。
// promptMode 控制解读提示词的选择（"direct" 表示面向基准测试的模式，vs 默认的解读流程）。
func New(reg *tools.Registry, llmClient llm.Chat, flashClient llm.Chat, store state.Store, locker state.Locker, tracer tracing.Tracer, promptMode string) *Orchestrator {
	return &Orchestrator{tools: reg, llm: llmClient, flash: flashClient, store: store, locker: locker, tracer: tracer, promptMode: promptMode}
}

// SetLLMModel 设置用于 LLM span 元数据的模型名称。
func (o *Orchestrator) SetLLMModel(model string) { o.llmModel = model }

// SetSupervisor 注入 supervisor 客户端用于阶段一路由。
// 为 nil 时，编排器回退到传统的 classifyAndExtract 路径。
func (o *Orchestrator) SetSupervisor(sv Supervisor) { o.supervisor = sv }

// SetSpecialists 注入八字、奇门和紫微斗数领域专家。
func (o *Orchestrator) SetSpecialists(baziSp, qimenSp, ziweiSp specialists.DomainHandler) {
	o.baziSp = baziSp
	o.qimenSp = qimenSp
	o.ziweiSp = ziweiSp
}

// Run 处理会话中的一条用户消息。这是主入口点，对于不同 sessionID 的并发调用是安全的，
// 因为会话状态通过 Locker 串行化。流程如下：
//
//  1. 获取会话锁（阻塞同一会话的并发请求）。
//  2. 加载或创建会话状态，启动 OpenTelemetry 跟踪。
//  3. 阶段一路由：如果设置了 supervisor，调用 Decide + 策略门生成 ApprovedRoute；否则回退到 classifyAndExtract。
//  4. 阶段一专家分发：通过主领域专家（八字/奇门/紫微）验证路由，并运行辅助领域。
//  5. 阶段 1.5 路由执行：执行工具、构建提示词、流式 LLM 解读、发送 SSE 事件（thinking、tool_call、component、text）。
//  6. 记录轮次，维护上下文窗口（溢出时裁剪 + 摘要）。
//  7. 发送跟踪摘要和完成事件。
//
// 工具执行或 LLM 流式传输的错误返回给调用方；部分助手文本可能仍然存在。sessionID 必须非空。
func (o *Orchestrator) Run(ctx context.Context, sink EventSink, sessionID, message string) error {
	unlock := o.locker.Lock(sessionID)
	defer unlock()

	ctx, trace := o.tracer.StartTrace(ctx, "chat.turn")
	defer trace.End()

	st := o.store.LoadOrCreate(sessionID)
	defer o.store.Save(st)

	// 使用元数据注释根跟踪
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

	// 阶段一：使用 supervisor（若可用），否则回退到传统 classify。
	if o.supervisor != nil {
		supSpan := tracing.SpanFromContext(ctx, "supervisor_decision", tracing.KindChain)
		decision, err := o.supervisor.Decide(ctx, message, st)
		if err != nil {
			supSpan.SetAttribute("error", err.Error())
			supSpan.SetStatus("degraded")
			log.Printf("[orchestrator] supervisor decision failed: %v", err)
			sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
				"agent": "orchestrator", "text": "⚠️ 服务暂时降级，使用保守策略继续。如持续出现请稍后重试。",
			}})
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

		// 规则 C: 当前会话尚无可用资料，但用户这轮消息里已经带了出生时间，
		// 若 supervisor 仍把它判成 interpret/followup 类问题，就强制拉回
		// collect_profile，并从原文做一次确定性提取补齐 profile patch。
		profileReady := st.IsProfileComplete() || st.HasBaziResult()
		if !profileReady && containsBirthTime(message) &&
			route.TaskIntent != "collect_profile" &&
			route.TaskIntent != "amend_profile" &&
			route.TaskIntent != "direct_bazi" {
			patch, extractedQuestion := extractProfileAndQuestion(message)
			if route.Slots.Profile == nil {
				route.Slots.Profile = make(map[string]any)
			}
			for k, v := range patch {
				if _, ok := route.Slots.Profile[k]; !ok {
					route.Slots.Profile[k] = v
				}
			}
			if route.Slots.QuestionText == "" || route.Slots.QuestionText == message {
				route.Slots.QuestionText = extractedQuestion
			}
			route.TaskIntent = "collect_profile"
			route.NeedsClarification = false
			route.ClarificationQuestion = ""
		}

		// 将路由快照记录到会话状态中。
		st.Routing = state.RoutingSnapshot{
			ConversationIntent:    route.ConversationIntent,
			PrimaryDomain:         route.PrimaryDomain,
			SecondaryDomains:      route.SecondaryDomains,
			TaskIntent:            route.TaskIntent,
			AwaitingClarification: route.NeedsClarification,
			Confidence:            decision.Confidence,
			TimeScope:             route.Slots.TimeScope,
			TargetSubject:         route.Slots.TargetSubject,
		}

		// 捕获已批准的路由用于后续的专家分发。
		approvedRoute = route

		// 从策略批准的路由中提取执行槽位。
		action, profilePatch, userQuestion, needsQimen, rawBazi = bridgeDecision(route, message)
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

	// 阶段一专家分发：当 supervisor 路径激活且有专家时，
	// 在执行前通过主领域专家验证路由。
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

		// 如果专家返回了最终答案（如澄清），则短路处理。
		if spResult.Final {
			sink.Emit(ctx, Event{Type: "text", Data: map[string]any{"content": spResult.Summary}})
			assistantText = spResult.Summary
			turnType = "ask_missing_profile"
			recordState = st
			specialistFinal = true
		}

		// 辅助领域分发：奇门 / 紫微作为补充上下文。
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

	// 路由驱动执行（阶段 1.5）：直接从 ApprovedRoute 分发。
	// 路由已由策略门和专家验证通过。
	if o.supervisor != nil {
		if !specialistFinal {
			turnType, assistantText, turnErr = o.executeRoute(ctx, sink, st, approvedRoute, profilePatch, userQuestion, rawBazi)
			recordState = st
		}
	} else {
		// 传统路径：classify → action 字符串 → switch。
		// 当会话已有资料时，重新分类误判的 new_profile。
		if action == "new_profile" && (st.HasBaziResult() || len(st.Profile) > 0) && !containsBirthTime(message) {
			action = "update_profile"
		}

		if !specialistFinal {
			switch action {
			case "bazi_input":
				candidate := st.Clone()
				// 合并从同一条消息中提取的性别字段（如 "乙巳 丁亥 甲申 甲子，女"）
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

	// 在跟踪上设置轮次类型和状态
	if t := tracing.TraceFromContext(ctx); t != nil {
		t.TurnType = turnType
	}
	if turnErr != nil {
		trace.SetStatus("error")
	}

	// 在完成前发送跟踪摘要
	o.emitTraceDigest(ctx, sink, turnType)
	sink.Emit(ctx, Event{Type: "done", Data: map[string]any{}})
	return turnErr
}

var (
	lunarHintRe   = regexp.MustCompile(`农历|阴历|正月|腊月`)
	yearRe        = regexp.MustCompile(`(\d{4})\s*年`)
	monthRe       = regexp.MustCompile(`(\d{1,2})\s*月`)
	dayRe         = regexp.MustCompile(`(\d{1,2})\s*[日号]`)
	morningRe     = regexp.MustCompile(`(?:早上|上午|早晨)\s*(\d{1,2})\s*点`)
	noonRe        = regexp.MustCompile(`中午\s*(\d{1,2})\s*点`)
	pmRe          = regexp.MustCompile(`(?:下午|晚上)\s*(\d{1,2})\s*点`)
	clockRe       = regexp.MustCompile(`(\d{1,2})(?::00|\s*[点时])`)
	timeRe        = regexp.MustCompile(`(\d{1,2}):(\d{2})`)
	genderRe      = regexp.MustCompile(`(?:性别[:：]?\s*)?(男|女)`)
	birthplaceRes = []*regexp.Regexp{
		regexp.MustCompile(`出生地[:：]?\s*([A-Za-z\p{Han}]{2,16})`),
		regexp.MustCompile(`在\s*([A-Za-z\p{Han}]{2,16})\s*出生`),
		regexp.MustCompile(`([A-Za-z\p{Han}]{2,16})(?:人|出生)`),
	}
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
	if birthplace, consumed := extractBirthplace(normalized, residual); birthplace != "" {
		patch["birthplace"] = birthplace
		if consumed != "" {
			residual = strings.Replace(residual, consumed, "", 1)
		}
	}

	residual = strings.NewReplacer(
		"，", " ", ",", " ", "。", " ", "；", " ", "！", " ", "？", " ",
	).Replace(residual)
	residual = strings.Join(strings.Fields(residual), " ")
	return patch, residual
}

func extractBirthplace(normalized, residual string) (birthplace, consumed string) {
	for _, re := range birthplaceRes {
		matches := re.FindStringSubmatch(normalized)
		if len(matches) != 2 {
			continue
		}
		candidate := strings.TrimSpace(matches[1])
		if candidate == "" {
			continue
		}
		return candidate, matches[0]
	}

	tokens := strings.Fields(strings.NewReplacer(
		"，", " ", ",", " ", "。", " ", "；", " ", "！", " ", "？", " ",
	).Replace(residual))
	for _, token := range tokens {
		candidate := strings.TrimSpace(strings.Trim(token, "，,。；！？:："))
		if !birthplaceTokenRe.MatchString(candidate) {
			continue
		}
		if _, skip := birthplaceStopwords[candidate]; skip {
			continue
		}
		return candidate, candidate
	}

	return "", ""
}

var fieldNames = map[string]string{
	"year": "出生年份", "month": "出生月份", "day": "出生日期",
	"hour": "出生时辰", "gender": "性别", "birthplace": "出生地（城市）",
}

var birthplaceTokenRe = regexp.MustCompile(`^[A-Za-z\p{Han}]{2,16}$`)

var birthplaceStopwords = map[string]struct{}{
	"看看":   {},
	"事业":   {},
	"财运":   {},
	"婚姻":   {},
	"感情":   {},
	"健康":   {},
	"分析":   {},
	"运势":   {},
	"请问":   {},
	"老师":   {},
	"出生":   {},
	"出生地":  {},
	"事业发展": {},
}

// chartFields 是影响八字排盘计算的资料键。
// 这些字段的变化会使缓存的 BaziResult 失效。
var chartFields = map[string]bool{"year": true, "month": true, "day": true, "hour": true}

// containsBirthTime 通过正则匹配检查消息中是否包含出生时间信息
//（年 + 月 + 日模式），区别于仅有的元数据如性别。
var birthTimeRe = regexp.MustCompile(`\d{4}\s*年.*\d{1,2}\s*月|\d{4}[-/]\d{1,2}|农历|阴历|正月|腊月`)

// containsBirthTime 报告消息中是否包含年月日模式或农历关键词，表示存在出生时间数据。
func containsBirthTime(msg string) bool {
	return birthTimeRe.MatchString(msg)
}

// profileChangesAffectChart 报告资料的补丁键是否会改变八字命盘。
// 性别、出生地、日历类型等变化本身不应该使现有 BaziResult 失效。
// 仅当影响命盘的字段有意义的非零值时才返回 true。
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

	// 性别不编码在八字中——如果缺失则提示。
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
	systemPrompt := o.buildInterpretPrompt(st, passages, extra, qimenPrimary)
	messages := []llm.Message{
		{Role: "user", Content: currentQuestion(st)},
	}

	var tail strings.Builder
	var fullText strings.Builder
	blocked := false

	var llmSpan tracing.Span
	if llm.IsEinoChat(o.llm) {
		attrs := map[string]any{}
		if o.llmModel != "" {
			attrs["model"] = o.llmModel
		}
		ctx = tracing.WithEinoCallbackSpan(ctx, tracing.EinoCallbackSpanConfig{
			Name:       "llm_generate",
			Kind:       tracing.KindLLM,
			Attributes: attrs,
		})
	} else {
		llmSpan = tracing.SpanFromContext(ctx, "llm_generate", tracing.KindLLM)
		if o.llmModel != "" {
			llmSpan.SetAttribute("model", o.llmModel)
		}
		llmSpan.SetAttribute("output_tokens", nil) // unavailable in streaming mode
	}

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

	if llmSpan != nil && err != nil {
		llmSpan.RecordError(err)
	}
	if llmSpan != nil {
		llmSpan.End()
	}
	return fullText.String(), err
}

func (o *Orchestrator) handleFullReading(ctx context.Context, sink EventSink, st *state.SessionState) (string, error) {
	sink.Emit(ctx, Event{Type: "thinking", Data: map[string]any{
		"agent": "orchestrator", "text": "信息齐全，开始排盘...",
	}})

	// 八字排盘工具
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

	// 用神分析（可选）
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

	// 大运分析（可选）
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

	// 复用八字 span
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
						extraPromptData = qm
						if !st.HasQimenResult() {
							sink.Emit(ctx, Event{Type: "component", Data: map[string]any{
								"type": "qimen-chart", "payload": qm,
							}})
						}
						st.QimenResult = qm
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
	// 后续轮次始终执行知识搜索——查询构建器已针对
	// 命盘+问题优化，"简单"问题也能从命理典籍参考（婚姻模式、事业描述、流年案例）中受益。
	// 命理典籍参考（婚姻模式、事业描述、流年案例）
	passages = o.runKnowledgeSearch(ctx, sink, st, nil)
	fullText, err := o.streamInterpretation(ctx, sink, st, passages, extraPromptData, false)
	if err != nil {
		sink.Emit(ctx, Event{Type: "error", Data: map[string]any{"message": "LLM 解读失败: " + err.Error()}})
		return fullText, err
	}
	return fullText, nil
}

// emitTraceDigest 从 TurnTrace 构建面向用户的摘要，并通过 component SSE 事件发送。
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

// recordTurnAndMaintainContext 记录用户和助手的对话轮次，然后裁剪轮次窗口，
// 并在窗口溢出时更新滚动摘要。
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
