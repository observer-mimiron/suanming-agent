import { describe, expect, it } from 'vitest'
import { buildAssistantTurnViewModel } from './assistantTurn'
import type { ChatMessage } from '../types/chat'

describe('buildAssistantTurnViewModel', () => {
  it('groups structured result blocks (bazi-chart, qimen-chart)', () => {
    const msg: ChatMessage = {
      id: '1',
      role: 'assistant',
      segments: [
        { type: 'component', componentType: 'bazi-chart', payload: { dayGan: '甲' } },
        { type: 'component', componentType: 'qimen-chart', payload: { method: '时家奇门' } },
        { type: 'text', content: '这是一段解释' },
      ],
    }
    const vm = buildAssistantTurnViewModel(msg)
    expect(vm.resultBlocks).toHaveLength(2)
    expect(vm.resultBlocks[0].type).toBe('bazi-chart')
    expect(vm.resultBlocks[1].type).toBe('qimen-chart')
    expect(vm.answerBlocks).toEqual(['这是一段解释'])
  })

  it('merges multiple text segments into ordered answer blocks', () => {
    const msg: ChatMessage = {
      id: '2',
      role: 'assistant',
      segments: [
        { type: 'text', content: '第一段' },
        { type: 'text', content: '第二段' },
      ],
    }
    const vm = buildAssistantTurnViewModel(msg)
    expect(vm.answerBlocks).toEqual(['第一段', '第二段'])
    expect(vm.resultBlocks).toHaveLength(0)
  })

  it('groups knowledge passages by source label', () => {
    const msg: ChatMessage = {
      id: '3',
      role: 'assistant',
      segments: [
        {
          type: 'component',
          componentType: 'knowledge-sources',
          // Backend sends passages as a direct array, not { passages: [...] }
          payload: [
            { content: '渊海子平内容', source: '渊海子平' },
            { content: '三命通会内容1', source: '三命通会' },
            { content: '渊海子平另一段', source: '渊海子平' },
            { content: '三命通会内容2', source: '三命通会' },
          ],
        },
      ],
    }
    const vm = buildAssistantTurnViewModel(msg)
    expect(vm.evidence).toHaveLength(2)

    const yhzp = vm.evidence!.find(g => g.source === '渊海子平')
    expect(yhzp).toBeDefined()
    expect(yhzp!.passages).toHaveLength(2)

    const smth = vm.evidence!.find(g => g.source === '三命通会')
    expect(smth).toBeDefined()
    expect(smth!.passages).toHaveLength(2)
  })

  it('preserves trace digest as process section', () => {
    const msg: ChatMessage = {
      id: '4',
      role: 'assistant',
      segments: [
        {
          type: 'component',
          componentType: 'process-panel',
          payload: {
            trace_id: 'abc',
            turn_type: 'full',
            total_ms: 2500,
            status: 'ok',
            phases: [
              { key: 'route', label: '路由判断', status: 'ok', ms: 500, summary: '已完成路由判断。' },
              { key: 'prepare', label: '排盘与资料准备', status: 'ok', ms: 300, summary: '已完成排盘与资料准备。' },
            ],
          },
        },
        {
          type: 'component',
          componentType: 'debug-trace',
          payload: {
            trace_id: 'abc',
            turn_type: 'full',
            total_ms: 2500,
            status: 'ok',
            steps: [
              { name: 'supervisor_decision', label: '路由决策', kind: 'CHAIN', status: 'ok', ms: 500 },
            ],
          },
        },
      ],
    }
    const vm = buildAssistantTurnViewModel(msg)
    expect(vm.process).not.toBeNull()
    expect(vm.process!.status).toBe('ok')
    expect(vm.process!.phaseCount).toBe(2)
    expect(vm.process!.digest.total_ms).toBe(2500)
    expect(vm.debugTrace?.steps).toHaveLength(1)
  })

  it('moves thinking and tool_calls into debug events', () => {
    const msg: ChatMessage = {
      id: '5',
      role: 'assistant',
      segments: [
        { type: 'thinking', text: '思考中...', agent: 'deepseek' },
        { type: 'tool_call', tool: 'bazi_calc', params: { birthday: '1990-01-01' } },
        { type: 'error', message: 'knowledge search timeout' },
      ],
    }
    const vm = buildAssistantTurnViewModel(msg)
    expect(vm.debugEvents).toHaveLength(2)
    expect(vm.debugEvents[0].type).toBe('thinking')
    expect(vm.debugEvents[0].preview).toBe('思考中...')
    expect(vm.debugEvents[1].type).toBe('tool_call')
    expect(vm.debugEvents[1].label).toContain('bazi_calc')
    expect(vm.errors).toEqual(['knowledge search timeout'])
  })

  it('handles empty segments', () => {
    const msg: ChatMessage = { id: '6', role: 'assistant', segments: [] }
    const vm = buildAssistantTurnViewModel(msg)
    expect(vm.resultBlocks).toHaveLength(0)
    expect(vm.answerBlocks).toHaveLength(0)
    expect(vm.process).toBeNull()
    expect(vm.evidence).toBeNull()
  })

  it('uses "未知来源" for passages without source', () => {
    const msg: ChatMessage = {
      id: '7',
      role: 'assistant',
      segments: [
        {
          type: 'component',
          componentType: 'knowledge-sources',
          payload: [
            { content: 'test content' },
          ],
        },
      ],
    }
    const vm = buildAssistantTurnViewModel(msg)
    expect(vm.evidence).toHaveLength(1)
    expect(vm.evidence![0].source).toBe('未知来源')
  })

  it('returns null evidence when no passages', () => {
    const msg: ChatMessage = {
      id: '8',
      role: 'assistant',
      segments: [
        { type: 'component', componentType: 'knowledge-sources', payload: [] },
      ],
    }
    const vm = buildAssistantTurnViewModel(msg)
    expect(vm.evidence).toBeNull()
  })
})
