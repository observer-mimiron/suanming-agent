# 项目状态

> 当前事实快照，不记录实施流水。历史过程查 Git、专项设计文档和 `eval/reports/`。

## 当前阶段

- **最后更新：** 2026-07-31
- **阶段：** v1.5 收口；多对象资产解析 Phase 1-3 已落地。
- **当前任务：** 收缩 runtime 控制层：保留 Manager 主链，移除未接线的恢复能力和重复执行合同投影；八字普通质量问题继续进入 eval/trace。

## 已验证事实

- 主链：`RouteAdvisor -> Policy Gate -> Manager -> ExecutionPlan -> Prefill -> ToolRunner -> specialist runner(s) -> manager compose -> final guard -> SSE`。
- `Manager` 是 runtime 内唯一 conversation owner；`ExecutionPlan` 是 route approval 之后的正式执行合同。
- checkpoint/resume 从未接入 handler、container 或 server 启动路径，已移除；运行时不再维护无调用方的 Eino checkpoint store。
- `ExecutionPlan` 只保留带 owner、subject 和历法规则的 `ArtifactRequirement`；`ExecutionSnapshot.RequiredArtifacts` 仅保留为 handler、trace 与调试的观测投影。
- 八字、奇门、紫微已接入同一 runtime；specialist 只返回领域结果，不能直接拥有最终答复权。
- 八字单域采用 authority-first：分析模式、证据规划、受控检索、rule materials、静态/动态综合器、程序 renderer 成文。链路为 `chart_facts -> rule_materials -> static/dynamic synthesis -> minimal_guard -> renderer -> eval`：排盘、藏干、透干、大运边界和标准关系可复算；runtime 不再注入默认 rule profile 或从 Go 代码生成 claim；静态/动态综合器输出裁断，renderer 只成文或展示 facts-only 降级事实。
- 资产合同为 `Subject -> ProfileRevision / Case -> DomainAsset -> ActiveFocus -> ArtifactRequirement -> Prefill`。自己、孩子、资料修订和新的奇门问事相互隔离；活动盘字段只是兼容投影。
- follow-up 解读复用 `InterpretationAsset`，必须绑定当前精确命盘引用；不能按领域复用其他对象、旧资料版本或旧 Case 的摘要。
- 八字大运结果包含出生分钟、顺逆和依据、起运时刻、每步日期边界；`dayun_analyzed` 透传每步日期边界，流年优先按真实交运日判断，历史旧盘才回退虚岁区间。大运工具不自动评分吉凶；当前运缓存缺失时只可按日期边界回补，不能猜测列表项。
- `balance_evidence_v1` 同时输出扶身与泄耗克身证据，不再由单边加分自动断“身旺极/身弱极”；双方接近时保留“中和附近”，不得被 runtime profile 以一分差重写为身强/身弱或一套固定喜忌。伤官格的官透/官藏分开输出，组合关系只作候选或受阻事实。
- 出生时分以原始消息为准：路由会补回明确给出的分钟，`bazi_calc` 与 `yongshen` 共享 `zi_zheng_true_solar_v2` 口径。真太阳时使用出生地经度与当天均时差；已识别的城市会映射到近似中心经度，用户显式经度优先。`2025-11-10 23:30 男，北京` 仍为 `乙巳 / 丁亥 / 癸未 / 壬子`；`2025-11-10 23:53 男，上海` 校正为 `2025-11-11 00:15`，为 `乙巳 / 丁亥 / 甲申 / 甲子`。旧版本缓存会自动重排。
- `bazi_rule_profile.go` 已删除：不再有 `defaultBaziRuleProfile`、`applyZipingBasicClaims`、`applyZipingMonthJieClaim` 或运行时调候单行 overlay。静态模型综合不可用或 hard validation 二次失败时，runtime 才进入完整 `facts_only_degraded`；静态有效而动态失败时保留原局解读，只把大运与流年切成事实展示。
- 静态 synthesis 保留 `assertions`：主轴、强弱、调候、层级和 topic answer 应引用可复算 `fact_refs`；`claim_refs` 仅在输入真实存在对应 claim/verdict 时填写，未知 claim 只进 trace soft audit，不触发降级。
- 动态 synthesis 新增逐运 `dayun_period` assertions，并保留 `dayun_judgments` / `dayun_path` 兼容投影。每个已计算大运 period 都必须有 assertion，且引用 `dayun[index].gan_zhi` 或 `dayun[index].relations`；冲、刑、害只作 relation trigger 事实，不直接推出医疗、法律、财务事故等具体应事。
- 八字校验保留 assertion/violation 合同作为模型输出 hard guard：可证明的事实冲突、大运覆盖缺失、未声明关系事实和直接医疗/法律/伤灾断语会返回机器可读 `baziValidationViolation`。未知 `fact_ref` 路径别名、未知 `claim_ref` 与普通命理措辞改为 trace soft audit，不触发整段重试或降级；第一次 hard failure 把 violation 注入 retry payload，第二次仍失败时只降级对应的静态或动态阶段。
- 运行时错误收口第一阶段已落地：`Executor` 将裸 graph error 归一为 `RuntimeFailure`，`Orchestrator` 对非取消、非等待确认的失败统一发 `error` 后再发 `done`；`RuntimeFailure` 现包含 code、retryable、degraded 和用户可见消息。八字 `consistency_flags` 已收口为固定允许值（`吉中有阻 / 机会伴随强变动 / 限制仍在 / 仅作结构观察`），动态反馈会把允许集合告知模型，二次仍非法时 runtime 本地收束为保守结构观察。
- recovery 已去专项补丁并收缩为 facts-only：`bazi_validation_recovery.go` 不再包含命理专项短语触发的修复分支，不再保留违规候选文本；具体样例进入 eval fixture，不进入运行时代码分支。
- 已固化“自己 -> 孩子 -> 修订孩子时辰 -> 回到自己 -> 两次奇门新问”回归，覆盖对象隔离、修订失效、解读来源和 Case 隔离。
- 本地开发入口：`make dev`（含 Langfuse）或 `make dev-core`（三服务）；检查用 `make status`，重启用 `make restart` / `make restart-core`。
- 官方回归入口：`make regression`。后端配置来源为 `backend/.env`；Docker 应用入口为 `deploy/app/`。

## 当前风险与待证实项

- 动态 baseline 已删除完整趋势生成：模型动态综合失败时，每步大运只展示干支、年龄、日期边界、运干十神和已计算关系，不输出“承托/压力/结构承接”等代码趋势。`current_dayun` 为空或过期时仍按透传日期边界回补；若目标时刻早于首步交运，明确显示“尚未交入第一步大运”，仅在事实缺失时显示未能定位。
- 本地 Langfuse v3 可用于 trace、session、dataset、dataset run 和 score；`Experiments / Evals` 不是主评测入口。
- cheap gate 仅允许保守的同域普通追问复用，不能演化成第二套路由器；继续扩面前要看 `eval/reports/cheap-gate-summary.json`。
- 规则材料当前来自 prompt、知识库检索和未来数据驱动规则表；Go runtime 不再承担子平、穷通或逐运趋势的 claim provider 职责。若要新增规则，必须作为知识/eval/数据表建设，不能回到运行时专项分支。
- 字段审计已从自然语言词表扩张收窄为合同校验：事实值或运序冲突、明确伪造三合三会局、直接医疗/法律/伤灾断语才 hard gate；未知引用别名、普通关系解释、“大吉/大凶/一飞冲天”等命理表达只进入 soft audit 或 eval，不得因词面被清洗。单个命盘或 trace 只能进入回归测试，不能成为业务逻辑专项补丁。

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
- 评测：`eval/datasets/runtime-smoke-v1.json`、`eval/runner/run-agent-regression.sh`、`eval/runner/run_langfuse_eval.py`。
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
- 路由审批与执行分离；`ArtifactRequirement` 持有精确 owner、subject 和历法规则，`RequiredArtifacts` 仅为迁移兼容。
- prefill 按 artifact 合同准备，不按主领域猜测；final guard 仅做最后保险。
- 纯八字保持 authority-first，renderer 只消费上游结构化结果，不隐藏语义路由。
- 知识库只返回证据片段，解释与最终答复归 runtime。
- 评测以合同测试、数据集和结构化报告为准，Langfuse 是观测层。
