# 八字命理测试集

用于「命理大师」AI 系统的标准化测试与准确率评估。

## 目录结构

```
testsets/
├── mingli-bench-160.json          # 主力测试集 (JSON, 160题)
├── mingli-bench-160-readable.md   # 同上，可读版 (32命例，带✅答案)
├── competition-extra.md           # 补充真题 (2020-2024, ~40题)
├── ten-gods.json                  # 十神定义 (10条)
├── sixty-jiazi.json               # 六十甲子纳音 (60条)
├── heavenly-stems.json            # 十天干 (10条)
├── earthly-branches.json          # 十二地支 (12条)
├── five-elements.json             # 五行生克基础
├── luck-cycles.json               # 大运流年规律 (120条)
├── element-combinations.json      # 五行合化规则 (25条)
├── compatibility-matrix.json      # 合婚矩阵 (144条)
└── seasonal-patterns.json         # 四季五行旺衰 (12条)
```

## 数据集说明

### 1. 主力测试集：MingLi-Bench (160 题)

| 属性 | 值 |
|------|-----|
| 来源 | GitHub DestinyLinker/MingLi-Bench |
| 数据 | 2022-2025 年全球算命师大赛真题 |
| 命例数 | 32 个（男/女各半，覆盖中国大陆/香港/台湾/马来西亚/美国） |
| 题目数 | 160 道四选一 |
| 覆盖维度 | 婚姻(44)、事业(25)、家庭(22)、健康(17)、性格(14)、财运(13)、学业(11)、子女(6) |
| 格式 | JSON (`data/questions[]`) |
| 可靠性 | ⭐⭐⭐⭐⭐ 官方比赛答案 |

**使用方式：**

```bash
# 评估某个模型
python -m mingli_bench.cli --model openai/gpt-4o --year 2025 --cot --astro

# 或自行解析 JSON
python3 -c "
import json
with open('testsets/mingli-bench-160.json') as f:
    data = json.load(f)
for q in data['questions']:
    print(f'[{q[\"case_id\"]}] {q[\"question\"]} → {q[\"answer\"]}')
"
```

### 2. 补充真题 (competition-extra.md)

MingLi-Bench 未覆盖的 2020-2024 年比赛真题，约 40 题。部分答案来自社区解析，可靠性略低于 MingLi-Bench。

### 3. 基础知识数据集 (FatePath)

9 个结构化 JSON 文件，覆盖八字基础知识，可用于单元测试：

| 数据集 | 测试场景 |
|--------|----------|
| `ten-gods.json` | 十神判定准确性 |
| `sixty-jiazi.json` | 六十甲子/纳音记忆 |
| `five-elements.json` | 五行生克规则 |
| `element-combinations.json` | 天干合化、地支合冲规则 |
| `heavenly-stems.json` | 十天干属性 |
| `earthly-branches.json` | 十二地支属性 |
| `seasonal-patterns.json` | 四季旺衰判断 |
| `luck-cycles.json` | 大运排法验证 |
| `compatibility-matrix.json` | 合婚配对规则 |

## 评分基准

| 场景 | 预期准确率 | 说明 |
|------|-----------|------|
| 随机基线 | 25% | 四选一瞎蒙 |
| 人类新手 | 30-35% | 学过基础但缺实战 |
| 人类从业者 | 40-45% | 有数年经验 |
| 大赛冠军 | 50% | 2024年冠军得分率 |
| AI 当前最佳 | 37-40% | DeepSeek-V3/Gemini-3-Pro |

> 比赛冠军得分率仅 ~50%，说明八字推理本质上是高难度、多解性任务，不要期待 80%+ 的准确率。

## 后续获取更多数据

- **全球算命师大赛官网**：香港青年术数家协会，每年6-8月公布新一届真题
- **MingLi-Bench**：`github.com/DestinyLinker/MingLi-Bench`，持续更新
- **知乎/公众号**：搜索「全球算命师大赛」可找到详细解析
