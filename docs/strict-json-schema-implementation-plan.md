# Strict JSON Schema 迁移实施方案

> 状态：已决策，尚未实施。本文是“模型结果被 Go 代码消费”的唯一迁移计划；不改变 Manager、外层 graph 或 SSE 的 owner 边界。

## 结论

1. 项目应彻底停止使用 JSON Mode。`json_object` 只保证载荷是 JSON，不保证必填字段、字段类型、枚举、未知字段和运行时事实引用。
2. 当前 `deepseek-v4-flash` 的有效 endpoint 不能作为 Strict Schema provider：2026-08-06 的真实探针向 `https://api.deepseek.com/chat/completions` 发送 `response_format.type=json_schema` 与 `json_schema.strict=true` 后，得到 HTTP 400：`This response_format type is unavailable now`。
3. 阻塞点是 provider capability，不是 Go DTO 或 Eino 核心。实现前必须选定并实测一个 Strict-Schema-capable endpoint；不得把失败静默回落为 JSON Mode。

这里的迁移对象很窄：只包括 BaZi 内部 graph 中“模型输出会反序列化为 Go DTO”的节点。普通自由文本、既有工具调用和最终 SSE 文本并不因此换模型。

## 已验证现状

### 为什么现在有两条模型链

当前不是两套业务 Agent，而是同一 Eino ADK specialist 构造器的两类模型实例：

```text
普通文本 / tool calling
BuildContainer
  -> mustNewToolCallingModel
  -> llm.NewToolCallingModel
  -> DeepSeek ChatModel
  -> AgentBuilder.model / fastModel
  -> ADK ChatModelAgent

Go 消费的 BaZi 结构化结果
BuildContainer
  -> mustNewToolCallingJSONModel
  -> llm.NewToolCallingModelWithJSON
  -> DeepSeek ChatModel(response_format=json_object)
  -> AgentBuilder.modelCreator / fastModelCreator
  -> BuildSpecialist(UseJSONMode=true)
  -> runBaziInnerAgentJSON
  -> stripMarkdownFence + json.Unmarshal
  -> Go DTO
```

历史上，DeepSeek 的 response format 是建模时配置；预建一个 `json_object` 模型，再由 `UseJSONMode` 选中它，是最少改动地复用 `BuildSpecialist` 的办法。它也是今天必须删除的旁路。

| 事实 | 当前证据 | 结论 |
|---|---|---|
| JSON Mode 工厂 | `backend/internal/llm/factory.go:62-68` | 只设置 `response_format: json_object`。 |
| 容器创建四个模型 | `backend/internal/container/container.go:193-229` | 普通 / JSON Mode 与 fast 普通 / fast JSON Mode 同时存在。 |
| Agent 分流 | `backend/internal/runtime/agent_route.go:111-120` | `UseJSONMode` 选择 JSON Mode 模型实例。 |
| 宽松解析 | `backend/internal/runtime/bazi_charter_graph.go:2591-2610` | `json.Unmarshal` 忽略未知字段，且先剥 markdown fence。 |
| 专项补丁 | `backend/internal/runtime/bazi_charter_graph.go:296-419` | `validateDynamicRelationFacts` 与 `validateDynamicFireBureauFacts` 从自然语言找未声明关系。 |

### JSON Mode 与 Strict Schema 的具体差异

现有请求等价于：

```json
{"response_format":{"type":"json_object"}}
```

它允许以下所有载荷，因为它们都是 JSON：

```json
{"summary":"日主偏弱","relations":["火局"],"anything_model_wants":"x"}
```

Strict Schema 的请求必须包含每个节点自己的合同：

```json
{
  "response_format": {
    "type": "json_schema",
    "json_schema": {
      "name": "dynamic_synthesis",
      "strict": true,
      "schema": {
        "type": "object",
        "additionalProperties": false,
        "required": ["assertions"],
        "properties": {"assertions": {"type": "array"}}
      }
    }
  }
}
```

它会拒绝未知字段、漏必填字段、错误类型和不在 enum 内的值。它**不能**证明 `relation_ref` 的 ID 真的存在；这部分仍必须由 runtime catalog 校验。因此 Strict Schema 是结构合同，事实引用合同是另一层，两者缺一不可。

### 依赖与端点能力

当前 `backend/go.mod` 锁定的相关版本：

| 组件 | 当前版本 | 可用能力 |
|---|---:|---|
| `github.com/cloudwego/eino` | `v0.9.12` | Tool 参数 JSON Schema。 |
| `github.com/eino-contrib/jsonschema` | `v1.0.3`（间接） | 可由 Go DTO 生成 JSON Schema；它不是事实校验器。 |
| `github.com/cloudwego/eino-ext/libs/acl/openai` | `v0.1.15`（间接） | 已定义 `json_schema`、`schema`、`strict` 并映射进请求。 |
| `github.com/cloudwego/eino-ext/components/model/deepseek` | `v0.1.6` | 仅暴露 `text` / `json_object`。 |

本机 Go module cache 中，OpenAI ACL 的 `chat_model.go:41-61` 已有 `json_schema` 与 `Strict bool`；DeepSeek adapter 的 `deepseek.go:41-45` 只列 `text` / `json_object`。即使给 DeepSeek adapter 补字段，当前 endpoint 也已拒绝该能力，故不是有效修复。

## 方案边界与选择

### 传输方案比较

| 方案 | 传输方式 | 需要改什么 | 影响 | 风险 | 适用条件 |
|---|---|---|---:|---|---|
| provider-native Strict Schema | `response_format.json_schema` + `strict:true` | 专用 Strict provider adapter、每节点 schema、严格 decoder | 中 | 需要独立 endpoint / 模型能力 | endpoint 实测接受请求并通过坏载荷测试。 |
| Eino output tool | 终止 tool 的 arguments 使用 JSON Schema | ADK output tool、工具结束语义、事件收集与调用次数 | 中高 | 把普通结果当 tool call，图节点与流式处理更复杂 | provider 对 tool arguments 有已验证的严格遵从，且团队愿意接受 tool 终止模型。 |
| 扩展当前 DeepSeek adapter | adapter 发 `json_schema` 字段 | fork/维护 provider adapter | 高且无收益 | 当前真实 endpoint 已返回 400 | 仅当 DeepSeek 日后正式支持且重新 probe 通过。 |

JSON Mode 不是候选方案：它没有 Schema 约束，而且“prompt 要求固定 JSON”只是把同一缺陷移到 prompt。

### 已作出的实现选择

1. 优先采用 provider-native Strict Schema；只有真正通过 capability probe 的 endpoint 才能被配置为 structured provider。
2. 普通 `LLM_*` 继续服务自由文本和现有工具调用。为结构化窄路径单独配置 `STRUCTURED_LLM_API_KEY`、`STRUCTURED_LLM_BASE_URL`、`STRUCTURED_LLM_MODEL`；缺任一项或 probe 失败即报明确 capability/configuration error。
3. 不允许 `STRUCTURED_LLM_*` 缺失时回落到 DeepSeek JSON Mode，也不允许退回 prompt JSON。
4. 不全量更换模型供应商。只有 BaZi 的结构 DTO 调用切换；DeepSeek 普通链路、Qimen/Ziwei 自由文本链路可保持不变。

## 目标架构与 owner

```text
模型输出合同 DTO
  -> provider-native Strict JSON Schema
  -> Go strict decode
  -> 通用 fact-ref / relation-ref / claim-ref 校验
  -> runtime projection / recovery policy
  -> renderer
  -> SSE
```

| 层 | 负责 | 不负责 |
|---|---|---|
| Schema | required、type、enum、数组形状、`additionalProperties:false` | 命理事实是否真实。 |
| provider adapter | 原样发送 `json_schema`、`strict:true`，报能力错误 | 把不支持的 provider 伪装成 Strict。 |
| strict decoder | 禁止 unknown field、尾随 JSON、错误类型 | 业务事实推理。 |
| runtime fact catalog | 生成事实/关系/方法 claim 的 ID、范围与显示材料 | 让模型自行增加事实。 |
| generic ref validator | ID 存在、kind 正确、period/subject/evidence 范围正确 | 通过词表猜“火局”是否合理。 |
| evaluator | 质量、覆盖、稳定性、回归判断 | 放行运行时不可信事实。 |
| renderer | 从已校验 DTO 与 catalog 组合可见文字 | 重算命理或把失败模型文本润色为可见答案。 |

### 模型可以输出、runtime 必须派生、模型禁止输出

| 类别 | 内容 |
|---|---|
| 模型输出 | 问题范围、检索主题、有限 enum 决策、`bound_claim`、引用 ID、模型解释文本、已定义的 limitation / advice boundary。 |
| runtime 派生 | fact catalog、relation catalog、claim catalog、evidence status、日期/年龄/大运投影、legacy 展示字段、source、recovery reason、field audit、contract audit、failure/retryable/recovery action。 |
| 绝不交给模型 | 确定性排盘结果、关系事实的文字和值、真实大运边界、catalog 内容和 ID 生成、用户安全策略、SSE event shape、runtime failure 分类。 |

模型可保留解释文本，但解释必须附着于 `bound_claim`，且 renderer 只展示与已通过 refs 的 claim 同行的文本。关系名称、干支组合、数值、日期、十神、来源名等事实性片段由 renderer 从 catalog 回读，不从自由解释文本取值。

## BaZi Schema 设计

### 共用原语

所有节点 Schema 默认：顶层与嵌套 object 均设置 `additionalProperties:false`；无 `omitempty` 语义混入模型合同；空数组使用 `[]`，不使用缺省字段表达“无”。

```text
fact_ref      := { "id": string }
relation_ref  := { "id": string }
claim_ref     := { "id": string }
bound_claim   := {
  "id": string,
  "kind": enum,
  "verdict": string,
  "fact_refs": [fact_ref],
  "relation_refs": [relation_ref],
  "claim_refs": [claim_ref],
  "explanation": string,
  "boundary": string
}
```

Ref object 只允许 `id`。模型不得回传 label、原始值、source、period 文本或自行归纳的关系；这些字段都可能与事实 catalog 脱节。运行时按节点输入提供可用 ID 列表和命名空间，但 Schema 本身不伪造“动态 enum”；校验器负责对每次运行生成的 catalog 做 membership 检查。

### 节点最小合同

| 节点 | required | optional | enum / 数组 | 禁止交给模型 |
|---|---|---|---|---|
| `analysis_plan` | `mode`、`retrieval_stage`、`need_dynamic`、`focus_topics`、`writer_template`、`stage_summary` | `topic_mode` | `mode`、`retrieval_stage`、`writer_template`、`topic_mode` 为闭合 enum；`focus_topics:string[]` | evidence status、工具参数、source。 |
| `evidence_plan` | `queries`、`required_topics`、`optional_topics` | `reason` | query object 只含 `topic`、`priority`；priority 为 enum | 实际 passages、citation、质量结论。 |
| `canonical_synthesis` | `claims`、`limitations`、`advice_boundary` | `reasoning_summary`、`citations` | `claims:bound_claim[]`；kind 为主轴/旺衰/调候/格局/层次/大运/流年等已定义 enum；citation 只用 passage ID | `source`、`recovery_reason`、`field_audit`、`contract_audit`、legacy static/dynamic 字段。 |
| `static_synthesis` | `claims`、`limitations` | `reasoning_summary` | 只保留 canonical projection 尚未替代的必要 claim；不得并行重述完整 canonical DTO | 同上及 runtime tier/evidence projection。 |
| `dynamic_synthesis` | `period_claims`、`liunian_claims`、`limitations` | `reasoning_summary` | 每个 period claim 必须有 `period_ref` 和 `bound_claim[]`；`outcome_domains` 为当前输入授予 enum 子集 | `dayun_path` 自由关系文本、交运日期、事实关系、recovery/runtime audit。 |
| `contract_audit` | `compliant`、`findings` | 无 | `compliant:boolean`；finding 只含 `code`、`claim_id`、`field`、`severity`；code/severity 为闭合 enum | recovery decision、retryable、source、field audit。 |

`static_synthesis` 与 `dynamic_synthesis` 是当前 legacy 桥接 DTO 的迁移对象，不是永久第二套语义。迁移后 canonical result 是唯一模型语义源；static/dynamic DTO 只保留尚未迁移的 renderer 输入，随后收缩到 runtime projection。

### 防止“丙戌火局”复发

当前模型把“火局”写进 `DayunPath` 或 `Interpretation`，runtime 只能通过自然语言正则和 `validateDynamicFireBureauFacts` 事后发现。迁移后：

```text
模型输出: relation_refs:[{"id":"relation.dayun.0.branch_combination"}]
  -> generic validator：ID 是否在本轮 catalog？kind 是否 relation？period 是否 dayun[0]？
  -> renderer：从 catalog 读取此关系的已声明文字
  -> 不存在 relation.fire_bureau 时，模型无论怎样解释都不会产生“火局”事实
```

它不是把“火局”列进禁词；它让模型没有可提交的未声明关系值。`knownBaziFactRefs`（`bazi_assertion_contract.go:692`）是现有起点，但 unknown ref 目前主要变 warning；迁移时必须改为硬失败，并把静态集合替换为本轮 catalog。

### 严格解码合同

provider 侧 Strict Schema 不能取代本地防御。所有结构化结果必须统一经同一 decoder：

1. `json.Decoder.DisallowUnknownFields()`；
2. decode 一次目标 DTO；
3. 再 decode 一次，必须得到 `io.EOF`，否则拒绝 trailing JSON；
4. 进行 DTO 级别 required/enum 复核，再进入 reference validation。

删除 `stripMarkdownFence`。Strict result 若仍含 markdown fence，就是 schema/transport failure，不能由字符串清洗悄悄修好。

## 失败、重试与降级

传输 retry 与业务 repair 是两个独立计数器，并在 trace 分别记录 `transport_attempt` 与 `schema_repair_attempt`。只有真正重新发出请求后，trace 才能写 `retryable=true` 或 attempt 递增。

| class | 是否重试 | 上限与反馈 | 模型可修什么 | 失败落点 |
|---|---|---|---|---|
| `schema_error` | 是 | schema repair 最多 1 次；只反馈缺字段/字段路径/期待类型，绝不反馈或回传事实值 | 按既定 Schema 重发结构 | synthesis 节点可丢弃候选并 facts-only；planner/audit 为 hard error；绝不 `model_partial`。 |
| `undeclared_fact_claim` | 否 | 0 次业务修复；trace 写 ID、期望 kind、scope | 不允许模型修改确定性事实或用换词掩盖冲突 | hard error；仅当 renderer 可由完整确定性结果独立生成时才由 runtime 生成 facts-only。 |
| `fact_value_mismatch` | 否 | 0 次 | 同上 | hard error；候选模型文本一律丢弃。 |
| `method_contract` | 有条件 | 最多 1 次语义 repair，反馈固定 method/profile/允许 claim ID，不反馈新事实 | 从现有 claim catalog 重新选择或收束结论 | 第二次失败为 hard error；不得靠弱化措辞把越权方法结论留下。 |
| `transport_transient` | 是 | 每次模型调用最多 2 次传输重试，指数退避；无业务反馈 | 无 | 耗尽后明确 transport error，不进入 schema repair / facts-only 伪成功。 |
| `renderer_contract` | 否 | 0 次；记录缺 catalog/ref/模板路径 | 无 | hard error 或仅输出已经通过 renderer 的独立 facts-only block；绝不重跑模型。 |

`model_partial` 只用于：Strict decode 与全部 refs 都已通过，核心裁断已存在，且缺的是独立展示字段。它不能承接 schema、事实、方法或 renderer 合同失败。

## 质量校验：不引入运行时第三方拦截器，也不造规则引擎

最终文本质量分三层，职责不能互换：

| 层 | 使用什么 | 当前行动 | 为什么 |
|---|---|---|---|
| 运行时硬合同 | Go Strict decoder、catalog/ref validator、existing recovery policy、renderer | 保留并收敛为通用引用校验 | 可判定、毫秒级、直接保护用户输出。 |
| 离线合同与回归 | 现有 `go test`、`make regression`、`make eval-repair`、BaZi datasets | 扩充 Strict Schema 与 ref fixture | 这些合同需要可重复、无模型 Judge 的断言。 |
| 语义质量 | 现有 Langfuse dataset/trace/score + `run_answer_quality_judge.py` + 人工标注校准 | 把 Judge 固定为按维度的二值 rubric，先对人工标注集校准 | 文风、解释清晰度、传统命理表达是否恰当不能由规则引擎可靠决定。 |

### GitHub / 第三方调研结论（2026-08-06）

| 项目 | 擅长什么 | 与本项目的关系 | 本阶段决定 |
|---|---|---|---|
| [Langfuse](https://github.com/langfuse/langfuse) | 自托管 trace、dataset、code evaluator、LLM-as-Judge、人工标注/score | 项目已有 OTel/Langfuse 与 eval runner | 继续作为观测、数据集、评测归档平台；不改变 runtime 判定。 |
| [Promptfoo](https://github.com/promptfoo/promptfoo) | endpoint 级 deterministic assertions、`llm-rubric`、CI 报告 | 适合未来把已稳定 dataset 作为独立黑盒 CI 运行 | 不在 Strict Schema Phase 1 加依赖；质量基线成熟后可单独做 PoC。 |
| [DeepEval](https://github.com/confident-ai/deepeval) | Python pytest 风格的 RAG / Agent metrics、LLM Judge | RAG faithfulness、tool trajectory 可补充 | 不接入运行时；当前 Python runner 与 Langfuse 已覆盖基本入口，避免双评测栈漂移。 |

不推荐另写“最终文本规则引擎”：那会重演 `validateDynamicFireBureauFacts` 的失败模式，把每一次模型失误变成新的字符串规则。新增的确定性规则必须只检查结构、ID、权限、scope、工具结果和 renderer 输入输出合同。主观文本质量由离线 Judge + 人工标注衡量，生产安全由确定性合同保障。

Judge 的最低合同：使用生成模型之外的 judge 或独立上下文；每个维度输出 boolean + reason；先对至少 100 条人工标注样本按版本校准；不把泛化的“helpfulness”当发布门槛。建议维度为：事实忠实、方法边界、用户问题覆盖、可读性、保守降级诚实性。每个真实故障必须进入 fixture，而不是再加 runtime 专项文本分支。

## 影响面与清理清单

| 模块 | 实施动作 | 保留 / 不改 |
|---|---|---|
| `llm` 工厂和 provider adapter | 新增 Strict provider capability/adapter；删除 `NewToolCallingModelWithJSON` | 普通 DeepSeek `NewToolCallingModel`。 |
| `container` | 移除 JSON/fast JSON 两模型；只注入普通模型与 structured model | 工具注册、tracer、Manager/Executor owner。 |
| `AgentBuilder` / specialists | 删除 `UseJSONMode`、`modelCreator`、`fastModelCreator` 分流；结构节点按 schema 显式调用 structured model | 普通 specialist 的 model/fastModel 选择。 |
| BaZi inner graph | 以 schema-aware runner 替代 `runBaziInnerAgentJSON`；所有六个 DTO 迁移 | bootstrap、evidence retrieval、外层 graph 顺序。 |
| DTO | 拆分 model-owned DTO 与 runtime projection，给嵌套对象 `additionalProperties:false` | runtime `Source`、`RecoveryReason`、`FieldAudit`、`ContractAudit` 所有权。 |
| prompt | 删除重复“返回 JSON / 不要 markdown fence”的格式指令；保留命理任务、边界和引用可用范围说明 | 领域方法、用户问题、检索证据 prompt。 |
| repair/retry | 新 taxonomy、分离 transport/schema repair attempt、精确 trace | 现有 `RuntimeFailure` 和 recovery state machine 的 owner。 |
| 专项 validator | 通用 catalog/ref validator 完整覆盖后，删除 `validateDynamicRelationFacts`、`validateDynamicFireBureauFacts` 及专用 tests | scope、age、method、evidence、enum 等非“关系文本扫描”合同。 |
| trace/error | 增加 schema name/provider capability/ref kind/attempt，保留现有 failure/recovery 字段 | trace ID、Run Inspector、SSE wire shape。 |
| eval fixtures | 增加严格请求、坏载荷、引用、retry/recovery fixture；把“丙戌火局”转为 generic invalid relation ref 回归 | 现有 eval 命令与报告路径。 |
| Supervisor | 保留 ADK output tool 的 Schema 与 `parseAndValidate`；它不是 JSON Mode | 需在独立 cleanup phase 删除 text fallback 中“prompt JSON -> Go decision”的结构消费，改为 output-tool 成功或 deterministic safe fallback；不得碰 tool schema 本身。 |
| Qimen / Ziwei | 不迁移它们的自由文本最终成文 | Manager/Prefill/ToolRunner 边界与资产合同。 |
| renderer / SSE / 前端 | renderer 改为回读 catalog；为 field omission 维持现有 `model_partial` 表达 | event type、payload shape、组件、前端 API 不变。 |
| 配置 / 部署 | 增加结构 provider 的明确配置与启动自检；部署注入这些变量 | `LLM_*` 的普通模型语义和无 JSON Mode 回落。 |

## 分阶段实施

每阶段只在前一阶段验收后进入下一阶段；没有 legacy JSON Mode 的生产双轨。Phase 1/2 的未接入组件可以在分支中存在，但不接收 production structured 调用；Phase 3 的所有 BaZi Go-consumed node 与 Phase 6 的删除必须作为同一 cutover 发布。灰度只能开关“已通过 probe 的 Strict provider 是否为 production structured provider”，不能把请求切回 JSON Mode。

### Phase 0：provider capability probe 与选择

| 项目 | 内容 |
|---|---|
| 改动范围 | 选定候选 endpoint，发送最小 `json_schema` / `strict:true` 探针，保存脱敏结果与 adapter request capture。 |
| 新增/删除 | 先不改生产源码；可新增临时、不提交的 probe 结果。选定 provider 后才新增 capability test。 |
| 不改边界 | 不改 `LLM_*`、DeepSeek 正常链、BaZi graph、SSE。 |
| 验证 | 200 响应；missing required、wrong type、unknown field、invalid enum 均不能作为成功 DTO 进入本地 decoder。 |
| 验收 | endpoint、model、adapter、`json_schema`、`strict:true`、错误语义均有证据；否则停止实施并向 owner 请求 provider 选择。 |
| 回滚 | 无生产变更；保留当前发布版本，不启用 JSON Mode 作为替代。 |

### Phase 1：结构 DTO、schema 与 strict decoder

| 项目 | 内容 |
|---|---|
| 改动范围 | 定义 model-owned schema DTO、schema generator、strict decoder 和统一错误类型。 |
| 新增/删除 | 新增 `backend/internal/runtime/structured_contract.go`、`structured_decode.go` 与测试；调整 `bazi_charter_types.go`；不删除 JSON Mode 代码。 |
| 不改边界 | 不动 provider、container、renderer、Supervisor、Qimen/Ziwei。 |
| 验证 | `go test ./backend/internal/runtime -run Strict`；缺字段、错误类型、非法 enum、unknown field、trailing JSON 全部失败。 |
| 验收 | 一个 DTO 能稳定生成闭合 Schema，decoder 对同一 Schema 严格拒绝坏载荷。 |
| 回滚 | 删除未接入调用方的新合同文件即可；不影响运行路径。 |

### Phase 2：Strict provider adapter 与配置

| 项目 | 内容 |
|---|---|
| 改动范围 | `config`、`llm`、`container` 注入 structured provider，并在启动/首调用前验证 capability。 |
| 新增/删除 | 新增 strict adapter 与 request-capture tests；更新环境变量示例和部署文档；仍不删 JSON Mode。 |
| 不改边界 | 普通 DeepSeek 工具/文本、Manager、outer graph、SSE。 |
| 验证 | 单元测试断言实际 request 含 `response_format.type=json_schema`、schema name、`strict:true`；真实 endpoint probe。 |
| 验收 | 未配置或不支持时得到 `STRUCTURED_MODEL_CAPABILITY_UNAVAILABLE`，绝不降级。 |
| 回滚 | 停用尚未接入的 structured provider 发布开关；修复 provider/config，不切回 JSON Mode。 |

### Phase 3：BaZi active node 迁移

| 项目 | 内容 |
|---|---|
| 改动范围 | 在同一变更集内依次完成 `analysis_plan`、`evidence_plan`、`canonical_synthesis`、`contract_audit`、legacy `static_synthesis`、`dynamic_synthesis` 的测试迁移；只有六个节点全部就绪才允许 production cutover。 |
| 新增/删除 | 迁移节点 tests 与 fixtures；用 strict runner 替换对应 `runBaziInnerAgentJSON` 调用。 |
| 不改边界 | 不能让任何节点绕过 Prefill、Manager、final guard 或 renderer；不改 SSE 形状。 |
| 验证 | 每迁移一个节点跑相关 runtime tests；完成后 `go test ./backend/... -count=1`、`go build ./backend/cmd/server/`。 |
| 验收 | production cutover 时，所有 Go 消费的 BaZi 模型输出都走 provider Strict Schema + local strict decode；不存在节点级 JSON Mode fallback。 |
| 回滚 | 在发布前撤回整个 cutover 版本；运行时不提供 JSON Mode 开关。 |

### Phase 4：runtime catalog 与通用 reference validator

| 项目 | 内容 |
|---|---|
| 改动范围 | 由 chart/dynamic facts 建立本轮 fact、relation、claim catalog；统一 validator 检查 ID/kind/period/subject/evidence scope。 |
| 新增/删除 | 新增 catalog/validator owner 文件及通用 tests；将 `knownBaziFactRefs` 升级或替换为 catalog。 |
| 不改边界 | runtime 不开始生成命理 claim，不把关系判断移进 renderer。 |
| 验证 | 不存在 relation ref、错 kind、错 period、错 claim ref 均失败；“丙戌火局” fixture 只依赖 generic validator。 |
| 验收 | renderer 从 catalog 展示事实关系；模型无路径直接提交“火局”这种事实文本。 |
| 回滚 | 撤回尚未删专项 validator 的提交；不把失败输入放行。 |

### Phase 5：收缩 runtime projection 与修复链

| 项目 | 内容 |
|---|---|
| 改动范围 | 统一错误 taxonomy、trace attempt、facts-only/model_partial 条件；canonical 成为唯一模型语义源，legacy 字段仅 runtime projection。 |
| 新增/删除 | 更新 `repair_policy.go`、contract failure/trace tests、renderer fixtures；删除已无调用方的 JSON fence repair。 |
| 不改边界 | 不改最终答案 owner，不把 recovery decision 交给模型。 |
| 验证 | `make eval-repair`；一次 schema repair 后进入正确 recovery；transport 与 repair attempts 不混淆。 |
| 验收 | trace 不再出现 `retryable=true` 却没有实际重跑；任何事实冲突都不因改措辞恢复。 |
| 回滚 | 版本回退或停发；不得在生产把失败类映射回 JSON Mode。 |

### Phase 6：删除 JSON Mode 与 prompt-JSON 结构消费

| 项目 | 内容 |
|---|---|
| 改动范围 | 与 Phase 3 同一 cutover 删除 JSON model 工厂、container JSON 实例、`UseJSONMode`、`NewToolCallingModelWithJSON`、markdown fence stripping 与重复 JSON 格式 prompt。收敛 Supervisor text JSON fallback。 |
| 新增/删除 | 删除对应代码、tests、配置文案；保留 Supervisor ADK output tool Schema。 |
| 不改边界 | 不删除 Supervisor output tool，不改变 Manager/ExecutionPlan 或 SSE。 |
| 验证 | `rg` 不再命中 `json_object`、`UseJSONMode`、`NewToolCallingModelWithJSON` 或 JSON Mode fallback；Supervisor tool tests 通过。 |
| 验收 | 项目中不存在“模型输出给 Go 消费但靠 prompt JSON/JSON Mode”的正式路径。 |
| 回滚 | 仅回滚未发布提交或停止发布；不保留 legacy runtime switch。 |

### Phase 7：删除专项关系 validator 与全量回归

| 项目 | 内容 |
|---|---|
| 改动范围 | 删除 `validateDynamicRelationFacts`、`validateDynamicFireBureauFacts`、正则辅助函数及专项 tests；保留通用 ref regression。 |
| 新增/删除 | 删除专项 validator；补 generic catalog fixture。 |
| 不改边界 | 不删 scope/method/evidence 校验；不放宽动态事实合同。 |
| 验证 | 完整测试矩阵，真实 endpoint 回归和 SSE shape snapshot。 |
| 验收 | “丙戌火局”由无效 `relation_ref` 失败，代码中没有火局/水局/金局专用业务 validator。 |
| 回滚 | 回滚这一删除提交；不恢复 JSON Mode。 |

## 测试矩阵与发布门槛

必跑命令：

```bash
go test ./backend/... -count=1
go build ./backend/cmd/server/
make regression
make eval-repair
make eval-bazi-quality
make eval-bazi-stability
```

新增测试必须覆盖：

1. 请求真实携带 `response_format.type=json_schema`、schema 名称、`strict:true`。
2. 缺 required、错误 type、非法 enum、unknown field、trailing JSON 都失败。
3. strict provider 未配置、adapter 无此能力、provider 400 capability error 都是明确错误，不降级。
4. 无效 `fact_ref` / `relation_ref` / `claim_ref` 被同一通用逻辑拦截；wrong kind、wrong period 也失败。
5. “丙戌火局” fixture 不依赖任何火局专项 validator。
6. 一次 schema repair 后正确进入 facts-only 或 hard error；不产生 `model_partial` 假成功。
7. transport retry 与 schema repair 计数、trace 字段、反馈内容互不混淆。
8. Supervisor 已有 output tool Schema 行为不回归；其新 fallback 不再接受 prompt JSON 结构结果。
9. renderer 从 catalog 回读，SSE wire shape 与前端组件 contract 不变。
10. `make eval-bazi-quality`、`make eval-bazi-stability`、answer-quality Judge 对照基线无回归；生产采样 Judge 先通过人工标注校准。

## 第一实施边界

**第一步只做：** 为一个候选 structured provider 做 capability probe，证明“真实 endpoint + model + adapter”能接受 provider-native `json_schema` 与 `strict:true`，并能在本地捕获完整 request。

**第一步不做：** 不改任何 BaZi DTO、renderer、recovery、container、现有 DeepSeek 普通模型、Supervisor、Qimen/Ziwei，也不添加 JSON Mode 兼容开关。

**第一步可能修改的文件：** 默认不改生产文件；在 provider 已由 owner 选定后，才新增最小的 adapter capability test，位置优先为 `backend/internal/llm/*_test.go`。不要先创建常驻 probe service、抽象 factory 或十多个空 Schema 文件。

**第一步验证：** request capture 断言 + 真实 probe；坏 Schema case 不能作为成功 DTO 进入 strict decoder；记录脱敏 status/error/model/endpoint family。

**第一步完成后再决定：** 选择 provider-native adapter 还是 Eino output tool，并确认 structured provider 的配置/成本/部署入口；之后才进入 Phase 1。
