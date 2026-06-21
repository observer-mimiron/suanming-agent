# 验收标准

> 按当前架构（Supervisor Agent + AgentAsTool + Specialist Agent）组织的验收用例。覆盖路由、执行、知识检索、降级、前端五大类。

## AC-1：路由审批

### AC-1.1 首轮出生信息识别
- **Given** 用户首次输入含出生时间（如"1990年5月15日 14:30 出生"）
- **When** Supervisor 处理消息
- **Then** `ApprovedRoute{ConversationIntent: "consult", PrimaryDomain: "bazi", TaskIntent: "collect_profile"}`，Slots.Profile 含提取的出生信息

### AC-1.2 八字追问复用命盘
- **Given** 会话中已有排好的八字命盘
- **When** 用户追问"我的财运怎么样"
- **Then** `ApprovedRoute{TaskIntent: "interpret_chart", PrimaryDomain: "bazi"}`，不触发 collect_profile

### AC-1.3 奇门主链触发
- **Given** 用户问"今天运气怎么样"且无出生信息
- **When** Supervisor 处理
- **Then** `ApprovedRoute{PrimaryDomain: "qimen", PolicyHints: {QimenMode: "primary"}}`，不追问出生信息

### AC-1.4 误路由纠偏
- **Given** 用户首轮消息含出生时间但 Supervisor 误判为 interpret_chart
- **When** Go 侧确定性纠偏检测到消息含出生时间
- **Then** 强制回退到 collect_profile

### AC-1.5 低置信度强制澄清
- **Given** Supervisor 返回 Confidence < 阈值
- **When** Policy Gate 处理
- **Then** NeedsClarification=true，强制追问

### AC-1.6 策略门白名单
- **Given** Supervisor 路由到未注册的领域
- **When** Policy Gate 处理
- **Then** 降级为 bazi

### AC-1.7 显式术数 obey
- **Given** 用户明确说"用紫微斗数看婚姻"或"用奇门看今天适不适合谈合作"
- **When** `normalizeApprovedRoute` 处理已批准路由
- **Then** `PrimaryDomain` 必须分别落到 `ziwei` / `qimen`，不被通用领域默认值覆盖

## AC-2：领域专家执行

### AC-2.1 八字 Specialist 排盘
- **Given** 会话中有完整出生信息
- **When** 八字 specialist 被 Supervisor Agent 调用
- **Then** 自动调用 bazi_calc 排盘，结果写回 SessionState.BaziResult

### AC-2.2 八字 Specialist 复用命盘
- **Given** SessionState 中已有 BaziResult
- **When** 八字 specialist 被调用
- **Then** 不重新调用 bazi_calc，直接使用已有命盘

### AC-2.3 奇门 Specialist 排盘
- **Given** QimenMode=primary 且 QimenResult 为空
- **When** 奇门 specialist 被调用
- **Then** 调用 qimen_dunjia 排盘，结果写回 SessionState.QimenResult

### AC-2.4 AgentAsTool 受控调度
- **Given** ApprovedRoute 只批准 bazi 域
- **When** Supervisor Agent 执行
- **Then** 只能调用 bazi_specialist，不可调用 qimen/ziwei specialist

### AC-2.5 Agent 事件桥接
- **Given** Specialist Agent 产出回答文本
- **When** agentEventBridge 处理 ADK event
- **Then** SSE 推送 `{type: "text", data: {content: "..."}}`

### AC-2.6 排盘结果自动推送卡牌
- **Given** bazi_calc 返回排盘结果
- **When** agentEventBridge 处理 Tool event
- **Then** SSE 推送 `{type: "component", data: {type: "bazi-chart", payload: {...}}}`

### AC-2.7 紫微主链只发一次命盘卡牌
- **Given** `PrimaryDomain=ziwei` 且本轮需要排紫微命盘
- **When** specialist 真正调用 `ziwei_calc`
- **Then** 只推送 1 次 `ziwei-chart` component，不允许 prefill 和 tool result 各发一次

## AC-3：知识检索

### AC-3.1 基本检索
- **Given** 八字 specialist 调用 knowledge_search with query
- **When** MCP 知识库返回结果
- **Then** 返回 Passage 列表，每项含 content 和 source

### AC-3.2 检索降级
- **Given** MCP search 端点不可用
- **When** knowledge_search 调用失败
- **Then** 自动降级到 /api/wiki/search REST API

### AC-3.3 检索失败不阻塞
- **Given** 知识库完全不可用
- **When** knowledge_search 调用
- **Then** 返回空 passages + fallback=true，不抛 error

### AC-3.4 检索结果注入指令
- **Given** knowledge_search 返回 passages
- **When** Specialist Agent 生成最终回答
- **Then** passages 以"参考资料"块注入 instruction，要求标注出处

## AC-4：降级与容错

### AC-4.1 Supervisor ADK 自纠正
- **Given** ADK route engine 产出 output tool 校验失败
- **When** route engine 检测到校验错误
- **Then** 抽取反馈并以同一消息做一次本地重试

### AC-4.2 Supervisor textDecide 降级
- **Given** ADK structured route 不可用
- **When** route engine 回退
- **Then** 走 textDecide -> fallbackExtract -> safeFallback 三层降级链

### AC-4.3 LangGraph 降级
- **Given** LangGraph 推理服务不可用
- **When** Go 后端检测到连接失败
- **Then** 跳过推理层，直接 llm_generate

### AC-4.4 失败不污染旧状态
- **Given** 排盘过程中发生错误
- **When** 错误被捕获
- **Then** 已存在的 BaziResult / QimenResult 不被覆盖

### AC-4.5 奇门主链未起盘不得给结论
- **Given** `ApprovedRoute{PrimaryDomain: "qimen"}` 进入 runtime 主路径
- **When** 本轮没有产生 `QimenResult`
- **Then** runtime 必须拦截最终奇门结论输出，而不是返回伪奇门回答

### AC-4.6 紫微主链未起盘不得给结论
- **Given** `ApprovedRoute{PrimaryDomain: "ziwei"}` 进入 runtime 主路径
- **When** 本轮没有产生 `ZiWeiResult`
- **Then** runtime 必须拦截最终紫微结论输出，而不是返回伪紫微回答

## AC-5：SSE 与前端

### AC-5.1 5 种事件类型
- **Given** 一轮完整对话
- **When** 从 Supervisor 到 Specialist 执行完毕
- **Then** SSE 流中包含 thinking / tool_call / component / text / done 事件

### AC-5.2 流式回答
- **Given** Specialist Agent 生成回答
- **When** LLM 以 streaming 模式输出
- **Then** 前端逐步渲染文本

### AC-5.2b 最终文本先验收后输出
- **Given** Agent 主路径已生成最终回答文本
- **When** runtime 准备发出 `text` 事件
- **Then** 必须先经过 post-run contract gate 校验，再允许输出最终文本

### AC-5.3 八字命盘卡牌渲染
- **Given** SSE 推送 bazi-chart component 事件
- **When** 前端收到事件
- **Then** BaziChartCard 正确渲染四柱、五行统计、神煞、大运时间轴

### AC-5.4 奇门遁甲盘渲染
- **Given** SSE 推送 qimen-chart component 事件
- **When** 前端收到事件
- **Then** QimenChart 按后天八卦方位排列九宫格

### AC-5.5 Trace 面板
- **Given** 一轮对话完成
- **When** SSE 推送 done 事件
- **Then** TracePanel 显示处理过程摘要，KnowledgeSourceCard 可展开折叠

### AC-5.6 上下文折叠默认状态
- **Given** 助手回答包含思考链和知识来源
- **When** 前端渲染 AssistantTurn
- **Then** "思考链"和"知识来源依据资料"卡片默认折叠

## AC-6：上下文工程

### AC-6.1 滚动摘要
- **Given** 会话超过 8 轮
- **When** 第 9 轮触发
- **Then** 前 4 轮生成 RunningSummary，保留最近 4 轮原文

### AC-6.2 历史上下文注入
- **Given** 会话有 RunningSummary 和 RecentTurns
- **When** 回答生成时构建 prompt
- **Then** instruction 中包含历史摘要块和最近对话块

### AC-6.3 跨轮状态不污染
- **Given** 上一轮问答涉及奇门
- **When** 下一轮问纯八字问题
- **Then** 不自动触发奇门，不弹出奇门盘

## 验证命令

```bash
go test ./... -v                    # Go 全量测试
go build ./cmd/server/              # 编译检查
cd web && npm run test:unit         # 前端单元测试
cd web && npx vue-tsc --noEmit      # 前端类型检查
cd web && npm run build             # 前端生产构建
```
