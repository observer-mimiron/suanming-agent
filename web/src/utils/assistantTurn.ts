import type { ChatMessage, DebugEvent, DebugTraceDigest, ProcessDigest } from '../types/chat'

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
  digest: ProcessDigest
  status: string
  phaseCount: number
}

export interface AssistantTurnViewModel {
  resultBlocks: ResultBlock[]
  answerBlocks: string[]
  process: ProcessInfo | null
  debugTrace: DebugTraceDigest | null
  debugEvents: DebugEvent[]
  evidence: EvidenceGroup[] | null
  errors: string[]
}

export function buildAssistantTurnViewModel(message: ChatMessage): AssistantTurnViewModel {
  const segments = message.segments ?? []

  const resultBlocks: ResultBlock[] = []
  const answerBlocks: string[] = []
  let process: ProcessInfo | null = null
  let debugTrace: DebugTraceDigest | null = null
  let rawPassages: Passage[] = []
  const debugEvents: DebugEvent[] = []
  const errors: string[] = []

  for (const seg of segments) {
    if (seg.type === 'text') {
      answerBlocks.push(seg.content)
    } else if (seg.type === 'thinking') {
      debugEvents.push({ type: 'thinking', label: `思考 · ${seg.agent}`, preview: seg.text.slice(0, 200), agent: seg.agent })
    } else if (seg.type === 'tool_call') {
      const result = seg.result || ''
      const resultDisplay = result.length > 300 ? result.slice(0, 300) + '...(' + result.length + '字符已截断)' : result
      debugEvents.push({
        type: 'tool_call',
        label: `工具 · ${seg.tool}`,
        preview: seg.params ? JSON.stringify(seg.params).slice(0, 200) : '',
        result: resultDisplay,
      })
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
        case 'process-panel':
          process = {
            digest: seg.payload as ProcessDigest,
            status: (seg.payload as ProcessDigest)?.status ?? 'ok',
            phaseCount: (seg.payload as ProcessDigest)?.phases?.length ?? 0,
          }
          break
        case 'debug-trace':
          debugTrace = seg.payload as DebugTraceDigest
          break
        case 'execution-tree':
          // execution-tree payload has { root: ExecutionNode, trace_id, ... },
          // root is merged into debugTrace for DebugTracePanel unified rendering
          debugTrace = seg.payload as DebugTraceDigest
          // payload.root is received by DebugTraceDigest's new root field
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
    debugTrace,
    debugEvents,
    evidence,
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
