"""规范 Trace 模型 — 与 agent 架构解耦的统一数据结构。

设计来源:
- agentverify ExecutionResult (steps, tool_calls, tool_results, final_output)
- agent-eval-harness Trajectory (turn_id, role, content, tool_calls)
- Claw-Eval 三通道证据 (trace + audit log + snapshot)
"""

from dataclasses import dataclass, field


@dataclass
class CanonicalTrace:
    """一次 agent turn 的完整执行轨迹。断言层唯一数据源。"""

    steps: list = field(default_factory=list)
    # steps[i] = {"index": int, "actions": list[dict]}

    actions: list = field(default_factory=list)
    # actions[i] = {"name": str, "arguments": dict, "result": any,
    #               "error": str|None, "step_index": int}

    retrievals: list = field(default_factory=list)
    # retrievals[i] = {"query": str, "chunks": [{"source": str, "content": str}],
    #                  "step_index": int}

    route: dict = field(default_factory=dict)
    # {"primary": str, "intent": str, "secondary": [str]}

    meta: dict = field(default_factory=dict)
    # {"http_code": str, "model": str, "duration_ms": float, "errors": list,
    #  "format": str, "adapter": str}

    final_output: str = ""

    raw_body: str = ""
    raw_events: list = field(default_factory=list)

    # ── 便捷方法 ──

    def step_count(self) -> int:
        return len(self.steps)

    def action_names(self) -> list:
        return [a.get("name", "") for a in self.actions]

    def action_by_name(self, name: str) -> dict | None:
        for a in self.actions:
            if a.get("name") == name:
                return a
        return None

    def retrieval_source_count(self) -> int:
        return sum(len(r.get("chunks", [])) for r in self.retrievals)

    def has_errors(self) -> bool:
        return bool(self.meta.get("errors", []))

    def has_retrievals(self) -> bool:
        return len(self.retrievals) > 0

    def __repr__(self):
        return (
            f"CanonicalTrace(steps={self.step_count()}, "
            f"actions={len(self.actions)}, "
            f"retrievals={len(self.retrievals)}, "
            f"route={self.route.get('primary', '?')})"
        )
