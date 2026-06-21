---
name: agent-test-suites
description: Use when prompted to run tests against the agent test suite, when code changes affect routing/prompt/classification logic, or when the user says "测试", "test", "跑测试", or references test sets. Also triggers on model switches or architecture changes that need regression verification.
---

# Agent Test Suites

## Overview

用结构化测试集验证 Agent 行为正确性。核心理念：**改动什么测什么，按改动范围选套件，绝不全量跑。**

## 三级分层

| 级别 | 触发条件 | 套件范围 | 耗时 |
|------|---------|---------|------|
| **Smoke** | 每次改动后 | flow-basic | ~2min |
| **Standard** | prompt/路由/分类改动 | + quiz-marriage, quiz-career-wealth, edge-input | ~5min |
| **Exhaustive** | 模型切换、架构变更、release 前 | + quiz-knowledge, quiz-year-event, edge-adversarial, edge-resilience | ~10min |

## When to Use

- 修改了 prompt、路由、分类、session 状态相关代码后
- 切换模型或模型配置后
- 用户直接要求跑测试
- 架构变更需要回归验证

**When NOT to use:**
- 纯前端改动（UI 组件、样式）
- 文档修改（README、设计文档）
- 不影响 Agent 行为的工具代码改动（先用 `go test ./...`）
- 首次探索代码库时

## 套件速查

| 套件 | 级别 | 覆盖维度 |
|------|------|---------|
| `flow-basic` | Smoke | 新建会话、追问复用、无八字闲聊 |
| `quiz-marriage` | Standard | 婚姻感情：时机、正缘、配偶、风险 |
| `quiz-career-wealth` | Standard | 事业财运：方向、转行、投资、升职 |
| `quiz-ziwei` | Standard | 紫微斗数：排盘、十二宫、流年、婚姻事业 |
| `quiz-ziwei-oral` | Standard | 紫微口语化路由：口语表达路由稳定性、多术数混合 domain 选择 |
| `edge-input` | Standard | 边界输入：短消息、缺性别、无关话题、中英混杂 |
| `quiz-knowledge` | Exhaustive | 知识库检索：古籍引用、出处标注、伪书检测 |
| `quiz-knowledge-edge` | Exhaustive | 知识图谱专项：catalog、3次预算、GetGraph、Agentic RAG |
| `quiz-year-event` | Exhaustive | 流年专项：具体年份、冲合刑害、大运区间 |
| `edge-adversarial` | Exhaustive | 对抗性：矛盾输入、不可能日期、prompt 注入 |
| `edge-resilience` | Exhaustive | 容错：多领域、超长输入、子时/高龄出生 |

详情见 `testsets/README.md` 第一节。

## 选择套件

**先读 `testsets/README.md` 第三节「Skill 识别规则」**，按改动文件→套件的映射表选择。

约束：
- Smoke: 只跑 `flow-basic`
- Standard: 加入 3 个 Standard 套件（最多 4 个）
- Exhaustive: 最多 4 个，根据改动面选择最相关的 Exhaustive 套件
- 不在映射表中的文件改动 → 只跑 `go test ./...` + `go build`

## 执行环境差异

### Claude Code
直接执行。支持长超时和后台任务，跑全套 Smoke + Standard 约 5 分钟。

### Codex (OpenAI Codex CLI)

**关键限制：**
- `exec_command` 有 ~30s yield 上限，单个 LLM 调用（6-120s）会超时
- 跨 `exec_command` 时后台进程被 SIGHUP 杀死，不能用 `nohup ... &` + 下一个 exec_command 发请求
- 同一 `exec_command` 内 Python 子进程 curl 可以正常访问 localhost（已验证，无沙箱限制）

**可行方案：**

1. **只跑 `go test ./...`** — 单元测试在 30s 内完成，Codex 内直接跑。最稳定。

2. **自包含 Python 脚本（推荐）** — 同进程启动服务 + 跑测试，用 `write_stdin` 轮询：
   ```python
   import subprocess, time, os, sys

   server = subprocess.Popen(
       ['/path/to/suanming-server'],
       env={**os.environ,
            'LLM_API_KEY': 'sk-xxx',
            'LLM_BASE_URL': 'https://api.deepseek.com/anthropic',
            'LLM_MODEL': 'deepseek-v4-pro',
            'KNOWLEDGE_MCP_URL': 'http://localhost:3100',
            'LISTEN_ADDR': ':18080'},
       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

   time.sleep(6)
   # 验证服务在线
   r = subprocess.run(['curl', '-s', 'http://localhost:18080/api/health'],
       capture_output=True, text=True)
   assert r.stdout.strip() == '{"status":"ok"}'

   sys.path.insert(0, 'testsets/suites')
   import runner
   runner.run_suite('http://localhost:18080', 'testsets/suites/flow-basic.jsonl',
       timeout=120, delay=2)

   server.terminate(); server.wait()
   ```
   关键约束：
   - **服务必须在同一 Python 进程内启动**，否则跨 exec_command 会 SIGHUP
   - **stdout/stderr 必须重定向到 DEVNULL**，避免缓冲区满阻塞
   - 单次请求耗时 1-2 分钟，完整 5 case 套件 5-10 分钟，需多次 `write_stdin` 轮询
   - 轮询间隔建议 2-3 分钟，runner 在 curl 阻塞期间不产生 stdout 输出

3. **Codex 只做代码审查和 go test** — E2E 回归手动在本地终端跑，最可靠。

**已验证失败的模式：**
- ❌ `nohup suanming-server &` → 等下一个 exec_command 跑 runner → 服务被 SIGHUP 杀死
- ❌ `nohup python3 runner.py &` 后台跑 → runner 子进程 curl 超时后 stdout 无输出，无法轮询

### 手动跑（任何环境）
```bash
# 终端 1：启动服务
cd "$(git rev-parse --show-toplevel)"
make build
set -a; source .env; set +a
LISTEN_ADDR=:18080 /tmp/suanming-server &

# 终端 2：跑测试
python3 testsets/suites/runner.py testsets/suites/flow-basic.jsonl http://localhost:18080
```

## 跑测试前检查清单

**必须按顺序执行，跳过任一步骤是时间黑洞。**

### 0. AI 探针门禁（硬性，最先执行）

**在启动服务或跑 runner 之前，AI 必须先发一条探针请求，确认单轮响应在可控范围内。**

这是防止 30 分钟时间黑洞的第一道防线。runner 内置的 `check_server_ready()` 会自动执行此检查并输出时间/大小，但 **AI 在调用 runner 前也要自行判断**：

```
runner 输出示例 (三层门禁):
✓ 服务启动校验通过 (commit=56e459c)
✓ 探针门禁通过 (probe=12.3s, body=45.2KB)
✓ 业务能力探针通过 (route_primary=bazi)
```

**AI 上下文健康指标**（宽松设置，只拦截明显异常）：

| 指标 | 默认上限 | 环境变量 | 含义 |
|------|---------|---------|------|
| 单轮时间 | 120s | `PROBE_TIME_LIMIT` | 单轮 2 分钟足够任何 specialist，超时说明模型卡死 |
| 响应大小 | 1MB | `PROBE_SIZE_LIMIT` | 正常 SSE 响应 < 100KB，1MB 已留足 10 倍余量 |
| 预估总时长 | 见 runner 输出 | — | runner 启动后打印 `Est. total: ~Xs`，AI 据此判断是否继续 |

**Tier 1/2 任一超限，runner 自动退出 (exit code 2)，AI 不得继续尝试跑 suite。Tier 3 仅 WARN 不退出。** 此时应：
- 检查模型是否切换导致输出变长
- 检查 prompt 是否膨胀
- 缩小套件范围或降低 workers

**AI 自主判断规则**（runner 不会拦截，由 AI 自行决策）：
- 预估总时长 > 15min → 建议拆分为多个子套件分批跑
- 预估总时长 > 30min → 必须拒绝，要求用户缩小范围或改跑 Smoke 级别
- body > 200KB 但 < 1MB → 警告但仍可继续，注意上下文消耗

### 1. 确认服务版本

`lsof -i :18080` 看 PID，如果服务在代码修改前启动，先 kill 再启动。

### 2. 构建二进制，不要 go run

`go build -o /tmp/suanming-server ./cmd/server/ && /tmp/suanming-server &`。`go run` 可能因进程退出而丢失。

### 3. 验证 health

`curl http://localhost:18080/api/health` 返回 `{"status":"ok"}`。

### 4. 抓一条真实 SSE 响应

对最小 case 手动 curl，检查 SSE body 中 route-decision 的 JSON 字段顺序和存在性。**不要跳过这步直接跑 suite。**

### 5. 先跑 1 个 case

`python3 runner.py suite.jsonl http://localhost:18080 --workers 1` 确认 runner 解析正确。

## 启动被测服务

```bash
cd "$(git rev-parse --show-toplevel)"
COMMIT=$(git rev-parse --short HEAD)
go build -ldflags="-X github.com/wikiglobal/suanming-agent/internal/container.BuildCommit=$COMMIT" \
    -o /tmp/suanming-server ./cmd/server/
set -a; source .env; set +a
LISTEN_ADDR=:18080 /tmp/suanming-server &
sleep 3
curl http://localhost:18080/api/health  # 确认 {"status":"ok","commit":"xxxxx"}
```
**不要用 `go run ./cmd/server/ &`** — 进程崩溃后不易察觉，导致用旧二进制测试。

**runner 内置 smoke check** — `run_suite`/`run_suite_parallel` 启动时自动执行三层门禁：
1. **Tier 1 (硬)**：校验 `/api/health` 可达 + 对比 commit → 失败 exit code 2
2. **Tier 2 (硬)**：发探针请求，测量耗时和响应大小 → 超限 exit code 2
3. **Tier 3 (软)**：检查 SSE 包含 `route-decision` 且 `route_primary` 可解析 → 不满足打 WARN 但继续执行

Tier 1/2 任一失败即退出，杜绝「旧二进制跑新测试」和「响应膨胀导致 suite 跑不完」两类时间黑洞。Tier 3 为软门禁，避免「服务正常但 `"你好"` 探针语义不稳定」的误判。

## 执行

```bash
# 单个套件
python3 testsets/suites/runner.py testsets/suites/<suite-name>.jsonl http://localhost:18080

# 多个相关套件（用 for 循环，别用 --all）
for f in quiz-marriage quiz-year-event; do
    python3 testsets/suites/runner.py testsets/suites/$f.jsonl http://localhost:18080
done
```

### workers 选择

- `--workers 1`：包含紫微斗数 specialist 的套件（`quiz-ziwei`、`quiz-ziwei-oral`、`edge-resilience`），ziwei 排盘 LLM 调用重，并行会超时
- `--workers 4`：纯路由/八字/奇门类套件，这些 LLM 调用轻量
- 默认值 4 对轻量级套件安全，但遇到间歇性空路由时先降到 1 复验

## Runner API

runner 内部用 `TurnResponse` dataclass，不要用元组索引：

```python
from runner import run_turn, TurnResponse
r = run_turn('http://localhost:18080', '消息', 'session-id', timeout=90)
# r.http_code  r.body  r.turn_type  r.full_text  r.thinking
# r.route_primary  r.task_intent  r.qimen_mode  r.secondary_domains
```

新增断言类型 (`expect` 字段):
- `route_primary` — 精确匹配路由主域
- `task_intent` — 精确匹配 task_intent
- `task_intent_any` — task_intent 在列表中任一即可
- `qimen_mode` — 精确匹配 qimen_mode
- `secondary_contains` — secondary_domains 至少包含一个指定值

## 结果解读

```
[PASS] T1 "消息..." → turn_type: direct_bazi
[FAIL] T2 "追问..." → contains_any: 未命中关键词 [...]
```

每个 FAIL 按以下分类诊断根因：

| 分类 | 症状 | 检查方式 |
|------|------|---------|
| **routing-error** | turn_type 不符合预期 | 检查 trace-panel 中的 classify_and_extract action |
| **state-error** | reuse_chart=false 或 session 状态丢失 | 检查 data/sessions/ 下 JSON 文件 |
| **interpretation-gap** | 路由和状态正确，但内容未命中关键词 | LLM 输出方向对但不够精确 |
| **knowledge-gap** | knowledge_search 未触发或结果不相关 | 检查知识库检索 query |
| **classification-error** | LLM 分类错误（如把 update 判为 new） | 检查 classify prompt + flash LLM 响应 |

## 编写用例

### Case 结构

```json
{
  "id": "唯一标识",
  "turns": [
    {
      "message": "用户输入文本",
      "session_id": "同一session的turn用相同ID",
      "expect": { "断言": "值" },
      "grading": { "pass": ["必须出现的关键词"] }
    }
  ]
}
```

### 断言类型

| 字段 | 类型 | 语义 | 注意事项 |
|------|------|------|---------|
| `http_status` | int | 期望 HTTP 状态码 | |
| `contains_any` | [string] | 至少命中一个关键词 (substring) | 用 3-5 个足够宽泛的词，不要用单字 |
| `contains_all` | [string] | 全部命中 | |
| `not_contains` | [string] | 全部不出现 | ⚠️ 检查的是 SSE body + 提取文本，trace-panel JSON 里的字段名（如 "error"）也会被匹配 |
| `knowledge_search` | bool | 检查 body 中是否出现 knowledge_search 和 knowledge-sources | |

### 新格式断言（架构无关，v2）

使用 `--adapter` 参数启用轨迹级验证：

```bash
python3 testsets/suites/runner.py <suite.jsonl> <server_url> \
  --adapter testsets/adapters/suanming-agent-sse.yaml
```

新增断言字段：`action_called`, `action_sequence`, `action_arg_match`, `action_result_not_empty`, `retrieval_happened`, `retrieval_has_results`, `retrieval_cited`, `step_count_range`, `no_errors`, `final_output_contains`, `final_output_not_contains`

Suite 用 `_context` 定义占位符，实现领域值与断言解耦：

```json
{"name": "...", "_context": {"paipan": "bazi_paipan", "domain": "bazi"}}
{"id": "...", "expect": {"route_primary": "{domain}", "action_called": ["{paipan}"]}}
```

不传 `--adapter` 时自动使用旧断言逻辑，旧 suite 完全向后兼容。

**注意：** `turn_type`、`turn_type_any`、`reuse_chart` 在当前 SSE 格式中不可靠，不要使用。
- 短路径（preflight 直接返回）不经过 trace-panel，没有 turn_type 输出
- "复用已有命盘" 字符串当前代码不输出

### 编写流程（必须按顺序）

**第 0 步：先抓一条真实响应。** 写任何 case 之前，手动 curl 一个最简单的请求，检查 SSE 格式、文本提取效果、trace-panel JSON 内容。不跳过这步。

**第 1 步：写 1 个 case → 跑 → 确认通过。** 不要一次写完整套件。

**第 2 步：扩展 3-5 个 → 跑 → 全部通过才提交。**

**第 3 步：新代码改动后，先跑已有套件做回归，再根据改动面加新 case。**

### 编写原则

1. **模拟真实对话** — 用自然的中文，不是拼凑关键词。参考 `testsets/raw/test-cases.md` 里的真实案例
2. **一个 case 一个场景** — 不要在一个 case 里塞多个不相关的验证点
3. **多 turn 要连贯** — 同一 session 的 turn 之间要有对话逻辑
4. **expect 宁松勿紧** — `contains_any` 给 3-5 个宽泛关键词，不要用精确匹配刁难模型
5. **不写一定会失败的 case** — 除非刻意做对抗性测试
6. **每个 suite 3-5 个 case** — 太少覆盖率不够，太多跑不完
7. **`not_contains` 不要检查通用英文词** — "error"、"null"、"status" 等会在 SSE JSON 数据中自然出现，导致误报。只检查面向用户的中文文本

### 新增套件 checklist

- [ ] 新 `.jsonl` 文件放在 `testsets/suites/`
- [ ] 第一行是 `{"name": "...", "description": "..."}`
- [ ] 每个 case 有唯一的 `id`
- [ ] 每个 turn 有 `session_id`
- [ ] 每个 case 至少有一个 expect 断言
- [ ] 更新 `testsets/README.md` 第一节和第三节
- [ ] 更新本 skill 的套件速查表

## Common Mistakes

| 错误 | 正确做法 |
|------|---------|
| **不检查探针耗时/大小就开跑** | runner 会自动检测，AI 也要看输出中的 `probe=XXs body=XX`，超限立即终止 |
| 跑全部套件 | 按级别选：Smoke → Standard → Exhaustive |
| 测试失败后改 expect 关键词 | 先诊断根因分类，确定是代码 bug 还是 expectation 过时 |
| 不在映射表中也跑 suite 测试 | 只跑 `go test ./...` + `go build` |
| 改了模块 A 跑模块 B 的套件 | 只跑改动模块对应的套件 |
| 一个 suite 超过 5 个 case | 拆分为多个套件 |
| expect 关键词太精确 | 用足够宽泛的关键词覆盖合理的变化 |
| **不验证服务就跑 suite** | 先 `curl health`，再 `curl` 一条真实 SSE 检查 `route-decision` 格式 |
| **用旧二进制跑新测试** | 每次代码修改后 `go build && restart`，检查 PID |
| **写 route 断言前不抓真实 SSE** | 第 0 步：curl 一个最小请求，确认 JSON 字段顺序和存在性 |
| **元组索引硬编码** | 用 `TurnResponse` dataclass，`r.route_primary` 而非 `result[5]` |
| **紫微 suite 用 4 workers** | 紫微排盘 LLM 重，用 `--workers 1` 或 `--workers 2` |
| **删断言不加替代** | 删 `reuse_chart` 可加 `not_contains: ["请提供出生信息"]` 验证 session 复用 |

## 不做什么

- **不跑 --all**：除非是 release 前的全量回归
- **不修改代码来让测试通过**：测试失败先诊断根因，不是改 expect 关键词
- **不写伪书引用 case 但 expect 精确文本**：模型可能用不同的措辞表达同一意思
