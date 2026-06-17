# 命理大师测试集

标准化测试与 Agent 行为回归验证。

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
│   ├── edge-input.jsonl               # [Standard] 边界输入
│   ├── quiz-knowledge.jsonl            # [Exhaustive] 知识库检索
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
| `internal/mcp/` (知识库) | quiz-knowledge |
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
