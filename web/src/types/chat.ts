export interface ChatMessage {
  id: string; role: 'user' | 'assistant'; segments: Segment[]
}
export type Segment =
  | { type: 'text'; content: string }
  | { type: 'thinking'; text: string; agent: string }
  | { type: 'tool_call'; tool: string; params: Record<string,any> }
  | { type: 'component'; componentType: string; payload: any }
  | { type: 'error'; message: string }
