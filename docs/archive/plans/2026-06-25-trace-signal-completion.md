# Trace Signal Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补全 eino callback 的 LLM 输入 messages 安全预览 + 验证 span 命名修复 + 给 graph Lambda 节点补手动 span，让 debug 时能直接从 trace 看到关键链路信号，同时避免默认外发完整 prompt / 出生资料。

**Architecture:** 三层改动：(1) 在 eino_callback.go 的 ChatModel OnStart 里默认只记录 message 数量 / role 序列，只有 `TRACE_LLM_INPUT_MESSAGES=1` 时才写入脱敏截断后的 `input.messages.preview`；(2) 加测试验证 Q5 span 命名修复（RunInfo.Name 覆盖 cfg.Name）；(3) 给 orchestration_graph.go 的 guard/short_circuit/agent Lambda 节点补 `tracing.SpanFromContext` 手动 span，prefill 只补 Graph 节点属性或改名，避免和 `Executor.prefill` 现有 span 重复。不改前端，只补底层信号。

**Tech Stack:** Go, eino ADK, eino callbacks (NewHandlerHelper), internal/tracing 包（NewRealTracer + SpanFromContext）

---

## File Structure

| 文件 | 责任 | 改动类型 |
|------|------|---------|
| `internal/tracing/eino_callback.go` | eino callback handler，ChatModel OnStart 记安全 messages 信号 | Modify |
| `internal/tracing/eino_callback_test.go` | callback handler 测试 | Modify（加 2 个测试） |
| `internal/runtime/orchestration_graph.go` | graph Lambda 节点，补手动 span | Modify |

---

## Task 1: ChatModel OnStart 记录 LLM 输入 messages 安全信号

**Files:**
- Modify: `internal/tracing/eino_callback.go`（ChatModel OnStart，约 line 50-73；新增 `summarizeLLMMessages` / `serializeLLMMessagePreview` / `shouldRecordLLMInputPreview` 辅助函数）
- Test: `internal/tracing/eino_callback_test.go`（新增 `TestEinoChatModelCallback_RecordsInputMessageSummary`、`TestEinoChatModelCallback_RecordsInputMessagePreviewWhenEnabled`）

**安全边界：**
- 默认路径不得写入完整 `input.Messages` 内容，因为 `realSpan.SetAttribute` 会同步到 OTel mirror，`OTEL_*` 开启时会外发到 Langfuse / OTLP backend。
- 默认只记录 `input.message_count`、`input.message_roles`。
- 只有 `TRACE_LLM_INPUT_MESSAGES=1` 时才记录 `input.messages.preview`。
- `input.messages.preview` 必须做脱敏和截断：单条 message 最多 200 rune，总长度最多 3000 rune；疑似 API key、Authorization header、身份证/手机号等敏感片段替换为 `[redacted]`。

- [ ] **Step 1: 写失败测试**

在 `internal/tracing/eino_callback_test.go` 末尾加：

```go
func TestEinoChatModelCallback_RecordsInputMessageSummary(t *testing.T) {
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(NewEinoTraceCallbackHandler())

	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	ctx = WithEinoCallbackSpan(ctx, EinoCallbackSpanConfig{
		Name: "bazi_specialist",
		Kind: KindLLM,
	})
	ctx = einocallbacks.ReuseHandlers(ctx, &einocallbacks.RunInfo{
		Name:      "bazi_specialist",
		Type:      "ChatModel",
		Component: components.ComponentOfChatModel,
	})
	ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{
		Messages: []*schema.Message{
			schema.SystemMessage("你是八字专家"),
			schema.UserMessage("1992年12月1日 男 北京"),
		},
	})
	einocallbacks.OnEnd(ctx, &einomodel.CallbackOutput{})

	tr := TraceFromContext(ctx)
	if tr == nil {
		t.Fatal("TraceFromContext returned nil")
	}
	var found bool
	for _, span := range tr.Spans {
		if span.Name != "bazi_specialist" || span.Kind != KindLLM {
			continue
		}
		found = true
		if got := span.Attributes["input.message_count"]; got != 2 {
			t.Fatalf("input.message_count = %v, want 2", got)
		}
		if got := span.Attributes["input.message_roles"]; got != "system,user" {
			t.Fatalf("input.message_roles = %v, want system,user", got)
		}
		if _, ok := span.Attributes["input.messages.preview"]; ok {
			t.Fatal("input.messages.preview should not be recorded by default")
		}
	}
	if !found {
		t.Fatal("expected bazi_specialist LLM span with input message summary")
	}
}
```

再加开关路径测试：

```go
func TestEinoChatModelCallback_RecordsInputMessagePreviewWhenEnabled(t *testing.T) {
	t.Setenv("TRACE_LLM_INPUT_MESSAGES", "1")
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(NewEinoTraceCallbackHandler())

	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	ctx = WithEinoCallbackSpan(ctx, EinoCallbackSpanConfig{
		Name: "bazi_specialist",
		Kind: KindLLM,
	})
	ctx = einocallbacks.ReuseHandlers(ctx, &einocallbacks.RunInfo{
		Name:      "bazi_specialist",
		Type:      "ChatModel",
		Component: components.ComponentOfChatModel,
	})
	ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{
		Messages: []*schema.Message{
			schema.SystemMessage("你是八字专家，Authorization: Bearer sk-test"),
			schema.UserMessage("1992年12月1日12点 男 北京"),
		},
	})
	einocallbacks.OnEnd(ctx, &einomodel.CallbackOutput{})

	tr := TraceFromContext(ctx)
	for _, span := range tr.Spans {
		if span.Name != "bazi_specialist" || span.Kind != KindLLM {
			continue
		}
		got, ok := span.Attributes["input.messages.preview"]
		if !ok {
			t.Fatal("input.messages.preview attribute not set on span")
		}
		s, ok := got.(string)
		if !ok {
			t.Fatalf("input.messages.preview type = %T, want string", got)
		}
		if !strings.Contains(s, "1992") {
			t.Fatalf("input.messages.preview = %q, want contains 1992", s)
		}
		if strings.Contains(s, "sk-test") {
			t.Fatalf("input.messages.preview leaked secret: %q", s)
		}
		return
	}
	t.Fatal("expected bazi_specialist LLM span with input.messages.preview")
}
```

同时更新 import block（加 `strings`、`einomodel`、`schema`）：

```go
import (
	"context"
	"strings"
	"testing"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	einomodel "github.com/cloudwego/eino/components/model"
	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/tracing/ -run 'TestEinoChatModelCallback_RecordsInputMessage(Summary|PreviewWhenEnabled)' -v`
Expected: FAIL — `input.message_count` / `input.messages.preview` 未写入。

- [ ] **Step 3: 加 messages 摘要与安全预览辅助函数**

在 `internal/tracing/eino_callback.go` 的 `finishEinoRetrieverErrorSpan` 函数后（约 line 255 附近）加：

```go
// summarizeLLMMessages 记录 LLM 输入 messages 的结构信号，不包含原文内容。
func summarizeLLMMessages(span Span, msgs []*schema.Message) {
	if span == nil {
		return
	}
	roles := make([]string, 0, len(msgs))
	for _, m := range msgs {
		if m == nil {
			continue
		}
		roles = append(roles, string(m.Role))
	}
	span.SetAttribute("input.message_count", len(roles))
	span.SetAttribute("input.message_roles", strings.Join(roles, ","))
}

// shouldRecordLLMInputPreview 控制是否记录脱敏后的 message 内容预览。
// 默认关闭，避免 OTel mirror 外发完整 prompt、出生资料和历史上下文。
func shouldRecordLLMInputPreview() bool {
	return os.Getenv("TRACE_LLM_INPUT_MESSAGES") == "1"
}

// serializeLLMMessagePreview 把 LLM 输入 messages 序列化为脱敏截断预览，用于本地排障。
func serializeLLMMessagePreview(msgs []*schema.Message) string {
	var b strings.Builder
	for i, m := range msgs {
		if m == nil {
			continue
		}
		role := string(m.Role)
		content := redactTracePreview(m.Content)
		if r := []rune(content); len(r) > 200 {
			content = string(r[:200]) + "...(truncated)"
		}
		fmt.Fprintf(&b, "[%d] %s: %s", i, role, content)
		if i < len(msgs)-1 {
			b.WriteString("\n")
		}
		if r := []rune(b.String()); len(r) > 3000 {
			return string(r[:3000]) + "...(truncated)"
		}
	}
	return b.String()
}

func redactTracePreview(s string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)authorization:\s*bearer\s+[^\s]+`),
		regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*[^\s]+`),
		regexp.MustCompile(`sk-[A-Za-z0-9_-]+`),
		regexp.MustCompile(`\b1[3-9]\d{9}\b`),
		regexp.MustCompile(`\b\d{17}[\dXx]\b`),
	}
	out := s
	for _, p := range patterns {
		out = p.ReplaceAllString(out, "[redacted]")
	}
	return out
}
```

同时在 import block 加 `"fmt"`、`"os"`、`"regexp"` 和 `"strings"`（如果还没有）。

- [ ] **Step 4: 在 ChatModel OnStart 里记 messages 安全信号**

在 `internal/tracing/eino_callback.go` 的 ChatModel OnStart handler 里，`span.SetAttribute("gen_ai.operation.name", cfg.Name)` 这行后加：

```go
				if input != nil && len(input.Messages) > 0 {
					summarizeLLMMessages(span, input.Messages)
					if shouldRecordLLMInputPreview() {
						span.SetAttribute("input.messages.preview", serializeLLMMessagePreview(input.Messages))
					}
				}
```

- [ ] **Step 5: 跑测试确认通过**

Run: `go test ./internal/tracing/ -run 'TestEinoChatModelCallback_RecordsInputMessage(Summary|PreviewWhenEnabled)' -v`
Expected: PASS

- [ ] **Step 6: 跑全部 tracing 测试确认无回归**

Run: `go test ./internal/tracing/ -count=1 -v`
Expected: 所有测试 PASS（含原有的 `TestEinoRetrieverCallback_EmitsTraceSpan`）

- [ ] **Step 7: 提交**

```bash
git add internal/tracing/eino_callback.go internal/tracing/eino_callback_test.go
git commit -m "feat(tracing): record safe LLM input signals in ChatModel callback span

ChatModel OnStart now records message count and role sequence by default.
Raw message preview is gated behind TRACE_LLM_INPUT_MESSAGES=1 and is
redacted/truncated before being written to span attributes, avoiding
accidental OTel export of full prompts or user birth data."
```

---

## Task 2: 验证 span 命名修复（Q5）+ 加测试

**背景**：Q5 修复已在工作区（[eino_callback.go:55-58](internal/tracing/eino_callback.go:55) 用 `info.Name` 覆盖 `cfg.Name`），但没有测试锁定。加测试防止回归。

**Files:**
- Test: `internal/tracing/eino_callback_test.go`（新增 `TestEinoChatModelCallback_UsesRunInfoNameNotCtxCfgName`）

- [ ] **Step 1: 写测试**

在 `internal/tracing/eino_callback_test.go` 末尾加：

```go
func TestEinoChatModelCallback_UsesRunInfoNameNotCtxCfgName(t *testing.T) {
	// Q5: specialist 的 LLM 调用应使用 RunInfo.Name（如 bazi_specialist），
	// 而不是继承 supervisor ctx 里 cfg.Name（如 adk_supervisor_agent）。
	einocallbacks.InitCallbackHandlers(nil)
	t.Cleanup(func() { einocallbacks.InitCallbackHandlers(nil) })
	einocallbacks.AppendGlobalHandlers(NewEinoTraceCallbackHandler())

	rt := NewRealTracer(nil)
	ctx, trace := rt.StartTrace(context.Background(), "chat.turn")
	defer trace.End()

	// 模拟 supervisor ctx 里设置的 span config（name=supervisor）
	ctx = WithEinoCallbackSpan(ctx, EinoCallbackSpanConfig{
		Name: "adk_supervisor_agent",
		Kind: KindLLM,
	})
	// 但 RunInfo.Name 是 specialist 自己的名字
	ctx = einocallbacks.ReuseHandlers(ctx, &einocallbacks.RunInfo{
		Name:      "bazi_specialist",
		Type:      "ChatModel",
		Component: components.ComponentOfChatModel,
	})
	ctx = einocallbacks.OnStart(ctx, &einomodel.CallbackInput{
		Messages: []*schema.Message{schema.UserMessage("test")},
	})
	einocallbacks.OnEnd(ctx, &einomodel.CallbackOutput{})

	tr := TraceFromContext(ctx)
	for _, span := range tr.Spans {
		if span.Kind != KindLLM {
			continue
		}
		if span.Name != "bazi_specialist" {
			t.Fatalf("span.Name = %q, want %q (RunInfo.Name should override cfg.Name)", span.Name, "bazi_specialist")
		}
		return
	}
	t.Fatal("no LLM span found")
}
```

- [ ] **Step 2: 跑测试确认通过（Q5 已修）**

Run: `go test ./internal/tracing/ -run TestEinoChatModelCallback_UsesRunInfoNameNotCtxCfgName -v`
Expected: PASS（如果 FAIL，说明 Q5 修复被回退了，需要检查 [eino_callback.go:55-58](internal/tracing/eino_callback.go:55) 的 `if info != nil && info.Name != "" { name = info.Name }` 是否还在）

- [ ] **Step 3: 跑全部 tracing 测试**

Run: `go test ./internal/tracing/ -count=1`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add internal/tracing/eino_callback_test.go
git commit -m "test(tracing): verify specialist span name uses RunInfo.Name not ctx cfg.Name

Locks in the Q5 fix: when supervisor's ctx has cfg.Name=adk_supervisor_agent
but the actual LLM call is from bazi_specialist (RunInfo.Name), the span
should be named bazi_specialist. Prevents regression of the span-naming
bug that made trace show '一堆 chatmodel' with no agent attribution."
```

---

## Task 3: 给 graph Lambda 节点补手动 span

**背景**：eino callback 只捕获组件级（ChatModel/Tool/Retriever），Lambda 节点（preflight/agent/guard/short_circuit）不触发 callback。preflight 已有 span；`Executor.prefill` 也已有业务 span。补齐后 debug 能看到 Graph 节点进入/退出 + 关键属性，同时避免重复记录同名 prefill。

**Files:**
- Modify: `internal/runtime/orchestration_graph.go`（guardNode、emitShortCircuitNode、agentNode；prefillNode 只在需要时补 `orchestration.prefill` 外层 span）

- [ ] **Step 1: 处理 prefillNode，避免重复 span**

当前 [executor.go](internal/runtime/executor.go:262) 的 `Executor.prefill` 已记录 `prefill` span，并带有 `domain` / `executed` 属性。不要在 `prefillNode` 再加同名 `prefill` span。

推荐最小改动：保持 `prefillNode` 不变，继续让 `Executor.prefill` 作为业务 span。

如果确实需要 Graph 节点层级，可加外层 `orchestration.prefill`，名字必须与业务 span 区分：

```go
func prefillNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	span := tracing.SpanFromContext(ctx, "orchestration.prefill", tracing.KindChain)
	span.SetAttribute("primary_domain", oc.Init.Route.PrimaryDomain)
	defer span.End()

	oc.RT.Executor.prefill(ctx, oc.RT.Sink, oc.Init.St, oc.Init.Route, oc.Init.Vals)
	return in, nil
}
```

- [ ] **Step 2: 给 emitShortCircuitNode 加 span**

在 `emitShortCircuitNode` 函数里，`loadOrchestrationCtx` 后加 span：

```go
func emitShortCircuitNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	span := tracing.SpanFromContext(ctx, "short_circuit", tracing.KindChain)
	span.SetAttribute("turn_type", oc.GS.PreflightResult.TurnType)
	span.SetAttribute("short_circuit", oc.GS.PreflightResult.ShortCircuit)
	defer span.End()

	oc.RT.Executor.updateGuidanceState(oc.Init.St, oc.Init.Route, oc.Init.UserMsg, oc.GS.PreflightResult)
	_ = emitEventWithTrace(ctx, oc.RT.Sink, Event{
		Type: "text",
		Data: map[string]any{"content": oc.GS.PreflightResult.Text},
	}, map[string]any{"turn_type": oc.GS.PreflightResult.TurnType})
	getOrchestrationResult(ctx).TurnType = oc.GS.PreflightResult.TurnType
	return oc.GS.PreflightResult.Text, nil
}
```

- [ ] **Step 3: 给 guardNode 加 span**

在 `guardNode` 函数里，`loadOrchestrationCtx` 后加 span：

```go
func guardNode(ctx context.Context, finalText string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	span := tracing.SpanFromContext(ctx, "final_guard", tracing.KindChain)
	defer span.End()

	turnType, guardedText := guardFinalAnswerWithTrace(ctx, oc.GS.Route, oc.Init.St, finalText)
	span.SetAttribute("turn_type", turnType)
	if shouldBufferFinalAnswer() && guardedText != "" {
		_ = emitEventWithTrace(ctx, oc.RT.Sink, Event{
			Type: "text",
			Data: map[string]any{"content": guardedText},
		}, map[string]any{"buffer_final": true, "turn_type": turnType})
	}
	getOrchestrationResult(ctx).TurnType = turnType
	return guardedText, nil
}
```

- [ ] **Step 4: 给 agentNode 加节点级 span，并让生命周期覆盖流式执行**

在 `agentNode` 函数里，`loadOrchestrationCtx` 后、`BuildSupervisor` 前加节点级 span（区别于已有的 `WithEinoCallbackSpan`——那个是给 supervisor 内部 LLM 调用用的，这个是 agentNode 整体）。

注意：`agentNode` 返回 `StreamReader` 后，真正的 `agentEventBridge` 执行发生在 goroutine 内。`nodeSpan.End()` 不能用函数级 `defer`，否则 span 只覆盖启动阶段，不能覆盖流式执行和错误。

```go
func agentNode(ctx context.Context, in string) (*schema.StreamReader[string], error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return nil, err
	}
	nodeSpan := tracing.SpanFromContext(ctx, "agent", tracing.KindChain)
	nodeSpan.SetAttribute("primary_domain", oc.GS.Route.PrimaryDomain)

	// ForcedRoute 覆盖（preflight 返回 ForcedRoute 时）
	route := oc.GS.Route
	if oc.GS.PreflightResult.ForcedRoute != nil {
		route = *oc.GS.PreflightResult.ForcedRoute
		oc.GS.Route = route
		nodeSpan.SetAttribute("forced_route", true)
	}
	// ... 其余不变
```

然后在 goroutine 内结束 span，并记录桥接错误：

```go
	go func() {
		defer sw.Close()
		defer nodeSpan.End()
		finalText, err := agentEventBridge(ctx, oc.RT.Sink, iter, func(toolName, resultJSON string) {
			oc.RT.Executor.saveToolResult(oc.Init.St, toolName, resultJSON)
		}, oc.RT.Executor.reg.DisplayName, shouldBufferFinalAnswer())
		if err != nil {
			nodeSpan.RecordError(err)
			nodeSpan.SetStatus("error")
			sw.Send("", err)
			return
		}
		nodeSpan.SetAttribute("final_text_len", len([]rune(finalText)))
		sw.Send(finalText, nil)
	}()
```

- [ ] **Step 5: 编译确认**

Run: `go build ./internal/runtime/...`
Expected: 无错误

- [ ] **Step 6: 跑 runtime 测试确认无回归**

Run: `go test ./internal/runtime/ -count=1`
Expected: PASS（含 `TestBuildSpecialistHandlers_WithNilSummarizer` 等原有测试）

- [ ] **Step 7: 跑 topology smoke test**

Run: `go test ./internal/runtime/ -run TestOrchestration -v -count=1`
Expected: PASS（topology compile smoke test 验证 graph 结构没坏）

- [ ] **Step 8: 提交**

```bash
git add internal/runtime/orchestration_graph.go
git commit -m "feat(runtime): add manual trace spans to graph Lambda nodes

short_circuit/final_guard/agent nodes now emit KindChain spans with key
attributes. agent span covers the async agentEventBridge execution, and
prefill keeps using the existing Executor.prefill business span to avoid
duplicate same-name trace entries."
```

---

## Task 4: 端到端验证

**Files:** 无代码改动，只跑复现

- [ ] **Step 1: 重建后端**

Run: `make build`
Expected: `Built: <commit> -> /tmp/suanming-server`

- [ ] **Step 2: 重启后端**

Run:
```bash
lsof -ti :8080 | xargs -I{} sh -c 'ps -p {} -o command= | grep -q suanming-server && kill -9 {}' 2>/dev/null
sleep 1
set -a; source .env; set +a; DEBUG_TRACE=1 LISTEN_ADDR=:8080 /tmp/suanming-server > /tmp/suanming-backend.log 2>&1 &
sleep 4
curl -s http://localhost:8080/api/health
```
Expected: `{"status":"ok"}`

- [ ] **Step 3: 发一轮 1992 男北京，验证 trace 里有默认安全信号**

Run:
```bash
SID="trace-verify-$(date +%s)"
curl -sN --max-time 120 -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"1992年12月1日12点 男 北京\",\"session_id\":\"$SID\"}" > /tmp/trace-verify.txt
```

检查本地 trace JSON 里有没有默认安全信号：
```bash
LATEST_TRACE="$(find logs/traces -name '*.json' -type f -print0 | xargs -0 ls -t | head -1)"
jq '.spans[] | select(.attributes["input.message_count"] != null) | {name, kind, attrs: .attributes}' "$LATEST_TRACE" | head -40
```
Expected: 有 `input.message_count` / `input.message_roles`，默认没有 `input.messages.preview`。

- [ ] **Step 4: 打开安全预览开关后复测 input.messages.preview**

重启后端时加：
```bash
lsof -ti :8080 | xargs -I{} sh -c 'ps -p {} -o command= | grep -q suanming-server && kill -9 {}' 2>/dev/null
sleep 1
set -a; source .env; set +a
TRACE_LLM_INPUT_MESSAGES=1 DEBUG_TRACE=1 LISTEN_ADDR=:8080 /tmp/suanming-server > /tmp/suanming-backend.log 2>&1 &
sleep 4
```

重新发一轮请求后检查：
```bash
LATEST_TRACE="$(find logs/traces -name '*.json' -type f -print0 | xargs -0 ls -t | head -1)"
jq '.spans[] | select(.attributes["input.messages.preview"] != null) | .attributes["input.messages.preview"]' "$LATEST_TRACE" | head -3
```
Expected: 能看到脱敏截断预览；不能出现 `sk-`、`Authorization: Bearer` 明文。

- [ ] **Step 5: 检查 trace 里节点级 span**

从最新 trace JSON 或 SSE 的 `debug-trace` component 找 trace 结构，确认有 `agent` / `final_guard`，短路 case 里有 `short_circuit`。`prefill` 应继续来自 `Executor.prefill`，如新增 Graph 外层则名字应为 `orchestration.prefill`。

- [ ] **Step 6: 确认 bazi_specialist span 命名正确**

在 trace 里找 LLM span，确认名字是 `bazi_specialist` 不是 `adk_supervisor_agent`。

---

## Self-Review

**1. Spec coverage:**
- "eino_callback OnStart 记 input.Messages 安全信号" → Task 1 ✅
- "修 span 命名 Q5" → Task 2（验证已修 + 测试锁定） ✅
- "graph Lambda 节点补手动 span" → Task 3 ✅
- "端到端验证" → Task 4 ✅

**2. Placeholder scan:** 无 TBD/TODO/"implement later"。所有 step 有具体代码或命令。

**3. Type一致性:**
- `summarizeLLMMessages(span Span, msgs []*schema.Message)` — Task 1 定义，Task 1 使用 ✅
- `serializeLLMMessagePreview(msgs []*schema.Message) string` — Task 1 定义，Task 1 使用，且只在 `TRACE_LLM_INPUT_MESSAGES=1` 时调用 ✅
- `EinoCallbackSpanConfig{Name, Kind}` — Task 1/2 测试用，已在 [eino_callback.go:25](internal/tracing/eino_callback.go:25) 定义 ✅
- `KindLLM` / `KindChain` — 在 [turn_trace.go:14-18](internal/tracing/turn_trace.go:14) 定义 ✅
- `components.ComponentOfChatModel` — 在 [eino components/types.go:72](eino-agent/eino/components/types.go:72) 定义 ✅
- `tracing.SpanFromContext(ctx, name, kind)` — 在 [real_tracer.go:244](internal/tracing/real_tracer.go:244) 定义 ✅

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-25-trace-signal-completion.md`. Two execution options:

**1. Subagent-Driven (recommended)** - 每个 task 派 fresh subagent，task 间 review，迭代快

**2. Inline Execution** - 在当前 session 用 executing-plans 批量执行，checkpoint review

Which approach?
