export interface ChatMessage {
  id: string; role: 'user' | 'assistant'; segments: Segment[]
}
export type Segment =
  | { type: 'text'; content: string }
  | { type: 'thinking'; text: string; agent: string }
  | { type: 'tool_call'; tool: string; params: Record<string,any>; result?: string }
  | { type: 'component'; componentType: string; payload: any }
  | { type: 'error'; message: string }

export interface ProcessPhase {
  key: string;
  label: string;
  ms: number;
  status: 'ok' | 'degraded' | 'fallback' | 'error';
  summary?: string;
  meta?: {
    model?: string;
    hits?: number;
    artifact_present?: boolean;
    guardrail_result?: string;
  };
}

export interface ProcessDigest {
  trace_id?: string;
  turn_type?: string;
  status: 'ok' | 'degraded' | 'fallback' | 'error';
  total_ms: number;
  phases: ProcessPhase[];
}

export interface DebugTraceStep {
  name: string;
  label: string;
  kind: string;
  ms: number;
  status: 'ok' | 'degraded' | 'fallback' | 'error';
  meta?: Record<string, any>;
}

export interface DebugTraceDigest {
  trace_id?: string;
  turn_type?: string;
  status: 'ok' | 'degraded' | 'fallback' | 'error';
  total_ms: number;
  steps: DebugTraceStep[];
}

export interface DebugEvent {
  type: 'thinking' | 'tool_call';
  label: string;
  preview: string;
  agent?: string;
  result?: string;
}

export interface ExecutionNode {
  label: string
  kind: string
  status: string
  ms: number
  meta?: Record<string, any>
  children?: ExecutionNode[]
}

export interface ExecutionTree {
  trace_id: string
  turn_type: string
  total_ms: number
  status: string
  root: ExecutionNode
}
