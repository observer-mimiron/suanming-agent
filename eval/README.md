# Eval

> 本目录把确定性合同、当前回答 Judge 和人工抽查分开。Judge 只评用户可见质量，不裁断八字专业结论。

## 真相顺序

按以下顺序判断一次改动是否通过：

1. Go 合同测试和构建。
2. `eval/datasets/*.json` 定义的真实请求与稳定断言。
3. `eval/reports/*.json` 的结构化结果。
4. `backend/.env` 中实际启用的观测配置。
5. Langfuse trace、session、dataset run、score 作为观测证据。

Langfuse 页面不是唯一验收信号；UI 没有显示正文，不等于运行时合同失败。

## 当前范围

| 能力 | 状态 | 说明 |
|---|---|---|
| Go 合同测试 | 已接入 | 覆盖路由、runtime、资产和大运确定性合同 |
| 本地 smoke | 已接入 | 真实 `/api/chat`、SSE done、route/task/turn、正文结构与 observation 断言；默认使用 `runtime-smoke-v2` |
| Langfuse trace/session/dataset/dataset run/score | 已验证 | 本地 WSL Docker 的 self-hosted v3 |
| 双回合 follow-up 在线 smoke | 待最新报告确认 | 评测器已修复总预算和 setup trace 排除，需重新跑完整在线样本 |
| 八字质量合同评测 | 已接入 | `bazi-quality-v2` 通过真实 `/api/chat`、SSE、Langfuse trace 与结构化响应/属性断言；不等于命理专业正确 |
| Repair Harness 回归 | 已接入 | `runtime-repair-v1` 固化最近静态调候投影失败样本；`make eval-repair` 先跑本地 repair 合同测试再跑真实 `/api/chat` |
| LLM Judge / 文本质量评测 | 已接入但非默认门禁 | 历史人工样本用于校准；`make eval-bazi-review` 对本次 `bazi-quality-v2` 回放逐条评估并把 `judge_*` 分数写回当前 trace |
| Experiments / Evals UI | 非主流程 | 当前 v3 部署不能作为稳定入口 |

## 常用命令

```bash
# 默认回归：Go 合同测试 + runtime smoke（线上仅 smoke）
make regression

# 单个 smoke 数据集
make eval-smoke

# 全部本地数据集
# 仅在明确需要全量回归时执行
make eval-suite

# 八字质量合同（真实请求，通常约数分钟）
make eval-bazi-quality

# 当前八字回答 Judge（先回放，再评本次报告；显式在线调用）
make eval-bazi-review

# 八字稳定性合同（同一输入重复 10 次）
make eval-bazi-stability

# Repair Harness 合同（本地 repair 单测 + 真实请求）
make eval-repair

# cheap gate 样本聚合
make cheap-gate-report

# 评测器单元测试
python3 -m unittest eval/runner/test_run_langfuse_eval.py -v
```

本地服务状态：

```bash
make status
make restart-core
```

完整开发栈用 `make dev`，不启动 Langfuse 的核心三服务用 `make dev-core`。

## 目录与报告

```text
eval/
  datasets/                         # 可执行案例合同
    runtime-smoke-v1.json           # 历史首轮、follow-up、资产隔离 smoke
    runtime-smoke-v2.json           # 当前 smoke：含正文和 final audit 合同
    retrieval-benchmark-v1.json     # 检索链路基准
    bazi-quality-v1.json            # 历史八字候选裁定、年龄边界与运行时合同质量
    bazi-quality-v2.json            # 当前八字候选裁定、年龄边界与 final audit 合同
    bazi-stability-v1.json          # 同一八字输入重复运行稳定性合同
    runtime-repair-v1.json          # Repair Harness 最近失败样本回归
  runner/
    run-agent-regression.sh         # make regression 底层编排
    run_langfuse_eval.py            # 真实请求、trace 轮询和断言
    langfuse_eval_common.py         # API / trace 公共逻辑
    test_run_langfuse_eval.py       # 评测器单元测试
    sync-langfuse-datasets.py       # 本地 JSON 同步为 hosted dataset
    run-langfuse-experiment.py      # 登记 dataset run 与 score
  reports/                          # 机器可读结果
```

`make regression` 运行本地 Go 合同测试和一次当前构建的 `runtime-smoke-v2`，默认把服务启动在独立的 `127.0.0.1:18080`，报告写入 `/tmp/suanming-agent/runtime-smoke-report.json`；只有设置 `AGENT_REGRESSION_SERVER` 才复用外部服务。在线评测不运行全量数据集。每份报告带 `git_revision`、`dataset_version`、`generated_at`、服务地址、通过/失败数、失败分类和 trace id。suite、hosted dataset run 和 cheap gate 报告写入 `eval/reports/`。

评测分三层：L1 是每次变更都跑的 Go/结构合同；L2 是显式执行的当前在线回放和独立 Judge；L3 是重大 prompt、renderer 或领域规则变更后的人工抽查。L2 Judge 失败是质量告警，不替代 L1 合同，也不证明命理专业结论。

## 数据集合同

最小 case 格式：

```json
{
  "id": "runtime-smoke-bazi-main",
  "message": "1991年10月5日中午12点出生，男，看看八字",
  "expected_route_primary": "bazi",
  "expected_task_intent_any": ["collect_profile", "interpret_chart"],
  "expected_turn_type_any": ["agent_reading"],
  "required_observations": ["preflight", "sse_emit"]
}
```

可选 `setup_message` 用同一 session 先建立上下文；需要把 setup 与正式追问分开核验。稳定断言应优先使用 route、task intent、turn type、SSE done、artifact 或 observation，不能把某句中文文案作为主断言。

当前数据集覆盖：

- 首轮八字主链和完成事件。
- 同 session follow-up。
- 检索证据链路。
- Repair Harness 最近静态调候投影失败样本。
- 多对象、出生资料修订、解读来源绑定与奇门 Case 隔离的 Go 合同回归。

## Hosted dataset 与 Langfuse

仅在需要把本地案例同步到 Langfuse 时执行：

```bash
python3 eval/runner/sync-langfuse-datasets.py \
  --langfuse-url http://localhost:3001

python3 eval/runner/run-langfuse-experiment.py \
  --dataset-path eval/datasets/runtime-smoke-v2.json \
  --server-url http://localhost:8080 \
  --langfuse-url http://localhost:3001 \
  --run-name runtime-smoke-v2-manual \
  --write-scores \
  --report-path eval/reports/runtime-smoke-v2-experiment.json
```

第二个脚本登记的是 dataset run item 和 score，不依赖当前不可作为主流程的 Experiments API。

## 已知限制

- 一次双回合 case 使用总时间预算；不要把两个真实请求误当成单轮 120 秒预算。
- follow-up 轮询 trace 时必须排除 `setup_message` 产生的 trace，否则会把首轮 `collect_profile` 误判为追问结果。
- 每个 case 使用唯一 `session_id`，避免旧 trace 污染断言。
- 当前最强的是运行时正确性，不是答案质量。新增质量评测前需定义 rubric、人工复核或稳定 Judge 合同。
- 不要恢复旧 `testsets` 作为正式入口；`eval/skills` 只作为评测操作说明保留，不能替代 runner、dataset 和 report。新增案例放在 `eval/datasets/`。

## 后续原则

- 新 case 先证明一个稳定合同，再扩大数据集；失败报告必须保留原因而不是只写 pass/fail。
- 扩 cheap gate 前先看 `eval/reports/cheap-gate-summary.json` 的样本分布。
- 完整双回合在线 smoke 出新报告前，文档只能称“评测器修复已测试”，不能称在线回归已通过。
