import pathlib
import json
import signal
import sys
import tempfile
import unittest
from types import SimpleNamespace
from unittest import mock

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import run_langfuse_eval as runner
import langfuse_eval_common as common
import write_human_answer_scores as human_scores
import run_answer_quality_judge as judge


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

    def test_parse_expected_any_values_preserves_list_items_with_commas(self):
        allowed = runner.parse_expected_any_values([
            "clean",
            "repaired: canonical_dynamic_projection_facts_only, contract_failure_class:domain_unauthorized, recovery_policy:dynamic_facts_only",
        ])

        self.assertEqual(
            allowed,
            [
                "clean",
                "repaired: canonical_dynamic_projection_facts_only, contract_failure_class:domain_unauthorized, recovery_policy:dynamic_facts_only",
            ],
        )

    def test_duplicate_json_keys_are_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = pathlib.Path(tmp) / "duplicate.json"
            path.write_text(
                '{"name":"duplicate","cases":[{"id":"x","expected_trace_attributes":{},"expected_trace_attributes":{}}]}',
                encoding="utf-8",
            )
            with self.assertRaisesRegex(ValueError, "duplicate JSON key: expected_trace_attributes"):
                runner.load_dataset(str(path))

    def test_report_contains_revision_dataset_and_endpoint_metadata(self):
        args = SimpleNamespace(
            server_url="http://127.0.0.1:18080",
            langfuse_url="http://localhost:3001",
            repeats=1,
        )
        with mock.patch.object(runner, "repository_revision", return_value="abc123"):
            report = runner.build_report(
                {"name": "smoke-v2", "version": "2"},
                args,
                [{"id": "one", "passed": True}],
            )

        self.assertEqual(report["git_revision"], "abc123")
        self.assertEqual(report["dataset_version"], "2")
        self.assertEqual(report["server_url"], "http://127.0.0.1:18080")
        self.assertIn("generated_at", report)

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

    def test_extract_response_text_decodes_sse_text_json(self):
        body = 'event: text\ndata: {"text":"强弱"}\n\nevent: text\ndata: {"payload":{"text":"调候"}}\n\nevent: done\ndata: {}\n\n'

        self.assertEqual(runner.extract_response_text(body), "强弱调候")

    def test_smoke_case_rejects_forbidden_sse_error_event(self):
        trace = {
            "metadata": {
                "resourceAttributes": {"service.name": "suanming-agent"},
                "attributes": {},
            },
            "observations": [{"name": "preflight"}, {"name": "sse_emit"}],
        }
        body = "event: error\ndata: {\"code\":\"BAZI_STATIC_PROJECTION_FAILED\"}\n\nevent: done\ndata: {}\n\n"
        with (
            mock.patch.object(runner, "invoke_chat", return_value=body),
            mock.patch.object(runner, "poll_trace_detail", return_value=("trace-1", trace)),
            mock.patch.object(runner.uuid, "uuid4", return_value=SimpleNamespace(hex="fixed")),
            mock.patch.object(runner.time, "monotonic", side_effect=[100.0, 101.0, 102.0, 103.0, 104.0]),
        ):
            with self.assertRaisesRegex(runner.SmokeCaseFailure, "forbidden SSE error event"):
                runner.smoke_case(
                    case={"id": "repair", "message": "分析八字", "sse_must_not_emit_error": True},
                    server_url="http://example.test",
                    langfuse_url="http://langfuse.test",
                    headers={},
                    timeout_seconds=120,
                    poll_interval_seconds=1,
                    max_polls=1,
                    write_scores=False,
                )

    def test_smoke_case_supports_answer_quality_checks(self):
        trace = {
            "metadata": {
                "resourceAttributes": {"service.name": "suanming-agent"},
                "attributes": {},
            },
            "observations": [{"name": "preflight"}, {"name": "sse_emit"}],
        }
        case = {
            "id": "quality",
            "message": "分析八字",
            "response_quality_checks": {
                "must_not_contain_any": ["未启用运行时规则 profile"],
                "max_phrase_occurrences": {"待规则裁断": 0},
                "max_total_phrase_occurrences": [
                    {"label": "deferred wording", "phrases": ["仅作结构观察", "证据不足"], "max": 1}
                ],
                "max_overview_conclusion_semicolons": 1,
                "max_heading_occurrences_in_section": [
                    {"section": "大运验证", "heading_prefix": "### ", "max": 1}
                ],
            },
        }
        body = "\n".join(
            [
                "event: text",
                "data: ## 总览结论",
                "",
                "event: text",
                "data: **结论：证据不足；仅作结构观察；待规则裁断**",
                "",
                "event: text",
                "data: 未启用运行时规则 profile",
                "",
                "event: text",
                "data: ## 大运验证\n### 丙戌运\n### 乙酉运",
                "",
                "event: done",
                "data: {}",
                "",
            ]
        )
        with (
            mock.patch.object(runner, "invoke_chat", return_value=body),
            mock.patch.object(runner, "poll_trace_detail", return_value=("trace-1", trace)),
            mock.patch.object(runner, "get_trace_detail", return_value=trace),
            mock.patch.object(runner.uuid, "uuid4", return_value=SimpleNamespace(hex="fixed")),
            mock.patch.object(runner.time, "monotonic", side_effect=[100.0, 101.0, 102.0, 103.0]),
        ):
            with self.assertRaises(runner.SmokeCaseFailure) as caught:
                runner.smoke_case(
                    case=case,
                    server_url="http://example.test",
                    langfuse_url="http://langfuse.test",
                    headers={},
                    timeout_seconds=120,
                    poll_interval_seconds=1,
                    max_polls=1,
                    write_scores=False,
                    include_response=True,
                )

        self.assertEqual(caught.exception.trace_id, "trace-1")
        self.assertIn("未启用运行时规则 profile", caught.exception.response_text)
        self.assertGreaterEqual(len(caught.exception.quality_violations), 3)

    def test_smoke_case_can_include_response_text_in_success_report(self):
        trace = {
            "metadata": {
                "resourceAttributes": {"service.name": "suanming-agent"},
                "attributes": {"approved_route.primary_domain": "bazi"},
            },
            "observations": [{"name": "preflight"}, {"name": "sse_emit"}],
        }
        body = "event: text\ndata: 强弱调候大运\n\nevent: done\ndata: {}\n\n"
        with (
            mock.patch.object(runner, "invoke_chat", return_value=body),
            mock.patch.object(runner, "poll_trace_detail", return_value=("trace-1", trace)),
            mock.patch.object(runner, "get_trace_detail", return_value=trace),
            mock.patch.object(runner.uuid, "uuid4", return_value=SimpleNamespace(hex="fixed")),
            mock.patch.object(runner.time, "monotonic", side_effect=[100.0, 101.0, 102.0, 103.0]),
        ):
            result = runner.smoke_case(
                case={"id": "quality-pass", "message": "分析八字", "response_must_contain_all": ["强弱", "调候"]},
                server_url="http://example.test",
                langfuse_url="http://langfuse.test",
                headers={},
                timeout_seconds=120,
                poll_interval_seconds=1,
                max_polls=1,
                write_scores=False,
                include_response=True,
            )

        self.assertEqual(result["response_text"], "强弱调候大运")
        self.assertEqual(result["quality_violations"], [])

    def test_smoke_case_supports_trace_attribute_any_assertions(self):
        trace = {
            "metadata": {
                "resourceAttributes": {"service.name": "suanming-agent"},
                "attributes": {
                    "approved_route.primary_domain": "bazi",
                    "bazi.dynamic.source": "facts_only_degraded",
                },
            },
            "observations": [{"name": "preflight"}, {"name": "sse_emit"}],
        }
        case = {
            "id": "contract-any",
            "message": "分析八字",
            "expected_trace_attribute_any": {"bazi.dynamic.source": ["model", "facts_only_degraded"]},
        }
        with (
            mock.patch.object(runner, "invoke_chat", return_value="event: done\n"),
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
        self.assertEqual(result["trace_attributes"]["bazi.dynamic.source"], "facts_only_degraded")

    def test_smoke_case_rejects_forbidden_trace_attribute(self):
        trace = {
            "metadata": {
                "resourceAttributes": {"service.name": "suanming-agent"},
                "attributes": {"bazi.final_writer.validation_err": "bad"},
            },
            "observations": [{"name": "preflight"}, {"name": "sse_emit"}],
        }
        with (
            mock.patch.object(runner, "invoke_chat", return_value="event: done\n"),
            mock.patch.object(runner, "poll_trace_detail", return_value=("trace-1", trace)),
            mock.patch.object(runner, "get_trace_detail", return_value=trace),
            mock.patch.object(runner.uuid, "uuid4", return_value=SimpleNamespace(hex="fixed")),
            mock.patch.object(runner.time, "monotonic", side_effect=[100.0, 101.0, 102.0, 103.0, 104.0]),
        ):
            with self.assertRaisesRegex(runner.SmokeCaseFailure, "forbidden trace attribute"):
                runner.smoke_case(
                    case={
                        "id": "forbidden-trace",
                        "message": "分析八字",
                        "trace_attributes_must_be_absent": ["bazi.final_writer.validation_err"],
                    },
                    server_url="http://example.test",
                    langfuse_url="http://langfuse.test",
                    headers={},
                    timeout_seconds=120,
                    poll_interval_seconds=1,
                    max_polls=1,
                    write_scores=False,
                )

    def test_smoke_case_reports_optional_trace_attributes_without_requiring_them(self):
        trace = {
            "metadata": {
                "resourceAttributes": {"service.name": "suanming-agent"},
                "attributes": {
                    "repair.stage": "dynamic_projection",
                },
            },
            "observations": [{"name": "preflight"}, {"name": "sse_emit"}],
        }
        case = {
            "id": "optional-trace",
            "message": "分析八字",
            "optional_trace_attribute_any": {
                "repair.stage": ["static_projection", "dynamic_projection"],
                "repair.field": [],
            },
        }
        with (
            mock.patch.object(runner, "invoke_chat", return_value="event: done\n"),
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
        self.assertEqual(result["trace_attributes"]["repair.stage"], "dynamic_projection")
        self.assertEqual(result["trace_attributes"]["repair.field"], "")

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
        self.assertEqual(runner.classify_failure(RuntimeError("<urlopen error [Errno 111] Connection refused>")), "transport")
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

    def test_failure_class_counts_aggregates_repeated_results(self):
        results = [
            {"passed": True},
            {"passed": False, "failure_class": "sse"},
            {"passed": False, "error": "route_primary mismatch: qimen"},
        ]
        self.assertEqual(runner.failure_class_counts(results), {"sse": 1, "route": 1})

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
        with mock.patch.object(common.urllib.request, "urlopen", return_value=self.FakeSSEResponse()):
            status, body = common.backend_request("http://backend.test", {"message": "x"}, timeout=30)

        self.assertEqual(status, 200)
        self.assertIn("event: done", body)

    def test_human_answer_score_items_validate_values(self):
        case = {
            "id": "case-1",
            "scores": {
                "answer_task_complete": True,
                "answer_factuality_pass": False,
                "answer_grounding_pass": None,
                "answer_scope_safe": True,
                "answer_failure_class": "task",
            },
        }

        items = human_scores.score_items(case)

        self.assertEqual(
            items,
            [
                ("answer_task_complete", True, "BOOLEAN"),
                ("answer_factuality_pass", False, "BOOLEAN"),
                ("answer_scope_safe", True, "BOOLEAN"),
                ("answer_failure_class", "task", "CATEGORICAL"),
            ],
        )
        with self.assertRaisesRegex(RuntimeError, "must be true/false"):
            human_scores.score_items({"id": "bad", "scores": {"answer_task_complete": 1}})
        with self.assertRaisesRegex(RuntimeError, "not in"):
            human_scores.score_items({"id": "bad", "scores": {"answer_failure_class": "timeout"}})

    def test_human_answer_scores_dry_run_does_not_write(self):
        annotation = {
            "name": "answer-quality-human-v1",
            "cases": [
                {
                    "id": "case-1",
                    "trace_id": "trace-1",
                    "scores": {
                        "answer_task_complete": True,
                        "answer_failure_class": "none",
                    },
                    "human_note": "ok",
                }
            ],
        }
        score_config_ids = {
            "answer_task_complete": "task-id",
            "answer_factuality_pass": "fact-id",
            "answer_grounding_pass": "ground-id",
            "answer_scope_safe": "scope-id",
            "answer_failure_class": "failure-id",
        }
        with mock.patch.object(human_scores, "write_score") as write:
            summary = human_scores.write_annotation_scores(
                annotation,
                "http://langfuse.test",
                {"Authorization": "Basic test"},
                score_config_ids,
                dry_run=True,
                allow_duplicates=True,
            )

        write.assert_not_called()
        self.assertEqual(summary["scores_ready"], 2)
        self.assertEqual(summary["scores_written"], 0)

    def test_human_answer_scores_write_binds_score_configs(self):
        annotation = {
            "name": "answer-quality-human-v1",
            "cases": [
                {
                    "id": "case-1",
                    "trace_id": "trace-1",
                    "scores": {
                        "answer_task_complete": False,
                        "answer_failure_class": "task",
                    },
                    "human_note": "missed task",
                }
            ],
        }
        score_config_ids = {
            "answer_task_complete": "task-id",
            "answer_factuality_pass": "fact-id",
            "answer_grounding_pass": "ground-id",
            "answer_scope_safe": "scope-id",
            "answer_failure_class": "failure-id",
        }
        with mock.patch.object(human_scores, "write_score") as write:
            summary = human_scores.write_annotation_scores(
                annotation,
                "http://langfuse.test",
                {"Authorization": "Basic test"},
                score_config_ids,
                dry_run=False,
                allow_duplicates=True,
            )

        self.assertEqual(summary["scores_written"], 2)
        self.assertEqual(write.call_args_list[0].kwargs["config_id"], "task-id")
        self.assertEqual(write.call_args_list[0].kwargs["data_type"], "BOOLEAN")
        self.assertEqual(write.call_args_list[0].args[4], 0)
        self.assertEqual(write.call_args_list[1].kwargs["config_id"], "failure-id")
        self.assertEqual(write.call_args_list[1].kwargs["data_type"], "CATEGORICAL")
        self.assertEqual(write.call_args_list[1].args[4], "task")

    def test_human_answer_scores_skip_existing_by_default(self):
        annotation = {
            "name": "answer-quality-human-v1",
            "cases": [
                {
                    "id": "case-1",
                    "trace_id": "trace-1",
                    "scores": {
                        "answer_task_complete": False,
                        "answer_failure_class": "task",
                    },
                }
            ],
        }
        score_config_ids = {
            "answer_task_complete": "task-id",
            "answer_factuality_pass": "fact-id",
            "answer_grounding_pass": "ground-id",
            "answer_scope_safe": "scope-id",
            "answer_failure_class": "failure-id",
        }
        with (
            mock.patch.object(human_scores, "existing_score_names", return_value={"answer_task_complete"}),
            mock.patch.object(human_scores, "write_score") as write,
        ):
            summary = human_scores.write_annotation_scores(
                annotation,
                "http://langfuse.test",
                {"Authorization": "Basic test"},
                score_config_ids,
                dry_run=False,
            )

        self.assertEqual(summary["scores_written"], 1)
        self.assertEqual(summary["scores_skipped_existing"], 1)
        self.assertEqual(write.call_args.args[3], "answer_failure_class")

    def test_answer_judge_uses_explicit_eval_env_first(self):
        config = judge.require_judge_config(
            {
                "EVAL_JUDGE_API_KEY": "judge-key",
                "EVAL_JUDGE_BASE_URL": "https://judge.example.com/v1",
                "EVAL_JUDGE_MODEL": "judge-model",
                "LLM_API_KEY": "backend-key",
                "LLM_BASE_URL": "https://api.deepseek.com/anthropic",
            }
        )

        self.assertEqual(config["api_key"], "judge-key")
        self.assertEqual(config["base_url"], "https://judge.example.com/v1")
        self.assertEqual(config["model"], "judge-model")
        self.assertEqual(config["source"], "eval_judge_env")

    def test_answer_judge_falls_back_to_backend_deepseek_flash(self):
        config = judge.require_judge_config(
            {
                "LLM_API_KEY": "backend-key",
                "LLM_BASE_URL": "https://api.deepseek.com/anthropic",
                "LLM_MODEL": "deepseek-v4-pro",
            }
        )

        self.assertEqual(config["api_key"], "backend-key")
        self.assertEqual(config["base_url"], "https://api.deepseek.com")
        self.assertEqual(config["model"], "deepseek-v4-flash")
        self.assertEqual(config["source"], "backend_llm_deepseek_flash_fallback")

    def test_answer_judge_current_mode_reads_response_from_online_report(self):
        dataset = {"name": "bazi-quality-v2", "version": "2", "cases": [{"id": "case-1", "message": "分析八字"}]}
        report = {
            "dataset": "bazi-quality-v2",
            "dataset_version": "2",
            "generated_at": "2026-08-15T00:00:00+00:00",
            "results": [{"id": "case-1", "trace_id": "trace-1", "response_text": "## 强弱视角"}],
        }
        with tempfile.TemporaryDirectory() as tmp:
            dataset_path = pathlib.Path(tmp) / "dataset.json"
            report_path = pathlib.Path(tmp) / "report.json"
            dataset_path.write_text(json.dumps(dataset), encoding="utf-8")
            report_path.write_text(json.dumps(report), encoding="utf-8")
            summary = judge.run_current_report(
                SimpleNamespace(
                    dataset_path=str(dataset_path),
                    current_report_path=str(report_path),
                    case_id=[],
                    dry_run=True,
                    timeout_seconds=60,
                    write_scores=False,
                    allow_duplicates=False,
                    langfuse_url="",
                )
            )

        self.assertEqual(summary["mode"], "current")
        self.assertEqual(summary["dataset_version"], "2")
        self.assertEqual(summary["rows"][0]["status"], "dry_run")
        self.assertIn("答案质量评测器", summary["rows"][0]["prompt_preview"])
        self.assertIn("分析八字", judge.build_judge_prompt({"message": "分析八字", "output": "x"}))

    def test_answer_judge_current_scores_write_to_trace_without_human_labels(self):
        row = {
            "id": "case-1",
            "trace_id": "trace-1",
            "judge_reason": "通过",
            "judge_scores": {
                "answer_task_complete": True,
                "answer_factuality_pass": True,
                "answer_grounding_pass": True,
                "answer_scope_safe": True,
                "answer_failure_class": "none",
            },
        }
        with (
            mock.patch.object(judge, "existing_score_names", return_value=set()),
            mock.patch.object(judge, "write_score") as write,
        ):
            result = judge.write_current_judge_scores("http://langfuse.test", {}, row, allow_duplicates=False)

        self.assertEqual(result, {"written": 5, "skipped_existing": 0})
        self.assertEqual(write.call_count, 5)
        self.assertEqual(write.call_args_list[0].args[3], "judge_answer_task_complete")


if __name__ == "__main__":
    unittest.main()
