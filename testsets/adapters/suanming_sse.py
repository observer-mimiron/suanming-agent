"""SSE 事件流适配器。"""

import json
import re
from base import TraceAdapter
from canonical import CanonicalTrace


class SSEAdapter(TraceAdapter):

    def _parse(self, raw_body: str, trace: CanonicalTrace):
        events = self._split_sse(raw_body)
        trace.raw_events = events
        fm, et = self._fm, self._et

        step_idx = 0
        current_actions = []

        for raw_event in events:
            obj = self._extract_json(raw_event)
            if obj is None:
                continue
            etype = obj.get("type", "")

            # 步骤边界
            if etype in et.get("step_boundary", []):
                if current_actions:
                    trace.steps.append({
                        "index": step_idx,
                        "actions": current_actions,
                    })
                    step_idx += 1
                    current_actions = []

            # 路由
            if etype in et.get("route_decision", []):
                trace.route = {
                    "primary": obj.get(fm.get("route_primary", ""), ""),
                    "intent": obj.get(fm.get("route_intent", ""), ""),
                    "secondary": obj.get(fm.get("route_secondary", ""), []),
                }

            # 动作调用
            if etype in et.get("action", []):
                action = {
                    "name": obj.get(fm.get("action_name", ""), ""),
                    "arguments": obj.get(fm.get("action_args", ""), {}),
                    "step_index": step_idx,
                }
                trace.actions.append(action)
                current_actions.append(action)

            # 动作结果
            if etype in et.get("action_result", []):
                a_name = obj.get(fm.get("action_name", ""), "")
                result = obj.get(fm.get("action_result", ""))
                error = obj.get(fm.get("action_error", ""), "")
                for a in trace.actions:
                    if a["name"] == a_name and "result" not in a:
                        a["result"] = result
                        a["error"] = error if error else None
                        break

            # 检索请求
            if etype in et.get("retrieval", []):
                trace.retrievals.append({
                    "query": obj.get(fm.get("retrieval_query", ""), ""),
                    "step_index": step_idx,
                    "chunks": [],
                })

            # 检索结果
            if etype in et.get("retrieval_result", []):
                sources = obj.get(fm.get("retrieval_sources", ""), [])
                chunks = []
                sn = fm.get("retrieval_source_name", "")
                cn = fm.get("retrieval_chunk_content", "")
                for s in (sources if isinstance(sources, list) else []):
                    if isinstance(s, dict):
                        chunks.append({
                            "source": s.get(sn, ""),
                            "content": s.get(cn, ""),
                        })
                if trace.retrievals:
                    trace.retrievals[-1]["chunks"] = chunks

            # 错误
            if etype in et.get("error", []):
                trace.meta.setdefault("errors", []).append(obj)

        # 最后一个 step
        if current_actions:
            trace.steps.append({
                "index": step_idx,
                "actions": current_actions,
            })

        # 最终输出
        final_cfg = fm.get("final_output", {})
        if final_cfg:
            parts = []
            target_type = final_cfg.get("event", "")
            target_field = final_cfg.get("field", "")
            for raw_event in events:
                obj = self._extract_json(raw_event)
                if obj and obj.get("type") == target_type:
                    text = obj.get(target_field, "")
                    if text:
                        parts.append(text)
            trace.final_output = "".join(parts)
        else:
            pattern = self._ce.get("full_text_pattern")
            if pattern:
                parts = re.findall(pattern, raw_body)
                trace.final_output = "".join(parts)

    def _split_sse(self, body: str) -> list:
        events = []
        for chunk in body.split("data:"):
            chunk = chunk.strip()
            if chunk and chunk != "[DONE]":
                events.append(chunk)
        return events

    def _extract_json(self, raw: str) -> dict | None:
        try:
            start = raw.find("{")
            if start == -1:
                return None
            return json.loads(raw[start:])
        except json.JSONDecodeError:
            return None
