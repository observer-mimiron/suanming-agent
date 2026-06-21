"""断言引擎 — 对 CanonicalTrace 做确定性检查。

每个断言函数签名: (trace: CanonicalTrace, params) -> list[str]
返回空列表 = 通过，非空列表 = 错误信息集合。
"""

import re
import json
from canonical import CanonicalTrace


# ── HTTP ──

def assert_http_status(trace: CanonicalTrace, expected: int) -> list:
    actual = str(trace.meta.get("http_code", "000"))
    if actual != str(expected):
        return [f"http_status: {actual} (expected {expected})"]
    return []


# ── 路由 ──

def assert_route_primary(trace: CanonicalTrace, expected: str) -> list:
    actual = trace.route.get("primary", "")
    if actual != expected:
        return [f"route_primary: '{actual}' != '{expected}'"]
    return []

def assert_route_intent_any(trace: CanonicalTrace, options: list) -> list:
    actual = trace.route.get("intent", "")
    if actual not in options:
        return [f"route_intent_any: '{actual}' not in {options}"]
    return []


# ── 动作 ──

def assert_action_called(trace: CanonicalTrace, names: list) -> list:
    actual = trace.action_names()
    errors = []
    for name in names:
        if name not in actual:
            errors.append(f"action_called: '{name}' not called (got {actual})")
    return errors

def assert_action_not_called(trace: CanonicalTrace, names: list) -> list:
    actual = trace.action_names()
    errors = []
    for name in names:
        if name in actual:
            errors.append(f"action_not_called: '{name}' was called (should not be)")
    return errors

def assert_action_sequence(trace: CanonicalTrace, sequence: list) -> list:
    """验证 subsequence 顺序。"""
    actual = trace.action_names()
    idx = 0
    for name in actual:
        if idx < len(sequence) and name == sequence[idx]:
            idx += 1
    if idx < len(sequence):
        return [f"action_sequence: missing {sequence[idx:]} in {actual}"]
    return []

def assert_action_arg_match(trace: CanonicalTrace, rules: dict) -> list:
    """rules: {"action_name": {"arg_key": "expected_value"}}"""
    errors = []
    for action_name, expected_args in rules.items():
        a = trace.action_by_name(action_name)
        if a is None:
            errors.append(f"action_arg_match: '{action_name}' not found")
            continue
        args = a.get("arguments", {})
        if isinstance(args, str):
            try:
                args = json.loads(args)
            except json.JSONDecodeError:
                pass
        if isinstance(args, dict):
            for k, v in expected_args.items():
                if str(v) not in str(args.get(k, "")):
                    errors.append(
                        f"action_arg_match: {action_name}.{k} does not contain '{v}'"
                    )
        else:
            errors.append(f"action_arg_match: {action_name} args not a dict: {args}")
    return errors

def assert_action_result_not_empty(trace: CanonicalTrace, names: list) -> list:
    errors = []
    for name in names:
        a = trace.action_by_name(name)
        if a is None:
            errors.append(f"action_result_not_empty: '{name}' not found")
            continue
        result = a.get("result")
        if result is None or result == "" or result == {} or result == []:
            err = a.get("error", "")
            errors.append(f"action_result_not_empty: '{name}' empty (error={err})")
    return errors


# ── 检索 ──

def assert_retrieval_happened(trace: CanonicalTrace, _=True) -> list:
    if not trace.has_retrievals():
        return ["retrieval_happened: no retrieval recorded"]
    return []

def assert_retrieval_has_results(trace: CanonicalTrace, _=True) -> list:
    if trace.retrieval_source_count() == 0:
        return ["retrieval_has_results: triggered but returned 0 chunks"]
    return []

def assert_retrieval_cited(trace: CanonicalTrace, _=True) -> list:
    for r in trace.retrievals:
        for chunk in r.get("chunks", []):
            content = chunk.get("content", "")
            if len(content) > 15 and content[:30] in trace.final_output:
                return []
    return ["retrieval_cited: no chunk found in final_output"]


# ── 步骤 ──

def assert_step_count_range(trace: CanonicalTrace, bounds: list) -> list:
    n = trace.step_count()
    lo, hi = bounds[0], bounds[1]
    if n < lo or n > hi:
        return [f"step_count_range: {n} not in [{lo}, {hi}]"]
    return []


# ── 错误 ──

def assert_no_errors(trace: CanonicalTrace, _=True) -> list:
    if trace.has_errors():
        errs = trace.meta.get("errors", [])
        msgs = [str(e)[:80] for e in errs[:3]]
        return [f"no_errors: {len(errs)} errors: {msgs}"]
    return []


# ── 最终输出 ──

def assert_final_output_contains(trace: CanonicalTrace, keywords: list) -> list:
    for kw in keywords:
        if re.search(kw, trace.final_output):
            return []
    return [f"final_output_contains: none of {keywords} found"]

def assert_final_output_not_contains(trace: CanonicalTrace, keywords: list) -> list:
    errors = []
    for kw in keywords:
        if re.search(kw, trace.final_output):
            errors.append(f"final_output_not_contains: '{kw}' found")
    return errors


# ── 旧格式兼容 (contains_any / contains_all / not_contains) ──

def assert_contains_any(trace: CanonicalTrace, keywords: list) -> list:
    """至少命中一个关键词 (substring match on final_output + raw_body)."""
    search_in = trace.final_output + " " + trace.raw_body
    for kw in keywords:
        if re.search(kw, search_in):
            return []
    return [f"contains_any 未命中: {keywords}"]


def assert_contains_all(trace: CanonicalTrace, keywords: list) -> list:
    """全部关键词命中."""
    search_in = trace.final_output + " " + trace.raw_body
    errors = []
    for kw in keywords:
        if not re.search(kw, search_in):
            errors.append(f"contains_all 缺失: {kw}")
    return errors


def assert_not_contains(trace: CanonicalTrace, keywords: list) -> list:
    """关键词全部不出现."""
    search_in = trace.final_output + " " + trace.raw_body
    errors = []
    for kw in keywords:
        if re.search(kw, search_in):
            errors.append(f"not_contains 违规出现: {kw}")
    return errors


# ── 注册表 ──

ASSERTION_REGISTRY = {
    "http_status":             (assert_http_status, "single"),
    "route_primary":           (assert_route_primary, "single"),
    "route_intent_any":        (assert_route_intent_any, "list"),
    "action_called":           (assert_action_called, "list"),
    "action_not_called":       (assert_action_not_called, "list"),
    "action_sequence":         (assert_action_sequence, "list"),
    "action_arg_match":        (assert_action_arg_match, "dict"),
    "action_result_not_empty": (assert_action_result_not_empty, "list"),
    "retrieval_happened":      (assert_retrieval_happened, "bool"),
    "retrieval_has_results":   (assert_retrieval_has_results, "bool"),
    "retrieval_cited":         (assert_retrieval_cited, "bool"),
    "step_count_range":        (assert_step_count_range, "list"),
    "no_errors":               (assert_no_errors, "bool"),
    "final_output_contains":   (assert_final_output_contains, "list"),
    "final_output_not_contains": (assert_final_output_not_contains, "list"),
    "contains_any":             (assert_contains_any, "list"),
    "contains_all":             (assert_contains_all, "list"),
    "not_contains":             (assert_not_contains, "list"),
}


def resolve_placeholders(obj, context: dict):
    """递归展开 {var} → context[var]"""
    if context is None:
        return obj
    if isinstance(obj, str):
        for k, v in context.items():
            obj = obj.replace(f"{{{k}}}", str(v))
        return obj
    elif isinstance(obj, list):
        return [resolve_placeholders(item, context) for item in obj]
    elif isinstance(obj, dict):
        return {k: resolve_placeholders(v, context) for k, v in obj.items()}
    return obj


def run_assertions(trace: CanonicalTrace, expect: dict, context: dict = None) -> list:
    """对 trace 执行 expect 中的所有断言。"""
    if context:
        expect = resolve_placeholders(expect, context)
    errors = []
    for key, params in expect.items():
        if key not in ASSERTION_REGISTRY:
            continue
        fn, _ = ASSERTION_REGISTRY[key]
        result = fn(trace, params)
        errors.extend(result)
    return errors
