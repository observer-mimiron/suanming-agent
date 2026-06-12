# 命理大师测试集

## 目录

```
testsets/
├── README.md                      # 本文件（skill 按此文件识别场景选择套件）
│
├── raw/                           # 原始资料（只读归档，不直接用于测试）
│   ├── mingli-bench-160.json      # 2022-2025 MingLi-Bench 汇总（160题）
│   ├── competition-2022.json      # 2022 第13届大赛官方答案（8命例×5题）
│   ├── competition-2023.json      # 2023 第14届大赛官方答案（4命例八字部分）
│   ├── competition-2024.json      # 2024 第15届大赛官方答案（6命例八字部分+答案键）
│   ├── competition-2025.json      # 2025 第16届大赛官方答案（8命例×5题）
│   ├── search-test-cases.json     # 知识库检索测试用例
│   └── test-cases.md              # 手写场景（旧格式，已迁移到 suites/）
│
├── suites/                        # 清洗后测试套件（JSONL 格式，一行一用例）
│   ├── quiz-marriage.jsonl        # 婚姻判例 (6)
│   ├── quiz-year-event.jsonl     # 流年事件判例 (6)
│   ├── quiz-wealth.jsonl         # 家境/学历/职业判例 (6)
│   ├── quiz-health.jsonl         # 健康/意外/手术判例 (6)
│   ├── flow-bazi-input.jsonl     # bazi_input 全路径 (4)
│   ├── flow-new-profile.jsonl    # new_profile 全路径 (7)
│   ├── edge-cases.jsonl          # 边界异常输入 (6)
│   └── runner.py                 # 测试执行器
│
└── reports/                       # 测试报告输出目录
```

---

## 一、套件详情

### 1. quiz-marriage.jsonl — 婚姻判例

**场景覆盖：** 婚姻质量诊断 — 夫亡、同性恋、多次恋爱未婚、外遇、离异有孩、二婚。

**何时使用：**
- 修改了 `prompts/marriage.md` 或 `prompts/forensic.md` 中的婚姻分析规则
- 修改了 `orchestrator.go` 中婚姻相关的 prompt 路由（`selectPrompt` kwMap）
- 修改了 `extract.go` 中性别/婚姻相关的分类逻辑
- 新增或修改了婚姻类测试数据

**用例清单：**

| ID | 命例 | 出生信息 | 问题 | 答案 | 关键信号 |
|----|------|---------|------|------|---------|
| marriage-01-fuzaowang | 坤造1951 | 1951.11.14 巳时 女 广东 | 婚姻如何？ | D. 夫早亡 | 伤官见官+羊刃+酉午破 |
| marriage-02-tongxinglian | 坤造1987 | 1987.7.5 午时 女 香港 | 现时婚姻状况？ | D. 同性恋 | 官杀混杂+比劫争夫 |
| marriage-03-duoci | 乾造1983 | 1983.4.21 6时 男 宫崎 | 婚恋和子女情况？ | C. 多次恋爱未婚无子女 | 财星不显+婚姻宫受克 |
| marriage-04-waiyu | 坤造1980 | 1980.8.24 16:30 女 广东 | 2022年婚姻经历？ | C. 丈夫外遇 | 正官坐桃花+冲动夫宫 |
| marriage-05-liyihun | 坤造1990 | 1990.5.23 17:30 女 大陆 | 婚姻子女状况？ | D. 离异有孩 | 日坐伤官+官星受损 |
| marriage-06-erhun | 乾造1980 | 1980.7.11 巳时 男 香港 | 第一次结婚年份？ | C. 2005年 | 财星被合+比劫夺妻 |

**期望行为：**
- T1 建立命盘：`turn_type=full_reading`
- T2 追问婚姻：`reuse_chart=true`，`knowledge_search=true`，回答命中答案关键词，不出现咨询口吻（"婚期推断""什么时候结婚"）

---

### 2. quiz-year-event.jsonl — 流年事件判例

**场景覆盖：** 具体年份事件诊断 — 夫亡、抢劫、抑郁、丧母、车祸、退学。

**何时使用：**
- 修改了 `prompts/forensic.md`（核心影响）
- 修改了 `prompts/interpret.md` 中的流年分析 SOP
- 修改了 `orchestrator.go` 中的年份路由逻辑（`yearEventRe`、`selectPrompt`）
- 修改了 `orchestrator.go` 中的 `handleFollowupReading` 流年分析流程
- 新增或修改了流年类测试数据

**用例清单：**

| ID | 命例 | 年份 | 问题 | 答案 | 易混淆点 |
|----|------|------|------|------|---------|
| year-01 | 坤造1951 | 1993 | 发生何事？ | B. 丈夫病亡 | — |
| year-02 | 坤造1951 | 1980 | 发生何事？ | C. 被抢劫受伤 | **易误判为结婚年** |
| year-03 | 乾造1974 | 1996 | 发生何事？ | A. 严重抑郁症 | 易忽略心理疾病信号 |
| year-04 | 乾造1983 | 2022 | 重要事件？ | A. 母亲去世 | 大运转换+印星受损 |
| year-05 | 乾造1993 | 2001 | 发生何事？ | D. 交通意外人平安 | 驿马冲+金舆 |
| year-06 | 乾造1988 | 2004 | 发生何事？ | A. 被赶出学校 | 伤官见官+桃花惹祸 |

**期望行为：**
- T1 建立命盘：`turn_type=full_reading`
- T2 追问年份事件：`reuse_chart=true`，回答命中答案关键词，使用 forensic 风格（干支分析+冲合关系）

---

### 3. quiz-wealth.jsonl — 家境/学历/职业判例

**场景覆盖：** 出身贫富判断、学历高低判断、职业方向判断（含特殊职业：入殓师、风水顾问）。

**何时使用：**
- 修改了 `prompts/interpret.md`（通用解读 prompt）
- 修改了 `prompts/career.md`
- 修改了 `orchestrator.go` 中 career/wealth 相关的 prompt 路由
- 修改了 `extract.go` 中出身/学历相关的分类逻辑
- 新增或修改了家境/学历/职业类测试数据

**用例清单：**

| ID | 命例 | 问题 | 答案 | 类型 |
|----|------|------|------|------|
| wealth-01-poor | 坤造1951 | 出生家境？ | B. 贫穷 | 家境-贫 |
| wealth-02-ordinary | 坤造1980 | 出身情况？ | B. 普通 | 家境-普通 |
| wealth-03-rich | 乾造1973 | 性格和出身？ | A. 有钱家庭 | 家境-富 |
| wealth-04-phd-engineer | 乾造1983 | 学历及工作？ | D. 博士工程师 | 学历-高 |
| wealth-05-mortician | 乾造1993 | 2022年后职业？ | B. 入殓师 | 职业-特殊 |
| wealth-06-fengshui | 乾造1986 | 职业？ | D. 风水顾问 | 职业-玄学 |

**期望行为：**
- T1 建立命盘：`turn_type=full_reading`
- T2 追问：`reuse_chart=true`，回答命中答案关键词

---

### 4. quiz-health.jsonl — 健康/意外判例

**场景覆盖：** 抑郁症、交通意外、癌症、自杀/六亲非正常死亡、脑部手术、慢性病致贫。

**何时使用：**
- 修改了 `prompts/health.md`
- 修改了 `prompts/forensic.md` 中的健康/意外相关规则
- 修改了 `orchestrator.go` 中 health 相关的 prompt 路由
- 新增或修改了健康类测试数据

**用例清单：**

| ID | 命例 | 问题 | 答案 | 类型 |
|----|------|------|------|------|
| health-01 | 乾造1974 USA | 1996年发生何事？ | A. 严重抑郁症 | 心理 |
| health-02 | 乾造1993 新加坡 | 2001年发生何事？ | D. 交通意外人平安 | 意外 |
| health-03 | 乾造1973 马来西亚 | 2024年健康状况？ | C. 吸烟引发鼻癌 | 癌症 |
| health-04 | 坤造1966 香港 | 叔叔哪年跳楼亡？ | B. 1984年 | 自杀/六亲 |
| health-05 | 乾造1981 香港 | 2014年手术部位？ | B. 脑部 | 手术 |
| health-06 | 坤造1966 香港 | 2008年至今命运？ | D. 多病穷困潦倒 | 慢性病 |

**期望行为：** T1 full_reading，T2 reuse_chart + 命中答案关键词。

---

### 5. flow-bazi-input.jsonl — bazi_input 全路径

**场景覆盖：** 用户直接输入八字的各种情况。

**何时使用：**
- 修改了 `orchestrator.go` 中 `case "bazi_input"` 分支
- 修改了 `handleBaziInput` 函数
- 修改了 `extract.go` 中 bazi_input 分类规则
- 修改了 `profileChangesAffectChart` 或 `containsBirthTime` 保护函数
- 修改了 bazi_input 相关的 session state 处理

**用例清单：**

| ID | 场景 | 轮次 | 验证点 |
|----|------|------|--------|
| flow-bazi-01 | 带性别输入 "乙巳…，女" | 1 | turn_type=direct_bazi，不追问性别 |
| flow-bazi-02 | 无性别输入 "甲子 乙丑…" | 1 | turn_type=direct_bazi，追问性别 |
| flow-bazi-03 | 先输八字再回"女" | 2 | T1=direct_bazi，T2=reuse_chart |
| flow-bazi-04 | 八字→婚姻→财运→运势 | 4 | 4轮全复用命盘不丢失 |

**期望行为：** 每个 turn 都正确路由，session 不丢失，性别提取后自动合并。

---

### 5. flow-new-profile.jsonl — new_profile 全路径

**场景覆盖：** 用户提供出生信息建立命盘的各种情况。

**何时使用：**
- 修改了 `orchestrator.go` 中 `case "new_profile"` / `case "update_profile"` / `case "incomplete"` 分支
- 修改了 `handleFullReading`、`handleAsk`、`handleFollowupReading`
- 修改了 `extract.go` 中 new_profile/update_profile 分类规则
- 修改了 `state/session.go` 中 Profile/BaziResult/MergeProfile
- 修改了 session 持久化逻辑

**用例清单：**

| ID | 场景 | 轮次 | 验证点 |
|----|------|------|--------|
| flow-np-01 | 完整出生信息 | 1 | turn_type=full_reading，不追问 |
| flow-np-02 | 不完整"想看八字" | 1 | turn_type=ask_missing_profile |
| flow-np-03 | 先说年份再补全 | 2 | T1=ask，T2=full_reading |
| flow-np-04 | 纠错性别不改出生时间 | 2 | T1=full_reading，T2=reuse_chart |
| flow-np-05 | 改年份应重排 | 2 | T1=full_reading，T2=full_reading（重排） |
| flow-np-06 | 建立命盘后3轮追问 | 4 | 全部复用命盘 |
| flow-np-07 | 两个session互不干扰 | 4 | session隔离正确 |

**期望行为：** 路由正确（full_reading/ask_missing_profile），状态保持（reuse_chart），session 隔离。

---

### 6. edge-cases.jsonl — 边界异常输入

**场景覆盖：** 系统面对异常输入时的鲁棒性。

**何时使用：**
- 修改了 `handler/chat.go` 的请求处理
- 修改了 `extract.go` 的输入解析/fallback 路径
- 修改了 `orchestrator.go` 的错误处理/turn_type 路由
- 修改了 SSE 输出格式

**用例清单：**

| ID | 输入 | 期望 | 验证点 |
|----|------|------|--------|
| edge-01 | ""（空字符串） | http_status=400 | handler 层拦截 |
| edge-02 | "   "（纯空白） | http_status=400 | handler 层拦截 |
| edge-03 | "asdfghjkl12345!@#$%"（乱码） | turn_type=ask_missing 或 incomplete | 不 panic，graceful 降级 |
| edge-04 | "12345678"（纯数字） | turn_type=ask_missing 或 incomplete | 不误判为出生年份 |
| edge-05 | 出生时间+八字混合 | turn_type=full_reading | 优先 new_profile 排盘 |
| edge-06 | "你好"（纯问候） | turn_type=ask_missing 或 incomplete | 正常响应不 panic |

**期望行为：** 空消息返回 400，异常输入 graceful 降级，不 panic，不 500。

---

## 二、JSONL 数据格式

每行一个 JSON 对象，所有套件共用此结构：

```json
{
  "id": "唯一标识（kebab-case）",
  "desc": "中文描述，说明此用例测试什么",
  "eval_type": "quiz | flow | edge",
  "tags": ["标签数组，用于过滤和分组"],
  "turns": [
    {
      "message": "用户输入文本",
      "session_id": "会话ID（跨turn共享。留空则自动生成 case-id-tN）",
      "expect": {
        "turn_type": "精确匹配turn_type（可选）",
        "turn_type_any": ["匹配任一即可（可选）"],
        "reuse_chart": true,
        "knowledge_search": true,
        "contains_any": ["正则关键词，至少命中一个"],
        "contains_all": ["正则关键词，须全部命中"],
        "not_contains": ["不应出现的文本"],
        "http_status": 400
      }
    }
  ],
  "answer": "官方答案（quiz有，flow/edge为null）",
  "source": "competition-YYYY | handwritten"
}
```

### expect 字段说明

| 字段 | 类型 | 适用 eval_type | 说明 |
|------|------|--------------|------|
| `turn_type` | string | 全部 | 精确匹配。如 `full_reading` |
| `turn_type_any` | []string | 全部 | 匹配任一即可。用于有多个合法路由的场景 |
| `reuse_chart` | bool | 全部 | 检查 `handleFollowupReading` 是否被执行 |
| `knowledge_search` | bool | quiz, flow | 检查是否触发了知识库检索 |
| `contains_any` | []string | quiz, flow | 回答文本中至少命中一个正则关键词 |
| `contains_all` | []string | quiz, flow | 回答文本中须全部命中 |
| `not_contains` | []string | 全部 | 回答/事件中不应出现这些关键词 |
| `http_status` | int | edge | HTTP 响应码检查 |

---

## 三、Skill 识别规则

测试 skill (`test-bazi-accuracy`) 根据**改动的文件**自动选择套件。映射规则如下：

```
改动文件匹配 → 必须跑的套件：

prompts/forensic.md                     → quiz-year-event, quiz-marriage, quiz-health
prompts/marriage.md                     → quiz-marriage
prompts/health.md                       → quiz-health
prompts/interpret.md                    → quiz-wealth, quiz-year-event
prompts/career.md                       → quiz-wealth
prompts/direct.md                       → quiz-year-event, quiz-marriage

orchestrator.go:bazi_input/handleBaziInput → flow-bazi-input
orchestrator.go:new_profile/update_profile/incomplete → flow-new-profile
orchestrator.go:handleFollowupReading    → quiz-year-event, flow-new-profile
orchestrator.go:handleFullReading        → quiz-wealth, flow-new-profile
orchestrator.go:Run/switch/路由          → flow-bazi-input, flow-new-profile, edge-cases
orchestrator.go:profileChangesAffectChart → flow-bazi-input
orchestrator.go:containsBirthTime        → flow-bazi-input, flow-new-profile
orchestrator.go:recordTurnAndMaintainContext → flow-new-profile

extract.go:classifyAndExtract            → flow-new-profile, edge-cases
extract.go:分类 prompt                    → flow-new-profile, flow-bazi-input

state/session.go                         → flow-new-profile, flow-bazi-input
state/store.go                           → flow-new-profile, flow-bazi-input

handler/chat.go                          → edge-cases
web/src/composables/useSSE.ts            → edge-cases（手动验证，非自动化）

Makefile / .env / go.mod                 → （无对应套件，仅需 go build 确认编译通过）
```

**匹配原则：**
1. 改动触及多个文件时，合并对应的所有套件
2. 每次至少跑 1 个、最多跑 4 个套件（超过 4 个说明改动面太大，应先缩小范围）
3. 不在上述映射中的文件改动 → 只跑 `go test ./...` 和 `go build`
4. 发布前全量回归 → 跑全部 6 套件（~35 用例，~20 分钟）

---

## 四、执行器使用

```bash
# 跑单个套件
python3 testsets/suites/runner.py testsets/suites/quiz-marriage.jsonl

# 指定后端地址
python3 testsets/suites/runner.py testsets/suites/flow-bazi-input.jsonl http://localhost:18080

# 跑多个相关套件
for f in quiz-marriage quiz-year-event; do
    python3 testsets/suites/runner.py testsets/suites/$f.jsonl
done

# 跑全部（仅发布前使用）
python3 testsets/suites/runner.py testsets/suites/quiz-marriage.jsonl
python3 testsets/suites/runner.py testsets/suites/quiz-year-event.jsonl
python3 testsets/suites/runner.py testsets/suites/quiz-wealth.jsonl
python3 testsets/suites/runner.py testsets/suites/flow-bazi-input.jsonl
python3 testsets/suites/runner.py testsets/suites/flow-new-profile.jsonl
python3 testsets/suites/runner.py testsets/suites/edge-cases.jsonl
```

---

## 五、套件变更记录

| 日期 | 变更 | 说明 |
|------|------|------|
| 2026-06-12 | 新建全部 7 套件 | 从 raw/ 竞赛数据 + 手写场景清洗得来 |
| 2026-06-12 | 删除旧 suite-*.json | 迁移到 JSONL 统一格式 |
| 2026-06-12 | runner.py 重构 | 支持 JSONL + JSON 自动识别 |
| 2026-06-12 | 新增 raw/competition-{2022,2023,2024}.json | 归档 2022-2024 大赛官方答案 |
| 2026-06-12 | 新增 quiz-health.jsonl | 健康/意外/手术判例（6用例） |

---

## 六、数据来源与准确性

- **quiz 套件答案**：香港青年术数家协会「全球算命师大赛」2022-2025 官方公布答案。准确性 ⭐⭐⭐⭐⭐
- **flow 套件**：根据实际代码路由逻辑手写，反映当前架构的期望行为。
- **edge 套件**：根据 handler 和 classify 的错误处理逻辑手写。

## 七、评分参考

| 水平 | 准确率 | 说明 |
|------|--------|------|
| 随机基线 | 25% | 四选一 |
| 人类从业者 | 40-45% | 数年经验 |
| 大赛冠军 | ~50% | 2024年冠军得分率 |
| AI 当前最佳 | 37-40% | 2024年AI参赛结果 |

八字推理本质高难度，冠军 ~50%，不要期望 80%+。
