import {mount} from '@vue/test-utils'
import {beforeEach, describe, expect, it, vi} from 'vitest'

import QimenChart from './QimenChart.vue'
import type {QimenChartPayload} from '../types/chat'

const payload: QimenChartPayload = {
  purpose: 'event_question',
  case_id: 'case-1',
  owner_ref: { kind: 'case', id: 'case-1' },
  question_time: '2026-08-05T14:30:00+08:00',
  time_source: 'question_time',
  symbol_system: 'eight_gate_eight_god',
  pan_schema: 'rotating_8',
  ju_text: '阳遁三局',
  cells: [{ palace: '坎', god: '太常', door: '中门', star: '天蓬' }],
}

describe('QimenChart', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('shows Case metadata and warns when rotating_8 contains forbidden symbols', () => {
    const wrapper = mount(QimenChart, { props: { data: payload } })

    expect(wrapper.find('.qm-case-meta').text()).toContain('event_question')
    expect(wrapper.find('.qm-case-meta').text()).toContain('case/case-1')
    expect(wrapper.find('.qm-case-meta').text()).toContain('question_time')
    expect(wrapper.find('.qm-case-meta').text()).toContain('eight_gate_eight_god')
    expect(wrapper.get('[role="alert"]').text()).toContain('rotating_8')
    expect(wrapper.get('[role="alert"]').text()).toContain('“中门”')
    expect(wrapper.get('[role="alert"]').text()).toContain('“太常”')
  })

  it('copies complete Case metadata and the warning as Markdown', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    })

    const wrapper = mount(QimenChart, { props: { data: payload } })
    await wrapper.get('.qm-copy-btn').trigger('click')

    const markdown = writeText.mock.calls[0][0] as string
    expect(markdown).toContain('- 问事目的：event_question')
    expect(markdown).toContain('- Case ID：case-1')
    expect(markdown).toContain('- 资产归属：case/case-1')
    expect(markdown).toContain('- 起局时间：2026-08-05T14:30:00+08:00')
    expect(markdown).toContain('- 起局来源：question_time')
    expect(markdown).toContain('- 符号体系：eight_gate_eight_god')
    expect(markdown).toContain('- 异常警告：盘式合同异常：rotating_8 payload 出现“中门”、“太常”')
  })
})
