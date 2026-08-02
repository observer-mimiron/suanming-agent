<template>
  <div class="welcome">
    <div class="brand">
      <h1>命理大师</h1>
      <p class="subtitle">AI 八字命理咨询</p>
    </div>
    <div class="quick-asks">
      <button v-for="q in prompts" :key="q.label" class="ask-card" @click="$emit('ask', q.text)">
        <div class="ask-icon" v-html="q.icon"></div>
        <div class="ask-text">
          <span class="ask-label">{{ q.label }}</span>
          <span class="ask-sub">{{ q.sub }}</span>
        </div>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
defineEmits<{ ask: [text: string] }>()

const prompts = [
  {
    label: '算八字',
    sub: '四柱·十神·大运',
    icon: `<svg width="28" height="28" viewBox="0 0 28 28" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"><circle cx="14" cy="10" r="8"/><path d="M8 14 L6 22 L14 18 L22 22 L20 14"/></svg>`,
    text: '帮我算一下八字',
  },
  {
    label: '排大运',
    sub: '流年·起运·岁运',
    icon: `<svg width="28" height="28" viewBox="0 0 28 28" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"><polygon points="14,2 26,8 26,20 14,26 2,20 2,8"/><line x1="14" y1="26" x2="14" y2="16"/><polyline points="2,8 14,14 26,8"/></svg>`,
    text: '帮我排大运',
  },
  {
    label: '五行分析',
    sub: '旺衰·喜用·调候',
    icon: `<svg width="28" height="28" viewBox="0 0 28 28" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"><circle cx="14" cy="14" r="10"/><path d="M14 4a7 7 0 0 1 0 20"/><path d="M14 4a3 3 0 0 0 0 20"/></svg>`,
    text: '分析一下我的五行',
  },
]
</script>

<style scoped>
.welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 32px;
}
.brand { text-align: center; }
.brand h1 {
  font-family: var(--serif);
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 8px;
}
.subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin: 0;
}
.quick-asks {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  max-width: 340px;
}
.ask-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 18px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  color: var(--text-primary);
  cursor: pointer;
  text-align: left;
  transition: transform 0.18s, border-color 0.18s, box-shadow 0.18s;
  font-family: inherit;
  font-size: inherit;
}
.ask-card:hover {
  transform: translateY(-2px);
  border-color: var(--accent);
  box-shadow: var(--shadow-card);
}
.ask-icon {
  color: var(--accent-dim);
  display: flex;
  align-items: center;
  flex-shrink: 0;
}
.ask-label {
  font-size: 14px;
  font-weight: 600;
  display: block;
}
.ask-sub {
  font-size: 11px;
  color: var(--text-muted);
  display: block;
  margin-top: 2px;
}
</style>
