# 项目进度

> 这是项目的上下文恢复文件。
> 目标是让新对话中的 AI 编码助手快速接手当前工作，因此这里只保留仍然有效的事实、当前优先级和关键入口。更细的历史过程请看 `git log` 和 `eval/reports/`。

---

## 当前阶段

- **最后更新：** 2026-07-23
- **当前阶段：** v1.5 收口，准备合并到 `master`
- **当前状态：**
  - 默认真实流量已经稳定在 manager-owned runtime 主链
  - 八字单域主链已经切到 authority-first inner graph
  - 前端、知识库、SSE、trace、最小 smoke 回归、Docker 本地部署入口都已打通

## 当前最重要的事实

1. **执行主链已经定型**
   - 当前有效主链：`RouteAdvisor -> Policy Gate -> Manager -> ExecutionPlan -> Prefill -> ToolRunner -> specialist runner(s) -> manager compose -> final guard -> SSE`
   - `Manager` 是 runtime 内唯一 conversation owner。
   - `ExecutionPlan` 是 route approval 进入执行层后的正式合同，显式承载 route、domains 和 required artifacts。
2. **领域执行已经收口**
   - 八字、奇门、紫微三领域都接入统一 runtime。
   - `specialist runner(s)` 是 bounded workers，不直接拥有最终用户答复权。
   - mixed-domain 与 follow-up 路径也走 manager-owned dispatch 和 compose。
3. **八字主链已经从自由生成收口为 authority-first**
   - 当前稳定链路是“分析模式判定 -> 证据规划 -> 受控检索 -> 静态/动态综合 -> 程序 renderer 成文”。
   - 最终成文优先依赖结构化结果 + Go renderer，而不是自由 writer。
4. **评测真相源已经收口**
   - 官方回归入口：`make regression`
   - 当前正式 truth layer：`Go 合同测试 + eval/datasets/*.json + eval/reports/*.json + backend/.env + Langfuse trace/session/dataset run/score`
   - 不要把某个 Langfuse UI 页面是否显示正文，当作唯一验收信号。
5. **本地部署入口已经固定**
   - 默认 Docker 入口是 `deploy/app/`
   - 该入口启动 `app`（Go 后端 + 内嵌前端构建产物）和 `knowledge`
   - 知识库目录采用仓库直挂载：`knowledge/wiki/`、`knowledge/raw/` 直接映射到容器内
6. **本地开发启动入口已经收敛**
   - 完整开发入口：`make dev`，启动 Docker Langfuse + WSL 本地知识库、后端、前端。
   - 核心开发入口：`make dev-core`，只启动知识库、后端、前端。
   - `start.sh` 只保留为 `make dev` 的兼容代理，不再维护第二套启动逻辑。

## 当前有效架构

`RouteAdvisor -> Policy Gate -> Manager -> ExecutionPlan -> Prefill -> ToolRunner -> specialist runner(s) -> manager compose -> final guard -> SSE`

- `RouteAdvisor` 负责 admission、routing、fallback。
- `Policy Gate` 负责策略修正和硬边界。
- `Manager` 负责上下文承接、执行规划和最终回复装配。
- `Prefill` 按 `RequiredArtifacts` 确定性准备命盘结果。
- `ToolRunner` 负责 runtime-owned 工具调用的合同校验、参数阻断、超时、重试、错误分类和 trace 元数据。
- `specialist runner(s)` 只做领域执行。
- `final guard` 是最后一道保护层，不是主 artifact 缺失检测器。

## 稳定能力

- 八字、奇门、紫微三领域主链已接入统一 runtime。
- prefill 已改为 artifact-driven，不再只按 `primary_domain` 猜测。
- 八字单域请求会进入 authority-first inner graph，结果由程序 renderer 固定成文。
- 独立知识库运行在 `:3100`，后端通过 MCP/RAG 检索古籍资料。
- 知识库 MCP 已新增 `retrieve_passages`：复用 `query-search` 的 BM25 + 向量 + RRF 混合召回，只返回 passages，不做 `query_wiki` 的答案综合生成；运行时 `knowledge_search` 已切到同一检索能力的 `/api/wiki/retrieve` 薄入口。
- SSE 事件流、前端分层渲染、知识依据卡片、处理过程卡和本地 trace 已接通。
- `deploy/app` 已可在 WSL2 Ubuntu 中通过 `docker compose up -d` 跑通本地部署。
- 已引入 Phase A 合同层：`LastInputState`、`GateContract`、`ExecutionSnapshot`，用于区分 UI 恢复态、策略门控结果和真实执行快照。
- 已完成 Phase B 第一段消费链：新增 `GET /api/session/:sessionID` 会话快照出口，前端会持久化 `session_id` 并恢复历史消息；`ExecutionSnapshot` 已进入 process/debug/execution-tree 投影，用于 trace/debug 展示。
- 已完成 Phase B 第二段第一版：在 `supervisor.Approve` 前新增 cheap follow-up gate，命中“已有命盘的普通追问”时直接复用上一轮执行合同，跳过一次完整 supervisor LLM 路由；补资料、显式换术数、时机问题、多域诉求仍回退完整路由链。
- 已完成 Phase B 第二段第二版：cheap gate 的 `decision_source / gate_reason / reuse_cached_result / reuse_session_profile` 已进入 trace runtime meta，前端 process/debug 面板可直接区分“正常路由”与“cheap gate 复用命中”。
- 已完成 Phase B 第二段第三版：新增本地 cheap gate 样本沉淀，命中时会写入 `logs/reports/cheap-gate/hits.jsonl`，用于后续统计命中率、回退原因和典型复用样本。
- 已完成 Phase B 第二段第四版：新增 `cheap gate` 聚合报告入口，`make cheap-gate-report` 会把 `logs/reports/cheap-gate/hits.jsonl` 汇总成 `eval/reports/cheap-gate-summary.json`，用于判断命中分布与后续是否安全扩面。
- 已增强会话恢复载荷：`GET /api/session/:sessionID` 除文本历史外，还会尽量回放最近一轮 assistant 的 `thinking / component / text / error` 结构化片段，前端刷新后可恢复更接近真实的上一轮展示态。
- 已修正前端会话切换交互：浏览器本地 `session_id` 只代表“当前活跃会话”，前端新增显式“新对话”入口，点击后会生成新的 `session_id` 并清空当前消息，不再把整个浏览器永久绑定到同一个 session。
- 已修复前端 session 初始化兼容性：`useSSE` 不再硬依赖 `crypto.randomUUID()`；缺失时会自动降级到 `getRandomValues` / 本地随机串，避免旧浏览器或异常运行环境直接白屏。
- 已对 cheap gate 做一版保守扩面：允许“上一轮为单域 `interpret_chart` 且已有缓存结果，本轮只是同域普通展开追问”直接复用执行合同；多域、补资料、时机类、显式换术数仍回退完整 supervisor。
- 已修复八字 authority-first 在“检索 0 命中”场景下的阻塞性 `agent_error`：`validateEvidenceBundlePreconditions` 不再把空 evidence bundle 当成致命错误，`2010年1月1日1点 男 北京` 已能在 `hits=0` 时继续走静态/动态综合并正常产出最终文本；同时 trace 会额外记录 `bazi.graph.error_*`、`bazi.inner_agent.*`、`bazi.final_writer.*` 便于后续排障。
- 已修复八字 authority-first 在“动态总述与当前大运首条口径打架”场景下的阻塞性 `agent_error`：当 `current_trend` 与 `dayun_path[0]` 一正一反时，运行时不再直接报 `dynamic_consistency` 终止，而会自动收束为“吉中有阻/承托与限制并存”的保守动态口径；真实回归 `2011年11月10月11点20点 男 北京` 已从 `agent_error` 恢复为 `turn_type=agent_reading`、`status=ok`。
- 已把八字 topic 追问的高层判题前移到结构化合同：`analysis_plan` 新增 `topic_mode`，`static_synthesis` 新增 `topic_direct_answer / topic_focus_answer`，`bazi_final_renderer` 不再按用户问句字面硬编码 case，而是只消费上游字段做成文。
- 已把 follow-up 处理模式正式收回到 `ExecutionPlan`：现在由 `Manager.BuildExecutionPlan` 先决定 `direct / rerun_specialist`，`preflight` 与 `orchestration graph` 只消费该合同，不再各自暗判“要不要重跑八字 graph”。
- 当前 manager-owned direct 只落了一条保守路径：`fortune_followup + primary=bazi + 已有命盘` 且命中“术语/常识解释型追问”时，直接返回通用八字释义；只有“我这盘为什么算 X”这类命盘依赖追问才继续进入八字执行链。
- 已新增 follow-up 解读资产复用第一版：单域正常解读完成后，会把最终解读摘要写入对应 `DomainContext.RuntimeValues`；后续同域 `fortune_followup` 会优先走 `reuse_artifact`，由 manager 续答，不再默认重跑八字 / 紫微 / 奇门执行链。
- 已把 manager-owned `runExecutionPlan` 热路径进一步从 legacy 兼容注入中剥离：bounded runner 现在只共享 `EventSink`，不再依赖 `legacyRunnerDeps` 才能运行；旧的 `LegacySpecialistRunner` 及其测试已删除。
- 已继续清理运行时遗留噪音：删除 `SessionState.NeedsKnowledge` 孤儿兼容字段，并把 `Executor`、`container`、`TurnLoopSessionManager`、`bridge` 等核心注释收口到当前 manager-owned 主链口径。
- 已补上 orchestrationGraph 行为级回归：除拓扑编译外，现已覆盖 manager-owned bounded runner 主链和“prefill 后 artifact 仍缺失 -> agent_error”失败路径。
- 已把最小正式数据集从“只测首轮 happy path”扩大到“首轮 + 同 session follow-up + 第二条 retrieval quote case”，避免 `eval/` 继续只证明 `/api/chat` 能跑完。
- 已补上生产闭环学习材料：`docs/security-boundary.md`、`docs/production-closure.md`、`docs/interview-agent-architecture.md`，明确当前安全边界、并发/失败闭环、面试讲法和不做的大型生产化能力。
- 已增强 final guard 的输出边界：除主域 artifact 合同外，现在会拦截明确泄漏 `system prompt`、`trace_id`、`tool_call` 等内部执行细节的最终回答，并用代码断言覆盖。
- 已新增 `MemoryStore` 并发隔离测试，覆盖同 session 并发创建只返回同一状态对象、不同 session 并发保存不串 `ExecutionSnapshot / RecentTurns / BaziResult`。
- 已新增生产平台版工具治理层第一版：`ToolContract` 记录工具版本、风险、副作用、参数、超时、重试、审批和幂等入口；`Registry` 同时保存工具与合同；`ToolRunner` 统一执行 runtime-owned 确定性工具调用，并写入工具 trace 元数据。当前覆盖 `Prefill` 里的 Go runtime 主动工具调用，specialist ADK 内部工具迁移后续单独推进。

## 当前操作约定

- **后端配置真相源：** `backend/.env`
  - 当前唯一有效的 Langfuse / OTEL 配置文件，不再依赖仓库根目录 `.env`
- **官方回归入口：** `make regression`
  - 负责自启动 / 自清理本地 `:8080` 后端，并执行最小 smoke
- **本地 Docker 入口：** `deploy/app/docker-compose.yml`
  - 优先通过 WSL 路径运行 Docker 命令
- **本地开发入口：** `make dev`
  - 启动 Langfuse `:3001`、知识库 `:3100`、后端 `:8080`、前端 `:5173`
  - 不需要观测时用 `make dev-core`，只启动知识库、后端和前端
  - 常用检查：`make status`；常用重启：`make restart` / `make restart-core`
- **本地 Langfuse 状态：**
  - 当前验证打通的是 `Traces`、`Sessions`、`Datasets`、`Dataset Runs`、`Scores`
  - 本地使用 self-hosted `v3`，`Experiments / Evals` 不是正式主工作流
- **模型价格同步：**
  - 价格文件：`eval/model-pricing/langfuse-model-pricing.json`
  - 同步脚本：`eval/runner/sync-langfuse-model-pricing.py`

## 合并后优先事项

- 继续删除 legacy execution-supervisor 兼容代码，减少双路径心智负担。
- 继续扩大 mixed-domain 与复杂 follow-up 回归覆盖，但不要重新引入整批过期数据集；当前 `eval/` 已补到首轮 / follow-up / retrieval 三类最小样本。
- 继续扩大八字结构化评测样本，降低对单盘 prompt 微调的依赖。
- 推进跨会话上下文工程，但保持 manager-owned contract 不被稀释。
- 继续把 `GateContract / LastInputState / ExecutionSnapshot` 扩展到更细的 cheap gate / route reuse 优化；当前已完成前端恢复、trace 展示、普通追问 cheap follow-up gate、cheap gate 可观测信号接线，以及本地样本沉淀。
- 在继续前移 cheap gate 前，先观察 `eval/reports/cheap-gate-summary.json` 的本地样本分布；当前只做了单域 `interpret_chart -> followup` 的保守扩面，不应把它演化成第二套路由器。

## 关键入口文件

### 运行时主链

- `backend/internal/supervisor/approved_route.go`
- `backend/internal/supervisor/cheap_gate.go`
- `backend/internal/supervisor/adk_engine.go`
- `backend/internal/runtime/orchestration_graph.go`
- `backend/internal/runtime/preflight.go`
- `backend/internal/runtime/manager.go`
- `backend/internal/runtime/final_guard.go`
- `backend/internal/runtime/observability.go`

### 八字 authority-first

- `backend/internal/runtime/bazi_charter_graph.go`
- `backend/internal/runtime/bazi_final_renderer.go`
- `backend/internal/runtime/bazi_evidence_bundle.go`
- `backend/internal/runtime/testdata/bazi_eval_cases/`

### 检索、状态、观测

- `backend/internal/tools/knowledge_search.go`
- `backend/internal/tools/contract.go`
- `backend/internal/tools/runner.go`
- `backend/internal/state/session.go`
- `backend/internal/handler/session.go`
- `backend/internal/tracing/`
- `backend/internal/observability/cheap_gate_reporter.go`
- `web/src/composables/useSSE.ts`
- `web/src/components/ChatPanel.vue`
- `eval/datasets/runtime-smoke-v1.json`
- `Makefile`

### 部署与文档

- `deploy/app/docker-compose.yml`
- `deploy/app/README.md`
- `docs/architecture.md`
- `docs/tool-governance.md`
- `docs/architecture.md`
- `docs/acceptance-criteria.md`

## 验证命令

```bash
go test ./backend/... -v
cd web && npx vue-tsc --noEmit
cd web && npm run build
```

## 关键决策

- 统一架构保持为 `thin supervisor + manager-owned runtime + bounded specialists`。
- `ApprovedRoute` 是 runtime 唯一主控输入，route approval 与执行层继续分离。
- manager 是 runtime 内唯一 conversation owner；specialist 不再拥有最终用户答复权。
- prefill 必须按 `RequiredArtifacts` 做确定性准备，不再按单一主域猜测。
- 纯八字主链继续坚持 authority-first graph，不回退到自由成文。
- 最终成文优先由结构化结果 + 程序 renderer 保证稳定性。
- 本地评测体系继续以 `Go 合同测试 + eval + Langfuse 观测` 为正式 truth layer。
- 本地 Docker 路径继续以 `deploy/app` 为默认入口，Langfuse 不强绑进默认部署。
- 以 `WeKnora` 为主参考引入外围成熟能力，但不回退 `RouteAdvisor -> Policy Gate -> Manager -> ExecutionPlan` 主链；Phase A 先落 `LastInputState / GateContract / ExecutionSnapshot`。
- Phase B 先做“真实消费点”而不是继续加合同定义：`LastInputState` 先服务前端会话恢复提示，`ExecutionSnapshot` 先服务 process/debug/execution-tree 投影。
- cheap gate 只做窄场景复用，不演化成第二套路由器；当前仅放行“已有可复用结果的普通追问”，显式换术数、补资料、时机类、多域诉求一律回退完整 supervisor。
- cheap gate 一旦存在，就必须自带可观测性：至少能在 trace / process/debug 面板里看见 `decision_source`、`gate_reason` 和复用信号，避免把“省了一次路由”变成黑盒行为。
- 在扩 cheap gate 命中面之前，先做本地样本沉淀；当前先落 `logs/reports/cheap-gate/hits.jsonl`，后续再决定是否做聚合统计脚本或 eval 报表接线。
- 会话恢复优先恢复“最近一轮 assistant 展示态”，而不是把 debug 日志做成完整事件存档系统；当前只回放最近一轮、且保持前端消费链简单不变。
- 在没有用户体系和会话列表前，浏览器本地 `session_id` 只能服务“当前会话恢复”；切换对话必须显式新建 session，不能依赖刷新页面隐式切换。
- 前端 session 生成必须做环境兼容降级，不能假定运行环境一定支持 `crypto.randomUUID()`。
- `Makefile` 已按当前 Windows + WSL 现实环境收口：`make dev` 启动带 Langfuse 的完整开发栈，`make dev-core` 启动核心三服务；长运行服务由 tmux 托管，`make status` 统一验活四个端口。
- authority-first 检索链允许“无古籍命中但继续保守成文”的降级路径；空 retrieval 结果应体现在 `thinking / trace` 中，而不是直接把整轮咨询打成 `agent_error`。
- authority-first 动态链也允许“当前趋势总结与大运首条方向冲突时继续保守成文”的降级路径；这类错误应被收束成“吉中有阻/机会与限制并存”的动态口径，而不是直接把整轮咨询打成 `agent_error`。
- final renderer 只保留模板骨架、展示映射与少量越界保护；topic 追问的“解释型/保守原因/岁运原因/普通分析”判题不再放在 renderer 内按问句字面猜测，而应由上游结构化节点显式给出。
- follow-up 的“是否直接回答 / 是否继续进 specialist 执行链”必须先落在 `ExecutionPlan`，由 manager 统一决策；`preflight`、`renderer`、领域 graph 不再各自偷偷再做一套路由判断。
- follow-up 分流不再把“含八字术语”直接等同于“必须重跑八字 specialist”；通用术语/常识解释优先由 manager-owned direct 处理，是否依赖当前命盘结构再决定要不要进入八字 graph。
- follow-up 默认应优先复用“上轮已完成的解读资产”，而不是只复用命盘 artifact；当前第一版先支持单域 `reuse_artifact`，跨域复用和“是否足以回答”的细判后续再细化。
- 知识库工具边界保持“检索归知识库、最终解释归 runtime”：`retrieve_passages`/`knowledge_search` 只返回证据片段，`query_wiki` 保留给知识库问答场景，不进入命理 runtime 的最终答案链。
- `docs/architecture.md` 已收口为单一架构真相源；分拆说明和历史归档已移除，不再作为 onboarding 主路径。
- 学习项目的生产闭环优先用 Go 合同测试和小规模 smoke/retrieval 验证来证明状态、权限、artifact、guard 与 trace 边界；暂不投入大规模命理质量 eval、LLM Judge 校准或重型 durable workflow 基础设施。
- 工具治理层先做“生产平台骨架”，不把它误写成全量 Tool Platform 已完成：当前先覆盖 runtime-owned 确定性工具；有副作用工具必须走审批、幂等键和服务端结果查询；specialist ADK tool adapter 迁移是下一阶段。
