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
    get_observation_names,
    get_route_primary,
    get_trace_field,
    langfuse_headers,
    load_env_map,
    poll_trace_detail,
    write_score,
)


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


def smoke_case(
    case: dict[str, Any],
    server_url: str,
    langfuse_url: str,
    headers: dict[str, str],
    timeout_seconds: int,
    poll_interval_seconds: int,
    max_polls: int,
    write_scores: bool,
) -> dict[str, Any]:
    session_base = str(case.get("session_id") or f"eval-{case['id']}")
    session_id = f"{session_base}-{uuid.uuid4().hex}"

    setup_message = str(case.get("setup_message") or "").strip()
    if setup_message:
        invoke_chat(server_url, session_id, setup_message, timeout_seconds)

    chat_content = invoke_chat(server_url, session_id, str(case.get("message") or ""), timeout_seconds)
    if "event: done" not in chat_content:
        raise RuntimeError("missing SSE done event")

    response_must_contain = str(case.get("response_must_contain") or "").strip()
    if response_must_contain and response_must_contain not in chat_content:
        raise RuntimeError(f"response missing expected content: {response_must_contain}")

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
    deadline = time.time() + max(1, timeout_seconds)
    while time.time() < deadline:
        trace_id, trace_detail = poll_trace_detail(
            langfuse_url,
            headers,
            session_id,
            timeout_seconds=max(1, timeout_seconds),
            poll_interval_seconds=max(1, poll_interval_seconds),
            max_limit=max_polls,
        )
        route_primary = get_route_primary(trace_detail)
        task_intent = get_trace_field(trace_detail, "task_intent")
        turn_type = get_trace_field(trace_detail, "turn_type")
        service_name = get_trace_field(trace_detail, "service.name")
        observation_names = get_observation_names(trace_detail)

        if service_name != "suanming-agent":
            raise RuntimeError(f"service.name mismatch: {service_name}")

        if expected_route_primary and route_primary != expected_route_primary:
            time.sleep(max(1, poll_interval_seconds))
            continue
        if expected_task_intents and task_intent not in expected_task_intents:
            time.sleep(max(1, poll_interval_seconds))
            continue
        if expected_turn_types and turn_type not in expected_turn_types:
            time.sleep(max(1, poll_interval_seconds))
            continue
        if any(name not in observation_names for name in required_observations):
            time.sleep(max(1, poll_interval_seconds))
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

    if write_scores and headers:
        write_score(langfuse_url, headers, trace_id, "langfuse_smoke_pass", 1)
        write_score(langfuse_url, headers, trace_id, "langfuse_trace_present", 1)
        write_score(langfuse_url, headers, trace_id, "langfuse_required_observations", 1)
        write_score(langfuse_url, headers, trace_id, "sse_done", 1)
        if expected_route_primary:
            write_score(langfuse_url, headers, trace_id, "route_primary_match", 1)
        if expected_task_intents:
            write_score(langfuse_url, headers, trace_id, "task_intent_match", 1)
        if expected_turn_types:
            write_score(langfuse_url, headers, trace_id, "turn_type_match", 1)
        for name in required_observations:
            write_score(langfuse_url, headers, trace_id, f"observation_{name}_present", 1)

    return {
        "id": str(case["id"]),
        "passed": True,
        "session_id": session_id,
        "trace_id": trace_id,
        "route_primary": route_primary,
        "task_intent": task_intent,
        "turn_type": turn_type,
        "observations": observation_names,
    }


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
            )
            results.append(result)
        except Exception as exc:  # noqa: BLE001
            results.append({"id": str(case["id"]), "passed": False, "error": str(exc)})

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
