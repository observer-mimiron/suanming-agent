# 标准化 Agent 评测

当任务涉及答案质量、检索质量、模型/prompt 比较或 LLM Judge 时读取本文件。评测先分层，
不能用一个总分掩盖硬合同失败。

| 层级 | 要回答的问题 | 主要证据 | 默认成本 |
|---|---|---|---|
| L1 确定性合同 | 系统是否遵守 API、状态、路由、权限和格式合同 | Go 测试、SSE、trace 属性、observation | 每次相关改动 |
| L2 检索与质量 Judge | 是否检到并使用正确证据，回答是否符合特定质量要求 | 版本化 dataset、retrieval source、结构化 Judge | 受影响模型/检索改动 |
| L3 人工复核 | 高风险、重大变更或 Judge 不确定时是否可接受 | trace、回答、证据、人工标签 | 发布前或抽样 |

## Dataset 设计

- 版本化 dataset，例如 `runtime-smoke-v1`；比较时不得静默替换 case。
- 覆盖正常、缺资料/歧义、follow-up 状态、需检索问题、降级/guard 与真实历史坏例。
- `setup_message` 与被评测 turn 必须分开；正式 turn 查 trace 时排除 setup trace。
- 运行时合同优先存结构化 expected outcome 与证据要求，参考答案不能天然成为精确文本合同。
- 单例用户问题只能作为 regression fixture，不能转化为业务专项分支或无限禁词表。

## 质量分数合同

不从未校准的 0-10“总质量分”开始。先定义独立硬门：

| score 名 | 类型 | pass 含义 |
|---|---|---|
| `answer_task_complete` | BOOLEAN | 完成支持范围内的用户任务 |
| `answer_factuality_pass` | BOOLEAN | 不与事实/结构化输出矛盾，不编造 |
| `answer_grounding_pass` | BOOLEAN | 关键主张有允许的事实或检索证据 |
| `answer_scope_safe` | BOOLEAN | 遵守产品边界和 guard |
| `answer_failure_class` | CATEGORICAL | `none`、`task`、`factuality`、`grounding`、`scope`、`style`、`insufficient_evidence` |

运行前固定 score 名、类型、允许分类、作用层级（trace 或 dataset run）、dataset 版本、
判定阈值和配置 revision。要做 Langfuse 比较，使用版本化 `ScoreConfig` 验证类型和值域。
后来新增 numeric 偏好分也不能覆盖任一硬门失败。

## 人工标注小样本

先用 10 条人工样本校准，再启用 LLM Judge。标注文件放在
`eval/annotation/answer-quality-human-v1.json`，每条 case 保留 trace、输入和完整输出；
人工只填写 `scores` 和 `human_note`：

```json
{
  "trace_id": "...",
  "session_id": "...",
  "input": "...",
  "output_excerpt": "...",
  "output": "...",
  "scores": {
    "answer_task_complete": null,
    "answer_factuality_pass": null,
    "answer_grounding_pass": null,
    "answer_scope_safe": null,
    "answer_failure_class": null
  },
  "human_note": ""
}
```

非专家标注规则：

- `answer_task_complete`：是否完成用户要求范围内的任务；例如用户给全出生信息却继续要求补资料，就是失败。
- `answer_factuality_pass`：只判断可观察事实，不判断高阶命理裁断；例如四柱、性别、地点、年龄边界、前后说法互相冲突才标失败。
- `answer_grounding_pass`：关键主张是否有允许的 trace、工具、检索或确定性事实支撑；看不到证据就标失败。
- `answer_scope_safe`：是否守住产品、安全和年龄边界；例如对未成年人做财运婚恋具体应事，就是失败。
- `answer_failure_class`：全通过填 `none`；否则填最主要失败类。

不要让普通人工标注“格局、用神、层次到底算得对不对”。这类领域正确性分三路处理：

- 可复算事实：放进 Go 单测、dataset trace 属性或确定性工具断言。
- 典籍/证据支撑：用检索 evidence、rule material 和 grounding score 检查。
- 高阶裁断是否专业：单独抽专家复核样本，不混进普通 10 条非专家标注。

写回 Langfuse 前先 dry-run：

```bash
python3 eval/runner/write_human_answer_scores.py \
  --annotation-path eval/annotation/answer-quality-human-v1.json
```

人工确认后再写：

```bash
python3 eval/runner/write_human_answer_scores.py \
  --annotation-path eval/annotation/answer-quality-human-v1.json \
  --write-scores
```

写回脚本会校验 `answer_*` ScoreConfig 类型，只写非 null 项；允许只完成部分 case 后先
写回已标注分数。若 Judge 与人工分歧，优先修改 rubric 或变量映射，不直接改 prompt。

## Judge 合同

Judge 只接收：用户问题、最终回答、可选 expected output、明确许可的 trace/retrieval
evidence 和 rubric。不得给全量历史或要求其猜测缺失真值。输出采用：

```json
{
  "decision": "pass | fail | insufficient_evidence",
  "failure_classes": ["none | task | factuality | grounding | scope | style"],
  "evidence": ["trace attribute、observation 或 source id"],
  "reason": "不超过 200 字"
}
```

`decision` 映射 BOOLEAN，`failure_classes` 映射 CATEGORICAL；
`insufficient_evidence` 暂不计通过或失败，交人工处理。先对固定人工标注样本校准 Judge，
再看全量失败、边界通过和 Judge/人工分歧样本。

## 线上与离线闭环

1. 在线 trace 发现真实坏例，记录 session、trace、上下文和失败分类。
2. 去隐私和偶然数据后，沉淀为版本化离线 dataset。
3. 每次改动先跑 L1，再只跑受影响 L2 dataset。
4. 需要比较时，用同 dataset、模型/配置、timeout、重复次数和 score schema 创建 dataset run。
5. 分数下降时，按输入/session → route → tool/retrieval → synthesis/guard → 最终输出排查。
6. 不能解释的结果标记为模型波动或证据不足，不直接改 prompt；保留报告和 trace。

## Langfuse 映射规则

Langfuse Judge 要显式映射变量。observation-level evaluator 不会自动获取 sibling/child
observation；跨步骤评测必须选含完整摘要的逻辑 root，或用带 input、output、expected
output 的实验/数据集 item。启用前用真实历史记录预览变量映射，确认 Judge 看到的是正确
turn 和正确证据。

## 最小报告

```json
{
  "dataset": "runtime-smoke-v1",
  "run": "runtime-smoke-v1-<revision>",
  "score_schema_version": "contract-v1",
  "model_config_revision": "...",
  "online_cases": 2,
  "repeats": 1,
  "passed": 2,
  "failed": 0,
  "results": [{"case_id": "...", "trace_id": "...", "scores": {"sse_done": 1}}]
}
```

报告不能只给平均分；必须包括分母、跳过/证据不足项、失败原因和成本字段是否已核验。

Script completed
Wall time 0.3 seconds
Output:
### sed -n '1,220p' eval/runner/test_run_langfuse_eval.py
import pathlib
import json
import signal
import sys
import unittest
from types import SimpleNamespace
from unittest import mock

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import run_langfuse_eval as runner
import langfuse_eval_common as common
import write_human_answer_scores as human_scores


class EvalTimeoutTest(unittest.TestCase):
    class FakeSSEResponse:
        def __init__(self):
            self.status = 200
            self.lines = iter([b"event: text\n", b"data: hi\n", b"\n", b"event: done\n", b"data: {}\n", b"\n"])

        def __enter__(self):
            return self

        def __exit__(self, exc_type, exc, tb):
            return False

        def readline(self):
            return next(self.lines, b"")

    def test_runtime_smoke_followup_accepts_both_valid_execution_turn_types(self):
        dataset_path = pathlib.Path(__file__).resolve().parents[1] / "datasets" / "runtime-smoke-v1.json"
        dataset = json.loads(dataset_path.read_text(encoding="utf-8"))
        followup = next(case for case in dataset["cases"] if case["id"] == "runtime-smoke-bazi-followup-reuse")

        self.assertEqual(followup["expected_task_intent_any"], ["fortune_followup", "interpret_chart"])
        self.assertEqual(followup["expected_turn_type_any"], ["agent_reading", "fortune_followup"])

    def test_remaining_timeout_seconds_rejects_expired_deadline(self):
        with mock.patch.object(runner.time, "monotonic", return_value=10.0):
            with self.assertRaises(TimeoutError):
                runner.remaining_timeout_seconds(10.0)

    def test_case_timeout_seconds_accepts_explicit_multiturn_budget(self):
        self.assertEqual(runner.case_timeout_seconds({"id": "followup", "timeout_seconds": 240}, 120), 240)
        with self.assertRaises(ValueError):
            runner.case_timeout_seconds({"id": "bad", "timeout_seconds": 0}, 120)

    def test_unique_case_session_id_respects_backend_length_contract(self):
        with mock.patch.object(runner.uuid, "uuid4", return_value=SimpleNamespace(hex="a" * 32)):
            session_id = runner.unique_case_session_id("eval-bazi-quality-2025-11-10-shanghai")
        self.assertLessEqual(len(session_id), 64)
        self.assertRegex(session_id, r"^[A-Za-z0-9_-]+$")

    def test_trace_lookup_excludes_setup_turn_for_followup(self):
        payload = {
            "data": [
                {"id": "followup", "sessionId": "shared"},
                {"id": "setup", "sessionId": "shared"},
            ]
        }
        with mock.patch.object(common, "api_request", return_value=payload):
            trace = common.list_traces_by_session("http://langfuse.test", {}, "shared", excluded_trace_ids={"setup"})
        self.assertEqual(trace["id"], "followup")

    def test_smoke_case_uses_one_total_deadline_for_chat_and_trace(self):
        trace = {
            "metadata": {
                "resourceAttributes": {"service.name": "suanming-agent"},
                "attributes": {"approved_route.primary_domain": "bazi", "task_intent": "collect_profile", "turn_type": "agent_reading"},
            },
            "observations": [{"name": "preflight"}, {"name": "sse_emit"}],
        }
        with (
            mock.patch.object(runner, "invoke_chat", return_value="event: done\n") as invoke,
            mock.patch.object(runner, "poll_trace_detail", return_value=("trace-1", trace)) as poll,
            mock.patch.object(runner, "get_trace_detail", return_value=trace),
            mock.patch.object(runner.uuid, "uuid4", return_value=SimpleNamespace(hex="fixed")),
            mock.patch.object(runner.time, "monotonic", side_effect=[100.0, 101.0, 102.0, 103.0]),
        ):
            result = runner.smoke_case(
                case={"id": "one", "message": "分析八字", "expected_route_primary": "bazi"},
                server_url="http://example.test",
                langfuse_url="http://langfuse.test",
                headers={},
                timeout_seconds=120,
                poll_interval_seconds=1,
                max_polls=1,
                write_scores=False,
            )

        self.assertTrue(result["passed"])
        self.assertEqual(invoke.call_args.args[3], 119)
        self.assertEqual(poll.call_args.kwargs["timeout_seconds"], 118)

    def test_smoke_case_supports_structured_response_and_trace_assertions(self):
        trace = {
            "metadata": {
                "resourceAttributes": {"service.name": "suanming-agent"},
                "attributes": {
                    "approved_route.primary_domain": "bazi",
                    "bazi.static.contract_audit": "clean",
                },
            },
            "observations": [{"name": "preflight"}, {"name": "sse_emit"}],
        }
        case = {
            "id": "contract",
            "message": "分析八字",
            "response_must_contain_all": ["强弱", "调候"],
            "response_must_not_contain": ["内部状态"],
            "expected_trace_attributes": {"bazi.static.contract_audit": "clean"},
        }
        with (
            mock.patch.object(runner, "invoke_chat", return_value="强弱\n调候\nevent: done\n"),
            mock.patch.object(runner, "poll_trace_detail", return_value=("trace-1", trace)),
            mock.patch.object(runner, "get_trace_detail", return_value=trace),
            mock.patch.object(runner.uuid, "uuid4", return_value=SimpleNamespace(hex="fixed")),
            mock.patch.object(runner.time, "monotonic", side_effect=[100.0, 101.0, 102.0, 103.0]),
        ):
            result = runner.smoke_case(
                case=case,
                server_url="http://example.test",
                langfuse_url="http://langfuse.test",
                headers={},
                timeout_seconds=120,
                poll_interval_seconds=1,
                max_polls=1,
                write_scores=False,
            )
        self.assertEqual(result["trace_attributes"]["bazi.static.contract_audit"], "clean")

    def test_smoke_case_rejects_forbidden_response_content(self):
        trace = {
            "metadata": {
                "resourceAttributes": {"service.name": "suanming-agent"},
                "attributes": {},
            },
            "observations": [{"name": "preflight"}, {"name": "sse_emit"}],
        }
        with (
            mock.patch.object(runner, "invoke_chat", return_value="疾病\nevent: done\n"),
            mock.patch.object(runner, "poll_trace_detail", return_value=("trace-1", trace)),
        ):
            with self.assertRaisesRegex(RuntimeError, "forbidden content"):
                runner.smoke_case(
                    case={"id": "forbidden", "message": "x", "response_must_not_contain": ["疾病"]},
                    server_url="http://example.test",
                    langfuse_url="http://langfuse.test",
                    headers={},
                    timeout_seconds=120,
                    poll_interval_seconds=1,
                    max_polls=1,
                    write_scores=False,
                )

    def test_classify_failure_uses_stable_categories(self):
        self.assertEqual(runner.classify_failure(RuntimeError("missing SSE done event")), "sse")
        self.assertEqual(runner.classify_failure(RuntimeError("route_primary mismatch: qimen")), "route")
        self.assertEqual(runner.classify_failure(RuntimeError("unexpected")), "unknown")

    def test_write_case_scores_binds_score_configs(self):
        with mock.patch.object(runner, "write_score") as write:
            runner.write_case_scores(
                "http://langfuse.test",
                {"Authorization": "Basic test"},
                "trace-1",
                {"eval_contract_pass": "pass-id", "eval_failure_class": "failure-id"},
                passed=False,
                error=RuntimeError("missing SSE done event"),
            )

        self.assertEqual(write.call_count, 2)
        self.assertEqual(write.call_args_list[0].kwargs["config_id"], "pass-id")
        self.assertEqual(write.call_args_list[0].kwargs["data_type"], "BOOLEAN")
        self.assertEqual(write.call_args_list[0].args[4], 0)
        self.assertEqual(write.call_args_list[1].kwargs["config_id"], "failure-id")
        self.assertEqual(write.call_args_list[1].args[4], "sse")

    def test_failed_case_result_preserves_trace_context(self):
        error = runner.SmokeCaseFailure("route_primary mismatch: qimen", session_id="session-1", trace_id="trace-1")
        result = runner.failed_case_result("case-1", error)

        self.assertFalse(result["passed"])
        self.assertEqual(result["session_id"], "session-1")
        self.assertEqual(result["trace_id"], "trace-1")
        self.assertEqual(result["failure_class"], "route")

    def test_timeout_failure_scores_trace_when_available(self):
        with (
            mock.patch.object(runner, "invoke_chat", side_effect=TimeoutError("backend request exceeded its total time budget")),
            mock.patch.object(runner, "poll_trace_detail", return_value=("trace-1", {})),
            mock.patch.object(runner, "write_case_scores") as write_scores,
        ):
            with self.assertRaises(runner.SmokeCaseFailure) as caught:
                runner.smoke_case(
                    case={"id": "one", "message": "分析八字"},
                    server_url="http://example.test",
                    langfuse_url="http://langfuse.test",
                    headers={},
                    timeout_seconds=120,
                    poll_interval_seconds=1,
                    max_polls=1,
                    write_scores=True,
                    score_config_ids={"eval_contract_pass": "pass-id", "eval_failure_class": "failure-id"},
                )

        self.assertEqual(caught.exception.trace_id, "trace-1")
        write_scores.assert_called_once()

    def test_backend_request_restores_alarm_handler_after_timeout(self):
        previous_handler = signal.getsignal(signal.SIGALRM)
        with (
            mock.patch.object(common.urllib.request, "urlopen", side_effect=TimeoutError("boom")),
            self.assertRaises(TimeoutError),
        ):
            common.backend_request("http://backend.test", {"message": "x"}, timeout=1)

        self.assertEqual(signal.getsignal(signal.SIGALRM), previous_handler)

    def test_backend_request_returns_after_sse_done_event(self):
