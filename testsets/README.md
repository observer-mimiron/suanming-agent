# 命理大师测试集

标准化测试与 Agent 行为回归验证。

## 测试框架架构

```
Suite (JSONL + _context) → 声明式断言 + {占位符}
    ↓
CanonicalTrace           → 规范模型 (steps/actions/retrievals/route)
    ↓
Adapter (YAML + .py)     → 映射配置 (这个 agent 的 event→规范字段)
    ↓
Agent Raw Output (SSE / JSONL / ...)
```

- **CanonicalTrace**: 架构无关的 agent 执行轨迹模型（`testsets/suites/canonical.py`），断言层唯一数据源
- **Adapter**: 描述 agent 特定输出格式到 CanonicalTrace 的映射，写成 YAML 配置（`testsets/adapters/`）
- **Suite**: 测试用例（JSONL），用 `{占位符}` 引用领域特定值（工具名、domain 名等），通过 `_context` 定义
- **架构变化时**：只改 adapter.yaml 或 suite 的 `_context`，不碰 Python 代码

### 使用方式

```bash
# 旧格式（向后兼容，不需要 adapter）
python3 testsets/suites/runner.py testsets/suites/flow-basic.jsonl http://localhost:18080

# 新格式（需要 --adapter）
python3 testsets/suites/runner.py testsets/suites/component/route-basic.jsonl \
  http://localhost:18080 \
  --adapter testsets/adapters/suanming-agent-sse.yaml
```

## 断言类型

### 新格式断言（架构无关）

| 类别 | 断言字段 | 参数类型 | 说明 |
|------|---------|---------|------|
| HTTP | `http_status` | int | HTTP 状态码 |
| 路由 | `route_primary` | str | 路由主域 |
| 路由 | `route_intent_any` | [str] | 任务意图（任一） |
| 动作 | `action_called` | [str] | 指定动作被调用 |
| 动作 | `action_not_called` | [str] | 指定动作未被调用 |
| 动作 | `action_sequence` | [str] | 动作调用顺序（subsequence） |
| 动作 | `action_arg_match` | {name: {arg: val}} | 动作参数匹配 |
| 动作 | `action_result_not_empty` | [str] | 动作返回非空 |
| 检索 | `retrieval_happened` | bool | 检索发生过 |
| 检索 | `retrieval_has_results` | bool | 检索返回了结果 |
| 检索 | `retrieval_cited` | bool | 检索结果被引用到回复 |
| 步骤 | `step_count_range` | [min, max] | 步骤数范围 |
| 步骤 | `no_errors` | bool | 无执行错误 |
| 输出 | `final_output_contains` | [str] | 最终输出含关键词 |
| 输出 | `final_output_not_contains` | [str] | 最终输出不含关键词 |

### 旧格式断言（向后兼容，不需要 adapter）

| 断言字段 | 说明 |
|---------|------|
| `http_status` | HTTP 状态码 |
| `turn_type` / `turn_type_any` | 轮次类型 |
| `route_primary` | 路由主域 |
| `task_intent` / `task_intent_any` | 任务意图 |
| `contains_any` / `contains_all` | 关键词包含 |
| `not_contains` | 禁止关键词 |
| `knowledge_search` | 知识库检索触发 |

## 目录结构

```
testsets/
├── README.md                           # 本文件
├── raw/                                # 原始数据
│   ├── mingli-bench-160.json           # 主力测试集 (160题, JSON)
│   ├── test-cases.md                   # 可读版测试用例
│   ├── competition-2022.json           # 2022年大赛真题
│   ├── competition-2023.json           # 2023年大赛真题
│   ├── competition-2024.json           # 2024年大赛真题
│   ├── competition-2025.json           # 2025年大赛真题
│   └── search-test-cases.json          # 检索类测试
├── suites/                             # 测试套件 (JSONL)
│   ├── runner.py                       # 回归测试执行器
│   ├── flow-basic.jsonl                # [Smoke] 基础会话流程
│   ├── quiz-marriage.jsonl             # [Standard] 婚姻感情类
│   ├── quiz-career-wealth.jsonl        # [Standard] 事业财运类
│   ├── quiz-ziwei.jsonl               # [Standard] 紫微斗数专项
│   ├── edge-input.jsonl               # [Standard] 边界输入
│   ├── quiz-knowledge.jsonl            # [Exhaustive] 知识库检索
│   ├── quiz-knowledge-edge.jsonl        # [Exhaustive] 知识图谱专项
│   ├── quiz-year-event.jsonl           # [Exhaustive] 流年专项
│   ├── edge-adversarial.jsonl          # [Exhaustive] 对抗性输入
│   └── edge-resilience.jsonl           # [Exhaustive] 容错降级
├── knowledge/                          # 基础知识数据集
│   ├── ten-gods.json                   # 十神定义
│   ├── sixty-jiazi.json                # 六十甲子纳音
│   ├── heavenly-stems.json             # 十天干属性
│   ├── earthly-branches.json           # 十二地支属性
│   ├── five-elements.json              # 五行生克基础
│   ├── element-combinations.json       # 五行合化规则
│   ├── luck-cycles.json                # 大运流年规律
│   ├── seasonal-patterns.json          # 四季五行旺衰
│   └── compatibility-matrix.json       # 合婚矩阵
└── reports/                            # 测试报告输出
```

---

## 第一节：套件说明

### flow-basic — 基础会话流程
验证新建会话、提供八字、追问、session 复用等基础流程。

| 用例 | 场景 |
|------|------|
| new-session-birth | 首次输入出生信息 |
| new-session-female | 农历日期 + 女性 |
| followup-after-birth | 同一会话追问财运 |
| followup-marriage | 同一会话追问婚姻 |
| no-birth-info | 未提供八字时的闲聊 |

### quiz-marriage — 婚姻感情
验证婚姻类问题的回答质量。

| 用例 | 场景 |
|------|------|
| marriage-female-timing | 女性问何时结婚 |
| marriage-male-quality | 男性问正缘 |
| marriage-followup-spouse | 追问配偶信息 |
| marriage-problem | 特定命例婚姻分析 |
| marriage-divorce-risk | 婚姻风险评估 |

### quiz-career-wealth — 事业财运
验证事业财运类问题。

| 用例 | 场景 |
|------|------|
| career-direction | 适合做什么行业 |
| wealth-trend | 财运什么时候好转 |
| career-change | 转行是否合适 |
| wealth-investment | 投资方向建议 |
| career-promotion | 升职机会 |

### quiz-ziwei — 紫微斗数专项
验证紫微斗数排盘、十二宫分析、流年等专项能力。

| 用例 | 场景 |
|------|------|
| ziwei-basic-chart | 紫微斗数排盘 |
| ziwei-career | 紫微斗数看事业财运 |
| ziwei-marriage | 紫微斗数看婚姻 |
| ziwei-liunian | 紫微斗数流年分析 |
| ziwei-followup | 排盘后追问命宫和感情 |

### edge-input — 边界输入
验证异常输入处理。

| 用例 | 场景 |
|------|------|
| edge-short-msg | 极短消息 "算" |
| edge-no-gender | 缺少性别信息 |
| edge-unrelated | 完全无关话题 |
| edge-topic-switch | 八字中途切换话题 |
| edge-english-mixed | 中英混杂输入 |

### quiz-knowledge — 知识库检索
验证 RAG 触发、古籍引用和出处标注。

| 用例 | 场景 |
|------|------|
| knowledge-marriage-ref | 追问古籍如何论婚姻 |
| knowledge-career-classic | 要求古籍引用 |
| knowledge-named-book | 指定书名引用原文 |
| knowledge-ditiansui | 经典名句解释对照 |
| knowledge-fake-book | 伪书检测——不应引用不存在的书 |

### quiz-knowledge-edge — 知识图谱专项
验证 knowledge_catalog、3次预算、GetGraph、Agentic RAG 策略。

| 用例 | 场景 |
|------|------|
| kedge-catalog-search | 排盘后调用 catalog + search，验证古籍引用 |
| kedge-specific-book | 指定《子平真诠》原文引用 |
| kedge-multi-source | 同时查两部古籍对比 |
| kedge-catalog-first | 要求先列出目录再选书检索 |
| kedge-no-fake-book | 伪书检测——不应引用不存在的书名 |

### quiz-year-event — 流年专项
验证具体年份分析能力。

| 用例 | 场景 |
|------|------|
| year-2026-specific | 具体年份运势分析 |
| year-multi-year-compare | 两个年份对比 |
| year-past-event-verify | 历史事件反推验证 |
| year-age-range | 年龄段运势分析 |
| year-this-year | 今年整体运势 |

### edge-adversarial — 对抗性输入
验证异常和恶意输入的处理。

| 用例 | 场景 |
|------|------|
| adv-contradict-birth | 自相矛盾的出生信息 |
| adv-impossible-date | 不存在的日期 (2月30日) |
| adv-prompt-injection | 指令注入攻击 |
| adv-future-date | 未来出生时间 |
| adv-garbled-text | 乱码/无意义输入 |

### edge-resilience — 容错降级
验证异常场景下的系统稳定性。

| 用例 | 场景 |
|------|------|
| resil-multi-domain | 同时要求八字+奇门 |
| resil-ziwei-domain | 紫微斗数 domain 切换 |
| resil-long-input | 超长描述性输入 |
| resil-midnight-birth | 子时出生 |
| resil-elderly-birth | 高龄命例 (1925年) |

---

## 第二节：使用方式

```bash
# 启动后端
cd "$(git rev-parse --show-toplevel)"
set -a; source .env; set +a
LISTEN_ADDR=:18080 go run ./cmd/server/ &
sleep 5
curl http://localhost:18080/api/health

# 跑单个套件
python3 testsets/suites/runner.py testsets/suites/flow-basic.jsonl http://localhost:18080

# 跑多个套件
for f in flow-basic quiz-marriage quiz-career-wealth edge-input; do
    python3 testsets/suites/runner.py testsets/suites/$f.jsonl http://localhost:18080
done
```

---

## 第三节：Skill 识别规则

当 `agent-test-suites` skill 被触发时，按以下映射选择套件：

| 改动文件 | 对应套件 |
|---------|---------|
| `internal/runtime/prompt.go` | flow-basic + quiz-marriage + quiz-career-wealth + quiz-knowledge |
| `internal/runtime/agent_route.go` | flow-basic + edge-input + edge-resilience |
| `internal/runtime/executor.go` | flow-basic |
| `internal/runtime/preflight.go` | flow-basic + edge-input |
| `internal/orchestrator/` | flow-basic |
| `internal/policy/gate.go` | edge-input + edge-adversarial |
| `internal/specialists/bazi/` | flow-basic + quiz-marriage + quiz-career-wealth + quiz-year-event |
| `internal/specialists/ziwei/` | quiz-ziwei + edge-resilience |
| `internal/tools/ziwei/` | quiz-ziwei + edge-resilience |
| `internal/mcp/` (知识库) | quiz-knowledge + quiz-knowledge-edge |
| `internal/tools/knowledge_*.go` | quiz-knowledge-edge |
| `internal/runtime/adapter.go` | quiz-knowledge-edge |
| `internal/runtime/bridge.go` (search budget) | quiz-knowledge-edge |
| `web/` (Vue 前端) | 不跑 suite |
| `reasoning/` (Python) | 不跑 suite |
| 模型切换 | Smoke + Standard + Exhaustive（全量） |

不在映射表中的文件改动 → 只跑 `go test ./...` + `go build`。

---

## 评分基准

| 场景 | 预期准确率 | 说明 |
|------|-----------|------|
| 随机基线 | 25% | 四选一瞎蒙 |
| 人类新手 | 30-35% | 学过基础但缺实战 |
| 人类从业者 | 40-45% | 有数年经验 |
| 大赛冠军 | 50% | 2024年冠军得分率 |

八字推理本质上是高难度、多解性任务，不要期待 80%+ 的准确率。

## 数据来源

- 全球算命师大赛真题 (香港青年术数家协会, 2022-2025)
- MingLi-Bench: github.com/DestinyLinker/MingLi-Bench
- 基础知识数据来自 FatePath 社区整理
