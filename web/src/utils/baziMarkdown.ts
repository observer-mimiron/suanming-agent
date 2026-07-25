type JsonLike = Record<string, unknown>

const KNOWN_TOP_LEVEL_KEYS = new Set([
  'birthday',
  'lunarDate',
  'gender',
  'dayGan',
  'dayGanWuxing',
  'pillars',
  'wuxing',
  'mingGong',
  'mingGongNaYin',
  'shenGong',
  'shenGongNaYin',
  'taiYuan',
  'taiYuanNaYin',
  'dayun',
  'yongshen',
  'dayun_analyzed',
  'liunian',
  'shensha_summary',
])

function isRecord(value: unknown): value is JsonLike {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function asString(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'string') return value.trim()
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return ''
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value
    .map((item) => asString(item))
    .filter(Boolean)
}

function formatListLine(label: string, values: string[]): string | null {
  if (!values.length) return null
  return `- ${label}：${values.join('、')}`
}

function omitKeys(record: JsonLike, keys: string[]): JsonLike {
  const omitted = new Set(keys)
  return Object.fromEntries(Object.entries(record).filter(([key]) => !omitted.has(key)))
}

function formatValue(value: unknown, indent = 0): string[] {
  const pad = '  '.repeat(indent)

  if (Array.isArray(value)) {
    if (!value.length) return [`${pad}- （空）`]
    return value.flatMap((item) => {
      if (isRecord(item) || Array.isArray(item)) {
        return [`${pad}-`].concat(formatValue(item, indent + 1))
      }
      return [`${pad}- ${asString(item)}`]
    })
  }

  if (isRecord(value)) {
    const entries = Object.entries(value)
    if (!entries.length) return [`${pad}- （空）`]
    return entries.flatMap(([key, nested]) => {
      if (isRecord(nested) || Array.isArray(nested)) {
        return [`${pad}- ${key}:`].concat(formatValue(nested, indent + 1))
      }
      return [`${pad}- ${key}: ${asString(nested)}`]
    })
  }

  return [`${pad}- ${asString(value)}`]
}

function pushSection(lines: string[], title: string, content: Array<string | null | undefined>) {
  const filtered = content.filter((line): line is string => Boolean(line))
  if (!filtered.length) return
  if (lines.length) lines.push('')
  lines.push(title)
  lines.push(...filtered)
}

function formatNestedBlock(title: string, value: unknown): string[] {
  const lines = [`### ${title}`]
  lines.push(...formatValue(value))
  return lines
}

function formatDayunAnalysis(value: unknown): string[] {
  if (!isRecord(value)) return []

  const lines: string[] = []
  const summary = asString(value.summary)
  if (summary) lines.push(`- 总述：${summary}`)

  const entries = Array.isArray(value.dayun_analyzed) ? value.dayun_analyzed.filter(isRecord) : []
  for (const item of entries) {
    const title = `${asString(item.ganZhi)}（${asString(item.startAge)}-${asString(item.endAge)}岁）`
    lines.push('')
    lines.push(`### ${title}`)

    const details = [
      asString(item.quality) ? `- 综合评价：${asString(item.quality)}` : null,
      asString(item.quality_base) ? `- 基础倾向：${asString(item.quality_base)}` : null,
      asString(item.tenGod) ? `- 十神主题：${asString(item.tenGod)}` : null,
      asString(item.tenGodType) ? `- 十神类别：${asString(item.tenGodType)}` : null,
    ].filter((line): line is string => Boolean(line))
    lines.push(...details)

    const reason = isRecord(item.quality_reason) ? asString(item.quality_reason.summary) : ''
    if (reason) lines.push(`- 说明：${reason}`)

    const chonghe = Array.isArray(item.dayun_chonghe) ? item.dayun_chonghe.filter(isRecord) : []
    const descriptions = chonghe
      .map((entry) => asString(entry.description))
      .filter(Boolean)
    if (descriptions.length) lines.push(`- 作用关系：${descriptions.join('；')}`)

    const remaining = omitKeys(item, [
      'startAge',
      'endAge',
      'ganZhi',
      'quality',
      'quality_base',
      'tenGod',
      'tenGodType',
      'quality_reason',
      'dayun_chonghe',
    ])
    if (Object.keys(remaining).length) {
      lines.push(...formatNestedBlock('其他字段', remaining))
    }
  }

  const remainingTopLevel = omitKeys(value, ['summary', 'dayun_analyzed'])
  if (Object.keys(remainingTopLevel).length) {
    if (lines.length) lines.push('')
    lines.push(...formatNestedBlock('其他大运分析字段', remainingTopLevel))
  }

  return lines
}

function formatLiunianAnalysis(value: unknown): string[] {
  if (!isRecord(value)) return []

  const lines: string[] = []
  const year = asString(value.liunian_year || value.target_year)
  const ganzhi = asString(value.liunian_ganzhi)
  if (year || ganzhi) {
    lines.push(`- 流年：${year}${ganzhi ? `（${ganzhi}）` : ''}`)
  }
  if (asString(value.liunian_shi_shen)) lines.push(`- 流年十神：${asString(value.liunian_shi_shen)}`)
  if (asString(value.liunian_branch)) lines.push(`- 流年地支：${asString(value.liunian_branch)}`)

  const currentDayun = isRecord(value.current_dayun) ? value.current_dayun : null
  if (currentDayun) {
    const dayunText = `${asString(currentDayun.ganZhi)}（${asString(currentDayun.startAge)}-${asString(currentDayun.endAge)}岁）`
    lines.push(`- 所在大运：${dayunText}`)
  }

  const summary = asString(value.summary)
  if (summary) lines.push(`- 提示：${summary}`)

  const chonghe = Array.isArray(value.liunian_chonghe) ? value.liunian_chonghe.filter(isRecord) : []
  const descriptions = chonghe
    .map((entry) => asString(entry.description))
    .filter(Boolean)
  if (descriptions.length) lines.push(`- 作用关系：${descriptions.join('；')}`)

  const remaining = omitKeys(value, [
    'liunian_year',
    'target_year',
    'liunian_ganzhi',
    'liunian_shi_shen',
    'liunian_branch',
    'current_dayun',
    'summary',
    'liunian_chonghe',
  ])
  if (Object.keys(remaining).length) {
    if (lines.length) lines.push('')
    lines.push(...formatNestedBlock('其他流年字段', remaining))
  }

  return lines
}

function formatShenshaSummary(value: unknown): string[] {
  if (!isRecord(value)) return []

  const lines: string[] = []
  const all = asStringArray(value.all)
  if (all.length) lines.push(`- 全盘神煞：${all.join('、')}`)

  const byPillar = isRecord(value.by_pillar) ? value.by_pillar : null
  if (byPillar) {
    for (const [pillarName, rawItems] of Object.entries(byPillar)) {
      const items = Array.isArray(rawItems) ? rawItems.filter(isRecord) : []
      const names = items
        .map((item) => asString(item.name))
        .filter(Boolean)
      if (!names.length) continue
      lines.push('')
      lines.push(`### ${pillarName}`)
      lines.push(`- ${names.join('、')}`)
    }
  }

  return lines
}

export function formatBaziChartMarkdown(data: Record<string, unknown>): string {
  const lines: string[] = ['# 八字命盘']
  const yongshen = isRecord(data.yongshen) ? data.yongshen : {}
  const pillars = Array.isArray(data.pillars) ? data.pillars.filter(isRecord) : []
  const dayun = Array.isArray(data.dayun) ? data.dayun.filter(isRecord) : []
  const wuxing = isRecord(data.wuxing) ? data.wuxing : {}

  pushSection(lines, '## 基本信息', [
    asString(data.birthday) ? `- 出生：${asString(data.birthday)}` : null,
    asString(data.lunarDate) ? `- 农历：${asString(data.lunarDate)}` : null,
    asString(data.gender) ? `- 性别：${asString(data.gender)}` : null,
    asString(data.dayGan) ? `- 日主：${asString(data.dayGan)}` : null,
    asString(data.dayGanWuxing) ? `- 日主五行：${asString(data.dayGanWuxing)}` : null,
  ])

  pushSection(lines, '## 格局与用神', [
    asString(yongshen.geju) ? `- 格局：${asString(yongshen.geju)}` : null,
    asString(yongshen.geju_status) ? `- 格局成败：${asString(yongshen.geju_status)}` : null,
    asString(yongshen.geju_qing_zhuo) ? `- 清浊：${asString(yongshen.geju_qing_zhuo)}` : null,
    asString(yongshen.geju_basis) ? `- 格局依据：${asString(yongshen.geju_basis)}` : null,
    asString(yongshen.geju_detail) ? `- 格局细节：${asString(yongshen.geju_detail)}` : null,
    asString(yongshen.geju_combination) ? `- 格局组合：${asString(yongshen.geju_combination)}` : null,
    formatListLine('用神', asStringArray(yongshen.yong_shen)),
    formatListLine('喜神', asStringArray(yongshen.xi_shen)),
    formatListLine('忌神', asStringArray(yongshen.ji_shen)),
  ])

  if (pillars.length) {
    if (lines.length) lines.push('')
    lines.push('## 四柱')
    for (const pillar of pillars) {
      lines.push('')
      lines.push(`### ${asString(pillar.name) || '柱位'}`)
      lines.push(`- 干支：${asString(pillar.stem)}${asString(pillar.branch)}`)
      if (asString(pillar.shiShen)) lines.push(`- 十神：${asString(pillar.shiShen)}`)
      if (asString(pillar.naYin)) lines.push(`- 纳音：${asString(pillar.naYin)}`)
      if (asString(pillar.xunKong)) lines.push(`- 空亡：${asString(pillar.xunKong)}`)
      if (asString(pillar.diShi)) lines.push(`- 地势：${asString(pillar.diShi)}`)
      if (asString(pillar.xun)) lines.push(`- 旬：${asString(pillar.xun)}`)
      const hideGan = asStringArray(pillar.hideGan)
      if (hideGan.length) lines.push(`- 藏干：${hideGan.join('、')}`)
      const subShiShen = asStringArray(pillar.subShiShen)
      if (subShiShen.length) lines.push(`- 副星：${subShiShen.join('、')}`)
      const shensha = Array.isArray(pillar.shensha) ? pillar.shensha.filter(isRecord) : []
      if (shensha.length) {
        const shenshaText = shensha
          .map((item) => asString(item.name))
          .filter(Boolean)
          .join('、')
        if (shenshaText) lines.push(`- 神煞：${shenshaText}`)
      }
    }
  }

  pushSection(lines, '## 宫位信息', [
    asString(data.mingGong)
      ? `- 命宫：${asString(data.mingGong)}${asString(data.mingGongNaYin) ? `（${asString(data.mingGongNaYin)}）` : ''}`
      : null,
    asString(data.shenGong)
      ? `- 身宫：${asString(data.shenGong)}${asString(data.shenGongNaYin) ? `（${asString(data.shenGongNaYin)}）` : ''}`
      : null,
    asString(data.taiYuan)
      ? `- 胎元：${asString(data.taiYuan)}${asString(data.taiYuanNaYin) ? `（${asString(data.taiYuanNaYin)}）` : ''}`
      : null,
  ])

  if (Object.keys(wuxing).length) {
    pushSection(
      lines,
      '## 五行统计',
      Object.entries(wuxing).map(([key, value]) => `- ${key}：${asString(value)}`),
    )
  }

  if (dayun.length) {
    pushSection(
      lines,
      '## 大运',
      dayun.map((item) => `- ${asString(item.startAge)}-${asString(item.endAge)}岁：${asString(item.ganZhi)}`),
    )
  }

  pushSection(lines, '## 大运分析', formatDayunAnalysis(data.dayun_analyzed))

  pushSection(lines, '## 流年分析', formatLiunianAnalysis(data.liunian))

  pushSection(lines, '## 神煞摘要', formatShenshaSummary(data.shensha_summary))

  const appendixEntries = Object.entries(data).filter(([key]) => !KNOWN_TOP_LEVEL_KEYS.has(key))
  if (appendixEntries.length) {
    if (lines.length) lines.push('')
    lines.push('## 附加字段')
    for (const [key, value] of appendixEntries) {
      lines.push('')
      lines.push(...formatNestedBlock(key, value))
    }
  }

  return lines.join('\n')
}
