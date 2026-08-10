# 验收标准

> 按当前架构 `thin supervisor + manager-owned runtime orchestrationGraph + bounded specialist runners` 组织的验收用例。覆盖路由、执行、知识检索、降级、前端、上下文六大类。

## AC-1：路由审批

### AC-1.1 首轮出生信息识别
- **Given** 用户首次输入包含出生时间，例如“1990年5月5日14:30出生”
- **When** RouteAdvisor 处理消息
- **Then** 产出 `ApprovedRoute{ConversationIntent: "consult", PrimaryDomain: "bazi", TaskIntent: "collect_profile"}`，且 `Slots.Profile` 含提取出的出生信息

### AC-1.2 八字追问复用命盘
- **Given** 会话中已有八字命盘
- **When** 用户追问“我的财运怎么样”
- **Then** 产出 `ApprovedRoute{PrimaryDomain: "bazi", TaskIntent: "interpret_chart"}`，不重新走 collect_profile

### AC-1.3 无出生资料的即时问事
- **Given** 用户询问“今天运气怎么样”且没有出生信息，且当前没有明确的本月/今年阶段运势上下文
- **When** RouteAdvisor 处理消息
- **Then** 产出 `ApprovedRoute{ConsultationKind: "event_question", PrimaryDomain: "qimen", PolicyHints: {QimenMode: "primary", ProfileRequirement: "none"}}`，不追问出生信息

### AC-1.7 近期运势综合分类
- **Given** 用户分别询问“本月运势如何”“这个面试能不能成”“最近身体健康如何”“分析八字”
- **When** RouteAdvisor 和 Policy Gate 完成本轮路由
- **Then** 四类路线分别为 `period_fortune`（bazi primary + ziwei support + qimen none）、`event_question`（qimen primary + profile none）、`health_risk`（bazi primary + ziwei support + health_observation）和 `natal_chart`（仅用户明确的方法）
- **And** “用八字看看最近身体”仍归入 `health_risk`，不因方法词改成普通出生盘
- **And** `collect_profile`、`amend_profile` 和澄清短路不伪装成四类咨询，也不创建 specialist plan

### AC-1.4 误路由纠偏
- **Given** 用户首轮消息包含出生时间，但 RouteAdvisor 误判为 interpret_chart
- **When** Policy Gate 检测到出生信息
- **Then** 强制改写为 collect_profile

### AC-1.5 低置信度强制澄清
- **Given** RouteAdvisor 返回低置信度
- **When** Policy Gate 处理
- **Then** `NeedsClarification=true`，主链进入澄清短路

### AC-1.6 未注册领域白名单降级
- **Given** RouteAdvisor 产出未注册领域
- **When** Policy Gate 处理
- **Then** 主域降级为 `bazi`

## AC-2：Manager-owned 执行

### AC-2.1 八字 runner 排盘
- **Given** 会话中有完整出生信息
- **When** manager-owned runtime dispatch 到八字 specialist runner
- **Then** 自动调用 `bazi_calc` 排盘，结果写入当前 `Subject + ProfileRevision` 的八字资产；`SessionState.BaziResult` 只作为活动资产兼容投影

### AC-2.2 八字 runner 复用命盘
- **Given** 当前对象与当前资料版本已有兼容的八字资产
- **When** manager-owned runtime 再次 dispatch 到八字 specialist runner
- **Then** 不重新调用 `bazi_calc`，直接复用该精确资产；其他对象或旧资料版本的盘不得满足该条件

### AC-2.3 奇门 runner 排盘
- **Given** `ConsultationKind=event_question` 且当前 `Case` 没有兼容的 `qimen_case_chart`
- **When** manager-owned runtime dispatch 到奇门 specialist runner
- **Then** 调用 `qimen_dunjia`，结果写入 OwnerRef.Kind 为 `case` 的 `DomainAsset`；`SessionState.QimenResult` 只作为活动资产兼容投影
- **And** `Case.EventTime`、payload.question_time 和本轮 `TurnContext.QuestionTime` 相等

### AC-2.4 manager-owned 受控调度
- **Given** `ApprovedRoute` 只批准八字域
- **When** manager 构建 `ExecutionPlan` 并执行 dispatch
- **Then** 只会选择 `bazi` runner，不会触发 `qimen` / `ziwei` runner

### AC-2.5 多域 dispatch 仍由 manager 收口
- **Given** `ExecutionPlan.Domains=["bazi","ziwei"]`
- **When** runtime 完成两个领域的 runner 执行
- **Then** specialist 结果先聚合，再由 manager 统一 compose 最终回复

### AC-2.6 RequiredArtifacts 前置校验
- **Given** `ExecutionPlan.Requirements` 包含归属指定 Case 的 `qimen_case_chart`
- **When** prefill 结束后没有该 Case 的兼容资产，或仅存在其他对象 / Case 的盘
- **Then** dispatch 在进入 `qimen` runner 前直接报错，不等待 final guard 才发现缺盘

### AC-2.11 动态事实能力状态
- **Given** 用户目标范围为流年或尚未实现的流月
- **When** Prefill 完成动态准备
- **Then** runtime 产出包含 `scope`、`target_at`、`status` 和结构化 `facts` 的动态事实对象
- **And** 流月未实现时 `status` 只能是 `unavailable` 或 `degraded`，最终回答明确说明缺口，不由模型补算流月
- **And** 只有 `ExecutionPlan.Route.Slots.TimeScope` 明确存在时，Manager 才能把 `unavailable/degraded` 缺口追加到最终回答；没有明确时间范围的静态或结构追问不得追加流年/流月缺口说明

### AC-2.7 排盘结果自动推送卡片
- **Given** `bazi_calc` 返回排盘结果
- **When** runtime specialist event bridge 处理工具事件
- **Then** SSE 推送 `{type: "component", data: {type: "bazi-chart", payload: {...}}}`

### AC-2.8 多对象与资料修订不覆盖
- **Given** 同一会话先后为自己和孩子排盘，或用户修正出生时刻
- **When** Manager 解析本轮目标
- **Then** 只切换 `ActiveFocus` 到对应 Subject / ProfileRevision；旧盘仍可追溯，不能静默被覆盖或复用

### AC-2.9 大运确定性合同
- **Given** 完整出生时分和性别
- **When** `bazi_calc` 生成命盘、`bazi_liunian` 判断当前大运
- **Then** 输出出生分钟、顺逆、顺逆依据、起运时刻、每步日期边界；流年在交运日之前不得仅因虚岁跨年就提前切运
- **And** `dayun_analyzed` 必须保留每步日期边界；`current_dayun` 缺失时仅可按日期边界回补，无法定位则明确显示未识别，不能猜测或重复“当前”前缀

### AC-2.10 八字规则治理
- **Given** 八字工具输出四柱、藏干、旺衰、格局和大运
- **When** runtime 构建静态或动态综合输入
- **Then** 排盘、藏干层级、透干和标准冲合刑害作为 `chart_facts`；旺衰、用忌、格局成败与调候必须携带 method/profile，不能被当作同一层确定事实
- **And** `official_visibility.hidden` 非空时不得写成“无官星”；组合检测只能输出候选或受阻事实，不得直接授予“富格/贵格/成立”
- **And** `balance_status=待选定流派裁断` 时，大运仍输出十神、顺逆、交运边界和关系事实，但 `quality` / `quality_base` 必须为“待裁定”
- **And** 动态综合不得使用未在关系事实中声明的暗合、相破，也不得从命理关系推导官非或具体疾病
- **And** 冲、刑、害、合、会只能作为关系触发面；不得按固定权重自动汇总成“偏吉 / 偏压 / 承压明显”大运结论

### AC-2.12 BaZi 确定性裁断 V2
- **Given** 用户请求完整八字解读
- **When** runtime 执行八字内部图
- **Then** trace 记录 `bazi.loop_step`、`bazi.next_action`、`bazi.termination_reason`；实际路径由 `decide_next` 按 state 选择 `analysis_plan`、`evidence_action`、`static_judgment`、`dynamic_judgment`、`repair`、`recover_facts` 或 `render`，编译上限为 24 步
- **And** `contract_check` 只校验并写入 failure；`fact_conflict`、`method_contract` 不调用模型 repair，允许 repair 的阶段最多一次
- **And** outer `orchestration` trace 记录 `orchestration.loop_step`、`orchestration.next_action`、`orchestration.termination_reason`，编译上限为 16 步；`final_guard` 在 Graph `Invoke` 后执行，最终 `text` 只发送一次
- **And** 2026 流年只引用 runtime 绑定的甲午运，不得引用 `dayun[0]`；完整大运目录只展示确定性事实
- **And** 本命层次固定为九级：核心命盘和主轴已成立但独立主证未闭合时，必须输出 `provisional` 的第 3-6 级；清浊、病药、救应、破格风险和何知章五项证据齐全时才可输出 `rated` 的第 1-9 级；只有核心事实或主轴无法建立时才允许 `withheld` 的 0 级
- **And** 当前大运只能输出 `repair|assist|maintain|disturb|suppress` 承接状态，不得改写本命基础等级
- **And** 官星未透时，静态原局风险必须为 `withheld`；岁运风险仅能在当前大运和流年关系已绑定时表达为条件风险
- **And** 用户可见依据不得包含 `dayun[0].gan_zhi` 等内部路径，主轴只在总览结论中出现一次

### AC-2.13 八字追问直接回答边界
- **Given** 会话中已有通过静态合同校验的八字结论，用户提出不带明确时间范围的普通结构追问
- **When** 静态结果没有专用 `TopicDirectAnswer`
- **Then** 最终 `直接回答` 依次回退到已验证的 `PatternOutcome`、`MainAxis` 或 `TopicFocusAnswer`，不得因为专用字段为空而输出“本轮未形成这次追问的直接裁断”
- **And** `timing_reason` 动态追问仍优先使用当前动态趋势，不复用普通结构追问的静态回退语义
- **And** 本轮没有明确 `TimeScope` 时，即使 `DynamicFacts.status` 为 `unavailable/degraded`，最终文本也不得追加流年/流月资料缺口

## AC-3：知识检索

### AC-3.1 基本检索
- **Given** specialist 调用 `knowledge_search`
- **When** MCP 知识库返回结果
- **Then** 返回 `passages` 列表，每项包含 `content` 和 `source`

### AC-3.2 检索降级
- **Given** MCP search 端点不可用
- **When** `knowledge_search` 调用失败
- **Then** 自动降级到 `/api/wiki/search` REST API

### AC-3.3 检索失败不阻塞
- **Given** 知识库完全不可用
- **When** `knowledge_search` 调用
- **Then** 返回空 `passages` 且 `fallback=true`，不直接抛出错误终止整轮

### AC-3.4 检索结果注入指令
- **Given** `knowledge_search` 返回 passages
- **When** specialist 生成最终回复
- **Then** passages 作为参考资料注入执行上下文，要求标注来源

## AC-4：降级与容错

### AC-4.1 Supervisor ADK 自纠错
- **Given** ADK route engine 的结构化输出校验失败
- **When** route engine 检测到错误
- **Then** 提取反馈并以同一消息做一次本地重试

### AC-4.2 Route 降级链
- **Given** ADK structured route 不可用
- **When** supervisor 回退
- **Then** 进入 `textDecide -> fallbackExtract -> safeFallback` 降级链

### AC-4.3 失败不污染旧状态
- **Given** 排盘过程中发生错误
- **When** 错误被捕获
- **Then** 已存在的 `BaziResult / QimenResult / ZiWeiResult` 不会被覆盖为坏值

### AC-4.4 Final guard 只做最终保险
- **Given** qimen 或 ziwei 主链缺少 artifact
- **When** 错误未被更早的不变量拦住而到达 final guard
- **Then** final guard 阻断用户可见结论并留下可诊断 trace

## AC-5：SSE 与前端

### AC-5.1 结构化事件流
- **Given** 一轮完整对话
- **When** runtime 从 preflight 执行到 done
- **Then** SSE 流中包含 `thinking / tool_call / component / text / done` 事件

### AC-5.2 流式回答
- **Given** specialist 生成回答
- **When** LLM 以 streaming 模式输出
- **Then** 前端逐步渲染文本

### AC-5.3 八字命盘卡渲染
- **Given** SSE 推送 `bazi-chart` component 事件
- **When** 前端收到事件
- **Then** `BaziChartCard` 正确渲染四柱、五行统计、神煞、大运时间轴

### AC-5.4 奇门盘渲染
- **Given** SSE 推送 `qimen-chart` component 事件
- **When** 前端收到事件
- **Then** `QimenChart` 按后天八卦方位排列九宫格，并展示结构化的 `case_id`、`purpose`、`question_time`、`time_source`、`pan_schema` 和 `symbol_system`
- **And** `pan_schema=rotating_8` 的异常符号不被静默删除，复制 Markdown 也保留 warning

### AC-5.5 Run Inspector 排障面板
- **Given** 一轮对话完成
- **When** SSE 推送 `done` 事件
- **Then** `RunInspector` 展示 trace_id、诊断结论、agent 链路、span tree 和 span detail，`KnowledgeSourceCard` 可展开折叠
- **And** 本地 debug 模式下可按 trace_id 懒加载完整 `TurnTrace`，默认折叠敏感字段

### AC-5.6 上下文卡默认折叠
- **Given** 助手回答包含 thinking 和知识来源
- **When** 前端渲染 AssistantTurn
- **Then** “思考过程”和“知识来源”卡片默认折叠

## AC-7：结构化输出合同

### AC-7.1 Draft-07 Schema 单一来源
- **Given** 四个 BaZi JSON Mode 节点或 Supervisor text fallback 需要结构化输出
- **When** 构建 prompt 或校验模型原始 content
- **Then** 两者都读取仓库内同一份嵌入式 Draft-07 JSON Schema 原文；V2 的 analysis、evidence、static、dynamic 节点各有独立 `bazi-*.schema.json` 文件，Schema 由 gojsonschema 校验，不能由 DTO 反射或 prompt 字段表生成旁路
- **And** registry 不注册已删除的 canonical/audit Schema，活跃节点只能使用 analysis、evidence、static、dynamic 四份 Schema
- **And** 当前传输仍是 DeepSeek Chat Completions response_format: {"type":"json_object"}，不是 provider-native Strict JSON Schema

### AC-7.2 原始 JSON 严格拒绝
- **Given** 模型返回空内容、Markdown fence、缺少 required、错误 type、非法 enum、unknown field 或 trailing JSON
- **When** 结构化输出进入 Go client
- **Then** 统一以 schema_error 拒绝，不进入 DTO、renderer 或成功路径
- **And** json.Decoder.DisallowUnknownFields() 后第二次 Decode 必须得到 io.EOF

### AC-7.3 引用 catalog 与 repair 边界
- **Given** 模型输出未声明 fact_ref、relation_ref 或 claim_ref
- **When** generic runtime catalog 校验
- **Then** 返回 undeclared_fact_claim；canonical synthesis 最多按 schema repair 重跑一次，并携带当轮允许 ID
- **And** fact_value_mismatch、方法合同冲突不通过改措辞重试；“丙戌火局”不依赖专项 validator

### AC-7.4 Transport 与业务 repair 分离
- **Given** transport transient 或 schema/reference contract failure
- **When** runtime 记录重试与 repair
- **Then** transport attempt 与 schema repair attempt 独立计数、独立 trace；失败后按 recovery policy 降级或硬失败

### AC-7.5 Supervisor 与 SSE 边界不变
- **Given** Supervisor ADK output tool 和 text fallback 分别返回结构化 tool output 或文本 JSON
- **When** 进行路由决策和 SSE 推送
- **Then** InferTool / ReturnDirectly 语义保持；text fallback 不宽松接受 fence/unknown/trailing JSON；SSE 事件名仍为 thinking/tool_call/component/text/done

### AC-7.6 节点输出职责不重叠
- **Given** BaZi static、dynamic JSON Mode 节点
- **When** 生成或校验各自 DTO
- **Then** static 固定输出主轴、强弱、调候、格局四个 claim 与结构化九级层次；dynamic 只输出 runtime 已绑定当前大运及流年，不输出完整大运吉凶标签
- **And** static/dynamic 的引用失败只重跑其所属节点，动态不得重跑静态或 canonical；调候 verdict 不以自然语言短语表作第二份合同

## AC-6：上下文工程

### AC-6.1 滚动摘要
- **Given** 会话超过 8 轮
- **When** 第 9 轮触发
- **Then** 前 4 轮写入 `RunningSummary`，保留最近 4 轮原文

### AC-6.2 历史上下文注入
- **Given** 会话包含 `RunningSummary` 和 `RecentTurns`
- **When** specialist 执行构建输入
- **Then** 执行上下文包含历史摘要块和最近对话块

### AC-6.3 跨轮状态不串域
- **Given** 上一轮涉及奇门
- **When** 下一轮只问纯八字问题
- **Then** 不自动触发奇门，也不弹出奇门盘

### AC-6.4 对象歧义只在必要时澄清
- **Given** 会话中已有自己和孩子两个 Subject
- **When** 用户说“他今年怎么样”且 RouteAdvisor 无法提供唯一对象
- **Then** Manager 返回对象澄清，不进入 specialist；单一 Subject 会话中的代词继续复用当前对象

### AC-6.5 follow-up 解读按命盘来源复用
- **Given** 自己的八字已有完成解读，之后切换到孩子的八字
- **When** 用户继续追问孩子
- **Then** 不得复用自己的 `InterpretationAsset`；切回自己的同一命盘才可复用其解读摘要

## Smoke Regression Coverage

官方 smoke suite 当前先覆盖本地可稳定运行的最小主链行为：

- conservative clarification fallback
- route-decision emission
- degraded follow-up / explanation / retrieval / qimen prompts do not crash
- successful stream completion
- 八字普通结构追问覆盖静态直接回答回退和无时间范围时不追加动态资料缺口：`TestRunFinalWriter_TopicFallbackUsesStaticConclusion`、`TestManager_DynamicFactsNoticeRequiresExplicitTimeScope`
- 多对象、资料修订、解读来源与奇门 Case 隔离：`TestExecutionPlan_SubjectAssetConversationRegression`

## 验证命令

```bash
go test ./backend/... -v
go build ./backend/cmd/server/
python3 -m unittest eval/runner/test_run_langfuse_eval.py -v
cd web && npm run test:unit
cd web && npx vue-tsc --noEmit
cd web && npm run build
```
