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
        "reuse_chart": true,                // 检查"复用已有命盘"
        "knowledge_search": true            // 检查 knowledge_search 触发
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

import json, sys, os, re, subprocess, argparse, time

GREEN  = "\033[0;32m"
RED    = "\033[0;31m"
YELLOW = "\033[1;33m"
CYAN   = "\033[0;36m"
NC     = "\033[0m"


def log_info(msg):
    print(f"{CYAN}[INFO]{NC} {msg}")


def log_pass(msg):
    print(f"{GREEN}[PASS]{NC} {msg}")


def log_fail(msg):
    print(f"{RED}[FAIL]{NC} {msg}")


def log_warn(msg):
    print(f"{YELLOW}[WARN]{NC} {msg}")


def run_turn(server_url, message, session_id, timeout=30):
    """向 /api/chat 发一次请求，返回 (http_code, body, turn_type, thinking, full_text)。"""
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
        # trace-panel component 事件中: "turn_type":"..."
        m = re.search(r'"turn_type"\s*:\s*"(\w+)"', body)
        if m:
            turn_type = m.group(1)

        thinking = ""
        # thinking 事件: event: thinking, data: {"agent":"orchestrator","text":"..."}
        for m in re.finditer(r'data:\s*\{"agent"\s*:\s*"(?:orchestrator|planner)","text"\s*:\s*"((?:[^"\\]|\\.)*)"', body):
            thinking = m.group(1)

        full_text = ""
        # SSE text 事件: data: {"content":"..."}
        for m in re.finditer(r'"content"\s*:\s*"((?:[^"\\]|\\.)*)"', body):
            full_text += m.group(1)

        return http_code, body, turn_type, thinking, full_text

    except Exception as e:
        return "000", str(e), "", "", ""


def check_expect(http_code, body, turn_type, full_text, expect):
    """检验 expect 规则，返回错误列表。"""
    errors = []
    search_in = full_text + " " + body

    if expect.get("http_status") and str(http_code) != str(expect["http_status"]):
        errors.append(f"HTTP {http_code} (expected {expect['http_status']})")

    if expect.get("turn_type") and turn_type != expect["turn_type"]:
        errors.append(f"turn_type={turn_type} (expected {expect['turn_type']})")

    if expect.get("turn_type_any") and turn_type not in expect["turn_type_any"]:
        errors.append(f"turn_type={turn_type} (expected one of {expect['turn_type_any']})")

    if expect.get("knowledge_search") and not ("knowledge_search" in body and "knowledge-sources" in body):
        errors.append("knowledge_search: 未触发知识库检索")

    for kw in expect.get("contains_any", []):
        if re.search(kw, search_in):
            break
    else:
        if expect.get("contains_any"):
            # find which ones were tried
            errors.append(f"contains_any 未命中: {expect['contains_any']}")

    for kw in expect.get("contains_all", []):
        if not re.search(kw, search_in):
            errors.append(f"contains_all 缺失: {kw}")

    # 兼容两种拼写: not_contains / not_contain
    for kw in expect.get("not_contains", expect.get("not_contain", [])):
        if re.search(kw, search_in):
            errors.append(f"not_contains 违规出现: {kw}")

    return errors


def grade_answer(full_text, body, grading):
    """评估答案质量。返回 (passed, issues)。"""
    search_in = full_text + " " + body
    issues = []

    # 命中 pass 关键词 → 通过
    for kw in grading.get("pass", []):
        if re.search(kw, search_in):
            return True, []

    # 检查 forbidden
    for kw in grading.get("forbidden", []):
        if re.search(kw, search_in):
            issues.append(f"违规词: {kw}")

    # 检查 fail (弱信号)
    for kw in grading.get("fail", []):
        if re.search(kw, search_in):
            issues.append(f"强度不足: {kw}")

    if not issues:
        issues.append(f"未命中 pass 关键词: {grading.get('pass', [])}")

    return False, issues


def normalize_case(case):
    """标准化 v1/v2 格式。返回 turns 列表。"""
    if "setup" in case and "turns" in case["setup"]:
        return case["setup"]["turns"]
    return case.get("turns", [])


def load_suite(suite_file):
    """加载 JSONL suite 文件。返回 (name, description, cases)。"""
    name = os.path.splitext(os.path.basename(suite_file))[0]
    desc = ""
    cases = []

    with open(suite_file) as f:
        first_line = f.readline().strip()

        try:
            obj = json.loads(first_line)
            if isinstance(obj, dict) and "id" in obj and ("turns" in obj or "setup" in obj):
                # 纯 case 列表（无 header 行），从第一行开始
                f.seek(0)
                for line in f:
                    line = line.strip()
                    if line:
                        cases.append(json.loads(line))
            elif isinstance(obj, dict) and "name" in obj:
                # 有 header 的 JSONL：第一行是 suite 元信息，后续行为 case
                name = obj["name"]
                desc = obj.get("description", "")
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
                cases = data.get("cases", [])

    return name, desc, cases


def run_case(server_url, case, timeout=30, delay=1):
    """跑一个 case 的全部 turn。返回 (passed, details)。"""
    turns = normalize_case(case)
    details = []

    if not turns:
        return True, [{"turn": 0, "passed": True, "info": "empty case"}]

    all_passed = True
    for i, turn in enumerate(turns):
        msg = turn.get("message", "")
        sid = turn.get("session_id", "test-session")
        expect = turn.get("expect", {})
        grading = turn.get("grading", None)

        http_code, body, turn_type, thinking, full_text = run_turn(
            server_url, msg, sid, timeout
        )

        errors = check_expect(http_code, body, turn_type, full_text, expect)

        # grading
        grading_passed, grading_issues = True, []
        if grading:
            grading_passed, grading_issues = grade_answer(full_text, body, grading)

        passed = len(errors) == 0 and grading_passed

        detail = {
            "turn": i + 1,
            "message": msg[:80],
            "http_code": http_code,
            "turn_type": turn_type,
            "passed": passed,
            "errors": errors,
            "grading_issues": grading_issues,
            "thinking": thinking[:120] if thinking else "",
            "full_text": full_text[:200] if full_text else "",
        }
        details.append(detail)

        if not passed:
            all_passed = False

        time.sleep(delay)

    return all_passed, details


def run_suite(server_url, suite_file, timeout=30, delay=1):
    """跑一个 suite 的所有 case。"""
    name, desc, cases = load_suite(suite_file)

    print(f"\n{'='*60}")
    print(f"Suite: {name}")
    if desc:
        print(f"Description: {desc}")
    print(f"Cases: {len(cases)}")
    print(f"Target: {server_url}")
    print(f"{'='*60}\n")

    passed_count = 0
    fail_count = 0

    for case in cases:
        case_id = case.get("id", "?")
        log_info(f"Case: {case_id}")
        case_passed, details = run_case(server_url, case, timeout, delay)

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


def main():
    parser = argparse.ArgumentParser(description="命理大师回归测试执行器")
    parser.add_argument("suite", help="Suite JSONL 文件路径")
    parser.add_argument("server", help="服务地址，如 http://localhost:18080")
    parser.add_argument("--timeout", type=int, default=60, help="单轮请求超时秒数 (default: 60)")
    parser.add_argument("--delay", type=float, default=1.0, help="轮间延迟秒数 (default: 1.0)")
    args = parser.parse_args()

    ok = run_suite(args.server, args.suite, args.timeout, args.delay)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
