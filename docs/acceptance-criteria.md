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
- **Then** 自动调用 `bazi_calc` 排盘，结果写回 `SessionState.BaziResult`

### AC-2.2 八字 runner 复用命盘
- **Given** `SessionState.BaziResult` 已存在
- **When** manager-owned runtime 再次 dispatch 到八字 specialist runner
- **Then** 不重新调用 `bazi_calc`，直接复用现有命盘

### AC-2.3 奇门 runner 排盘
- **Given** `QimenMode=primary` 且 `SessionState.QimenResult` 为空
- **When** manager-owned runtime dispatch 到奇门 specialist runner
- **Then** 调用 `qimen_dunjia`，结果写回 `SessionState.QimenResult`

### AC-2.4 manager-owned 受控调度
- **Given** `ApprovedRoute` 只批准八字域
- **When** manager 构建 `ExecutionPlan` 并执行 dispatch
- **Then** 只会选择 `bazi` runner，不会触发 `qimen` / `ziwei` runner

### AC-2.5 多域 dispatch 仍由 manager 收口
- **Given** `ExecutionPlan.Domains=["bazi","ziwei"]`
- **When** runtime 完成两个领域的 runner 执行
- **Then** specialist 结果先聚合，再由 manager 统一 compose 最终回复

### AC-2.6 RequiredArtifacts 前置校验
- **Given** `ExecutionPlan.RequiredArtifacts` 包含 `qimen_chart`
- **When** prefill 结束后 `SessionState.QimenResult` 仍为空
- **Then** dispatch 在进入 `qimen` runner 前直接报错，不等待 final guard 才发现缺盘

### AC-2.7 排盘结果自动推送卡片
- **Given** `bazi_calc` 返回排盘结果
- **When** runtime specialist event bridge 处理工具事件
- **Then** SSE 推送 `{type: "component", data: {type: "bazi-chart", payload: {...}}}`

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

## Smoke Regression Coverage

官方 smoke suite 当前先覆盖本地可稳定运行的最小主链行为：

- conservative clarification fallback
- route-decision emission
- degraded follow-up / explanation / retrieval / qimen prompts do not crash
- successful stream completion

## 验证命令

```bash
go test ./backend/... -v
go build ./backend/cmd/server/
cd web && npm run test:unit
cd web && npx vue-tsc --noEmit
cd web && npm run build
```
