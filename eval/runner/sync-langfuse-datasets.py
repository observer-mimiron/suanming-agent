import argparse
import json
import pathlib

from langfuse_eval_common import ensure_dataset, langfuse_headers, load_env_map, repo_root, upsert_dataset_item


def build_item_payload(case):
    input_payload = {"message": case.get("message", "")}
    if case.get("setup_message"):
        input_payload["setup_message"] = case["setup_message"]

    expected_output = {
        "expected_route_primary": case.get("expected_route_primary", ""),
        "expected_task_intent_any": case.get("expected_task_intent_any", []),
        "expected_turn_type_any": case.get("expected_turn_type_any", []),
        "required_observations": case.get("required_observations", []),
    }
    metadata = {
        "case_id": case.get("id", ""),
        "source": "local-eval-json",
    }
    return input_payload, expected_output, metadata


def sync_dataset(base_url, headers, dataset_path):
    dataset_file = pathlib.Path(dataset_path).resolve()
    payload = json.loads(dataset_file.read_text(encoding="utf-8"))
    dataset_name = payload["name"]
    dataset = ensure_dataset(
        base_url,
        headers,
        dataset_name,
        payload.get("description", ""),
        {"source_path": str(dataset_file.relative_to(repo_root()))},
    )

    synced_items = []
    for case in payload.get("cases", []):
        item_id = f"{dataset_name}:{case['id']}"
        input_payload, expected_output, metadata = build_item_payload(case)
        item = upsert_dataset_item(
            base_url,
            headers,
            dataset_name,
            item_id,
            input_payload,
            expected_output,
            metadata,
        )
        synced_items.append({"case_id": case["id"], "dataset_item_id": item["id"]})

    return {
        "dataset": dataset_name,
        "dataset_id": dataset.get("id"),
        "items_synced": len(synced_items),
        "items": synced_items,
    }


def main():
    parser = argparse.ArgumentParser(description="Sync local eval datasets into Langfuse hosted datasets.")
    parser.add_argument("--langfuse-url", default="http://localhost:3001")
    parser.add_argument("--env-file", default=str(repo_root() / "backend" / ".env"))
    parser.add_argument("--dataset-path", action="append", help="Single dataset JSON path. Can be repeated.")
    args = parser.parse_args()

    env_map = load_env_map(args.env_file)
    headers = langfuse_headers(env_map)

    dataset_paths = args.dataset_path
    if not dataset_paths:
        dataset_paths = [str(p) for p in sorted((repo_root() / "eval" / "datasets").glob("*.json"))]

    results = [sync_dataset(args.langfuse_url, headers, path) for path in dataset_paths]
    print(json.dumps({"datasets": results}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
