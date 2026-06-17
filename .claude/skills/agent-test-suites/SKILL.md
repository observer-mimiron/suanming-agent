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
set -a; source .env; set +a
LISTEN_ADDR=:18080 go run ./cmd/server/

# 终端 2：跑测试
python3 testsets/suites/runner.py testsets/suites/flow-basic.jsonl http://localhost:18080
```

## 启动被测服务

```bash
cd "$(git rev-parse --show-toplevel)"
set -a; source .env; set +a
LISTEN_ADDR=:18080 go run ./cmd/server/ &
sleep 5
curl http://localhost:18080/api/health  # 确认 {"status":"ok"}
```

## 执行

```bash
# 单个套件
python3 testsets/suites/runner.py testsets/suites/<suite-name>.jsonl http://localhost:18080

# 多个相关套件（用 for 循环，别用 --all）
for f in quiz-marriage quiz-year-event; do
    python3 testsets/suites/runner.py testsets/suites/$f.jsonl http://localhost:18080
done
```

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

| 字段 | 类型 | 语义 |
|------|------|------|
| `http_status` | int | 期望 HTTP 状态码 |
| `turn_type` | string | 精确匹配 turn_type |
| `turn_type_any` | [string] | 匹配任意一个 |
| `contains_any` | [string] | 至少命中一个关键词 (regex) |
| `contains_all` | [string] | 全部命中 |
| `not_contains` | [string] | 全部不出现 |
| `reuse_chart` | bool | 检查 body 中是否包含"复用已有命盘" |
| `knowledge_search` | bool | 检查 knowledge_search 和 knowledge-sources 是否触发 |

### 编写原则

1. **模拟真实对话** — 用自然的中文，不是拼凑关键词。参考 `testsets/raw/test-cases.md` 里的真实案例
2. **一个 case 一个场景** — 不要在一个 case 里塞多个不相关的验证点
3. **多 turn 要连贯** — 同一 session 的 turn 之间要有对话逻辑
4. **expect 宁松勿紧** — `contains_any` 给 3-5 个足够宽泛的关键词，不要用精确匹配来"刁难"模型
5. **不写一定会失败的 case** — 除非刻意做对抗性测试，否则 case 应该预期当前 Agent 能够通过
6. **每个 suite 3-5 个 case** — 太少覆盖面不够，太多跑不完

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
| 跑全部套件 | 按级别选：Smoke → Standard → Exhaustive |
| 测试失败后改 expect 关键词 | 先诊断根因分类，确定是代码 bug 还是 expectation 过时 |
| 不在映射表中也跑 suite 测试 | 只跑 `go test ./...` + `go build` |
| 改了模块 A 跑模块 B 的套件 | 只跑改动模块对应的套件 |
| 一个 suite 超过 5 个 case | 拆分为多个套件 |
| expect 关键词太精确 | 用足够宽泛的关键词覆盖合理的变化 |

## 不做什么

- **不跑 --all**：除非是 release 前的全量回归
- **不修改代码来让测试通过**：测试失败先诊断根因，不是改 expect 关键词
- **不写伪书引用 case 但 expect 精确文本**：模型可能用不同的措辞表达同一意思
