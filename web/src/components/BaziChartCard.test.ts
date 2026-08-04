import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import BaziChartCard from './BaziChartCard.vue'

describe('BaziChartCard', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('labels lunar date separately and shows the actual four pillars in the header', () => {
    const wrapper = mount(BaziChartCard, {
      props: {
        data: {
          dayGan: '甲',
          dayGanWuxing: '木',
          lunarDate: '乙巳年丁亥月癸未日',
          pillars: [
            { name: '年柱', stem: '乙', branch: '巳', shiShen: '劫财', naYin: '佛灯火' },
            { name: '月柱', stem: '丁', branch: '亥', shiShen: '伤官', naYin: '屋上土' },
            { name: '日柱', stem: '甲', branch: '申', shiShen: '日主', naYin: '泉中水' },
            { name: '时柱', stem: '甲', branch: '子', shiShen: '比肩', naYin: '海中金' },
          ],
        },
      },
    })

    const headerText = wrapper.find('.bz-hero').text().replace(/\s+/g, ' ').trim()
    expect(headerText).toContain('日主 甲 · 木')
    expect(headerText).toContain('四柱 乙巳 · 丁亥 · 甲申 · 甲子')
    expect(headerText).toContain('农历 乙巳年丁亥月癸未日')
  })

  it('marks heavenly-stem and earthly-branch ten-god labels inline', () => {
    const wrapper = mount(BaziChartCard, {
      props: {
        data: {
          dayGan: '甲',
          dayGanWuxing: '木',
          pillars: [
            {
              name: '年柱',
              stem: '戊',
              branch: '辰',
              shiShen: '偏财',
              naYin: '大林木',
              hideGan: ['戊', '乙', '癸'],
              subShiShen: ['偏财', '劫财', '正印'],
            },
          ],
        },
      },
    })

    const marks = wrapper.findAll('.bz-god-mark').map((item) => item.text())
    expect(marks).toEqual(['偏财', '偏财'])
    expect(wrapper.find('.branch-main').text()).toBe('偏财')
    const hiddenPairs = wrapper.findAll('.bz-hidden-pair').map((item) => item.text())
    expect(hiddenPairs).toEqual(['戊偏财', '乙劫财', '癸正印'])
  })

  it('copies formatted markdown with geju information and shows copied feedback', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    })

    const wrapper = mount(BaziChartCard, {
      props: {
        data: {
          birthday: '1990-05-20 08:00',
          dayGan: '甲',
          dayGanWuxing: '木',
          lunarDate: '庚午年四月廿六',
          pillars: [
            { name: '年柱', stem: '庚', branch: '午', shiShen: '七杀', naYin: '路旁土' },
            { name: '月柱', stem: '辛', branch: '巳', shiShen: '正官', naYin: '白蜡金' },
            { name: '日柱', stem: '甲', branch: '寅', shiShen: '日主', naYin: '大溪水' },
            { name: '时柱', stem: '丙', branch: '子', shiShen: '食神', naYin: '涧下水' },
          ],
          yongshen: {
            geju: '建禄格',
            geju_qing_zhuo: '浊中有清',
            yong_shen: ['火', '土'],
          },
        },
      },
    })

    const button = wrapper.get('[data-testid="copy-bazi-markdown"]')
    await button.trigger('click')

    expect(writeText).toHaveBeenCalledTimes(1)
    const copiedText = writeText.mock.calls[0][0]
    expect(copiedText).toContain('# 八字命盘')
    expect(copiedText).toContain('- 格局：建禄格')
    expect(copiedText).toContain('- 清浊：浊中有清')
    expect(copiedText).toContain('- 用神：火、土')
    expect(button.text()).toContain('已复制')
  })

  it('prefers backend-provided current_dayun when highlighting the timeline', () => {
    const wrapper = mount(BaziChartCard, {
      props: {
        data: {
          birthday: '2025-11-10 23:00',
          dayGan: '甲',
          dayGanWuxing: '木',
          dayun: [
            { startAge: 2, endAge: 11, ganZhi: '丙戌' },
            { startAge: 12, endAge: 21, ganZhi: '乙酉' },
          ],
          liunian: {
            liunian_year: 2026,
            current_dayun: { startAge: 2, endAge: 11, ganZhi: '丙戌' },
          },
        },
      },
    })

    const activeNode = wrapper.find('.bz-timeline-node.active')
    expect(activeNode.exists()).toBe(true)
    expect(activeNode.text()).toContain('丙戌')
  })
})
