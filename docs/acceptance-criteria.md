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

### AC-1.3 奇门主链触发
- **Given** 用户询问“今天运气怎么样”且没有出生信息
- **When** RouteAdvisor 处理消息
- **Then** 产出 `ApprovedRoute{PrimaryDomain: "qimen", PolicyHints: {QimenMode: "primary"}}`，不追问出生信息

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
- **Given** `QimenMode=primary` 且当前 `Case` 没有兼容的 `qimen_chart`
- **When** manager-owned runtime dispatch 到奇门 specialist runner
- **Then** 调用 `qimen_dunjia`，结果写入该 Case 的 `DomainAsset`；`SessionState.QimenResult` 只作为活动资产兼容投影

### AC-2.4 manager-owned 受控调度
- **Given** `ApprovedRoute` 只批准八字域
- **When** manager 构建 `ExecutionPlan` 并执行 dispatch
- **Then** 只会选择 `bazi` runner，不会触发 `qimen` / `ziwei` runner

### AC-2.5 多域 dispatch 仍由 manager 收口
- **Given** `ExecutionPlan.Domains=["bazi","ziwei"]`
- **When** runtime 完成两个领域的 runner 执行
- **Then** specialist 结果先聚合，再由 manager 统一 compose 最终回复

### AC-2.6 RequiredArtifacts 前置校验
- **Given** `ExecutionPlan.Requirements` 包含归属指定 Case 的 `qimen_chart`
- **When** prefill 结束后没有该 Case 的兼容资产，或仅存在其他对象 / Case 的盘
- **Then** dispatch 在进入 `qimen` runner 前直接报错，不等待 final guard 才发现缺盘

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
- **Then** `QimenChart` 按后天八卦方位排列九宫格

### AC-5.5 Trace 面板
- **Given** 一轮对话完成
- **When** SSE 推送 `done` 事件
- **Then** `TracePanel` 展示处理过程摘要，`KnowledgeSourceCard` 可展开折叠

### AC-5.6 上下文卡默认折叠
- **Given** 助手回答包含 thinking 和知识来源
- **When** 前端渲染 AssistantTurn
- **Then** “思考过程”和“知识来源”卡片默认折叠

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
