from __future__ import annotations

import argparse
import json
import pathlib
import sys
from typing import Any

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from langfuse_eval_common import (  # noqa: E402
    langfuse_headers,
    list_scores_v3,
    load_env_map,
    require_score_config_ids,
    write_score,
)

ANSWER_SCORE_CONFIG_TYPES = {
    "answer_task_complete": "BOOLEAN",
    "answer_factuality_pass": "BOOLEAN",
    "answer_grounding_pass": "BOOLEAN",
    "answer_scope_safe": "BOOLEAN",
    "answer_failure_class": "CATEGORICAL",
}

ANSWER_FAILURE_CLASSES = {
    "none",
    "task",
    "factuality",
    "grounding",
    "scope",
    "style",
    "insufficient_evidence",
}


def load_annotation(path: pathlib.Path) -> dict[str, Any]:
    """Load one human annotation file and reject missing case arrays early."""
    payload = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(payload.get("cases"), list) or not payload["cases"]:
        raise RuntimeError(f"annotation file has no cases: {path}")
    return payload


def score_items(case: dict[str, Any]) -> list[tuple[str, Any, str]]:
    """Return validated non-empty score values for one annotated case."""
    scores = case.get("scores") or {}
    items: list[tuple[str, Any, str]] = []
    for name, data_type in ANSWER_SCORE_CONFIG_TYPES.items():
        value = scores.get(name)
        if value is None:
            continue
        if data_type == "BOOLEAN" and not isinstance(value, bool):
            raise RuntimeError(f"{case.get('id', '')}: {name} must be true/false or null")
        if data_type == "CATEGORICAL" and value not in ANSWER_FAILURE_CLASSES:
            allowed = ", ".join(sorted(ANSWER_FAILURE_CLASSES))
            raise RuntimeError(f"{case.get('id', '')}: {name}={value!r} not in [{allowed}]")
        items.append((name, value, data_type))
    return items


def score_api_value(value: Any, data_type: str) -> Any:
    """Convert validated annotation values to Langfuse v3 score API values."""
    if data_type == "BOOLEAN":
        return 1 if value else 0
    return value


def existing_score_names(
    langfuse_url: str,
    headers: dict[str, str],
    trace_id: str,
    names: list[str],
) -> set[str]:
    """Return score names already present on a trace so repeated runs stay idempotent."""
    scores = list_scores_v3(langfuse_url, headers, trace_ids=[trace_id], names=names)
    return {str(score.get("name")) for score in scores}


def write_annotation_scores(
    annotation: dict[str, Any],
    langfuse_url: str,
    headers: dict[str, str],
    score_config_ids: dict[str, str],
    dry_run: bool,
    allow_duplicates: bool = False,
) -> dict[str, Any]:
    """Write completed human labels to Langfuse scores, or report the dry-run plan."""
    results: list[dict[str, Any]] = []
    for case in annotation["cases"]:
        trace_id = str(case.get("trace_id") or "").strip()
        if not trace_id:
            raise RuntimeError(f"{case.get('id', '')}: missing trace_id")
        items = score_items(case)
        case_result = {
            "id": case.get("id", ""),
            "trace_id": trace_id,
            "scores_ready": len(items),
            "written": 0,
            "skipped_existing": 0,
        }
        existing_names = set()
        if items and not allow_duplicates:
            existing_names = existing_score_names(
                langfuse_url,
                headers,
                trace_id,
                [name for name, _, _ in items],
            )
        comment = str(case.get("human_note") or "").strip()
        for name, value, data_type in items:
            if name in existing_names:
                case_result["skipped_existing"] += 1
                continue
            if not dry_run:
                write_score(
                    langfuse_url,
                    headers,
                    trace_id,
                    name,
                    score_api_value(value, data_type),
                    comment=comment,
                    config_id=score_config_ids[name],
                    data_type=data_type,
                )
                case_result["written"] += 1
        results.append(case_result)
    return {
        "annotation": annotation.get("name", ""),
        "dry_run": dry_run,
        "cases": len(results),
        "scores_ready": sum(item["scores_ready"] for item in results),
        "scores_written": sum(item["written"] for item in results),
        "scores_skipped_existing": sum(item["skipped_existing"] for item in results),
        "results": results,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Write human answer-quality annotations to Langfuse scores")
    parser.add_argument("--annotation-path", default="eval/annotation/answer-quality-human-v1.json")
    parser.add_argument("--langfuse-url", default="http://localhost:3001")
    parser.add_argument("--write-scores", action="store_true")
    parser.add_argument("--allow-duplicates", action="store_true")
    args = parser.parse_args()

    annotation = load_annotation(pathlib.Path(args.annotation_path))
    headers = langfuse_headers(load_env_map())
    score_config_ids = require_score_config_ids(args.langfuse_url, headers, ANSWER_SCORE_CONFIG_TYPES)
    summary = write_annotation_scores(
        annotation,
        args.langfuse_url,
        headers,
        score_config_ids,
        dry_run=not args.write_scores,
        allow_duplicates=args.allow_duplicates,
    )
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
