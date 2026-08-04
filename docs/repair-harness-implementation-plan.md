# 全局 Repair Harness 实施方案

> 目标：先实现功能，再逐步抽象。本文是执行计划，不表示功能已全部落地。

## 结论

全局 Repair Harness 是运行时的“验证-修复-降级”控制层。

它不替代 Eino，也不替代领域 validator。Eino 负责模型调用重试、Graph 分支、状态和 tracing；Repair Harness 负责把业务合同失败分类、限次修复、沉淀学习提示，并在失败耗尽后走降级或硬错误。

目标控制流：

~~~text
模型/runner 输出
-> 代码校验
-> 错误分类
-> 判断 retry budget
-> 带结构化错误反馈修复一次
-> 再校验
-> 仍失败则 fallback/facts-only/hard error
-> trace + eval 样本沉淀
~~~

第一批接入八字，因为最近的 BAZI_STATIC_PROJECTION_FAILED 暴露了当前主链缺少统一 repair 回环；但设计必须能复用于奇门、紫微和 final guard。

## 当前问题

当前项目已有分散机制，但没有全局闭环：

- 旧 runStaticSynthesisWithFeedback 仍在，但服务旧 static synthesis 路径；当前八字主链已迁到 canonical graph。
- 当前八字内部 graph 是 canonical_synthesis -> projection -> static_validation -> recovery_decision，静态校验失败后没有回到模型修复。
- 外层 orchestrationGraph 目前是 preflight -> prefill -> agent -> final_guard，没有统一的 validate_output -> repair_decision -> fallback 节点。
- validator 错误如果没有变成机器可读分类，recovery_decision 会把它当 unknown，导致内部错误码暴露给 SSE。

所以要修的是“统一分类 + 有限 repair + 降级兜底 + 学习提示”，不是继续堆词表或扩大自然语言 sanitizer。

## 非目标

- 不做无限 self-reflection。
- 不让模型修排盘、四柱、大运、流年、十神等确定性事实。
- 不把完整 prompt、完整 trace 或用户隐私塞回模型。
- 不引入 LangGraph、Instructor、Guardrails、Temporal 等外部依赖。
- 不一次性重构所有领域 runner。
- 不让 specialist 拥有最终答复权或绕过 Manager-owned runtime。

## 分层职责

| 层 | owner | 职责 | 不负责 |
|---|---|---|---|
| Eino ModelRetry | Eino / model wrapper | 429、5xx、timeout、空输出等模型调用级重试 | 业务合同判断 |
| Runtime RepairPolicy | runtime | 错误分类、retry budget、repair/fallback/hard error 决策 | 命理裁断 |
| Domain Adapter | 各领域 | validate、feedback、repair、fallback | 全局预算与 SSE 错误策略 |
| Final Guard | runtime | 最终泄露、缺资产、输出边界拦截 | 重做领域推理 |
| Eval / Trace | eval + tracing | 失败样本沉淀、repair chain 可观测 | 在线业务修复 |

## 执行组织（Subagent）

实现时主 agent 不直接写业务代码，只做协调、审查和状态更新。

| 角色 | 负责范围 | 交付物 | 禁止 |
|---|---|---|---|
| 主 agent | 读事实、冻结本轮目标、派发 subagent、审查 diff、跑验证、更新 PROGRESS.md | 阶段合并结论和下一步任务 | 直接写业务实现、并发修改同一接口 |
| runtime-contract-subagent | Phase 0：全局类型、分类、预算、trace、八字错误映射 | 最小 runtime diff + runtime 测试结果 | 改八字内图控制流 |
| model-retry-subagent | Phase 1：模型调用级 transient retry | retry 单测和 trace 证据 | 处理业务合同错误 |
| bazi-repair-subagent | Phase 3：八字 canonical repair 回环 | 八字内图 diff + BAZI_STATIC_PROJECTION_FAILED 回归 | 改 renderer 兜底语义 |
| learning-hint-subagent | Phase 4：短 learning hint 注入 | hint 匹配测试和 trace 计数 | 自动沉淀线上 trace 到 prompt |
| eval-repair-subagent | Phase 6：runtime-repair-v1 与 make eval-repair | eval dataset、runner/Makefile diff、报告样例 | 扩写领域业务逻辑 |
| reviewer-subagent | 每阶段只读审查 | 风险清单：无限重试、裸错误、边界越权、trace 泄露、缺 eval | 写代码 |

执行规则：

- 一次只派一个实现 subagent；实现完成并通过 reviewer-subagent 后再派下一个。
- subagent 只能改自己阶段的最小文件集合；跨阶段发现只记录风险，不顺手实现。
- 主 agent 合并前必须读 diff、跑该阶段验证，并确认没有绕过 Manager-owned runtime、ExecutionPlan、Prefill 或 final guard。
- reviewer-subagent 发现边界问题时，先让原实现 subagent 返工；不要由主 agent 叠补丁掩盖问题。

## 全局错误分类

新增全局 RepairClass，领域错误只映射到这些通用类型。

| Class | 示例 | 默认动作 |
|---|---|---|
| transport_transient | 429、5xx、timeout | Eino/model 层 retry |
| transport_fatal | 400、401、402 | 不 retry，返回配置错误或降级 |
| parse_error | JSON 截断、非法 JSON | 同节点 retry 1 次 |
| schema_error | 缺字段、枚举非法 | repair 1 次 |
| projection_mismatch | 字段投影不满足合同 | repair 1 次 |
| evidence_overclaim | 证据不足却强裁断 | repair 1 次，失败 facts-only |
| domain_unauthorized | 未成年人输出事业/财务等越权领域 | 优先 fallback/facts-only |
| fact_conflict | 与确定性工具事实冲突 | 不让模型修，hard error 或 facts-only |
| method_contract | 方法论边界冲突 | 不让模型修，hard error |
| guardrail_blocked | final text 泄露 prompt/trace/tool 参数 | final text repair 1 次，失败 hard error |

## Retry Budget

第一版固定预算，不先配置化：

| Budget | 数值 | 说明 |
|---|---:|---|
| MaxTurnRepairAttempts | 2 | 单轮业务 repair 总次数 |
| MaxNodeRepairAttempts | 1 | 同一 stage/field 最多 repair 一次 |
| TransportMaxAttempts | 2 | 429/5xx/timeout 的模型调用级 retry |
| JSONParseMaxAttempts | 1 | JSON parse/schema 修复次数 |

硬规则：

- 业务 repair 和 transport retry 分开计数。
- 400、401、402 不消耗 repair budget，直接终止或降级。
- repair 后必须重新跑原 validator。
- budget 耗尽后必须进入 fallback 或 hard error，不能继续循环。
- fact_conflict、method_contract 默认不 repair。

## 核心类型

新增 backend/internal/runtime/repair_types.go。

~~~go
type RepairClass string

const (
    RepairTransportTransient RepairClass = "transport_transient"
    RepairTransportFatal     RepairClass = "transport_fatal"
    RepairParseError         RepairClass = "parse_error"
    RepairSchemaError        RepairClass = "schema_error"
    RepairProjectionMismatch RepairClass = "projection_mismatch"
    RepairEvidenceOverclaim  RepairClass = "evidence_overclaim"
    RepairDomainUnauthorized RepairClass = "domain_unauthorized"
    RepairFactConflict       RepairClass = "fact_conflict"
    RepairMethodContract     RepairClass = "method_contract"
    RepairGuardrailBlocked   RepairClass = "guardrail_blocked"
)

type RepairAction string

const (
    RepairActionAccept     RepairAction = "accept"
    RepairActionRetry      RepairAction = "retry"
    RepairActionRepairNode RepairAction = "repair_node"
    RepairActionFallback   RepairAction = "fallback"
    RepairActionHardError  RepairAction = "hard_error"
)

type RepairFailure struct {
    Domain string
    Stage  string
    Class  RepairClass
    Field  string

    Code    string
    Message string
    Excerpt string

    Retryable  bool
    Repairable bool
    Fallback   string
    Cause      error
}

type RepairAttempt struct {
    Domain   string
    Stage    string
    Class    RepairClass
    Field    string
    Attempt  int
    Action   RepairAction
    Feedback map[string]any
}

type RepairState struct {
    Attempts              []RepairAttempt
    MaxTurnRepairAttempts int
    MaxNodeRepairAttempts int
}
~~~

## 领域接入接口

新增 backend/internal/runtime/repair_domain.go。

~~~go
type RepairableDomain interface {
    Domain() string

    Validate(ctx context.Context, value any) error
    Classify(ctx context.Context, stage string, err error) RepairFailure

    BuildRepairFeedback(ctx context.Context, failure RepairFailure, value any) map[string]any
    Repair(ctx context.Context, input RepairInput) (any, error)
    Fallback(ctx context.Context, failure RepairFailure, value any) (any, error)
}
~~~

第一版只实现八字 adapter。奇门、紫微先只接 schema/empty output/final guard 类通用校验，等八字试点稳定再补领域 repair。

## Repair Feedback

新增 backend/internal/runtime/repair_feedback.go。

反馈必须短、结构化、字段级。不要回灌完整 trace 或完整候选文本。

~~~json
{
  "retry_attempt": 1,
  "failed_stage": "static_projection",
  "failure_class": "projection_mismatch",
  "field": "static.tiaohou_anchor",
  "reason": "调候证据已覆盖，但调候锚点没有形成明确裁断",
  "allowed_fix": [
    "只修改 canonical.tiaohou.verdict",
    "必要时修改 canonical.tiaohou.boundary"
  ],
  "must_preserve": [
    "四柱",
    "日主",
    "月令",
    "大运",
    "流年",
    "main_axis",
    "strength",
    "pattern"
  ],
  "forbidden": [
    "不得改排盘事实",
    "不得新增具体现实应事",
    "不得把证据不足写成强裁断"
  ],
  "learning_hints": []
}
~~~

## Learning Hint

新增 backend/internal/runtime/repair_learning.go。

retry 只修当前请求；Learning Hint 负责提高下一次模型正确率。

~~~go
type RepairLearningHint struct {
    Domain string
    Stage  string
    Class  RepairClass
    Field  string

    Pattern     string
    Instruction string
    BadExample  string
    GoodExample string
    AppliesWhen []string
}
~~~

规则：

- 按 domain + stage + class + field 匹配。
- 每个字段最多注入 3 条 hint。
- hint 必须是短指令或短示例，不是长历史。
- 线上 trace 不自动污染 prompt；高频 hint 需要通过代码审查固化。

八字调候初始 hint：

~~~json
{
  "domain": "bazi",
  "stage": "static_projection",
  "class": "projection_mismatch",
  "field": "static.tiaohou_anchor",
  "instruction": "调候证据覆盖时必须给出明确裁断词，如调候不足、调候受限、调候得力、调候受损。",
  "bad_example": "调候上喜水润局，但水星不透，调候力度受限。",
  "good_example": "调候受限：秋月戊土需火暖、水润；原局水不透，调候以水润为要但力度不足。"
}
~~~

## Eino 接入方式

Eino 提供底座，不负责业务合同：

- ModelRetryConfig：处理 transient error、空输出。
- ModelFailoverConfig：后续可接备用模型，第一版不做。
- Graph Branch：承载 repair_decision、fallback 等显式节点。
- State / callbacks：保存 attempt、trace repair chain。

第一版优先：

1. 在模型/agent 构建处接基础 ModelRetryConfig。
2. 保持业务 repair 在 runtime policy 中，不塞进 Eino callback。
3. 后续再把当前 wrapper 迁到外层 orchestrationGraph 显式节点。

## 八字第一批接入

八字不再维护一套孤立策略，而是桥接到全局 RepairFailure。

新增：

~~~go
func repairFailureFromBaziContract(stage string, err error) (RepairFailure, bool)
~~~

映射：

| BaZi class | Global class |
|---|---|
| evidence_overclaim | RepairEvidenceOverclaim |
| domain_unauthorized | RepairDomainUnauthorized |
| projection_mismatch | RepairProjectionMismatch |
| fact_conflict | RepairFactConflict |
| method_contract | RepairMethodContract |

改动点：

- bazi_contract_failure.go：输出可转全局 RepairFailure。
- bazi_internal_graph.go：加入 repair_decision / canonical_repair。
- bazi_canonical_synthesis.go：新增 runCanonicalSynthesisRepair，复用 canonical payload，只追加 validation_feedback。

目标八字控制流：

~~~text
canonical_synthesis
-> projection
-> static_validation
-> repair_decision
-> canonical_repair
-> projection
-> static_validation
-> dynamic_validation/render 或 recovery_decision
~~~

## 外层全局接入

短期先包在 agentNode 内部：

~~~text
domain runner -> repair harness -> final_guard
~~~

中期再拆成正式 Eino graph 节点：

~~~text
agent -> validate_output -> repair_decision -> repair_agent/fallback -> final_guard
~~~

原因：当前 agentNode 处理 streaming 和八字内部 graph 分派，第一版直接大改外层 graph 容易扩大影响。

## Trace 设计

新增 backend/internal/runtime/repair_trace.go。

必须写入：

~~~text
repair.domain
repair.stage
repair.class
repair.field
repair.attempt
repair.max_attempts
repair.action
repair.feedback_keys
repair.learning_hint_count
repair.exhausted
repair.final_action
~~~

Run Inspector 展示摘要：

~~~text
修复链：static_projection -> repair_node(attempt=1) -> validate_ok
~~~

普通 SSE 不发送完整 feedback。

## Eval Ratchet

新增 eval/datasets/runtime-repair-v1.json 和 make eval-repair。

样本格式：

~~~json
{
  "case_id": "repair_bazi_static_tiaohou_anchor_v1",
  "input": "1991年10月5日12点40分 南京 男",
  "expected": {
    "no_sse_error": true,
    "forbidden_error_codes": ["BAZI_STATIC_PROJECTION_FAILED"],
    "allowed_final_actions": ["repair_success", "facts_only"],
    "trace_has": [
      "repair.stage=static_projection",
      "repair.class=projection_mismatch"
    ]
  }
}
~~~

每个线上失败沉淀四字段：

~~~text
conjectured: 原来以为哪个机制能拦住
refuted_by: 哪个 trace/日志证明不成立
learned: 新理解
criterion_now: 新增什么断言或 eval
~~~

## 分阶段实施

### Phase 0：只建全局类型和分类

目标：不改变行为，先让错误可分类、可观测。

改动：

- 新增 repair_types.go
- 新增 repair_policy.go
- 新增 repair_trace.go
- 八字 contract failure 映射到全局 RepairFailure

验证：

~~~bash
go test ./backend/internal/runtime -count=1
~~~

验收：

- 不改变当前输出行为。
- trace 能看到 repair.class / repair.stage。

### Phase 1：模型调用级 retry

目标：Eino 层处理 transient error。

改动：

- 在 agent/model 构建处加 ModelRetryConfig
- 429、5xx、timeout、空输出可 retry
- 400、401、402 不 retry

验收：

- mock 429 最多 retry 2 次。
- mock 402 不 retry。
- retry 次数写入 trace。

### Phase 2：JSON / schema repair wrapper

目标：结构化输出坏了能带错误重试一次。

改动：

- 抽 runRepairableJSON[T]
- 八字 canonical synthesis 先接入
- parse/schema 错误注入 validation_feedback

验收：

- JSON 截断 retry 1 次成功。
- retry 仍失败进入 fallback 或 RuntimeFailure。
- 不无限打模型。

### Phase 3：业务合同 repair

目标：解决 BAZI_STATIC_PROJECTION_FAILED 同类问题。

改动：

- 八字内图新增 repair_decision
- 八字内图新增 canonical_repair
- static_validation 失败后先问 RepairPolicy
- 可修复则回 canonical_repair -> projection -> validation
- 不可修复或预算耗尽进 recovery_decision

验收：

- static.tiaohou_anchor 第一次失败，repair 成功。
- repair 失败，进入 facts-only，无 SSE error。
- fact_conflict 不允许 repair。

### Phase 4：Learning Hint

目标：提高下一次模型正确率。

改动：

- 新增 repair_learning.go
- 内置少量高频 hint
- feedback builder 自动注入匹配 hint
- trace 记录 hint 命中数

验收：

- 调候错误 feedback 包含对应 good/bad example。
- 非匹配错误不注入无关 hint。

### Phase 5：外层全局化

目标：八字以外也能接。

改动：

- 新增 runWithRepairHarness
- specialist result 增加可选 validate/fallback hook
- 奇门/紫微先接 schema/artifact/empty output/final guard

验收：

- 非八字空输出可 repair/fallback，不裸 error。
- 紫微/奇门缺 artifact 走统一 RuntimeFailure 或 fallback。

### Phase 6：Eval 与报告

目标：防回归。

改动：

- 新增 runtime-repair-v1 dataset
- 新增 make eval-repair
- Run Inspector 展示 repair chain

验证：

~~~bash
go test ./backend/... -count=1
go build ./backend/cmd/server/
make eval-repair
~~~

## 第一版文件清单

| 文件 | 动作 |
|---|---|
| backend/internal/runtime/repair_types.go | 新增全局类型 |
| backend/internal/runtime/repair_policy.go | 新增分类和决策 |
| backend/internal/runtime/repair_budget.go | 新增预算控制 |
| backend/internal/runtime/repair_feedback.go | 新增结构化反馈 |
| backend/internal/runtime/repair_learning.go | 新增学习提示 |
| backend/internal/runtime/repair_trace.go | 新增 trace 投影 |
| backend/internal/runtime/bazi_contract_failure.go | 桥接全局错误分类 |
| backend/internal/runtime/bazi_internal_graph.go | 加 repair branch |
| backend/internal/runtime/bazi_canonical_synthesis.go | 加 canonical repair |
| backend/internal/runtime/orchestration_graph.go | 后续接外层 harness |
| eval/datasets/runtime-repair-v1.json | 新增回归集 |
| Makefile | 新增 eval-repair |

## 硬验收标准

- BAZI_STATIC_PROJECTION_FAILED 不再裸出到 SSE。
- 可修复错误最多 repair 1 次。
- 同轮业务 repair 最多 2 次。
- 402、401、400 不 retry。
- fact_conflict 不让模型修。
- repair 失败必须 fallback 或 hard error，不能继续循环。
- 每次 repair trace 可查。
- 每个线上失败能转 eval case。
- go test ./backend/... -count=1 通过。
- go build ./backend/cmd/server/ 通过。
- make eval-repair 通过。

## 推荐执行顺序

第一轮按串行 subagent gate 执行：

1. 主 agent 冻结 Phase 0 目标 -> runtime-contract-subagent 实现 -> reviewer-subagent 审查 -> 主 agent 跑 runtime 测试。
2. 主 agent 冻结 Phase 1 目标 -> model-retry-subagent 实现 -> reviewer-subagent 审查 -> 主 agent 验证 429 retry 与 400/401/402 不 retry。
3. 主 agent 冻结 Phase 3 目标 -> bazi-repair-subagent 实现 -> reviewer-subagent 审查 -> 主 agent 回放 BAZI_STATIC_PROJECTION_FAILED 样例。
4. 主 agent 冻结 Phase 4 目标 -> learning-hint-subagent 实现 -> reviewer-subagent 审查 -> 主 agent 验证 hint 命中和非匹配不注入。
5. 主 agent 冻结 Phase 6 目标 -> eval-repair-subagent 实现 -> reviewer-subagent 审查 -> 主 agent 跑 go test、go build、make eval-repair。

暂缓：

- Phase 5 外层全局化等八字试点稳定后再做。
- Model failover 暂缓，先完成 retry/fallback 基础闭环。

## 持续执行目标提示词

把下面提示词设置为目标，让后续 AI 持续执行：

~~~text
目标：在 /home/huang/workspace/suanming-agent 实现 docs/repair-harness-implementation-plan.md 定义的全局 Repair Harness。主 agent 只负责读事实、冻结计划、派发 subagent、审查 diff、合并结果和更新状态，不直接写业务代码；实际实现由 subagent 分阶段完成。

执行原则：
1. 先读 PROGRESS.md、docs/architecture.md、docs/repair-harness-implementation-plan.md，再改代码。
2. 保持 Manager-owned runtime 边界；不得让 specialist 拥有最终答复权，不得绕过 ExecutionPlan、Prefill、final guard。
3. 先做 Phase 0、Phase 1、Phase 3、Phase 4、Phase 6；Phase 5 外层全局化等八字试点稳定后再做。
4. 不引入 LangGraph、Instructor、Guardrails、Temporal 等新依赖；使用 Eino 现有 ModelRetryConfig、Graph Branch、State、Callbacks 做底座。
5. repair 是有限重试：单 stage/field 最多 1 次，同轮业务 repair 最多 2 次；400/401/402 不 retry；fact_conflict 和 method_contract 不让模型修。
6. 所有 validator 错误必须变成机器可读 RepairFailure 或领域可映射错误，不能用裸 fmt.Errorf 让恢复节点无法分类。
7. 重试反馈必须是字段级结构化 feedback，包含 failed_stage、failure_class、field、reason、allowed_fix、must_preserve、forbidden、learning_hints；不得回灌完整 trace、完整 prompt 或用户隐私。
8. Learning Hint 只用短指令和短示例提升下一次正确率；不要把线上 trace 自动污染 prompt。
9. 每个生产失败必须沉淀 eval case；新增 runtime-repair-v1 和 make eval-repair。
10. 每个 subagent 只改自己负责的最小文件集合，交付 diff 摘要、测试结果和遗留风险；主 agent 审查后再允许下一阶段继续。
11. 每阶段完成后运行最小测试；最终必须通过 go test ./backend/... -count=1、go build ./backend/cmd/server/、make eval-repair。若修改影响启动链路，执行 make backend-restart 并用真实 /api/chat 回放最近失败输入。

主 agent 工作方式：
- 不直接写业务实现代码，除非只是修正文档、解决合并冲突或做极小 glue patch。
- 先把本轮 phase 拆成一个可验证任务，再派给单个 subagent；不要同时派多个 subagent 修改同一目录或同一接口。
- 每个 subagent 返回后，主 agent 必须读 diff、跑对应验证、检查是否破坏 runtime 边界，再决定是否合并下一步。
- 如果 subagent 的实现偏离本文档，主 agent 必须要求返工或自己回滚该 diff，不得继续叠加补丁。

建议 subagent 分工：
- runtime-contract-subagent：负责 Phase 0，新增 repair_types / repair_policy / repair_budget / repair_trace，并把八字 contract failure 映射到全局 RepairFailure。
- model-retry-subagent：负责 Phase 1，只在模型构建或 wrapper 层接 transient retry；验证 429/5xx/timeout 会 retry，400/401/402 不 retry。
- bazi-repair-subagent：负责 Phase 3，只改八字内图和 canonical repair 回环；验证 static.tiaohou_anchor repair、facts-only 降级和 fact_conflict 不 repair。
- learning-hint-subagent：负责 Phase 4，新增短 learning hint 注入和 trace 计数；不得把线上 trace 自动写进 prompt。
- eval-repair-subagent：负责 Phase 6，新增 runtime-repair-v1、make eval-repair 和最小报告检查。
- reviewer-subagent：每个阶段结束后只读审查 diff，重点查无限重试、裸错误、边界越权、trace 泄露、缺 eval。

第一批验收：
- 最近失败输入“1991年10月5日12点40分 南京 男”不得再返回 SSE error。
- 禁止裸出 BAZI_STATIC_PROJECTION_FAILED。
- static.tiaohou_anchor 可修复时走 canonical_repair；修不好时进入 facts-only。
- trace 包含 repair.domain、repair.stage、repair.class、repair.field、repair.attempt、repair.final_action。
- 402/401/400 不 retry。
- fact_conflict 不 repair。

持续执行要求：
- 每次开始先说明本轮要做哪个 Phase。
- 主 agent 每次只派发一个清晰 subagent 任务；subagent 完成并通过审查后再进入下一任务。
- 每次改动保持最小文件范围，禁止跨 phase 顺手重构。
- 更新 PROGRESS.md 当前事实，不追加流水账。
- 如果发现 docs/repair-harness-implementation-plan.md 与代码事实冲突，先修正文档或说明冲突，再继续实现。
~~~
