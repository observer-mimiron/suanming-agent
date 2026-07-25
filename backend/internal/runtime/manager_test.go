package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/observer-mimiron/suanming-agent/internal/llm"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

func TestManager_ComposesFinalReplyFromSpecialistResult(t *testing.T) {
	manager := &Manager{}
	result := specialists.Result{
		Domain:       "bazi",
		Summary:      "summary",
		ManagerBrief: "focus on the user's wealth question",
	}

	reply := manager.ComposeFinalReply("wealth", result)
	if !strings.Contains(reply, "wealth") {
		t.Fatalf("reply = %q, want follow-up question in final reply", reply)
	}
	if !strings.Contains(reply, "bazi") {
		t.Fatalf("reply = %q, want domain hint in final reply", reply)
	}
	if !strings.Contains(reply, "focus on the user's wealth question") {
		t.Fatalf("reply = %q, want manager brief in final reply", reply)
	}
}

func TestManager_ReconcileRoute_ReusesExistingProfile(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	st.Profile = map[string]any{"year": 1991.0}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}

	got := manager.ReconcileRoute(st, route, "I was born in 1991")
	if got.TaskIntent != "amend_profile" {
		t.Fatalf("TaskIntent = %q, want amend_profile", got.TaskIntent)
	}
	if !got.PolicyHints.CanReuseSessionProfile {
		t.Fatal("CanReuseSessionProfile = false, want true")
	}
}

func TestManager_ReconcileRoute_ConvertsInterpretToFollowupWhenChartExists(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	st.BaziResult = map[string]any{"dayGan": "jia"}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "interpret_chart",
	}

	got := manager.ReconcileRoute(st, route, "How is wealth over the next two years?")
	if got.TaskIntent != "fortune_followup" {
		t.Fatalf("TaskIntent = %q, want fortune_followup", got.TaskIntent)
	}
	if !got.PolicyHints.CanReuseCachedResult {
		t.Fatal("CanReuseCachedResult = false, want true")
	}
	if !got.PolicyHints.CanReuseSessionProfile {
		t.Fatal("CanReuseSessionProfile = false, want true")
	}
}

func TestManager_BuildExecutionPlan_UsesMainRuntimePathForMultiDomain(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	route := policy.ApprovedRoute{
		PrimaryDomain:    "bazi",
		SecondaryDomains: []string{"ziwei"},
		TaskIntent:       "cross_domain_consult",
	}

	plan := manager.BuildExecutionPlan(st, route, "show me the full picture")
	if len(plan.Domains) != 2 {
		t.Fatalf("len(Domains) = %d, want 2", len(plan.Domains))
	}
	if plan.Domains[0] != "bazi" || plan.Domains[1] != "ziwei" {
		t.Fatalf("Domains = %v, want [bazi ziwei]", plan.Domains)
	}
	if len(plan.RequiredArtifacts) != 2 {
		t.Fatalf("len(RequiredArtifacts) = %d, want 2", len(plan.RequiredArtifacts))
	}
	if plan.RequiredArtifacts[0] != artifactBaziChart || plan.RequiredArtifacts[1] != artifactZiweiChart {
		t.Fatalf("RequiredArtifacts = %v, want [%s %s]", plan.RequiredArtifacts, artifactBaziChart, artifactZiweiChart)
	}
	if plan.Snapshot.PrimaryDomain != "bazi" {
		t.Fatalf("Snapshot.PrimaryDomain = %q, want bazi", plan.Snapshot.PrimaryDomain)
	}
	if len(plan.Snapshot.Domains) != 2 || plan.Snapshot.Domains[1] != "ziwei" {
		t.Fatalf("Snapshot.Domains = %v, want [bazi ziwei]", plan.Snapshot.Domains)
	}
	if len(plan.Snapshot.RequiredArtifacts) != 2 || plan.Snapshot.RequiredArtifacts[1] != artifactZiweiChart {
		t.Fatalf("Snapshot.RequiredArtifacts = %v, want bazi/ziwei artifacts", plan.Snapshot.RequiredArtifacts)
	}
}

func TestManager_BuildExecutionPlan_UsesMainRuntimePathForSingleDomain(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	plan := manager.BuildExecutionPlan(st, route, "How is career?")
	if len(plan.Domains) != 1 || plan.Domains[0] != "bazi" {
		t.Fatalf("Domains = %v, want [bazi]", plan.Domains)
	}
	if len(plan.RequiredArtifacts) != 1 || plan.RequiredArtifacts[0] != artifactBaziChart {
		t.Fatalf("RequiredArtifacts = %v, want [%s]", plan.RequiredArtifacts, artifactBaziChart)
	}
	if plan.FollowupMode != followupModeRerunSpecialist {
		t.Fatalf("FollowupMode = %q, want %q", plan.FollowupMode, followupModeRerunSpecialist)
	}
	if plan.Snapshot.FollowupMode != followupModeRerunSpecialist {
		t.Fatalf("Snapshot.FollowupMode = %q, want %q", plan.Snapshot.FollowupMode, followupModeRerunSpecialist)
	}
}

func TestManager_BuildExecutionPlan_DirectsBaziGlossaryFollowup(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	st.BaziResult = map[string]any{"dayGan": "jia"}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	plan := manager.BuildExecutionPlan(st, route, "财星破印是啥意思")
	if plan.FollowupMode != followupModeDirect {
		t.Fatalf("FollowupMode = %q, want %q", plan.FollowupMode, followupModeDirect)
	}
	if plan.FollowupDirectAnswer == "" {
		t.Fatal("FollowupDirectAnswer = empty, want direct glossary answer")
	}
	if plan.Snapshot.FollowupMode != followupModeDirect {
		t.Fatalf("Snapshot.FollowupMode = %q, want %q", plan.Snapshot.FollowupMode, followupModeDirect)
	}
}

func TestManager_BuildExecutionPlan_KeepsChartSpecificBaziFollowupOnExecutionPath(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	st.BaziResult = map[string]any{"dayGan": "jia"}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	plan := manager.BuildExecutionPlan(st, route, "那我这盘里为什么算财星破印")
	if plan.FollowupMode != followupModeRerunSpecialist {
		t.Fatalf("FollowupMode = %q, want %q", plan.FollowupMode, followupModeRerunSpecialist)
	}
	if plan.FollowupDirectAnswer != "" {
		t.Fatalf("FollowupDirectAnswer = %q, want empty", plan.FollowupDirectAnswer)
	}
}

func TestManager_BuildExecutionPlan_ReusesCachedSingleDomainInterpretation(t *testing.T) {
	manager := &Manager{
		flash: &llm.NoopClient{
			GenerateFn: func(_ context.Context, systemPrompt string, messages []llm.Message) (string, llm.TokenUsage, error) {
				if !strings.Contains(systemPrompt, "不要重新排盘") {
					t.Fatalf("systemPrompt = %q, want no-rerun rule", systemPrompt)
				}
				if len(messages) != 1 || !strings.Contains(messages[0].Content, "上轮解读摘要：上轮已经判断事业主线可走稳") {
					t.Fatalf("messages = %+v, want cached interpretation in prompt", messages)
				}
				return "沿着上一轮结论继续看，事业主线还是以稳步推进为主。", llm.TokenUsage{}, nil
			},
		},
	}
	st := state.NewSession("s1")
	st.DomainContexts.Bazi.RuntimeValues = map[string]any{
		followupArtifactKey: map[string]any{
			"domain":  "bazi",
			"summary": "上轮已经判断事业主线可走稳",
		},
	}
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	plan := manager.BuildExecutionPlan(st, route, "那事业具体怎么推进")
	if plan.FollowupMode != followupModeReuseArtifact {
		t.Fatalf("FollowupMode = %q, want %q", plan.FollowupMode, followupModeReuseArtifact)
	}
	if !strings.Contains(plan.FollowupDirectAnswer, "稳步推进") {
		t.Fatalf("FollowupDirectAnswer = %q, want reused interpretation answer", plan.FollowupDirectAnswer)
	}
}

func TestManager_BuildExecutionPlan_DoesNotReuseCachedInterpretationForCrossDomainFollowup(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	st.DomainContexts.Bazi.RuntimeValues = map[string]any{
		followupArtifactKey: map[string]any{
			"domain":  "bazi",
			"summary": "上轮已经判断事业主线可走稳",
		},
	}
	route := policy.ApprovedRoute{
		PrimaryDomain:    "bazi",
		SecondaryDomains: []string{"ziwei"},
		TaskIntent:       "fortune_followup",
		PolicyHints: schemas.PolicyHints{
			QimenMode: "none",
		},
	}

	plan := manager.BuildExecutionPlan(st, route, "八字和紫微一起补充看看")
	if plan.FollowupMode == followupModeReuseArtifact {
		t.Fatalf("FollowupMode = %q, want not reuse for cross-domain follow-up", plan.FollowupMode)
	}
}

func TestManager_ComposesExistingFinalTextVerbatimWhenNoManagerBrief(t *testing.T) {
	manager := &Manager{}
	result := specialists.Result{
		Domain:  "bazi",
		Summary: "This is the guarded final reply.",
	}

	reply := manager.ComposeFinalReply("wealth", result)
	if reply != "This is the guarded final reply." {
		t.Fatalf("reply = %q, want unchanged guarded final text", reply)
	}
}

func TestManager_ComposesStructuredResultWhenSummaryMissing(t *testing.T) {
	manager := &Manager{}
	result := specialists.Result{
		Domain:          "bazi",
		DirectAnswer:    "整体来看，这一步可以推进。",
		KeyPoints:       []string{"事业主线偏稳", "节奏宜循序渐进"},
		EvidenceSummary: "八字主轴有承接，大运支持有限但不转坏。",
	}

	reply := manager.ComposeFinalReply("事业怎么走", result)
	if !strings.Contains(reply, "整体来看，这一步可以推进。") {
		t.Fatalf("reply = %q, want direct answer from structured result", reply)
	}
	if !strings.Contains(reply, "事业主线偏稳") {
		t.Fatalf("reply = %q, want key point from structured result", reply)
	}
	if !strings.Contains(reply, "依据：八字主轴有承接") {
		t.Fatalf("reply = %q, want evidence summary from structured result", reply)
	}
}

func TestManager_ComposesCrossDomainFollowupViaSynthesis(t *testing.T) {
	manager := &Manager{
		flash: &llm.NoopClient{
			GenerateFn: func(_ context.Context, systemPrompt string, messages []llm.Message) (string, llm.TokenUsage, error) {
				if !strings.Contains(systemPrompt, "多个领域 specialist 的结果综合成面向用户的最终回答") {
					t.Fatalf("systemPrompt = %q, want synthesis instruction", systemPrompt)
				}
				if len(messages) != 1 {
					t.Fatalf("len(messages) = %d, want 1", len(messages))
				}
				if !strings.Contains(messages[0].Content, "当前问题：八字和紫微一起看下事业和感情") {
					t.Fatalf("message content = %q, want user question context", messages[0].Content)
				}
				if !strings.Contains(messages[0].Content, "涉及领域：bazi+ziwei") {
					t.Fatalf("message content = %q, want aggregated domains", messages[0].Content)
				}
				if !strings.Contains(messages[0].Content, "专家结果：\n八字看事业稳中有升\n\n紫微看感情节奏偏慢") {
					t.Fatalf("message content = %q, want specialist summaries", messages[0].Content)
				}
				return "综合来看，事业以稳步推进为主，感情要慢慢磨合。", llm.TokenUsage{}, nil
			},
		},
	}
	result := specialists.Result{
		Domain:  "bazi+ziwei",
		Summary: "八字看事业稳中有升\n\n紫微看感情节奏偏慢",
	}

	reply := manager.ComposeFinalReply("八字和紫微一起看下事业和感情", result)
	if reply != "综合来看，事业以稳步推进为主，感情要慢慢磨合。" {
		t.Fatalf("reply = %q, want synthesized final answer", reply)
	}
}

func TestManager_BeginTurnSeedsManagerContextFromRoute(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	route := policy.ApprovedRoute{
		PrimaryDomain: "qimen",
		TaskIntent:    "fortune_followup",
		Slots: schemas.DecisionSlots{
			QuestionText: "career timing",
		},
	}

	manager.BeginTurn(st, route)

	if st.ManagerContext.ActiveDomain != "qimen" {
		t.Fatalf("ActiveDomain = %q, want qimen", st.ManagerContext.ActiveDomain)
	}
	if st.ManagerContext.CurrentTopic != "career timing" {
		t.Fatalf("CurrentTopic = %q, want career timing", st.ManagerContext.CurrentTopic)
	}
}

func TestManager_RecordInterruptStoresCheckpointInDomainContext(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
	}

	manager.RecordInterrupt(st, route, "cp-1", "interrupt-1", "solar_time_confirm")

	if st.ManagerContext.WaitingOn != "solar_time_confirm" {
		t.Fatalf("WaitingOn = %q, want solar_time_confirm", st.ManagerContext.WaitingOn)
	}
	if st.ManagerContext.LastReplyOwner != "manager" {
		t.Fatalf("LastReplyOwner = %q, want manager", st.ManagerContext.LastReplyOwner)
	}
	if st.DomainContexts.Bazi.InterruptID != "interrupt-1" {
		t.Fatalf("InterruptID = %q, want interrupt-1", st.DomainContexts.Bazi.InterruptID)
	}
	if st.DomainContexts.Bazi.CheckpointID != "cp-1" {
		t.Fatalf("CheckpointID = %q, want cp-1", st.DomainContexts.Bazi.CheckpointID)
	}
	if st.DomainContexts.Bazi.WorkingSummary != "solar_time_confirm" {
		t.Fatalf("WorkingSummary = %q, want solar_time_confirm", st.DomainContexts.Bazi.WorkingSummary)
	}
	if got := st.DomainContexts.Bazi.RuntimeValues["interrupt_reason"]; got != "solar_time_confirm" {
		t.Fatalf("interrupt_reason = %v, want solar_time_confirm", got)
	}
	if st.DomainContexts.Bazi.Version != 1 {
		t.Fatalf("Version = %d, want 1", st.DomainContexts.Bazi.Version)
	}
}

func TestManager_ResolveResumeInterruptIDFallsBackToStoredContext(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	st.Routing.PrimaryDomain = "bazi"
	st.DomainContexts.Bazi.CheckpointID = "cp-1"
	st.DomainContexts.Bazi.InterruptID = "interrupt-1"

	interruptID := manager.ResolveResumeInterruptID(st, "cp-1", "")
	if interruptID != "interrupt-1" {
		t.Fatalf("interruptID = %q, want interrupt-1", interruptID)
	}
}

func TestManager_FinishTurnClearsCheckpointAfterSuccessfulReply(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "fortune_followup",
		Slots: schemas.DecisionSlots{
			QuestionText: "wealth",
		},
	}

	manager.RecordInterrupt(st, route, "cp-1", "interrupt-1", "solar_time_confirm")
	manager.FinishTurn(st, route, "fortune_followup")

	if st.ManagerContext.WaitingOn != "" {
		t.Fatalf("WaitingOn = %q, want empty", st.ManagerContext.WaitingOn)
	}
	if st.ManagerContext.LastReplyOwner != "manager" {
		t.Fatalf("LastReplyOwner = %q, want manager", st.ManagerContext.LastReplyOwner)
	}
	if st.ManagerContext.CurrentTopic != "wealth" {
		t.Fatalf("CurrentTopic = %q, want wealth", st.ManagerContext.CurrentTopic)
	}
	if st.DomainContexts.Bazi.CheckpointID != "" {
		t.Fatalf("CheckpointID = %q, want cleared", st.DomainContexts.Bazi.CheckpointID)
	}
	if st.DomainContexts.Bazi.WorkingSummary != "" {
		t.Fatalf("WorkingSummary = %q, want cleared", st.DomainContexts.Bazi.WorkingSummary)
	}
	if st.DomainContexts.Bazi.RuntimeValues != nil {
		if _, ok := st.DomainContexts.Bazi.RuntimeValues["interrupt_reason"]; ok {
			t.Fatalf("interrupt_reason should be cleared, got %v", st.DomainContexts.Bazi.RuntimeValues["interrupt_reason"])
		}
	}
	if st.DomainContexts.Bazi.Version != 2 {
		t.Fatalf("Version = %d, want 2", st.DomainContexts.Bazi.Version)
	}
}

func TestManager_FinishTurnKeepsClarificationWaitingState(t *testing.T) {
	manager := &Manager{}
	st := state.NewSession("s1")
	route := policy.ApprovedRoute{
		PrimaryDomain: "bazi",
		TaskIntent:    "collect_profile",
	}

	manager.FinishTurn(st, route, "clarification")

	if st.ManagerContext.WaitingOn != "user_reply" {
		t.Fatalf("WaitingOn = %q, want user_reply", st.ManagerContext.WaitingOn)
	}
}
