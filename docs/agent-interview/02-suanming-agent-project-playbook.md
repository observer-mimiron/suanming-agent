# suanming-agent 项目讲解手册

结论：这个项目二面最该讲的不是“命理问答效果”，而是你把一个多轮、多工具、多领域系统收口成了可解释、可恢复、可观测、可回归的 Go runtime。

## 1. 先背这句总讲法

> 这是一个用 Go 做的 Agent runtime 工程化项目。  
> 我重点不是做一个会聊天的 demo，而是把 `RouteAdvisor -> Policy Gate -> Manager -> ExecutionPlan -> Prefill -> specialist runner(s) -> manager compose -> final guard -> SSE` 这条主链做稳。  
> 它的核心价值是把“理解用户意图”和“系统真正怎么执行”拆开，再用显式合同、前置产物、限域执行、最终保护、trace truth layer（运行真相层，负责给出可验证执行事实）把整条链闭起来。

## 2. 30 秒、2 分钟、5 分钟讲法

### 2.1 30 秒讲法

> 这是一个面向复杂咨询场景的 Agent 系统练习项目。  
> 我做的核心不是 prompt 调参，而是把 runtime 主链工程化。  
> 当前系统里，`Manager`（运行时主控器，负责统一收口）是唯一 conversation owner，`specialist runner`（限域执行器，只负责单领域任务）只负责产出领域结果，最终回复必须经过 manager compose 和 final guard。

### 2.2 2 分钟讲法

先讲三服务：

- 前端是 Vue 3 + SSE，负责聊天界面、结构化卡片、process/debug 视图
- 后端是 Go runtime，负责路由、状态、执行合同、工具调度、trace、SSE
- 知识库是独立服务，负责命理资料检索，不直接拥有最终解释权

再讲主链：

- `RouteAdvisor`（路由审批器，负责理解意图和领域）先给出 `ApprovedRoute`
- `Policy Gate`（策略门控层，负责确定性硬边界）做程序硬控
- `Manager.BuildExecutionPlan`（执行规划函数，负责把 route 变正式执行合同）决定 domains、required artifacts、follow-up mode
- `Prefill`（前置产物准备层，负责确定性补齐 artifact）先准备命盘等硬产物
- `specialist runner(s)` 执行单领域任务
- `manager compose` 统一组织最终答复
- `final guard`（最终保护层，负责最后一道合同校验）阻断越界输出

最后讲设计原则：

- `thin supervisor`
- `manager-owned runtime`
- `bounded specialists`

### 2.3 5 分钟讲法

#### 2.3.1 这个项目解决的不是“会不会答”，而是“能不能控”

我主要在解决 8 个问题：

1. route 只是理解结果，不能直接等于执行合同
2. LLM 不应该同时拥有路由权、执行权、最终答复权
3. 强结构化产物要程序先准备，不能全部等模型临场生成
4. 检索要受控，不能让回答链无限自由搜
5. follow-up 要能复用资产，不能每轮全重跑
6. 多领域并发执行时，结果要能并行跑、顺序收
7. 失败要能分类恢复，而不是一律 `agent_error`
8. 系统质量要靠 trace、测试、eval 证明，不靠“看起来答得还行”

#### 2.3.2 为什么不是单 Agent

如果把所有事情塞进一个 Agent，常见问题是：

- prompt 迅速变肥，策略和业务混在一起
- 路由、检索、工具、最终答复边界全糊掉
- 多轮追问时很容易漂移
- 出错后定位成本很高，不知道是 route、artifact、retrieval、compose 还是 guard 出问题

所以这里拆成路由审批层、运行时主控层、领域执行层和输出保护层。

#### 2.3.3 为什么 `Manager` 必须是唯一 owner

因为用户感知到的是一个连续对话，而不是几个 specialist 轮流发言。

这个仓库里，`Manager.BuildExecutionPlan` 会先决定：

- 跑哪些领域
- 每个领域运行前需要哪些 artifact
- 当前 follow-up 是直接答、复用资产，还是重跑 specialist

这件事必须放在 manager，而不是散落到每个 specialist 里。这样多轮对话、失败恢复、trace 归因才会稳定。

## 3. 一定要讲清的真实主链与代码锚点

### 3.1 主链长什么样

当前有效主链是：

`RouteAdvisor -> Policy Gate -> Manager -> ExecutionPlan -> Prefill -> specialist runner(s) -> manager compose -> final guard -> SSE`

### 3.2 每一层分别负责什么

| 环节 | 负责什么 | 真实代码锚点 |
|---|---|---|
| `RouteAdvisor` | 理解本轮问题，审批主领域和任务意图 | `RouteAdvisor` 接口：`backend/internal/orchestrator/types.go` |
| `Policy Gate` | 做策略修正、cheap gate 复用、硬边界控制 | `backend/internal/supervisor/` |
| `Manager` | 接管会话 owner、规划执行、统一收口 | `backend/internal/runtime/manager.go` |
| `ExecutionPlan` | 显式记录 domains、required artifacts、follow-up mode | `ExecutionPlan`：`backend/internal/runtime/execution_plan.go` |
| `orchestrationGraph`（运行编排图，负责串 preflight/prefill/agent/guard） | 组织每轮固定执行拓扑 | `backend/internal/runtime/orchestration_graph.go` |
| `Prefill` | 在 LLM 执行前补齐确定性 artifact | `prefill`：`backend/internal/runtime/executor.go` |
| `specialist runner(s)` | 限域执行单领域任务 | `runExecutionPlan` / `ADKSpecialistRunner.Run`：`backend/internal/runtime/execution_dispatch.go`、`backend/internal/runtime/specialist_runner.go` |
| `manager compose` | 组织最终用户可见回答 | `backend/internal/runtime/manager.go` |
| `final guard` | 校验主域 artifact 和输出边界 | `guardFinalAnswerWithTrace`：`backend/internal/runtime/observability.go` |
| `SSE` | 推送 thinking、tool_call、component、text、done | `emitEventWithTrace`：`backend/internal/runtime/observability.go` |

### 3.3 函数级锚点怎么讲

面试里尽量把“功能”和“函数”绑定着说：

| 先讲功能 | 再给函数与路径 |
|---|---|
| 把 route 变成正式执行合同 | `Manager.BuildExecutionPlan`，`backend/internal/runtime/manager.go` |
| 开启一轮 manager-owned 执行 | `Manager.BeginTurn`，`backend/internal/runtime/manager.go` |
| 进入 runtime 主入口执行 | `Executor.Execute`，`backend/internal/runtime/executor.go` |
| 在中断后恢复执行 | `Executor.Resume`，`backend/internal/runtime/executor.go` |
| 按合同准备命盘 artifact | `Executor.prefill`，`backend/internal/runtime/executor.go` |
| 把 route 和 plan 同步进运行时真相 | `Executor.syncExecutionRoute`，`backend/internal/runtime/executor.go` |
| 并发调度多个领域 runner | `Executor.runExecutionPlan`，`backend/internal/runtime/execution_dispatch.go` |
| 装载图执行上下文 | `loadOrchestrationCtx`，`backend/internal/runtime/orchestration_graph.go` |
| 做 preflight 短路判断 | `preflightNode`，`backend/internal/runtime/orchestration_graph.go` |
| 做 prefill 确定性准备 | `prefillNode`，`backend/internal/runtime/orchestration_graph.go` |
| 做真正领域 dispatch | `agentNode`，`backend/internal/runtime/orchestration_graph.go` |
| 做最终保护 | `guardNode`，`backend/internal/runtime/orchestration_graph.go` |
| 把 SSE 事件写进 trace 真相 | `emitEventWithTrace`，`backend/internal/runtime/observability.go` |
| 给最终答复做合同校验 | `guardFinalAnswerWithTrace`，`backend/internal/runtime/observability.go` |
| 把批准后的 route 标注进 trace | `annotateApprovedRouteTrace`，`backend/internal/runtime/observability.go` |

## 4. 二面最爱追的 8 个工程化问题

这一节统一用同一个答题骨架：

- `短答`：先给结论
- `展开`：讲设计原因和链路
- `风险`：讲如果不这样做会怎样
- `项目证据`：落到真实代码和真实行为

### 4.1 为什么不是单 Agent

`短答`

单 Agent 更像 prompt 组装，这个项目要解决的是 runtime 可控性，所以我把理解、合同、执行、保护拆开了。

`展开`

单 Agent 的好处是快，但它会把 route、tool、retrieval、final reply 混成一个黑盒。这个项目当前是多领域、多轮 follow-up、还要做结构化 artifact 和 SSE 过程展示，所以我选择 `thin supervisor + manager-owned runtime + bounded specialists`。这样做之后，LLM 负责“理解”和“限域执行”，程序负责“合同、状态、保护、恢复、观测”。

`风险`

如果做成单 Agent：

- 多轮 follow-up 容易漂
- 一旦输出不对，很难判断是理解错、证据错，还是工具错
- 很难做前置 artifact 校验和最终边界拦截

`项目证据`

- route 入口和执行入口是分开的：`RouteAdvisor` 在 `backend/internal/orchestrator/types.go`
- 执行合同独立存在：`ExecutionPlan` 在 `backend/internal/runtime/execution_plan.go`
- 最终答复不归 specialist：`runExecutionPlan` 后仍要 `manager.ComposeFinalReply`，代码在 `backend/internal/runtime/orchestration_graph.go`

### 4.2 为什么 route 不等于 execution contract

`短答`

因为 route 只回答“系统理解用户要干什么”，而 execution contract 回答“系统这轮最终怎么执行”。

`展开`

`ApprovedRoute` 更像审批结果，关注 primary domain、task intent、policy hints。进入运行时后，manager 还要继续把它收口成 `ExecutionPlan`，明确：

- 这轮到底跑几个 domain
- 每个 domain 之前要准备什么 artifact
- follow-up 是 direct、reuse 还是 rerun

这一步一旦显式化，后面的 preflight、prefill、dispatch、guard 才能统一消费同一份合同。

`风险`

如果把 route 直接拿去执行：

- 执行层会再次各自暗猜 domain 和 artifact
- follow-up 政策容易分散到多个组件
- trace 里会看到“理解是一套，执行是另一套”

`项目证据`

- `ExecutionPlan` 字段里直接有 `Domains`、`RequiredArtifacts`、`FollowupMode`、`Snapshot`，定义在 `backend/internal/runtime/execution_plan.go`
- `Manager.BuildExecutionPlan` 是显式入口，`backend/internal/runtime/manager.go`
- `prefillNode` 和 `agentNode` 都消费 plan，而不是自己重建路由，`backend/internal/runtime/orchestration_graph.go`

### 4.3 并发与状态隔离是怎么做的

`短答`

我的策略是“同一轮可并发执行，多轮和会话状态必须单点 owner”，也就是并发跑 runner，但状态归 manager 和 session state 统一承接。

`展开`

在 `runExecutionPlan` 里，不同 domain 会用 goroutine 并发执行，每个 domain 拿自己的 `route` 和 `DomainContext` 副本去跑，但最终结果会按 `plan.Domains` 的顺序回收和聚合。这样做的目标是两件事同时成立：

- 领域任务之间可以并发，控制总时延
- 最终对用户输出仍然由 manager 统一收口，避免“谁先跑完谁先说”

对状态隔离来说，核心原则是：

- `SessionState` 只有 manager-owned 主链承接
- runner 拿到的是本轮请求上下文，不拥有最终会话控制权
- trace、snapshot、guidance 由统一运行时写入，不让多个 specialist 随意改对话真相

`风险`

并发如果只做“快”，不做“收口”：

- 会出现结果覆盖、顺序错乱
- 多领域结果可能先后不一致
- 同 session 下更容易串状态

`项目证据`

- `runExecutionPlan` 用 `sync.WaitGroup` 和 goroutine 并发调度，代码在 `backend/internal/runtime/execution_dispatch.go`
- 结果数组 `results` 按原域顺序聚合，而不是按返回时间聚合，还是这个文件
- `Request` 里把 `ManagerContext`、`DomainContext`、`Session` 明确拆开，定义在 `backend/internal/specialists/runner.go`
- 项目里补过 `MemoryStore` 并发隔离测试，`PROGRESS.md` 已记录该事实

### 4.4 失败恢复怎么做

`短答`

这个项目不是“失败了就报错”，而是先分类，再决定短路、恢复、降级还是拦截。

`展开`

当前主链里，失败恢复主要有 4 种：

- preflight 短路：资料不全、需要澄清时不进完整执行链
- artifact 缺失前移：dispatch 前用 `validatePlanArtifacts` 阻断
- interrupt resume：中断后可用 `Executor.Resume` 继续
- final guard 拦截：如果主域 artifact 没拿到，或者最终回答泄漏内部细节，就阻断输出

也就是说，恢复不是只有一种路径，而是和失败位置绑定。

`风险`

如果所有失败都统一成 `agent_error`：

- 用户看不到清晰的下一步
- 工程上很难判断到底该修 route、tool、guard 还是 compose
- 很难做针对性的回归样本

`项目证据`

- `preflightNode` 会根据 `ShortCircuit` 直接走短路分支，`backend/internal/runtime/orchestration_graph.go`
- `validatePlanArtifacts` 在 dispatch 前校验 artifact，`backend/internal/runtime/specialist_runner.go`
- `Executor.Resume` 支持 checkpoint 恢复，`backend/internal/runtime/executor.go`
- `guardFinalAnswerWithTrace` 会对 artifact 缺失和输出边界分别阻断，`backend/internal/runtime/observability.go`

### 4.5 trace truth layer 是什么，为什么重要

`短答`

trace truth layer 的意思是，不把 UI 观感当真相，而是把“系统这轮实际怎么执行过”沉淀成可验证事实。

`展开`

这个项目的一个关键设计是把 trace 当运行真相层，而不是日志装饰层。也就是说，trace 不只记耗时，还要记决策来源、follow-up mode、artifact 是否存在、SSE 实际发了什么。这样在看一次错误回答时，能回答这些问题：

- 这轮 route 是 supervisor 给的还是 cheap gate 复用的
- plan 里到底准备了哪些 artifact
- preflight 是否发生了强制改路由
- agent 节点到底派发了哪些 domain
- 最终发给前端的 text/component 是什么

`风险`

如果没有 truth layer：

- 很容易把“模型表面上说得顺”当作系统稳定
- 前端只看到最终文本，无法追到执行事实
- 评测和回归样本无法对齐真实运行时

`项目证据`

- `annotateApprovedRouteTrace` 会把 `decision_source`、`gate.reason`、`reuse_cached_result` 等写进 trace，`backend/internal/runtime/observability.go`
- `emitEventWithTrace` 会把 SSE 发出的 event type 和附加属性写进 trace，`backend/internal/runtime/observability.go`
- `guardNode`、`preflightNode`、`agentNode` 都会打 span 和 attribute，`backend/internal/runtime/orchestration_graph.go`

### 4.6 成本和时延怎么控

`短答`

我没有只追求“能答”，而是通过 cheap gate、资产复用、并发 dispatch、受控检索来减少不必要的 LLM 路径。

`展开`

成本和时延控制主要来自 5 个点：

- 普通 follow-up 命中 cheap gate 时不再完整重跑 route
- `ExecutionPlan` 先决定 direct、reuse、rerun，避免每轮都走最重路径
- 多 domain 可以并发 dispatch，减少尾延迟
- `knowledge_search` 是薄检索工具，不让知识库直接走大答案生成
- artifact-driven prefill 把确定性工作前移，减少 specialist 临场补救成本

`风险`

如果没有这些控制：

- 每轮都走完整 supervisor + specialist，成本会快速失控
- follow-up 时延会比用户预期高很多
- retrieval 越自由，token 和失败面都会变大

`项目证据`

- `decisionSourceForRoute` 会把 cheap follow-up reuse 标成 `cheap_followup_reuse`，`backend/internal/runtime/observability.go`
- `ExecutionPlan.FollowupMode` 是显式字段，`backend/internal/runtime/execution_plan.go`
- `runExecutionPlan` 做并发调度，`backend/internal/runtime/execution_dispatch.go`
- `PROGRESS.md` 里已经记录了 cheap gate 聚合报告、follow-up reuse、最小正式数据集扩面这些工程事实

### 4.7 工具边界怎么设，为什么知识库不能直接回答

`短答`

工具只负责取事实，不负责拥有最终立场；知识库在这个项目里是 retrieval 边界，不是 answer owner。

`展开`

命理 runtime 的核心不是“搜到什么就直接念什么”，而是：

- 工具负责给证据
- specialist 负责领域分析
- manager 负责把领域结果装配成最终答复

如果让知识库工具直接输出最终答案，等于把主链又拆出一个隐式 owner。那样会让 retrieval 和 answer contract 混在一起。

`风险`

工具边界一旦变糊：

- 很难判断 hallucination 来自模型还是来自工具输出
- 很难统一 final guard
- 很难在 follow-up 时复用结构化资产

`项目证据`

- `knowledge_search` 在项目设计里是薄工具边界，真实入口在 `backend/internal/tools/knowledge_search.go`
- `ADKSpecialistRunner.Run` 只把工具结果保存进 session，不直接给最终用户答复，`backend/internal/runtime/specialist_runner.go`
- 最终答复仍要经过 `manager.ComposeFinalReply` 和 `guardFinalAnswerWithTrace`

### 4.8 为什么说 manager compose 和 final guard 不能互相替代

`短答`

compose 负责“把能说的组织好”，guard 负责“把不该过的拦住”，它们不是一回事。

`展开`

manager compose 解决的是内容组织问题，比如单域、多域、follow-up 的结果怎样表达更稳定。final guard 解决的是合同和边界问题，比如：

- 主域 artifact 是否真实存在
- 最终文本有没有泄漏 system prompt、trace_id、tool_call 这类内部细节

一个是生成后的组织层，一个是验收前的保护层。

`风险`

如果没有 guard，只靠 compose：

- compose 再小心，也可能吃到错误输入
- 一旦上游漏掉 artifact，最终还是会生成伪结论

如果没有 compose，只靠 guard：

- guard 只能拦错，不能把多领域结果组织得更好

`项目证据`

- `agentNode` 里先 `runExecutionPlan`，再 `manager.ComposeFinalReply`，`backend/internal/runtime/orchestration_graph.go`
- `guardNode` 最后调用 `guardFinalAnswerWithTrace`，还是这个文件
- 输出边界拦截逻辑在 `outputBoundaryGuard`，`backend/internal/runtime/observability.go`

## 5. 你要能完整讲出的工程化专题

### 5.1 并发与状态隔离

一句话：

> 同一轮允许并发跑 worker，但会话真相只能由 manager-owned runtime 统一承接。

展开时讲 4 点：

- 并发目标是降低多领域尾延迟，不是让多个 specialist 抢答
- `runExecutionPlan` 是并发 dispatch，但结果按 plan 顺序收口
- `SessionState`、`ExecutionSnapshot`、guidance 这类真相态不交给 specialist 持有
- 同 session 并发安全要靠状态存储隔离测试证明，不靠主观相信

### 5.2 失败恢复

一句话：

> 恢复不是一个补丁，而是按失败位置分层设计：preflight 短路、artifact 前移校验、checkpoint resume、guard 阻断。

展开时讲 4 点：

- 资料不够时短路给用户明确下一步
- artifact 缺失前移，不把脏输入交给 specialist
- 交互型中断用 checkpoint 恢复
- guard 阶段专门处理合同违规和内部细节泄漏

### 5.3 trace truth layer

一句话：

> trace 在这里不是日志附属品，而是运行时真相源。

展开时讲 4 点：

- trace 记录 route 来源，而不只记录耗时
- trace 记录 plan 和 turn type，而不只记录“调用过哪个模型”
- trace 记录 SSE 实际发了什么，能对齐前端感知
- trace 和 eval、测试一起组成 truth layer，而不是相互替代

### 5.4 成本与时延

一句话：

> 成本优化不是只减 token，而是减少不必要的重路由、重执行、重检索。

展开时讲 4 点：

- 普通 follow-up 优先走 cheap gate 或 reuse
- direct / reuse / rerun 先在 `ExecutionPlan` 层分流
- 多域 dispatch 并发化，避免串行拖长尾
- 检索只拿证据，不把知识库变成第二个答题系统

### 5.5 工具边界

一句话：

> 工具是 bounded capability，不是独立 owner。

展开时讲 4 点：

- 工具返回事实和结构化结果
- specialist 在本领域内消费工具
- manager 负责收口
- final guard 负责验收

### 5.6 为什么 route 不等于 execution contract

一句话：

> route 是理解，plan 是执行；理解错要修路由，执行错要修合同和运行时，两个问题不能混着看。

展开时讲 4 点：

- route 解决 admission
- plan 解决 domains、artifacts、follow-up policy
- prefill 和 dispatch 只消费 plan
- trace 上要同时能看见 route 和 execution snapshot

## 6. 高频追问的结构化回答模板

这一节是面试时可以直接套用的答案骨架。

### 6.1 追问：你在这个项目里最有技术含量的设计是什么

`短答`

我认为最有技术含量的不是某个 prompt，而是把 route、execution contract、artifact、bounded specialist、final guard、trace truth layer 串成了一条 manager-owned 主链。

`展开`

它的难点不在单点能力，而在边界收口。尤其是 route 不等于执行、follow-up 不等于全重跑、检索不等于最终答案、trace 不等于普通日志，这几层如果不拆开，系统很快就会变黑盒。

`风险`

如果只看表面回答质量，很容易把一个偶尔答得不错的 demo 当作稳定系统。

`项目证据`

- 主链落在 `backend/internal/runtime/orchestration_graph.go`
- 合同落在 `backend/internal/runtime/execution_plan.go`
- 保护落在 `backend/internal/runtime/observability.go`

### 6.2 追问：你最满意的工程化取舍是什么

`短答`

我最满意的是把 follow-up 分流前移到了 `ExecutionPlan`，而不是让 renderer、specialist、guard 各猜各的。

`展开`

多轮系统最容易失控的地方不是首轮，而是 follow-up。当前这个项目把 direct、reuse、rerun 先变成 manager 的合同决策，后面所有节点只消费结果，这样系统解释性会高很多。

`风险`

如果 follow-up 判断散落在多个层，会出现同一句追问在不同路径里给出不同处理策略。

`项目证据`

- `ExecutionPlan.FollowupMode` 明确定义在 `backend/internal/runtime/execution_plan.go`
- `prefillNode`、`agentNode` 统一消费 plan，`backend/internal/runtime/orchestration_graph.go`

### 6.3 追问：这个项目里你觉得最像生产问题的点是什么

`短答`

并发状态隔离、失败恢复和 trace truth layer，这三件事最接近真实生产问题。

`展开`

因为它们不是“回答对不对”的问题，而是“系统在复杂情况下还能不能稳”的问题。真实线上往往不是模型单次推理失败，而是同 session 多轮交互、数据不完整、工具部分失败、检索 0 命中、前端要恢复上一轮展示态，这些组合起来才是生产难点。

`风险`

如果只做 happy path，很难发现系统在复杂状态转换下的脆弱点。

`项目证据`

- Resume 路径在 `Executor.Resume`
- artifact 前移校验在 `validatePlanArtifacts`
- trace 标注在 `annotateApprovedRouteTrace` 和 `emitEventWithTrace`

## 7. 场景追问与答案

这一节专门补二面喜欢出的“你会怎么处理”类问题。

### 7.1 场景：多领域同时执行，一个 runner 很慢，另一个很快，你怎么保证最终回答稳定

`短答`

并发执行可以无序返回，但最终聚合必须按 plan 顺序收口，不能按“谁先完成谁先说”。

`展开`

我会让领域任务 goroutine 并发执行，但结果落到固定下标的结果数组里，最后按 `plan.Domains` 聚合。这样做有两个好处：

- 利用并发缩短总体时延
- 保证最终结果顺序稳定，便于测试和回归

`风险`

如果按返回时间拼接结果：

- 最终文本顺序会不稳定
- trace 和测试都不好对齐

`项目证据`

`runExecutionPlan` 用 `results[idx] = result` 固定收口顺序，`backend/internal/runtime/execution_dispatch.go`

### 7.2 场景：用户连续追问，但这轮其实不需要重跑完整八字链路，你怎么降成本

`短答`

我会先让 manager 判断这轮是 direct、reuse 还是 rerun，而不是默认重跑。

`展开`

follow-up 里很多问题只是解释术语、追问上一轮结论或者复用已有命盘资产。这类问题如果还走完整 supervisor + specialist，会徒增成本和时延。所以我会在 `ExecutionPlan` 先做分流，普通追问优先 direct 或 reuse，只有确实依赖新证据或新推理时才 rerun。

`风险`

如果一律重跑：

- 成本高
- 用户感知更慢
- 上下轮口径更容易漂

`项目证据`

- `ExecutionPlan` 含 `FollowupMode`
- `decisionSourceForRoute` 能区分 cheap follow-up reuse

### 7.3 场景：RAG 0 命中时你怎么处理，才能既不乱编也不直接崩

`短答`

0 命中要进 trace，但不等于整轮失败；如果主链还能靠静态结构化产物保守成文，就继续走降级路径。

`展开`

RAG 是证据增强，不是所有结论的唯一来源。遇到 0 命中时，我会要求系统显式记录“没有拿到外部证据”，但如果已有命盘 artifact 和保守规则足以支撑基础回答，就继续产出一个有边界感的答复，而不是假装查到了资料，也不是直接把整轮打成 `agent_error`。

`风险`

两种极端都不好：

- 假装命中：会制造伪证据
- 直接报错：会让系统过度脆弱

`项目证据`

`PROGRESS.md` 已记录 authority-first 在 `hits=0` 场景下做了保守降级，不再直接终止

### 7.4 场景：面试官质疑“你这个 trace 有什么用，不就是日志吗”

`短答`

不是。日志通常记录现象，trace truth layer 记录的是可复盘的执行事实。

`展开`

我会强调这里的 trace 至少解决 3 件事：

- 看到决策来源，比如 supervisor 还是 cheap gate
- 看到执行合同，比如 follow-up mode、主域、artifact 状态
- 看到实际发给前端的 SSE 事件

所以 trace 在这里能和用户看到的结果一一对应。

`风险`

如果只是普通日志，很难把一轮错误回答和具体执行节点绑定起来。

`项目证据`

- `annotateApprovedRouteTrace`
- `emitEventWithTrace`
- `guardNode` / `agentNode` span

### 7.5 场景：如果 specialist 直接输出最终用户答复，会有什么问题

`短答`

会破坏 manager-owned runtime，让多轮对话 owner 和最终验收边界都变模糊。

`展开`

specialist 适合做限域分析，不适合拥有最终答复权。因为最终答复需要综合：

- 当前用户问题
- 是否多域
- 是否 follow-up
- 是否要复用已有资产
- 是否要经过 final guard

这些都是 manager 层视角，不是单领域 specialist 能独立决定的。

`风险`

如果 specialist 直接答：

- 多域收口会乱
- follow-up 口径容易不连续
- 最终边界校验很难统一

`项目证据`

`agentNode` 里在 `runExecutionPlan` 之后仍会 `manager.ComposeFinalReply`

### 7.6 场景：如果 route 判成了八字，但 prefill 没拿到八字命盘，你会怎么办

`短答`

不会继续生成最终结论，而是把它当成合同违规，在前面或最后一道 guard 拦掉。

`展开`

这个场景说明“理解”和“执行事实”已经脱钩了。系统应该优先暴露合同失配，而不是继续让模型猜答案。理想路径是 dispatch 前用 `validatePlanArtifacts` 拦截；如果前面漏掉了，`final guard` 还要再次校验主域 artifact。

`风险`

如果放过去：

- 用户会看到没有真实命盘支撑的伪结论
- 后续 follow-up 还会继续建立在错误状态上

`项目证据`

- `validatePlanArtifacts` 在 `backend/internal/runtime/specialist_runner.go`
- `primaryArtifactGuard` 在 `backend/internal/runtime/observability.go`

### 7.7 场景：如果并发执行里一个 domain 失败、另一个成功，你会怎么设计返回策略

`短答`

默认不要悄悄吞失败拼半成品，要先看这轮是否允许降级，再决定阻断还是保守回答。

`展开`

这个项目当前更偏保守，因为命理输出对主域 artifact 和一致性要求高。我的思路是：

- 主域失败时，优先阻断或转成明确降级提示
- 非主域失败时，只有在合同允许的情况下才输出部分结果
- trace 必须能区分“完整成功”和“部分降级成功”

`风险`

如果悄悄吃掉失败：

- 最终回答可能看似完整，实际缺了关键依据
- 评测和用户反馈都很难定位问题

`项目证据`

当前 `runExecutionPlan` 会记录 `firstErr` 并整体返回错误，`backend/internal/runtime/execution_dispatch.go`

### 7.8 场景：如果让你把这个项目再往生产推进一步，你会先做什么

`短答`

我会先补强 truth layer 和失败闭环，而不是先加更多花哨能力。

`展开`

下一步优先级我会放在：

- mixed-domain 和复杂 follow-up 的更系统回归
- 更完整的在线指标和告警
- 工具风险分级和权限审计
- 更细的 degrade 分类和数据集

这些能力比继续加更多“会答什么”更接近真实生产价值。

`风险`

如果先盲目扩能力，不先补闭环，系统复杂度会先涨，可信度反而下降。

`项目证据`

`PROGRESS.md` 已明确当前后续优先项是删 legacy、扩 follow-up 回归、扩 cheap gate 可观测、继续强化 truth layer

## 8. 面试时可以主动讲出来的工程化亮点

如果面试官没有追到这么细，你可以主动补这 8 点：

1. route approval 和 execution contract 是分离的
2. manager 是唯一 conversation owner
3. prefill 是 artifact-driven，不是按主域猜
4. specialist 是 bounded worker，不直接拥有最终答复权
5. multi-domain dispatch 可以并发，但最终收口顺序稳定
6. final guard 不只查 artifact，还拦内部执行细节泄漏
7. trace 会记录 route 来源、SSE 事件、guard 结果，是真相层
8. follow-up 有 direct / reuse / rerun 分流，不是每轮重跑

## 9. 公开资料怎么映射到这个项目

| 公开资料 | 关键观点 | 在本项目里的对应点 |
|---|---|---|
| OpenAI Agents Guide | 先用简单、可控、可观测链路收口，再扩能力 | 主链先收口，再补 cheap gate、trace、retrieval、follow-up reuse |
| Anthropic Effective Agents | routing、orchestrator-worker、bounded workflow 是高频模式 | 当前主链本质上是 routing + manager-worker + controlled workflow |
| Langfuse Observability | trace、dataset run、score 要形成真相闭环 | 本项目把 trace、回归样本、Langfuse 观测一起当 truth layer |

## 10. 最后一句收尾怎么讲

> 这个项目的价值不在“做了一个算命聊天页面”，而在于我把一个 Agent 系统里最容易失控的部分，包括路由、执行合同、状态隔离、失败恢复、受控检索、最终保护和 trace 真相层，做成了一条能解释、能排障、能继续演进的工程化主链。
