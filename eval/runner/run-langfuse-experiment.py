import argparse
import json
import pathlib
import time

from langfuse_eval_common import (
    backend_request,
    create_dataset_run_item,
    ensure_dataset,
    get_observation_names,
    get_route_primary,
    get_trace_detail,
    get_trace_field,
    langfuse_headers,
    load_env_map,
    poll_trace_detail,
    repo_root,
    unique_session_id,
    upsert_dataset_item,
    write_score,
)


def as_list(value):
    if value is None:
        return []
    if isinstance(value, list):
        return value
    return [value]


def fallback_field_from_observations(trace_detail, primary_key, alternate_key=""):
    for observation in trace_detail.get("observations", []):
        metadata = observation.get("metadata") or {}
        attributes = metadata.get("attributes") or {}
        if primary_key and attributes.get(primary_key) not in (None, ""):
            return attributes.get(primary_key)
        if alternate_key and attributes.get(alternate_key) not in (None, ""):
            return attributes.get(alternate_key)
    return ""


def assert_case(case, trace_detail, chat_content):
    route_primary = get_route_primary(trace_detail)
    task_intent = get_trace_field(trace_detail, "task_intent")
    turn_type = get_trace_field(trace_detail, "turn_type")
    service_name = get_trace_field(trace_detail, "service.name")
    observations = get_observation_names(trace_detail)

    if not route_primary:
        route_primary = fallback_field_from_observations(trace_detail, "primary_domain", "approved_route.primary_domain")
    if not task_intent:
        task_intent = fallback_field_from_observations(trace_detail, "task_intent")
    if not turn_type:
        turn_type = fallback_field_from_observations(trace_detail, "turn_type")

    if "event: done" not in chat_content:
        raise RuntimeError("missing SSE done event")
    if service_name != "suanming-agent":
        raise RuntimeError(f"service.name mismatch: {service_name}")

    expected_route = case.get("expected_route_primary")
    if expected_route and route_primary != expected_route:
        raise RuntimeError(f"route_primary mismatch: {route_primary}")

    expected_task = as_list(case.get("expected_task_intent_any"))
    if expected_task and task_intent not in expected_task:
        raise RuntimeError(f"task_intent mismatch: {task_intent}")

    expected_turn = as_list(case.get("expected_turn_type_any"))
    if expected_turn and turn_type not in expected_turn:
        raise RuntimeError(f"turn_type mismatch: {turn_type}")

    for observation in as_list(case.get("required_observations")):
        if observation not in observations:
            raise RuntimeError(f"missing observation: {observation}")

    return {
        "route_primary": route_primary,
        "task_intent": task_intent,
        "turn_type": turn_type,
        "observations": observations,
    }


def wait_for_case_verdict(case, trace_id, trace_detail, chat_content, langfuse_url, headers, timeout_seconds, poll_interval_seconds):
    deadline = time.time() + timeout_seconds
    last_error = None
    current_detail = trace_detail

    while time.time() < deadline:
        try:
            return assert_case(case, current_detail, chat_content)
        except RuntimeError as exc:
            last_error = exc
            time.sleep(poll_interval_seconds)
            current_detail = get_trace_detail(langfuse_url, headers, trace_id)

    raise last_error if last_error else RuntimeError("case verdict timeout")


def sync_case_dataset_item(langfuse_url, headers, dataset_name, case):
    item_id = f"{dataset_name}:{case['id']}"
    input_payload = {"message": case.get("message", "")}
    if case.get("setup_message"):
        input_payload["setup_message"] = case["setup_message"]
    expected_output = {
        "expected_route_primary": case.get("expected_route_primary", ""),
        "expected_task_intent_any": case.get("expected_task_intent_any", []),
        "expected_turn_type_any": case.get("expected_turn_type_any", []),
        "required_observations": case.get("required_observations", []),
    }
    upsert_dataset_item(
        langfuse_url,
        headers,
        dataset_name,
        item_id,
        input_payload,
        expected_output,
        {"case_id": case["id"], "source": "local-eval-json"},
    )
    return item_id


def main():
    parser = argparse.ArgumentParser(
        description="Run a local eval dataset against /api/chat and register dataset run items in Langfuse."
    )
    parser.add_argument("--dataset-path", required=True)
    parser.add_argument("--server-url", default="http://localhost:8080")
    parser.add_argument("--langfuse-url", default="http://localhost:3001")
    parser.add_argument("--env-file", default=str(repo_root() / "backend" / ".env"))
    parser.add_argument("--run-name", default="")
    parser.add_argument("--run-description", default="")
    parser.add_argument("--timeout-seconds", type=int, default=120)
    parser.add_argument("--poll-interval-seconds", type=int, default=3)
    parser.add_argument("--write-scores", action="store_true")
    parser.add_argument("--report-path", default="")
    args = parser.parse_args()

    dataset_file = pathlib.Path(args.dataset_path).resolve()
    payload = json.loads(dataset_file.read_text(encoding="utf-8"))
    dataset_name = payload["name"]
    run_name = args.run_name or f"{dataset_name}-run"

    env_map = load_env_map(args.env_file)
    headers = langfuse_headers(env_map)

    ensure_dataset(
        args.langfuse_url,
        headers,
        dataset_name,
        payload.get("description", ""),
        {"source_path": str(dataset_file.relative_to(repo_root()))},
    )

    results = []
    for case in payload.get("cases", []):
        session_base = case.get("session_id") or f"eval-{case['id']}"
        session_id = unique_session_id(session_base)

        if case.get("setup_message"):
            status, _ = backend_request(
                args.server_url,
                {"session_id": session_id, "message": case["setup_message"]},
                timeout=args.timeout_seconds,
            )
            if status != 200:
                raise RuntimeError(f"setup request failed for {case['id']}: status={status}")

        status, chat_content = backend_request(
            args.server_url,
            {"session_id": session_id, "message": case["message"]},
            timeout=args.timeout_seconds,
        )
        if status != 200:
            raise RuntimeError(f"chat request failed for {case['id']}: status={status}")

        trace_id, trace_detail = poll_trace_detail(
            args.langfuse_url,
            headers,
            session_id,
            timeout_seconds=args.timeout_seconds,
            poll_interval_seconds=args.poll_interval_seconds,
        )

        verdict = wait_for_case_verdict(
            case,
            trace_id,
            trace_detail,
            chat_content,
            args.langfuse_url,
            headers,
            args.timeout_seconds,
            args.poll_interval_seconds,
        )

        if args.write_scores:
            write_score(args.langfuse_url, headers, trace_id, "langfuse_smoke_pass", 1)
            write_score(args.langfuse_url, headers, trace_id, "langfuse_trace_present", 1)
            write_score(args.langfuse_url, headers, trace_id, "langfuse_required_observations", 1)
            write_score(args.langfuse_url, headers, trace_id, "sse_done", 1)
            if case.get("expected_route_primary"):
                write_score(args.langfuse_url, headers, trace_id, "route_primary_match", 1)
            if case.get("expected_task_intent_any"):
                write_score(args.langfuse_url, headers, trace_id, "task_intent_match", 1)
            if case.get("expected_turn_type_any"):
                write_score(args.langfuse_url, headers, trace_id, "turn_type_match", 1)
            for observation in as_list(case.get("required_observations")):
                write_score(args.langfuse_url, headers, trace_id, f"observation_{observation}_present", 1)

        dataset_item_id = sync_case_dataset_item(args.langfuse_url, headers, dataset_name, case)
        create_dataset_run_item(
            args.langfuse_url,
            headers,
            run_name,
            dataset_item_id,
            trace_id,
            run_description=args.run_description or payload.get("description", ""),
            metadata={
                "case_id": case["id"],
                "session_id": session_id,
                "dataset_path": str(dataset_file.relative_to(repo_root())),
            },
        )

        results.append(
            {
                "id": case["id"],
                "session_id": session_id,
                "trace_id": trace_id,
                "dataset_item_id": dataset_item_id,
                **verdict,
            }
        )

    summary = {
        "dataset": dataset_name,
        "run_name": run_name,
        "registration_mode": "dataset_run_item",
        "uses_experiments_api": False,
        "results": results,
    }

    if args.report_path:
        report_path = pathlib.Path(args.report_path)
        report_path.parent.mkdir(parents=True, exist_ok=True)
        report_path.write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")

    print(json.dumps(summary, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
