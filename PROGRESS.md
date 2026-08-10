# 项目状态

> 当前事实快照，不记录实施流水。历史过程查 Git、专项设计文档和 `eval/reports/`。

## 当前阶段

- **最后更新：** 2026-08-11
- **阶段：** v1.5 收口；Eino 迁移完成；外层 orchestration 和八字内 Graph 已切换为 bounded self-loop；完整改造计划已补齐目标目录树、文件/文件组处置表、逐批验证、失败回退和 pre-mortem。
- **当前任务：** 保持 `orchestration`/`bazi_deterministic` 两个 Graph 的状态机、预算、错误出口和 SSE 合同稳定；共享 repair 已迁入 `backend/internal/repair/`。八字循环控制已在无反向依赖的 `specialists/bazi/graph`，runtime 上下文、证据阶段和确定性投影已按职责切分；事实胶囊、年龄授权和引用目录 DTO 已下沉到无 runtime 依赖的 Bazi domain。Batch 2、Batch 3 已完成；Batch 4 正在完成事件桥接、事件 trace 和 final guard 的同 package 文件重组，Batch 5-7 尚未开始。
- **代码原则：** 普通命理分歧进 `eval/` 数据集和 Langfuse trace，不进运行时专项分支。

## 当前批次

- Batch 0：基线冻结已完成，仅作只读验证。
- Batch 1：已完成 `docs/architecture.md` 与本文件的事实快照更新；完整改造计划已补齐目标树、文件/文件组处置表、逐批验证、失败回退、事实标记、pre-mortem 和统一执行协议。
- Batch 2：已完成。模型调用级 retry 已移到已有 `backend/internal/llm/`，消除 `supervisor -> runtime.ModelCallRetryDecision` 反向依赖；未改变 API、SSE、Graph 拓扑、错误出口或领域语义。
- Batch 3：已完成。`executor.go` 已在同一 `runtime` package 内重组为 `executor_entry.go`、`executor_prefill.go`、`executor_tools.go`；未改变函数签名、API、SSE、Graph 拓扑、错误出口或领域语义。
- Batch 4：已完成。事件桥接、事件 trace 和 final guard 已在同一 `runtime` package 内按职责重组为 `event_bridge.go`、`event_trace.go`、`final_guard.go`；未改变函数签名、API、Graph 拓扑、错误出口、SSE 顺序或 trace 字段。
- 当前停在 Batch 4 完成；Batch 5-7 尚未开始。

## 已验证事实

- 主链：`RouteAdvisor -> Policy Gate -> Manager -> ExecutionPlan -> orchestration Graph loop -> Prefill/dispatch -> aggregate -> Executor final guard -> SSE`。
- `Manager` 是 runtime 内唯一 conversation owner；负责会话焦点、追问策略、执行计划、通用直答和最终综合，不持有完整 ReAct 工具循环。
- Bazi Graph 编译会逐项拒绝缺失 callback；Graph 测试覆盖缺依赖、静态完成、缺盘硬错、步数上限降级和全程大运先于当前动态，runtime adapter 测试保护 Graph phase 不被领域 payload 反写。
- 共享 `internal/repair` 已有独立分类、预算和 HTTP 状态测试；runtime 兼容别名仍保留到所有调用点迁移完成。
- 模型调用级 retry 由 `backend/internal/llm/model_retry.go` 负责；`supervisor/adk_engine.go` 和 `runtime/agent_route.go` 共享 `llm.DefaultModelRetryConfig`，固定 `MaxRetries=2` 和 `ModelCallRetryDecision`。
- Batch 2 focused test、全量 backend test、server build、残余引用审计和 `git diff --check` 已通过；主 agent 已验证 `make eval-smoke` 2/2 通过，trace 为 `ce4c557e92d1eb753c842c14524812df`、`3d809c0d65d6c3cec4b14f3f644bb8ad`。
- `executor_entry.go` 负责执行入口、Graph 调用和 final guard 后的会话收口；`executor_prefill.go` 负责确定性资产预填充；`executor_tools.go` 负责 ToolRunner 薄接入和出生资料参数转换。会话上下文、路由快照、指导状态同步仍由 `executor_context.go` 负责，未改变调用签名、SSE 顺序或最终 guard 边界。
- Batch 4 focused runtime test、全量 backend test、server build、旧路径引用审计和 `git diff --check` 已通过；`make eval-smoke` 2/2 通过，trace 为 `347e9dc04bd90b5dc8fd41316d882fea`、`5f9dba08bf8e170973d6ac14011b864d`，均观察到 `sse_emit` 与 `contract_gate`。
- 八字 Graph 运行职责已按边界拆分：`bazi_graph_entry.go` 负责内图选择和领域失败归一，`bazi_charter_graph.go` 负责补证、审计、阶段事件和最终 writer 适配，`bazi_contract_validation.go` 负责静态/动态合同，`bazi_final_contract.go` 负责最终文本合同，`bazi_model_runtime.go` 负责分析规划、提示构建和内层 agent 适配；证据规划、受控检索、引用归并和有限补证仍由 `bazi_evidence_runtime.go` 承载，函数签名和 Graph 拓扑不变。
- `bazi_projection_views.go` 负责阶段摘要、模型输入 payload、核心命盘/动态事实和年龄范围投影；它只格式化已验证事实，不新增命理裁断。
- `specialists/bazi/domain/` 负责无运行时依赖的事实胶囊、中文事实视图、年龄授权范围和稳定引用目录 DTO；runtime 只负责把图状态映射为 `FactInput`/`SubjectContextInput`/`ReferenceCatalogInput`，保留既有调用合同。
- checkpoint/resume 从未接入 handler、container 或 server 启动路径，已移除；运行时不维护无调用方的 Eino checkpoint store。
- `ExecutionPlan` 只保留带 owner、subject 和历法规则的 `ArtifactRequirement`；`ExecutionSnapshot.RequiredArtifacts` 仅是 handler、trace 与调试的观测投影。
- runtime 先解析对象、合并 ProfileRevision，再生成 ExecutionPlan；prefill 写入资产的 owner 必须与 ArtifactRequirement 对齐。
- target_subject 只允许切换明确人物或既有非主题对象；婚姻、事业、财运等问题范围进入 question_text/current_topic，不能创建新的 Subject。
- 八字资产合同使用 tools/bazi.CalendarRuleVersion 作为历法规则单一来源；validatePlanArtifacts 继续严格匹配 owner、subject 和 calendar rule。
- 八字、奇门、紫微已接入同一 runtime；specialist 只返回领域结果，不能直接拥有最终答复权。
- 资产合同为 `Subject -> ProfileRevision / Case -> DomainAsset -> ActiveFocus -> ArtifactRequirement -> Prefill`；自己、孩子、资料修订和新的奇门问事相互隔离，新的奇门盘统一写入 `qimen_case_chart` 并绑定 Case owner。
- follow-up 解读复用 `InterpretationAsset`，必须绑定当前精确命盘引用；不能按领域复用其他对象、旧资料版本或旧 Case 摘要。
- 本地开发入口：`make dev`（含 Langfuse）或 `make dev-core`（三服务）；检查用 `make status`，重启用 `make restart` / `make restart-core`。
- 聊天页排障入口统一为 Run Inspector：后端只发送 `run-inspection` component；旧 `process-panel / debug-trace / execution-tree` 投影和前端旧面板已下线。全量 TurnTrace 通过本地 `GET /api/debug/traces/:trace_id` 懒加载，需 `DEBUG_HTTP=1` 且依赖 `DEBUG_TRACE=1` 落盘。
- 当前主工作区已迁到 WSL：`/home/huang/workspace/suanming-agent`；关联资料在 `/home/huang/workspace/research`，Agent 技能库在 `/home/huang/workspace/agent-engineering-guide`。
- 官方回归入口：`make regression`；八字质量合同入口为 `make eval-bazi-quality`，并已纳入 `make eval-suite`。
- 后端配置来源为 `backend/.env`；Docker 应用入口为 `deploy/app/`。
- `.gitignore` 分层：外层 `/mnt/d/Workspace/.gitignore` 放跨项目/个人工具规则；本仓库根 `.gitignore` 放仓库级规则；`web/`、`knowledge/`、`deploy/app/` 保留子项目规则。
- `eval/reports/`、`eval/annotation/*.json` 和含具体个人样例的新 eval 数据集默认是本地产物，不进入普通提交；需要固化为合同样本时再显式强制加入。
- 当前普通 `LLM_*` 使用 `deepseek-v4-flash + https://api.deepseek.com/chat/completions` 的 `text/json_object`。已确认该路径拒绝 native `response_format.type=json_schema`；本项目不再以 provider-native Strict 为阻塞条件，而以 `json_object + Schema 注入 prompt + Go Schema 校验 + 严格解码 + 事实引用校验` 实现结构化合同。DeepSeek Responses/Beta strict 是未来独立评估项，不进入当前改造。实施方案见 `docs/strict-json-schema-implementation-plan.md`。

## Graph 主链事实

- 外层 Graph 拓扑：`preflight -> decide_next -> (prefill | dispatch_batch -> aggregate | terminal | terminal_error)`，编译上限 `WithMaxRunSteps(16)`，图名保持 `orchestration`。
- 八字内 Graph 拓扑：`bootstrap -> decide_next -> (analysis_plan | evidence_action -> validate_evidence | static_judgment | lifetime_dayun_judgment | dynamic_judgment | repair -> contract_check | recover_facts -> render | hard_error)`，编译上限 `WithMaxRunSteps(24)`，图名保持 `bazi_deterministic`。
- `decide_next` 是两层 Graph 的唯一动作选择器；evidence、prefill、dispatch、repair 和 business retry 都以 state 计数，不在节点内部隐式结束 Graph。
- `final_guard` 已移出 Graph：`Executor.Execute` 在 `Invoke` 返回后执行 guard、发送唯一最终 `text`、保存 follow-up artifact 并调用 `Manager.FinishTurn`。
- guided fallback 强制切换到奇门时，Graph 会重建 event-question `ExecutionPlan`、同步待执行步骤，并由终态计划执行 final guard；不会按旧八字计划 dispatch 或拦截奇门结果。
- Graph state 只存单轮可描述值；Session、Executor、model client 和 SSE sink 通过 context 注入，不接入 checkpoint。
- 排盘、藏干、透干、大运边界和标准关系可复算；runtime 不注入默认 rule profile，也不从 Go 代码生成 claim。
- V2 模型 DTO 仅为 analysis_plan、evidence_plan、static_judgment、dynamic_judgment；静态模型为四个固定槽位输出受长度与引用合同约束的短裁断和状态，动态模型必须回填唯一 `current_period_ref`。
- runtime 派生 evidence status、层次资格和大运事实对齐；renderer 只转写已验证的结构化投影，不重新裁断。
- 证据质量按 A 级主题逐题验收，输出 `required_topics / covered_topics / missing_topics / degraded_topics`；B 级命例不能替代格局、调候、病药等主证。
- 反思仅重试缺失或高冲突的 A 级主题；查询采用稳定的“典籍 + 主题”并与首轮证据合并。
- 静态综合不得以“月令本气未透”单因推出暗格、清浊或层次降级；候选路线必须比较透干、藏干层级、根气、时令与结构闭环。
- 静态综合保留 `pattern_adjudication` 候选矩阵；月令候选不能仅因未透被拒，藏支组合不能无完整比较越级。
- 静态层次固定观察主轴、有情、有力、清浊、病、药、救应、调候与何知章印证。`rated` 的 1-9 级要求清浊、病药、救应、破格风险、何知章五项独立证据齐全；主证不全但核心事实和主轴成立时输出 `provisional` 的 3-6 级；只有核心事实或静态主轴无法建立时才 `withheld=0`。runtime 只限制证据上下限，不以印星根气等单因直接算等级。
- canonical 通过确定性投影、事实引用和静态/动态合同校验后才进入 renderer；事实/方法冲突硬失败，结构或授权缺口按 recovery policy facts-only，当前没有 `model_partial` 成功态。
- 动态综合接收由出生年和目标流年推导的 `subject_context`，并以 `allowed_outcome_domains / outcome_domains` 做结构化授权。
- 未成年人只允许结构、成长环境、照护节奏和可观察发展；遗漏或越权进入 violation 重试，不能靠扩张自然语言禁词表兜底。
- V2 动态只裁断当前大运与流年，`current_period_ref` 必须等于 runtime 绑定 period，`current_period_realization` 只可为 `repair|assist|maintain|disturb|suppress`；动态 assertions 只从模型实际的当前运裁断生成，并按已声明干支回查完整大运目录索引，不能把稀疏裁断数组位置当作大运索引，也不要求模型覆盖全量大运目录。
- 静态层拥有本命基础结构；完整命盘新增独立 `lifetime_dayun_judgment`，逐步覆盖全部已计算大运并输出全程运路；动态层仍只拥有当前大运、流年走势与承接状态。三层互不改写，最终成文按“强弱/调候/格局视角（含命格层次、古籍参照、断语所限）→ 全程运路 → 当前应期 → 末尾总览结论”呈现；动态仍不得伪造本命事实或改写全程逐运结论。
- 动态流年 assertion 必须绑定 runtime 已选 current_dayun；大运引用仅作 trace provenance，renderer 投影为干支、年龄与已计算关系，不能泄露 `dayun[0].gan_zhi`。
- 原局官星未透时，静态合同拒绝把“伤官见官”写成既成限制；涉及岁运引动只能由动态综合按当前大运说明条件风险。
- renderer 在所有最终文本出口删除内部引用路径；总览结论置于完整报告末尾，收束主轴、层次、可发挥处、限制、发挥方向和当前阶段；格局、强弱、调候和层次各自只呈现所属裁断；动态 facts-only 只展示已绑定当前大运或明确未定位，不把全量人生大运目录伪装为动态解读。
- 冲、刑、害只作 relation trigger 事实，不直接推出医疗、法律、财务事故等具体应事。
- `bazi_rule_profile.go` 已删除；不存在 `defaultBaziRuleProfile`、`applyZipingBasicClaims`、`applyZipingMonthJieClaim` 或运行时调候 overlay。
- recovery_decision 使用显式状态机：canonical parse failure 可全量 facts-only；静态仅证据越权可 facts-only；动态仅领域越权可 facts-only；事实冲突和方法合同冲突默认 hard error。
- facts-only 输出由 runtime 生成并标记 clean contract audit；候选模型文本被丢弃，FieldAudit / RecoveryReason 保留降级原因。
- 未成年人静态 projection 会把未授权成人现实落点收束为结构、成长环境、照护节奏和可观察发展，不让候选文本进入 static renderer。
- 动态合同把投资建议类文本视为未授权财务建议，触发 dynamic facts-only 或硬错路径，不保留违规候选文本。
- trace 记录 `orchestration.loop_step / next_action / termination_reason` 和 `bazi.loop_step / next_action / termination_reason`，并保留 `bazi.internal_graph.node / branch / recovery_state`、`bazi.contract.finding_code / failure_class / recovery_policy`、`bazi.static.source`、`bazi.dynamic.source`、`bazi.final.audit_result`。
- 普通出生资料八字请求不会因 supervisor 猜测自动扩展到紫微；只有用户显式提到紫微时才保留明确要求的紫微路径。阶段运势是八字 primary + 紫微 support，具体事件是无出生资料要求的奇门 primary，健康风险是八字 primary + 紫微 support + final guard 免责声明。

## 历法与确定性事实

- 出生时分以原始消息为准：路由会补回明确分钟，`bazi_calc` 与 `yongshen` 共享 `zi_zheng_true_solar_v2` 口径。
- 真太阳时使用出生地经度与当天均时差；已识别城市映射近似中心经度，用户显式经度优先。
- 子正换日回归样例覆盖“23 点后仍按原日”和“真太阳时校正后跨日”两类边界；具体出生时间只保留在 eval fixture 中，旧版本缓存会自动重排。
- 奇门主链按本轮问事时刻起时家奇门盘，不再用会话出生资料排问事盘；默认盘式为拆补法转盘八门八神 `rotating_8`，输出值符宫和值使宫独立字段。`qimen_dunjia` 的工具合同和 Eino 适配器只接受 `question_time`。
- `TurnContext.QuestionTime` 在本轮入口捕获一次；Manager 是唯一 Case 创建 owner，Case.EventTime、qimen payload.question_time 和 QuestionTime 保持一致。Qimen specialist 使用只含当前 Case 盘和问事事实的最小 Session view。
- 八字大运结果包含出生分钟、顺逆和依据、起运时刻、每步日期边界；`dayun_analyzed` 透传每步日期边界，流年优先按真实交运日判断。
- 大运结果新增 `branch / branchHiddenStems / branchTenGods / branchMainTenGod`；动态模型只能引用这些确定性地支十神字段，不能自行推算。
- `balance_evidence_v1` 同时输出扶身与泄耗克身证据；双方接近时保留“中和附近”，不得被 runtime profile 以一分差重写为身强/身弱或固定喜忌。
- 伤官格的官透/官藏分开输出，组合关系只作候选或受阻事实。

## 错误处理与评测事实

- `Executor` 将裸 graph error 归一为 `RuntimeFailure`；`Orchestrator` 对非取消、非等待确认的失败统一发 `error` 后再发 `done`。
- `RuntimeFailure` 包含 code、retryable、degraded 和用户可见消息。
- 最近一次 REQUIRED_ARTIFACT_UNAVAILABLE 已修复：旧污染 session 回放「紫薇斗数 看一下 婚姻」与「那本月运气如何」均返回 text + done 且无 error；新 session 单轮带资料紫微婚姻也通过。
- 调候旧问题已定位为过渡期自然语言短语表与静态 claim 合同双重校验冲突；已删除短语表，保留已覆盖证据却声明缺失的通用合同校验。模型合法输出“调候不足”等表述不会再因未命中固定短语降级。
- 本轮 `bazi-quality-v1` 真实回放 2/2 通过，静态/动态/最终审计均为 clean；`bazi-answer-quality-v1` 两次回放均为 2/3，未通过项是无正文响应导致的“强弱”缺失，重试时失败样本轮换，单独重试可通过，仍需观察在线模型稳定性。
- 2026-08-10 本轮结构重构验证：Graph/adapter 定向测试、`go test ./backend/internal/runtime -count=1`、服务编译和沙箱外 MCP httptest 均通过；沙箱内全量测试仅因本地端口监听限制无法运行 MCP 用例，其余后端包通过。证据阶段、确定性投影、合同校验、Graph 入口和模型适配拆分均未改变 Graph、SSE 或动态只拥有当前大运的边界。
- 2026-08-10 domain/runtime 边界收口验证：`specialists/bazi/domain` 已拥有事实胶囊、中文事实视图、年龄授权和引用目录 DTO；runtime 只保留窄状态适配、catalog allow-list 与合同兼容入口。引用注册表、统一合同错误和 final writer 合同已按职责拆文件；授权环境下 `GOCACHE=/tmp/suanming-go-cache go test ./backend/... -count=1` 与 `go build ./backend/cmd/server/` 均通过。
- 2026-08-10 动态模型若把内部引用路径写入用户可见字段，runtime 会丢弃候选动态文本并以 facts-only 保留静态结论；当前大运绑定、事实冲突等真正方法合同仍为 hard error。用户资料回放 `trc_36657e827ebd` 已返回唯一 text + done、无 error，最终审计为 clean。
- 共享 Repair Harness 合同已迁入 `backend/internal/repair/`：包含 failure class、action、policy、budget 和 attempt 记录；runtime 仅保留兼容别名。八字 static/dynamic validator 错误映射为机器可读 contract failure；`fact_conflict` / `method_contract` 不调用模型 repair；业务 repair 受单阶段一次和全局预算约束。
- 静态主轴必须保持 `yongshen.geju_candidate` 的确定性主格框架；模型可解释伤官佩印等成局路线，但若把主轴改写为另一命名格局，静态合同按 `fact_conflict` 拒绝。回归覆盖“伤官格(官未透) 被写成建禄月劫”的历史输出。
- 新增 `make eval-bazi-stability`，底层使用 `bazi-stability-v1` 和 `--repeats 10`，报告写入 `eval/reports/bazi-stability-v1.json`。
- 八字 `consistency_flags` 固定为 `吉中有阻 / 机会伴随强变动 / 限制仍在 / 仅作结构观察`；二次仍非法时 runtime 本地收束为保守结构观察。
- `bazi_validation_recovery.go` 不包含命理专项短语触发的修复分支，不保留违规候选文本；具体样例进 eval fixture。
- 已固化“自己 -> 孩子 -> 修订孩子时辰 -> 回到自己 -> 两次奇门新问”回归，覆盖对象隔离、修订失效、解读来源和 Case 隔离。
- 近期运势回归覆盖 period/event/health/natal 四类分类、主次 dispatch、Qimen Case owner/time、出生字段隔离、rotating_8 异常符号、specialist tool 白名单和前端 Case 元信息/warning/Markdown 复制。
- `DynamicFacts` 只作为本轮 Prefill 投影；流月当前固定为 `unavailable` 或 `degraded`，仅在执行计划明确有 `TimeScope` 时由最终回答说明缺口，不能由模型补算；静态或结构追问不追加无关的流年/流月缺口。
- 静态 DTO 已收窄：模型只能输出四项 4-80 字的静态短裁断、状态、事实/规则引用和九级状态；不接收原局 relation ID、自由边界、限制或推理文本。runtime 依据 `BaziFactCapsule` 投影这些边界，避免未声明 `relation.natal.*` 耗尽唯一修复机会。
- Batch 3 已验证：focused runtime 测试、`go test ./backend/internal/runtime -count=1`、授权环境下 `go test ./backend/... -count=1`、`go build ./backend/cmd/server/` 和 `git diff --check` 均通过；`make eval-smoke` 在授权环境 2/2 通过，评测器检查 SSE `done`，trace 为 `96bb5df8a7b3d321701f811646d722b5`、`e44cf23c596448ae5c336186a1576e53`。
- 本轮重启后 `make eval-bazi-quality` 已通过 1/1；trace `ca1c85b60f5e47ade382592730161765` 验证静态合同 clean、动态合同 clean、最终审计 clean，首步大运未交运时动态 source 为 `facts_only_degraded`。
- 新增 `make eval-bazi-answer-quality`，使用 `bazi-answer-quality-v1` 的确定性质量检查拦截内部状态泄露、保守话术过密、未成年人越权和 facts-only 冒充完整解读；runner 可用 `--include-response` 在报告中保存抽取后的正文。
- Langfuse 评测体检入口：`python3 eval/runner/check_langfuse_setup.py`；本机 `eval_*` 与 `answer_*` ScoreConfig 类型已通过校验。
- 本地 Langfuse v3 支持 `/api/public/llm-connections`，但当前 `LLM Connections` 数量为 0；Python `langfuse` SDK 未安装。
- 答案质量 judge 入口：`python3 eval/runner/run_answer_quality_judge.py`；默认只写 JSON 报告，人工确认后才用 `--write-scores` 写 `judge_*` score。
- 当前 `eval/reports/answer-quality-judge-v1.json` 显示 10 条有效、9 条 exact match；不要默认全量跑 judge，先用 `--limit 2` 或 `--case-id` 控制成本。

## 当前风险与待证实项

- `bazi-quality-v1` 是运行时合同评测，不等于完整命理文本质量或用户满意度评测。
- 1991 命例已在本轮真实 SSE 返回完整章节且无 `error`；此前 `natal_risk_status` 事实冲突已由 runtime 按官星透藏事实收束为 `withheld`。
- 新鲜两轮 smoke 的首轮建盘 trace `trc_01a3138b8ca0` 因模型把静态首条 claim 标为 `candidate` 触发既定 `method_contract -> hard_error`；这不是本次追问 renderer 修复，仍需单独处理静态合同稳定性。
- `bazi-stability-v1` 尚无本轮新报告：它会在十次串行调用全部结束后才写入报告，前次长调用期间被人为停止。取得新的十轮稳定性结论仍需允许完整在线运行。
- `bazi-answer-quality-v1` 已完成真实在线回放；答案质量合同本身通过的样本无质量违规，剩余风险是偶发无正文响应，不属于最终 renderer 格式失败。
- 动态 baseline 已删除完整趋势生成：模型动态综合失败时，只展示已绑定当前大运的干支、年龄、日期边界、运干十神和已计算关系；无法定位时明确说明，不能回退成全量人生大运目录。
- `current_dayun` 为空或过期时仍按透传日期边界回补；若目标时刻早于首步交运，明确显示“尚未交入第一步大运”。
- cheap gate 仅允许 `period_fortune` 或单域 `natal_chart` 的同域普通追问复用；具体事件、健康、方法、时间和跨域变化均回完整 Supervisor，不重新分类或扩域。
- 规则材料来自 prompt、知识库检索和未来数据驱动规则表；Go runtime 不承担子平、穷通或逐运趋势 claim provider 职责。
- V2 字段审计以类型化 `BaziSemanticPolicy` 为准，不扫描最终中文判断。trace 额外记录 `bazi.tier.{status,level,evidence_complete}` 与 `bazi.dynamic.{current_period_ref,current_period_realization}`。
- `case_005_2025_topic_coverage_and_age_scope` 只验证证据主题覆盖、主轴一致性、子正换日分钟与幼儿年龄边界；运行时代码不包含该命盘、session、trace 或目标格局分支。

## 下一步

- 保持当前批次停在 Batch 3；如后续单独批准，再开始 Batch 4 的事件桥接、事件 trace 和 final guard 文件重组。不得自动进入下一批。
- 结构化输出合同：V2 只调用 BaZi `analysis_plan`、`evidence_plan`、`static_synthesis`、`dynamic_synthesis` 四类 JSON Mode DTO。每份 Schema 通过 go:embed 进入同一个 registry；prompt 注入、原始 JSON 校验和 hash 都消费对应文件原文，Go DTO 只负责严格解码后的语义校验。链路为 registry -> prompt -> gojsonschema -> DisallowUnknownFields + EOF -> DTO/引用 catalog；catalog 向模型提供 `{id, hint}` 供选择，输出仍只允许 ID。静态节点输出固定槽位的短裁断和事实/规则引用，边界与限制由 runtime 投影；动态节点才允许岁运关系引用。空输出、fence、缺字段、错类型、未知字段与 trailing JSON 均拒绝。未知引用允许一次本节点定向 repair；事实值冲突和方法合同冲突仍硬失败。ADK output tool 维持 InferTool/ReturnDirectly 与既有 optional/Normalize 语义；Manager、ExecutionPlan、Prefill、outer graph、Qimen/Ziwei 自由文本与 SSE wire shape 均未改。json_object 不是 provider-native strict schema；动态未授权范围按既定 facts-only policy 降级。
- 保持 `backend/internal/specialists/bazi/graph/` 的 Graph state、Pregel 拓扑、动作选择、repair 预算和终止不依赖 `internal/runtime`；事实胶囊、年龄授权和引用目录 DTO 已迁入 domain。catalog allow-list、projection、合同、recovery 和 renderer 只有在能定义稳定窄 DTO 时再迁移，不为缩短文件制造双轨模型。
- 真实回放已验证八字追问的 `Graph Invoke -> final guard -> 唯一 text -> done` 收口，以及静态直接回答回退和动态资料提示边界；primary/support、cancel、facts-only 和 mixed-domain 的完整回放仍待补齐，并继续确认 `bazi.loop_step`、`bazi.next_action`、`bazi.termination_reason` 可在 trace 检索。
- 流月确定性工具和完整紫微流月复核仍未实现；继续维持 `DynamicFacts{status: unavailable|degraded}` 合同，不把占位实现描述为已完成。
- 让 specialist 返回 Claim 与来源引用，前端命盘卡继续补对象、资料版本和解释来源。
- 若要恢复子平、穷通或盲派规则，先做数据驱动规则表与 eval fixture，再接入 synthesis 输入；不得新增 Go runtime 专项 case。
- 增加真实的多对象比较合同，不能用单个 `ActiveFocus` 冒充比较。
- 继续把旧 static/dynamic synthesis 桥接字段从 renderer 路径中收缩到 projection 兼容层；不得让模型同时维护 canonical 与 legacy 双轨语义。
- 继续增加 mixed-domain 与复杂追问回归，特别覆盖 rule-profile 未实现范围的降级输出。

## 核心入口

- 运行时：`backend/internal/runtime/manager.go`、`orchestration_graph.go`、`orchestration_graph_loop.go`、`artifact_resolver.go`、`specialist_runner.go`、`event_bridge.go`、`event_trace.go`、`final_guard.go`。
- 资产状态：`backend/internal/state/session.go`、`assets.go`。
- 八字：`backend/internal/runtime/bazi_internal_graph.go`、`bazi_graph_loop.go`、`bazi_graph_entry.go`、`bazi_charter_graph.go`、`bazi_contract_validation.go`、`bazi_final_contract.go`、`bazi_model_runtime.go`、`bazi_canonical_synthesis.go`、`bazi_final_renderer.go`；事实胶囊、年龄授权和引用目录 DTO 在 `backend/internal/specialists/bazi/domain/`；确定性排盘在 `backend/internal/tools/bazi/`。
- repair：`backend/internal/repair/`。
- 评测：`eval/datasets/*.json`、`eval/runner/run-agent-regression.sh`、`eval/runner/run_langfuse_eval.py`、`eval/README.md`。
- 架构与验收：`docs/architecture.md`、`docs/bazi-graph-current-snapshot.md`、`docs/acceptance-criteria.md`、`eval/README.md`。
- Strict Schema 实施：`docs/strict-json-schema-implementation-plan.md`。

## 最小验证

```bash
go test ./backend/... -count=1
go build ./backend/cmd/server/
python3 -m unittest eval/runner/test_run_langfuse_eval.py -v
make eval-repair
make status
```

## 仍有效决策

- 统一架构保持 `thin supervisor + manager-owned runtime + bounded specialists`。
- Manager 不升级为开放式 ReAct 主控；综合领域或多工具问题走多域 `ExecutionPlan`。
- 路由审批与执行分离；`ArtifactRequirement` 持有精确 owner、subject 和历法规则，`RequiredArtifacts` 仅为迁移兼容。
- prefill 按 artifact 合同准备，不按主领域猜测；final guard 仅做最后保险。
- 纯八字保持 authority-first，renderer 只消费上游结构化结果，不隐藏语义路由。
- 知识库只返回证据片段，解释与最终答复归 runtime。
- 评测以合同测试、数据集和结构化报告为准；Langfuse 是观测、dataset run、score 归档和可选平台 evaluator 层。
