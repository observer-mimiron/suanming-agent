# 近期运势综合判断实施方案

> 目标：把“本月 / 今年 / 最近一段时间运势”从模糊路由收敛为可执行、可验收的 Manager-owned runtime 合同。
>
> 本文是实施目标文档，不表示功能已全部落地。执行时必须先由 subagent 分域审查，主 agent 再审核合并。

> 本版是审核后的冻结方案。除本文明确的字段、取值和阶段顺序外，不接受“新增字段或等价表达”“先兼容再决定”等实现分支。方案审核阶段只修改本文，不修改运行时代码，也不把未实现内容写入 `PROGRESS.md`。

## 结论

近期运势综合判断应固定为“八字主导、紫微复核、奇门只处理具体问事”。

本次最小改动不新增六爻、梅花，也不把 Manager 升级成开放式 ReAct 主控。改动重点是任务分类、ExecutionPlan 资产合同、Prefill 确定性事实、Manager 合成规则和奇门前端展示。

核心修复目标：

- 个人阶段运势：八字大运 / 流年 / 流月为主，紫微流年 / 流月或宫位复核为辅，奇门不参与。
- 具体事件问事：奇门按本轮提问时间创建新 Case 起问事盘，不使用出生时间。
- 健康风险类：八字主线 + 紫微疾厄复核，必须安全免责声明，不做医疗诊断。
- 出生盘分析：按用户明确方法排盘，不自动扩成三域综合。

## 当前架构约束

当前主链保持不变：

~~~text
RouteAdvisor
-> Policy Gate
-> Manager
-> ExecutionPlan
-> Prefill
-> ToolRunner
-> specialist runner(s)
-> Manager compose
-> final guard
-> SSE
~~~

职责边界：

| 层 | 负责 | 不负责 |
|---|---|---|
| RouteAdvisor | 粗路由、准入信号、槽位提取 | 会话资产选择、最终综合 |
| Policy Gate | 白名单、低置信度、资料完整性、硬准入 | 领域资产 owner 绑定 |
| Manager | 会话 owner、任务纠偏、ExecutionPlan、最终综合 | 开放式工具循环 |
| ExecutionPlan | 本轮 DomainSteps 与 ArtifactRequirement | 命理裁断 |
| Prefill | 确定性排盘和动态事实准备 | 白话解释 |
| specialist | 领域解读和证据组织 | 重排已有盘、最终答复权 |
| final guard | 最后一层输出边界保护 | 代替前置合同 |

架构铁律：

- 确定性事实必须由工具 / 规则生成，不能让 LLM 猜干支、流年、流月、奇门局。
- 奇门问事盘归属 Case，不归属 ProfileRevision。
- Manager 统一综合，不允许三个 specialist 平均投票。
- renderer 只展示上游结构化结果，不做路由和语义补丁。
- 不为单个命盘、单条 trace、单个用户话术写专项分支。

## 审核结论与冻结接口

上一版可以表达目标，但不能直接交给实现 agent，原因是关键合同仍有多个可选解释：

- 路由既允许新增 `consultation_kind`，又允许只用 `task_intent`；不同实现会得到不同的可观测合同。
- `qimen_chart` / `qimen_case_chart`、Case 创建时机和 `question_time` 来源没有唯一规则；重建 ExecutionPlan 可能重复建 Case 或复用旧盘。
- 动态运势、主次领域和健康免责声明主要停留在文本说明，缺少可由代码断言的 owner、角色和失败策略。
- 现有八字内部 Graph 已负责动态事实准备，本期不应再设计一套持久化 `bazi_dynamic` / `ziwei_dynamic` 资产来扩大改动面。

本版冻结以下最小合同：

### 1. 路由和计划字段

`ApprovedRoute` 新增唯一规范字段 `ConsultationKind`，取值只能是：

`period_fortune`、`event_question`、`health_risk`、`natal_chart`。

`ExecutionPlan` 必须包含：

| 字段 | 合同 |
|---|---|
| `ConsultationKind` | 由 `ApprovedRoute` 透传，Manager 不重新猜测 |
| `SafetyProfile` | `none` 或 `health_observation` |
| `DomainSteps` | 有序的 `{Domain, Role}`；`Role` 只能是 `primary` / `support`，primary 必须成功，support 允许 degraded |
| `Requirements` | 本轮唯一的资产和动态事实准备合同 |
| `TurnContext` | 本轮统一时间、目标时点和 Case 引用 |

四值分类只适用于“已准入的咨询执行轮次”。`collect_profile`、`amend_profile`、`direct_bazi` 和纯澄清短路属于前置/等待状态，`ConsultationKind` 必须保持空值，不能为了填满枚举而伪装成 `natal_chart`；这些轮次不创建 specialist ExecutionPlan，也不创建 Case。

`Snapshot.PrimaryDomain`、`SecondaryDomains`、`QimenMode` 和 `Snapshot.Gate.ProfileRequirement` 保留为兼容/观测投影；执行和合成以 `ConsultationKind`、`DomainSteps`、`SafetyProfile` 为准，不能从结果顺序反推主次。

### 2. TurnContext（本轮执行上下文）

```text
TurnContext {
  TurnID                // 本轮唯一标识，用于同轮重建幂等
  QuestionTime          // 入口捕获的一次性 RFC3339 时间；event_question 必填
  TargetAt              // 用户指定的运势目标时点；未指定时由 QuestionTime 推导
  TemporalGranularity   // instant / day / month / year / range
  Source                // server_clock 或 user_explicit
  CaseID                // event_question 已创建的 Case；其它类型为空
}
```

规则：

- `QuestionTime` 在 handler/orchestrator 进入本轮时捕获一次，默认使用项目统一的 Asia/Shanghai 时区；Prefill、ToolRunner 和 specialist 不得再次调用 `time.Now()` 推导本轮时间。
- `event_question` 的 `Case.EventTime` 必须等于 `TurnContext.QuestionTime`；如果用户描述未来事件时间，也不能用它替换问事起局时间。
- 同一 `TurnID` 重建或强制路由时复用 `CaseID`；一个新 `event_question` 用户轮次最多创建一个新 Case。

TurnContext 和 Case 的唯一写入顺序固定为：

1. Executor 作为 runtime turn entry 在进入 Manager 前捕获一次 `QuestionTime`，并从已批准路由 / 槽位确定 `TargetAt`；本期没有可解析的用户 RFC3339 目标时间槽位，因此 `TargetAt` 默认等于 `QuestionTime`、`Source` 固定为 `server_clock`。生产路径缺少时间时失败或进入已有澄清，不以 `time.Now()` 在后续阶段补齐；`user_explicit` 只能在未来增加确定性时间解析后启用。
2. Manager 是唯一的 Case 创建 owner。只有新的 `event_question` 用户轮次可以以 `fresh=true` 创建 Case，并把 CaseID 写入 TurnContext；同一轮的 forced route、Plan rebuild 和 resume 只能复用它。
3. `artifact_resolver.go` 只解析 Subject、ProfileRevision 和问题焦点，不创建 Case；`selectArtifactRequirements` 必须是无副作用纯函数，不读取或修改 ActiveFocus 来决定新 owner。
4. forced route 最多重建一次有效 ExecutionPlan；重建时传入原 TurnContext 和 CaseID，Prefill 写回 graph-local effective plan，agent 节点只消费该计划，不再次 BuildExecutionPlan。

`Source` 的取值固定为 `server_clock` 或 `user_explicit`，描述 `TargetAt` 的来源；`QuestionTime` 当前始终是本轮 runtime 捕获的 server clock。`TemporalGranularity` 只允许 `instant`、`day`、`month`、`year`、`range`。

### 3. ArtifactRequirement（资产准备合同）

字段固定为：

```text
ArtifactRequirement {
  Kind
  OwnerRef
  SubjectIDs
  CalendarRule
  Scope
  TargetAt
  Purpose
  InputRefs
}
```

本期持久化资产 kind 只保留现有 `bazi_chart`、`ziwei_chart`，以及新增的规范名 `qimen_case_chart`。八字/紫微动态事实按 `Scope` 和 `TargetAt` 作为本轮 Prefill 输入，不新增另一套持久化动态资产；已有 `bazi_liunian` / `ziwei_liunian` 只能作为 runtime-owned 确定性工具调用。

动态 Prefill 结果至少带 `scope`、`target_at`、`status` 和结构化 `facts`；`status` 只能是 `ready`、`degraded` 或 `unavailable`。`unavailable/degraded` 是运行时合同状态，不得由模型用推测文本填平。

动态事实的最小结构固定为：

```text
DynamicFacts {
  scope: dayun | liunian | liuyue
  target_at: RFC3339
  status: ready | degraded | unavailable
  facts: structured object
}
```

`DynamicFacts` 是本轮 Prefill 的结果对象，不是新的持久化 asset kind，也不是 specialist 自由输出；它应附着在 runtime 私有的 `StepOutcome` / trace 投影上。`liuyue` 当前能力未实现时必须返回 `unavailable` 或 `degraded`、空或不完整的 `facts`，并由 Manager 在最终回答中显式说明缺口；不得让 specialist 或 Manager 从自然语言重新推算流月。只有确定性结果与 `TargetAt` 对齐时才可标记 `ready`。每个动态 requirement 必须实际填写 `Scope`、`TargetAt`、`Purpose` 和 `InputRefs`，不能只在 schema 中保留空字段。

奇门资产规则固定为：

- 新写入统一使用 `qimen_case_chart`；旧 `qimen_chart` 只作为读取迁移别名，不能作为新合同或新写入 kind。
- `OwnerRef.Kind == case`，`OwnerRef.ID == TurnContext.CaseID`，`Purpose == event_question`。
- Prefill、specialist、final guard 都引用同一 `ArtifactRequirement`，不得各自从 `ActiveFocus` 猜资产。

### 4. Specialist 和安全合同

- Specialist 的 `ToolNames` 只允许知识检索工具；`bazi_calc`、`bazi_liunian`、`qimen_dunjia`、`ziwei_calc`、`ziwei_liunian` 只能由 Prefill/ToolRunner 调用。
- Qimen specialist 的上下文只提供 `qimen_case_chart`、问题文本和结构化问事事实，不注入 `profile`、`gender`、`birthplace` 或任何出生字段。
- Qimen specialist 只接收当前用户问题和当前 Case 的最小结构化视图，不注入包含出生资料的 `RecentTurns`、`RunningSummary` 或完整 SessionState；prompt 中的“不要使用出生资料”不是唯一隔离措施。
- `SafetyProfile=health_observation` 时，输出只能表达风险倾向、可观察事项和就医建议；禁止诊断、治疗、病名确定、预后和生死断言。
- final guard 由 `SafetyProfile` 强制附加固定免责声明：`命理仅供参考，不能替代医学诊断；如有不适请及时就医`。不得把免责声明留给 prompt 或前端 renderer。

`qimen_dunjia` 的 ToolContract 必须声明且只声明一个必填参数 `question_time:string`；未知参数和 `gender`、`birthplace`、`year`、`month`、`day`、`hour`、`minute`、`longitude` 等出生字段必须在 ToolRunner 边界拒绝，或根本不进入调用。

## 任务分类合同

`ConsultationKind` 是唯一规范分类字段；`task_intent` 继续表达子任务，但不能替代分类，也不能把分类留给 Manager 二次猜测。

| consultation_kind | 用户信号 | DomainSteps | QimenMode | ProfileRequirement | SafetyProfile |
|---|---|---|---|---|---|
| `period_fortune` | 本月、今年、最近半年、近期运势 | `bazi:primary`、`ziwei:support` | `none` | `full` | `none` |
| `event_question` | 这个面试、这次签约、今天谈合作、出行择时、这件事能否成 | `qimen:primary` | `primary` | `none` | `none` |
| `health_risk` | 会不会生病、身体有没有问题、最近健康如何 | `bazi:primary`、`ziwei:support` | `none` | `full` | `health_observation` |
| `natal_chart` + 显式八字 | 分析八字、看八字命盘 | `bazi:primary` | `none` | `full` | `none` |
| `natal_chart` + 显式紫微 | 分析紫微、看紫微命盘 | `ziwei:primary` | `none` | `full` | `none` |

补充约束：

- `period_fortune` 和 `health_risk` 的 `SecondaryDomains` 必须包含 `ziwei`；紫微不可用时只能 support degraded，不能改写八字为紫微主导。
- `event_question` 默认不带 bazi/ziwei support；即使会话已有出生资料，`ProfileRequirement` 仍为 `none`，Qimen specialist 也不得看到出生资料。
- 本期不定义“健康 + 具体短期行动”的第三种路由语义；这类问题进入澄清，不新增 `event_question` 的健康例外，也不让奇门问事输出医疗行动结论。
- 没有完整出生资料时，“今天/此刻运气如何”按问事时刻归入 `event_question`，保留现有 AC-1.3 的 qimen primary 合同；有完整资料且明确询问本月/今年/近期个人阶段时，才归入 `period_fortune`。
- `natal_chart` 只执行用户明确点名的方法。紫微工具若内部需要出生资料或八字基础字段，只能作为该工具的确定性输入，不得把 bazi 追加为用户可见的 `DomainStep`；普通八字出生盘不得追加紫微或奇门。
- `natal_chart + ziwei` 的底层 `ziwei_calc` 若需要出生字段或八字基础字段，只能作为紫微 Prefill 的确定性输入；不得因此调度 bazi specialist，也不得把 bazi 加入用户可见的 DomainSteps。
- 用户只说“看命盘”但未点名方法时，进入澄清，不默认三域。

### 分类优先级

按以下顺序确定 `ConsultationKind`，顺序固定：

1. 先识别具体事件 / 择时语义。明确询问面试、签约、合作、出行或某件事能否成时归入 `event_question`，奇门主导；已有出生资料也不改变这一点。
2. 如果具体事件同时明确要求用八字 / 紫微出生盘判断，视为方法与任务冲突，进入澄清；不得静默改成出生盘，也不得把奇门偷偷变成个人阶段运势的 supplement。明确要求奇门起局的具体事件不冲突，仍归入 `event_question`。
3. 健康风险语义优先于出生盘方法词：例如“用八字看看最近身体”“紫微看健康”仍归入 `health_risk`，保留八字主导、紫微复核和免责声明；只有没有健康目标的纯“分析 / 排盘 / 命盘”请求才进入下一条。
4. 没有具体事件或健康目标时，明确包含“分析 / 排盘 / 命盘”等出生盘语义且点名方法，归入 `natal_chart`，只保留点名方法。
5. 个人阶段时间语义归入 `period_fortune`。
6. 仍无法判断时澄清，不用 qimen 兜底。

方法词只在上述语义允许的范围内生效：不能用“奇门”把个人阶段运势偷偷变成 qimen supplement，也不能用“八字”把具体事件改成出生盘分析。明确要求不支持的组合只进入现有 `NeedsClarification`，不新增分类值或专项 case 分支。

## 领域编排规则

### 八字必须主导

以下场景八字必须是 PrimaryDomain：

- period_fortune：本月、今年、近期、最近半年、最近运势。
- health_risk：近期身体风险、健康倾向。
- natal_chart 中用户明确要求八字。

八字主导时应准备：

- bazi_chart：本命盘。
- 动态事实：由现有八字内部 Graph / runtime-owned 工具按 `TurnContext.TargetAt` 准备。第一版至少保证当前大运和目标流年有确定性输入；流月未完整实现时返回显式 `unavailable/degraded`，不得让模型补算。

### 紫微参与规则

默认参与：

- period_fortune 且出生资料完整：`ziwei` 作为 support step。
- health_risk 且出生资料完整：`ziwei` 作为 support step，使用疾厄宫 / 流年宫位复核。

以下场景不加入紫微 step：

- 用户只问纯八字命盘。
- 用户只问单点术语解释。
- 当前资料不足时先按 `ProfileRequirement=full` 澄清；不能用缺资料的 support step 猜测结果。

紫微动态工具缺少目标范围时仍保留 support step，但必须返回 `unavailable/degraded`，不能静默删除该 step 或让模型补算。

紫微参与时应准备：

- ziwei_chart：本命盘。
- 流年复核事实：只由 runtime-owned `ziwei_liunian` 准备；流月若未实现，返回显式 `unavailable/degraded`。

### 奇门参与条件

奇门参与：

- event_question。
- 用户明确要求奇门、起局、问事盘、择时，且问题语义确实是具体事件。

奇门不参与：

- 本月运势、今年运势、最近半年运势。
- 出生盘分析。
- 健康阶段风险，除非用户问一个具体短期事件，例如“今天要不要去检查”。

奇门起局时间：

- 只使用 `TurnContext.QuestionTime`，由入口一次性捕获并传入。
- 不使用出生年月日时。
- 不读取 `gender`、`birthplace`、`longitude` 或任何出生专属字段；用户描述的未来事件时间也不替换问事起局时间。

## 数据与资产合同

`ArtifactRequirement` 扩展到“基础盘 + 动态范围 + Case 盘”，但不新增一套持久化动态资产。

| 资产概念 | kind | OwnerRef | 输入引用 | 说明 |
|---|---|---|---|---|
| 八字本命盘 | bazi_chart | profile_revision | ProfileRevision | 已有核心资产 |
| 紫微本命盘 | ziwei_chart | profile_revision | ProfileRevision | 已有核心资产 |
| 奇门问事盘 | qimen_case_chart | case | Case + question_time | 不能复用出生资料盘 |

兼容策略只有一条：现有 `qimen_chart` 资产可被迁移读取，但任何新写入、计划快照和前端 payload 都必须使用 `qimen_case_chart` 语义。不得把“先继续使用旧 kind”作为实现分支。

ArtifactRequirement 语义：

| 字段 | 用途 | 示例 |
|---|---|---|
| Kind | 资产类型 | bazi_chart、ziwei_chart、qimen_case_chart |
| OwnerRef | 资产 owner | profile_revision 或 case |
| SubjectIDs | 所属对象 | 自己 / 孩子等 subject id |
| CalendarRule | 八字历法口径 | zi_zheng_true_solar_v2 |
| Scope | 动态范围 | none、dayun、liunian、liuyue |
| TargetAt | 目标时点 | `TurnContext.TargetAt` |
| Purpose | 起盘用途 | period_fortune、event_question |
| InputRefs | 上游资产引用 | 对应基础盘和 Case ref |

防误用要求：

- qimen_case_chart 的 OwnerRef.Kind 必须为 case。
- qimen_case_chart 的 Purpose 必须为 event_question。
- qimen_case_chart 的 `question_time` 必须等于 `TurnContext.QuestionTime`，Case.EventTime 也必须相等。
- `qimen_dunjia` 的 runtime 调用参数只接受 `question_time`；适配层内部可将其转换为排盘库需要的日期字段，但这些字段必须标记为 question-time derived，而不是出生字段。
- `prefillQimen` 只接受计划中的 Case 和 `TurnContext`，不得直接读取 `ActiveChart(qimen_chart)` 作为本轮结果。
- 计划重建必须复用原 `CaseID`、`TurnContext` 和 requirement；新 Case 只能在新用户轮次的 Manager 初始化阶段创建。
- specialist 不能直接调用 qimen_dunjia 重新起局。

Qimen chart payload 必须补齐 `case_id`、`owner_ref`、`purpose`、`question_time`、`time_source=question_time`、`pan_schema=rotating_8` 和 `symbol_system=eight_gate_eight_god`；其中 `owner_ref.kind` 必须为 `case`。这些字段由 runtime 绑定，不能由 specialist 或前端猜测。

## 输出合成规则

Manager 合成必须按主次，不投票。

合成输入必须携带 `DomainStep.Role`，不得用 specialist 返回顺序、字符串前缀或 renderer 规则猜主次：

runtime 内部使用以下最小 step outcome，不扩张公共 specialist 结果合同：

```text
StepOutcome {
  domain: string
  role: primary | support
  status: ready | degraded | failed
  result: specialists.Result
  error: optional error
}
```

- `primary` 失败：本轮执行失败或进入已有的确定性降级，不得伪造该领域结论。
- `support` 失败：保留 primary 结果，记录 `support_degraded`，用户可见说明只能说“复核资料暂不可用”，不得把 support 结果补写出来。
- support 与 primary 冲突：primary 保持主线，support 只输出同向、差异和限制，不进行平均投票或覆盖主线。
- Manager compose 只能消费按 Role 分组后的 outcome；support 结果不能通过字符串拼接顺序覆盖 primary。
- 当 runtime 已提供结构化 Role outcome 时，Manager 直接输出带主线 / 复核锚点的确定性投影，不再交给 fast model 自由改写主次；只有没有 Role outcome 的兼容结果才允许走 best-effort 合成。

final guard 必须接收当前 `ExecutionPlan`（或其中精确 requirement），而不是只接收 ApprovedRoute 和 legacy projection。qimen 只有在 `qimen_case_chart.OwnerRef.ID == TurnContext.CaseID`、`case_id`、`question_time` 和 `purpose` 全部匹配时才算满足主资产；旧 `QimenResult` 或 ActiveFocus 中不匹配的 qimen 盘不得通过 guard。bazi / ziwei 也优先按当前 requirement 校验 owner、subject 和历法规则。

### period_fortune 输出结构

~~~text
1. 八字主线结论
   - 当前大运位置
   - 目标流年 / 流月触发点
   - 对事业、财运、感情、健康等主题的主线判断

2. 紫微复核结论
   - 哪些宫位被流年 / 流月点亮
   - 与八字主线同向或差异处
   - 只做复核，不推翻主线

3. 近期建议
   - 可观察事项
   - 行动节奏
   - 风险边界

4. 免责声明
   - 命理仅供参考，不构成人生决策建议
~~~

### event_question 输出结构

~~~text
1. 奇门问事结论
   - 此事此时是否顺
   - 门、星、神、宫位的短期态势

2. 操作建议
   - 时机、方向、沟通姿态
   - 哪些动作适合做，哪些动作应谨慎

3. 八字 / 紫微背景
   - 只有在计划里被要求时出现
   - 不抢奇门对此事此时的主线

4. 边界说明
   - 奇门问事盘只代表本事本时，不代表长期命运
~~~

### health_risk 输出结构

~~~text
1. 八字阶段风险倾向
2. 紫微疾厄 / 流年复核
3. 现实观察建议
4. 医疗免责声明
~~~

健康免责声明必须包含以下固定语义，直接使用固定文案，不能只依赖同义词 prompt：

`命理仅供参考，不能替代医学诊断；如有不适请及时就医`

健康类禁止：

- 禁止断具体疾病。
- 禁止断生死。
- 禁止输出“你一定会 / 一定不会生病”。
- 禁止替代医学诊断。
- 禁止把 support 缺失或命理冲突写成医学确定性结论。

冲突处理：

| 冲突类型 | 处理 |
|---|---|
| 八字说阶段压力高，紫微复核弱 | 以八字主线为主，紫微降为“宫位未强烈同向” |
| 八字长期平稳，奇门短期不利 | 对具体事件以奇门为主，说明只是短期问事 |
| 紫微宫位强，八字动态弱 | 标为“领域关注被点亮，但岁运主线不足以强化为确定结论” |
| 健康类任意冲突 | 全部降为风险倾向和观察建议 |

## 前端展示规则

奇门盘必须展示四个元信息：

| 展示项 | 字段 | 普通用户文案 |
|---|---|---|
| 起局用途 | purpose | 问事盘 |
| 起局时间 | question_time | 2026-08-05 14:30 |
| 时间来源 | time_source | 本轮提问时间 |
| 盘式 | pan_schema | 标准转盘 |
| 门神体系 | symbol_system | 八门八神 |

后端 payload 必须同时带 `case_id`、`purpose=event_question`、`owner_ref.kind=case`；前端只展示这些字段，不根据文本判断“这是问事盘”。

默认展示口径：

- pan_schema=rotating_8 显示为“标准转盘”。
- symbol_system=eight_gate_eight_god 显示为“八门八神”。
- rotating_8 不应出现“中门、太常、勾陈、朱雀”等九门九神符号。

异常展示：

- 如果 pan_schema=rotating_8 但 cells 出现“中门 / 太常 / 勾陈 / 朱雀”，前端显示“盘式字段与符号体系不一致”。
- 后端 contract guard 必须拒绝新生成的不一致 payload；前端 warning 只负责展示历史/legacy 异常 payload，不能删除、替换或隐藏异常 cell。
- 不能静默隐藏异常。
- 复制 Markdown 也必须包含 Case、起局用途、起局时间、盘式、门神体系和异常 warning。

## Subagent 执行组织

主 agent 不直接进入大改。先派 subagent 做只读分析和局部方案，再由主 agent 审核后执行最小 diff。

| Subagent | 范围 | 交付 | 禁止 |
|---|---|---|---|
| A Runtime Contract | 只读检查 schemas、supervisor、policy、manager、execution_plan、artifact_resolver；实现阶段只写 contracts、supervisor、policy、`execution_plan.go`、`artifact_resolver.go` | 分类合同、路由硬纠偏、ExecutionPlan 字段和快照 | 改工具算法、改前端、改 `manager.go` 合成逻辑 |
| B Artifacts / Prefill | executor、specialist_runner、state/assets、bazi/ziwei/qimen tools | Case 归属、question_time 参数链、确定性事实、最小上下文视图 | 让 LLM 生成确定性事实、改路由语义、改 Manager 合成 |
| C Manager Synthesis | `manager.go`、execution_dispatch / execution_plan_runner、agent_route、specialists、prompts | primary/support 合成、健康安全合同接线、Specialist 工具白名单和 prompt | renderer 语义补丁、重排命盘、改 A/B 的 owner 合同 |
| D Frontend | QimenChart、chat types、assistantTurn、前端测试 | 奇门元信息展示和异常提示 | 改后端路由 |
| E Regression / Eval | backend 测试、web 测试、必要 eval dataset | 最小回归测试集 | 大规模样本堆砌 |
| Reviewer | 全局只读审查 | 边界违约清单、返工项 | 直接写实现 |

执行顺序：

1. A 只读分析。
2. B 在读取 A 的合同结论后只读分析；不得与 A 并行猜接口。
3. 主 agent 审核 A/B，冻结本文字段、取值、调用顺序和测试映射。
4. A 实现 route/policy/ExecutionPlan contract；C 不得在此阶段改写字段语义。
5. B 实现 Case、Prefill、ToolRunner、资产 owner 校验和 specialist 最小上下文视图。
6. C 实现 Manager 合成、role-aware dispatch、specialist 配置 / prompt 白名单和 health guard 接线。
7. D 修前端 payload 展示与复制 Markdown。
8. E 补回归测试；Reviewer 只读审查全 diff。
9. 主 agent 最终审核、跑验证，并只在事实发生变化后更新 `PROGRESS.md`、`docs/architecture.md` 或 `docs/acceptance-criteria.md`。

主 agent 审核清单：

- Manager 是否仍是 runtime conversation owner。
- RouteAdvisor 是否只做粗路由和准入。
- ExecutionPlan 是否是唯一执行合同。
- Prefill 是否负责确定性事实。
- specialist 是否没有重排已有盘。
- 奇门是否只在 Case 下创建问事盘。
- 出生时间是否没有进入 qimen_dunjia 参数。
- renderer 是否没有承担语义路由。
- cheap gate 是否没有扩大成第二路由器。
- 是否没有新增六爻 / 梅花。
- 是否没有专项 case 分支。
- 每个验收项是否有单一可证伪测试。

## 分阶段实施计划

### Phase 0：只读确认

修改点：

- 不改代码。
- 读取 PROGRESS.md、docs/architecture.md、docs/acceptance-criteria.md。
- 使用 CodeGraph 定位 Manager、ExecutionPlan、ArtifactRequirement、Prefill、QimenChart、specialist config。

验收标准：

- 主 agent 能列出本次会改和不会改的文件。
- 每个目标文件职责边界清楚。

风险：

- 直接改 prompt 会掩盖合同问题。
- 忽略 dirty worktree 会覆盖已有未提交修改。

### Phase 1：任务分类与路由硬纠偏

修改点：

- 在 `ApprovedRoute` 和 `ExecutionSnapshot` 增加规范 `ConsultationKind`；`task_intent` 只保留子任务语义。
- period_fortune 强制 bazi primary + ziwei secondary + qimen none。
- event_question 强制 qimen primary + profile none。
- health_risk 强制 bazi primary + ziwei secondary + `SafetyProfile=health_observation`。
- natal_chart 按显式方法，不自动三域。
- 具体事件优先于方法词；方法与具体事件冲突时进入现有澄清路径，不添加新的分类值。
- 把旧的“今天运气怎么样默认 qimen primary”验收语义改成带上下文的具体事件/阶段运势用例，避免与本方案冲突。

验收标准：

- “本月运势如何”：`PrimaryDomain=bazi`、`SecondaryDomains` 包含 `ziwei`、`QimenMode=none`，且不创建 Case、不调用 qimen。
- “这个面试能不能成”：`PrimaryDomain=qimen`、`QimenMode=primary`、`ProfileRequirement=none`。
- “最近身体健康如何”：`PrimaryDomain=bazi`、`SecondaryDomains` 包含 `ziwei`、`SafetyProfile=health_observation`。
- “用八字看看最近身体”：仍是 `health_risk`，不能因为出现“八字”而降为普通 `natal_chart`，最终必须经过健康免责声明。
- “分析八字”：DomainSteps 只有 bazi primary，不自动加紫微或奇门。
- “用八字分析这个面试能不能成”：进入澄清，不静默改成 natal_chart 或 qimen supplement。
- Snapshot 或 Run Inspector 能看到 `ConsultationKind`、DomainSteps 角色和安全 profile。

风险：

- `ApprovedRoute`/snapshot schema 变化会带来测试更新；不得退回隐式 `task_intent` 方案。

### Phase 2：资产合同与 Prefill

修改点：

- 扩展 ArtifactRequirement 支持动态范围和 purpose。
- 不新增持久化 `bazi_dynamic` / `ziwei_dynamic` kind；把动态范围、目标时点和 capability status 放入本轮 Prefill/trace 合同。
- 把 `qimen_chart` 的新写入迁移到 `qimen_case_chart`，并强制 Case owner。
- 在 turn 入口捕获一次 `TurnContext`；Manager 是唯一 Case 创建 owner，按 `TurnID` 固定 `QuestionTime`、`TargetAt` 和 `CaseID`，传给 prefillQimen。
- 让 `artifact_resolver` 和 `selectArtifactRequirements` 保持无副作用；forced route / plan rebuild 复用同一 `TurnContext`、`CaseID` 和 requirement，消除 `ActiveChart` 旧盘污染。
- graph-local 保存 forced route 生成的有效 Plan，agent 节点消费它，不重复 BuildExecutionPlan。
- Qimen 工具调用对外只传 `question_time`；适配层再转换为 qimen-go 所需的 question-time date fields。
- 生产 prefill 缺少 `TargetAt` / `QuestionTime` 时返回合同缺口，不回退系统当前时间。

验收标准：

- qimen_case_chart.OwnerRef.Kind=case。
- `qimen_case_chart.OwnerRef.ID == CaseID`，且 `Case.EventTime == TurnContext.QuestionTime`。
- 连续两次新的 event_question 创建两个不同 Case；同一轮重建 Plan 不创建第二个 Case。
- qimen_dunjia 调用参数只有 question_time，不含 gender、birthplace 或出生资料字段。
- Prefill 不会从上一个 Case 的 ActiveChart 复用本轮 qimen 盘。
- `selectArtifactRequirements` 不修改 SessionState；每个 requirement 的 owner、scope、target_at、purpose、input_refs 可断言。
- 缺必要资产时 specialist 前失败。

风险：

- 流月未完整实现时不要造假；必须产生可断言的 `DynamicFacts{scope:"liuyue", target_at, status}`，状态只能是 `unavailable` 或 `degraded`，并在最终回答说明缺口。

### Phase 3：Manager 综合输出

修改点：

- Manager 按 `DomainSteps.Role` 合成，support 失败只产生 degraded，不改变 primary。
- period_fortune 按“八字主线 -> 紫微复核 -> 建议 -> 免责声明”。
- event_question 按“奇门结论 -> 操作建议 -> 背景 -> 边界说明”。
- health_risk 加固定医疗免责声明和保守 Claim 合同。
- Qimen specialist 上下文去掉 profile/birth fields；Specialist ToolNames 移除所有确定性排盘/动态工具。
- final guard 按当前 ExecutionPlan 的精确 requirement 校验资产；不再只检查 `HasQimenResult` / `HasBaziResult` 等 legacy projection。
- primary/support 结果先形成 runtime 私有 StepOutcome，再交给 Manager compose；support 失败允许 degraded，primary 失败不得伪造。

验收标准：

- 多领域输出不是简单拼接。
- 冲突时有主次，不平均投票。
- support 缺失时仍保留 primary，并记录 support degraded。
- 健康类包含固定免责声明，不出现确定性诊断、病名、治疗或生死结论。
- specialist ToolNames 不包含 bazi_calc、bazi_liunian、qimen_dunjia、ziwei_calc、ziwei_liunian。
- qimen ToolContract 只接受一个必填 `question_time` 参数。

风险：

- 不要把这些规则塞进 renderer。

### Phase 4：Frontend 奇门展示

修改点：

- QimenChart 展示起局用途、起局时间、盘式、门神体系。
- 展示 Case/owner 元信息，来源只取结构化 payload。
- rotating_8 翻译为标准转盘 / 八门八神。
- 后端先校验符号体系，前端对 legacy/异常 payload 显示 warning。
- 复制 Markdown 补齐元信息。

验收标准：

- 用户能看出这是问事盘，不是出生盘。
- `pan_schema=rotating_8` 时不出现“中门 / 太常 / 勾陈 / 朱雀”；异常 payload 必须显示 warning。
- 复制 Markdown 包含 case、purpose、question_time、pan_schema、symbol_system。
- vue-tsc 和 build 通过。

风险：

- 只改文案不校验字段，会继续误导普通用户。

### Phase 5：最小回归

新增测试：

1. “本月运势如何”：`PrimaryDomain=bazi`、`SecondaryDomains` 包含 `ziwei`、`QimenMode=none`，无 qimen tool call 和新 Case。
2. “这个面试能不能成”：`PrimaryDomain=qimen`、`QimenMode=primary`、`ProfileRequirement=none`，生成 `qimen_case_chart`。
3. 两个新的 event_question 轮次得到两个不同 Case；同轮 forced route 重建仍只有一个 Case。
4. `qimen_case_chart.OwnerRef.Kind=case`，`OwnerRef.ID=CaseID`，`Case.EventTime=QuestionTime`。
5. qimen_dunjia 的 runtime 入参只有 `question_time`，出生资料不会进入 Qimen specialist context。
6. `pan_schema=rotating_8` 时不得出现“中门 / 太常 / 勾陈 / 朱雀”；前端和复制 Markdown 都保留异常提示。
7. Specialist ToolNames 不包含 bazi_calc、bazi_liunian、qimen_dunjia、ziwei_calc、ziwei_liunian。
8. 健康风险输出包含固定免责声明，不出现确定性医疗诊断。
9. 普通八字出生盘只走 bazi，不隐式加入紫微或奇门；显式紫微也不自动扩成三域。
10. qimen ToolContract 只声明必填 `question_time:string`，多余出生字段被拒绝或不进入工具。
11. 目标月份为流月但能力未实现时，输出 `DynamicFacts{status: unavailable|degraded}`，不得出现模型自推的流月事实。
12. 具体事件与“用八字 / 紫微命盘判断”同时出现时进入澄清，不新增隐式组合路由。

验证命令：

~~~bash
go test ./backend/internal/runtime -count=1
go test ./backend/internal/supervisor -count=1
go test ./backend/internal/tools/qimen -count=1
go test ./backend/internal/specialists/... -count=1
cd web && npx vue-tsc --noEmit && npm run build
~~~

风险较大时再跑：

~~~bash
go test ./backend/... -count=1
go build ./backend/cmd/server/
make regression
~~~

## 不做事项

- 不新增六爻 / 梅花。
- 不做多对象比较重构。
- 不把所有运势问题强制三域执行。
- 不扩大 cheap gate。
- 不让 qimen supplement 在本月运势里偷偷起局。
- 不让 specialist 拥有最终答复权。
- 不让 LLM 自推流月。
- 不靠 prompt 作为唯一防线。
- 不新增大而全 taxonomy。
- 不把健康问题写成医疗诊断。

## 完成标准

本任务完成必须同时满足：

- 四类分类和 `DomainSteps` 与本文件冻结表完全一致，且 Snapshot/Run Inspector 可观测。
- period_fortune 无 qimen tool call、无新 Case；event_question 创建同轮唯一 Case。
- `qimen_case_chart.OwnerRef.Kind=case`，且 owner、Case.EventTime、QuestionTime 三者一致。
- qimen runtime 入参只含 `question_time`；出生资料不进入 qimen tool 或 specialist context。
- `rotating_8` 的后端合同和前端展示均不把“中门 / 太常 / 勾陈 / 朱雀”当合法普通符号。
- specialist 不能调用确定性排盘/动态工具，也不能重排已有盘。
- Manager 以 primary/support 合成，support 缺失可降级但不能覆盖 primary。
- health_observation 输出含固定免责声明且没有医疗诊断性 Claim。
- 流月未实现时显式 `unavailable/degraded`，没有模型自推事实。
- 最小回归和前端构建通过；若实际改变架构事实，再同步 `PROGRESS.md`、`docs/architecture.md` 或 `docs/acceptance-criteria.md`。

## 可执行目标提示词

方案冻结后，把下面提示词交给实现 agent：

~~~text
你是资深 Agent 架构与 Go/Vue 全栈实现协作者。请在 /home/huang/workspace/suanming-agent 中实施 docs/recent-fortune-implementation-plan.md 的冻结方案。

必须遵守：
- AGENTS.md
- PROGRESS.md
- docs/architecture.md
- docs/acceptance-criteria.md
- docs/recent-fortune-implementation-plan.md

目标：
1. 个人阶段运势：八字主导、紫微复核、奇门不参与。
2. 具体事件问事：奇门按本轮提问时间创建新 Case 起问事盘，不使用出生资料。
3. 健康风险类：八字主导、紫微复核、必须免责声明，不做医疗诊断。
4. 出生盘分析：按用户明确方法走，不自动扩成三域。

执行方式：
1. 先派 subagent A Runtime Contract 做只读分析。
2. 再派 subagent B Artifacts / Prefill 做只读分析。
3. 主 agent 汇总 A/B，冻结最小接口后再允许代码修改。
4. 依次执行 Phase 1 到 Phase 5。
5. 每个实现 subagent 只改自己范围，Reviewer subagent 只读审查。
6. 主 agent 最终审核 diff、跑验证、必要时更新 PROGRESS.md 和 docs/acceptance-criteria.md。

冻结合同：
- 唯一分类字段是 ConsultationKind：period_fortune / event_question / health_risk / natal_chart。
- ExecutionPlan 必须有 SafetyProfile、DomainSteps、Requirements、TurnContext；主次只看 DomainSteps.Role。
- 具体事件优先于方法词；方法与具体事件冲突时进入现有澄清路径，不增加分类值。
- TurnContext.QuestionTime 在入口捕获一次；Manager 是唯一 Case 创建 owner；event_question 的 Case.EventTime、qimen_case_chart owner 和 QuestionTime 必须一致。
- 新写入奇门资产只用 qimen_case_chart；旧 qimen_chart 只作迁移读取别名。
- qimen_dunjia ToolContract / runtime 入参只含必填 question_time；不得传 gender、birthplace 或任何出生字段。Qimen specialist context 也不得含 profile、出生历史、摘要或完整 session。
- 本期不新增持久化 bazi_dynamic / ziwei_dynamic；动态事实由 runtime-owned 确定性路径按 Scope/TargetAt 准备，流月未实现必须 degraded/unavailable。
- `selectArtifactRequirements` 无副作用；forced route 只重建一次有效 Plan，agent 消费 graph-local Plan，不重复创建 Case。
- 健康 SafetyProfile 固定附加“命理仅供参考，不能替代医学诊断；如有不适请及时就医”。

Subagent 分工：
- A Runtime Contract：只读分析上述 runtime owner；实现阶段负责 contracts、route/policy、ExecutionPlan 和 artifact resolver，不改 manager 合成。
- B Artifacts / Prefill：executor、specialist_runner、state/assets、bazi/ziwei/qimen tools，负责 Case / Prefill / ToolContract 和最小上下文视图。
- C Manager Synthesis：manager、execution_dispatch / execution_plan_runner、agent_route、specialists、prompts，负责 role-aware compose 和 health guard。
- D Frontend：QimenChart、chat types、assistantTurn、前端测试。
- E Regression / Eval：runtime/supervisor/qimen/specialists/web 最小测试集。
- Reviewer：只读审查边界违约。

硬性验收：
- “本月运势如何”：PrimaryDomain=bazi，SecondaryDomains 包含 ziwei，QimenMode=none，且无 qimen tool call/新 Case。
- “这个面试能不能成”：PrimaryDomain=qimen，QimenMode=primary，ProfileRequirement=none。
- qimen_case_chart.OwnerRef.Kind=case，OwnerRef.ID=CaseID，Case.EventTime=QuestionTime。
- 连续两次新奇门问事创建两个不同 Case，同轮 Plan 重建不重复创建。
- qimen_dunjia runtime 入参只有 question_time；出生资料不进入 Qimen specialist context，ToolContract 也不接受多余出生字段。
- pan_schema=rotating_8 时不得出现“中门 / 太常 / 勾陈 / 朱雀”，异常必须可见。
- specialist ToolNames 不包含 bazi_calc、bazi_liunian、qimen_dunjia、ziwei_calc、ziwei_liunian。
- 健康风险输出包含固定免责声明，不得出现确定性医疗诊断。
- 普通八字出生盘不隐式加入紫微或奇门；显式紫微不自动扩成三域。

禁止：
- 不新增六爻 / 梅花。
- 不写专项 case 分支。
- 不让 renderer 做语义路由。
- 不扩大 cheap gate 成第二路由器。
- 不让 LLM 自己推干支、流年、流月或奇门局。
- 不覆盖用户已有未提交改动。

最小验证：
go test ./backend/internal/runtime -count=1
go test ./backend/internal/supervisor -count=1
go test ./backend/internal/tools/qimen -count=1
go test ./backend/internal/specialists/... -count=1
cd web && npx vue-tsc --noEmit && npm run build

风险较大时追加：
go test ./backend/... -count=1
go build ./backend/cmd/server/
make regression

最终回复：
1. 改了什么：Runtime / Tools / Frontend / Tests。
2. 四类用户问题现在如何路由。
3. 实际跑过的验证命令和结果。
4. 未完成事项，例如流月是否只是合同占位。
5. 剩余风险和下一步。
~~~
