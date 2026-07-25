import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import TracePanel from './TracePanel.vue'

describe('TracePanel', () => {
  it('renders phase summary from process digest', () => {
    const wrapper = mount(TracePanel, {
      props: {
        digest: {
          status: 'ok',
          total_ms: 1200,
          phases: [
            {
              key: 'policy',
              label: '资料与策略校验',
              status: 'ok',
              ms: 300,
              summary: '已完成资料校验。',
            },
          ],
        },
      },
    })

    expect(wrapper.text()).toContain('资料与策略校验')
    expect(wrapper.text()).toContain('已完成资料校验。')
  })
})
