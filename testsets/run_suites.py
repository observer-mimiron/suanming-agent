#!/usr/bin/env python3
"""
╔══════════════════════════════════════════════════════════════════╗
║  DEPRECATED — 此文件已废弃，请使用统一入口                        ║
║  python3 testsets/suites/runner.py <suite.jsonl> <server_url>    ║
║  详见 testsets/README.md                                        ║
╚══════════════════════════════════════════════════════════════════╝

Run smoke + standard test suites against a running server at localhost:18080.
(保留仅作为旧脚本参考，不保证与新 runner 行为一致)
Each suite is a JSONL file with a header row (name/description) followed by test cases.

SSE format from /api/chat:
  event: text
  data: {"content":"..."}
  ...
  event: component
  data: {"payload":...,"type":"trace-panel"}
  event: done
  data: {}
"""

import json
import os
import subprocess
import sys
import time

BASE_URL = "http://localhost:18080"
SUITES_DIR = "/Users/wikiglobal/workSapce/suanming-agent/testsets/suites"
SUITE_FILES = [
    "flow-basic.jsonl",
    "quiz-marriage.jsonl",
    "quiz-career-wealth.jsonl",
    "edge-input.jsonl",
]

GREEN = "\033[0;32m"
RED = "\033[0;31m"
NC = "\033[0m"


def parse_sse(sse_text: str) -> str:
    """
    Parse SSE text and accumulate all 'content' fields from 'event: text' blocks.
    Returns the concatenated full text.
    """
    lines = sse_text.strip().split("\n")
    content_parts = []
    i = 0
    while i < len(lines):
        line = lines[i].strip()
        if line == "event: text":
            # Next line should be data: {...}
            i += 1
            if i < len(lines):
                data_line = lines[i].strip()
                if data_line.startswith("data: "):
                    data_str = data_line[len("data: "):]
                    try:
                        data = json.loads(data_str)
                        if "content" in data:
                            content_parts.append(data["content"])
                    except json.JSONDecodeError:
                        pass
        i += 1
    return "".join(content_parts)


def call_chat(session_id: str, message: str) -> tuple[int, str, str]:
    """Send a chat request, return (exit_code, raw_sse, accumulated_content)."""
    body = json.dumps({"session_id": session_id, "message": message}, ensure_ascii=False)
    cmd = [
        "curl", "-s", "-X", "POST",
        f"{BASE_URL}/api/chat",
        "-H", "Content-Type: application/json",
        "-d", body,
        "--max-time", "180",
    ]
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
        raw = result.stdout
        exit_code = result.returncode
        if exit_code != 0:
            return (exit_code, raw, "")

        full_text = parse_sse(raw)
        return (0, raw, full_text)
    except subprocess.TimeoutExpired:
        return (-1, "", "TIMEOUT")
    except Exception as e:
        return (-2, "", str(e))


def run_suite(filepath: str) -> tuple[int, int]:
    """Run a suite file. Returns (passed, failed)."""
    with open(filepath, "r", encoding="utf-8") as f:
        lines = [l.strip() for l in f if l.strip()]

    if not lines:
        return (0, 0)

    header = json.loads(lines[0])
    suite_name = header.get("name", "unknown")
    cases = [json.loads(l) for l in lines[1:]]

    print("")
    print("=" * 60)
    print(f" Suite: {suite_name} ({len(cases)} cases)")
    print("=" * 60)

    passed = 0
    failed = 0

    for case in cases:
        case_id = case.get("id", "unknown")
        turns = case.get("turns", [])
        print("")
        print(f"  --- {case_id} ({len(turns)} turns) ---")

        overall_pass = True
        last_full_text = ""
        failures = []

        for idx, turn in enumerate(turns):
            msg = turn["message"]
            sess_id = turn["session_id"]
            expect = turn.get("expect", {})
            has_assertions = bool(expect)

            if idx > 0:
                time.sleep(1.5)

            msg_preview = msg[:80] + "..." if len(msg) > 80 else msg
            print(f"    Turn {idx+1}: session={sess_id}, msg={msg_preview}")

            # Call API
            exit_code, raw_sse, full_text = call_chat(sess_id, msg)

            if exit_code != 0:
                print(f"      CURL ERROR (code={exit_code})")
                overall_pass = False
                failures.append(f"curl error (exit={exit_code})")
                continue

            last_full_text = full_text
            resp_preview = full_text[:120] + "..." if len(full_text) > 120 else full_text
            print(f"      response ({len(full_text)} chars): {resp_preview}")

            # Only check assertions when expect block exists
            if not has_assertions:
                continue

            # Check contains_any
            contains_any = expect.get("contains_any", [])
            if contains_any:
                found = False
                for kw in contains_any:
                    if kw.lower() in full_text.lower():
                        found = True
                        break
                if not found:
                    print(f"      [FAIL] contains_any: none of {contains_any} found in response")
                    overall_pass = False
                    failures.append(f"contains_any: none of {contains_any} found")
                else:
                    print(f"      [OK] contains_any matched")

            # Check not_contains
            not_contains = expect.get("not_contains", [])
            if not_contains:
                for kw in not_contains:
                    if kw.lower() in full_text.lower():
                        print(f"      [FAIL] not_contains: '{kw}' found in response")
                        overall_pass = False
                        failures.append(f"not_contains: '{kw}' found in response")
                    else:
                        print(f"      [OK] not_contains: '{kw}' not found")

        if overall_pass:
            print(f"    {GREEN}[PASS]{NC} {case_id}")
            passed += 1
        else:
            pref = last_full_text[:100] + "..." if len(last_full_text) > 100 else last_full_text
            print(f"    {RED}[FAIL]{NC} {case_id}")
            for f in failures:
                print(f"         reason: {f}")

            failed += 1

        time.sleep(1.5)

    print("")
    print(f"  Suite: {suite_name}  {passed}/{passed + failed} passed")
    return (passed, failed)


def main():
    total_passed = 0
    total_failed = 0
    total_cases = 0

    for suite_file in SUITE_FILES:
        filepath = os.path.join(SUITES_DIR, suite_file)
        if not os.path.exists(filepath):
            print(f"WARNING: suite file not found: {filepath}")
            continue
        p, f = run_suite(filepath)
        total_passed += p
        total_failed += f
        total_cases += p + f

    print("")
    print("=" * 60)
    print(" FINAL TOTALS")
    print("=" * 60)
    if total_failed == 0:
        print(f"{GREEN}TOTAL: {total_passed}/{total_cases} passed (100%){NC}")
    else:
        pct = total_passed * 100 // total_cases if total_cases > 0 else 0
        print(f"{RED}TOTAL: {total_passed}/{total_cases} passed ({pct}%){NC}")
    print("")

    return 0 if total_failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
