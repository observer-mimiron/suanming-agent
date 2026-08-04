import { beforeEach, describe, expect, it } from "vitest";
import { createSSEChunkParser, useSSE } from "./useSSE";

describe("useSSE", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("startNewSession creates a fresh session and clears current state", () => {
    const sse = useSSE();
    const originalSessionId = sse.sessionId.value;

    sse.messages.value.push({
      id: "m1",
      role: "assistant",
      segments: [{ type: "text", content: "旧对话" }],
    });
    sse.restoredPrompt.value = "继续旧问题";

    sse.startNewSession();

    expect(sse.sessionId.value).not.toBe(originalSessionId);
    expect(window.localStorage.getItem("suanming-agent.session-id")).toBe(
      sse.sessionId.value,
    );
    expect(sse.messages.value).toEqual([]);
    expect(sse.restoredPrompt.value).toBe("");
  });

  it("works when crypto.randomUUID is unavailable", () => {
    const originalCrypto = globalThis.crypto;
    Object.defineProperty(globalThis, "crypto", {
      configurable: true,
      value: {
        getRandomValues(array: Uint8Array) {
          for (let index = 0; index < array.length; index += 1) {
            array[index] = index + 1;
          }
          return array;
        },
      },
    });

    try {
      const sse = useSSE();
      expect(sse.sessionId.value).toBeTruthy();
      expect(window.localStorage.getItem("suanming-agent.session-id")).toBe(
        sse.sessionId.value,
      );
    } finally {
      Object.defineProperty(globalThis, "crypto", {
        configurable: true,
        value: originalCrypto,
      });
    }
  });

  it("parses event and data split across chunks", () => {
    const events: Array<{ event: string; data: any }> = [];
    const warnings: string[] = [];
    const parser = createSSEChunkParser(
      (evt) => events.push(evt),
      (warning) => warnings.push(warning),
    );

    parser.push('event: comp');
    parser.push('onent\ndata: {"type":"run-inspection","payload":{"trace_id":"t1"}}\n');
    parser.push('event: done\n');
    parser.push('data: {}');
    parser.finish();

    expect(events).toEqual([
      {
        event: "component",
        data: { type: "run-inspection", payload: { trace_id: "t1" } },
      },
      { event: "done", data: {} },
    ]);
    expect(warnings).toEqual([]);
  });

  it("records parse warnings instead of silently swallowing invalid data", () => {
    const events: Array<{ event: string; data: any }> = [];
    const warnings: string[] = [];
    const parser = createSSEChunkParser(
      (evt) => events.push(evt),
      (warning) => warnings.push(warning),
    );

    parser.push("event: component\n");
    parser.push("data: {bad json}\n");

    expect(events).toEqual([]);
    expect(warnings).toHaveLength(1);
    expect(warnings[0]).toContain("无法解析 SSE component 数据");
  });
});
