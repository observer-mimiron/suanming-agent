#!/usr/bin/env python3
"""命理大师回归测试执行器。

Suite 格式 (JSONL):
  第一行: {"name": "suite-name", "description": "..."}
  后续行: 每行一个 case JSON 对象

Case 格式:
{
  "id": "case_1",
  "turns": [
    {
      "message": "用户输入",
      "session_id": "test-session-1",
      "expect": {
        "http_status": 200,
        "turn_type": "direct_bazi",         // 精确匹配
        "turn_type_any": ["x", "y"],        // 任意一个
        "contains_any": ["关键词A", "B"],   // 至少命中一个
        "contains_all": ["必须包含A"],      // 全部命中
        "not_contains": ["禁止词"],         // 不可出现
        "final_output_contains": ["关键词"], // 只检查最终回答文本
        "final_output_not_contains": ["禁词"],
        "knowledge_search": true,           // 检查 knowledge_search 触发
        "conversation_intent_any": ["consult"],
        "route_primary": "bazi",            // 精确匹配路由主域
        "task_intent": "collect_profile",   // 精确匹配 task_intent
        "task_intent_any": ["x", "y"],      // 任意一个
        "qimen_mode": "primary",            // 精确匹配 qimen_mode
        "secondary_contains": ["ziwei"]     // secondary_domains 至少包含一个
      },
      "grading": {
        "pass": ["正确关键词"],
        "forbidden": ["不应该出现的词"],
        "fail": ["弱信号关键词"]
      }
    }
  ]
}

使用:
  python3 runner.py testsets/suites/quiz-marriage.jsonl http://localhost:18080
"""

import json, sys, os, re, subprocess, argparse, time, threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass, field

# ── 新增：适配器 + 断言引擎导入 ──
import sys as _sys
from pathlib import Path as _Path
_ADAPTERS_DIR = str(_Path(__file__).resolve().parent.parent / "adapters")
_SUITES_DIR = str(_Path(__file__).resolve().parent)
if _ADAPTERS_DIR not in _sys.path:
    _sys.path.insert(0, _ADAPTERS_DIR)
if _SUITES_DIR not in _sys.path:
    _sys.path.insert(0, _SUITES_DIR)

from canonical import CanonicalTrace
from assertions import run_assertions, resolve_placeholders

GREEN  = "\033[0;32m"
RED    = "\033[0;31m"
YELLOW = "\033[1;33m"
CYAN   = "\033[0;36m"
BOLD   = "\033[1m"
NC     = "\033[0m"

_print_lock = threading.Lock()

def tprint(*args, **kwargs):
    """线程安全的 print。"""
    with _print_lock:
        print(*args, **kwargs)


def log_info(msg):
    print(f"{CYAN}[INFO]{NC} {msg}")


def log_pass(msg):
    print(f"{GREEN}[PASS]{NC} {msg}")


def log_fail(msg):
    print(f"{RED}[FAIL]{NC} {msg}")


def log_warn(msg):
    print(f"{YELLOW}[WARN]{NC} {msg}")


# ── 数据类型 ──────────────────────────────────────────────

@dataclass
class TurnResponse:
    """单轮 SSE 响应的解析结果。"""
    http_code: str
    body: str
    turn_type: str = ""
    thinking: str = ""
    full_text: str = ""
    conversation_intent: str = ""
    route_primary: str = ""
    task_intent: str = ""
    qimen_mode: str = ""
    secondary_domains: list = field(default_factory=list)
    trace: CanonicalTrace = field(default_factory=CanonicalTrace)


def _parse_route_fields(body: str):
    """从 SSE body 中解析 route-decision 事件字段。"""
    conversation_intent = ""
    route_primary = ""
    task_intent = ""
    qimen_mode = ""
    secondary_domains = []

    if '"type":"route-decision"' not in body:
        return conversation_intent, route_primary, task_intent, qimen_mode, secondary_domains

    m = re.search(r'"conversation_intent":"([^"]*)"', body)
    if m:
        conversation_intent = m.group(1)

    m = re.search(r'"primary_domain":"([^"]*)"', body)
    if m:
        route_primary = m.group(1)
    m = re.search(r'"task_intent":"([^"]*)"', body)
    if m:
        task_intent = m.group(1)
    m = re.search(r'"qimen_mode":"([^"]*)"', body)
    if m:
        qimen_mode = m.group(1)
    m = re.search(r'"secondary_domains":\[([^\]]*)\]', body)
    if m:
        secondary_domains = re.findall(r'"([^"]*)"', m.group(1))
    return conversation_intent, route_primary, task_intent, qimen_mode, secondary_domains


def run_turn(server_url, message, session_id, timeout=30, adapter=None) -> TurnResponse:
    """向 /api/chat 发一次请求，返回结构化的 TurnResponse。"""
    payload = json.dumps({"message": message, "session_id": session_id}, ensure_ascii=False)

    try:
        result = subprocess.run(
            ["curl", "-s", "-w", "\n%{http_code}", "-X", "POST",
             f"{server_url}/api/chat",
             "-H", "Content-Type: application/json",
             "-d", payload,
             "--max-time", str(timeout)],
            capture_output=True, text=True, timeout=timeout + 5
        )

        raw = result.stdout
        lines = raw.rsplit("\n", 1)
        if len(lines) == 2 and lines[1].strip().isdigit():
            body, http_code = lines[0], lines[1].strip()
        else:
            body, http_code = raw, "000"

        turn_type = ""
        m = re.search(r'"turn_type"\s*:\s*"(\w+)"', body)
        if m:
            turn_type = m.group(1)

        conversation_intent, route_primary, task_intent, qimen_mode, secondary_domains = _parse_route_fields(body)

        thinking = ""
        for m in re.finditer(r'data:\s*\{"agent"\s*:\s*"(?:orchestrator|planner)","text"\s*:\s*"((?:[^"\\]|\\.)*)"', body):
            thinking = m.group(1)

        full_text = ""
        for m in re.finditer(r'"content"\s*:\s*"((?:[^"\\]|\\.)*)"', body):
            full_text += m.group(1)

        r = TurnResponse(
            http_code=http_code,
            body=body,
            turn_type=turn_type,
            thinking=thinking,
            full_text=full_text,
            conversation_intent=conversation_intent,
            route_primary=route_primary,
            task_intent=task_intent,
            qimen_mode=qimen_mode,
            secondary_domains=secondary_domains,
        )

        # 用适配器构建 CanonicalTrace（新断言引擎数据源）
        if adapter is not None:
            r.trace = adapter.convert(body, http_code)

        return r

    except Exception as e:
        return TurnResponse(http_code="000", body=str(e))


def check_expect(r: TurnResponse, expect: dict, context: dict = None) -> list:
    """统一断言入口。有 trace 且含新断言键时走新引擎，否则回退旧逻辑。"""

    # 仅这些断言强依赖 CanonicalTrace。final_output_* 故意留在 legacy 路径，
    # 避免 trace 存在时吞掉 turn_type_any / conversation_intent_any 等 guidance 断言。
    trace_only_keys = {
        "action_called", "action_not_called", "action_sequence",
        "action_arg_match", "action_result_not_empty",
        "retrieval_happened", "retrieval_has_results", "retrieval_cited",
        "step_count_range", "no_errors",
    }

    # 如果 trace 有数据且期望中有 trace 专属断言，走新引擎
    if r.trace and r.trace.raw_body and (trace_only_keys & set(expect.keys())):
        return run_assertions(r.trace, expect, context)

    # ── 回退：旧断言逻辑（完全保留原有代码） ──
    errors = []
    search_in = r.full_text + " " + r.body
    final_output = r.full_text or ""

    if expect.get("http_status") and str(r.http_code) != str(expect["http_status"]):
        errors.append(f"HTTP {r.http_code} (expected {expect['http_status']})")

    if expect.get("turn_type") and r.turn_type != expect["turn_type"]:
        errors.append(f"turn_type={r.turn_type} (expected {expect['turn_type']})")

    if expect.get("turn_type_any") and r.turn_type not in expect["turn_type_any"]:
        errors.append(f"turn_type={r.turn_type or '(empty)'} (expected one of {expect['turn_type_any']})")

    want_ci = expect.get("conversation_intent_any", [])
    if want_ci and r.conversation_intent not in want_ci:
        errors.append(
            f"conversation_intent={r.conversation_intent or '(empty)'} "
            f"(expected one of {want_ci})"
        )

    if expect.get("route_primary") and r.route_primary != expect["route_primary"]:
        errors.append(f"route_primary={r.route_primary or '(empty)'} (expected {expect['route_primary']})")

    if expect.get("task_intent") and r.task_intent != expect["task_intent"]:
        errors.append(f"task_intent={r.task_intent or '(empty)'} (expected {expect['task_intent']})")

    want_ti = expect.get("task_intent_any", [])
    if want_ti and r.task_intent not in want_ti:
        errors.append(f"task_intent={r.task_intent or '(empty)'} (expected one of {want_ti})")

    if expect.get("qimen_mode") and r.qimen_mode != expect["qimen_mode"]:
        errors.append(f"qimen_mode={r.qimen_mode or '(empty)'} (expected {expect['qimen_mode']})")

    want_secondary = expect.get("secondary_contains", [])
    if want_secondary:
        found_sec = any(s in r.secondary_domains for s in want_secondary)
        if not found_sec:
            errors.append(f"secondary_contains 未命中: {want_secondary} (got {r.secondary_domains})")

    if expect.get("knowledge_search") and not ("knowledge_search" in r.body and "knowledge-sources" in r.body):
        errors.append("knowledge_search: 未触发知识库检索")

    want_final_any = expect.get("final_output_contains", [])
    if want_final_any:
        for kw in want_final_any:
            if re.search(kw, final_output):
                break
        else:
            preview = final_output[:80] or "(empty)"
            errors.append(
                f"final_output_contains 未命中: {want_final_any} "
                f"(final_output={preview})"
            )

    for kw in expect.get("final_output_not_contains", []):
        if re.search(kw, final_output):
            preview = final_output[:80] or "(empty)"
            errors.append(
                f"final_output_not_contains 违规出现: {kw} "
                f"(final_output={preview})"
            )

    for kw in expect.get("contains_any", []):
        if re.search(kw, search_in):
            break
    else:
        if expect.get("contains_any"):
            errors.append(f"contains_any 未命中: {expect['contains_any']}")

    for kw in expect.get("contains_all", []):
        if not re.search(kw, search_in):
            errors.append(f"contains_all 缺失: {kw}")

    for kw in expect.get("not_contains", expect.get("not_contain", [])):
        if re.search(kw, search_in):
            errors.append(f"not_contains 违规出现: {kw}")

    return errors


def _selfcheck_mixed_guidance_assertions():
    """验证 mixed expect 在 trace 存在时仍会检查 guidance 字段和 final_output_*。"""
    trace = CanonicalTrace(raw_body="trace-present", final_output="这里提到事业")

    ok = TurnResponse(
        http_code="200",
        body='data: {"type":"route-decision","conversation_intent":"consult","task_intent":"collect_profile"}',
        turn_type="clarification",
        full_text="这里提到事业",
        conversation_intent="consult",
        task_intent="collect_profile",
        trace=trace,
    )
    ok_errors = check_expect(ok, {
        "turn_type_any": ["clarification"],
        "conversation_intent_any": ["consult"],
        "task_intent_any": ["collect_profile"],
        "final_output_contains": ["事业"],
    })
    if ok_errors:
        raise AssertionError(f"mixed expect should pass, got {ok_errors}")

    bad = TurnResponse(
        http_code="200",
        body='data: {"type":"route-decision","conversation_intent":"consult","task_intent":"collect_profile"}',
        turn_type="agent_reading",
        full_text="这里提到事业",
        conversation_intent="consult",
        task_intent="fortune_followup",
        trace=trace,
    )
    bad_errors = check_expect(bad, {
        "turn_type_any": ["clarification"],
        "conversation_intent_any": ["consult"],
        "task_intent_any": ["collect_profile"],
        "final_output_contains": ["事业"],
    })
    if not any("turn_type=" in err for err in bad_errors):
        raise AssertionError(f"mixed expect lost turn_type_any failure: {bad_errors}")
    if not any("task_intent=" in err for err in bad_errors):
        raise AssertionError(f"mixed expect lost task_intent_any failure: {bad_errors}")


def grade_answer(r: TurnResponse, grading: dict):
    """评估答案质量。返回 (passed, issues)。"""
    search_in = r.full_text + " " + r.body
    issues = []

    for kw in grading.get("pass", []):
        if re.search(kw, search_in):
            return True, []

    for kw in grading.get("forbidden", []):
        if re.search(kw, search_in):
            issues.append(f"违规词: {kw}")

    for kw in grading.get("fail", []):
        if re.search(kw, search_in):
            issues.append(f"强度不足: {kw}")

    if not issues:
        issues.append(f"未命中 pass 关键词: {grading.get('pass', [])}")

    return False, issues


# ── Suite 加载 ────────────────────────────────────────────

def normalize_case(case):
    """标准化 v1/v2 格式。返回 turns 列表。"""
    if "setup" in case and "turns" in case["setup"]:
        return case["setup"]["turns"]
    return case.get("turns", [])


def load_suite(suite_file):
    """加载 JSONL suite 文件。返回 (name, description, cases, meta)。"""
    name = os.path.splitext(os.path.basename(suite_file))[0]
    desc = ""
    cases = []
    meta = {}

    with open(suite_file) as f:
        first_line = f.readline().strip()

        try:
            obj = json.loads(first_line)
            if isinstance(obj, dict) and "id" in obj and ("turns" in obj or "setup" in obj):
                f.seek(0)
                for line in f:
                    line = line.strip()
                    if line:
                        cases.append(json.loads(line))
            elif isinstance(obj, dict) and "name" in obj:
                name = obj["name"]
                desc = obj.get("description", "")
                meta = obj
                for line in f:
                    line = line.strip()
                    if line:
                        cases.append(json.loads(line))
            else:
                raise json.JSONDecodeError("not a case", first_line, 0)
        except json.JSONDecodeError:
            data = json.load(f)
            if isinstance(data, list):
                cases = data
            elif isinstance(data, dict):
                name = data.get("name", name)
                desc = data.get("description", "")
                meta = data
                cases = data.get("cases", [])

    return name, desc, cases, meta


# ── 服务校验 ──────────────────────────────────────────────

PROBE_MESSAGE = "你好"  # 用于 smoke check 的最短消息

# AI 探针门禁 —— 超过任一上限即终止，不进入 suite 执行。
# 上限设置宽松，只拦截明显异常的情况，不误杀正常测试。
PROBE_TIME_LIMIT = int(os.environ.get("PROBE_TIME_LIMIT", "120"))   # 秒，单轮 2 分钟足够任何 specialist
PROBE_SIZE_LIMIT = int(os.environ.get("PROBE_SIZE_LIMIT", "1024"))  # KB，1MB 远超正常 SSE 响应


def _fmt_bytes(n: int) -> str:
    """人类可读的字节数。"""
    if n < 1024:
        return f"{n}B"
    elif n < 1024 * 1024:
        return f"{n / 1024:.1f}KB"
    return f"{n / (1024 * 1024):.1f}MB"


def check_server_ready(server_url: str, timeout: int = 30) -> bool:
    """跑任何 suite 之前执行三层门禁校验：
    Tier 1 (HARD) — 服务启动：health 端点 + commit 匹配
    Tier 2 (HARD) — 探针门禁：响应时效与大小上限
    Tier 3 (SOFT) — 业务能力：route-decision 可解析性

    返回 True 表示服务就绪，False 表示校验失败。
    """
    # ── Tier 1: 服务启动校验 (HARD) ──
    try:
        health_resp = subprocess.run(
            ["curl", "-s", f"{server_url}/api/health"],
            capture_output=True, text=True, timeout=10)
        health = json.loads(health_resp.stdout)
        if health.get("status") != "ok":
            print(f"\n{RED}✗ Tier 1 服务启动校验失败: health status={health.get('status')}{NC}\n")
            return False
    except Exception as e:
        print(f"\n{RED}✗ Tier 1 服务启动校验失败: health 不可达: {e}{NC}\n")
        return False

    # commit 匹配
    server_commit = health.get("commit", "")
    if server_commit and server_commit != "unknown":
        try:
            local = subprocess.run(
                ["git", "rev-parse", "--short", "HEAD"],
                capture_output=True, text=True, timeout=5)
            local_commit = local.stdout.strip()
            if local_commit and server_commit != local_commit:
                print(f"\n{RED}✗ Tier 1 服务启动校验失败: commit 不匹配: server={server_commit} local={local_commit}{NC}\n")
                return False
        except Exception:
            pass  # 不在 git 仓库中时跳过

    print(f"{GREEN}✓{NC} 服务启动校验通过 (commit={server_commit or '?'})")

    # ── Tier 2: 探针门禁 (HARD) ──
    try:
        t0 = time.time()
        probe_r = run_turn(server_url, PROBE_MESSAGE, "smoke-check", timeout=timeout)
        elapsed = time.time() - t0
        body_size = len(probe_r.body.encode("utf-8"))

        if probe_r.http_code != "200":
            print(f"\n{RED}✗ Tier 2 探针门禁失败: HTTP {probe_r.http_code}{NC}\n")
            return False

        if elapsed > PROBE_TIME_LIMIT:
            print(f"\n{RED}✗ Tier 2 探针门禁失败: 超时 {elapsed:.0f}s > {PROBE_TIME_LIMIT}s{NC}\n")
            return False
        if body_size > PROBE_SIZE_LIMIT * 1024:
            print(f"\n{RED}✗ Tier 2 探针门禁失败: 响应过大 {_fmt_bytes(body_size)} > {_fmt_bytes(PROBE_SIZE_LIMIT * 1024)}{NC}\n")
            return False

        print(f"{GREEN}✓{NC} 探针门禁通过 (probe={elapsed:.1f}s, body={_fmt_bytes(body_size)})")
    except Exception as e:
        print(f"\n{RED}✗ Tier 2 探针门禁失败: {e}{NC}\n")
        return False

    # ── Tier 3: 业务能力探针 (SOFT) ──
    if '"type":"route-decision"' not in probe_r.body:
        print(f'{YELLOW}⚠ 业务能力探针: 响应中缺少 route-decision 事件，服务基础链路正常，继续执行{NC}')
    elif not probe_r.route_primary:
        print(f'{YELLOW}⚠ 业务能力探针: route_primary 为空（路由可能因 "你好" 过于简短而不稳定），服务基础链路正常，继续执行{NC}')
    else:
        print(f"{GREEN}✓{NC} 业务能力探针通过 (route_primary={probe_r.route_primary})")

    return True


# ── 执行 ──────────────────────────────────────────────────

def run_case(server_url, case, timeout=30, delay=1, adapter=None, suite_context=None):
    """跑一个 case 的全部 turn。返回 (passed, details)。"""
    turns = normalize_case(case)
    details = []
    # 合并 suite 级和 case 级 context，case 级覆盖 suite 级
    context = dict(suite_context or {})
    context.update(case.get("_context", {}))

    if not turns:
        return True, [{"turn": 0, "passed": True, "info": "empty case"}]

    all_passed = True
    for i, turn in enumerate(turns):
        msg = turn.get("message", "")
        sid = turn.get("session_id", "test-session")
        expect = turn.get("expect", {})
        grading = turn.get("grading", None)

        r = run_turn(server_url, msg, sid, timeout, adapter=adapter)

        errors = check_expect(r, expect, context)

        grading_passed, grading_issues = True, []
        if grading:
            grading_passed, grading_issues = grade_answer(r, grading)

        passed = len(errors) == 0 and grading_passed

        detail = {
            "turn": i + 1,
            "message": msg[:80],
            "http_code": r.http_code,
            "turn_type": r.turn_type,
            "passed": passed,
            "errors": errors,
            "grading_issues": grading_issues,
            "thinking": r.thinking[:120] if r.thinking else "",
            "full_text": r.full_text[:200] if r.full_text else "",
        }
        details.append(detail)

        if not passed:
            all_passed = False

        time.sleep(delay)

    return all_passed, details


def run_suite(server_url, suite_file, timeout=30, delay=1, adapter=None):
    """串行跑一个 suite 的所有 case。"""
    if not check_server_ready(server_url, timeout):
        sys.exit(2)
    name, desc, cases, meta = load_suite(suite_file)
    suite_context = meta.get("_context", {})

    total_turns = sum(len(normalize_case(c)) for c in cases)

    print(f"\n{'='*60}")
    print(f"Suite: {name}")
    if desc:
        print(f"Description: {desc}")
    print(f"Cases: {len(cases)}  Turns: {total_turns}")
    print(f"Target: {server_url}")
    print(f"Est. total: ~{total_turns * timeout + total_turns * delay:.0f}s ({total_turns} turns × {timeout}s timeout + {delay}s delay)")
    print(f"{'='*60}\n")

    passed_count = 0
    fail_count = 0

    for case in cases:
        case_id = case.get("id", "?")
        log_info(f"Case: {case_id}")
        case_passed, details = run_case(server_url, case, timeout, delay, adapter=adapter, suite_context=suite_context)

        for d in details:
            prefix = f"  T{d['turn']}"
            if d["passed"]:
                log_pass(f"{prefix} \"{d['message']}\" → turn_type: {d['turn_type']}")
            else:
                log_fail(f"{prefix} \"{d['message']}\" → turn_type: {d['turn_type']}")
                for err in d["errors"]:
                    log_fail(f"    ↳ {err}")
                for issue in d["grading_issues"]:
                    log_warn(f"    ↳ {issue}")

        if case_passed:
            passed_count += 1
        else:
            fail_count += 1

    print(f"\n---")
    print(f"Results: {GREEN}{passed_count} passed{NC}, {RED}{fail_count} failed{NC}")
    return fail_count == 0


def run_suite_parallel(server_url, suite_file, timeout=60, delay=1, workers=4, adapter=None):
    """并行跑一个 suite 的所有 case（case 间并行，case 内 turn 串行）。"""
    if not check_server_ready(server_url, timeout):
        sys.exit(2)
    name, desc, cases, meta = load_suite(suite_file)
    suite_context = meta.get("_context", {})

    total_turns = sum(len(normalize_case(c)) for c in cases)
    n_workers = min(workers, len(cases)) if cases else 1
    # 并行时，最坏情况是最大 case 的耗时（case 内 turn 串行）
    max_turns = max((len(normalize_case(c)) for c in cases), default=0)

    print(f"\n{'='*60}")
    print(f"Suite: {name}")
    if desc:
        print(f"Description: {desc}")
    print(f"Cases: {len(cases)}  Turns: {total_turns}  Workers: {n_workers}")
    print(f"Target: {server_url}")
    print(f"Est. total: ~{max_turns * timeout + max_turns * delay:.0f}s (parallel, {max_turns} max turns × {timeout}s timeout + {delay}s delay)")
    print(f"{'='*60}\n")

    results = {}

    with ThreadPoolExecutor(max_workers=n_workers) as executor:
        future_to_case = {
            executor.submit(run_case, server_url, case, timeout, delay, adapter=adapter, suite_context=suite_context): case
            for case in cases
        }

        for future in as_completed(future_to_case):
            case = future_to_case[future]
            case_id = case.get("id", "?")
            try:
                case_passed, details = future.result()
                results[case_id] = (case_passed, details)

                status = f"{GREEN}PASS{NC}" if case_passed else f"{RED}FAIL{NC}"
                tprint(f"[{status}] {case_id}")
                for d in details:
                    if not d["passed"]:
                        for err in d["errors"]:
                            tprint(f"    {RED}↳{NC} T{d['turn']}: {err}")
                        for issue in d.get("grading_issues", []):
                            tprint(f"    {YELLOW}↳{NC} T{d['turn']}: {issue}")
            except Exception as e:
                tprint(f"[{RED}ERROR{NC}] {case_id}: {e}")
                results[case_id] = (False, [{"turn": 0, "passed": False, "message": "", "errors": [str(e)]}])

    # 按原始顺序输出详情
    print(f"\n{'─'*60}")
    print(f"{BOLD}Details{NC}")
    print(f"{'─'*60}")
    for case in cases:
        case_id = case.get("id", "?")
        case_passed, details = results.get(case_id, (False, []))
        status = f"{GREEN}OK{NC}" if case_passed else f"{RED}FAIL{NC}"
        print(f"  [{status}] {case_id}")
        for d in details:
            if d.get("errors"):
                for err in d["errors"]:
                    print(f"      T{d['turn']}: {RED}{err}{NC}")
            if d.get("grading_issues"):
                for issue in d["grading_issues"]:
                    print(f"      T{d['turn']}: {YELLOW}{issue}{NC}")

    passed = sum(1 for p, _ in results.values() if p)
    failed = len(results) - passed
    print(f"\n---")
    print(f"Results: {GREEN}{passed} passed{NC}, {RED}{failed} failed{NC}")
    return failed == 0


def main():
    parser = argparse.ArgumentParser(description="命理大师回归测试执行器")
    parser.add_argument("suite", nargs="?", help="Suite JSONL 文件路径")
    parser.add_argument("server", nargs="?", help="服务地址，如 http://localhost:18080")
    parser.add_argument("--timeout", type=int, default=60, help="单轮请求超时秒数 (default: 60)")
    parser.add_argument("--delay", type=float, default=1.0, help="轮间延迟秒数 (default: 1.0)")
    parser.add_argument("-w", "--workers", type=int, default=4, help="并行 workers 数 (default: 4, 设为 1 即串行)")
    parser.add_argument("--adapter", help="适配器配置 YAML 路径 (启用新断言引擎)")
    parser.add_argument("--self-check", action="store_true", help="运行 mixed guidance 断言自检后退出")
    args = parser.parse_args()

    if args.self_check:
        _selfcheck_mixed_guidance_assertions()
        print(f"{GREEN}✓ mixed guidance assertions self-check passed{NC}")
        sys.exit(0)

    if not args.suite or not args.server:
        parser.error("需要提供 suite 和 server 参数，或使用 --self-check")

    if args.adapter:
        from base import TraceAdapter as _TraceAdapter
        adapter = _TraceAdapter.load(args.adapter)
    else:
        adapter = None

    if args.workers > 1:
        ok = run_suite_parallel(args.server, args.suite, args.timeout, args.delay, args.workers, adapter=adapter)
    else:
        ok = run_suite(args.server, args.suite, args.timeout, args.delay, adapter=adapter)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
