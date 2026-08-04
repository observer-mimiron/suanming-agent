<!--
  ZiweiChartCard renders the Zi Wei Dou Shu chart surface.
  It owns the twelve-palace visual layout and star tags only; all chart facts
  are supplied by the backend Zi Wei deterministic calculator.
-->
<template>
  <div class="zw-card">
    <header class="zw-header">
      <div>
        <div class="zw-kicker">紫微斗数</div>
        <div class="zw-title">十二宫星曜盘</div>
        <div class="zw-meta">
          <span v-if="data.gender">{{ data.gender }}</span>
          <span v-if="data.lunar_date">{{ data.lunar_date }}</span>
          <span v-if="data.five_elements_class">{{ data.five_elements_class }}</span>
        </div>
      </div>

      <div v-if="fourPillarEntries.length" class="zw-pillars" aria-label="四柱">
        <span v-for="item in fourPillarEntries" :key="item.key" class="zw-pillar-chip">
          {{ item.key }} {{ item.value }}
        </span>
      </div>
    </header>

    <section class="zw-board" aria-label="紫微十二宫">
      <article
        v-for="(p, i) in palaces"
        :key="p.index ?? i"
        class="zw-palace"
        :class="[
          palacePosition(i),
          {
            'is-life': p.name === '命宫',
            'is-body': p.is_body_palace,
            'is-origin': p.is_original_palace,
          },
        ]"
        :style="{ animationDelay: i * 36 + 'ms' }"
      >
        <div class="zw-palace-head">
          <div>
            <div class="zw-palace-name">{{ p.name || '未命名宫' }}</div>
            <div class="zw-palace-branch">{{ p.heavenly_stem || '' }}{{ p.earthly_branch || '' }}</div>
          </div>
          <div class="zw-palace-tags">
            <span v-if="p.name === '命宫'" class="zw-tag life">命</span>
            <span v-if="p.is_body_palace" class="zw-tag body">身</span>
            <span v-if="p.is_original_palace" class="zw-tag origin">因</span>
          </div>
        </div>

        <div class="zw-main-stars">
          <span v-if="!p.major_stars?.length" class="zw-empty-star">无主星</span>
          <span
            v-for="star in p.major_stars || []"
            :key="star.name"
            class="zw-star major"
            :class="brightnessClass(star.brightness)"
          >
            <strong>{{ star.name }}</strong>
            <em v-if="star.brightness">{{ star.brightness }}</em>
            <i v-if="star.mutagen" :class="['zw-mutagen', mutagenClass(star.mutagen)]">
              {{ mutagenShort(star.mutagen) }}
            </i>
          </span>
        </div>

        <div v-if="p.minor_stars?.length || p.adjective_stars?.length" class="zw-support-stars">
          <span
            v-for="star in supportStars(p)"
            :key="star.type + '-' + star.name"
            class="zw-star support"
          >
            {{ star.name }}
            <i v-if="star.mutagen" :class="['zw-mutagen', mutagenClass(star.mutagen)]">
              {{ mutagenShort(star.mutagen) }}
            </i>
          </span>
        </div>

        <footer class="zw-palace-foot">
          <span v-if="p.changsheng_12">{{ p.changsheng_12 }}</span>
          <span v-if="p.boshi_12">{{ p.boshi_12 }}</span>
          <span v-if="p.decadal">大限 {{ p.decadal.start_age }}-{{ p.decadal.end_age }}岁</span>
        </footer>
      </article>

      <aside class="zw-center">
        <div class="zw-center-title">命盘核心</div>
        <div class="zw-core-grid">
          <div class="zw-core-item">
            <span>命宫</span>
            <strong>{{ data.soul_palace_ganzhi || data.soul_palace_branch || '—' }}</strong>
          </div>
          <div class="zw-core-item">
            <span>身宫</span>
            <strong>{{ data.body_palace_branch || '—' }}</strong>
          </div>
          <div class="zw-core-item">
            <span>命主</span>
            <strong>{{ data.soul_master || '—' }}</strong>
          </div>
          <div class="zw-core-item">
            <span>身主</span>
            <strong>{{ data.body_master || '—' }}</strong>
          </div>
        </div>
        <div class="zw-center-note">
          主星看格局骨架，四化看触发点，大限看阶段。
        </div>
      </aside>
    </section>

    <div v-if="palaces.length" class="zw-legend">
      <span class="zw-legend-item"><i class="zw-legend-dot life"></i> 命宫</span>
      <span class="zw-legend-item"><i class="zw-legend-dot body"></i> 身宫</span>
      <span class="zw-legend-item"><i class="zw-legend-dot origin"></i> 来因宫</span>
      <span class="zw-legend-item"><i class="zw-legend-dot transform"></i> 禄 / 权 / 科 / 忌 为四化</span>
    </div>

    <div class="zw-actions">
      <button type="button" class="zw-copy-btn" @click="copyMarkdown">
        {{ copied ? '已复制' : '复制命盘' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{ data: any }>()

const palaces = computed<any[]>(() => props.data?.palaces || [])
const fourPillarEntries = computed(() =>
  Object.entries(props.data?.four_pillars || {}).map(([key, value]) => ({ key, value })),
)
const copied = ref(false)

// palacePosition places the twelve palaces around a square board perimeter.
function palacePosition(index: number): string {
  return 'zw-pos-' + (index % 12)
}

// supportStars merges smaller star groups while preserving their source type.
function supportStars(palace: any) {
  const minor = (palace.minor_stars || []).map((star: any) => ({ ...star, type: 'minor' }))
  const adjective = (palace.adjective_stars || []).map((star: any) => ({ ...star, type: 'adjective' }))
  return [...minor, ...adjective].slice(0, 8)
}

// brightnessClass maps star brightness to visual emphasis.
function brightnessClass(brightness?: string): string {
  if (!brightness) return 'zm-br-default'
  const map: Record<string, string> = {
    庙: 'zm-br-miao',
    旺: 'zm-br-wang',
    得: 'zm-br-de',
    利: 'zm-br-li',
    平: 'zm-br-ping',
    不: 'zm-br-bu',
    陷: 'zm-br-xian',
  }
  return map[brightness] || 'zm-br-default'
}

// mutagenShort renders Four Transformations as compact palace badges.
function mutagenShort(mutagen: string): string {
  const map: Record<string, string> = {
    化禄: '禄',
    化权: '权',
    化科: '科',
    化忌: '忌',
  }
  return map[mutagen] || mutagen
}

// mutagenClass color-codes the Four Transformations.
function mutagenClass(mutagen: string): string {
  const map: Record<string, string> = {
    化禄: 'zm-mu-lu',
    化权: 'zm-mu-quan',
    化科: 'zm-mu-ke',
    化忌: 'zm-mu-ji',
  }
  return map[mutagen] || ''
}

// copyMarkdown copies the complete Zi Wei chart as readable Markdown.
async function copyMarkdown() {
  await navigator.clipboard.writeText(formatZiweiMarkdown(props.data || {}))
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 2000)
}

// formatZiweiMarkdown keeps the copied chart readable without another formatter file.
function formatZiweiMarkdown(data: any): string {
  const lines = ['# 紫微斗数命盘', '']
  lines.push('## 基本信息')
  lines.push('- 性别：' + field(data.gender))
  lines.push('- 农历：' + field(data.lunar_date))
  lines.push('- 公历：' + field(data.solar_date))
  lines.push('- 五行局：' + field(data.five_elements_class))
  lines.push('- 命宫：' + field(data.soul_palace_ganzhi || data.soul_palace_branch))
  lines.push('- 身宫：' + field(data.body_palace_branch))
  lines.push('- 命主：' + field(data.soul_master))
  lines.push('- 身主：' + field(data.body_master))

  const pillars = Object.entries(data.four_pillars || {})
  if (pillars.length) {
    lines.push('', '## 四柱')
    for (const [key, value] of pillars) lines.push('- ' + key + '：' + field(value))
  }

  lines.push('', '## 十二宫')
  for (const p of data.palaces || []) {
    const tags = [
      p.name === '命宫' ? '命宫' : '',
      p.is_body_palace ? '身宫' : '',
      p.is_original_palace ? '来因宫' : '',
    ].filter(Boolean)
    lines.push('### ' + field(p.name) + '（' + field((p.heavenly_stem || '') + (p.earthly_branch || '')) + '）')
    if (tags.length) lines.push('- 标记：' + tags.join('、'))
    lines.push('- 主星：' + starsText(p.major_stars))
    if (p.minor_stars?.length) lines.push('- 辅曜：' + starsText(p.minor_stars))
    if (p.adjective_stars?.length) lines.push('- 杂曜：' + starsText(p.adjective_stars))
    if (p.changsheng_12 || p.boshi_12) {
      lines.push('- 长生/博士：' + [p.changsheng_12, p.boshi_12].filter(Boolean).join('、'))
    }
    if (p.decadal) lines.push('- 大限：' + field(p.decadal.start_age) + '-' + field(p.decadal.end_age) + '岁')
  }
  return lines.join('\n')
}

// starsText renders star arrays with brightness and Four Transformation tags.
function starsText(stars: any[]): string {
  if (!stars?.length) return '—'
  return stars
    .map((star) => {
      const marks = [star.brightness, star.mutagen].filter(Boolean)
      return star.name + (marks.length ? '（' + marks.join('，') + '）' : '')
    })
    .join('、')
}

// field normalizes absent copied fields to an em dash.
function field(value: unknown): string {
  if (value === undefined || value === null || value === '') return '—'
  return String(value)
}
</script>

<style scoped>
.zw-card {
  text-align: left;
  max-width: 100%;
  padding: 18px;
  border: 1px solid rgba(115, 92, 68, 0.14);
  border-radius: 20px;
  background:
    linear-gradient(180deg, rgba(255, 251, 244, 0.95), rgba(248, 243, 233, 0.9)),
    radial-gradient(circle at top right, rgba(184, 149, 106, 0.16), transparent 38%);
  font-size: 13px;
}

.zw-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 16px;
}

.zw-kicker {
  color: #8b745f;
  font-size: 11px;
  letter-spacing: 0.14em;
}

.zw-title {
  margin-top: 2px;
  font-family: var(--serif);
  color: #4d3928;
  font-size: 22px;
  font-weight: 800;
}

.zw-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 4px;
  color: #8b745f;
  font-size: 12px;
}

.zw-pillars {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 7px;
}

.zw-pillar-chip {
  padding: 5px 9px;
  border: 1px solid rgba(125, 92, 63, 0.12);
  border-radius: 999px;
  background: rgba(125, 92, 63, 0.08);
  color: #6d533e;
  font-size: 11px;
}

.zw-board {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  grid-template-rows: repeat(4, minmax(150px, auto));
  gap: 10px;
}

.zw-palace,
.zw-center {
  border: 1px solid rgba(125, 92, 63, 0.12);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.78);
  box-shadow: 0 8px 20px rgba(108, 84, 62, 0.06);
}

.zw-palace {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 150px;
  padding: 11px;
  opacity: 0;
  animation: zw-cell-in 0.34s cubic-bezier(0.22, 0.61, 0.36, 1) forwards;
}

.zw-pos-0 { grid-column: 1; grid-row: 1; }
.zw-pos-1 { grid-column: 2; grid-row: 1; }
.zw-pos-2 { grid-column: 3; grid-row: 1; }
.zw-pos-3 { grid-column: 4; grid-row: 1; }
.zw-pos-4 { grid-column: 4; grid-row: 2; }
.zw-pos-5 { grid-column: 4; grid-row: 3; }
.zw-pos-6 { grid-column: 4; grid-row: 4; }
.zw-pos-7 { grid-column: 3; grid-row: 4; }
.zw-pos-8 { grid-column: 2; grid-row: 4; }
.zw-pos-9 { grid-column: 1; grid-row: 4; }
.zw-pos-10 { grid-column: 1; grid-row: 3; }
.zw-pos-11 { grid-column: 1; grid-row: 2; }

.zw-center {
  grid-column: 2 / 4;
  grid-row: 2 / 4;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 12px;
  padding: 18px;
  background:
    radial-gradient(circle at center, rgba(184, 149, 106, 0.16), rgba(255, 255, 255, 0.8) 66%),
    rgba(255, 255, 255, 0.84);
}

@keyframes zw-cell-in {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}

.zw-palace.is-life {
  border-color: #b8956a;
  background: rgba(250, 244, 234, 0.94);
}

.zw-palace.is-body {
  box-shadow: inset 0 0 0 1px rgba(122, 158, 126, 0.38), 0 8px 20px rgba(108, 84, 62, 0.06);
}

.zw-palace.is-origin {
  outline: 1px dashed rgba(107, 138, 168, 0.5);
  outline-offset: -5px;
}

.zw-palace-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  padding-bottom: 7px;
  border-bottom: 1px solid rgba(125, 92, 63, 0.08);
}

.zw-palace-name {
  font-family: var(--serif);
  color: #543b28;
  font-size: 16px;
  font-weight: 800;
}

.zw-palace-branch {
  color: #8b745f;
  font-size: 11px;
}

.zw-palace-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.zw-tag,
.zw-mutagen {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  border-radius: 999px;
  font-size: 10px;
  font-style: normal;
  font-weight: 800;
}

.zw-tag.life {
  background: rgba(184, 149, 106, 0.16);
  color: #8a6a3c;
}

.zw-tag.body {
  background: rgba(122, 158, 126, 0.16);
  color: #55765a;
}

.zw-tag.origin {
  background: rgba(107, 138, 168, 0.16);
  color: #526f8c;
}

.zw-main-stars,
.zw-support-stars {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.zw-star {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  border-radius: 8px;
  line-height: 1.35;
}

.zw-star.major {
  padding: 4px 6px;
  background: rgba(125, 92, 63, 0.08);
  color: #5a3f2a;
}

.zw-star.major strong {
  font-size: 12px;
}

.zw-star.major em {
  color: #9a7350;
  font-size: 10px;
  font-style: normal;
}

.zw-star.support {
  padding: 2px 5px;
  background: rgba(255, 255, 255, 0.56);
  color: #6f6257;
  font-size: 10px;
}

.zw-empty-star {
  color: #9b8c7b;
  font-size: 11px;
}

.zm-br-miao,
.zm-br-wang {
  box-shadow: inset 0 0 0 1px rgba(184, 149, 106, 0.24);
}

.zm-br-xian {
  background: rgba(196, 122, 106, 0.12) !important;
}

.zm-mu-lu {
  background: rgba(122, 158, 126, 0.15);
  color: #4f7a55;
}

.zm-mu-quan {
  background: rgba(184, 149, 106, 0.16);
  color: #8a6a3c;
}

.zm-mu-ke {
  background: rgba(107, 138, 168, 0.15);
  color: #526f8c;
}

.zm-mu-ji {
  background: rgba(196, 122, 106, 0.15);
  color: #9a453a;
}

.zw-palace-foot {
  display: flex;
  flex-wrap: wrap;
  gap: 5px 8px;
  margin-top: auto;
  padding-top: 8px;
  border-top: 1px dashed rgba(125, 92, 63, 0.12);
  color: #8b745f;
  font-size: 10px;
}

.zw-center-title {
  font-family: var(--serif);
  color: #4d3928;
  font-size: 18px;
  font-weight: 800;
  text-align: center;
}

.zw-core-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 9px;
}

.zw-core-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 10px;
  border: 1px solid rgba(125, 92, 63, 0.1);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.56);
}

.zw-core-item span {
  color: #8b745f;
  font-size: 11px;
}

.zw-core-item strong {
  color: #4d3928;
  font-family: var(--serif);
  font-size: 15px;
}

.zw-center-note {
  color: #8b745f;
  text-align: center;
  font-size: 11px;
}

.zw-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 14px;
  margin-top: 14px;
  color: #8b745f;
  font-size: 11px;
}

.zw-legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.zw-legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  display: inline-block;
}

.zw-legend-dot.life { background: #b8956a; }
.zw-legend-dot.body { background: #7a9e7e; }
.zw-legend-dot.origin { background: #6b8aa8; }
.zw-legend-dot.transform { background: #c47a6a; }

.zw-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 18px;
}

.zw-copy-btn {
  border: 1px solid rgba(125, 92, 63, 0.16);
  background: rgba(255, 255, 255, 0.6);
  color: #6d533e;
  border-radius: 999px;
  padding: 8px 14px;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
  transition: background-color 0.2s ease, border-color 0.2s ease, color 0.2s ease;
}

.zw-copy-btn:hover {
  background: rgba(255, 255, 255, 0.88);
  border-color: #b8956a;
  color: #4d3928;
}

@media (max-width: 980px) {
  .zw-board {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    grid-template-rows: none;
  }

  .zw-palace,
  .zw-center {
    grid-column: auto;
    grid-row: auto;
  }

  .zw-center {
    order: -1;
    grid-column: 1 / -1;
  }
}

@media (max-width: 640px) {
  .zw-card {
    padding: 14px;
    border-radius: 16px;
  }

  .zw-header {
    flex-direction: column;
  }

  .zw-pillars {
    justify-content: flex-start;
  }

  .zw-board,
  .zw-core-grid {
    grid-template-columns: 1fr;
  }
}
</style>
