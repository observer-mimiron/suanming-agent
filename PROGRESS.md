# 项目状态

> 当前事实快照，不记录实施流水。历史过程查 Git、专项设计文档和 `eval/reports/`。

## 当前阶段

- **最后更新：** 2026-08-01
- **阶段：** v1.5 收口；Eino 迁移完成；多对象资产解析 Phase 1-3 已落地。
- **当前任务：** 收口八字候选裁定、证据主题状态、逐运领域授权与独立语义审计。
- **代码原则：** 普通命理分歧进 `eval/` 数据集和 Langfuse trace，不进运行时专项分支。

## 已验证事实

- 主链：`RouteAdvisor -> Policy Gate -> Manager -> ExecutionPlan -> Prefill -> ToolRunner -> specialist runner(s) -> manager compose -> final guard -> SSE`。
- `Manager` 是 runtime 内唯一 conversation owner；负责会话焦点、追问策略、执行计划、通用直答和最终综合，不持有完整 ReAct 工具循环。
- checkpoint/resume 从未接入 handler、container 或 server 启动路径，已移除；运行时不维护无调用方的 Eino checkpoint store。
- `ExecutionPlan` 只保留带 owner、subject 和历法规则的 `ArtifactRequirement`；`ExecutionSnapshot.RequiredArtifacts` 仅是 handler、trace 与调试的观测投影。
- 八字、奇门、紫微已接入同一 runtime；specialist 只返回领域结果，不能直接拥有最终答复权。
- 资产合同为 `Subject -> ProfileRevision / Case -> DomainAsset -> ActiveFocus -> ArtifactRequirement -> Prefill`；自己、孩子、资料修订和新的奇门问事相互隔离。
- follow-up 解读复用 `InterpretationAsset`，必须绑定当前精确命盘引用；不能按领域复用其他对象、旧资料版本或旧 Case 摘要。
- 本地开发入口：`make dev`（含 Langfuse）或 `make dev-core`（三服务）；检查用 `make status`，重启用 `make restart` / `make restart-core`。
- 官方回归入口：`make regression`；八字质量合同入口为 `make eval-bazi-quality`，并已纳入 `make eval-suite`。
- 后端配置来源为 `backend/.env`；Docker 应用入口为 `deploy/app/`。
- `.gitignore` 分层：外层 `/mnt/d/Workspace/.gitignore` 放跨项目/个人工具规则；本仓库根 `.gitignore` 放仓库级规则；`web/`、`knowledge/`、`deploy/app/` 保留子项目规则。
- `eval/reports/`、`eval/annotation/*.json` 和含具体个人样例的新 eval 数据集默认是本地产物，不进入普通提交；需要固化为合同样本时再显式强制加入。

## 八字主链事实

- 八字单域采用 authority-first：`chart_facts -> rule_materials -> static/dynamic synthesis -> minimal_guard -> renderer -> eval`。
- 排盘、藏干、透干、大运边界和标准关系可复算；runtime 不注入默认 rule profile，也不从 Go 代码生成 claim。
- 静态/动态综合输出裁断；renderer 只转写上游 verdict 或 partial 可展示字段，不替失败综合生成兜底解读。
- 证据质量按 A 级主题逐题验收，输出 `required_topics / covered_topics / missing_topics / degraded_topics`；B 级命例不能替代格局、调候、病药等主证。
- 反思仅重试缺失或高冲突的 A 级主题；查询采用稳定的“典籍 + 主题”并与首轮证据合并。
- 静态综合不得以“月令本气未透”单因推出暗格、清浊或层次降级；候选路线必须比较透干、藏干层级、根气、时令与结构闭环。
- 静态综合保留 `pattern_adjudication` 候选矩阵；月令候选不能仅因未透被拒，藏支组合不能无完整比较越级。
- 静态 assertion 绑定 `evidence_topics / evidence_status`；缺失 A 级主题只能 withheld。
- 静态和动态综合通过确定性校验后，各自经过独立 fast-model 二值合同审计；严重合同错误返回 `RuntimeFailure`，展示性缺漏可保留为 `model_partial`。
- 动态综合接收由出生年和目标流年推导的 `subject_context`，并以 `allowed_outcome_domains / outcome_domains` 做结构化授权。
- 未成年人只允许结构、成长环境、照护节奏和可观察发展；遗漏或越权进入 violation 重试，不能靠扩张自然语言禁词表兜底。
- 动态每步大运声明 `outcome_domains`；每个已计算大运 period 都必须有 assertion，并引用 `dayun[index].gan_zhi` 或 `dayun[index].relations`。
- 冲、刑、害只作 relation trigger 事实，不直接推出医疗、法律、财务事故等具体应事。
- `bazi_rule_profile.go` 已删除；不存在 `defaultBaziRuleProfile`、`applyZipingBasicClaims`、`applyZipingMonthJieClaim` 或运行时调候 overlay。
- synthesis 不可用或 hard validation 重试后仍严重失败时，runtime 返回合同错误；facts-only 不再作为失败综合自动恢复路径。

## 历法与确定性事实

- 出生时分以原始消息为准：路由会补回明确分钟，`bazi_calc` 与 `yongshen` 共享 `zi_zheng_true_solar_v2` 口径。
- 真太阳时使用出生地经度与当天均时差；已识别城市映射近似中心经度，用户显式经度优先。
- 子正换日回归样例覆盖“23 点后仍按原日”和“真太阳时校正后跨日”两类边界；具体出生时间只保留在 eval fixture 中，旧版本缓存会自动重排。
- 八字大运结果包含出生分钟、顺逆和依据、起运时刻、每步日期边界；`dayun_analyzed` 透传每步日期边界，流年优先按真实交运日判断。
- 大运结果新增 `branch / branchHiddenStems / branchTenGods / branchMainTenGod`；动态模型只能引用这些确定性地支十神字段，不能自行推算。
- `balance_evidence_v1` 同时输出扶身与泄耗克身证据；双方接近时保留“中和附近”，不得被 runtime profile 以一分差重写为身强/身弱或固定喜忌。
- 伤官格的官透/官藏分开输出，组合关系只作候选或受阻事实。

## 错误处理与评测事实

- `Executor` 将裸 graph error 归一为 `RuntimeFailure`；`Orchestrator` 对非取消、非等待确认的失败统一发 `error` 后再发 `done`。
- `RuntimeFailure` 包含 code、retryable、degraded 和用户可见消息。
- 八字 `consistency_flags` 固定为 `吉中有阻 / 机会伴随强变动 / 限制仍在 / 仅作结构观察`；二次仍非法时 runtime 本地收束为保守结构观察。
- `bazi_validation_recovery.go` 不包含命理专项短语触发的修复分支，不保留违规候选文本；具体样例进 eval fixture。
- 已固化“自己 -> 孩子 -> 修订孩子时辰 -> 回到自己 -> 两次奇门新问”回归，覆盖对象隔离、修订失效、解读来源和 Case 隔离。
- 最新 `bazi-quality-v1` 报告已通过，trace 属性为 `bazi.static.contract_audit=clean`、`bazi.dynamic.contract_audit=clean`。
- Langfuse 评测体检入口：`python3 eval/runner/check_langfuse_setup.py`；本机 `eval_*` 与 `answer_*` ScoreConfig 类型已通过校验。
- 本地 Langfuse v3 支持 `/api/public/llm-connections`，但当前 `LLM Connections` 数量为 0；Python `langfuse` SDK 未安装。
- 答案质量 judge 入口：`python3 eval/runner/run_answer_quality_judge.py`；默认只写 JSON 报告，人工确认后才用 `--write-scores` 写 `judge_*` score。
- 当前 `eval/reports/answer-quality-judge-v1.json` 显示 10 条有效、9 条 exact match；不要默认全量跑 judge，先用 `--limit 2` 或 `--case-id` 控制成本。

## 当前风险与待证实项

- `bazi-quality-v1` 是运行时合同评测，不等于完整命理文本质量或用户满意度评测。
- 动态 baseline 已删除完整趋势生成：模型动态综合失败时，每步大运只展示干支、年龄、日期边界、运干十神和已计算关系。
- `current_dayun` 为空或过期时仍按透传日期边界回补；若目标时刻早于首步交运，明确显示“尚未交入第一步大运”。
- cheap gate 仅允许保守的同域普通追问复用；继续扩面前要看 `eval/reports/cheap-gate-summary.json`。
- 规则材料来自 prompt、知识库检索和未来数据驱动规则表；Go runtime 不承担子平、穷通或逐运趋势 claim provider 职责。
- 字段审计已从自然语言词表扩张收窄为合同校验；普通关系解释和命理表达进 soft audit 或 eval，不得因词面被清洗。
- `case_005_2025_topic_coverage_and_age_scope` 只验证证据主题覆盖、主轴一致性、子正换日分钟与幼儿年龄边界；运行时代码不包含该命盘、session、trace 或目标格局分支。

## 下一步

- 让 specialist 返回 Claim 与来源引用，前端命盘卡显示对象、资料版本和奇门起局时间。
- 若要恢复子平、穷通或盲派规则，先做数据驱动规则表与 eval fixture，再接入 synthesis 输入；不得新增 Go runtime 专项 case。
- 增加真实的多对象比较合同，不能用单个 `ActiveFocus` 冒充比较。
- 为 assertion 与 synthesis 投影选择唯一 canonical schema 后，再删除旧字段桥接；不得由 renderer 暗中兼容两套语义。
- 继续增加 mixed-domain 与复杂追问回归，特别覆盖 rule-profile 未实现范围的降级输出。

## 核心入口

- 运行时：`backend/internal/runtime/manager.go`、`orchestration_graph.go`、`artifact_resolver.go`、`specialist_runner.go`、`observability.go`。
- 资产状态：`backend/internal/state/session.go`、`assets.go`。
- 八字：`backend/internal/runtime/bazi_charter_graph.go`、`bazi_final_renderer.go`；确定性排盘在 `backend/internal/tools/bazi/`。
- 评测：`eval/datasets/*.json`、`eval/runner/run-agent-regression.sh`、`eval/runner/run_langfuse_eval.py`、`eval/README.md`。
- 架构与验收：`docs/architecture.md`、`docs/acceptance-criteria.md`、`eval/README.md`。

## 最小验证

```bash
go test ./backend/... -v
go build ./backend/cmd/server/
python3 -m unittest eval/runner/test_run_langfuse_eval.py -v
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
