# Embedding-based Intent Router 设计

**日期：** 2026-06-26
**状态：** Draft — 待用户审核
**作者：** Hyde + Claude（结对）

## 1. 背景与问题

项目当前在 LLM supervisor 之上保留了一层**确定性硬规则覆盖**——[applyExplicitMethodPreference](../../internal/supervisor/approved_route.go:97) 在用户消息命中「紫微/奇门/八字」关键词时，无条件覆盖 LLM 的 `PrimaryDomain` 决策。

该设计基于正则匹配（[internal/intent/markers.go](../../internal/intent/markers.go:1)），存在两个痛点：

1. **假阳性**：正则只看字面，不识否定/对比/疑问。例如「我不看紫微」「紫微和八字哪个准」「紫微准吗」都会触发紫微路由。
2. **覆盖 LLM 正确判断**：[applyExplicitMethodPreference](../../internal/supervisor/approved_route.go:97) 不看 `Confidence`，LLM 即使高置信判断对了也会被硬规则改错。

## 2. 目标 / 非目标

### 目标

- 用 **embedding 语义路由**替代 `MentionsZiweiMethod / MentionsQimenMethod / MentionsBaziMethod` 三个 regex 函数的调用点
- 解决假阳性：能识别「提到了但不想算」的句式（否定/对比/疑问/价格质疑）
- 解决 LLM 被错误覆盖：高置信时跳过覆盖
- 保留确定性兜底：embedding 故障时 supervisor 不失能

### 非目标

- **不动** [ContainsBirthInfo](../../internal/intent/markers.go:35) —— 事实提取（日期模式），regex 是对的工具
- **不动** [ContainsExplicitDivinationAction](../../internal/intent/markers.go:55) —— 动作意图（"帮我算/排盘"），不同维度
- **不动** [guidance.Sniff()](../../internal/guidance/sniff.go:28) —— guidance 嗅探独立 concern
- **不替换** LLM supervisor 本身——embedding 只做"显式方法提及"这一子判断
- **不上向量 DB**——utterance 集小（~60 向量），内存常驻即可

## 3. 方案选型

### 3.1 为什么是 embedding 而非其他

参考 [aurelio-labs/semantic-router](https://github.com/aurelio-labs/semantic-router) 的 production 模式：每个 route 配一组**正向 utterance**（"我想看紫微"）+ **负向 utterance**（"我不看紫微"），用余弦相似度匹配，sub-ms 路由，无 LLM 调用。

对比三个候选方案：

| 方案 | 假阳性解决 | LLM 覆盖解决 | 新依赖 | 学习价值 |
|---|---|---|---|---|
| A. regex 加否定识别 + confidence 守卫 | 部分 | 部分 | 0 | 低 |
| **B. Embedding 语义路由（本方案）** | 彻底 | 彻底 | Eino Embedder | 高 |
| C. LLM 自判 explicit_method_seen 字段 | 彻底 | 彻底 | 0 | 中 |

选 **B**：项目核心栈是 Eino（AGENTS.md 关键决策 3），eino-ext 有现成 dashscope 实现；utterance 向量启动时一次性 embed，每轮只 embed 用户消息，不增加 LLM 调用次数。

### 3.2 Embedding provider 选型

复用知识库（KB）已验证的配置：

| 项 | 值 | 来源 |
|---|---|---|
| Provider | DashScope（阿里云） | 国内 provider，中文强 |
| Model | `text-embedding-v4` | KB 实际在用 |
| 实现 | [eino-ext/components/embedding/dashscope](../../eino-agent/eino-ext/components/embedding/dashscope/) 原生 ext | 无需 OpenAI-compat 中间层 |
| Env vars | `EMBEDDING_API_KEY` / `EMBEDDING_MODEL` | 沿用 KB 同款变量名 |

理由：用户明确要求"国内 + 跟知识库一样"；DashScope 是国内 provider；`text-embedding-v4` 是阿里最新中文 embedding 模型；eino-ext 原生支持，配置仅 `APIKey + Model + Timeout` 三字段。

## 4. 架构

### 4.1 改造范围

**替换**：`MentionsZiweiMethod / MentionsQimenMethod / MentionsBaziMethod` 三个 regex 函数及其调用点：

| 调用点 | 文件 | 当前行为 | 改造后 |
|---|---|---|---|
| 1 | [applyExplicitMethodPreference](../../internal/supervisor/approved_route.go:97) | 无条件覆盖 LLM `PrimaryDomain` | 优先 router.Match，nil/失败退回 regex；加 Confidence 守卫 |
| 2 | [anyHardNegative](../../internal/runtime/guidance_gate.go:51) | 关键词命中即退 guidance | 同上 |

**不替换**：`ContainsBirthInfo`、`ContainsExplicitDivinationAction`、`HasTimingFocus`、`ContainsTimingKeyword`、`guidance.Sniff()`。

**保留**：`MentionsXxxMethod` 函数**不删**，作为 regex 兜底路径长期存在（P4 阶段才考虑清理）。

### 4.2 三条核心策略

| 策略 | 解决的痛点 | 实现方式 |
|---|---|---|
| **Negative 优先** | 假阳性（"我不看紫微"） | 每方法配 5-10 条正向 + 5-10 条负向 utterance。`Match` 时若任一负向得分 ≥ 最佳正向得分 → 不覆盖 |
| **Confidence 守卫** | LLM 被错误覆盖 | `route.Confidence >= 0.7` 时跳过 router，完全信任 LLM（多数情况 0 次 embedding 调用） |
| **Regex 兜底** | embedding 故障失能 | 启动期/运行期 embedder 失败 → 退回 `MentionsXxxMethod`，log 记录。supervisor 永不因 embedding 故障失能 |

### 4.3 数据流

**启动期**（container 构造时）：

```
container 启动
  → embedding_factory.NewEmbedder(cfg) → dashscope.NewEmbedder(APIKey, Model, Timeout)
  → intent.NewSemanticRouter(embedder, utterances.Utterances)
       → embedder.EmbedStrings(all_utterances)  // 一次性 embed ~60 条
       → 向量存内存
  → router 注入 supervisor.Client 和 runtime
  → 任一环节失败：log + router=nil，应用继续启动
```

**每轮**（supervisor.Approve）：

```
Decide (LLM) → SupervisorDecision{Confidence, PrimaryDomain}
  ↓
policy.Apply → ApprovedRoute
  ↓
normalizeApprovedRoute → applyExplicitMethodPreference:
  ├─ c.router == nil              → 走旧 regex MentionsXxxMethod（兜底）
  ├─ route.Confidence >= 0.7      → 跳过覆盖（信任 LLM）
  └─ else → c.router.Match(ctx, msg):
       ├─ embed msg (~50ms)
       ├─ 每方法算 best_pos = max(cosine(msg, pos_vecs))
       │            best_neg = max(cosine(msg, neg_vecs))
       ├─ best_pos < 0.75          → 不覆盖
       ├─ best_neg >= best_pos      → 不覆盖（Negative 优先）
       └─ else                      → 覆盖 PrimaryDomain 到 best_pos 方法
```

### 4.4 错误降级链

| 故障场景 | 行为 | 用户影响 |
|---|---|---|
| 启动期 embedder 初始化失败（API key 错、网络不通） | log error + `router=nil` | 应用正常启动，路由走旧 regex |
| 启动期 utterance embedding 部分失败 | log warning，跳过失败方法，已成的仍用 router | router 部分生效 |
| 运行期 `router.Match` 调 embedder 失败 | log warning，**本次**退回 regex | 不影响后续轮 |

降级粒度到「单次调用」——一次 embedding 超时不影响下轮。

## 5. 组件与文件清单

### 新增文件

| 文件 | 职责 |
|---|---|
| `internal/llm/embedding_factory.go` | `NewEmbedder(cfg) (embedding.Embedder, error)`——构造 dashscope ext |
| `internal/intent/semantic_router.go` | `SemanticRouter` 类型：持 embedder + utterance 向量；`Match(ctx, msg) (method, score, ok)` |
| `internal/intent/utterances.go` | 每方法 5-10 条正向 + 5-10 条负向 utterance 定义 |
| `internal/intent/semantic_router_test.go` | 单元测试（mock Embedder） |
| `internal/intent/semantic_router_regression_test.go` | 回归测试集（真 embedder，无 key 则 `t.Skip`） |

### 修改文件

| 文件 | 改什么 |
|---|---|
| [internal/config/config.go](../../internal/config/config.go:39) | 加 `EmbeddingProvider/EmbeddingApiKey/EmbeddingBaseUrl/EmbeddingModel` 字段 + env 读取 |
| [internal/supervisor/client.go](../../internal/supervisor/client.go:70) | `Client` 加 `router *intent.SemanticRouter` 字段 + `WithSemanticRouter` option |
| [internal/supervisor/approved_route.go](../../internal/supervisor/approved_route.go:97) | `applyExplicitMethodPreference` 改为 Client 方法；优先 router，nil/失败退回 regex；加 `Confidence < 0.7` 守卫 |
| [internal/runtime/guidance_gate.go](../../internal/runtime/guidance_gate.go:51) | `anyHardNegative` 同样优先 router，失败退回 regex。**注入方式**：`ShouldEnterGuidance` 和 `anyHardNegative` 加 `router *intent.SemanticRouter` 参数（当前是自由函数），调用方（orchestration_graph）从 container 取 router 传入 |
| [internal/container/container.go](../../internal/container/container.go:1) | 启动时构造 Embedder → SemanticRouter，注入 Client 和 runtime |

### 5.1 Router 注入路径

router 是无状态、线程安全的（启动后 utterance 向量只读，embedder 本身线程安全），构造一次后全局共享。注入路径：

```
container.NewContainer(cfg)
  → embedding_factory.NewEmbedder(cfg) → embedder
  → intent.NewSemanticRouter(embedder, utterances.Utterances) → router
  → supervisor.NewClient(flash, supervisor.WithSemanticRouter(router))
  → runtime 的 ShouldEnterGuidance 调用点：把 router 作为参数传入
```

`ShouldEnterGuidance` 当前签名 `func ShouldEnterGuidance(message string, route, st) bool` 改为 `func ShouldEnterGuidance(router *intent.SemanticRouter, message string, route, st) bool`。所有调用方（在 orchestration_graph.go）同步加参数。

### 不动

- [internal/intent/markers.go](../../internal/intent/markers.go:1) 的 `MentionsXxxMethod`——留作 regex 兜底
- `ContainsBirthInfo / ContainsExplicitDivinationAction / HasTimingFocus / ContainsTimingKeyword`
- [internal/guidance/sniff.go](../../internal/guidance/sniff.go:1)

## 6. 关键算法：SemanticRouter.Match

```go
func (r *SemanticRouter) Match(ctx context.Context, msg string) (method string, score float64, ok bool) {
    if r == nil || r.embedder == nil || strings.TrimSpace(msg) == "" {
        return "", 0, false
    }

    vectors, err := r.embedder.EmbedStrings(ctx, []string{msg})
    if err != nil || len(vectors) == 0 {
        return "", 0, false  // 调用方退回 regex
    }
    msgVec := vectors[0]

    bestMethod := ""
    bestScore := 0.0

    for name, route := range r.routes {
        posScore := maxCosine(msgVec, route.Positive)
        negScore := maxCosine(msgVec, route.Negative)

        if posScore < r.threshold {
            continue
        }
        if negScore >= posScore {
            continue  // Negative 优先
        }
        if posScore > bestScore {
            bestScore = posScore
            bestMethod = name
        }
    }

    return bestMethod, bestScore, bestMethod != ""
}
```

**余弦相似度** 用本地 helper，无需外部依赖。

**Threshold 默认 0.75**，P2 阶段用回归集校准。

## 7. 测试策略

### 7.1 单元测试（mock Embedder）

`internal/intent/semantic_router_test.go`，用 [eino/internal/mock/components/embedding](../../eino-agent/eino/internal/mock/components/embedding)：

- positive 命中 → 返回对应方法
- negative 优先 → "我不看紫微" 不命中
- threshold 卡控 → 低分不命中
- empty 输入 → ok=false
- embedder 返回 error → ok=false（调用方退回 regex）

### 7.2 回归测试集（真 embedder）

`internal/intent/semantic_router_regression_test.go`，需 `EMBEDDING_API_KEY`，无则 `t.Skip`：

| 类型 | 用例 | 期望 |
|---|---|---|
| 正向 | "排个紫微盘"、"我想看紫微斗数"、"用奇门遁甲起一局"、"帮我排八字"、"看下我的八字命盘" | 命中对应方法 |
| 负向 | "我不看紫微"、"紫微和八字哪个准"、"紫微准吗"、"什么是紫微"、"紫微太贵了"、"我不信奇门"、"八字是什么"、"八字准吗" | **不覆盖** |
| 边缘 | "今天天气怎么样"、"排盘"（无方法）、"" | 不覆盖 |

### 7.3 集成测试

`internal/supervisor/approved_route_test.go` 加用例：

- LLM `Confidence=0.9` + router 命中 → 不覆盖（守卫生效）
- LLM `Confidence=0.5` + router 命中 → 覆盖
- `router=nil` → 退回 regex
- router 调用失败 → 退回 regex

`internal/runtime/guidance_gate_test.go`（新增或扩充）加用例：

- `anyHardNegative` 用 router 判定："我不看紫微" → 不退 guidance
- `anyHardNegative` 用 router 判定："排个紫微盘" → 退 guidance
- `router=nil` → 退回 regex `MentionsXxxMethod`

### 7.4 现有测试不破

`go test ./...` 全过，特别 [preflight_test.go](../../internal/runtime/preflight_test.go) 和 [client_test.go](../../internal/supervisor/client_test.go)。

实现阶段调用项目的 **`agent-test-suites` skill** 跑路由回归测试集（AGENTS.md 列出的 skill 之一）。

## 8. 上线分阶段

| 阶段 | 做什么 | 验证标准 |
|---|---|---|
| **P1 旁路** | router 跑出结果但**只 log**，决策仍走旧 regex | 收集一周「router 会怎么判 vs regex 实际怎么判」的分歧日志 |
| **P2 校准** | 用 P1 日志 + 回归集调 threshold 和 utterance | 回归集准确率 ≥ 95% |
| **P3 切换** | router 接入决策，regex 降为兜底；加 metric：router 命中率/退回率/override 率 | 线上跑 2 周无事故 |
| **P4 清理**（未来） | 删 `MentionsXxxMethod` 和兜底分支 | — |

P1 旁路模式是关键——零风险上线，纯观测，先看分歧再决定是否切换。

## 9. 风险与开放问题

### 风险

- **DashScope API key 需要单独申请**：KB 已有，Go 后端可复用同一 key 或新申请——上线前确认
- **`text-embedding-v4` 在 eino-ext dashscope ext 的 model 列表里只列了 v1/v2/v3**：但 `Model` 是字符串参数，ext 不做白名单校验，v4 应能直接用。P1 旁路阶段验证
- **Threshold 0.75 是经验值**：不同 embedding 模型的分数分布不同，必须用 P2 校准
- **utterance 集冷启动**：初版 ~60 条需要人工编写，P1 旁路日志会暴露遗漏的句式，迭代补充

### 开放问题

无——四个 section 均已获用户确认。实现阶段如发现新问题，回写本 spec。

## 10. 参考资源

- [aurelio-labs/semantic-router](https://github.com/aurelio-labs/semantic-router) —— 开源 semantic router 库，本方案的 negative utterance 模式来源
- [eino-ext/components/embedding/dashscope](../../eino-agent/eino-ext/components/embedding/dashscope/) —— Eino 框架的 DashScope embedding 实现
- [Eino Embedder 接口](../../eino-agent/eino/components/embedding/interface.go:37) —— `EmbedStrings(ctx, texts, opts) ([][]float64, error)`
- 知识库 embedding 配置参考：[knowledge/src/lib/embeddings.ts](../../knowledge/src/lib/embeddings.ts:1)
