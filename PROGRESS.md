# 项目状态

> 当前事实快照，不记录实施流水。历史过程查 Git、专项设计文档和 `eval/reports/`。

## 当前阶段

- **最后更新：** 2026-08-06
- **阶段：** v1.5 收口；Eino 迁移完成；多对象资产解析 Phase 1-3 已落地。
- **当前任务：** 近期运势综合判断的四类路由、Case 问事盘、健康安全边界和 Qimen 前端展示已落地并完成最小验证；全局 Repair Harness Phase 0/1/3/4/6 已落地，Phase 5 外层全局化继续暂缓。Strict JSON Schema 迁移已决策、尚未实施。
- **代码原则：** 普通命理分歧进 `eval/` 数据集和 Langfuse trace，不进运行时专项分支。

## 已验证事实

- 主链：`RouteAdvisor -> Policy Gate -> Manager -> ExecutionPlan -> Prefill -> ToolRunner -> specialist runner(s) -> manager compose -> final guard -> SSE`。
- `Manager` 是 runtime 内唯一 conversation owner；负责会话焦点、追问策略、执行计划、通用直答和最终综合，不持有完整 ReAct 工具循环。
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
- 当前 DeepSeek structured 路径仍使用 `response_format: json_object`；`deepseek-v4-flash` 的有效 endpoint 对 `response_format.type=json_schema` + `strict:true` 实测返回 HTTP 400。阻塞是 provider capability，下一步先选择并验证 Strict-Schema-capable endpoint，不得回落 JSON Mode。实施方案见 `docs/strict-json-schema-implementation-plan.md`。

## 八字主链事实

- 八字单域采用 authority-first：`chart_facts -> rule_materials -> canonical_synthesis -> runtime projection -> contract guards -> renderer -> eval`。
- 八字单域在外层 `agent` 节点内运行内部 Eino Graph；外层 `preflight -> prefill -> agent -> final_guard`、Manager owner 和 SSE wire shape 不变。
- 内部 graph 节点为 bootstrap、analysis_plan、evidence、evidence_reflection、evidence_validation、dynamic_evidence、canonical_synthesis、projection、static_validation、repair_decision、canonical_repair、dynamic_validation、recovery_decision、render、done。
- 排盘、藏干、透干、大运边界和标准关系可复算；runtime 不注入默认 rule profile，也不从 Go 代码生成 claim。
- canonical synthesis 只让模型输出最小裁断单元：主轴、强弱、调候、格局、层次、岁运总纲、关键大运、流年、限制和证据引用。
- runtime 派生 evidence status、legacy 展示字段、tier 暂缓文案和大运事实对齐；renderer 只转写结构化投影，不重新裁断。
- 证据质量按 A 级主题逐题验收，输出 `required_topics / covered_topics / missing_topics / degraded_topics`；B 级命例不能替代格局、调候、病药等主证。
- 反思仅重试缺失或高冲突的 A 级主题；查询采用稳定的“典籍 + 主题”并与首轮证据合并。
- 静态综合不得以“月令本气未透”单因推出暗格、清浊或层次降级；候选路线必须比较透干、藏干层级、根气、时令与结构闭环。
- 静态综合保留 `pattern_adjudication` 候选矩阵；月令候选不能仅因未透被拒，藏支组合不能无完整比较越级。
- 静态 assertion 绑定 `evidence_topics / evidence_status`；缺失 A 级主题只能 withheld，tier 缺证据时由 runtime 固定为“证据不足，暂不定级”。
- 静态和动态综合通过确定性校验后，各自经过独立 fast-model 二值合同审计；严重合同错误返回 `RuntimeFailure`，展示性缺漏可保留为 `model_partial`。
- 动态综合接收由出生年和目标流年推导的 `subject_context`，并以 `allowed_outcome_domains / outcome_domains` 做结构化授权。
- 未成年人只允许结构、成长环境、照护节奏和可观察发展；遗漏或越权进入 violation 重试，不能靠扩张自然语言禁词表兜底。
- 动态每步大运声明 `outcome_domains`；canonical 关键大运优先按 `gan_zhi` 对齐已计算 period，`index` 只作明确补充，避免空 index 默认绑定第一步大运。
- 冲、刑、害只作 relation trigger 事实，不直接推出医疗、法律、财务事故等具体应事。
- `bazi_rule_profile.go` 已删除；不存在 `defaultBaziRuleProfile`、`applyZipingBasicClaims`、`applyZipingMonthJieClaim` 或运行时调候 overlay。
- recovery_decision 使用显式状态机：canonical parse failure 可全量 facts-only；静态仅证据越权可 facts-only；动态仅领域越权可 facts-only；事实冲突和方法合同冲突默认 hard error。
- facts-only 输出由 runtime 生成并标记 clean contract audit；候选模型文本被丢弃，FieldAudit / RecoveryReason 保留降级原因。
- 未成年人静态 projection 会把未授权成人现实落点收束为结构、成长环境、照护节奏和可观察发展，不让候选文本进入 static renderer。
- 动态合同把投资建议类文本视为未授权财务建议，触发 dynamic facts-only 或硬错路径，不保留违规候选文本。
- trace 记录 `bazi.internal_graph.node / branch / recovery_state`，并保留 `bazi.contract.finding_code / failure_class / recovery_policy`、`bazi.static.source`、`bazi.dynamic.source`、`bazi.final.audit_result`。
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
- 最近一次八字稳定性问题已修复：原输入「1994年1月21日20点30分 女 南京」连续 10 次真实 `/api/chat` 全部返回 text + done，`bazi.final.audit_result=clean`，无 error。
- 最近一次静态调候投影失败已修复：调候锚点不够具体时保留合同校验，但进入 static facts-only 降级，不再返回 `BAZI_STATIC_PROJECTION_FAILED` SSE error；repair 后再次出现 static strength/balance 反转时，按 `static.strength_balance` 机器可读冲突进入 static facts-only，不新增第二次模型 repair。
- 全局 Repair Harness Phase 0/1/3/4/6 已落地：runtime 已有通用 `RepairFailure` / `RepairPolicy` / `RepairState` / 安全 trace 投影；八字 static/dynamic validator 错误已映射为机器可读 contract failure；模型调用级 retry 只允许 429、5xx、timeout 和空输出，400/401/402、取消和业务错误不 retry；八字 static canonical repair 已接入一次有限回环，`fact_conflict` / `method_contract` 不让模型修；learning hint 只使用代码固化短提示；`runtime-repair-v1` 和 `make eval-repair` 已覆盖最近静态调候投影失败样本，Go 合同测试覆盖 static strength/balance 后续失败恢复和 dynamic validator 分类。
- 新增 `make eval-bazi-stability`，底层使用 `bazi-stability-v1` 和 `--repeats 10`，报告写入 `eval/reports/bazi-stability-v1.json`。
- 八字 `consistency_flags` 固定为 `吉中有阻 / 机会伴随强变动 / 限制仍在 / 仅作结构观察`；二次仍非法时 runtime 本地收束为保守结构观察。
- `bazi_validation_recovery.go` 不包含命理专项短语触发的修复分支，不保留违规候选文本；具体样例进 eval fixture。
- 已固化“自己 -> 孩子 -> 修订孩子时辰 -> 回到自己 -> 两次奇门新问”回归，覆盖对象隔离、修订失效、解读来源和 Case 隔离。
- 近期运势回归覆盖 period/event/health/natal 四类分类、主次 dispatch、Qimen Case owner/time、出生字段隔离、rotating_8 异常符号、specialist tool 白名单和前端 Case 元信息/warning/Markdown 复制。
- `DynamicFacts` 只作为本轮 Prefill 投影；流月当前固定为 `unavailable` 或 `degraded`，最终回答说明缺口，不能由模型补算。
- 最新目标验证已通过：`go test ./backend/... -count=1`、`go build ./backend/cmd/server/`、`make eval-repair`；`runtime-repair-v1` 最近失败样本 1/1 pass，trace_id=`9f99e38d1a4cd4353a9922fc35b34c30`，并观测到 `static.tiaohou_anchor` 一次 repair 后 facts-only fallback。
- 最新 `bazi-quality-v1` 报告已通过，trace 属性为 `bazi.static.contract_audit=clean`、`bazi.dynamic.contract_audit=clean`、`bazi.final.audit_result=clean`。
- 新增 `make eval-bazi-answer-quality`，使用 `bazi-answer-quality-v1` 的确定性质量检查拦截内部状态泄露、保守话术过密、未成年人越权和 facts-only 冒充完整解读；runner 可用 `--include-response` 在报告中保存抽取后的正文。
- Langfuse 评测体检入口：`python3 eval/runner/check_langfuse_setup.py`；本机 `eval_*` 与 `answer_*` ScoreConfig 类型已通过校验。
- 本地 Langfuse v3 支持 `/api/public/llm-connections`，但当前 `LLM Connections` 数量为 0；Python `langfuse` SDK 未安装。
- 答案质量 judge 入口：`python3 eval/runner/run_answer_quality_judge.py`；默认只写 JSON 报告，人工确认后才用 `--write-scores` 写 `judge_*` score。
- 当前 `eval/reports/answer-quality-judge-v1.json` 显示 10 条有效、9 条 exact match；不要默认全量跑 judge，先用 `--limit 2` 或 `--case-id` 控制成本。

## 当前风险与待证实项

- `bazi-quality-v1` 是运行时合同评测，不等于完整命理文本质量或用户满意度评测。
- `bazi-answer-quality-v1` 当前只完成离线 runner 单测和数据集校验；真实在线评测需后端 :8080 启动后再跑。
- 动态 baseline 已删除完整趋势生成：模型动态综合失败时，每步大运只展示干支、年龄、日期边界、运干十神和已计算关系。
- `current_dayun` 为空或过期时仍按透传日期边界回补；若目标时刻早于首步交运，明确显示“尚未交入第一步大运”。
- cheap gate 仅允许 `period_fortune` 或单域 `natal_chart` 的同域普通追问复用；具体事件、健康、方法、时间和跨域变化均回完整 Supervisor，不重新分类或扩域。
- 规则材料来自 prompt、知识库检索和未来数据驱动规则表；Go runtime 不承担子平、穷通或逐运趋势 claim provider 职责。
- 字段审计已从自然语言词表扩张收窄为合同校验；`canonical_tier_withheld_by_runtime` 是安全投影 note，不计入 `bazi.final.audit_result` repaired。
- `case_005_2025_topic_coverage_and_age_scope` 只验证证据主题覆盖、主轴一致性、子正换日分钟与幼儿年龄边界；运行时代码不包含该命盘、session、trace 或目标格局分支。

## 下一步

- Strict JSON Schema Phase 0：先完成 provider capability probe；通过后才定义 DTO/schema、接入 adapter 并迁移 BaZi Go-consumed node。普通 DeepSeek 文本/tool calling、Manager owner、outer graph、Supervisor output tool Schema、Qimen/Ziwei 自由文本和 SSE wire shape 不是此迁移的改动目标。
- Phase 5 外层全局化继续暂缓；若要推进，先确认八字试点 eval 连续稳定，再把 repair harness 提升到外层 runtime 边界。
- 流月确定性工具和完整紫微流月复核仍未实现；继续维持 `DynamicFacts{status: unavailable|degraded}` 合同，不把占位实现描述为已完成。
- 让 specialist 返回 Claim 与来源引用，前端命盘卡继续补对象、资料版本和解释来源。
- 若要恢复子平、穷通或盲派规则，先做数据驱动规则表与 eval fixture，再接入 synthesis 输入；不得新增 Go runtime 专项 case。
- 增加真实的多对象比较合同，不能用单个 `ActiveFocus` 冒充比较。
- 继续把旧 static/dynamic synthesis 桥接字段从 renderer 路径中收缩到 projection 兼容层；不得让模型同时维护 canonical 与 legacy 双轨语义。
- 继续增加 mixed-domain 与复杂追问回归，特别覆盖 rule-profile 未实现范围的降级输出。

## 核心入口

- 运行时：`backend/internal/runtime/manager.go`、`orchestration_graph.go`、`artifact_resolver.go`、`specialist_runner.go`、`observability.go`。
- 资产状态：`backend/internal/state/session.go`、`assets.go`。
- 八字：`backend/internal/runtime/bazi_charter_graph.go`、`bazi_internal_graph.go`、`bazi_canonical_synthesis.go`、`bazi_final_renderer.go`；确定性排盘在 `backend/internal/tools/bazi/`。
- 评测：`eval/datasets/*.json`、`eval/runner/run-agent-regression.sh`、`eval/runner/run_langfuse_eval.py`、`eval/README.md`。
- 架构与验收：`docs/architecture.md`、`docs/acceptance-criteria.md`、`eval/README.md`。
- Strict Schema 实施：`docs/strict-json-schema-implementation-plan.md`。

## 最小验证

```bash
go test ./backend/... -v
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
