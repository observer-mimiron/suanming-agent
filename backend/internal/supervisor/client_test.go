// This test file belongs to the route approval layer.
// It verifies external client behavior and protects the related contract from regressions.
// It approves routes; execution contracts are built later by Manager.
package supervisor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/observer-mimiron/suanming-agent/internal/intent"
	"github.com/observer-mimiron/suanming-agent/internal/llm"
	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/schemas"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tracing"
)

type stubRouteEngine struct {
	decideFn func(ctx context.Context, prompt, msg string) (schemas.SupervisorDecision, error)
}

func (s stubRouteEngine) Decide(ctx context.Context, prompt, msg string) (schemas.SupervisorDecision, error) {
	return s.decideFn(ctx, prompt, msg)
}

type fakeToolCallingModel struct {
	emitCallbacks bool
	tools         []*schema.ToolInfo
	generateFn    func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error)
}

type supervisorHTTPStatusError struct {
	HTTPStatusCode int
}

func (e supervisorHTTPStatusError) Error() string {
	return fmt.Sprintf("status code: %d", e.HTTPStatusCode)
}

func (m *fakeToolCallingModel) Generate(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error) {
	if m.emitCallbacks {
		ctx = einocallbacks.EnsureRunInfo(ctx, "fakeToolCallingModel", components.ComponentOfChatModel)
		ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{Messages: input})
		msg, err := m.generateFn(ctx, input, opts...)
		if err != nil {
			einocallbacks.OnError(ctx, err)
			return nil, err
		}
		einocallbacks.OnEnd(ctx, &einomodel.CallbackOutput{Message: msg})
		return msg, nil
	}
	if m.generateFn != nil {
		return m.generateFn(ctx, input, opts...)
	}
	return schema.AssistantMessage("unused", nil), nil
}

func (m *fakeToolCallingModel) Stream(_ context.Context, _ []*schema.Message, _ ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("stream not implemented in fakeToolCallingModel")
}

func (m *fakeToolCallingModel) WithTools(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
	m.tools = tools
	return m, nil
}

func TestClientDecide_UsesInjectedRouteEngine(t *testing.T) {
	origPromptLoader := loadSupervisorPrompt
	loadSupervisorPrompt = func() (string, error) { return "test prompt", nil }
	t.Cleanup(func() { loadSupervisorPrompt = origPromptLoader })

	want := schemas.SupervisorDecision{
		ConversationIntent: "consult",
		PrimaryDomain:      "bazi",
		TaskIntent:         "interpret_chart",
		Confidence:         0.9,
	}
	client := NewClient(&llm.NoopClient{}, WithRouteEngine(stubRouteEngine{
		decideFn: func(ctx context.Context, prompt, msg string) (schemas.SupervisorDecision, error) {
			if prompt == "test prompt" || prompt == "" {
				t.Fatalf("prompt should include injected session context, got %q", prompt)
			}
			if msg != "看看事业" {
				t.Fatalf("msg = %q, want 看看事业", msg)
			}
			return want, nil
		},
	}))

	got, err := client.Decide(context.Background(), "看看事业", state.NewSession("s1"))
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if got.PrimaryDomain != want.PrimaryDomain || got.TaskIntent != want.TaskIntent {
		t.Fatalf("Decide() = %+v, want %+v", got, want)
	}
}

func TestClientDecide_RouteEngineFallsBackToTextDecide(t *testing.T) {
	origPromptLoader := loadSupervisorPrompt
	loadSupervisorPrompt = func() (string, error) { return "test prompt", nil }
	t.Cleanup(func() { loadSupervisorPrompt = origPromptLoader })

	flash := &llm.NoopClient{
		GenerateFn: func(ctx context.Context, systemPrompt string, messages []llm.Message) (string, llm.TokenUsage, error) {
			return `{
				"conversation_intent":"consult",
				"primary_domain":"bazi",
				"secondary_domains":[],
				"task_intent":"interpret_chart",
				"needs_clarification":false,
				"clarification_question":"",
				"parallelizable":false,
				"confidence":0.88,
				"slots":{"profile":{},"question_text":"看看事业","time_scope":"","target_subject":"","language":"zh"},
				"policy_hints":{"needs_knowledge":true,"needs_qimen":false,"can_reuse_session_profile":false,"can_reuse_cached_result":false}
			}`, llm.TokenUsage{}, nil
		},
	}
	client := NewClient(flash, WithRouteEngine(stubRouteEngine{
		decideFn: func(ctx context.Context, prompt, msg string) (schemas.SupervisorDecision, error) {
			return schemas.SupervisorDecision{}, errors.New("engine failed")
		},
	}))

	got, err := client.Decide(context.Background(), "看看事业", state.NewSession("s1"))
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if got.TaskIntent != "interpret_chart" {
		t.Fatalf("TaskIntent = %q, want interpret_chart", got.TaskIntent)
	}
}

func TestTextDecideInjectsDraft07FallbackSchema(t *testing.T) {
	flash := &llm.NoopClient{GenerateFn: func(ctx context.Context, systemPrompt string, messages []llm.Message) (string, llm.TokenUsage, error) {
		if !strings.Contains(systemPrompt, "http://json-schema.org/draft-07/schema#") {
			t.Fatal("text fallback prompt omitted Draft-07 schema")
		}
		return `{"conversation_intent":"consult","primary_domain":"bazi","secondary_domains":[],"task_intent":"interpret_chart","needs_clarification":false,"clarification_question":"","parallelizable":false,"confidence":0.88,"slots":{"profile":{},"question_text":"测试","time_scope":"","target_subject":"","language":"zh"},"policy_hints":{"needs_knowledge":true,"needs_qimen":false,"can_reuse_session_profile":false,"can_reuse_cached_result":false}}`, llm.TokenUsage{}, nil
	}}
	client := NewClient(flash)
	if _, err := client.textDecide(context.Background(), "router rules", []llm.Message{{Role: "user", Content: "测试"}}, state.NewSession("test"), "测试"); err != nil {
		t.Fatal(err)
	}
}

func TestADKRouteEngine_DecideReturnsStructuredDecision(t *testing.T) {
	model := &fakeToolCallingModel{
		generateFn: func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error) {
			return schema.AssistantMessage("calling output", []schema.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: schema.FunctionCall{
						Name: "output",
						Arguments: `{
							"conversation_intent":"consult",
							"primary_domain":"bazi",
							"task_intent":"collect_profile",
							"confidence":0.93,
							"slots":{
								"profile":{"year":1990,"month":5,"day":20,"hour":8,"gender":"男","birthplace":"北京"},
								"question_text":""
							},
							"policy_hints":{"needs_knowledge":true}
						}`,
					},
				},
			}), nil
		},
	}

	engine, err := NewADKRouteEngine(context.Background(), model)
	if err != nil {
		t.Fatalf("NewADKRouteEngine() error = %v", err)
	}

	got, err := engine.Decide(context.Background(), "系统提示", "我1990年5月20日早上8点，男，北京")
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if got.TaskIntent != "collect_profile" {
		t.Fatalf("TaskIntent = %q, want collect_profile", got.TaskIntent)
	}
	if got.Slots.Profile["birthplace"] != "北京" {
		t.Fatalf("birthplace = %v, want 北京", got.Slots.Profile["birthplace"])
	}
}

func TestADKRouteEngine_DecideSelfCorrectsAfterValidationError(t *testing.T) {
	callCount := 0
	model := &fakeToolCallingModel{
		generateFn: func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error) {
			callCount++
			switch callCount {
			case 1:
				return schema.AssistantMessage("calling output", []schema.ToolCall{
					{
						ID:   "call-1",
						Type: "function",
						Function: schema.FunctionCall{
							Name: decisionToolName,
							Arguments: `{
									"conversation_intent":"consult",
									"primary_domain":"bazi",
									"task_intent":"collect_profile",
									"confidence":0.72,
									"slots":{
										"profile":{"year":1990,"month":1,"day":1,"hour":0,"gender":"男"},
										"question_text":""
									},
									"policy_hints":{"needs_knowledge":true}
								}`,
						},
					},
				}), nil
			case 2:
				last := input[len(input)-1]
				if last.Role != schema.User {
					t.Fatalf("last message role = %s, want user", last.Role)
				}
				if !strings.Contains(last.Content, "系统纠错反馈") {
					t.Fatalf("retry message = %q, want correction marker", last.Content)
				}
				if !strings.Contains(last.Content, "slots.profile") {
					t.Fatalf("retry message = %q, want slots.profile guidance", last.Content)
				}
				return schema.AssistantMessage("calling output again", []schema.ToolCall{
					{
						ID:   "call-2",
						Type: "function",
						Function: schema.FunctionCall{
							Name: decisionToolName,
							Arguments: `{
								"conversation_intent":"consult",
								"primary_domain":"bazi",
								"task_intent":"collect_profile",
								"confidence":0.94,
								"slots":{
									"profile":{"year":1990,"month":5,"day":20,"hour":8,"gender":"男","birthplace":"北京"},
									"question_text":""
								},
								"policy_hints":{"needs_knowledge":true}
							}`,
						},
					},
				}), nil
			default:
				t.Fatalf("unexpected Generate call #%d", callCount)
				return nil, nil
			}
		},
	}

	engine, err := NewADKRouteEngine(context.Background(), model)
	if err != nil {
		t.Fatalf("NewADKRouteEngine() error = %v", err)
	}

	got, err := engine.Decide(context.Background(), "系统提示", "我1990年5月20日早上8点，男，北京")
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if callCount != 2 {
		t.Fatalf("Generate call count = %d, want 2", callCount)
	}
	if got.TaskIntent != "collect_profile" {
		t.Fatalf("TaskIntent = %q, want collect_profile", got.TaskIntent)
	}
	if got.Slots.Profile["birthplace"] != "北京" {
		t.Fatalf("birthplace = %v, want 北京", got.Slots.Profile["birthplace"])
	}
}

func TestADKRouteEngine_DecideEmitsSupervisorModelSpan(t *testing.T) {
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(tracing.NewEinoTraceCallbackHandler())

	model := &fakeToolCallingModel{
		emitCallbacks: true,
		generateFn: func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error) {
			return schema.AssistantMessage("calling output", []schema.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: schema.FunctionCall{
						Name: decisionToolName,
						Arguments: `{
							"conversation_intent":"consult",
							"primary_domain":"bazi",
							"task_intent":"collect_profile",
							"confidence":0.93,
							"slots":{
								"profile":{"year":1990,"month":5,"day":20,"hour":8,"gender":"男","birthplace":"北京"},
								"question_text":""
							},
							"policy_hints":{"needs_knowledge":true}
						}`,
					},
				},
			}), nil
		},
	}

	engine, err := NewADKRouteEngine(context.Background(), model)
	if err != nil {
		t.Fatalf("NewADKRouteEngine() error = %v", err)
	}

	rt := tracing.NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	_, err = engine.Decide(ctx, "系统提示", "我1990年5月20日早上8点，男，北京")
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}

	tr := tracing.TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	var count int
	for _, span := range tr.Spans {
		if span.Name == "supervisor_model" && span.Kind == tracing.KindLLM {
			count++
		}
	}
	if count < 1 {
		t.Fatalf("supervisor_model span count = %d, want >= 1", count)
	}
}

func TestADKRouteEngine_DecideStopsAfterSecondValidationError(t *testing.T) {
	callCount := 0
	model := &fakeToolCallingModel{
		generateFn: func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error) {
			callCount++
			if callCount == 2 {
				last := input[len(input)-1]
				if last.Role != schema.User {
					t.Fatalf("last message role = %s, want user", last.Role)
				}
				if !strings.Contains(last.Content, "系统纠错反馈") {
					t.Fatalf("retry message = %q, want correction marker", last.Content)
				}
			}
			return schema.AssistantMessage("calling output", []schema.ToolCall{
				{
					ID:   "call-retry",
					Type: "function",
					Function: schema.FunctionCall{
						Name: decisionToolName,
						Arguments: `{
								"conversation_intent":"consult",
								"primary_domain":"bazi",
								"task_intent":"collect_profile",
								"confidence":0.72,
								"slots":{
									"profile":{"year":1990,"month":1,"day":1,"hour":0,"gender":"男"},
									"question_text":""
								},
								"policy_hints":{"needs_knowledge":true}
							}`,
					},
				},
			}), nil
		},
	}

	engine, err := NewADKRouteEngine(context.Background(), model)
	if err != nil {
		t.Fatalf("NewADKRouteEngine() error = %v", err)
	}

	_, err = engine.Decide(context.Background(), "系统提示", "我1990年5月20日早上8点，男，北京")
	if err == nil {
		t.Fatal("Decide() error = nil, want validation failure after retry exhaustion")
	}
	if callCount != 2 {
		t.Fatalf("Generate call count = %d, want 2", callCount)
	}
	if !strings.Contains(err.Error(), "slots.profile") {
		t.Fatalf("Decide() error = %q, want slots.profile guidance", err.Error())
	}
}

func TestADKRouteEngine_DecideDoesNotRetryNonTransientError(t *testing.T) {
	callCount := 0
	var seenCorrectionFeedback bool
	model := &fakeToolCallingModel{
		generateFn: func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error) {
			callCount++
			last := input[len(input)-1]
			if strings.Contains(last.Content, "系统纠错反馈") {
				seenCorrectionFeedback = true
			}
			return nil, errors.New("upstream model unavailable")
		},
	}

	engine, err := NewADKRouteEngine(context.Background(), model)
	if err != nil {
		t.Fatalf("NewADKRouteEngine() error = %v", err)
	}

	_, err = engine.Decide(context.Background(), "系统提示", "看看事业")
	if err == nil {
		t.Fatal("Decide() error = nil, want upstream model error")
	}
	if callCount != 1 {
		t.Fatalf("Generate call count = %d, want 1 without transient retry", callCount)
	}
	if seenCorrectionFeedback {
		t.Fatal("non-validation error should not trigger outer correction retry message")
	}
	if !strings.Contains(err.Error(), "upstream model unavailable") {
		t.Fatalf("Decide() error = %q, want upstream model unavailable", err.Error())
	}
}

func TestADKRouteEngine_DecideRetriesTransientModelError(t *testing.T) {
	callCount := 0
	model := &fakeToolCallingModel{
		generateFn: func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error) {
			callCount++
			if callCount < 3 {
				return nil, supervisorHTTPStatusError{HTTPStatusCode: http.StatusTooManyRequests}
			}
			return schema.AssistantMessage("calling output", []schema.ToolCall{
				{
					ID:   "call-ok",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      decisionToolName,
						Arguments: "{\"conversation_intent\":\"consult\",\"primary_domain\":\"bazi\",\"task_intent\":\"collect_profile\",\"confidence\":0.82,\"slots\":{\"profile\":{\"year\":1991,\"month\":10,\"day\":5,\"hour\":12,\"minute\":40,\"gender\":\"男\",\"birthplace\":\"南京\"},\"question_text\":\"\"},\"policy_hints\":{\"needs_knowledge\":true}}",
					},
				},
			}), nil
		},
	}

	engine, err := NewADKRouteEngine(context.Background(), model)
	if err != nil {
		t.Fatalf("NewADKRouteEngine() error = %v", err)
	}

	_, err = engine.Decide(context.Background(), "系统提示", "1991年10月5日12点40分 南京 男")
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if callCount != 3 {
		t.Fatalf("Generate call count = %d, want 3 with transient retries", callCount)
	}
}

func TestADKRouteEngine_DecideDoesNotRetryFatalModelStatus(t *testing.T) {
	callCount := 0
	model := &fakeToolCallingModel{
		generateFn: func(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error) {
			callCount++
			return nil, supervisorHTTPStatusError{HTTPStatusCode: http.StatusPaymentRequired}
		},
	}

	engine, err := NewADKRouteEngine(context.Background(), model)
	if err != nil {
		t.Fatalf("NewADKRouteEngine() error = %v", err)
	}

	_, err = engine.Decide(context.Background(), "系统提示", "看看事业")
	if err == nil {
		t.Fatal("Decide() error = nil, want fatal model error")
	}
	if callCount != 1 {
		t.Fatalf("Generate call count = %d, want 1 without fatal retry", callCount)
	}
}

func TestParseDecision_ValidJSON(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "bazi",
		"secondary_domains": [],
		"task_intent": "interpret_chart",
		"needs_clarification": false,
		"clarification_question": "",
		"parallelizable": false,
		"confidence": 0.95,
		"slots": {
			"profile": {"name": "张三", "gender": "男"},
			"question_text": "我的财运如何",
			"time_scope": "今年",
			"target_subject": "",
			"language": "zh"
		},
		"policy_hints": {
			"needs_knowledge": true,
			"needs_qimen": false,
			"can_reuse_session_profile": false,
			"can_reuse_cached_result": false
		}
	}`

	got, err := parseDecision(raw)
	if err != nil {
		t.Fatalf("parseDecision() error = %v", err)
	}
	if got.ConversationIntent != "consult" {
		t.Fatalf("ConversationIntent: got %q, want %q", got.ConversationIntent, "consult")
	}
	if got.PrimaryDomain != "bazi" {
		t.Fatalf("PrimaryDomain: got %q, want %q", got.PrimaryDomain, "bazi")
	}
	if got.TaskIntent != "interpret_chart" {
		t.Fatalf("TaskIntent: got %q, want %q", got.TaskIntent, "interpret_chart")
	}
	if got.Confidence != 0.95 {
		t.Fatalf("Confidence: got %f, want %f", got.Confidence, 0.95)
	}
	if got.Slots.QuestionText != "我的财运如何" {
		t.Fatalf("Slots.QuestionText: got %q, want %q", got.Slots.QuestionText, "我的财运如何")
	}
	if got.Slots.TimeScope != "今年" {
		t.Fatalf("Slots.TimeScope: got %q, want %q", got.Slots.TimeScope, "今年")
	}
	if !got.PolicyHints.NeedsKnowledge {
		t.Fatal("PolicyHints.NeedsKnowledge: got false, want true")
	}
	if got.PolicyHints.NeedsQimen {
		t.Fatal("PolicyHints.NeedsQimen: got true, want false")
	}
}

func TestParseDecision_RejectsMissingRequiredFields(t *testing.T) {
	raw := `{"primary_domain":"bazi"}`
	if _, err := parseDecision(raw); err == nil {
		t.Fatal("parseDecision() error = nil, want missing-field rejection")
	}
}

func TestParseDecision_MalformedJSON(t *testing.T) {
	raw := `not valid json at all {{{`
	if _, err := parseDecision(raw); err == nil {
		t.Fatal("parseDecision() error = nil, want malformed JSON rejection")
	}
}

func TestParseDecision_EmptyOutput(t *testing.T) {
	raw := ""
	if _, err := parseDecision(raw); err == nil {
		t.Fatal("parseDecision() error = nil, want empty-output rejection")
	}
}

func TestParseDecision_InvalidConfidence(t *testing.T) {
	raw := `{"conversation_intent":"consult","primary_domain":"bazi","confidence":-0.5}`
	if _, err := parseDecision(raw); err == nil {
		t.Fatal("parseDecision() error = nil, want missing-field rejection")
	}
}

func TestParseAndValidate_CollectProfileWithoutProfileData(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "bazi",
		"secondary_domains": [],
		"task_intent": "collect_profile",
		"needs_clarification": false,
		"clarification_question": "",
		"parallelizable": false,
		"confidence": 0.9,
		"slots": {
			"profile": {},
			"question_text": "",
			"time_scope": "",
			"target_subject": "",
			"language": "zh"
		},
		"policy_hints": {"needs_knowledge": false, "needs_qimen": false, "can_reuse_session_profile": false, "can_reuse_cached_result": false}
	}`

	_, err := parseAndValidate(raw)
	if err != nil {
		t.Fatalf("collect_profile with empty profile is valid: %v", err)
	}
}

func TestParseAndValidate_CollectProfileWithProfileData(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "bazi",
		"secondary_domains": [],
		"task_intent": "collect_profile",
		"needs_clarification": false,
		"clarification_question": "",
		"parallelizable": false,
		"confidence": 0.9,
		"slots": {
			"profile": {"year": 1990, "month": 5, "day": 20, "hour": 8, "gender": "男", "birthplace": "北京"},
			"question_text": "",
			"time_scope": "",
			"target_subject": "",
			"language": "zh"
		},
		"policy_hints": {"needs_knowledge": false, "needs_qimen": false, "can_reuse_session_profile": false, "can_reuse_cached_result": false}
	}`

	d, err := parseAndValidate(raw)
	if err != nil {
		t.Fatalf("collect_profile with profile data should pass: %v", err)
	}
	if d.TaskIntent != "collect_profile" {
		t.Fatalf("TaskIntent: got %q, want collect_profile", d.TaskIntent)
	}
}

func TestParseAndValidate_TimingFollowupWithoutQuestionText(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "qimen",
		"task_intent": "timing_followup",
		"confidence": 0.9,
		"slots": {
			"profile": {},
			"question_text": ""
		}
	}`

	_, err := parseAndValidate(raw)
	if err == nil {
		t.Fatal("timing_followup with empty question_text must return error")
	}
}

func TestParseAndValidate_ValidNonCollectProfilePasses(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "bazi",
		"secondary_domains": [],
		"task_intent": "interpret_chart",
		"needs_clarification": false,
		"clarification_question": "",
		"parallelizable": false,
		"confidence": 0.95,
		"slots": {
			"profile": {},
			"question_text": "我的财运如何",
			"time_scope": "",
			"target_subject": "",
			"language": "zh"
		},
		"policy_hints": {"needs_knowledge": false, "needs_qimen": false, "can_reuse_session_profile": false, "can_reuse_cached_result": false}
	}`

	d, err := parseAndValidate(raw)
	if err != nil {
		t.Fatalf("interpret_chart with empty profile should pass (profile not needed): %v", err)
	}
	if d.TaskIntent != "interpret_chart" {
		t.Fatalf("TaskIntent: got %q, want interpret_chart", d.TaskIntent)
	}
}

func TestDecisionRetryPrompt_IncludesValidationGuidance(t *testing.T) {
	got := decisionRetryPrompt(errors.New("bad json"))
	want := "返回的 JSON 有误: bad json。请重新返回完整的 JSON，特别注意 slots.profile 必须从用户原始消息中提取实际值，不要用示例值或空对象。"
	if got != want {
		t.Fatalf("decisionRetryPrompt() = %q, want %q", got, want)
	}
}

func TestParseDecision_QimenSecondaryDomain(t *testing.T) {
	raw := `{
		"conversation_intent": "consult",
		"primary_domain": "bazi",
		"secondary_domains": ["qimen"],
		"task_intent": "cross_domain_consult",
		"confidence": 0.85
	}`

	if _, err := parseDecision(raw); err == nil {
		t.Fatal("parseDecision() error = nil, want incomplete fallback schema rejection")
	}
}

// routerStub implements intent.Router for testing option injection.
type routerStub struct{}

func (r routerStub) Match(_ context.Context, _ string) (intent.MatchResult, error) {
	return intent.MatchResult{Decision: intent.DecisionNone}, nil
}

func TestNewClient_WithSemanticRouter(t *testing.T) {
	flash := &llm.NoopClient{}
	router := routerStub{}
	c := NewClient(flash, WithSemanticRouter(router))
	if c.router != router {
		t.Fatal("WithSemanticRouter did not set router field")
	}
}

func TestNewClient_WithoutSemanticRouter(t *testing.T) {
	flash := &llm.NoopClient{}
	c := NewClient(flash)
	if c.router != nil {
		t.Fatal("router should be nil by default")
	}
}

func TestNewClient_WithRouterMode(t *testing.T) {
	flash := &llm.NoopClient{}
	c := NewClient(flash, WithRouterMode("shadow"))
	if c.routerMode != "shadow" {
		t.Fatalf("routerMode = %q, want shadow", c.routerMode)
	}
}

func TestNormalizeApprovedRoute_StopsOwningSessionAwareTaskRewrite(t *testing.T) {
	client := &Client{}

	cases := []struct {
		name  string
		msg   string
		state *state.SessionState
		route policy.ApprovedRoute
		want  string
	}{
		{
			name: "existing profile no longer rewrites collect_profile",
			msg:  "我是1991年生的",
			state: func() *state.SessionState {
				st := state.NewSession("s1")
				st.Profile = map[string]any{"year": 1991.0}
				return st
			}(),
			route: policy.ApprovedRoute{PrimaryDomain: "bazi", TaskIntent: "collect_profile"},
			want:  "collect_profile",
		},
		{
			name: "cached chart no longer rewrites collect_profile",
			msg:  "看看这两年财运",
			state: func() *state.SessionState {
				st := state.NewSession("s2")
				st.BaziResult = map[string]any{"dayGan": "甲"}
				return st
			}(),
			route: policy.ApprovedRoute{PrimaryDomain: "bazi", TaskIntent: "collect_profile"},
			want:  "collect_profile",
		},
		{
			name: "cached chart no longer rewrites interpret_chart",
			msg:  "看看这两年财运",
			state: func() *state.SessionState {
				st := state.NewSession("s3")
				st.BaziResult = map[string]any{"dayGan": "甲"}
				return st
			}(),
			route: policy.ApprovedRoute{PrimaryDomain: "bazi", TaskIntent: "interpret_chart"},
			want:  "interpret_chart",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := client.normalizeApprovedRoute(context.Background(), tc.msg, tc.state, tc.route)
			if got.TaskIntent != tc.want {
				t.Fatalf("TaskIntent = %q, want %q", got.TaskIntent, tc.want)
			}
		})
	}
}

func TestNormalizeApprovedRoute_CompletesExplicitBirthProfileAfterDegradedAmendRoute(t *testing.T) {
	client := &Client{}
	st := state.NewSession("s-birth")
	route := policy.ApprovedRoute{
		PrimaryDomain:      "bazi",
		TaskIntent:         "amend_profile",
		NeedsClarification: true,
		Slots: schemas.DecisionSlots{Profile: map[string]any{
			"hour":   23.0,
			"minute": 53.0,
		}},
	}

	got := client.normalizeApprovedRoute(context.Background(), "2025年11月10日23点53分 男 上海", st, route)
	if got.TaskIntent != "collect_profile" {
		t.Fatalf("TaskIntent = %q, want collect_profile", got.TaskIntent)
	}
	if got.NeedsClarification {
		t.Fatal("complete explicit birth profile should not request clarification")
	}
	for field, want := range map[string]any{
		"year": 2025.0, "month": 11.0, "day": 10.0, "hour": 23.0, "minute": 53.0, "gender": "男", "birthplace": "上海",
	} {
		if got.Slots.Profile[field] != want {
			t.Fatalf("profile[%q] = %v, want %v", field, got.Slots.Profile[field], want)
		}
	}
}
