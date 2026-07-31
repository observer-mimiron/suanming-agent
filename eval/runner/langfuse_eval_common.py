import base64
import json
import pathlib
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid


def repo_root():
    return pathlib.Path(__file__).resolve().parents[2]


def load_env_map(env_path=None):
    env_path = pathlib.Path(env_path) if env_path else repo_root() / "backend" / ".env"
    data = {}
    if not env_path.exists():
        raise FileNotFoundError(f"missing backend env file: {env_path}")
    for raw in env_path.read_text(encoding="utf-8").splitlines():
        raw = raw.strip()
        if not raw or raw.startswith("#") or "=" not in raw:
            continue
        key, value = raw.split("=", 1)
        key = key.strip()
        value = value.strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in ("'", '"'):
            value = value[1:-1]
        data[key] = value
    return data


def parse_header_map(raw):
    headers = {}
    if not raw:
        return headers
    for item in raw.split(","):
        if "=" not in item:
            continue
        key, value = item.split("=", 1)
        key = key.strip()
        value = value.strip()
        if key and value:
            headers[key] = value
    return headers


def langfuse_headers(env_map):
    headers = parse_header_map(env_map.get("OTEL_EXPORTER_OTLP_HEADERS", ""))
    if "Authorization" not in headers:
        public_key = env_map.get("LANGFUSE_PUBLIC_KEY", "")
        secret_key = env_map.get("LANGFUSE_SECRET_KEY", "")
        if public_key and secret_key:
            token = base64.b64encode(f"{public_key}:{secret_key}".encode("utf-8")).decode("ascii")
            headers["Authorization"] = f"Basic {token}"
    if "Authorization" not in headers:
        raise RuntimeError("missing Langfuse Authorization header in backend/.env")
    return headers


def api_request(base_url, method, path, headers, body=None, timeout=30):
    url = base_url.rstrip("/") + path
    payload = None
    request_headers = dict(headers)
    if body is not None:
        payload = json.dumps(body, ensure_ascii=False).encode("utf-8")
        request_headers["Content-Type"] = "application/json; charset=utf-8"
    req = urllib.request.Request(url, data=payload, method=method, headers=request_headers)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read()
            if not raw:
                return None
            return json.loads(raw.decode("utf-8"))
    except urllib.error.HTTPError as exc:
        body_text = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{method} {path} -> HTTP {exc.code}: {body_text}") from exc


def backend_request(base_url, body, timeout=120):
    url = base_url.rstrip("/") + "/api/chat"
    payload = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=payload,
        method="POST",
        headers={"Content-Type": "application/json; charset=utf-8"},
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.status, resp.read().decode("utf-8", errors="replace")


def get_trace_field(trace, key):
    if trace is None:
        return ""
    if key in trace and trace[key] not in (None, ""):
        return trace[key]
    metadata = trace.get("metadata") or {}
    attributes = metadata.get("attributes") or {}
    if key in attributes and attributes[key] not in (None, ""):
        return attributes[key]
    resource_attributes = metadata.get("resourceAttributes") or {}
    if key in resource_attributes and resource_attributes[key] not in (None, ""):
        return resource_attributes[key]
    return ""


def get_route_primary(trace):
    value = get_trace_field(trace, "primary_domain")
    if value:
        return value
    return get_trace_field(trace, "approved_route.primary_domain")


def list_traces_by_session(base_url, headers, expected_session_id, limit=20, excluded_trace_ids=None):
    excluded_trace_ids = set(excluded_trace_ids or [])
    payload = api_request(base_url, "GET", f"/api/public/traces?limit={limit}", headers)
    for item in payload.get("data", []):
        session_id = item.get("sessionId") or get_trace_field(item, "session_id") or get_trace_field(item, "langfuse.session.id")
        if session_id == expected_session_id and item.get("id") not in excluded_trace_ids:
            return item
    return None


def get_trace_detail(base_url, headers, trace_id):
    return api_request(base_url, "GET", f"/api/public/traces/{trace_id}", headers)


def get_observation_names(trace_detail):
    names = []
    for item in trace_detail.get("observations", []):
        name = item.get("name")
        if name and name not in names:
            names.append(name)
    return names


def unique_session_id(prefix):
    return f"{prefix}-{uuid.uuid4().hex}"


def ensure_dataset(base_url, headers, dataset_name, description, metadata=None):
    encoded_name = urllib.parse.quote(dataset_name, safe="")
    try:
        dataset = api_request(base_url, "GET", f"/api/public/v2/datasets/{encoded_name}", headers)
    except RuntimeError as exc:
        if "HTTP 404" not in str(exc):
            raise
        dataset = api_request(
            base_url,
            "POST",
            "/api/public/v2/datasets",
            headers,
            {
                "name": dataset_name,
                "description": description,
                "metadata": metadata or {},
            },
        )
    return dataset


def upsert_dataset_item(base_url, headers, dataset_name, item_id, input_payload, expected_output, metadata=None):
    return api_request(
        base_url,
        "POST",
        "/api/public/dataset-items",
        headers,
        {
            "id": item_id,
            "datasetName": dataset_name,
            "input": input_payload,
            "expectedOutput": expected_output,
            "metadata": metadata or {},
        },
    )


def create_dataset_run_item(base_url, headers, run_name, dataset_item_id, trace_id, run_description="", metadata=None):
    return api_request(
        base_url,
        "POST",
        "/api/public/dataset-run-items",
        headers,
        {
            "runName": run_name,
            "runDescription": run_description,
            "datasetItemId": dataset_item_id,
            "traceId": trace_id,
            "metadata": metadata or {},
        },
    )


def write_score(base_url, headers, trace_id, name, value, comment=""):
    return api_request(
        base_url,
        "POST",
        "/api/public/scores",
        headers,
        {
            "traceId": trace_id,
            "name": name,
            "value": value,
            "comment": comment,
        },
    )


def poll_trace_detail(base_url, headers, session_id, timeout_seconds=120, poll_interval_seconds=3, max_limit=20, excluded_trace_ids=None):
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        summary = list_traces_by_session(base_url, headers, session_id, max_limit, excluded_trace_ids)
        if summary:
            trace_id = summary["id"]
            detail = get_trace_detail(base_url, headers, trace_id)
            return trace_id, detail
        time.sleep(poll_interval_seconds)
    raise RuntimeError(f"langfuse trace not found for session_id={session_id}")
