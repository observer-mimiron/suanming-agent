# Embedding Intent Router Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用 Eino Embedder + dashscope `text-embedding-v4` 的语义路由替换 `MentionsXxxMethod` regex 关键词识别，解决假阳性和 LLM 被错误覆盖两个痛点。

**Architecture:** 启动期一次性 embed ~60 条 utterance（正向+负向）存内存；每轮用户消息 embedding 后算余弦相似度。`Match` 返回 `(MatchResult, error)`，`Decision = positive | negative | none`——error 才退回 regex 兜底，negative/none 不退回（避免被 regex 击穿）。三态开关 `ROUTER_MODE=off|shadow|enforce` 控制上线节奏。

**Tech Stack:** Go + Eino框架 + eino-ext openai embedding ext（指向 DashScope OpenAI-compatible 端点）+ `text-embedding-v4`。

**Spec:** [docs/superpowers/specs/2026-06-26-embedding-intent-router-design.md](../specs/2026-06-26-embedding-intent-router-design.md)

---

## File Structure

**新增文件：**
- `internal/llm/embedding_factory.go` — `NewEmbedder(cfg)` 工厂，用 eino-ext openai ext 指向 DashScope
- `internal/intent/utterances.go` — 每方法 5-10 条正向 + 5-10 条负向 utterance
- `internal/intent/semantic_router.go` — `SemanticRouter` 类型 + `Match` + cosine helper
- `internal/intent/semantic_router_test.go` — 单元测试（mock Embedder）
- `internal/intent/semantic_router_regression_test.go` — 回归测试（真 embedder，无 key 则 skip）
- `internal/runtime/guidance_gate_test.go` — anyHardNegative 集成测试

**修改文件：**
- `internal/config/config.go` — 加 `EmbeddingApiKey/EmbeddingBaseUrl/EmbeddingModel/RouterMode`
- `internal/supervisor/client.go` — `Client` 加 `router` 字段 + `WithSemanticRouter` option
- `internal/supervisor/approved_route.go` — `applyExplicitMethodPreference` 改 Client 方法，用 router
- `internal/runtime/executor.go` — `Executor` 加 `router` 字段，传给 orchestrationRuntime
- `internal/runtime/orchestration_state.go` — `orchestrationRuntime` 加 `Router` 字段
- `internal/runtime/orchestration_graph.go` — `preflightNode` 从 `oc.RT.Router` 取 router 传给 preflight
- `internal/runtime/preflight.go` — `preflight` 签名加 `router`
- `internal/runtime/guidance_gate.go` — `ShouldEnterGuidance` + `anyHardNegative` 加 `router`
- `internal/container/container.go` — 构造 Embedder + SemanticRouter，注入 Client 和 Executor

---

## Task 1: Config — 加 embedding + ROUTER_MODE 字段

**Files:**
- Modify: `internal/config/config.go:14-56`
- Test: `internal/config/config_test.go`（新建或扩充）

- [ ] **Step 1: 写失败测试**

创建 `internal/config/config_test.go`：

```go
package config

import (
	"os"
	"testing"
)

func TestLoad_EmbeddingAndRouterMode(t *testing.T) {
	t.Setenv("EMBEDDING_API_KEY", "sk-test-embed-key")
	t.Setenv("EMBEDDING_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1")
	t.Setenv("EMBEDDING_MODEL", "text-embedding-v4")
	t.Setenv("ROUTER_MODE", "shadow")

	cfg := Load()

	if cfg.EmbeddingApiKey != "sk-test-embed-key" {
		t.Fatalf("EmbeddingApiKey = %q, want %q", cfg.EmbeddingApiKey, "sk-test-embed-key")
	}
	if cfg.EmbeddingBaseUrl != "https://dashscope.aliyuncs.com/compatible-mode/v1" {
		t.Fatalf("EmbeddingBaseUrl = %q", cfg.EmbeddingBaseUrl)
	}
	if cfg.EmbeddingModel != "text-embedding-v4" {
		t.Fatalf("EmbeddingModel = %q", cfg.EmbeddingModel)
	}
	if cfg.RouterMode != "shadow" {
		t.Fatalf("RouterMode = %q, want shadow", cfg.RouterMode)
	}
}

func TestLoad_RouterModeDefault(t *testing.T) {
	os.Unsetenv("ROUTER_MODE")
	cfg := Load()
	if cfg.RouterMode != "off" {
		t.Fatalf("default RouterMode = %q, want off", cfg.RouterMode)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/config/ -run TestLoad_EmbeddingAndRouterMode -v`
Expected: FAIL with `cfg.EmbeddingApiKey undefined`（字段不存在）

- [ ] **Step 3: 加字段到 Config struct**

修改 `internal/config/config.go`，在 `Config` struct 末尾加 4 个字段：

```go
type Config struct {
	LLMApiKey         string
	LLMBaseURL        string
	LLMModel          string
	LLMFlashModel     string
	LLMTemperature    float64
	KnowledgeURL      string
	StaticDir         string
	ListenAddr        string
	DebugHTTP         bool
	DebugTrace        bool
	OTelEnabled       bool
	OTelEndpoint      string
	OTelHeaders       string
	OTelServiceName   string
	OTelInsecure      bool
	ConversationLimit int

	// Embedding 配置——用于 semantic router（意图识别）。沿用 KB 同款 env 变量名。
	EmbeddingApiKey  string
	EmbeddingBaseUrl string
	EmbeddingModel   string
	// RouterMode 控制 semantic router 上线节奏：off | shadow | enforce。
	// off=不初始化 router，走旧 regex；shadow=旁路只 log；enforce=接入决策。
	RouterMode string
}
```

- [ ] **Step 4: 在 Load() 里读 env**

修改 `Load()` 函数，在 return 的 struct 字面量末尾加：

```go
		EmbeddingApiKey:  os.Getenv("EMBEDDING_API_KEY"),
		EmbeddingBaseUrl: getEnv("EMBEDDING_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
		EmbeddingModel:   getEnv("EMBEDDING_MODEL", "text-embedding-v4"),
		RouterMode:       getEnv("ROUTER_MODE", "off"),
```

- [ ] **Step 5: 跑测试确认通过**

Run: `go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add embedding + router mode fields for semantic router"
```

---

## Task 2: Utterances 数据

**Files:**
- Create: `internal/intent/utterances.go`
- Test: `internal/intent/utterances_test.go`

- [ ] **Step 1: 写失败测试**

创建 `internal/intent/utterances_test.go`：

```go
package intent

import "testing"

func TestUtterances_EachMethodHasMinimum(t *testing.T) {
	for _, method := range []string{"ziwei", "qimen", "bazi"} {
		r, ok := Utterances[method]
		if !ok {
			t.Fatalf("method %q missing from Utterances", method)
		}
		if len(r.Positive) < 5 {
			t.Fatalf("method %q: only %d positive utterances, want >= 5", method, len(r.Positive))
		}
		if len(r.Negative) < 5 {
			t.Fatalf("method %q: only %d negative utterances, want >= 5", method, len(r.Negative))
		}
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/intent/ -run TestUtterances -v`
Expected: FAIL with `Utterances undefined`

- [ ] **Step 3: 写 utterances 数据**

创建 `internal/intent/utterances.go`：

```go
package intent

// RouteUtterances 是单个术数方法的正向和负向 utterance 集合。
// 正向：用户显式请求该方法。负向：提及方法但不请求（否定/对比/疑问/价格质疑）。
type RouteUtterances struct {
	Positive []string
	Negative []string
}

// Utterances 是 semantic router 的训练数据。
// 启动时一次性 embed 所有 utterance，存内存常驻。
var Utterances = map[string]RouteUtterances{
	"ziwei": {
		Positive: []string{
			"我想看紫微",
			"排个紫微盘",
			"紫微斗数分析",
			"看下我的星盘",
			"用紫微算命",
			"紫微排盘",
		},
		Negative: []string{
			"我不看紫微",
			"紫微和八字哪个准",
			"紫微准吗",
			"什么是紫微",
			"紫微太贵了",
			"紫微和八字区别",
		},
	},
	"qimen": {
		Positive: []string{
			"用奇门看一下",
			"起个奇门局",
			"遁甲预测",
			"奇门遁甲排盘",
			"帮我起奇门",
		},
		Negative: []string{
			"奇门是什么",
			"我不信奇门",
			"奇门和紫微区别",
			"奇门准吗",
			"奇门太玄乎了",
		},
	},
	"bazi": {
		Positive: []string{
			"排八字",
			"看我的八字",
			"算算命盘",
			"八字分析",
			"帮我排个八字",
		},
		Negative: []string{
			"什么是八字",
			"八字准吗",
			"我不信八字",
			"八字和紫微哪个准",
			"八字太复杂了",
		},
	},
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/intent/ -run TestUtterances -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/intent/utterances.go internal/intent/utterances_test.go
git commit -m "feat(intent): add positive/negative utterances for ziwei/qimen/bazi"
```

---

## Task 3: SemanticRouter 类型 + cosine helper

**Files:**
- Create: `internal/intent/semantic_router.go`
- Test: `internal/intent/semantic_router_test.go`

- [ ] **Step 1: 写失败测试（cosine helper）**

创建 `internal/intent/semantic_router_test.go`：

```go
package intent

import (
	"math"
	"testing"
)

func TestMaxCosine_ReturnsHighestSimilarity(t *testing.T) {
	msg := []float64{1, 0}
	candidates := [][]float64{
		{1, 0},    // 相同方向，cos=1.0
		{0, 1},    // 正交，cos=0
		{-1, 0},   // 反向，cos=-1
	}
	got := maxCosine(msg, candidates)
	if math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("maxCosine = %v, want 1.0", got)
	}
}

func TestMaxCosine_EmptyCandidates(t *testing.T) {
	got := maxCosine([]float64{1, 0}, nil)
	if got != 0 {
		t.Fatalf("maxCosine(nil) = %v, want 0", got)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/intent/ -run TestMaxCosine -v`
Expected: FAIL with `maxCosine undefined`

- [ ] **Step 3: 写类型定义和 cosine helper**

创建 `internal/intent/semantic_router.go`：

```go
// Package intent 提供面向用户消息的共享 lexical detector 和 semantic router。
package intent

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/embedding"
)

// Decision 是 semantic router 对一条用户消息的判定结果。
type Decision string

const (
	// DecisionPositive 表示用户显式提及并请求某方法，应覆盖 LLM 的 PrimaryDomain。
	DecisionPositive Decision = "positive"
	// DecisionNegative 表示用户提及方法但属于否定/对比/疑问，不应覆盖，也不退回 regex。
	DecisionNegative Decision = "negative"
	// DecisionNone 表示无足够信号，不覆盖，不退回 regex。
	DecisionNone Decision = "none"
)

// MatchResult 是 SemanticRouter.Match 的判定结果。
type MatchResult struct {
	Decision Decision
	Method   string  // positive 时为命中方法；negative 时为最佳负向匹配方法
	Score    float64 // positive 时为最佳正向相似度；negative 时为最佳负向相似度
}

// cachedRoute 是已 embed 的 utterance 向量，启动时一次性计算存内存。
type cachedRoute struct {
	Positive [][]float64
	Negative [][]float64
}

// SemanticRouter 持 embedder + 已 embed 的 utterance 向量，提供 Match 方法。
// router 是无状态、线程安全的（utterance 向量启动后只读，embedder 本身线程安全）。
type SemanticRouter struct {
	embedder  embedding.Embedder
	routes    map[string]cachedRoute
	threshold float64
}

// maxCosine 返回 msg 与 candidates 中所有向量的最大余弦相似度。
// 空候选集返回 0。维度不匹配的候选向量跳过。
func maxCosine(msg []float64, candidates [][]float64) float64 {
	best := 0.0
	for _, c := range candidates {
		if len(c) != len(msg) {
			continue
		}
		s := cosine(msg, c)
		if s > best {
			best = s
		}
	}
	return best
}

// cosine 计算两个向量的余弦相似度。
func cosine(a, b []float64) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
```

注意：文件顶部需加 `import "math"`，请补在 import 块里。

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/intent/ -run TestMaxCosine -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/intent/semantic_router.go internal/intent/semantic_router_test.go
git commit -m "feat(intent): add SemanticRouter types + cosine helper"
```

---

## Task 4: SemanticRouter.Match 算法

**Files:**
- Modify: `internal/intent/semantic_router.go`（加 Match 方法）
- Test: `internal/intent/semantic_router_test.go`（加测试用例）

- [ ] **Step 1: 写失败测试（5 个分支）**

在 `internal/intent/semantic_router_test.go` 末尾加：

```go
import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/cloudwego/eino/components/embedding"
	"go.uber.org/mock/gomock"
	mockembedding "github.com/cloudwego/eino/internal/mock/components/embedding"
)

func newMockEmbedder(t *testing.T, vectors map[string][][]float64, errByMsg map[string]error) *mockembedding.MockEmbedder {
	ctrl := gomock.NewController(t)
	m := mockembedding.NewMockEmbedder(ctrl)
	m.EXPECT().EmbedStrings(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, texts []string, _ ...embedding.Option) ([][]float64, error) {
			if len(texts) == 0 {
				return nil, nil
			}
			if err, ok := errByMsg[texts[0]]; ok {
				return nil, err
			}
			if v, ok := vectors[texts[0]]; ok {
				return v, nil
			}
			return [][]float64{{0, 0}}, nil
		}).AnyTimes()
	return m
}

func TestMatch_PositiveHit(t *testing.T) {
	// msg "排个紫微盘" 接近 ziwei positive，远离 negative
	embedder := newMockEmbedder(t, map[string][][]float64{
		"排个紫微盘": {{1, 0}},
		// ziwei positive 第一条假设向量 {0.95, 0} —— 测试里我们直接预设 utterance 向量
	}, nil)

	r := &SemanticRouter{
		embedder: embedder,
		routes: map[string]cachedRoute{
			"ziwei": {
				Positive: [][]float64{{1, 0}},
				Negative: [][]float64{{-1, 0}},
			},
		},
		threshold: 0.75,
	}

	got, err := r.Match(context.Background(), "排个紫微盘")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Decision != DecisionPositive {
		t.Fatalf("Decision = %q, want positive", got.Decision)
	}
	if got.Method != "ziwei" {
		t.Fatalf("Method = %q, want ziwei", got.Method)
	}
	if math.Abs(got.Score-1.0) > 1e-9 {
		t.Fatalf("Score = %v, want 1.0", got.Score)
	}
}

func TestMatch_NegativePriority(t *testing.T) {
	// msg "我不看紫微" 负向得分 ≥ 正向 → DecisionNegative，不退回 regex
	embedder := newMockEmbedder(t, map[string][][]float64{
		"我不看紫微": {{-1, 0}},
	}, nil)

	r := &SemanticRouter{
		embedder: embedder,
		routes: map[string]cachedRoute{
			"ziwei": {
				Positive: [][]float64{{1, 0}},  // msg 与正向反向，cos=-1
				Negative: [][]float64{{-1, 0}}, // msg 与负向同向，cos=1
			},
		},
		threshold: 0.75,
	}

	got, err := r.Match(context.Background(), "我不看紫微")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Decision != DecisionNegative {
		t.Fatalf("Decision = %q, want negative", got.Decision)
	}
}

func TestMatch_BelowThreshold(t *testing.T) {
	embedder := newMockEmbedder(t, map[string][][]float64{
		"今天天气": {{0.1, 0}},
	}, nil)

	r := &SemanticRouter{
		embedder: embedder,
		routes: map[string]cachedRoute{
			"ziwei": {Positive: [][]float64{{1, 0}}, Negative: [][]float64{{-1, 0}}},
		},
		threshold: 0.75,
	}

	got, err := r.Match(context.Background(), "今天天气")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Decision != DecisionNone {
		t.Fatalf("Decision = %q, want none", got.Decision)
	}
}

func TestMatch_EmptyMsg(t *testing.T) {
	r := &SemanticRouter{
		embedder: newMockEmbedder(t, nil, nil),
		routes:   map[string]cachedRoute{"ziwei": {Positive: [][]float64{{1, 0}}}},
		threshold: 0.75,
	}
	got, err := r.Match(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Decision != DecisionNone {
		t.Fatalf("Decision = %q, want none for empty msg", got.Decision)
	}
}

func TestMatch_EmbedderError(t *testing.T) {
	embedder := newMockEmbedder(t, nil, map[string]error{
		"排个紫微盘": errors.New("network timeout"),
	})

	r := &SemanticRouter{
		embedder: embedder,
		routes: map[string]cachedRoute{
			"ziwei": {Positive: [][]float64{{1, 0}}},
		},
		threshold: 0.75,
	}

	_, err := r.Match(context.Background(), "排个紫微盘")
	if err == nil {
		t.Fatal("expected err, got nil")
	}
}
```

注意：`newMockEmbedder` 函数会和 Task 3 里的 import 重复，请把 Task 3 的 import 块改成包含 `context`、`errors`、`mockembedding`、`gomock`，或在 Task 3 的测试文件里只放 cosine 测试，把 Match 测试单独放新文件。**为避免冲突，把 cosine 测试和 Match 测试都放在 `semantic_router_test.go` 同一个文件里，统一一个 import 块。**

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/intent/ -run TestMatch -v`
Expected: FAIL with `r.Match undefined`

- [ ] **Step 3: 实现 Match 方法**

在 `internal/intent/semantic_router.go` 末尾加：

```go
// Match 对用户消息做语义路由判定。
// 返回 (MatchResult, error)——error 与 Decision 分离，调用方据此区分：
//   - err != nil：embedder 调用失败，调用方退回 regex 兜底（受 Confidence 守卫约束）
//   - Decision == positive：用户显式提及某方法，应覆盖 LLM 的 PrimaryDomain
//   - Decision == negative：提及但属于否定/对比/疑问，不覆盖，**不退回 regex**
//   - Decision == none：无足够信号，不覆盖，不退回 regex
func (r *SemanticRouter) Match(ctx context.Context, msg string) (MatchResult, error) {
	if r == nil || r.embedder == nil || strings.TrimSpace(msg) == "" {
		return MatchResult{Decision: DecisionNone}, nil
	}

	vectors, err := r.embedder.EmbedStrings(ctx, []string{msg})
	if err != nil || len(vectors) == 0 {
		return MatchResult{}, err
	}
	msgVec := vectors[0]

	bestMethod, bestPosScore := "", 0.0
	bestNegMethod, bestNegScore := "", 0.0

	for name, route := range r.routes {
		posScore := maxCosine(msgVec, route.Positive)
		negScore := maxCosine(msgVec, route.Negative)
		if posScore > bestPosScore {
			bestPosScore, bestMethod = posScore, name
		}
		if negScore > bestNegScore {
			bestNegScore, bestNegMethod = negScore, name
		}
	}

	switch {
	case bestPosScore >= r.threshold && bestNegScore < bestPosScore:
		return MatchResult{Decision: DecisionPositive, Method: bestMethod, Score: bestPosScore}, nil
	case bestNegScore >= bestPosScore && bestNegScore >= r.threshold:
		return MatchResult{Decision: DecisionNegative, Method: bestNegMethod, Score: bestNegScore}, nil
	default:
		return MatchResult{Decision: DecisionNone}, nil
	}
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/intent/ -v`
Expected: PASS（所有 cosine + Match 用例）

- [ ] **Step 5: 提交**

```bash
git add internal/intent/semantic_router.go internal/intent/semantic_router_test.go
git commit -m "feat(intent): implement SemanticRouter.Match with negative priority"
```

---

## Task 5: NewSemanticRouter 构造函数（启动期 embed utterances）

**Files:**
- Modify: `internal/intent/semantic_router.go`（加构造函数）
- Test: `internal/intent/semantic_router_test.go`（加端到端测试）

- [ ] **Step 1: 写失败测试**

在 `semantic_router_test.go` 末尾加：

```go
func TestNewSemanticRouter_EmbedsAllUtterancesAtStartup(t *testing.T) {
	// mock embedder：任何输入都返回 {1, 0}，便于断言向量数 = utterance 数
	ctrl := gomock.NewController(t)
	m := mockembedding.NewMockEmbedder(ctrl)
	m.EXPECT().EmbedStrings(gomock.Any(), gomock.Any(), gomock.Any()).Return([][]float64{{1, 0}}, nil).AnyTimes()

	utterances := map[string]RouteUtterances{
		"ziwei": {
			Positive: []string{"a", "b"},
			Negative: []string{"c"},
		},
	}

	r, err := NewSemanticRouter(context.Background(), m, utterances, 0.75)
	if err != nil {
		t.Fatalf("NewSemanticRouter err: %v", err)
	}
	if r.routes["ziwei"].Positive == nil || len(r.routes["ziwei"].Positive) != 2 {
		t.Fatalf("positive vectors not cached at startup: %v", r.routes["ziwei"].Positive)
	}
	if len(r.routes["ziwei"].Negative) != 1 {
		t.Fatalf("negative vectors not cached at startup: %v", r.routes["ziwei"].Negative)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/intent/ -run TestNewSemanticRouter -v`
Expected: FAIL with `NewSemanticRouter undefined`

- [ ] **Step 3: 实现构造函数**

在 `semantic_router.go` 末尾加：

```go
// NewSemanticRouter 构造 router，启动时一次性 embed 所有 utterance 存内存。
// 失败时返回 error，调用方据此决定是否走 regex 兜底（router=nil）。
func NewSemanticRouter(ctx context.Context, embedder embedding.Embedder, utterances map[string]RouteUtterances, threshold float64) (*SemanticRouter, error) {
	if embedder == nil {
		return nil, errors.New("embedder is nil")
	}

	routes := make(map[string]cachedRoute, len(utterances))
	for name, u := range utterances {
		pos, err := embedder.EmbedStrings(ctx, u.Positive)
		if err != nil {
			return nil, fmt.Errorf("embed positive utterances for %s: %w", name, err)
		}
		neg, err := embedder.EmbedStrings(ctx, u.Negative)
		if err != nil {
			return nil, fmt.Errorf("embed negative utterances for %s: %w", name, err)
		}
		routes[name] = cachedRoute{Positive: pos, Negative: neg}
	}

	return &SemanticRouter{
		embedder:  embedder,
		routes:    routes,
		threshold: threshold,
	}, nil
}
```

记得在 import 块加 `"errors"` 和 `"fmt"`。

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/intent/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/intent/semantic_router.go internal/intent/semantic_router_test.go
git commit -m "feat(intent): add NewSemanticRouter constructor with startup embedding"
```

---

## Task 6: Embedding factory

**Files:**
- Create: `internal/llm/embedding_factory.go`
- Test: `internal/llm/embedding_factory_test.go`

- [ ] **Step 1: 写失败测试**

创建 `internal/llm/embedding_factory_test.go`：

```go
package llm

import (
	"context"
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/config"
)

func TestNewEmbedder_ReturnsNonNilWhenApiKeySet(t *testing.T) {
	cfg := &config.Config{
		EmbeddingApiKey:  "sk-test",
		EmbeddingBaseUrl: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		EmbeddingModel:   "text-embedding-v4",
	}
	embedder, err := NewEmbedder(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewEmbedder err: %v", err)
	}
	if embedder == nil {
		t.Fatal("embedder is nil")
	}
}

func TestNewEmbedder_ReturnsNilWhenApiKeyEmpty(t *testing.T) {
	cfg := &config.Config{
		EmbeddingApiKey: "",
	}
	embedder, err := NewEmbedder(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected nil err when api key empty, got %v", err)
	}
	if embedder != nil {
		t.Fatal("expected nil embedder when api key empty")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/llm/ -run TestNewEmbedder -v`
Expected: FAIL with `NewEmbedder undefined`

- [ ] **Step 3: 实现工厂**

创建 `internal/llm/embedding_factory.go`：

```go
package llm

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	einoopenai "github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/wikiglobal/suanming-agent/internal/config"
)

// NewEmbedder 根据配置构造 Eino Embedder。
// 用 eino-ext 的 openai embedding ext 指向 DashScope 的 OpenAI-compatible 端点
// （和知识库 KB 同款方案，支持自定义 base_url）。
// API key 为空时返回 nil, nil——调用方据此走 regex 兜底（router=nil）。
func NewEmbedder(ctx context.Context, cfg *config.Config) (embedding.Embedder, error) {
	if cfg.EmbeddingApiKey == "" {
		log.Println("WARNING: EMBEDDING_API_KEY not set — semantic router will fall back to regex")
		return nil, nil
	}

	embedder, err := einoopenai.NewEmbedder(ctx, &einoopenai.EmbeddingConfig{
		APIKey:  cfg.EmbeddingApiKey,
		BaseURL: cfg.EmbeddingBaseUrl,
		Model:   cfg.EmbeddingModel,
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return embedder, nil
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/llm/ -run TestNewEmbedder -v`
Expected: PASS

注意：测试可能因为 `go.mod` 缺少 eino-ext openai embedding 依赖而失败。如果失败，运行：

```bash
go get github.com/cloudwego/eino-ext/components/embedding/openai
go mod tidy
```

然后重跑测试。

- [ ] **Step 5: 提交**

```bash
git add internal/llm/embedding_factory.go internal/llm/embedding_factory_test.go go.mod go.sum
git commit -m "feat(llm): add NewEmbedder factory using eino-ext openai embedding"
```

---

## Task 7: 把 router 注入 supervisor.Client

**Files:**
- Modify: `internal/supervisor/client.go:70-82`
- Test: `internal/supervisor/client_test.go`（扩充）

- [ ] **Step 1: 写失败测试**

在 `internal/supervisor/client_test.go` 加：

```go
func TestNewClient_WithSemanticRouter(t *testing.T) {
	flash := &mockChat{}  // 复用已有的 mockChat
	router := &intent.SemanticRouter{}  // 空 router，仅测注入
	c := NewClient(flash, WithSemanticRouter(router))
	if c.router != router {
		t.Fatal("WithSemanticRouter did not set router field")
	}
}

func TestNewClient_WithoutSemanticRouter(t *testing.T) {
	flash := &mockChat{}
	c := NewClient(flash)
	if c.router != nil {
		t.Fatal("router should be nil by default")
	}
}
```

如果 `mockChat` 不在 client_test.go 里，参考已有测试找到 chat mock 的名字。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/supervisor/ -run TestNewClient_WithSemanticRouter -v`
Expected: FAIL with `c.router undefined` 或 `WithSemanticRouter undefined`

- [ ] **Step 3: 加字段和 option**

修改 `internal/supervisor/client.go`：

在 `Client` struct 加 `router` 字段：

```go
type Client struct {
	flash       llm.Chat
	routeEngine RouteEngine
	router      *intent.SemanticRouter
}
```

在 `WithRouteEngine` 后面加 `WithSemanticRouter`：

```go
// WithSemanticRouter 注入 semantic router，用于 applyExplicitMethodPreference
// 替代 regex MentionsXxxMethod。传 nil 等于不启用（走 regex 兜底）。
func WithSemanticRouter(router *intent.SemanticRouter) ClientOption {
	return func(c *Client) {
		c.router = router
	}
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/supervisor/ -run TestNewClient -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/supervisor/client.go internal/supervisor/client_test.go
git commit -m "feat(supervisor): add WithSemanticRouter option to Client"
```

---

## Task 8: applyExplicitMethodPreference 用 router（含三态开关 + confidence 守卫）

**Files:**
- Modify: `internal/supervisor/approved_route.go:39-133`
- Test: `internal/supervisor/approved_route_test.go`（新建或扩充）

- [ ] **Step 1: 写失败测试（enforce 模式）**

创建 `internal/supervisor/approved_route_test.go`：

```go
package supervisor

import (
	"context"
	"errors"
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/intent"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"go.uber.org/mock/gomock"
	mockembedding "github.com/cloudwego/eino/internal/mock/components/embedding"
)

// stubRouter 用预设 MatchResult，避免依赖真 embedder
type stubRouter struct {
	result intent.MatchResult
	err    error
	called bool
}

func (s *stubRouter) Match(ctx context.Context, msg string) (intent.MatchResult, error) {
	s.called = true
	return s.result, s.err
}

func newRouteWithConfidence(primary string, conf float64) policy.ApprovedRoute {
	return policy.ApprovedRoute{
		PrimaryDomain: primary,
		Confidence:    conf,
		PolicyHints:   schemas.PolicyHints{},
	}
}

func TestApplyExplicitMethodPreference_EnforcePositiveOverridesLLM(t *testing.T) {
	router := &stubRouter{result: intent.MatchResult{Decision: intent.DecisionPositive, Method: "ziwei"}}
	c := &Client{router: router, routerMode: "enforce"}

	route := newRouteWithConfidence("bazi", 0.9)  // LLM 高置信说 bazi
	c.applyExplicitMethodPreference(context.Background(), "排个紫微盘", &route)

	if route.PrimaryDomain != "ziwei" {
		t.Fatalf("PrimaryDomain = %q, want ziwei (router positive overrides LLM)", route.PrimaryDomain)
	}
	if !router.called {
		t.Fatal("router.Match not called in enforce mode")
	}
}

func TestApplyExplicitMethodPreference_EnforceNegativeDoesNotFallbackToRegex(t *testing.T) {
	router := &stubRouter{result: intent.MatchResult{Decision: intent.DecisionNegative}}
	c := &Client{router: router, routerMode: "enforce"}

	route := newRouteWithConfidence("bazi", 0.5)
	c.applyExplicitMethodPreference(context.Background(), "我不看紫微", &route)

	// negative 时不覆盖，不退回 regex——PrimaryDomain 应保持原值 bazi
	if route.PrimaryDomain != "bazi" {
		t.Fatalf("PrimaryDomain = %q, want bazi (negative should not override)", route.PrimaryDomain)
	}
}

func TestApplyExplicitMethodPreference_EnforceErrorFallbackGuardedByConfidence(t *testing.T) {
	router := &stubRouter{err: errors.New("network")}
	c := &Client{router: router, routerMode: "enforce"}

	// 高置信 + err → 不走 regex，信任 LLM
	high := newRouteWithConfidence("bazi", 0.9)
	c.applyExplicitMethodPreference(context.Background(), "排个紫微盘", &high)
	if high.PrimaryDomain != "bazi" {
		t.Fatalf("high confidence + err: PrimaryDomain = %q, want bazi", high.PrimaryDomain)
	}

	// 低置信 + err → 走 regex 兜底，"紫微" 命中 → ziwei
	router.err = errors.New("network")
	low := newRouteWithConfidence("bazi", 0.5)
	c.applyExplicitMethodPreference(context.Background(), "排个紫微盘看紫微", &low)
	if low.PrimaryDomain != "ziwei" {
		t.Fatalf("low confidence + err: PrimaryDomain = %q, want ziwei (regex fallback)", low.PrimaryDomain)
	}
}

func TestApplyExplicitMethodPreference_ShadowModeDoesNotOverride(t *testing.T) {
	router := &stubRouter{result: intent.MatchResult{Decision: intent.DecisionPositive, Method: "ziwei"}}
	c := &Client{router: router, routerMode: "shadow"}

	route := newRouteWithConfidence("bazi", 0.5)
	c.applyExplicitMethodPreference(context.Background(), "排个紫微盘", &route)

	// shadow 模式 router 跑了但只 log，决策仍走 regex（"紫微" 命中 regex → ziwei）
	if !router.called {
		t.Fatal("router.Match should be called in shadow mode for logging")
	}
	if route.PrimaryDomain != "ziwei" {
		t.Fatalf("shadow: PrimaryDomain = %q, want ziwei (regex decision)", route.PrimaryDomain)
	}
}

func TestApplyExplicitMethodPreference_OffModeSkipsRouter(t *testing.T) {
	router := &stubRouter{}
	c := &Client{router: router, routerMode: "off"}

	route := newRouteWithConfidence("bazi", 0.5)
	c.applyExplicitMethodPreference(context.Background(), "排个紫微盘看紫微", &route)

	if router.called {
		t.Fatal("router.Match should NOT be called in off mode")
	}
}
```

注意：`stubRouter` 需要实现 `Match(ctx, msg) (intent.MatchResult, error)` 方法——但这会让它满足某个 interface 吗？为了让 `Client.router` 字段类型可以是 `*intent.SemanticRouter` 或 stub，需要把 `Client.router` 改成 interface 类型。

**调整设计：** 把 `Client.router` 字段类型从 `*intent.SemanticRouter` 改成 interface：

在 `internal/intent/semantic_router.go` 加：

```go
// Router 是 semantic router 的接口，便于测试注入 stub。
type Router interface {
	Match(ctx context.Context, msg string) (MatchResult, error)
}
```

并让 `*SemanticRouter` 实现该接口（已经实现了 Match，自动满足）。

`Client.router` 字段类型改为 `intent.Router`。

`stubRouter` 自然实现 `intent.Router`，测试通过。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/supervisor/ -run TestApplyExplicitMethodPreference -v`
Expected: FAIL with `c.applyExplicitMethodPreference undefined` 或 `c.routerMode undefined`

- [ ] **Step 3: 实现 Router interface + 改 applyExplicitMethodPreference**

先在 `internal/intent/semantic_router.go` 加 Router interface（紧挨 MatchResult 定义后面）：

```go
// Router 是 semantic router 的接口，便于测试注入 stub。
// *SemanticRouter 默认满足此接口。
type Router interface {
	Match(ctx context.Context, msg string) (MatchResult, error)
}
```

然后修改 `internal/supervisor/client.go`：

- `router` 字段类型从 `*intent.SemanticRouter` 改成 `intent.Router`
- 加 `routerMode string` 字段

```go
type Client struct {
	flash       llm.Chat
	routeEngine RouteEngine
	router      intent.Router
	routerMode  string  // off | shadow | enforce
}
```

加 option：

```go
// WithRouterMode 设置 semantic router 的运行模式。
func WithRouterMode(mode string) ClientOption {
	return func(c *Client) {
		c.routerMode = mode
	}
}
```

然后修改 `internal/supervisor/approved_route.go`，把 `applyExplicitMethodPreference` 从自由函数改成 Client 方法：

```go
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

// applyRegexMethodPreference 是旧的 regex 硬覆盖逻辑，从原 applyExplicitMethodPreference 提取。
func applyRegexMethodPreference(msg string, route *policy.ApprovedRoute) {
	switch {
	case intent.MentionsZiweiMethod(msg):
		route.PrimaryDomain = "ziwei"
		route.SecondaryDomains = removeDomain(route.SecondaryDomains, "ziwei")
		if !hasDomain(route.SecondaryDomains, "bazi") {
			route.SecondaryDomains = append(route.SecondaryDomains, "bazi")
		}
		route.PolicyHints.QimenMode = "none"
		route.PolicyHints.NeedsQimen = false
		if route.PolicyHints.ProfileRequirement == "" {
			route.PolicyHints.ProfileRequirement = "full"
		}
	case intent.MentionsQimenMethod(msg):
		route.PrimaryDomain = "qimen"
		route.SecondaryDomains = removeDomain(route.SecondaryDomains, "qimen")
		route.PolicyHints.QimenMode = "primary"
		route.PolicyHints.NeedsQimen = true
		if route.PolicyHints.ProfileRequirement == "" {
			route.PolicyHints.ProfileRequirement = "none"
		}
	case intent.MentionsBaziMethod(msg):
		route.PrimaryDomain = "bazi"
		route.SecondaryDomains = removeDomain(route.SecondaryDomains, "bazi")
		if route.PolicyHints.QimenMode == "" {
			route.PolicyHints.QimenMode = "none"
		}
	case intent.ContainsTimingKeyword(msg) && route.PolicyHints.QimenMode == "none":
		route.PolicyHints.QimenMode = "supplement"
		route.PolicyHints.NeedsQimen = true
	}
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
```

最后改 `normalizeApprovedRoute` 调用点：原来是 `applyExplicitMethodPreference(msg, &route)`，改成 `c.applyExplicitMethodPreference(ctx, msg, &route)`。需要给 `normalizeApprovedRoute` 也传 `ctx`（如果它没有的话）。

检查 `normalizeApprovedRoute` 签名——它已经是 `func (c *Client) normalizeApprovedRoute(ctx context.Context, msg string, st *state.SessionState, route policy.ApprovedRoute) policy.ApprovedRoute`，有 ctx。直接把里面的 `applyExplicitMethodPreference(trimmed, &route)` 改成 `c.applyExplicitMethodPreference(ctx, trimmed, &route)`。

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/supervisor/ -v`
Expected: PASS（所有 applyExplicitMethodPreference 用例 + 已有用例不破）

- [ ] **Step 5: 跑全量测试确认没破其他包**

Run: `go test ./...`
Expected: PASS（特别 [client_test.go](../../internal/supervisor/client_test.go) 和 [preflight_test.go](../../internal/runtime/preflight_test.go)）

- [ ] **Step 6: 提交**

```bash
git add internal/intent/semantic_router.go internal/supervisor/client.go internal/supervisor/approved_route.go internal/supervisor/approved_route_test.go
git commit -m "feat(supervisor): applyExplicitMethodPreference uses router with 3-mode switch + confidence guard"
```

---

## Task 9: 把 router 注入 Executor + orchestrationRuntime

**Files:**
- Modify: `internal/runtime/executor.go:25-51`
- Modify: `internal/runtime/orchestration_state.go:41-44`
- Test: `internal/runtime/executor_test.go`（新建或扩充）

- [ ] **Step 1: 写失败测试**

创建或扩充 `internal/runtime/executor_test.go`：

```go
package runtime

import (
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/intent"
)

func TestExecutor_RouterField(t *testing.T) {
	e := &Executor{router: &intent.SemanticRouter{}}
	if e.router == nil {
		t.Fatal("router field not set")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/runtime/ -run TestExecutor_RouterField -v`
Expected: FAIL with `e.router undefined`

- [ ] **Step 3: 加 router 字段到 Executor**

修改 `internal/runtime/executor.go`，在 `Executor` struct 加字段：

```go
type Executor struct {
	reg                *tools.Registry
	flashChat          llm.Chat
	summarizerModel    einomodel.ToolCallingChatModel
	specialistRegistry *specialists.Registry
	builder            *AgentBuilder
	llmModel           string
	historyLimit       int
	orchestrationGraph compose.Runnable[string, string]
	cpStore            compose.CheckPointStore
	router             intent.Router  // semantic router，用于 preflight/guidance_gate；nil 走 regex
}
```

加 setter：

```go
// SetRouter 注入 semantic router，供 preflight/guidance_gate 使用。
// 传 nil 等于走旧 regex 兜底。
func (e *Executor) SetRouter(r intent.Router) { e.router = r }
```

修改 `internal/runtime/orchestration_state.go`，给 `orchestrationRuntime` 加 Router 字段：

```go
type orchestrationRuntime struct {
	Sink     EventSink
	Executor *Executor
	Router   intent.Router  // 从 Executor.router 传入，供 preflight 使用
}
```

修改 `internal/runtime/executor.go` 里两处 `withOrchestrationRuntime` 调用（`Execute` 和 `Resume`），加 Router 字段：

```go
ctx = withOrchestrationRuntime(ctx, &orchestrationRuntime{
	Sink:     sink,
	Executor: e,
	Router:   e.router,
})
```

（两处都要加 `Router: e.router`）

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/runtime/ -run TestExecutor_RouterField -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/runtime/executor.go internal/runtime/orchestration_state.go internal/runtime/executor_test.go
git commit -m "feat(runtime): add router field to Executor + orchestrationRuntime"
```

---

## Task 10: preflight + ShouldEnterGuidance + anyHardNegative 签名变更

**Files:**
- Modify: `internal/runtime/preflight.go:28-30`
- Modify: `internal/runtime/guidance_gate.go:16-87`
- Modify: `internal/runtime/orchestration_graph.go:41-57`
- Test: `internal/runtime/guidance_gate_test.go`（新建）

- [ ] **Step 1: 写失败测试**

创建 `internal/runtime/guidance_gate_test.go`：

```go
package runtime

import (
	"context"
	"testing"

	"github.com/wikiglobal/suanming-agent/internal/guidance"
	"github.com/wikiglobal/suanming-agent/internal/intent"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

type stubRouter struct {
	result intent.MatchResult
	err    error
}

func (s *stubRouter) Match(ctx context.Context, msg string) (intent.MatchResult, error) {
	return s.result, s.err
}

func newTestRoute(primary string, conf float64) policy.ApprovedRoute {
	return policy.ApprovedRoute{
		PrimaryDomain: primary,
		Confidence:    conf,
		PolicyHints:   schemas.PolicyHints{},
	}
}

func TestAnyHardNegative_RouterPositiveBreaksGuidance(t *testing.T) {
	r := &stubRouter{result: intent.MatchResult{Decision: intent.DecisionPositive, Method: "ziwei"}}
	route := newTestRoute("bazi", 0.5)
	signal := guidance.Signal{}

	got := anyHardNegative(r, "排个紫微盘", route, signal)
	if !got {
		t.Fatal("router positive should break guidance")
	}
}

func TestAnyHardNegative_RouterNegativeDoesNotBreak(t *testing.T) {
	r := &stubRouter{result: intent.MatchResult{Decision: intent.DecisionNegative}}
	route := newTestRoute("bazi", 0.5)
	signal := guidance.Signal{}

	got := anyHardNegative(r, "我不看紫微", route, signal)
	if got {
		t.Fatal("router negative should NOT break guidance (avoid regex hit)")
	}
}

func TestAnyHardNegative_NilRouterFallsBackToRegex(t *testing.T) {
	route := newTestRoute("bazi", 0.5)  // 低置信，regex 兜底启用
	signal := guidance.Signal{}

	got := anyHardNegative(nil, "排个紫微盘", route, signal)
	if !got {
		t.Fatal("nil router + low confidence + 紫微 keyword should break guidance via regex")
	}
}

func TestAnyHardNegative_NilRouterHighConfidenceSkipsRegex(t *testing.T) {
	route := newTestRoute("bazi", 0.9)  // 高置信，regex 兜底禁用
	signal := guidance.Signal{}

	got := anyHardNegative(nil, "排个紫微盘看紫微", route, signal)
	if got {
		t.Fatal("nil router + high confidence should skip regex, trust LLM")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/runtime/ -run TestAnyHardNegative -v`
Expected: FAIL with `anyHardNegative signature mismatch` 或 `undefined`

- [ ] **Step 3: 修改 guidance_gate.go 签名和实现**

修改 `internal/runtime/guidance_gate.go`：

```go
// ShouldEnterGuidance 判断本轮消息是否允许进入或继续 guidance。
// 只做 hard gate，不改 route / session / domain。
// router 非空时优先用 router，nil 走 regex 兜底（受 Confidence 守卫）。
func ShouldEnterGuidance(router intent.Router, message string, route policy.ApprovedRoute, st *state.SessionState) bool {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return false
	}

	signal := guidance.Sniff(trimmed)

	// ── Active guidance: check break conditions, then allow continuation ──
	if st.Guidance != nil {
		if anyHardNegative(router, trimmed, route, signal) {
			return false // break guidance: birth info / explicit method / explicit action
		}
		if route.PolicyHints.QimenMode == "primary" || intent.HasTimingFocus(trimmed) {
			return false
		}
		return true
	}

	// ── No active guidance: hard negative + collect_profile gate blocks entry ──
	if anyHardNegativeForNewEntry(router, trimmed, route, signal) {
		return false
	}

	if signal.ShouldOfferConsult() || signal.ShouldChooseTopic() {
		return true
	}

	return false
}

// anyHardNegative 检查硬性不进入/断开 guidance 的条件。
// router 非空时优先用 router；nil 或 err 走 regex 兜底（受 Confidence 守卫）。
func anyHardNegative(router intent.Router, msg string, route policy.ApprovedRoute, signal guidance.Signal) bool {
	if intent.ContainsBirthInfo(msg) {
		return true
	}
	if intent.ContainsExplicitDivinationAction(msg) {
		return true
	}

	// 术数方法提及——router 优先
	if router != nil {
		result, err := router.Match(context.Background(), msg)
		if err == nil {
			if result.Decision == intent.DecisionPositive {
				return true  // 显式提及方法 → 断 guidance
			}
			if result.Decision == intent.DecisionNegative || result.Decision == intent.DecisionNone {
				return false  // negative/none 不断——避免被 regex 击穿
			}
		}
		// err 落到 regex 兜底
	}

	// regex 兜底（router nil 或 err），受 Confidence 守卫
	if route.Confidence >= 0.7 {
		return false  // 高置信，禁用 dumb regex，信任 LLM
	}
	if intent.MentionsQimenMethod(msg) || intent.MentionsZiweiMethod(msg) || intent.MentionsBaziMethod(msg) {
		return true
	}
	return false
}

// anyHardNegativeForNewEntry 检查新入场的额外 hard negative。
func anyHardNegativeForNewEntry(router intent.Router, msg string, route policy.ApprovedRoute, signal guidance.Signal) bool {
	if anyHardNegative(router, msg, route, signal) {
		return true
	}
	hasGuidanceSignal := signal.ShouldOfferConsult() || signal.ShouldChooseTopic()
	if hasGuidanceSignal {
		return false
	}
	if route.PolicyHints.QimenMode == "primary" {
		return true
	}
	if intent.HasTimingFocus(msg) {
		return true
	}
	if route.TaskIntent == "collect_profile" || route.TaskIntent == "amend_profile" {
		return true
	}
	return false
}
```

修改 `internal/runtime/preflight.go`，给 `preflight` 函数加 router 参数：

```go
func preflight(st *state.SessionState, route policy.ApprovedRoute, message string, router intent.Router) preflightResult {
	if st.Guidance != nil || ShouldEnterGuidance(router, message, route, st) {
		// ... 其余不变
```

修改 `internal/runtime/orchestration_graph.go` 的 `preflightNode`，传 router：

```go
func preflightNode(ctx context.Context, in string) (string, error) {
	oc, err := loadOrchestrationCtx(ctx)
	if err != nil {
		return "", err
	}
	preflightSpan := tracing.SpanFromContext(ctx, "preflight", tracing.KindChain)
	preflightSpan.SetAttribute("primary_domain", oc.Init.Route.PrimaryDomain)
	preflightSpan.SetAttribute("task_intent", oc.Init.Route.TaskIntent)
	result := preflight(oc.Init.St, oc.Init.Route, oc.Init.UserMsg, oc.RT.Router)
	// ... 其余不变
```

需要给 preflight.go 加 `intent` 的 import。

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/runtime/ -v`
Expected: PASS（所有 anyHardNegative 用例 + 已有的 preflight_test 不破）

- [ ] **Step 5: 跑全量测试确认没破其他包**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/runtime/guidance_gate.go internal/runtime/guidance_gate_test.go internal/runtime/preflight.go internal/runtime/orchestration_graph.go
git commit -m "feat(runtime): preflight + guidance_gate use router with confidence guard"
```

---

## Task 11: Container 组装全部

**Files:**
- Modify: `internal/container/container.go:128-141`
- Test: `internal/container/container_test.go`（扩充）

- [ ] **Step 1: 写失败测试**

在 `internal/container/container_test.go` 加：

```go
func TestBuildContainer_RouterModeOff(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("ROUTER_MODE", "off")
	t.Setenv("EMBEDDING_API_KEY", "")  // 不配 embedding

	c := BuildContainer()
	if c == nil {
		t.Fatal("container is nil")
	}
	if c.Config.RouterMode != "off" {
		t.Fatalf("RouterMode = %q, want off", c.Config.RouterMode)
	}
}
```

（如果 BuildContainer 在 off 模式下因为没 embedding key 而 panic，说明构造逻辑有 bug，需要让 off 模式跳过 embedder 构造）

- [ ] **Step 2: 跑测试确认失败/通过**

Run: `go test ./internal/container/ -run TestBuildContainer_RouterModeOff -v`
Expected: 如果 panic 则 FAIL，需要修；如果通过则说明 off 模式已经能跳过

- [ ] **Step 3: 修改 container.go 构造 router 并注入**

在 `internal/container/container.go` 的 `BuildContainer()` 里，在构造 supervisor Client 之前加 router 构造：

```go
	// Semantic Router 构造（仅 enforce/shadow 模式）
	// off 模式或 embedder 构造失败时 router=nil，走旧 regex 兜底
	var router intent.Router
	if cfg.RouterMode == "enforce" || cfg.RouterMode == "shadow" {
		embedder, err := llm.NewEmbedder(context.Background(), cfg)
		if err != nil {
			log.Printf("[container] embedder init failed: %v — router disabled, falling back to regex", err)
		} else if embedder != nil {
			sr, err := intent.NewSemanticRouter(context.Background(), embedder, intent.Utterances, 0.75)
			if err != nil {
				log.Printf("[container] semantic router init failed: %v — falling back to regex", err)
			} else {
				router = sr
				log.Printf("[container] semantic router initialized in %s mode", cfg.RouterMode)
			}
		}
	}

	// Supervisor 客户端固定使用 ADK route engine；外层 text fallback 仍由 Go supervisor 保留。
	routeEngine := mustNewSupervisorRouteEngine(cfg, flashModel)
	supervisorClient := supervisor.NewClient(
		flashClient,
		supervisor.WithRouteEngine(routeEngine),
		supervisor.WithSemanticRouter(router),
		supervisor.WithRouterMode(cfg.RouterMode),
	)
	orch.SetSupervisor(supervisorClient)

	// Executor 也注入 router（preflight 用）
	executor.SetRouter(router)
```

注意：需要给 container.go 加 `log` 和 `github.com/wikiglobal/suanming-agent/internal/intent` 的 import。

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/container/ -v`
Expected: PASS

- [ ] **Step 5: 跑全量测试**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/container/container.go internal/container/container_test.go
git commit -m "feat(container): wire semantic router into supervisor + executor"
```

---

## Task 12: 回归测试集（真 embedder）

**Files:**
- Create: `internal/intent/semantic_router_regression_test.go`

- [ ] **Step 1: 写回归测试**

创建 `internal/intent/semantic_router_regression_test.go`：

```go
package intent

import (
	"context"
	"os"
	"testing"

	einoopenai "github.com/cloudwego/eino-ext/components/embedding/openai"
)

// TestRegression_RouterAccuracy 用真 DashScope embedder 跑回归集。
// 无 EMBEDDING_API_KEY 时 skip——CI 不跑，本地手动验证。
func TestRegression_RouterAccuracy(t *testing.T) {
	apiKey := os.Getenv("EMBEDDING_API_KEY")
	if apiKey == "" {
		t.Skip("EMBEDDING_API_KEY not set — skipping regression test")
	}

	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-v4"
	}

	embedder, err := einoopenai.NewEmbedder(context.Background(), &einoopenai.EmbeddingConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
		Timeout: 30 * 1000000000,  // 30s
	})
	if err != nil {
		t.Fatalf("NewEmbedder err: %v", err)
	}

	router, err := NewSemanticRouter(context.Background(), embedder, Utterances, 0.75)
	if err != nil {
		t.Fatalf("NewSemanticRouter err: %v", err)
	}

	positiveCases := []struct {
		msg    string
		method string
	}{
		{"排个紫微盘", "ziwei"},
		{"我想看紫微斗数", "ziwei"},
		{"用奇门遁甲起一局", "qimen"},
		{"帮我排八字", "bazi"},
		{"看下我的八字命盘", "bazi"},
	}
	for _, c := range positiveCases {
		t.Run("positive_"+c.msg, func(t *testing.T) {
			got, err := router.Match(context.Background(), c.msg)
			if err != nil {
				t.Fatalf("Match err: %v", err)
			}
			if got.Decision != DecisionPositive {
				t.Fatalf("msg=%q Decision=%q, want positive (score=%v)", c.msg, got.Decision, got.Score)
			}
			if got.Method != c.method {
				t.Fatalf("msg=%q Method=%q, want %q", c.msg, got.Method, c.method)
			}
		})
	}

	negativeCases := []string{
		"我不看紫微",
		"紫微和八字哪个准",
		"紫微准吗",
		"什么是紫微",
		"紫微太贵了",
		"我不信奇门",
		"八字是什么",
		"八字准吗",
	}
	for _, msg := range negativeCases {
		t.Run("negative_"+msg, func(t *testing.T) {
			got, err := router.Match(context.Background(), msg)
			if err != nil {
				t.Fatalf("Match err: %v", err)
			}
			if got.Decision == DecisionPositive {
				t.Fatalf("msg=%q should NOT be positive (got positive, method=%q)", msg, got.Method)
			}
		})
	}

	edgeCases := []string{
		"今天天气怎么样",
		"排盘",  // 无方法
		"",
	}
	for _, msg := range edgeCases {
		t.Run("edge_"+msg, func(t *testing.T) {
			got, _ := router.Match(context.Background(), msg)
			if got.Decision == DecisionPositive {
				t.Fatalf("msg=%q should NOT be positive", msg)
			}
		})
	}
}
```

- [ ] **Step 2: 跑测试确认 skip（无 key）**

Run: `go test ./internal/intent/ -run TestRegression -v`
Expected: SKIP with `EMBEDDING_API_KEY not set`

- [ ] **Step 3: 配置 env 跑真测试（手动验证）**

```bash
export EMBEDDING_API_KEY=sk-your-dashscope-key
export EMBEDDING_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
export EMBEDDING_MODEL=text-embedding-v4
go test ./internal/intent/ -run TestRegression -v
```

Expected: 所有 positive 命中对应方法，所有 negative 不命中 positive，edge 不命中 positive。

如果准确率 < 95%：
- 调 threshold（0.75 太高 → 降到 0.7；太低 → 升到 0.8）
- 补充 utterance（看哪些 case 失败，加正向或负向 utterance）
- 验证 v4 是否真兼容 ext（如不兼容，改用 `text-embedding-v3`）

- [ ] **Step 4: 提交**

```bash
git add internal/intent/semantic_router_regression_test.go
git commit -m "test(intent): add regression test set for semantic router (real embedder)"
```

---

## Self-Review

**1. Spec coverage:**

- ✅ Section 4.1 改造范围（applyExplicitMethodPreference + anyHardNegative）→ Task 8 + Task 10
- ✅ Section 4.2 三条核心策略（Negative 优先 / Confidence 守卫 / Regex 兜底）→ Task 4 + Task 8 + Task 10
- ✅ Section 4.3 数据流 → Task 8 + Task 10 + Task 11
- ✅ Section 4.4 错误降级链 → Task 4 (Match 返回 error) + Task 8 + Task 10 + Task 11
- ✅ Section 5 组件与文件清单 → 所有 task 覆盖
- ✅ Section 5.1 Router 注入路径 → Task 9 + Task 10 + Task 11
- ✅ Section 6 Match 算法 → Task 4
- ✅ Section 7.1 单元测试 → Task 3 + Task 4 + Task 5
- ✅ Section 7.2 回归测试集 → Task 12
- ✅ Section 7.3 集成测试（shadow + enforce）→ Task 8 + Task 10
- ✅ Section 8 三态开关 → Task 1 (config) + Task 8 (applyExplicitMethodPreference)
- ✅ Section 9 开放问题（v4 兼容性）→ Task 12 Step 3 验证

**2. Placeholder scan:**

无 TBD/TODO。每步都有完整代码。

**3. Type consistency:**

- `MatchResult{Decision, Method, Score}` — Task 3 定义，Task 4/8/10 使用 ✅
- `Router interface { Match(ctx, msg) (MatchResult, error) }` — Task 8 定义，Task 9/10/11 使用 ✅
- `SemanticRouter` struct — Task 3 定义，Task 4/5 扩展 ✅
- `cachedRoute{Positive, Negative}` — Task 3 定义，Task 4/5 使用 ✅
- `Client.router` 字段类型 `intent.Router`（interface）— Task 8 定义，Task 7/11 使用 ✅
- `Client.routerMode` 字段 — Task 8 定义，Task 11 通过 `WithRouterMode` 设置 ✅
- `Executor.router` 字段类型 `intent.Router` — Task 9 定义，Task 11 通过 `SetRouter` 设置 ✅
- `orchestrationRuntime.Router` 字段 — Task 9 定义，Task 10 通过 `oc.RT.Router` 读取 ✅

类型一致。

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-26-embedding-intent-router.md`. Two execution options:

1. **Subagent-Driven (recommended)** — 每个 task 派 fresh subagent，task 之间 review，快速迭代
2. **Inline Execution** — 当前 session 内顺序跑，batch + checkpoint review

Which approach?
