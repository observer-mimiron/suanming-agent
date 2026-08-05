export interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  segments: Segment[];
  transportInspection?: TransportInspection;
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

/** QimenOwnerRef 标识问事盘所属的 Case 资产。 */
export interface QimenOwnerRef {
  kind: string;
  id: string;
}

/** QimenCell 描述奇门九宫中的结构化盘面事实。 */
export interface QimenCell {
  palace: string;
  god?: string;
  star?: string;
  door?: string;
  guest_gan?: string;
  host_gan?: string;
}

/** QimenChartPayload 是后端奇门 Case 盘及其展示元信息的前端合同。 */
export interface QimenChartPayload {
  purpose?: string;
  case_id?: string;
  owner_ref?: QimenOwnerRef;
  question_time?: string;
  time_source?: string;
  symbol_system?: string;
  pan_schema?: string;
  ju_text?: string;
  duty_text?: string;
  duty_star_palace?: string;
  duty_door_palace?: string;
  duty_palace?: string;
  cells?: QimenCell[];
}

export interface RunInspection {
  trace_id: string;
  session_id: string;
  status: "ok" | "degraded" | "fallback" | "error" | string;
  turn_type: string;
  total_ms: number;
  summary: RunSummary;
  diagnostics: RunDiagnostic[];
  spans: RunSpan[];
}

export interface RunSummary {
  primary_domain?: string;
  task_intent?: string;
  decision_source?: string;
  gate_reason?: string;
  inspection_text: string;
}

export interface RunDiagnostic {
  severity: "info" | "warn" | "error" | string;
  stage: string;
  code: string;
  title: string;
  evidence?: string[];
  next_action?: string;
  span_id?: string;
}

export interface RunSpan {
  span_id: string;
  parent_span_id?: string;
  name: string;
  label: string;
  kind: string;
  category: string;
  status: string;
  duration_ms: number;
  error?: string;
  attributes?: Record<string, any>;
}

export interface TransportInspection {
  doneReceived: boolean;
  componentTypesReceived: string[];
  parseWarnings: string[];
  requestError?: string;
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
