import type { ChatMessage, TraceDigest } from '../types/chat'

export interface Passage {
  content: string
  source?: string
}

export interface ResultBlock {
  type: 'bazi-chart' | 'qimen-chart' | 'ziwei-chart'
  payload: unknown
}

export interface EvidenceGroup {
  source: string
  passages: string[]
}

export interface ProcessInfo {
  trace: TraceDigest
  status: string
  stepCount: number
}

export interface ToolCallInfo {
  name: string
  arguments?: string
}

export interface AssistantTurnViewModel {
  resultBlocks: ResultBlock[]
  answerBlocks: string[]
  process: ProcessInfo | null
  evidence: EvidenceGroup[] | null
  thoughts: string[]
  toolCalls: ToolCallInfo[]
  errors: string[]
}

export function buildAssistantTurnViewModel(message: ChatMessage): AssistantTurnViewModel {
  const segments = message.segments ?? []

  const resultBlocks: ResultBlock[] = []
  const answerBlocks: string[] = []
  let process: ProcessInfo | null = null
  let rawPassages: Passage[] = []
  const thoughts: string[] = []
  const toolCalls: ToolCallInfo[] = []
  const errors: string[] = []

  for (const seg of segments) {
    if (seg.type === 'text') {
      answerBlocks.push(seg.content)
    } else if (seg.type === 'thinking') {
      thoughts.push(seg.text)
    } else if (seg.type === 'tool_call') {
      toolCalls.push({ name: seg.tool, arguments: seg.params ? JSON.stringify(seg.params) : undefined })
    } else if (seg.type === 'error') {
      errors.push(seg.message)
    } else if (seg.type === 'component') {
      switch (seg.componentType) {
        case 'bazi-chart':
          resultBlocks.push({ type: 'bazi-chart', payload: seg.payload })
          break
        case 'qimen-chart':
          resultBlocks.push({ type: 'qimen-chart', payload: seg.payload })
          break
        case 'ziwei-chart':
          resultBlocks.push({ type: 'ziwei-chart', payload: seg.payload })
          break
        case 'trace-panel':
          process = {
            trace: seg.payload as TraceDigest,
            status: (seg.payload as TraceDigest)?.status ?? 'ok',
            stepCount: (seg.payload as TraceDigest)?.steps?.length ?? 0,
          }
          break
        case 'knowledge-sources': {
          // Backend sends passages as a direct array, not wrapped in { passages: [...] }
          if (Array.isArray(seg.payload)) {
            rawPassages = seg.payload as Passage[]
          } else if (seg.payload && typeof seg.payload === 'object' && Array.isArray((seg.payload as any).passages)) {
            rawPassages = (seg.payload as any).passages as Passage[]
          }
          break
        }
      }
    }
  }

  const evidence = groupPassages(rawPassages)

  return {
    resultBlocks,
    answerBlocks,
    process,
    evidence,
    thoughts,
    toolCalls,
    errors,
  }
}

function groupPassages(passages: Passage[]): EvidenceGroup[] | null {
  if (!passages || passages.length === 0) return null

  const groupMap = new Map<string, string[]>()
  for (const p of passages) {
    const key = p.source || '未知来源'
    if (!groupMap.has(key)) {
      groupMap.set(key, [])
    }
    groupMap.get(key)!.push(p.content)
  }

  return Array.from(groupMap.entries()).map(([source, contents]) => ({
    source,
    passages: contents,
  }))
}
