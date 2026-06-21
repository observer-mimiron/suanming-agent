"""SSE 事件流适配器。

处理实际 SSE 格式：
  event: <type>
  data: <json>

适配器将 SSE event 类型与 JSON 内部 "type" 字段结合以识别事件类别。
"""

import json
import re
from base import TraceAdapter
from canonical import CanonicalTrace


class SSEAdapter(TraceAdapter):

    def _get_field(self, obj: dict, path: str):
        """支持点号路径的字段提取，如 payload.primary_domain"""
        parts = path.split(".")
        current = obj
        for part in parts:
            if isinstance(current, dict):
                current = current.get(part, "")
            else:
                return ""
        return current

    def _parse(self, raw_body: str, trace: CanonicalTrace):
        events = self._split_sse(raw_body)
        trace.raw_events = [d for _, d in events]
        fm, et = self._fm, self._et

        step_idx = 0
        current_actions = []

        for sse_type, raw_data in events:
            obj = self._extract_json(raw_data)
            if obj is None:
                continue

            # 用 SSE 事件类型优先，其次尝试 JSON 内 type 字段
            json_type = obj.get("type", "")
            etype = sse_type or json_type

            # --- 步骤边界：component 事件（除 route-decision / trace-panel）---
            if etype in et.get("step_boundary", []):
                # 排除 route-decision 和 trace-panel
                if json_type not in ("route-decision", "trace-panel"):
                    if current_actions:
                        trace.steps.append({
                            "index": step_idx,
                            "actions": current_actions,
                        })
                        step_idx += 1
                        current_actions = []

            # --- 路由 ---
            if json_type in et.get("route_decision", []):
                trace.route = {
                    "primary": self._get_field(obj, fm.get("route_primary", "")),
                    "intent": self._get_field(obj, fm.get("route_intent", "")),
                    "secondary": self._get_field(obj, fm.get("route_secondary", "")),
                }
                if not isinstance(trace.route["secondary"], list):
                    trace.route["secondary"] = []

            # --- 动作调用：tool_call 事件 ---
            if etype in et.get("action", []):
                action = {
                    "name": self._get_field(obj, fm.get("action_name", "")),
                    "arguments": self._get_field(obj, fm.get("action_args", "")),
                    "result": self._get_field(obj, fm.get("action_result", "")),
                    "error": self._get_field(obj, fm.get("action_error", "")) or None,
                    "step_index": step_idx,
                }
                trace.actions.append(action)
                current_actions.append(action)

                # 若 action 自身已携带 result，无需等待单独 action_result
                if action["result"] and action["result"] not in ("", {}):
                    # 结果已到位
                    pass

            # --- 检索请求（knowledge_catalog / knowledge_search）---
            tool_name = self._get_field(obj, fm.get("action_name", ""))
            if tool_name in ("knowledge_catalog", "knowledge_search"):
                # 记录为检索事件（retrieval_happened + retrieval_has_results 断言依赖此字段）
                result_raw = self._get_field(obj, fm.get("action_result", ""))
                retrieval_entry = {
                    "query": self._get_field(obj, fm.get("action_args", "")),
                    "step_index": step_idx,
                    "tool": tool_name,
                    "chunks": [],
                }
                # 如果 action 自带 result，从中提取 sources/chunks
                if result_raw and result_raw not in ("", {}):
                    self._fill_retrieval_chunks(retrieval_entry, result_raw)
                trace.retrievals.append(retrieval_entry)

            # --- 错误 ---
            if etype in et.get("error", []):
                trace.meta.setdefault("errors", []).append(obj)

        # 最后一个 step
        if current_actions:
            trace.steps.append({
                "index": step_idx,
                "actions": current_actions,
            })

        # --- 最终输出：收集 agent / text 事件中的文本 ---
        final_cfg = fm.get("final_output", {})
        if isinstance(final_cfg, dict) and final_cfg.get("event"):
            target_sse = final_cfg.get("event", "")
            target_field = final_cfg.get("field", "")
            accumulate = final_cfg.get("accumulate", False)
            parts = []
            for sse_type, raw_data in events:
                obj = self._extract_json(raw_data)
                if obj is None:
                    continue
                if sse_type == target_sse:
                    text = obj.get(target_field, "")
                    if text:
                        parts.append(text)
                elif "type" in final_cfg:
                    # fallback: match by JSON type field
                    if obj.get("type") == final_cfg["type"]:
                        text = obj.get(target_field, "")
                        if text:
                            parts.append(text)
            if accumulate:
                trace.final_output = "".join(parts)
            else:
                trace.final_output = parts[-1] if parts else ""
        else:
            # 回退：正则提取
            pattern = self._ce.get("full_text_pattern")
            if pattern:
                matches = re.findall(pattern, raw_body)
                trace.final_output = "".join(matches)

        # 没有 action 但有 chart 数据时，把 chart 事件记为一个 action
        if not trace.actions:
            for sse_type, raw_data in events:
                obj = self._extract_json(raw_data)
                if obj and obj.get("type") in ("bazi-chart", "ziwei-chart"):
                    trace.actions.append({
                        "name": obj.get("type", ""),
                        "arguments": obj.get("payload", {}),
                        "result": None,
                        "error": None,
                        "step_index": 0,
                    })
                    if not trace.steps:
                        trace.steps.append({
                            "index": 0,
                            "actions": [trace.actions[-1]],
                        })
                    break

    def _fill_retrieval_chunks(self, retrieval_entry: dict, raw_result):
        """从 tool result 中提取 chunks。支持 passages / books / sources 等格式。"""
        # 解析 JSON 字符串
        if isinstance(raw_result, str):
            try:
                raw_result = json.loads(raw_result)
            except (json.JSONDecodeError, ValueError):
                retrieval_entry["chunks"] = [{"source": "", "content": raw_result[:500]}]
                return
        if not isinstance(raw_result, dict):
            retrieval_entry["chunks"] = [{"source": "", "content": str(raw_result)[:500]}]
            return
        # passages (knowledge_search)
        passages = raw_result.get("passages", [])
        if passages:
            for p in passages:
                retrieval_entry["chunks"].append({
                    "source": p.get("source", ""),
                    "content": p.get("content", ""),
                })
            return
        # books (knowledge_catalog)
        books = raw_result.get("books", [])
        if books:
            for b in books:
                retrieval_entry["chunks"].append({
                    "source": b.get("slug", b.get("name", "")),
                    "content": b.get("name", ""),
                })
            return
        # sources / results (通用)
        fm = self._fm
        sn = fm.get("retrieval_source_name", "source")
        cn = fm.get("retrieval_chunk_content", "chunk")
        sources = raw_result.get("sources", raw_result.get("results", []))
        if isinstance(sources, list):
            for s in sources:
                if isinstance(s, dict):
                    retrieval_entry["chunks"].append({
                        "source": s.get(sn, ""),
                        "content": s.get(cn, ""),
                    })
                elif isinstance(s, str):
                    retrieval_entry["chunks"].append({"source": "", "content": s[:300]})

    def _split_sse(self, body: str) -> list:
        """解析 SSE 格式，返回 (event_type, raw_data) 列表。"""
        events = []
        current_event_type = ""
        for line in body.split("\n"):
            if line.startswith("event:"):
                current_event_type = line[6:].strip()
            elif line.startswith("data:"):
                data = line[5:].strip()
                if data and data != "[DONE]":
                    events.append((current_event_type, data))
        return events

    def _extract_json(self, raw: str) -> dict | None:
        """安全地从字符串提取第一个 JSON 对象，容忍尾部多余数据。"""
        try:
            start = raw.find("{")
            if start == -1:
                return None
            # raw_decode 处理尾部多余数据
            decoder = json.JSONDecoder()
            obj, _ = decoder.raw_decode(raw, start)
            return obj
        except (json.JSONDecodeError, ValueError):
            return None
