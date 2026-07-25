export interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  segments: Segment[];
}
export type Segment =
  | { type: "text"; content: string }
  | { type: "thinking"; text: string; agent: string }
  | {
      type: "tool_call";
      tool: string;
      params: Record<string, any>;
      result?: string;
    }
  | { type: "component"; componentType: string; payload: any }
  | { type: "error"; message: string };

export interface ProcessPhase {
  key: string;
  label: string;
  ms: number;
  status: "ok" | "degraded" | "fallback" | "error";
  summary?: string;
  meta?: {
    model?: string;
    hits?: number;
    artifact_present?: boolean;
    guardrail_result?: string;
  };
}

export interface RuntimeDigestMeta {
  primary_domain?: string;
  domains?: string[];
  task_intent?: string;
  required_artifacts?: string[];
  execution_mode?: string;
  gate_reason?: string;
  followup_policy?: string;
  decision_source?: string;
  reuse_cached_result?: boolean;
  reuse_session_profile?: boolean;
  needs_clarification?: boolean;
}

export interface ProcessDigest {
  trace_id?: string;
  turn_type?: string;
  status: "ok" | "degraded" | "fallback" | "error";
  total_ms: number;
  runtime?: RuntimeDigestMeta;
  phases: ProcessPhase[];
}

export interface DebugTraceStep {
  name: string;
  label: string;
  kind: string;
  ms: number;
  status: "ok" | "degraded" | "fallback" | "error";
  meta?: Record<string, any>;
}

export interface DebugTraceDigest {
  trace_id?: string;
  turn_type?: string;
  status: "ok" | "degraded" | "fallback" | "error";
  total_ms: number;
  runtime?: RuntimeDigestMeta;
  steps?: DebugTraceStep[]; // legacy flat format; optional when root is present
  root?: ExecutionNode; // unified execution tree (from execution-tree component event)
}

export interface DebugEvent {
  type: "thinking" | "tool_call";
  label: string;
  preview: string;
  agent?: string;
  result?: string;
}

export interface ExecutionNode {
  label: string;
  kind: string;
  status: string;
  ms: number;
  meta?: Record<string, any>;
  children?: ExecutionNode[];
}

export interface ExecutionTree {
  trace_id: string;
  turn_type: string;
  total_ms: number;
  status: string;
  runtime?: RuntimeDigestMeta;
  root: ExecutionNode;
}

export interface SessionMessageSnapshot {
  role: "user" | "assistant";
  content: string;
}

export interface SessionSegmentSnapshot {
  type: "text" | "thinking" | "tool_call" | "component" | "error";
  content?: string;
  text?: string;
  agent?: string;
  tool?: string;
  params?: Record<string, any>;
  result?: string;
  component_type?: string;
  payload?: any;
  message?: string;
}

export interface LastInputSnapshot {
  preferred_domain?: string;
  secondary_domains?: string[];
  explicit_method?: string;
  consult_mode?: string;
  time_scope?: string;
  target_subject?: string;
  question_text?: string;
  guidance_active?: boolean;
  guidance_directive_kind?: string;
}

export interface ExecutionSnapshot {
  primary_domain?: string;
  secondary_domains?: string[];
  domains?: string[];
  task_intent?: string;
  conversation_intent?: string;
  required_artifacts?: string[];
  needs_clarification?: boolean;
  qimen_mode?: string;
  target_subject?: string;
  time_scope?: string;
  gate?: {
    admitted?: boolean;
    reason?: string;
    allowed_domains?: string[];
    profile_requirement?: string;
    reuse_session_profile?: boolean;
    reuse_cached_result?: boolean;
    execution_mode?: string;
    guidance_policy?: string;
    followup_policy?: string;
  };
}

export interface SessionSnapshot {
  session_id: string;
  messages: SessionMessageSnapshot[];
  segments?: SessionSegmentSnapshot[];
  last_input?: LastInputSnapshot;
  execution?: ExecutionSnapshot;
}
