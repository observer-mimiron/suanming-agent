---
name: langfuse-eval
description: 使用本仓库的 Langfuse 评测链路完成 Agent 代码、数据集、trace、dataset run、score 和质量评测。适用于新增或审查 eval case、运行 smoke/检索/八字质量合同、定位评测回归、设计 LLM Judge，或决定一次代码改动需要跑哪些最小评测。
---

# Langfuse 评测

用本 skill 评测 `suanming-agent`。本地 dataset 与 JSON 报告是运行时合同事实；
Langfuse 提供 trace、session、dataset run 与 score，服务于诊断和版本比较。

本 skill 是项目事实层，不是官方 `langfuse` skill 的替代。通用 Langfuse 能力、API、
SDK、CLI、UI 和 evaluator 语义以 `.agents/skills/langfuse` 为准；本 skill 只补充
本项目已经验证的评测入口、数据集合同、ScoreConfig、成本边界和本机 Langfuse 限制。

## 官方 Langfuse 对齐规则

涉及 Langfuse SDK、API、CLI、evaluator、dataset run 或 UI 行为时，先读取
`.agents/skills/langfuse/SKILL.md` 及相关 reference；需要写代码或改调用方式时，
再获取当前官方文档，不能凭记忆实现。

查询或修改 Langfuse 对象时，优先用官方 CLI 做 schema/参数发现：

```bash
npx langfuse-cli api __schema
npx langfuse-cli api <resource> --help
npx langfuse-cli api <resource> <action> --help
```

已有项目脚本覆盖的稳定端点仍优先复用；新增端点、参数或批量操作前必须先确认
CLI help 或官方 API 文档。需要密钥时只读取 `.env` 或 shell 环境，要求用户自行设置，
禁止让用户把 secret 粘贴到对话里。

如果用户明确要 Langfuse Experiments UI 中可比较的结果，优先按官方
`judge-calibration` reference 设计 SDK dataset experiment；本项目 runner 仍负责
CI、合同断言和 JSON 报告。不要因为官方推荐 SDK experiment，就把本地合同主链迁移到
平台 UI。

当任务是“为什么输出差、失败怎么分类、该修什么”时，读取官方
`references/error-analysis.md`，按 sample selection → open coding → clustering →
labelling → deciding what to fix 做错误分析；不要把单条坏例直接变成 prompt 或 runtime
专项补丁。若 trace 级 input/output 为空，优先检查 GENERATION observation。

使用 annotation queue 前先查已有 ScoreConfig；queue 创建后不可更新或删除，所以必须先
确定 score configs，再建 queue。需要新增标注维度时建新 queue，而不是假设能原地修改。

## 先判定范围与成本

每次先说明本次验证的是“确定性合同、在线 smoke、检索合同、八字质量合同、
答案质量”中的哪一项。不要把 smoke 通过说成答案质量通过。

评测成本包括模型费用、等待时间、Langfuse 数据噪声和非确定性。默认跑最小足够
集合，不因普通代码修改跑全量在线评测。

| 改动范围 | 必跑 | 何时加在线评测 | 不要做 |
|---|---|---|---|
| 文档、skill、静态配置 | skill 校验、相关格式检查 | 不加 | `make regression` / `make eval-suite` |
| 单个确定性 Go 包 | 受影响包的 `go test` | 改动会改变 `/api/chat` 可见合同才加 | 直接跑全量 suite |
| runner 或 dataset schema | runner 单测、JSON 解析 | 改了真实请求、trace 选择或断言时，跑最小相关 dataset | 手写临时轮询脚本 |
| 路由、Manager、SSE、session、trace | 受影响 Go 测试 | `make regression` 或最小 smoke | 未定位前重复全量运行 |
| 检索 | 受影响测试 | `retrieval-benchmark-v1` | 用普通 smoke 代替检索合同 |
| 八字综合、prompt、证据、年龄边界 | 受影响测试 | `make eval-bazi-quality` | 把单个样例写成 runtime 特判 |
| 跨域改动、模型/提示词/发布前 | 全部必要确定性测试 | `make eval-suite`，必要时同配置重复受影响样本 | 在每个小改动后跑 suite |

当前 runner 没有 `--case-id`。若最小 dataset 仍包含多个 case，使用最小现有 dataset；
不要复制 runner 来省调用。只有频繁需要时，才把 case selector 作为正式 runner 能力，
连同单元测试一起实现。

开发排障默认不写 score；只有要保留可比较证据时才加 `--write-scores`。一次模型或
prompt 变更先跑一轮；只有需要区分模型波动和确定性退化时，才对失败 case 做有限重复。

## 固定工作流

1. 读 `PROGRESS.md`、`eval/README.md` 和目标 dataset，确认当前能力与已知限制。
2. 按上表选最小命令；在线运行前先执行 `make status`。
3. 每个 case 必须使用唯一 `session_id`；有 `setup_message` 时，正式追问必须排除 setup trace。现有 runner 已实现，禁止另写轮询。
4. 先看结构化合同：`route`、`task_intent`、`turn_type`、SSE `done`、observation、trace 属性、artifact/guard；文字断言只能补充稳定的展示或安全合同。
5. 失败时依次看：报告错误 → session/trace 对应关系 → route/task/turn → tool 与检索 → synthesis/guard → 最终 SSE/文本。
6. 修复后只重跑最小复现，再跑受影响 dataset。保留前后报告和 trace id；生产坏例进入 dataset，不进入专项业务分支。

## 命令

### 纯代码合同

```bash
go test ./backend/internal/<受影响包> -v
python3 -m unittest eval/runner/test_run_langfuse_eval.py -v
```

只在确有跨包影响时执行：

```bash
go test ./backend/... -v
```

这些命令不证明 `/api/chat`、trace 或 SSE 的真实行为。

### 最小在线 smoke

确认服务可用后运行，并写入可检查报告：

```bash
make status
python3 eval/runner/run_langfuse_eval.py \
  --dataset-path eval/datasets/runtime-smoke-v1.json \
  --server-url http://localhost:8080 \
  --langfuse-url http://localhost:3001 \
  --report-path eval/reports/runtime-smoke-v1-local.json
```

需要把本轮成功合同登记为 Langfuse score 时才添加 `--write-scores`。

### 集成回归与专项合同

```bash
# 受影响 runtime 合同：选定 Go 测试 + 真实 smoke
make regression

# 检索链路
python3 eval/runner/run_langfuse_eval.py \
  --dataset-path eval/datasets/retrieval-benchmark-v1.json \
  --server-url http://localhost:8080 \
  --langfuse-url http://localhost:3001 \
  --report-path eval/reports/retrieval-benchmark-v1-local.json

# 八字候选裁定、证据主题、年龄边界与审计属性
make eval-bazi-quality

# 跨域改动或发布前，才跑全部本地 dataset
make eval-suite
```

`make regression` 是集成命令，会发起真实 runtime smoke，不是单纯 unit test。检查
`/tmp/suanming-agent/runtime-smoke-report.json`，不能只看退出码。

### 可比较的 Langfuse dataset run

只在模型、prompt、配置或候选实现需要比较时运行。run 名必须包含 dataset 版本和
revision，报告必须落盘：

```bash
python3 eval/runner/run-langfuse-experiment.py \
  --dataset-path eval/datasets/runtime-smoke-v1.json \
  --server-url http://localhost:8080 \
  --langfuse-url http://localhost:3001 \
  --run-name runtime-smoke-v1-<revision> \
  --write-scores \
  --report-path eval/reports/runtime-smoke-v1-experiment.json
```

本项目的 self-hosted Langfuse v3 以 dataset-run item 为已验证路径；不要把
Experiments/Evals UI/API 当成前置条件。

## Dataset 合同

case 放在 `eval/datasets/*.json`，一例只证明一个稳定行为。基础字段：

```json
{
  "id": "runtime-smoke-bazi-main",
  "message": "男，1990年5月20日巳时，北京，分析八字",
  "expected_route_primary": "bazi",
  "expected_task_intent_any": ["collect_profile", "interpret_chart"],
  "expected_turn_type_any": ["agent_reading"],
  "required_observations": ["preflight", "sse_emit"]
}
```

可用字段：`setup_message`、`timeout_seconds`、`response_must_contain_all`、
`response_must_not_contain`、`expected_trace_attributes`。文本断言只用于稳定协议、
展示结构或明确安全边界；路线、证据、年龄授权、guard 等语义优先断言 trace 属性。

一个 case 不要混合路由、检索、答案质量和时延主张。每个失败必须能解释为一个明确
合同缺口，禁止为某个命盘、session 或 trace 加 runtime 特判。

## Langfuse score 与成本事实

当前平台已创建两个基础 `ScoreConfig`：`eval_contract_pass` 为 BOOLEAN，
`eval_failure_class` 为 CATEGORICAL。`run_langfuse_eval.py` 在启用 `--write-scores`
时会对通过和失败都写平台 score；失败如果能定位 trace，会把 `trace_id`、`session_id`
和 `failure_class` 写入 JSON 报告。

`eval_contract_pass` 是运行时合同通过率，不是答案质量分。答案质量使用独立
`answer_*` ScoreConfig：`answer_task_complete`、`answer_factuality_pass`、
`answer_grounding_pass`、`answer_scope_safe`、`answer_failure_class`。不可从
`eval_contract_pass` 或旧的 `langfuse_smoke_pass` 推导答案质量。

记录每次在线运行的 dataset、case 数、模型/配置 revision、timeout、重复次数、报告
路径和 trace id。只有本机 trace 的 token/cost 字段被实际核验后，才把金额或 token
预算做硬门槛；当前不能以未验证的成本数据宣称节省或超支。

## Langfuse 平台内置评测

Langfuse 自带 trace/session/dataset/run/score 和 evaluator 能力，但本项目不把平台 UI
当成唯一事实来源。本地 runner 负责 CI、合同断言和可复现 JSON 报告；Langfuse 平台负责
沉淀 trace、score、人工标注和跨版本比较。

先检查本机平台状态：

```bash
python3 eval/runner/check_langfuse_setup.py
```

当前已验证：ScoreConfig 与 score API 可用；`/api/public/llm-connections` 可查询；本机尚未
配置 LLM Connection。要使用平台托管的 LLM-as-a-Judge，先在 Langfuse UI 的项目设置中
添加模型供应商连接，再创建 evaluator，并显式映射 input/output/expected output 或
trace 字段。observation evaluator 默认拿不到同 trace 的 sibling/child observation；
需要全链路事实时，评估 dataset item 或带完整摘要的逻辑 root。

平台 Judge 启用前必须先用人工样本校准。不要让未校准 evaluator 自动污染 `answer_*`
人工分；平台 Judge 分数使用 `judge_*` score，人工分继续使用 `answer_*` score。

## 答案质量与 LLM Judge

需要评估“回答是否好”时，读取 `references/evaluation-standard.md` 和
`references/langfuse-official.md`；若要做 Langfuse 平台 experiment 或 Judge 校准，
同步读取 `.agents/skills/langfuse/references/judge-calibration.md`。先跑确定性合同，
再定义质量门：任务完成、事实性、证据支撑、范围安全，以及分类失败原因。

先准备 10 条人工标注样本，不直接启用未校准 Judge。当前人工标注入口：

```bash
# 人工填写 scores 中的 null 值
eval/annotation/answer-quality-human-v1.json

# 预览会写哪些 score；默认不写平台
python3 eval/runner/write_human_answer_scores.py \
  --annotation-path eval/annotation/answer-quality-human-v1.json

# 人工确认后才写回 Langfuse answer_* score
python3 eval/runner/write_human_answer_scores.py \
  --annotation-path eval/annotation/answer-quality-human-v1.json \
  --write-scores
```

重复执行写回默认跳过同 trace 上已有的同名 score；只有确实要保留多轮标注历史时才加
`--allow-duplicates`。

人工标注不要求标注者懂八字。标注者只判断可观察质量：是否答了用户问题、是否与回答内
工具事实自相矛盾、是否引用了可见证据、是否越过产品/安全边界。不要让非专家判断
“格局、用神、层次”是否命理正确；这类领域正确性只进入确定性工具测试、权威材料合同或
专家复核样本。

人工标注只改 `scores` 和 `human_note`；不要改 trace id、session id、input 或 output。
写回脚本只处理非 null 分数，允许分批标注、分批写回。看不出证据时，优先把
`answer_grounding_pass` 标为 `false`，并把 `answer_failure_class` 标为
`insufficient_evidence`；不要硬猜八字对错。

本地 Judge 校准入口：

```bash
# 只预览 prompt，不调用模型
python3 eval/runner/run_answer_quality_judge.py --dry-run --limit 2

# 使用 OpenAI-compatible judge endpoint 跑 10 条人工样本
EVAL_JUDGE_API_KEY=... \
EVAL_JUDGE_BASE_URL=... \
EVAL_JUDGE_MODEL=... \
python3 eval/runner/run_answer_quality_judge.py \
  --report-path eval/reports/answer-quality-judge-v1.json

# 人工确认后，才把 judge_* 分数写回 Langfuse
python3 eval/runner/run_answer_quality_judge.py --write-scores
```

Judge 输入只能有用户问题、最终回答、可用 expected output 和明确允许的 trace/检索
证据。Judge 必须返回结构化质量分数和简短理由；`insufficient_evidence` 不能算 pass。
先用人工标注小样本校准，再作为质量信号，且抽查所有失败与部分通过样本。10 条样本只够
发现 rubric 或变量映射问题，不足以证明 Judge 可作为发布门禁；门禁前至少扩到稳定
故障模式覆盖集。

校准 Judge 时，`expectedOutput` 只能交给 evaluator 比对，不能进入 Judge prompt 或
task 输入；否则会泄露答案并使校准失效。二分类或多分类标签必须先定义允许值，未知标签
计为 invalid 并排除分母，不能默默当作失败或负类。

Langfuse observation evaluator 不会自动得到同 trace 的 sibling/child observation。
需要全链路事实时，评估带完整摘要的逻辑 root，或评估显式包含 input/output/expected
output 的 dataset item；启用前必须预览真实映射。

## 交付

汇报必须给出：执行命令、选它的成本理由、dataset/run/revision、报告路径、通过/失败数、
失败 case 与原因、trace id、是否写 score、成本字段是否已核验，以及仍未覆盖的风险。
只有已验证的评测节点、环境事实或阻塞问题变化时才更新 `PROGRESS.md`。
