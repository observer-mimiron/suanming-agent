from __future__ import annotations

import argparse
import json
import pathlib
import re
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

    def __init__(
        self,
        message: str,
        session_id: str = "",
        trace_id: str = "",
        response_text: str = "",
        quality_violations: list[str] | None = None,
    ):
        super().__init__(message)
        self.session_id = session_id
        self.trace_id = trace_id
        self.response_text = response_text
        self.quality_violations = quality_violations or []


class ResponseQualityFailure(RuntimeError):
    """Report deterministic answer-quality violations as eval failures."""

    def __init__(self, violations: list[str]):
        super().__init__("response quality check failed: " + "; ".join(violations))
        self.violations = violations


def parse_csv_list(values: list[str]) -> list[str]:
    items: list[str] = []
    for value in values:
        for part in str(value).split(","):
            part = part.strip()
            if part:
                items.append(part)
    return items


def parse_expected_any_values(value: Any) -> list[str]:
    """Normalize one trace-attribute allowed-value declaration."""
    if isinstance(value, list):
        return [str(item).strip() for item in value if str(item).strip()]
    if value is None:
        return []
    return parse_csv_list([str(value)])


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


def decode_sse_text_data(raw: str) -> str:
    """Decode one text SSE data payload while accepting plain text chunks."""
    raw = raw.strip()
    if not raw:
        return ""
    try:
        payload = json.loads(raw)
    except json.JSONDecodeError:
        return raw
    if isinstance(payload, str):
        return payload
    if isinstance(payload, dict):
        for key in ("text", "content", "delta", "message"):
            value = payload.get(key)
            if isinstance(value, str):
                return value
        nested = payload.get("payload")
        if isinstance(nested, dict):
            for key in ("text", "content", "delta", "message"):
                value = nested.get(key)
                if isinstance(value, str):
                    return value
    return ""


def extract_response_text(chat_content: str) -> str:
    """Extract user-visible text from an SSE response, falling back to raw body."""
    event_name = ""
    data_lines: list[str] = []
    chunks: list[str] = []
    saw_sse = False

    def flush_event() -> None:
        nonlocal event_name, data_lines
        if event_name == "text" and data_lines:
            chunks.append(decode_sse_text_data("\n".join(data_lines)))
        event_name = ""
        data_lines = []

    for line in chat_content.splitlines():
        stripped = line.strip()
        if stripped == "":
            flush_event()
            continue
        if stripped.startswith("event:"):
            saw_sse = True
            flush_event()
            event_name = stripped.split(":", 1)[1].strip()
            continue
        if stripped.startswith("data:"):
            saw_sse = True
            data_lines.append(stripped.split(":", 1)[1].lstrip())
    flush_event()

    text = "".join(chunks).strip()
    if text:
        return text
    if saw_sse:
        fallback_lines = [
            line
            for line in chat_content.splitlines()
            if line.strip() and not line.strip().startswith(("event:", "data:"))
        ]
        return "\n".join(fallback_lines).strip()
    return chat_content.strip()


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
    if "response quality check failed" in message:
        return "response_quality"
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
    if (
        "http" in message
        or "trace not found" in message
        or "service.name mismatch" in message
        or "connection refused" in message
        or "urlopen error" in message
    ):
        return "transport"
    return "unknown"


def count_occurrences(text: str, phrase: str) -> int:
    """Count non-overlapping occurrences of a configured phrase."""
    phrase = str(phrase).strip()
    if not phrase:
        return 0
    return text.count(phrase)


def first_conclusion_in_section(text: str, heading: str) -> str:
    """Return the first bold conclusion in one markdown section."""
    section = markdown_section(text, heading)
    match = re.search(r"\*\*结论：(.+?)\*\*", section, flags=re.S)
    if not match:
        return ""
    return re.sub(r"\s+", " ", match.group(1)).strip()


def markdown_section(text: str, heading: str) -> str:
    """Return one markdown section body by its level-two heading."""
    section_start = text.find(f"## {heading}")
    if section_start < 0:
        return text
    next_start = text.find("\n## ", section_start + len(heading) + 3)
    return text[section_start:] if next_start < 0 else text[section_start:next_start]


def count_section_line_prefix(text: str, section: str, prefix: str) -> int:
    """Count lines with a prefix inside one markdown section."""
    section_text = markdown_section(text, section)
    return sum(1 for line in section_text.splitlines() if line.startswith(prefix))


def validate_response_quality(case: dict[str, Any], response_text: str) -> list[str]:
    """Run deterministic answer-quality checks declared by one dataset case."""
    checks = case.get("response_quality_checks") or {}
    violations: list[str] = []

    forbidden_terms = list(case.get("response_must_not_contain_any") or [])
    forbidden_terms.extend(checks.get("must_not_contain_any") or [])
    for forbidden in forbidden_terms:
        forbidden = str(forbidden).strip()
        if forbidden and forbidden in response_text:
            violations.append(f"quality forbidden content: {forbidden}")

    for phrase, max_allowed_raw in (checks.get("max_phrase_occurrences") or {}).items():
        phrase = str(phrase)
        max_allowed = int(max_allowed_raw)
        actual = count_occurrences(response_text, phrase)
        if actual > max_allowed:
            violations.append(f"phrase {phrase!r} occurs {actual} times > {max_allowed}")

    grouped_checks = checks.get("max_total_phrase_occurrences") or []
    if isinstance(grouped_checks, dict):
        grouped_checks = [grouped_checks]
    for grouped in grouped_checks:
        phrases = [str(item) for item in grouped.get("phrases") or []]
        max_allowed = int(grouped.get("max", 0))
        label = str(grouped.get("label") or "grouped phrases")
        total = sum(count_occurrences(response_text, phrase) for phrase in phrases)
        if total > max_allowed:
            violations.append(f"{label} occurs {total} times > {max_allowed}")

    if "max_overview_conclusion_semicolons" in checks:
        max_allowed = int(checks["max_overview_conclusion_semicolons"])
        conclusion = first_conclusion_in_section(response_text, "总览结论")
        actual = conclusion.count("；") + conclusion.count(";")
        if actual > max_allowed:
            violations.append(f"overview conclusion has {actual} semicolons > {max_allowed}")

    if "max_overview_conclusion_chars" in checks:
        max_allowed = int(checks["max_overview_conclusion_chars"])
        conclusion = first_conclusion_in_section(response_text, "总览结论")
        actual = len(conclusion)
        if actual > max_allowed:
            violations.append(f"overview conclusion has {actual} chars > {max_allowed}")

    section_heading_checks = checks.get("max_heading_occurrences_in_section") or []
    if isinstance(section_heading_checks, dict):
        section_heading_checks = [section_heading_checks]
    for check in section_heading_checks:
        section = str(check.get("section") or "").strip()
        prefix = str(check.get("heading_prefix") or "").strip()
        max_allowed = int(check.get("max", 0))
        if not section or not prefix:
            continue
        actual = count_section_line_prefix(response_text, section, prefix)
        if actual > max_allowed:
            violations.append(f"{section} section has {actual} {prefix!r} headings > {max_allowed}")

    return violations


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
    include_response: bool = False,
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
    response_text = extract_response_text(chat_content)
    quality_violations: list[str] = []
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
        if response_must_contain and response_must_contain not in response_text:
            raise RuntimeError(f"response missing expected content: {response_must_contain}")
        for expected in case.get("response_must_contain_all") or []:
            expected = str(expected).strip()
            if expected and expected not in response_text:
                raise RuntimeError(f"response missing expected content: {expected}")
        for forbidden in case.get("response_must_not_contain") or []:
            forbidden = str(forbidden).strip()
            if forbidden and forbidden in response_text:
                raise RuntimeError(f"response contains forbidden content: {forbidden}")
        quality_violations = validate_response_quality(case, response_text)
        if quality_violations:
            raise ResponseQualityFailure(quality_violations)

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

        expected_trace_attribute_any = case.get("expected_trace_attribute_any") or {}
        for key, allowed_raw in expected_trace_attribute_any.items():
            allowed = parse_expected_any_values(allowed_raw)
            actual = get_trace_field(trace_detail, str(key))
            if allowed and str(actual) not in allowed:
                raise RuntimeError(f"trace attribute mismatch for {key}: {actual!r} not in {allowed!r}")

        if write_scores:
            if not score_config_ids:
                raise RuntimeError("missing resolved Langfuse ScoreConfig ids")
            write_case_scores(langfuse_url, headers, trace_id, score_config_ids, passed=True)

        result = {
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
            }
            | {
                str(key): get_trace_field(trace_detail, str(key))
                for key in expected_trace_attribute_any
            },
            "quality_violations": [],
        }
        if include_response or case.get("include_response") or case.get("store_response"):
            result["response_text"] = response_text
        return result
    except Exception as exc:
        if write_scores and score_config_ids:
            write_case_scores(langfuse_url, headers, trace_id, score_config_ids, passed=False, error=exc)
        if isinstance(exc, ResponseQualityFailure):
            quality_violations = exc.violations
        raise SmokeCaseFailure(
            str(exc),
            session_id=session_id,
            trace_id=trace_id,
            response_text=response_text,
            quality_violations=quality_violations,
        ) from exc


def failed_case_result(case_id: str, error: Exception, include_response: bool = False) -> dict[str, Any]:
    """Build a report row for failures, preserving trace context when available."""
    result = {"id": case_id, "passed": False, "error": str(error)}
    if isinstance(error, SmokeCaseFailure):
        if error.session_id:
            result["session_id"] = error.session_id
        if error.trace_id:
            result["trace_id"] = error.trace_id
            result["failure_class"] = classify_failure(error)
        if error.quality_violations:
            result["quality_violations"] = error.quality_violations
        if include_response and error.response_text:
            result["response_text"] = error.response_text
    return result


def failure_class_counts(results: list[dict[str, Any]]) -> dict[str, int]:
    """Aggregate stable failure classes for repeated-case reports."""
    counts: dict[str, int] = {}
    for result in results:
        if result.get("passed"):
            continue
        failure_class = str(result.get("failure_class") or classify_failure(RuntimeError(str(result.get("error") or ""))))
        counts[failure_class] = counts.get(failure_class, 0) + 1
    return counts


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
    parser.add_argument("--repeats", type=int, default=1)
    parser.add_argument("--include-response", action="store_true", help="Store extracted assistant text in the JSON report")
    args = parser.parse_args()
    if args.repeats <= 0:
        raise SystemExit("--repeats must be positive")

    payload = load_dataset(args.dataset_path)
    env_map = load_env_map()
    headers = langfuse_headers(env_map)
    score_config_ids = require_score_config_ids(args.langfuse_url, headers, EVAL_SCORE_CONFIG_TYPES) if args.write_scores else None

    results: list[dict[str, Any]] = []
    for case in payload["cases"]:
        for repeat in range(1, args.repeats + 1):
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
                    include_response=args.include_response,
                )
                result["repeat"] = repeat
                results.append(result)
            except Exception as exc:  # noqa: BLE001
                result = failed_case_result(str(case["id"]), exc, include_response=args.include_response)
                result["repeat"] = repeat
                results.append(result)

    passed = sum(1 for item in results if item["passed"])
    failed = sum(1 for item in results if not item["passed"])
    total = passed + failed
    summary = {
        "dataset": str(payload["name"]),
        "repeats": args.repeats,
        "passed": passed,
        "failed": failed,
        "pass_rate": 0 if total == 0 else passed / total,
        "failure_classes": failure_class_counts(results),
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
