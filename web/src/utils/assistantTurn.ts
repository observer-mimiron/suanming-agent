import type {ChatMessage, QimenChartPayload, RunInspection, TransportInspection,} from '../types/chat'

export interface Passage {
  content: string
  source?: string
}

/** ResultBlock 保留后端组件类型，不承担语义路由或领域决策。 */
export type ResultBlock =
  | { type: 'bazi-chart'; payload: unknown }
  | { type: 'qimen-chart'; payload: QimenChartPayload }
  | { type: 'ziwei-chart'; payload: unknown }

export interface EvidenceGroup {
  source: string
  passages: string[]
}

export interface ThinkingEvent {
  label: string
  preview: string
  agent?: string
}

export interface AssistantTurnViewModel {
  resultBlocks: ResultBlock[]
  answerBlocks: string[]
  routeDecision: unknown | null
  runInspection: RunInspection | null
  transportInspection: TransportInspection | null
  thinkingEvents: ThinkingEvent[]
  evidence: EvidenceGroup[] | null
  errors: string[]
}

export function buildAssistantTurnViewModel(message: ChatMessage): AssistantTurnViewModel {
  const segments = message.segments ?? []

  const resultBlocks: ResultBlock[] = []
  const answerBlocks: string[] = []
  let routeDecision: unknown | null = null
  let runInspection: RunInspection | null = null
  let rawPassages: Passage[] = []
  const thinkingEvents: ThinkingEvent[] = []
  const errors: string[] = []

  for (const seg of segments) {
    if (seg.type === 'text') {
      answerBlocks.push(seg.content)
    } else if (seg.type === 'thinking') {
      thinkingEvents.push({ label: '思考 · ' + seg.agent, preview: seg.text.slice(0, 200), agent: seg.agent })
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
        case 'route-decision':
          routeDecision = seg.payload
          break
        case 'run-inspection':
          runInspection = seg.payload as RunInspection
          break
        case 'knowledge-sources':
          // Backend sends passages as a direct array, not wrapped in { passages: [...] }.
          if (Array.isArray(seg.payload)) {
            rawPassages = seg.payload as Passage[]
          } else if (seg.payload && typeof seg.payload === 'object' && Array.isArray((seg.payload as any).passages)) {
            rawPassages = (seg.payload as any).passages as Passage[]
          }
          break
      }
    }
  }

  const evidence = groupPassages(rawPassages)

  return {
    resultBlocks,
    answerBlocks,
    routeDecision,
    runInspection,
    transportInspection: message.transportInspection ?? null,
    thinkingEvents,
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
