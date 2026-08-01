from __future__ import annotations

import argparse
import json
import pathlib
import sys
import time
import uuid
from typing import Any

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from langfuse_eval_common import (  # noqa: E402
    backend_request,
    get_trace_detail,
    get_observation_names,
    get_route_primary,
    get_trace_field,
    langfuse_headers,
    load_env_map,
    poll_trace_detail,
    require_score_config_ids,
    write_score,
)

EVAL_SCORE_CONFIG_TYPES = {
    "eval_contract_pass": "BOOLEAN",
    "eval_failure_class": "CATEGORICAL",
}


class SmokeCaseFailure(RuntimeError):
    """Carry trace context for a failed case so reports and scores remain diagnosable."""

    def __init__(self, message: str, session_id: str = "", trace_id: str = ""):
        super().__init__(message)
        self.session_id = session_id
        self.trace_id = trace_id


def parse_csv_list(values: list[str]) -> list[str]:
    items: list[str] = []
    for value in values:
        for part in str(value).split(","):
            part = part.strip()
            if part:
                items.append(part)
    return items


def load_dataset(dataset_path: str) -> dict[str, Any]:
    path = pathlib.Path(dataset_path)
    if not path.exists():
        raise FileNotFoundError(f"dataset not found: {dataset_path}")
    payload = json.loads(path.read_text(encoding="utf-8"))
    cases = list(payload.get("cases") or [])
    if not cases:
        raise RuntimeError(f"dataset has no cases: {dataset_path}")
    return payload


def invoke_chat(server_url: str, session_id: str, message: str, timeout_seconds: int) -> str:
    status, body = backend_request(server_url, {"session_id": session_id, "message": message}, timeout=timeout_seconds)
    if status != 200:
        raise RuntimeError(f"chat status = {status}")
    return body


def remaining_timeout_seconds(deadline: float) -> int:
    """Return a positive request timeout without exceeding one case deadline."""
    remaining = deadline - time.monotonic()
    if remaining <= 0:
        raise TimeoutError("evaluation case exceeded its total time budget")
    return max(1, int(remaining + 0.999))


def case_timeout_seconds(case: dict[str, Any], default_timeout_seconds: int) -> int:
    """Return a positive case-specific budget, falling back to the suite default."""
    value = case.get("timeout_seconds", default_timeout_seconds)
    try:
        timeout_seconds = int(value)
    except (TypeError, ValueError) as exc:
        raise ValueError(f"invalid timeout_seconds for case {case.get('id', '')}: {value!r}") from exc
    if timeout_seconds <= 0:
        raise ValueError(f"timeout_seconds for case {case.get('id', '')} must be positive")
    return timeout_seconds


def unique_case_session_id(session_base: str) -> str:
    """Build a storage-safe unique session id within the backend's 64-char contract."""
    suffix = uuid.uuid4().hex
    prefix = str(session_base).strip() or "eval"
    prefix = prefix[: 64 - len(suffix) - 1]
    return f"{prefix}-{suffix}"


def classify_failure(error: Exception) -> str:
    """Map a runner error to the stable categorical score used for aggregation."""
    message = str(error).lower()
    if "timeout" in message or "exceeded its total time budget" in message:
        return "timeout"
    if "sse done" in message:
        return "sse"
    if "response missing" in message or "forbidden content" in message:
        return "response_contract"
    if "route_primary mismatch" in message:
        return "route"
    if "task_intent mismatch" in message:
        return "task_intent"
    if "turn_type mismatch" in message:
        return "turn_type"
    if "missing observation" in message:
        return "observation"
    if "trace attribute mismatch" in message:
        return "trace_attribute"
    if "http" in message or "trace not found" in message or "service.name mismatch" in message:
        return "transport"
    return "unknown"


def write_case_scores(langfuse_url, headers, trace_id, score_config_ids, passed, error=None):
    """Persist the complete eval verdict once a trace id exists for the evaluated turn."""
    comment = "" if passed else str(error)
    write_score(
        langfuse_url,
        headers,
        trace_id,
        "eval_contract_pass",
        1 if passed else 0,
        comment=comment,
        config_id=score_config_ids["eval_contract_pass"],
        data_type="BOOLEAN",
    )
    if not passed:
        write_score(
            langfuse_url,
            headers,
            trace_id,
            "eval_failure_class",
            classify_failure(error),
            comment=comment,
            config_id=score_config_ids["eval_failure_class"],
            data_type="CATEGORICAL",
        )


def write_failure_scores_if_trace_exists(
    langfuse_url,
    headers,
    session_id,
    excluded_trace_ids,
    max_polls,
    score_config_ids,
    error,
) -> str:
    """Best-effort score a failed case when the backend emitted a trace before timing out."""
    if not score_config_ids:
        return ""
    try:
        trace_id, _ = poll_trace_detail(
            langfuse_url,
            headers,
            session_id,
            timeout_seconds=10,
            poll_interval_seconds=1,
            max_limit=max_polls,
            excluded_trace_ids=excluded_trace_ids,
        )
    except Exception:  # noqa: BLE001
        return ""
    write_case_scores(langfuse_url, headers, trace_id, score_config_ids, passed=False, error=error)
    return trace_id


def smoke_case(
    case: dict[str, Any],
    server_url: str,
    langfuse_url: str,
    headers: dict[str, str],
    timeout_seconds: int,
    poll_interval_seconds: int,
    max_polls: int,
    write_scores: bool,
    score_config_ids: dict[str, str] | None = None,
) -> dict[str, Any]:
    session_base = str(case.get("session_id") or f"eval-{case['id']}")
    session_id = unique_case_session_id(session_base)
    deadline = time.monotonic() + case_timeout_seconds(case, timeout_seconds)

    setup_message = str(case.get("setup_message") or "").strip()
    excluded_trace_ids: set[str] = set()
    if setup_message:
        invoke_chat(server_url, session_id, setup_message, remaining_timeout_seconds(deadline))
        setup_trace_id, _ = poll_trace_detail(
            langfuse_url,
            headers,
            session_id,
            timeout_seconds=remaining_timeout_seconds(deadline),
            poll_interval_seconds=max(1, poll_interval_seconds),
            max_limit=max_polls,
        )
        excluded_trace_ids.add(setup_trace_id)

    try:
        chat_content = invoke_chat(
            server_url,
            session_id,
            str(case.get("message") or ""),
            remaining_timeout_seconds(deadline),
        )
    except Exception as exc:
        trace_id = write_failure_scores_if_trace_exists(
            langfuse_url,
            headers,
            session_id,
            excluded_trace_ids,
            max_polls,
            score_config_ids if write_scores else None,
            exc,
        )
        raise SmokeCaseFailure(str(exc), session_id=session_id, trace_id=trace_id) from exc
    expected_route_primary = str(case.get("expected_route_primary") or "").strip()
    expected_task_intents = parse_csv_list(list(case.get("expected_task_intent_any") or []))
    expected_turn_types = parse_csv_list(list(case.get("expected_turn_type_any") or []))
    required_observations = parse_csv_list(list(case.get("required_observations") or ["preflight", "sse_emit"]))
    trace_id = ""
    trace_detail: dict[str, Any] = {}
    route_primary = ""
    task_intent = ""
    turn_type = ""
    service_name = ""
    observation_names: list[str] = []
    trace_id, trace_detail = poll_trace_detail(
        langfuse_url,
        headers,
        session_id,
        timeout_seconds=remaining_timeout_seconds(deadline),
        poll_interval_seconds=max(1, poll_interval_seconds),
        max_limit=max_polls,
        excluded_trace_ids=excluded_trace_ids,
    )

    try:
        if "event: done" not in chat_content:
            raise RuntimeError("missing SSE done event")

        response_must_contain = str(case.get("response_must_contain") or "").strip()
        if response_must_contain and response_must_contain not in chat_content:
            raise RuntimeError(f"response missing expected content: {response_must_contain}")
        for expected in case.get("response_must_contain_all") or []:
            expected = str(expected).strip()
            if expected and expected not in chat_content:
                raise RuntimeError(f"response missing expected content: {expected}")
        for forbidden in case.get("response_must_not_contain") or []:
            forbidden = str(forbidden).strip()
            if forbidden and forbidden in chat_content:
                raise RuntimeError(f"response contains forbidden content: {forbidden}")

        while time.monotonic() < deadline:
            trace_detail = get_trace_detail(langfuse_url, headers, trace_id)
            route_primary = get_route_primary(trace_detail)
            task_intent = get_trace_field(trace_detail, "task_intent")
            turn_type = get_trace_field(trace_detail, "turn_type")
            service_name = get_trace_field(trace_detail, "service.name")
            observation_names = get_observation_names(trace_detail)

            if service_name != "suanming-agent":
                raise RuntimeError(f"service.name mismatch: {service_name}")

            if expected_route_primary and route_primary != expected_route_primary:
                time.sleep(min(max(1, poll_interval_seconds), remaining_timeout_seconds(deadline)))
                continue
            if expected_task_intents and task_intent not in expected_task_intents:
                time.sleep(min(max(1, poll_interval_seconds), remaining_timeout_seconds(deadline)))
                continue
            if expected_turn_types and turn_type not in expected_turn_types:
                time.sleep(min(max(1, poll_interval_seconds), remaining_timeout_seconds(deadline)))
                continue
            if any(name not in observation_names for name in required_observations):
                time.sleep(min(max(1, poll_interval_seconds), remaining_timeout_seconds(deadline)))
                continue
            break

        if expected_route_primary and route_primary != expected_route_primary:
            raise RuntimeError(f"route_primary mismatch: {route_primary}")
        if expected_task_intents and task_intent not in expected_task_intents:
            raise RuntimeError(f"task_intent mismatch: {task_intent}")
        if expected_turn_types and turn_type not in expected_turn_types:
            raise RuntimeError(f"turn_type mismatch: {turn_type}")
        for name in required_observations:
            if name not in observation_names:
                raise RuntimeError(f"missing observation: {name}")

        expected_trace_attributes = case.get("expected_trace_attributes") or {}
        for key, expected in expected_trace_attributes.items():
            actual = get_trace_field(trace_detail, str(key))
            if str(actual) != str(expected):
                raise RuntimeError(f"trace attribute mismatch for {key}: {actual!r} != {expected!r}")

        if write_scores:
            if not score_config_ids:
                raise RuntimeError("missing resolved Langfuse ScoreConfig ids")
            write_case_scores(langfuse_url, headers, trace_id, score_config_ids, passed=True)

        return {
            "id": str(case["id"]),
            "passed": True,
            "session_id": session_id,
            "trace_id": trace_id,
            "route_primary": route_primary,
            "task_intent": task_intent,
            "turn_type": turn_type,
            "observations": observation_names,
            "trace_attributes": {
                str(key): get_trace_field(trace_detail, str(key))
                for key in expected_trace_attributes
            },
        }
    except Exception as exc:
        if write_scores and score_config_ids:
            write_case_scores(langfuse_url, headers, trace_id, score_config_ids, passed=False, error=exc)
        raise SmokeCaseFailure(str(exc), session_id=session_id, trace_id=trace_id) from exc


def failed_case_result(case_id: str, error: Exception) -> dict[str, Any]:
    """Build a report row for failures, preserving trace context when available."""
    result = {"id": case_id, "passed": False, "error": str(error)}
    if isinstance(error, SmokeCaseFailure):
        if error.session_id:
            result["session_id"] = error.session_id
        if error.trace_id:
            result["trace_id"] = error.trace_id
            result["failure_class"] = classify_failure(error)
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description="Run a local eval dataset against /api/chat and Langfuse")
    parser.add_argument("--dataset-path", required=True)
    parser.add_argument("--server-url", default="http://localhost:8080")
    parser.add_argument("--langfuse-url", default="http://localhost:3001")
    parser.add_argument("--report-path", default="")
    parser.add_argument("--timeout-seconds", type=int, default=120)
    parser.add_argument("--poll-interval-seconds", type=int, default=3)
    parser.add_argument("--max-polls", type=int, default=20)
    parser.add_argument("--write-scores", action="store_true")
    args = parser.parse_args()

    payload = load_dataset(args.dataset_path)
    env_map = load_env_map()
    headers = langfuse_headers(env_map)
    score_config_ids = require_score_config_ids(args.langfuse_url, headers, EVAL_SCORE_CONFIG_TYPES) if args.write_scores else None

    results: list[dict[str, Any]] = []
    for case in payload["cases"]:
        try:
            result = smoke_case(
                case=case,
                server_url=args.server_url,
                langfuse_url=args.langfuse_url,
                headers=headers,
                timeout_seconds=args.timeout_seconds,
                poll_interval_seconds=args.poll_interval_seconds,
                max_polls=args.max_polls,
                write_scores=args.write_scores,
                score_config_ids=score_config_ids,
            )
            results.append(result)
        except Exception as exc:  # noqa: BLE001
            results.append(failed_case_result(str(case["id"]), exc))

    summary = {
        "dataset": str(payload["name"]),
        "passed": sum(1 for item in results if item["passed"]),
        "failed": sum(1 for item in results if not item["passed"]),
        "results": results,
    }

    if args.report_path:
        report_path = pathlib.Path(args.report_path)
        report_path.parent.mkdir(parents=True, exist_ok=True)
        report_path.write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")

    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0 if summary["failed"] == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
