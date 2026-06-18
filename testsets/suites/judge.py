"""LLM-as-Judge — 对 CanonicalTrace 做主观质量评估。

原则 (Hamel Husain / Shreya Shankar):
- 二值输出 (true/false)，不用 Likert
- 独立模型，不与被测 agent 共享上下文
- 需人工标注 ≥50 条校准后才可作为门禁
"""

import json, os, subprocess
from canonical import CanonicalTrace


class AgentJudge:
    """二值质量判断。维度: faithfulness, citation, coherence, relevance"""

    DIMENSIONS = {
        "faithfulness": """判断回复是否忠于工具返回数据而非编造。
- 回复中的具体事实（数字、日期、干支、五行）能在 actions 或 retrievals 中找到 → true
- 回复中出现了来源中没有的具体信息 → false""",

        "citation": """判断引用是否真实。
- 引用的内容与 retrievals 中的原文一致 → true
- 引用不存在的书名、或内容与原文不符 → false""",

        "coherence": """判断推理逻辑是否自洽。
- 推理步骤有因果链，前后不矛盾 → true
- 出现明显自相矛盾 → false""",

        "relevance": """判断是否切题回答了用户问题。
- 直接回应了用户的核心意图 → true
- 偏离主题、答非所问 → false""",
    }

    def __init__(self, api_key=None, base_url=None, model=None):
        self.api_key = api_key or os.environ.get("JUDGE_API_KEY", os.environ.get("LLM_API_KEY", ""))
        self.base_url = base_url or os.environ.get("JUDGE_BASE_URL", os.environ.get("LLM_BASE_URL", ""))
        self.model = model or os.environ.get("JUDGE_MODEL", "claude-sonnet-4-6")

    def judge(self, dimension: str, trace: CanonicalTrace, user_message: str = "") -> tuple:
        """返回 (passed: bool, reason: str)"""
        if dimension not in self.DIMENSIONS:
            return False, f"Unknown dimension: {dimension}"

        prompt = f"""评估 Agent 输出质量。标准:
{self.DIMENSIONS[dimension]}

用户消息: {user_message}
Agent回复: {trace.final_output[:3000]}
工具返回: {json.dumps([{k: a.get(k) for k in ('name','result','error') if k in a} for a in trace.actions], ensure_ascii=False)[:2000]}
知识库: {json.dumps([{'source': c.get('source',''), 'content': c.get('content','')[:200]} for r in trace.retrievals for c in r.get('chunks',[])], ensure_ascii=False)[:1000]}

仅输出 true 或 false，然后一行理由。"""

        try:
            payload = json.dumps({
                "model": self.model, "max_tokens": 256,
                "messages": [{"role": "user", "content": prompt}],
            })
            r = subprocess.run(
                ["curl", "-s", f"{self.base_url}/v1/messages",
                 "-H", f"x-api-key: {self.api_key}",
                 "-H", "Content-Type: application/json",
                 "-d", payload, "--max-time", "30"],
                capture_output=True, text=True, timeout=35,
            )
            resp = json.loads(r.stdout)
            text = ""
            if "content" in resp:
                text = resp["content"][0].get("text", "") if isinstance(resp["content"], list) else str(resp["content"])
            elif "choices" in resp:
                text = resp["choices"][0]["message"]["content"]
            passed = text.strip().lower().startswith("true")
            return passed, text.strip()[:200]
        except Exception as e:
            return False, f"JUDGE_ERROR: {e}"
