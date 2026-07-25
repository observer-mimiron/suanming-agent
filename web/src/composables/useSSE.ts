import { ref } from "vue";
import type {
  ChatMessage,
  Segment,
  SessionSegmentSnapshot,
  SessionSnapshot,
} from "../types/chat";

const SESSION_STORAGE_KEY = "suanming-agent.session-id";

function generateSessionId(): string {
  const cryptoApi = globalThis.crypto;
  if (cryptoApi?.randomUUID) {
    return cryptoApi.randomUUID();
  }
  if (cryptoApi?.getRandomValues) {
    const bytes = new Uint8Array(16);
    cryptoApi.getRandomValues(bytes);
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    const hex = Array.from(bytes, (value) =>
      value.toString(16).padStart(2, "0"),
    );
    return `${hex.slice(0, 4).join("")}-${hex.slice(4, 6).join("")}-${hex.slice(6, 8).join("")}-${hex.slice(8, 10).join("")}-${hex.slice(10, 16).join("")}`;
  }
  return `sess-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

function resolveSessionId(): string {
  if (typeof window === "undefined") {
    return generateSessionId();
  }
  const existing = window.localStorage.getItem(SESSION_STORAGE_KEY)?.trim();
  if (existing) return existing;
  const next = generateSessionId();
  window.localStorage.setItem(SESSION_STORAGE_KEY, next);
  return next;
}

function replaceSessionId(next: string): string {
  if (typeof window !== "undefined") {
    window.localStorage.setItem(SESSION_STORAGE_KEY, next);
  }
  return next;
}

function mapSessionSegment(seg: SessionSegmentSnapshot): Segment | null {
  if (seg.type === "text" && seg.content) {
    return { type: "text", content: seg.content };
  }
  if (seg.type === "thinking" && seg.text) {
    return { type: "thinking", text: seg.text, agent: seg.agent ?? "" };
  }
  if (seg.type === "tool_call" && seg.tool) {
    return {
      type: "tool_call",
      tool: seg.tool,
      params: seg.params ?? {},
      result: seg.result,
    };
  }
  if (seg.type === "component" && seg.component_type) {
    return {
      type: "component",
      componentType: seg.component_type,
      payload: seg.payload,
    };
  }
  if (seg.type === "error" && seg.message) {
    return { type: "error", message: seg.message };
  }
  return null;
}

function mapSnapshotToMessages(snapshot: SessionSnapshot): ChatMessage[] {
  const history: ChatMessage[] = (snapshot.messages ?? []).map((msg, index) => ({
    id: `history-${index}-${msg.role}`,
    role: msg.role,
    segments: [{ type: "text", content: msg.content } as Segment],
  }));

  const restoredSegments = (snapshot.segments ?? [])
    .map(mapSessionSegment)
    .filter((seg): seg is Segment => seg !== null);

  if (restoredSegments.length === 0) {
    return history;
  }

  const lastAssistantIdx = [...history]
    .map((m, i) => ({ m, i }))
    .reverse()
    .find(({ m }) => m.role === "assistant")?.i;
  if (lastAssistantIdx == null) {
    history.push({
      id: `history-restored-assistant`,
      role: "assistant",
      segments: restoredSegments,
    });
    return history;
  }

  history[lastAssistantIdx] = {
    ...history[lastAssistantIdx],
    segments: restoredSegments,
  };
  return history;
}

export function useSSE() {
  const messages = ref<ChatMessage[]>([]);
  const isLoading = ref(false);
  const sessionId = ref(resolveSessionId());
  const restoredPrompt = ref("");

  async function restoreSession() {
    try {
      const resp = await fetch(
        `/api/session/${encodeURIComponent(sessionId.value)}`,
      );
      if (!resp.ok) return;
      const snapshot = (await resp.json()) as SessionSnapshot;
      messages.value = mapSnapshotToMessages(snapshot);

      const lastInput = snapshot.last_input;
      const execution = snapshot.execution;
      restoredPrompt.value =
        lastInput?.question_text ||
        lastInput?.target_subject ||
        execution?.task_intent ||
        "";
    } catch {
      // ignore restore failures; chatting should still work with a fresh session
    }
  }

  function sendMessage(content: string) {
    const msgs = messages.value;
    msgs.push({
      id: Date.now().toString(),
      role: "user",
      segments: [{ type: "text", content }],
    });
    const assistIdx = msgs.length;
    msgs.push({
      id: (Date.now() + 1).toString(),
      role: "assistant",
      segments: [],
    });
    isLoading.value = true;

    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/chat");
    xhr.setRequestHeader("Content-Type", "application/json");
    let prevLen = 0;
    let timer: ReturnType<typeof setInterval> | null = null;

    const parseChunk = () => {
      const raw = xhr.responseText;
      if (!raw || raw.length <= prevLen) return;
      const chunk = raw.substring(prevLen);
      prevLen = raw.length;

      let currentEvent = "";
      const lines = chunk.split("\n");
      const target = msgs[assistIdx];
      for (const line of lines) {
        if (!line) continue;
        if (line.startsWith("event: ")) {
          currentEvent = line.slice(7).trim();
          continue;
        }
        if (!line.startsWith("data: ")) continue;
        try {
          const data = JSON.parse(line.slice(6));
          if (currentEvent === "thinking") {
            target.segments.push({
              type: "thinking",
              text: data.text,
              agent: data.agent,
            });
          } else if (currentEvent === "tool_call") {
            target.segments.push({
              type: "tool_call",
              tool: data.tool,
              params: data.params,
              result: data.result,
            });
          } else if (currentEvent === "component") {
            target.segments.push({
              type: "component",
              componentType: data.type,
              payload: data.payload,
            });
          } else if (currentEvent === "error") {
            target.segments.push({ type: "error", message: data.message });
          } else if (currentEvent === "text") {
            const segs = target.segments;
            const lastIdx = segs.length - 1;
            if (lastIdx >= 0 && segs[lastIdx].type === "text") {
              segs[lastIdx] = {
                ...segs[lastIdx],
                content: segs[lastIdx].content + data.content,
              };
            } else {
              segs.push({ type: "text", content: data.content });
            }
          }
        } catch {
          /* skip */
        }
      }
    };

    xhr.onreadystatechange = () => {
      if (xhr.readyState === 2) {
        // Start polling as soon as headers are received
        timer = setInterval(parseChunk, 50);
      }
      if (xhr.readyState === 4) {
        if (timer) {
          clearInterval(timer);
          timer = null;
        }
        parseChunk();
        isLoading.value = false;
      }
    };

    xhr.onerror = () => {
      if (timer) {
        clearInterval(timer);
        timer = null;
      }
      msgs[assistIdx].segments.push({ type: "error", message: "网络请求失败" });
      isLoading.value = false;
    };

    xhr.send(JSON.stringify({ message: content, session_id: sessionId.value }));
  }

  function startNewSession() {
    if (isLoading.value) return;
    sessionId.value = replaceSessionId(generateSessionId());
    messages.value = [];
    restoredPrompt.value = "";
  }

  return {
    messages,
    isLoading,
    sendMessage,
    sessionId,
    restoredPrompt,
    restoreSession,
    startNewSession,
  };
}
