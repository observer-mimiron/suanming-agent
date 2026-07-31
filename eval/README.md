# Eval

> 本目录验证运行时合同和最小回归，不宣称已经完成命理文本质量评测。

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
| 本地 smoke | 已接入 | 真实 `/api/chat`、SSE done、route/task/turn 与 observation 断言 |
| Langfuse trace/session/dataset/dataset run/score | 已验证 | 本地 WSL Docker 的 self-hosted v3 |
| 双回合 follow-up 在线 smoke | 待最新报告确认 | 评测器已修复总预算和 setup trace 排除，需重新跑完整在线样本 |
| LLM Judge / 文本质量评测 | 未接入 | 当前不能证明回答质量或用户满意度 |
| Experiments / Evals UI | 非主流程 | 当前 v3 部署不能作为稳定入口 |

## 常用命令

```bash
# 官方最小回归：Go 合同测试 + runtime smoke
make regression

# 单个 smoke 数据集
make eval-smoke

# 全部本地数据集
make eval-suite

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
    runtime-smoke-v1.json           # 首轮、follow-up、资产隔离 smoke
    retrieval-benchmark-v1.json     # 检索链路基准
  runner/
    run-agent-regression.sh         # make regression 底层编排
    run_langfuse_eval.py            # 真实请求、trace 轮询和断言
    langfuse_eval_common.py         # API / trace 公共逻辑
    test_run_langfuse_eval.py       # 评测器单元测试
    sync-langfuse-datasets.py       # 本地 JSON 同步为 hosted dataset
    run-langfuse-experiment.py      # 登记 dataset run 与 score
  reports/                          # 机器可读结果
```

`make regression` 的默认 runtime smoke 报告写入 `/tmp/suanming-agent/runtime-smoke-report.json`。suite、hosted dataset run 和 cheap gate 报告写入 `eval/reports/`。

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
- 多对象、出生资料修订、解读来源绑定与奇门 Case 隔离的 Go 合同回归。

## Hosted dataset 与 Langfuse

仅在需要把本地案例同步到 Langfuse 时执行：

```bash
python3 eval/runner/sync-langfuse-datasets.py \
  --langfuse-url http://localhost:3001

python3 eval/runner/run-langfuse-experiment.py \
  --dataset-path eval/datasets/runtime-smoke-v1.json \
  --server-url http://localhost:8080 \
  --langfuse-url http://localhost:3001 \
  --run-name runtime-smoke-v1-manual \
  --write-scores \
  --report-path eval/reports/runtime-smoke-v1-experiment.json
```

第二个脚本登记的是 dataset run item 和 score，不依赖当前不可作为主流程的 Experiments API。

## 已知限制

- 一次双回合 case 使用总时间预算；不要把两个真实请求误当成单轮 120 秒预算。
- follow-up 轮询 trace 时必须排除 `setup_message` 产生的 trace，否则会把首轮 `collect_profile` 误判为追问结果。
- 每个 case 使用唯一 `session_id`，避免旧 trace 污染断言。
- 当前最强的是运行时正确性，不是答案质量。新增质量评测前需定义 rubric、人工复核或稳定 Judge 合同。
- 不要恢复旧 `testsets` 作为正式入口；新增案例放在 `eval/datasets/`。

## 后续原则

- 新 case 先证明一个稳定合同，再扩大数据集；失败报告必须保留原因而不是只写 pass/fail。
- 扩 cheap gate 前先看 `eval/reports/cheap-gate-summary.json` 的样本分布。
- 完整双回合在线 smoke 出新报告前，文档只能称“评测器修复已测试”，不能称在线回归已通过。
