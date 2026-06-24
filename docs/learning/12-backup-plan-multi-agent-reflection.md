# 命理大师 v2 备用方案：多 Agent + 反思模式

**日期：** 2026-06-11  
**状态：** 备用设计，暂不默认实施  
**定位：** 这是给未来 `v2 / 实验线` 准备的方案，不替代当前 `v1` Go 单主线  
**目标：** 在不破坏现有执行层稳定性的前提下，引入多 Agent 分工和受控反思，提高复杂问答的路由质量与答案稳定性

---

## 0. 一句话结论

如果你想做多 Agent + 反思模式，**最好的切入点不是重写整个后端，而是新增一个独立的 reasoning 层**：

- Go 继续负责：session、tool、SSE、前端组件、降级
- reasoning service 负责：问题理解、术数路由、回答规划、反思检查、最终文案

换句话说：

> 执行层保持单主线，推理层升级为多 Agent。

这是本项目里最稳、也最有展示价值的做法。

---

## 1. 为什么要单独写这份备用方案

当前仓库已有明确结论：

- `v1` 先做 Go / Eino 单主线
- `v2` 再做 LangGraph 对照版
- 多 Agent 和重型 reflection 不应该直接塞进 `v1`

但你的真实目标并不只是“先把产品跑起来”，你还想展示：

- 更像现代 Agent 系统的结构
- 复杂问题下的分工与协作
- 回答前后的自检与修正

因此这份文档不是反对多 Agent，而是提供一个**不伤害当前主线**的升级路径。

---

## 2. 这份方案适合什么时候启用

只有当以下至少满足 3 条时，才建议启用：

1. `v1` 已稳定支持：排盘、追问、知识检索、SSE、前端展示
2. 八字、神煞、奇门至少已有 2 类能力接入
3. 你已经能明确感受到“单 prompt 直接回答”在复杂场景下容易跑偏
4. 你希望把项目往“推理系统”而不是“工具聚合器”方向展示
5. 你接受多一次服务调用和更高延迟

如果以上条件不满足，先不要启用。

---

## 3. 这份方案解决什么问题

## 3.1 当前单主线最容易遇到的问题

### 问题 A：用户问题的真正意图不稳定

例如：

- “这个月适合跳槽吗”
- “我最近感情是不是会有变化”
- “什么时候更容易谈成合作”

这类问题不只是“继续回答”，而是要先判断：

- 是不是时机类问题
- 要不要起奇门
- 八字和奇门的权重怎么分

### 问题 B：回答容易偏一头

典型偏差：

- 神煞被讲得过重
- 用神被误当成流年结论
- 奇门只给象，不回答用户问题
- 只讲背景，不给时机结论

### 问题 C：单次生成很难做显式质量控制

虽然 prompt 里可以加入元认知规则，但它仍然是一次生成，很难清楚拆分：

- 规划
- 执行
- 审核
- 修订

---

## 4. 设计原则

这份方案必须遵守以下原则。

## 4.1 原则一：执行层不重写

不要把现有 Go 后端改成“多 Agent 控制器”。

Go 层继续负责：

- session state
- tool registry
- SSE
- 前端 component 事件
- 降级与错误传播

## 4.2 原则二：多 Agent 只放在推理层

多 Agent 只负责：

- 分析问题
- 决定用哪些术数能力
- 审核回答质量
- 修正文案

不直接持有会话总状态。

## 4.3 原则三：反思必须有预算上限

不允许无限循环反思。

必须限制：

- 最多 1 次 critic 审核
- 最多 1 次 rewrite
- 总延迟超时后直接降级为单轮回答

## 4.4 原则四：可完全降级

如果 reasoning service 不可用，Go 仍然可以：

- 直接走旧版 prompt
- 照常调用 `bazi_calc / knowledge_search / qimen`
- 返回基本可用结果

---

## 5. 推荐总体架构

```text
Vue 3
  │
  │ POST /api/chat
  ▼
Gin / Go (:8080)
  ├── Session Store
  ├── Tool Registry
  │    ├── bazi_calc
  │    ├── knowledge_search
  │    ├── qimen_dunjia
  │    └── future tools...
  ├── SSE Writer
  ├── Execution Facade
  └── HTTP Client → Reasoning Service (:8000)
                         │
                         ├── planner agent
                         ├── domain router agent
                         ├── answer composer agent
                         ├── critic agent
                         └── final rewriter agent
```

### 说明

- Go 仍是用户入口
- Python / LangGraph 只做 reasoning
- Tool 的真实执行仍然回到 Go

这点非常关键：

> reasoning service 决策“要做什么”，Go 决策“怎么执行以及怎么对外输出”。

---

## 6. 推荐 Agent 分工

不要一上来设计 7-8 个 agent。第一版只建议 4 个。

## 6.1 Planner Agent

职责：

- 读用户问题
- 判断问题类型
- 判断是否需要八字背景
- 判断是否需要奇门
- 输出执行计划

输入：

- `profile`
- `bazi_result`（如果已有）
- `last_user_question`
- 当前用户消息

输出示例：

```json
{
  "mode": "followup",
  "question_type": "timing_career",
  "need_bazi_context": true,
  "need_knowledge": true,
  "need_qimen": true,
  "need_reflection": true,
  "response_style": "direct_but_gentle"
}
```

## 6.2 Domain Router Agent

职责：

- 根据 planner 输出，补充分领域策略
- 决定这轮主要按哪条分析路径组织

例如：

- 事业时机
- 婚恋时机
- 健康风险
- 普通格局问答

这个 agent 不执行工具，只产出“分析框架”。

输出示例：

```json
{
  "primary_domain": "career",
  "analysis_order": ["bazi_background", "qimen_timing", "knowledge_support", "practical_advice"],
  "must_answer": ["是否适合跳槽", "风险点", "更合适的行动窗口"],
  "forbidden_patterns": ["空泛鼓励", "只讲神煞", "跳过结论"]
}
```

## 6.3 Answer Composer Agent

职责：

- 消费 Go 执行后的结果
- 生成“第一版答案”

输入：

- `profile`
- `bazi_result`
- `qimen_result`（可选）
- `passages`
- `planner_output`
- `domain_output`

输出：

- 面向 critic 的草稿答案

## 6.4 Critic Agent

职责：

- 检查答案有没有违背数据
- 检查有没有答非所问
- 检查有没有把辅助信号说成主结论

输出示例：

```json
{
  "approved": false,
  "issues": [
    "没有明确回答是否适合跳槽",
    "把神煞桃花错误用于事业判断",
    "没有利用 qimen_result 的时机信息"
  ],
  "rewrite_brief": "先给结论，再补八字背景，最后给时机建议"
}
```

## 6.5 Final Rewriter Agent

职责：

- 只有 critic 不通过时才触发
- 根据 critic 的修订意见输出最终版

注意：

- 它不是重新自由发挥
- 它只能在已有结果基础上修文案

---

## 7. 为什么不推荐一上来做“辩论式多 Agent”

不建议第一版做：

- 正方 agent / 反方 agent / 裁判 agent
- 反复多轮 debate
- 自反思直到满意为止

原因：

1. 延迟高
2. 成本高
3. 结果不稳定
4. 对当前命理问答场景收益不一定高

这个项目更适合的是：

> 规划型多 Agent + 有上限的 critic/rewrite

而不是“辩论秀”。

---

## 8. Reflection 模式怎么放

这是整份方案里最需要克制的部分。

## 8.1 推荐用“选择性反思”，不要“全量反思”

只有在以下场景之一时，才启用 critic：

- 涉及婚姻破裂、重大疾病、死亡、官非等高风险话题
- 涉及“时机判断”的问题，且已经动用了奇门
- planner 判断答案结构复杂，容易跑偏
- 需要引用多来源知识时

普通问题不启用。

## 8.2 Reflection 的最小流程

```text
compose draft
  ↓
critic review
  ├── approved      → final answer
  └── not approved  → rewrite once → final answer
```

只允许一次 rewrite。

## 8.3 Critic 的检查清单

critic 不应该泛泛地说“可以更好”，必须按固定清单审：

1. 是否回答了用户明确问题
2. 是否使用了错误的数据字段
3. 是否把辅助信号当主结论
4. 是否遗漏了已执行工具的关键信息
5. 是否出现空话、套话、模糊结论

---

## 9. 与现有 Go 的协作方式

这是最关键的实现边界。

## 9.1 Go 侧新增的不是“新大脑”，而是“reasoning adapter”

建议新增抽象：

- `internal/reasoning/client.go`
- `internal/reasoning/types.go`

Go 向 reasoning service 发送：

```json
{
  "session_id": "xxx",
  "profile": {...},
  "bazi_result": {...},
  "user_question": "这个月适合跳槽吗",
  "available_capabilities": ["bazi", "knowledge", "qimen"]
}
```

reasoning service 返回：

```json
{
  "plan": {
    "need_knowledge": true,
    "need_qimen": true,
    "need_reflection": true
  },
  "domain": {
    "primary_domain": "career",
    "must_answer": ["是否适合跳槽", "风险点", "窗口期"]
  }
}
```

然后 Go 再实际执行工具。

## 9.2 工具执行顺序仍由 Go 控制

例如：

1. `planner` 说需要 `qimen`
2. Go 调 `qimen_dunjia`
3. `planner` 说需要 knowledge
4. Go 调 `knowledge_search`
5. Go 把结果回传 reasoning service 做 compose / critic / rewrite
6. Go 再把最终文本流式发给前端

这样能保留：

- 工具统一出口
- SSE 一致性
- 降级能力

---

## 10. LangGraph 里的状态设计

如果这个备用方案真的落地到 LangGraph，建议状态尽量小，不要把整轮细节全塞进去。

推荐 State：

```python
class ReasoningState(TypedDict):
    session_id: str
    user_question: str
    profile: dict
    bazi_result: dict | None
    qimen_result: dict | None
    passages: list
    planner_output: dict | None
    domain_output: dict | None
    draft_answer: str | None
    critic_output: dict | None
    final_answer: str | None
    need_knowledge: bool
    need_qimen: bool
    need_reflection: bool
```

### 不要放进 State 的东西

- 原始 SSE chunks
- 前端组件树
- 大量历史 message 全文
- 第三方库原始对象

---

## 11. 推荐图结构

第一版只建议做很小的图：

```text
START
  ↓
planner
  ↓
domain_router
  ↓
return plan to Go
  ↓
Go executes tools
  ↓
composer
  ↓
need_reflection?
  ├── no  → final
  └── yes → critic
              ├── approved     → final
              └── rewrite_once → final
```

重点：

- planner / domain_router 在工具执行前
- composer / critic / rewrite 在工具执行后
- 不做多轮执行反馈回 planner 的闭环

---

## 12. 备用方案的实施顺序

如果未来真做，建议分 4 步，而不是一步到位。

## 第一步：先把 reasoning service 跑起来，但只做 planner

目标：

- Go 把分类与路由外包给 reasoning service
- 仍然由 Go 完成回答

这样可以先验证：

- 多一层服务调用值不值
- planner 是否比 Go 手写规则更清晰

## 第二步：让 reasoning service 负责 compose

目标：

- Go 仍执行工具
- 但最终答案由 reasoning service 生成

## 第三步：加选择性 critic

目标：

- 只在高风险 / 时机问题触发 critic
- 不做全量反思

## 第四步：再考虑 rewrite

目标：

- 只有 critic 拒绝时才 rewrite 一次

这四步里，前三步就已经足够展示“多 Agent + reflection”的工程化思路了。

---

## 13. 文件级建议

如果未来要落地，推荐以下文件布局。

### Go 侧新增

- `internal/reasoning/client.go`
- `internal/reasoning/types.go`
- `internal/orchestrator/reasoning.go`

### Python / LangGraph 侧新增

- `reasoning/server.py`
- `reasoning/graph.py`
- `reasoning/state.py`
- `reasoning/agents/planner.py`
- `reasoning/agents/domain_router.py`
- `reasoning/agents/composer.py`
- `reasoning/agents/critic.py`
- `reasoning/agents/rewriter.py`

### 文档补充

- `docs/v2/tech-reasoning.md`
- `docs/v2/implementation/m2-langgraph.md`
- `docs/v2/acceptance-criteria.md`

---

## 14. 验收标准（备用方案版）

只有全部满足，才算这个备用方案“值得继续”。

1. reasoning service 挂掉时，Go 能自动降级回旧链路
2. planner 对“时机类问题”的识别比旧关键词规则更稳
3. 启用 critic 后，复杂问题答案更少跑偏
4. 平均延迟增长可接受
5. 代码结构比“把所有逻辑塞进 Go Orchestrator”更清晰

如果做完后发现：

- 延迟显著变差
- 答案稳定性没提升
- 调试复杂度明显变高

那就应当停止，不要强推。

---

## 15. 风险与代价

## 风险 1：系统复杂度明显上升

你会同时维护：

- Go 执行层
- Python reasoning 层
- 两层协议

## 风险 2：反思收益可能低于预期

强模型不一定因为 critic 就显著变好，可能只是更慢。

## 风险 3：调试链变长

一个问题出错时，你要排查：

- planner 错了
- tool 结果错了
- composer 没用上数据
- critic 误杀了正确答案

## 风险 4：演示价值和真实收益可能不一致

多 Agent 看起来很高级，但如果产品收益不明显，就容易变成“为了像 Agent 而做 Agent”。

---

## 16. 最终建议

如果你只是想把项目做成“更像现代 Agent 工程”的样子，这份方案值得保留，而且很适合做 `v2 / backup architecture`。

但默认实施顺序仍然建议是：

1. `v1.2` 把神煞、奇门这些能力接入做实
2. `v2` 先做 LangGraph planner / routing
3. 在此基础上，**只加受控的 critic/rewrite**
4. 最后再决定要不要升级成更完整的多 Agent 结构

一句话收尾：

> 多 Agent + 反思模式适合作为“推理层升级方案”，不适合作为当前主线的第一落点。

