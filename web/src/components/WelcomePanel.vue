<template>
  <div class="welcome">
    <div class="brand">
      <div class="brand-icon" v-html="Icons.logo"></div>
      <h1>命理大师</h1>
      <p class="subtitle">AI 八字命理咨询</p>
    </div>
    <div class="quick-asks">
      <button v-for="q in prompts" :key="q.label" class="ask-btn" @click="$emit('ask', q.text)">
        <span class="ask-icon" v-html="q.icon"></span>
        <span>{{ q.label }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Icons } from './icons'

defineEmits<{ ask: [text: string] }>()

const prompts = [
  { icon: Icons.crystal, label: '帮我算八字', text: '帮我算一下八字' },
  { icon: Icons.calendar, label: '排大运', text: '帮我排大运' },
  { icon: Icons.yinyang, label: '五行分析', text: '分析一下我的五行' },
]
</script>

<style scoped>
.welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  padding: 40px 24px;
}

.brand { text-align: center; margin-bottom: 40px; }

.brand-icon {
  color: var(--accent);
  margin-bottom: 12px;
  display: flex;
  justify-content: center;
}

.brand-icon :deep(svg) {
  width: 40px;
  height: 40px;
}

.brand h1 {
  font-size: 28px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 8px;
}

.subtitle { color: var(--text-secondary); font-size: 15px; margin: 0; }

.quick-asks {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  max-width: 320px;
}

.ask-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 18px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 10px;
  color: var(--text-primary);
  font-size: 15px;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}
.ask-btn:hover { border-color: var(--accent); background: #2a2a2a; }

.ask-icon {
  display: flex;
  align-items: center;
  color: var(--accent);
}
</style>
