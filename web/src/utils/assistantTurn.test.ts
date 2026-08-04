import { describe, expect, it } from 'vitest'
import { buildAssistantTurnViewModel } from './assistantTurn'
import type { ChatMessage } from '../types/chat'

describe('buildAssistantTurnViewModel', () => {
  it('extracts route-decision, run-inspection, thinking, and transport inspection', () => {
    const message: ChatMessage = {
      id: 'assistant-1',
      role: 'assistant',
      transportInspection: {
        doneReceived: true,
        componentTypesReceived: ['run-inspection'],
        parseWarnings: [],
      },
      segments: [
        {
          type: 'thinking',
          agent: 'bazi_graph',
          text: '正在核对静态合同。',
        },
        {
          type: 'component',
          componentType: 'route-decision',
          payload: { primary_domain: 'bazi' },
        },
        {
          type: 'component',
          componentType: 'run-inspection',
          payload: {
            trace_id: 'trc_1',
            session_id: 'sess_1',
            status: 'ok',
            turn_type: 'agent_reading',
            total_ms: 10,
            summary: { inspection_text: '本轮运行未发现确定性异常。' },
            diagnostics: [],
            spans: [],
          },
        },
      ],
    }

    const vm = buildAssistantTurnViewModel(message)
    expect(vm.routeDecision).toEqual({ primary_domain: 'bazi' })
    expect(vm.runInspection?.trace_id).toBe('trc_1')
    expect(vm.transportInspection?.doneReceived).toBe(true)
    expect(vm.thinkingEvents[0]?.preview).toBe('正在核对静态合同。')
  })

})
