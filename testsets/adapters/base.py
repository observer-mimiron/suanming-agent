"""适配器基类。"""

import yaml
import sys
from pathlib import Path

# 确保 suites 目录在 path 中
_suites_dir = Path(__file__).resolve().parent.parent / "suites"
if str(_suites_dir) not in sys.path:
    sys.path.insert(0, str(_suites_dir))

from canonical import CanonicalTrace


class TraceAdapter:
    """将 agent 原始响应映射到 CanonicalTrace。

    映射规则由 YAML 配置驱动，子类实现 _parse()。
    """

    def __init__(self, config: dict):
        self.config = config
        self.name = config.get("name", "unknown")
        self.format = config.get("format", "raw")
        self._et = config.get("event_types", {})
        self._fm = config.get("field_mapping", {})
        self._ce = config.get("content_extraction", {})

    def convert(self, raw_body: str, http_code: str = "200") -> CanonicalTrace:
        trace = CanonicalTrace(raw_body=raw_body)
        trace.meta["http_code"] = http_code
        self._parse(raw_body, trace)
        trace.meta["format"] = self.format
        trace.meta["adapter"] = self.name
        if not trace.final_output:
            trace.meta.setdefault("warnings", []).append("final_output is empty")
        return trace

    def _parse(self, raw_body: str, trace: CanonicalTrace):
        raise NotImplementedError

    @staticmethod
    def load(config_path: str) -> "TraceAdapter":
        path = Path(config_path)
        config = yaml.safe_load(path.read_text())
        fmt = config.get("format", "sse")
        if fmt == "sse":
            from suanming_sse import SSEAdapter
            return SSEAdapter(config)
        raise ValueError(f"Unknown adapter format: {fmt}")
