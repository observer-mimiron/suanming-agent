import argparse
import pathlib

from langfuse_eval_common import api_request, langfuse_headers, load_env_map, repo_root


def load_pricing_file(pricing_path: pathlib.Path):
    import json

    payload = json.loads(pricing_path.read_text(encoding="utf-8"))
    models = payload.get("models") or []
    if not isinstance(models, list) or not models:
        raise RuntimeError(f"no models defined in pricing file: {pricing_path}")
    return models


def list_existing_models(base_url, headers):
    page = 1
    all_items = []
    while True:
        payload = api_request(base_url, "GET", f"/api/public/models?limit=100&page={page}", headers)
        items = payload.get("data", [])
        all_items.extend(items)
        meta = payload.get("meta") or {}
        total_pages = meta.get("totalPages") or 1
        if page >= total_pages:
            break
        page += 1
    return all_items


def index_existing_models(items):
    by_name = {}
    for item in items:
        name = item.get("modelName")
        if name:
            by_name[name] = item
    return by_name


def build_model_payload(item):
    model_name = item["model_name"]
    match_pattern = item.get("match_pattern") or f"(?i)^({model_name})$"
    tokenizer_id = item.get("tokenizer_id")
    unit = item.get("unit") or "TOKENS"
    input_price = item.get("input_price")
    output_price = item.get("output_price")

    payload = {
        "modelName": model_name,
        "matchPattern": match_pattern,
        "unit": unit,
        "inputPrice": input_price,
        "outputPrice": output_price,
    }
    if tokenizer_id:
        payload["tokenizerId"] = tokenizer_id
    return payload


def sync_model(base_url, headers, existing, item):
    payload = build_model_payload(item)
    current = existing.get(payload["modelName"])
    if current is None:
        created = api_request(base_url, "POST", "/api/public/models", headers, payload)
        return "created", created

    changed = False
    comparisons = {
        "matchPattern": payload["matchPattern"],
        "tokenizerId": payload.get("tokenizerId"),
        "unit": payload["unit"],
        "inputPrice": payload["inputPrice"],
        "outputPrice": payload["outputPrice"],
    }
    for key, expected in comparisons.items():
        current_value = current.get(key)
        if current_value != expected:
            changed = True
            break
    if not changed:
        return "unchanged", current

    updated = api_request(base_url, "PUT", f"/api/public/models/{current['id']}", headers, payload)
    return "updated", updated


def main():
    parser = argparse.ArgumentParser(description="Sync project model pricing definitions into Langfuse Models API.")
    parser.add_argument("--langfuse-url", default="http://localhost:3001")
    parser.add_argument(
        "--pricing-file",
        default=str(repo_root() / "eval" / "model-pricing" / "langfuse-model-pricing.json"),
    )
    parser.add_argument("--env-file", default=str(repo_root() / "backend" / ".env"))
    args = parser.parse_args()

    env_map = load_env_map(args.env_file)
    headers = langfuse_headers(env_map)
    pricing_path = pathlib.Path(args.pricing_file)
    models = load_pricing_file(pricing_path)
    existing = index_existing_models(list_existing_models(args.langfuse_url, headers))

    results = []
    for item in models:
        status, synced = sync_model(args.langfuse_url, headers, existing, item)
        results.append(
            {
                "model_name": item["model_name"],
                "status": status,
                "langfuse_model_id": synced.get("id"),
                "input_price": synced.get("inputPrice"),
                "output_price": synced.get("outputPrice"),
            }
        )

    import json

    print(json.dumps({"results": results}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
