# Langfuse 官方依据与本机边界

做 Langfuse 设计或查新行为时读取本文件。先核实本机 self-hosted 部署能力，再引用新版
官方 UI/API；不要把文档能力直接当成本机已启用能力。

| 官方资料 | 可复用结论 | 本项目落地 |
|---|---|---|
| [Evaluation overview](https://langfuse.com/docs/evaluation/overview) | 评测同时包含线上 trace 打分和离线 dataset 比较。 | 本地 JSON 报告是合同，Langfuse 用于 trace/run 证据。 |
| [Core concepts](https://langfuse.com/docs/evaluation/core-concepts) | dataset item、task、evaluation method 共同产生带 score 的 experiment run。 | 本机 v3 用 dataset-run item，不以 Experiments 为主路径。 |
| [Datasets](https://langfuse.com/docs/evaluation/experiments/datasets) | 版本化 dataset 使比较可复现。 | 版本化仓库 JSON，比较时保持 case 不变。 |
| [Scores](https://langfuse.com/docs/evaluation/scores/overview) | score 可为 numeric、categorical、boolean、text，附着于 trace/observation/session/dataset run。 | 硬合同用 boolean，失败原因用 categorical。 |
| [Score data model](https://langfuse.com/docs/evaluation/scores/data-model) | ScoreConfig 固化类型和值域约束。 | runner 未绑定 config 前，不宣称 score 可规范横比。 |
| [LLM-as-a-Judge](https://langfuse.com/docs/evaluation/evaluation-methods/llm-as-a-judge) | evaluator 必须显式变量映射；observation 不自动带 sibling/child 数据。 | 评估逻辑 root 或显式 dataset item，先预览映射。 |
| [Human Annotation](https://langfuse.com/docs/evaluation/human-annotation) | 人工可在 Langfuse UI 或 API 中给 trace、observation、session 等对象打 score。 | 本项目先用 JSON 小样本人工标注，再通过 `write_human_answer_scores.py` 写 `answer_*` score。 |

## 已验证能力与限制

项目已验证 trace、session、hosted dataset、dataset-run item、ScoreConfig 与 score；
本机 Langfuse v3 不把 Experiments/Evals UI/API 作为必需路径。遇到 experiment
endpoint 失败，先执行 `make status`、核实版本和 API 返回，再走已支持的 dataset-run
路径，不能盲改评测代码。

当前基础 ScoreConfig：

| name | type | 用途 |
|---|---|---|
| `eval_contract_pass` | BOOLEAN | 本地 eval case 的运行时合同是否通过 |
| `eval_failure_class` | CATEGORICAL | 失败首因：`transport`、`timeout`、`sse`、`response_contract`、`route`、`task_intent`、`turn_type`、`observation`、`trace_attribute`、`unknown` |

当前答案质量 ScoreConfig：

| name | type | 用途 |
|---|---|---|
| `answer_task_complete` | BOOLEAN | 回答是否完成用户任务 |
| `answer_factuality_pass` | BOOLEAN | 回答是否无明显事实矛盾或编造 |
| `answer_grounding_pass` | BOOLEAN | 关键主张是否有允许证据支撑 |
| `answer_scope_safe` | BOOLEAN | 回答是否遵守产品和安全边界 |
| `answer_failure_class` | CATEGORICAL | 失败首因：`none`、`task`、`factuality`、`grounding`、`scope`、`style`、`insufficient_evidence` |

本机 v3 写入 BOOLEAN score 时，`POST /api/public/scores` 的 `value` 必须传 `1` 或 `0`；
读取 `/api/public/v3/scores` 时会显示为 `true` / `false`。标注 JSON 仍使用
`true` / `false`，由 `write_human_answer_scores.py` 负责转换，不要让人工文件改成数字。

本机 cost/token 观测必须先实际核验才可成为门禁。未核验时，只记录在线 case 数、模型/配置
revision、timeout、重复次数和报告路径，不能根据 UI 空字段推导零成本或无 trace。

## 资料检索

当前技术资料优先使用 `agent-reach` 的 Exa，并合并 stderr，避免传输层异常被误判为“没有
搜索结果”：

```bash
mcporter call 'exa.web_search_exa(query: "site:langfuse.com/docs evaluation <topic>", numResults: 5)' 2>&1
```

Exa snippet 只用于定位；改变本地合同前必须阅读其对应官方页。Jina Reader 作为有限时的
备用路径，本环境曾出现直接请求超时。
