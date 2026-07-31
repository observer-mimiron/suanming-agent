import pathlib
import json
import sys
import unittest
from types import SimpleNamespace
from unittest import mock

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import run_langfuse_eval as runner
import langfuse_eval_common as common


class EvalTimeoutTest(unittest.TestCase):
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


if __name__ == "__main__":
    unittest.main()
