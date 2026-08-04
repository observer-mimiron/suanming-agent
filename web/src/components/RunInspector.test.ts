import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import RunInspector from './RunInspector.vue'
import type { RunInspection } from '../types/chat'

function sampleInspection(): RunInspection {
  return {
    trace_id: 'trc_1234567890',
    session_id: 'sess_1',
    status: 'degraded',
    turn_type: 'agent_reading',
    total_ms: 1250,
    summary: {
      primary_domain: 'bazi',
      task_intent: 'direct_bazi',
      decision_source: 'supervisor',
      inspection_text: '八字合同发生修复或恢复策略介入。',
    },
    diagnostics: [
      {
        severity: 'warn',
        stage: 'agent',
        code: 'contract.repaired',
        title: '八字合同发生修复或恢复策略介入。',
        evidence: ['bazi.final.audit_result=repaired'],
        next_action: '查看八字内部 graph path。',
        span_id: 'spn_child',
      },
    ],
    spans: [
      {
        span_id: 'root',
        name: 'chat.turn',
        label: 'chat.turn',
        kind: 'AGENT',
        category: 'agent',
        status: 'ok',
        duration_ms: 1250,
        attributes: {
          primary_domain: 'bazi',
          task_intent: 'direct_bazi',
        },
      },
      {
        span_id: 'spn_child',
        parent_span_id: 'root',
        name: 'contract_gate',
        label: '结果验收',
        kind: 'CHAIN',
        category: 'guard',
        status: 'degraded',
        duration_ms: 30,
        attributes: {
          'bazi.final.audit_result': 'repaired',
          'bazi.contract.recovery_policy': 'dynamic_facts_only',
        },
      },
    ],
  }
}

describe('RunInspector', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders triage, span tree, and selected span detail', async () => {
    const wrapper = mount(RunInspector, {
      props: {
        inspection: sampleInspection(),
        transport: {
          doneReceived: true,
          componentTypesReceived: ['run-inspection'],
          parseWarnings: [],
        },
      },
    })

    expect(wrapper.text()).toContain('Run Inspector')
    expect(wrapper.text()).toContain('Agent Chain')
    expect(wrapper.text()).toContain('合同/护栏')
    expect(wrapper.text()).toContain('八字合同发生修复或恢复策略介入')
    expect(wrapper.text()).toContain('八字合同')
    expect(wrapper.text()).toContain('bazi.final.audit_result=repaired')
    expect(wrapper.text()).toContain('结果验收')
  })

  it('shows transport warnings without backend inspection', () => {
    const wrapper = mount(RunInspector, {
      props: {
        inspection: null,
        isLoading: false,
        transport: {
          doneReceived: false,
          componentTypesReceived: [],
          parseWarnings: ['无法解析 SSE component 数据：{bad json}'],
        },
      },
    })

    expect(wrapper.text()).toContain('无法解析 SSE component 数据')
    expect(wrapper.text()).toContain('前端未收到 done 事件')
  })

  it('loads raw trace lazily and redacts sensitive fields by default', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        trace_id: 'trc_1234567890',
        user_message: '完整用户输入',
        attributes: {
          'input.value': '原始输入',
          'output.value': '模型完整输出',
          model: 'deepseek-v4-flash',
        },
        spans: [],
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = mount(RunInspector, {
      props: {
        inspection: sampleInspection(),
        transport: {
          doneReceived: true,
          componentTypesReceived: ['run-inspection'],
          parseWarnings: [],
        },
      },
    })

    await wrapper.findAll('button').find((button) => button.text().includes('加载全量 Trace'))!.trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith('/api/debug/traces/trc_1234567890')
    expect(wrapper.text()).toContain('Raw Trace')
    expect(wrapper.text()).toContain('model')
    expect(wrapper.text()).toContain('[已折叠')
    expect(wrapper.text()).not.toContain('完整用户输入')

    await wrapper.find('input[type="checkbox"]').setValue(true)
    expect(wrapper.text()).toContain('完整用户输入')
  })
})
