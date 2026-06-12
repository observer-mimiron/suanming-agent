#!/usr/bin/env python3
"""命理大师回归测试执行器 — 支持 v2 丰富格式 + v1 兼容"""

import json, sys, os, re, subprocess, argparse

GREEN = '\033[0;32m'; RED = '\033[0;31m'; YELLOW = '\033[1;33m'; CYAN = '\033[0;36m'; NC = '\033[0m'
def log_info(msg): print(f"{CYAN}[INFO]{NC} {msg}")
def log_pass(msg): print(f"{GREEN}[PASS]{NC} {msg}")
def log_fail(msg): print(f"{RED}[FAIL]{NC} {msg}")
def log_warn(msg): print(f"{YELLOW}[WARN]{NC} {msg}")

def run_turn(server_url, message, session_id, timeout=120):
    payload = json.dumps({"message": message, "session_id": session_id}, ensure_ascii=False)
    try:
        result = subprocess.run(
            ["curl", "-s", "-w", "\n%{http_code}", "-X", "POST",
             f"{server_url}/api/chat", "-H", "Content-Type: application/json",
             "-d", payload, "--max-time", str(timeout)],
            capture_output=True, text=True, timeout=timeout + 5)
        raw = result.stdout
        lines = raw.rsplit('\n', 1)
        body, http_code = (lines[0], lines[1].strip()) if len(lines) == 2 and lines[1].strip().isdigit() else (raw, "000")
        turn_type = ""
        m = re.search(r'"turn_type":"(\w+)"', body)
        if m: turn_type = m.group(1)
        thinking = ""
        m = re.search(r'"agent":"orchestrator","text":"([^"]+)"', body)
        if m: thinking = m.group(1)
        full_text = ""
        for m in re.finditer(r'"content":"((?:[^"\\]|\\.)*)"', body):
            full_text += m.group(1)
        return http_code, body, turn_type, thinking, full_text
    except Exception as e:
        return "000", str(e), "", "", ""

def check_expect(http_code, body, turn_type, thinking, full_text, expect):
    errors = []
    search_in = full_text + " " + body

    if expect.get("http_status") and str(http_code) != str(expect["http_status"]):
        errors.append(f"HTTP {http_code} (expected {expect['http_status']})")
    if expect.get("turn_type") and turn_type != expect["turn_type"]:
        errors.append(f"turn_type={turn_type} (expected {expect['turn_type']})")
    if expect.get("turn_type_any") and turn_type not in expect["turn_type_any"]:
        errors.append(f"turn_type={turn_type} (expected one of {expect['turn_type_any']})")
    if expect.get("reuse_chart") and "复用已有命盘" not in body:
        errors.append("reuse_chart: 未复用已有命盘")
    if expect.get("knowledge_search") and "knowledge_search" not in body and "knowledge-sources" not in body:
        errors.append("knowledge_search: 未触发知识库检索")
    for kw in expect.get("contains_any", []):
        if not re.search(kw, search_in):
            errors.append(f"contains_any 未命中: {kw}")
    for kw in expect.get("contains_all", []):
        if not re.search(kw, search_in):
            errors.append(f"contains_all 缺失: {kw}")
    for kw in expect.get("not_contains", []):
        if re.search(kw, search_in):
            errors.append(f"not_contains 违规出现: {kw}")
    return errors

def grade_answer(full_text, body, grading):
    """Grade quiz answer against rubric. Returns (passed, details)."""
    search_in = full_text + " " + body
    issues = []

    # Check pass conditions
    for kw in grading.get("pass", []):
        if re.search(kw, search_in):
            return True, []

    # Check forbidden patterns
    for kw in grading.get("forbidden", []):
        if re.search(kw, search_in):
            issues.append(f"违规词: {kw}")

    # Check fail patterns
    for kw in grading.get("fail", []):
        if re.search(kw, search_in):
            issues.append(f"强度不足: {kw}")

    if not issues:
        issues.append(f"未命中 pass 关键词: {grading.get('pass', [])}")
    return False, issues

def normalize_case(case):
    """Normalize v1/v2 format to internal representation."""
    # v2 format has "setup.turns"
    if "setup" in case and "turns" in case["setup"]:
        return case["setup"]["turns"]
    # v1 format has top-level "turns"
    return case.get("turns", [])

def run_case(server_url, case):
    case_id = case["id"]
    desc = case.get("desc", "")
    difficulty = case.get("difficulty", "")
    answer = case.get("answer", {})
    grading = case.get("grading", {})
    signals = case.get("signals", {})
    meta = case.get("meta", {})

    # Normalize answer: v1 string → v2 dict
    if isinstance(answer, str):
        answer = {"text": answer}
    answer_text = answer.get("text", "")
    distractor_hint = ", ".join(answer.get("distractors", []))

    turns = normalize_case(case)

    print(f"\n{CYAN}━━━ {case_id} ━━━{NC}")
    print(f"  {desc}")
    if difficulty:
        print(f"  难度: {difficulty}")
    if answer_text:
        info = f"答案: {answer_text}"
        if answer.get("letter"): info = f"答案: {answer['letter']}. {answer_text}"
        print(f"  {info}")
    if signals.get("primary"):
        print(f"  命理信号: {signals['primary']}")

    case_errors = 0
    quiz_grade_ok = True
    quiz_grade_details = []

    for i, turn in enumerate(turns):
        msg = turn.get("message", "")
        sid = turn.get("session_id", f"{case_id}-t{i}")
        expect = turn.get("expect", {})
        short = msg[:50] + "..." if len(msg) > 50 else msg
        label = f"T{i+1} \"{short}\""

        http_code, body, turn_type, thinking, full_text = run_turn(server_url, msg, sid)

        # HTTP status short-circuit
        if expect.get("http_status") and str(http_code) == str(expect["http_status"]):
            log_pass(f"{label} → HTTP {http_code}")
            continue

        errs = check_expect(http_code, body, turn_type, thinking, full_text, expect)
        if errs:
            for e in errs:
                log_fail(f"{label} → {e}")
            case_errors += len(errs)
        else:
            info = thinking[:50] if thinking else (turn_type or http_code)
            log_pass(f"{label} → {info}")

        # On the LAST turn of a quiz case, grade the answer
        if i == len(turns) - 1 and grading:
            ok, details = grade_answer(full_text, body, grading)
            quiz_grade_ok = ok
            quiz_grade_details = details
            if ok:
                log_pass(f"  📝 答案评分: 通过")
            else:
                log_fail(f"  📝 答案评分: 未通过 — {'; '.join(details)}")
                # Show rubric hints
                if grading.get("rubric"):
                    log_info(f"  rubric: {grading['rubric'][:120]}...")
                case_errors += 1

    if case_errors == 0:
        log_pass(f"{case_id}: 全部通过")
        return True
    else:
        log_fail(f"{case_id}: {case_errors} 项失败")
        if meta.get("easy_mistake"):
            log_info(f"  常见误判: {meta['easy_mistake']}")
        return False

def load_suite(suite_file):
    cases, name, desc = [], os.path.splitext(os.path.basename(suite_file))[0], ""
    with open(suite_file) as f:
        first_line = f.readline().strip(); f.seek(0)
        try:
            obj = json.loads(first_line)
            if isinstance(obj, dict) and "id" in obj and ("turns" in obj or "setup" in obj):
                for line in f:
                    line = line.strip()
                    if line: cases.append(json.loads(line))
                return name, desc, cases
        except json.JSONDecodeError: pass
        data = json.load(f)
        if isinstance(data, list): cases = data
        elif isinstance(data, dict):
            name = data.get("name", name); desc = data.get("description", "")
            cases = data.get("cases", [])
    return name, desc, cases

def run_suite(suite_file, server_url):
    name, desc, cases = load_suite(suite_file)
    print("=" * 60)
    print(f"  {CYAN}{name}{NC}")
    if desc: print(f"  {desc}")
    print(f"  Server: {server_url}  |  {len(cases)} cases")
    print("=" * 60)
    passed = sum(1 for c in cases if run_case(server_url, c))
    failed = len(cases) - passed
    print(f"\n{'=' * 60}")
    print(f"  结果: {GREEN}{passed} 通过{NC} / {RED}{failed} 失败{NC} / {len(cases)} 总计")
    print("=" * 60)
    return failed == 0

def clean_sessions():
    """清理持久化 session 文件，避免多轮测试数据污染。"""
    project_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    sessions_dir = os.path.join(project_root, "data", "sessions")
    if os.path.isdir(sessions_dir):
        count = 0
        for f in os.listdir(sessions_dir):
            path = os.path.join(sessions_dir, f)
            if os.path.isfile(path):
                os.remove(path)
                count += 1
        if count > 0:
            log_info(f"清理了 {count} 个旧 session 文件")

def main():
    parser = argparse.ArgumentParser(description="命理大师回归测试执行器")
    parser.add_argument("suite", nargs="?", help="套件文件路径")
    parser.add_argument("server", nargs="?", default="http://localhost:8080", help="后端地址")
    parser.add_argument("--all", action="store_true", help="跑全部套件（仅发布前使用）")
    parser.add_argument("--list", action="store_true", help="列出套件")
    parser.add_argument("--no-clean", action="store_true", help="不清理旧 session 文件")
    args = parser.parse_args()

    if not args.no_clean:
        clean_sessions()

    suite_dir = os.path.dirname(os.path.abspath(__file__))
    if args.list:
        print("可用测试套件:")
        for f in sorted(os.listdir(suite_dir)):
            if f.endswith(".jsonl"):
                name, _, cases = load_suite(os.path.join(suite_dir, f))
                print(f"  {name}: {len(cases)} cases")
        return
    if args.all:
        ok = True
        for f in sorted(os.listdir(suite_dir)):
            if f.endswith(".jsonl"):
                if not run_suite(os.path.join(suite_dir, f), args.server): ok = False
        sys.exit(0 if ok else 1)
    if not args.suite:
        parser.print_help(); sys.exit(1)
    sys.exit(0 if run_suite(args.suite, args.server) else 1)

if __name__ == "__main__":
    main()
