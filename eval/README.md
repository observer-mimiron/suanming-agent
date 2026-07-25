# Eval

这个目录是项目当前唯一正式保留的评测层。

结论先说：

- 官方主回归入口是 `make regression`。
- 本地 truth layer 是 `Go 合同测试 + eval/datasets/*.json + 结构化 report + backend/.env`，不是 Langfuse 某个 UI 栏目。
- 当前 `eval/` 已经能稳定替代旧 `testsets` 的主流程，但替代的是“运行时正确性 / 最小回归门禁”，不是“完整答案质量评测平台”。
- 当前本地 Langfuse 是跑在 WSL Docker 里的 `v3` self-hosted，端口是 `3001`；`Traces / Sessions / Datasets / Dataset Runs / Scores` 可用，`Experiments / Evals` 仍不能作为本地主工作流。
- 当前最小正式数据集已补到三类入口：首轮主链、同 session follow-up、原文检索链路。

## 当前本地实测状态（2026-07-11）

下面这些结论都基于本地实测，不是理论猜测：

| 项目 | 当前状态 | 本地证据 | 工程含义 |
|------|----------|----------|----------|
| Langfuse 部署 | 已运行 | WSL `docker ps` 显示 `langfuse/langfuse:3`、`langfuse-worker:3`、Web 暴露 `3001:3000` | 当前正式实例就是 `v3`，不要按 `v4` 设计日常主流程 |
| Compose 真相源 | 已确认 | `deploy/langfuse/docker-compose.yml` 使用 `langfuse/langfuse:3` / `langfuse-worker:3`，本地凭据从忽略的 `deploy/langfuse/.env` 注入 | 仓库不提交 Langfuse secret |
| 后端 OTEL 配置 | 已接通 | `backend/.env` 指向 `http://localhost:3001/api/public/otel`，并带 `Authorization` header | 后端 trace 会镜像到本地 Langfuse |
| Traces | 已接通 | `GET /api/public/traces?limit=2` 返回 trace，且 `service.name=suanming-agent` | trace 已不是阻塞项 |
| Sessions | 已接通 | `GET /api/public/sessions?limit=2` 返回数据 | `session_id -> langfuse.session.id` 映射已生效 |
| Datasets | 已接通 | `GET /api/public/datasets` 返回 `runtime-smoke-v1`、`retrieval-benchmark-v1` | 本地 JSON 可同步为 hosted datasets |
| Dataset Runs | 已接通 | `datasets` 响应里的 `runs` 字段已有 `runtime-smoke-v1-demo` 等记录 | Python runner 已能登记 dataset run item |
| Scores | 已接通 | `GET /api/public/traces` 返回的 trace 已带 `scores` | trace-level score writeback 可用 |
| Prompts | 未接入 | 仓库内没有 Langfuse Prompt Management 接入代码；现有 runner 也不上传 prompts | `Prompts` 栏为空不代表 tracing/eval 失败 |
| Experiments API | 当前不可用 | `GET /api/public/experiments?...` 返回 “only available in a Langfuse v4 write mode” | 当前本地 v3 不能把 `Experiments / Evals` 当主入口 |
| Evals UI | 不能作为主评测面 | 本地已能写入 dataset / run / scores，但 experiments API 仍不可用 | 不要继续围绕 “为什么 Evals 页空” 兜圈子 |

## 当前能力边界

### 已接入的 Langfuse 能力

- `Traces`
  已通过 OTel mirror 写入；本地 trace 详情里能看到 `approved_route.primary_domain`、`task_intent`、`turn_type` 等属性。
- `Sessions`
  已通过 `session_id` 顶层字段和 `langfuse.session.id` 别名聚合。
- `Datasets`
  已能把 `eval/datasets/*.json` 同步成 hosted datasets。
- `Dataset Runs`
  已能把真实 `/api/chat` 运行登记为 dataset run item。
- `Scores`
  已能给 trace 写回 rule-based / contract-based 分数。

### 还没接入的能力

- `Prompts`
  仓库还没有接入 Langfuse Prompt Management，也没有 prompt 版本同步逻辑。
- `Prompt-linked eval`
  现在的评测不是以 Langfuse prompt registry 为中心。
- `完整 evaluator workflow`
  现在主要是 rule-based / contract-based smoke，不是完整 judge / rubric / annotation 平台。

### 哪些是“代码还没做”

- Prompt Management 接入
- 人工 review / annotation 队列
- 更丰富的质量 rubric
- 长期趋势统计和版本对比

### 哪些是“当前部署形态本身不支持”

- 在当前本地 `v3` self-hosted 上，把 `Experiments / Evals` 当作可靠主入口
- 通过现有 `3001` 实例，直接得到“完整 v4 eval UI”

## 本地 truth layer

这是后续 AI 和人都必须遵守的判断顺序：

1. **第一层：Go 合同测试**
   `make regression` 会跑 runtime / supervisor / policy 相关合同测试。
2. **第二层：`eval/datasets/*.json`**
   这是最小正式评测数据集，定义我们到底在验证哪些主链信号。
   当前至少覆盖首轮主链、同 session follow-up、原文检索链路。
3. **第三层：结构化 report**
   `eval/reports/*.json` 是机器可消费的结果沉淀层。
4. **第四层：`backend/.env`**
   这是唯一有效的后端 Langfuse / OTEL 配置来源；不要再写根目录 `.env`。
5. **第五层：Langfuse UI / API**
   这是观测层，不是最终真相源。尤其 `Evals` 页不应反向定义本地主流程。

一句话说清楚：
本地评测先看“回归脚本 + 本地数据集 + 报告”，再看 Langfuse 页面是否方便观察。

## `eval/` 与旧 `testsets` 的关系

- 旧 `testsets/` 已退出正式主流程，不再是官方入口。
- 当前仓库默认应继续扩 `eval/`，而不是恢复旧 `testsets` 工作流。
- 旧 `testsets` 只可作为历史参考，不应再承担：
  - 官方回归入口
  - 当前 truth layer
  - 新增 case 的主要落点

清理规则：

- 只有当某部分旧链路已经被 `eval/` 明确替代，才允许删。
- 当前不要为了“把仓库看起来更干净”去恢复、重写或重新挂载 `testsets`。
- 如果工作区里已经存在 `testsets` 删除记录，先按“已退役资产”理解，不要把它当成待修复主链。

## 目录职责

```text
eval/
  README.md                       # 本文档：唯一入口说明
  datasets/                       # 本地评测数据集
    runtime-smoke-v1.json         # 官方最小 smoke
    retrieval-benchmark-v1.json   # 检索链路基准
  runner/
    run-agent-regression.sh       # 官方回归的底层编排器
    run-langfuse-eval.sh          # 跑单个本地数据集
    run-langfuse-eval-suite.sh    # 跑整个本地 suite
    build-cheap-gate-report.sh    # 汇总 cheap gate 本地样本
    run_langfuse_eval.py          # /api/chat + Langfuse trace 断言
    langfuse_eval_common.py       # Langfuse / backend API 公共逻辑
    sync-langfuse-datasets.py     # 本地 JSON -> Langfuse hosted dataset
    run-langfuse-experiment.py    # 真实请求 -> trace 校验 -> dataset run item
  reports/                        # 结构化结果输出目录
    cheap-gate-summary.json       # 本地 cheap gate 命中样本汇总
```

配套但不在本目录内的关键文件：

- `make regression`
  官方主回归入口，会在本地 `:8080` 不可用时自启动后端，再执行 Go 合同测试和最小 smoke。
- `make eval-smoke`
  单数据集 smoke 入口，负责发请求、轮询 Langfuse、断言 trace / observation / route / task / turn。
- `deploy/langfuse/docker-compose.yml`
  当前正式 Langfuse 本地部署文件，跑的是 `v3` self-hosted。
- `backend/.env`
  当前唯一有效的后端 Langfuse / OTEL 配置文件。
- `deploy/langfuse/.env`
  Langfuse Docker 的本地凭据文件；从 `.env.example` 创建，不能提交真实 project secret 或生产密码。

## 什么时候用 Bash，什么时候用 Python

### 用 Bash 的场景

- `make regression`
  用于官方主回归门禁。它会负责：
  - 启动或复用本地后端
  - 跑 Go 合同测试
  - 跑 `runtime-smoke-v1.json`
- `make eval-smoke`
  用于单个本地数据集 smoke。
- `make eval-suite`
  用于多数据集 suite。

Bash 层的定位是：
让这套 WSL 环境下的“本地最小回归”稳定、低依赖、立刻可跑。

### 用 Python 的场景

- `eval/runner/sync-langfuse-datasets.py`
  用于把本地 JSON 同步成 Langfuse hosted datasets。
- `eval/runner/run-langfuse-experiment.py`
  用于把真实 `/api/chat` 请求、trace 校验、score writeback 和 dataset run item 登记串起来。

Python 层的定位是：
承接 Langfuse hosted dataset / run 这条更易演进的工程化路径。

### 不要混淆的边界

- `make` 是人用入口。
- `sh` 是 Makefile 背后的本地编排层。
- `py` 是 hosted dataset / run 与 trace 断言实现层。
- 现在没有任何一个本地命令可以把 `Evals` UI 直接变成可靠主入口。

## 入口命令

### 1. 官方主回归

```bash
make regression
```

适用：

- 改完代码后先看主链有没有炸
- 需要官方门禁结论时

当前会做三件事：

1. 跑 runtime 合同测试
2. 跑 supervisor / policy 测试
3. 跑 `eval/datasets/runtime-smoke-v1.json`

### 2. 跑单个本地数据集

```bash
make eval-smoke
```

适用：

- 新增 case 后先本地验证
- 调试某个数据集
- 不想跑完整官方回归

### 3. 跑整个本地 suite

```bash
make eval-suite
```

适用：

- 一次性看 smoke + retrieval 两类基准
- 需要结构化 suite 报告
- 需要顺手把 score 写回 Langfuse

### 4. 同步 hosted datasets

```bash
python3 eval/runner/sync-langfuse-datasets.py \
  --langfuse-url http://localhost:3001
```

适用：

- 让 Langfuse `Datasets` 栏出现本地数据集
- 校准 hosted dataset 和本地 JSON 一致

### 5. 登记一次 dataset run

```bash
python3 eval/runner/run-langfuse-experiment.py \
  --dataset-path eval/datasets/runtime-smoke-v1.json \
  --server-url http://localhost:8080 \
  --langfuse-url http://localhost:3001 \
  --run-name runtime-smoke-v1-demo \
  --write-scores \
  --report-path eval/reports/runtime-smoke-v1-experiment.json
```

适用：

- 让 Langfuse dataset run 中出现一次真实运行
- 把 trace、dataset item、dataset run item 关联起来

注意：
这个脚本当前登记的是 **dataset run item**，不是调用本地可用的 `Experiments` API。

### 6. 汇总 cheap gate 本地样本

```bash
make cheap-gate-report
```

适用：

- 想看 cheap gate 到底命中了多少次
- 想判断是否值得继续安全扩面
- 想把 `logs/reports/cheap-gate/hits.jsonl` 变成可读的结构化报告

默认输出：

- 输入：`logs/reports/cheap-gate/hits.jsonl`
- 输出：`eval/reports/cheap-gate-summary.json`

当前报告包含：

- 总命中数
- 按 `primary_domain` / `task_intent` / `gate_reason` / `execution_mode` / `decision_source` 的分布
- 前 N 条样本预览

## Langfuse UI 栏目怎么理解

| 栏目 | 理论上对应什么 | 当前本地状态 |
|------|----------------|--------------|
| `Traces` | OTel 写入的 trace / observation | 已可用 |
| `Sessions` | 按原生 session 字段聚合的 trace | 已可用 |
| `Prompts` | Langfuse Prompt Management 中登记的 prompt | 未接入 |
| `Datasets` | hosted dataset 元数据与 items | 已可用 |
| `Dataset Runs` | dataset item 对应的 run 记录 | 已可用 |
| `Runs / Experiments / Evals` | 更完整的 experiment / evaluator workflow | 当前本地 v3 不能作为可靠主流程 |

最容易误判的地方：

- `Traces` 有数据，不等于 `Datasets` 一定有。
- `Datasets` 和 `Dataset Runs` 有数据，不等于 `Evals` 一定有。
- `Evals` 为空，不等于我们没有把数据写进 Langfuse。

## 数据集格式

当前每个数据集是一个 JSON 文件，基本结构如下：

```json
{
  "name": "runtime-smoke-v1",
  "description": "最小 smoke",
  "cases": [
    {
      "id": "runtime-smoke-bazi-main",
      "message": "用户问题",
      "expected_route_primary": "bazi",
      "expected_task_intent_any": ["collect_profile", "interpret_chart"],
      "expected_turn_type_any": ["agent_reading"],
      "required_observations": ["preflight", "sse_emit"]
    }
  ]
}
```

字段含义：

- `name`
  数据集名，同时用于 report 与 Langfuse dataset 名称。
- `description`
  数据集用途说明。
- `cases`
  case 列表。
- `id`
  case 稳定标识，用于 report、dataset item 和排错。
- `message`
  发送给 `/api/chat` 的真实输入。
- `setup_message`
  可选；会先用同一 session 发送铺垫消息，再发正式问题。
- `expected_route_primary`
  期望主领域，例如 `bazi` / `qimen` / `ziwei`。
- `expected_task_intent_any`
  允许的 task intent 集合。
- `expected_turn_type_any`
  允许的 turn type 集合。
- `required_observations`
  trace 中必须出现的 observation 名称。

## 当前已有数据集

### `runtime-smoke-v1.json`

定位：
官方最小 smoke，负责验证“主链没炸”。

主要验证：

- `/api/chat` 主链可达
- SSE 有 `event: done`
- trace 能进 Langfuse
- route / task / turn 落在合理范围
- 关键 observation 至少存在

### `retrieval-benchmark-v1.json`

定位：
检索链路可观测性基准。

主要验证：

- 带知识检索的一轮真实请求能跑通
- trace 中能看到 `knowledge_search`
- authority-first bazi 路径下检索链路没有静默丢失

## 当前评分模型

V1 评分以 rule-based / contract-based 为主，不做复杂主观判分。

当前 score 规范见：

- `docs/agent-engineering/langfuse-score-schema-v1.md`

当前核心 score 类型包括：

- 通用 smoke
  `langfuse_smoke_pass`、`langfuse_trace_present`、`langfuse_required_observations`、`sse_done`
- route / intent
  `route_primary_match`、`task_intent_match`、`turn_type_match`
- observation
  `observation_<name>_present`

这套分数的目标不是“评价答案好坏”，而是先判断“链路有没有按设计工作”。

## 已知实现约束

### 1. 当前 eval 更偏“运行时正确性”

它更擅长回答：

- trace 有没有落
- observation 有没有丢
- route 有没有明显漂
- done 事件有没有收口

它还不擅长回答：

- 最终中文回答质量是否稳定优秀
- 用户视角是否真正满意
- 不同 prompt / model / router 版本谁更好

### 2. `session_id` 必须每次唯一

当前 runner 必须为每个 case 生成唯一 session。

原因：

- 如果复用固定 `session_id`
- 轮询 Langfuse 时可能命中旧 trace
- 会造成假失败或误判

当前实现已经按 `case id + GUID` 自动生成唯一 session；后续重写 runner 时必须保留这个约束。

### 3. 断言要“窄而稳”

V1 优先断言这些稳定信号：

- route
- task intent
- turn type
- required observations
- response 是否正常收口

不要把“回答必须包含某一句固定中文”当主断言，否则会非常脆弱。

### 4. 当前 `3001` 的 v3 实例不要被破坏

- 这是当前正式可用实例
- 所有后续探索都应避免覆盖它
- 如果未来要做并行新实例，必须换端口、换 volume、换 compose 文件

## 为什么现在不加 `docker-compose.v4-preview.yml`

当前结论是：**先不加。**

原因不是保守，而是避免继续围绕一个本地尚未站稳的路径消耗时间：

1. 本地实测 `GET /api/public/experiments` 仍明确返回“only available in a Langfuse v4 write mode”。
2. 当前仓库正式 compose 与实际运行实例都还是 `langfuse:3`。
3. 官方公开文档当前把 v4 描述为新的 write mode / Fast Preview 路线，而不是“本地这套 v3 self-hosted 直接加个开关就能完整获得 Evals”。
4. 在没有稳定官方 self-hosted 升级路径前，仓库里新增一个“看起来先进、实际不可持续”的 preview compose，会误导后续 AI 和人。

所以现在最合理的工程策略是：

- 保留现有 `3001` v3 实例继续承担日常 trace / session / dataset / dataset-run 工作流
- 把 `eval/` 主流程做稳
- 等官方 self-hosted v4 路线明确稳定后，再单独加平行实例

未来如果满足下面两个条件，再考虑新增并行 compose：

1. 官方 self-hosted 文档明确给出可持续的 v4 write mode 部署路径
2. 本地验证确认 `Experiments / Evals` API 与 UI 都可稳定使用

届时建议新增而不是覆盖：

- 新文件：`deploy/langfuse/docker-compose.v4-preview.yml`
- 新端口：例如 `3002`
- 新 volumes：完全隔离，不复用 `3001` 的数据

## 下一阶段推荐路线

### 阶段 A：把当前 v3 工作流做成稳定主评测体系

先完成这些：

- 扩充 3 到 8 个最小稳定 case
- 补更清晰的失败原因字段
- 让 report 更容易被 AI 二次消费
- 继续把 `eval/` 和官方回归入口绑定紧

### 阶段 B：把“运行时正确性”扩成“初级质量评测”

再做这些：

- 引入 answer rubric
- 引入人工 review / annotation 队列
- 区分链路失败与答案质量退化
- 记录趋势，而不是只看单次 pass/fail

### 阶段 C：等官方 self-hosted 路线稳定后，再补真正的 Evals 平台

到那时再考虑：

- experiment 对比
- prompt / model / router 版本回归对比
- 线上样本回放
- 更长期质量看板

## 给后续 AI 的工作协议

1. 先判断你是在扩“运行时正确性”还是“答案质量评估”。
2. 如果只是验证主链没坏，优先改现有 `runtime-smoke-v1` 或新增同类型小数据集。
3. 如果是验证检索链路，优先扩 `retrieval-benchmark-v1`。
4. 新增 case 时，优先加稳定元信息断言，不要先加脆弱文本断言。
5. 如果要让 Langfuse `Datasets` 里可见，先跑 `sync-langfuse-datasets.py`。
6. 如果要让 Langfuse `Dataset Runs` 里可见，使用 `run-langfuse-experiment.py`。
7. 改完后先跑单个数据集，再跑 `make regression`。
8. 只有当 `eval/` 已经覆盖原目标时，才允许删除明确过时的旧脚本或旧数据。

AI 新增 case 时建议模板：

```json
{
  "id": "your-case-id",
  "message": "真实用户问题",
  "expected_route_primary": "bazi",
  "expected_task_intent_any": ["interpret_chart"],
  "expected_turn_type_any": ["agent_reading"],
  "required_observations": ["preflight", "sse_emit", "knowledge_search"]
}
```
