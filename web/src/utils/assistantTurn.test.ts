import { describe, expect, it } from 'vitest'
import { buildAssistantTurnViewModel } from './assistantTurn'
import type { ChatMessage } from '../types/chat'

describe('buildAssistantTurnViewModel', () => {
  it('merges debug-trace and execution-tree runtime payloads', () => {
    const message: ChatMessage = {
      id: 'assistant-1',
      role: 'assistant',
      segments: [
        {
          type: 'component',
          componentType: 'debug-trace',
          payload: {
            trace_id: 't1',
            status: 'ok',
            total_ms: 12,
            runtime: { primary_domain: 'bazi', decision_source: 'cheap_followup_reuse', gate_reason: 'cheap_followup_reuse' },
            steps: [],
          },
        },
        {
          type: 'component',
          componentType: 'execution-tree',
          payload: {
            trace_id: 't1',
            turn_type: 'agent_reading',
            status: 'ok',
            total_ms: 12,
            runtime: { task_intent: 'direct_bazi' },
            root: {
              label: 'chat.turn',
              kind: 'AGENT',
              status: 'ok',
              ms: 12,
              children: [],
            },
          },
        },
      ],
    }

    const vm = buildAssistantTurnViewModel(message)
    expect(vm.debugTrace?.runtime?.task_intent).toBe('direct_bazi')
    expect(vm.debugTrace?.runtime?.decision_source).toBe('cheap_followup_reuse')
    expect(vm.debugTrace?.root?.label).toBe('chat.turn')
  })
})
