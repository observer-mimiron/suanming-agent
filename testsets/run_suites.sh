#!/usr/bin/env bash
set -o pipefail

BASE_URL="http://localhost:18080"
SUITES_DIR="/Users/wikiglobal/workSapce/suanming-agent/testsets/suites"

TOTAL_PASSED=0
TOTAL_FAILED=0
TOTAL_CASES=0

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

call_chat() {
    local session_id="$1"
    local message="$2"
    local tmpfile="$3"

    local body
    body=$(printf '{"session_id":"%s","message":"%s"}' "$(echo "$session_id" | sed 's/"/\\"/g')" "$(echo "$message" | sed 's/"/\\"/g')")

    curl -s -X POST "$BASE_URL/api/chat" \
        -H "Content-Type: application/json" \
        -d "$body" \
        --max-time 60 \
        -o "$tmpfile" 2>/dev/null

    local curl_exit=$?
    if [ $curl_exit -ne 0 ]; then
        echo "CURL_ERROR:$curl_exit"
        return 1
    fi

    # Extract full_text from SSE content events
    full_text=$(grep -o '"full_text":"[^"]*"' "$tmpfile" 2>/dev/null | head -1 | sed 's/"full_text":"//;s/"$//')
    if [ -z "$full_text" ]; then
        # Try to extract from last "content":"..." event
        full_text=$(grep -o '"content":"[^"]*"' "$tmpfile" 2>/dev/null | tail -1 | sed 's/"content":"//;s/"$//')
    fi

    echo "$full_text"
    return 0
}

process_suite() {
    local suite_file="$1"
    local suite_name
    suite_name=$(head -1 "$suite_file" | python3 -c "import sys,json; print(json.load(sys.stdin).get('name','unknown'))" 2>/dev/null)

    echo ""
    echo "=========================================="
    echo " Suite: $suite_name"
    echo "=========================================="

    local suite_passed=0
    local suite_failed=0
    local suite_total=0

    # Read all lines except the first (header)
    local line_num=1
    while IFS= read -r line; do
        line_num=$((line_num + 1))
        # skip empty lines and header line
        [ -z "$line" ] && continue
        if [ "$line_num" -eq 1 ]; then continue; fi

        # Parse case
        local case_id
        case_id=$(echo "$line" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id','unknown'))" 2>/dev/null)
        local turns_json
        turns_json=$(echo "$line" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin).get('turns',[])))" 2>/dev/null)

        local num_turns
        num_turns=$(echo "$turns_json" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null)

        suite_total=$((suite_total + 1))
        TOTAL_CASES=$((TOTAL_CASES + 1))

        echo ""
        echo "--- Running: $case_id ($num_turns turns) ---"

        local overall_pass=true
        local last_full_text=""

        for ((turn_idx=0; turn_idx<num_turns; turn_idx++)); do
            local msg
            msg=$(echo "$turns_json" | python3 -c "
import sys, json
turns = json.load(sys.stdin)
t = turns[$turn_idx]
print(t.get('message',''))
" 2>/dev/null)

            local sess_id
            sess_id=$(echo "$turns_json" | python3 -c "
import sys, json
turns = json.load(sys.stdin)
t = turns[$turn_idx]
print(t.get('session_id',''))
" 2>/dev/null)

            # Check if last turn has assertions
            local has_assertions
            has_assertions=$(echo "$turns_json" | python3 -c "
import sys, json
turns = json.load(sys.stdin)
t = turns[$turn_idx]
e = t.get('expect', {})
if e:
    print('yes')
else:
    print('no')
" 2>/dev/null)

            echo "  Turn $((turn_idx+1)): session=$sess_id"
            echo "    message: ${msg:0:80}..."

            # Add delay between turns (not first turn of case)
            if [ "$turn_idx" -gt 0 ]; then
                sleep 1.5
            fi

            tmpfile=$(mktemp)
            result=$(call_chat "$sess_id" "$msg" "$tmpfile")

            if [[ "$result" == CURL_ERROR:* ]]; then
                echo "    CURL ERROR: ${result#CURL_ERROR:}"
                overall_pass=false
                rm -f "$tmpfile"
                continue
            fi

            # Save last full_text for assertion check
            last_full_text="$result"

            http_status=$(head -1 "$tmpfile" 2>/dev/null | python3 -c "
import sys
line = sys.stdin.readline().strip()
if 'HTTP' in line:
    parts = line.split()
    if len(parts) >= 2:
        print(parts[1])
" 2>/dev/null)
            [ -z "$http_status" ] && http_status="200"

            # Extract content from SSE lines for preview
            content_preview=$(grep -o '"content":"[^"]*"[^}]*}' "$tmpfile" 2>/dev/null | tail -1 | python3 -c "
import sys, json
try:
    line = sys.stdin.readline().strip()
    if line.endswith('}}'):
        data = json.loads('{' + line + '}')
    else:
        data = json.loads('{' + line + '}')
    c = data.get('content','')
    print(c[:100])
except:
    print('(extraction failed)')
" 2>/dev/null)

            if [ -z "$content_preview" ] || [ "$content_preview" = "(extraction failed)" ]; then
                # Try directly from full_text
                content_preview="${last_full_text:0:100}"
            fi
            echo "    response preview: ${content_preview:0:100}..."

            # Run assertions if this is the last turn or the turn has expect
            if [ "$has_assertions" = "yes" ]; then
                local contains_any
                contains_any=$(echo "$turns_json" | python3 -c "
import sys, json
turns = json.load(sys.stdin)
t = turns[$turn_idx]
e = t.get('expect', {})
print(json.dumps(e.get('contains_any', [])))
" 2>/dev/null)

                local not_contains
                not_contains=$(echo "$turns_json" | python3 -c "
import sys, json
turns = json.load(sys.stdin)
t = turns[$turn_idx]
e = t.get('expect', {})
print(json.dumps(e.get('not_contains', [])))
" 2>/dev/null)

                local expected_status
                expected_status=$(echo "$turns_json" | python3 -c "
import sys, json
turns = json.load(sys.stdin)
t = turns[$turn_idx]
e = t.get('expect', {})
print(e.get('http_status', 200))
" 2>/dev/null)

                # Check HTTP status
                if [ "$http_status" != "$expected_status" ]; then
                    echo "    [FAIL] HTTP status: got $http_status, expected $expected_status"
                    overall_pass=false
                    rm -f "$tmpfile"
                    continue
                fi

                # Check contains_any
                local contains_list
                IFS=',' read -ra contains_list <<< "$(echo "$contains_any" | python3 -c "
import sys, json
items = json.loads(sys.stdin.readline())
for i in items:
    print(i)
" 2>/dev/null)"

                if [ "$contains_any" != "[]" ] && [ -n "$contains_any" ]; then
                    local found_any=false
                    while IFS= read -r keyword; do
                        [ -z "$keyword" ] && continue
                        if echo "$last_full_text" | grep -iq "$keyword"; then
                            found_any=true
                            break
                        fi
                    done < <(echo "$contains_any" | python3 -c "
import sys, json
for k in json.loads(sys.stdin.readline()):
    print(k)
" 2>/dev/null)

                    if [ "$found_any" = false ]; then
                        echo "    [FAIL] contains_any: none of $contains_any found in response"
                        overall_pass=false
                    else
                        echo "    [OK] contains_any matched"
                    fi
                fi

                # Check not_contains
                if [ "$not_contains" != "[]" ] && [ -n "$not_contains" ]; then
                    while IFS= read -r keyword; do
                        [ -z "$keyword" ] && continue
                        if echo "$last_full_text" | grep -iq "$keyword"; then
                            echo "    [FAIL] not_contains: '$keyword' found in response"
                            overall_pass=false
                        else
                            echo "    [OK] not_contains: '$keyword' not found"
                        fi
                    done < <(echo "$not_contains" | python3 -c "
import sys, json
for k in json.loads(sys.stdin.readline()):
    print(k)
" 2>/dev/null)
                fi
            fi

            rm -f "$tmpfile"

            # Wait between requests within a case
            if [ "$turn_idx" -lt "$((num_turns - 1))" ]; then
                sleep 1.5
            fi
        done

        # Report case result
        if [ "$overall_pass" = true ]; then
            echo -e "  ${GREEN}[PASS]${NC} $case_id"
            suite_passed=$((suite_passed + 1))
            TOTAL_PASSED=$((TOTAL_PASSED + 1))
        else
            echo -e "  ${RED}[FAIL]${NC} $case_id"
            suite_failed=$((suite_failed + 1))
            TOTAL_FAILED=$((TOTAL_FAILED + 1))
        fi

        # Wait between cases
        sleep 1.5

    done < <(tail -n +2 "$suite_file")

    echo ""
    echo "Suite: $suite_name  ${suite_passed}/${suite_total} passed"
}

# Process all four suites
process_suite "$SUITES_DIR/flow-basic.jsonl"
process_suite "$SUITES_DIR/quiz-marriage.jsonl"
process_suite "$SUITES_DIR/quiz-career-wealth.jsonl"
process_suite "$SUITES_DIR/edge-input.jsonl"

# Grand total
echo ""
echo "=========================================="
echo " FINAL TOTALS"
echo "=========================================="
if [ $TOTAL_FAILED -eq 0 ]; then
    echo -e "${GREEN}TOTAL: ${TOTAL_PASSED}/${TOTAL_CASES} passed (100%)${NC}"
else
    local pct=$((TOTAL_PASSED * 100 / TOTAL_CASES))
    echo -e "${RED}TOTAL: ${TOTAL_PASSED}/${TOTAL_CASES} passed (${pct}%)${NC}"
fi
echo ""
