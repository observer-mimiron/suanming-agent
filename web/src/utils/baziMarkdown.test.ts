import { describe, expect, it } from 'vitest'

import { formatBaziChartMarkdown } from './baziMarkdown'

describe('formatBaziChartMarkdown', () => {
  it('formats critical bazi fields and preserves remaining payload data', () => {
    const markdown = formatBaziChartMarkdown({
      birthday: '1990-05-20 08:00',
      lunarDate: '庚午年四月廿六',
      gender: '男',
      dayGan: '甲',
      dayGanWuxing: '木',
      pillars: [
        { name: '年柱', stem: '庚', branch: '午', shiShen: '七杀', naYin: '路旁土', hideGan: ['丁', '己'] },
        { name: '月柱', stem: '辛', branch: '巳', shiShen: '正官', naYin: '白蜡金', subShiShen: ['食神'] },
        { name: '日柱', stem: '甲', branch: '寅', shiShen: '日主', naYin: '大溪水' },
        { name: '时柱', stem: '丙', branch: '子', shiShen: '食神', naYin: '涧下水' },
      ],
      wuxing: { 木: 2, 火: 3, 土: 1, 金: 1, 水: 1 },
      mingGong: '丑',
      shenGong: '寅',
      taiYuan: '壬申',
      dayun: [
        { startAge: 1, endAge: 10, ganZhi: '壬午' },
        { startAge: 11, endAge: 20, ganZhi: '癸未' },
      ],
      yongshen: {
        geju: '建禄格',
        geju_status: '成格',
        geju_basis: '月令得令，日主通根',
        geju_detail: '比劫旺，食神泄秀',
        geju_qing_zhuo: '浊中有清',
        geju_combination: '[主] 食神制杀',
        yong_shen: ['火', '土'],
        xi_shen: ['木'],
        ji_shen: ['金', '水'],
      },
      dayun_analyzed: {
        summary: '前运平稳，中运渐起',
        dayun_analyzed: [
          {
            startAge: 1,
            endAge: 10,
            ganZhi: '壬午',
            quality: '偏吉',
            quality_base: '大吉',
            tenGod: '偏印',
            quality_reason: {
              summary: '天干基础倾向稳定，地支没有明显冲克。',
            },
          },
        ],
      },
      liunian: {
        liunian_year: 2026,
        liunian_ganzhi: '丙午',
        liunian_shi_shen: '正财',
        current_dayun: {
          startAge: 1,
          endAge: 10,
          ganZhi: '壬午',
        },
        target_year: 2026,
        summary: '有阶段性机会，但节奏偏快。',
      },
      shensha_summary: {
        by_pillar: {
          年柱: [{ name: '天乙贵人' }],
          时柱: [{ name: '桃花' }, { name: '福神' }],
        },
      },
      custom_extra_signal: {
        note: '需要保留在附录中',
      },
    })

    expect(markdown).toContain('# 八字命盘')
    expect(markdown).toContain('## 基本信息')
    expect(markdown).toContain('- 出生：1990-05-20 08:00')
    expect(markdown).toContain('## 格局与用神')
    expect(markdown).toContain('- 格局：建禄格')
    expect(markdown).toContain('- 格局成败：成格')
    expect(markdown).toContain('- 清浊：浊中有清')
    expect(markdown).toContain('- 用神：火、土')
    expect(markdown).toContain('- 喜神：木')
    expect(markdown).toContain('- 忌神：金、水')
    expect(markdown).toContain('## 四柱')
    expect(markdown).toContain('### 年柱')
    expect(markdown).toContain('- 干支：庚午')
    expect(markdown).toContain('## 大运')
    expect(markdown).toContain('- 1-10岁：壬午')
    expect(markdown).toContain('## 大运分析')
    expect(markdown).toContain('### 壬午（1-10岁）')
    expect(markdown).toContain('- 综合评价：偏吉')
    expect(markdown).toContain('- 基础倾向：大吉')
    expect(markdown).toContain('- 十神主题：偏印')
    expect(markdown).toContain('- 说明：天干基础倾向稳定，地支没有明显冲克。')
    expect(markdown).not.toContain('- dayun_analyzed:')
    expect(markdown).toContain('## 流年分析')
    expect(markdown).toContain('- 流年：2026（丙午）')
    expect(markdown).toContain('- 流年十神：正财')
    expect(markdown).toContain('- 所在大运：壬午（1-10岁）')
    expect(markdown).toContain('- 提示：有阶段性机会，但节奏偏快。')
    expect(markdown).toContain('## 神煞摘要')
    expect(markdown).toContain('### 年柱')
    expect(markdown).toContain('- 天乙贵人')
    expect(markdown).toContain('### 时柱')
    expect(markdown).toContain('- 桃花、福神')
    expect(markdown).toContain('## 附加字段')
    expect(markdown).toContain('custom_extra_signal')
    expect(markdown).toContain('需要保留在附录中')
  })
})
