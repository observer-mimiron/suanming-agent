from __future__ import annotations

import importlib.util
import json
import pathlib
import sys
from typing import Any

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from langfuse_eval_common import api_request, langfuse_headers, list_score_configs, load_env_map  # noqa: E402
from run_answer_quality_judge import infer_deepseek_flash_judge_config  # noqa: E402
from run_langfuse_eval import EVAL_SCORE_CONFIG_TYPES  # noqa: E402
from write_human_answer_scores import ANSWER_SCORE_CONFIG_TYPES  # noqa: E402


def check_score_configs(base_url: str, headers: dict[str, str]) -> dict[str, Any]:
    """Summarize required score configs without printing secrets."""
    expected = {**EVAL_SCORE_CONFIG_TYPES, **ANSWER_SCORE_CONFIG_TYPES}
    configs = {item.get("name"): item for item in list_score_configs(base_url, headers)}
    rows = {}
    for name, expected_type in expected.items():
        config = configs.get(name)
        rows[name] = {
            "exists": bool(config),
            "expected_type": expected_type,
            "actual_type": config.get("dataType") if config else "",
            "ok": bool(config) and config.get("dataType") == expected_type,
        }
    return rows


def check_llm_connections(base_url: str, headers: dict[str, str]) -> dict[str, Any]:
    """Check whether Langfuse platform-hosted evaluators can call a model."""
    try:
        payload = api_request(base_url, "GET", "/api/public/llm-connections", headers)
    except Exception as exc:  # noqa: BLE001
        return {"supported": False, "count": 0, "error": str(exc)}
    data = payload.get("data") if isinstance(payload, dict) else payload
    data = data or []
    safe_rows = []
    for item in data[:10]:
        safe_rows.append(
            {
                key: value
                for key, value in item.items()
                if "key" not in str(key).lower() and "secret" not in str(key).lower()
            }
        )
    return {"supported": True, "count": len(data), "connections": safe_rows}


def main() -> int:
    env_map = load_env_map()
    headers = langfuse_headers(env_map)
    langfuse_url = env_map.get("LANGFUSE_HOST") or env_map.get("LANGFUSE_BASE_URL") or "http://localhost:3001"
    judge_config = infer_deepseek_flash_judge_config(env_map)
    judge_ready = all(judge_config.get(name) for name in ("api_key", "base_url", "model"))
    summary = {
        "langfuse_url": langfuse_url,
        "python_sdk_installed": importlib.util.find_spec("langfuse") is not None,
        "score_configs": check_score_configs(langfuse_url, headers),
        "llm_connections": check_llm_connections(langfuse_url, headers),
        "local_judge_env": {
            "ready": judge_ready,
            "required": ["EVAL_JUDGE_API_KEY", "EVAL_JUDGE_BASE_URL", "EVAL_JUDGE_MODEL"],
            "source": judge_config.get("source", ""),
            "base_url": judge_config.get("base_url", ""),
            "model": judge_config.get("model", ""),
        },
    }
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
