<template>
  <div class="ks-card" v-if="groups && groups.length">
    <div class="ks-header" @click="showSources = !showSources">
      <span>依据资料</span>
      <span class="ks-toggle-icon">{{ showSources ? '▾' : '▸' }}</span>
    </div>
    <div v-if="showSources" class="ks-body">
      <div v-for="(g, gi) in groups" :key="gi" class="ks-group">
        <div class="ks-source">{{ g.source }}</div>
        <div v-for="(p, pi) in g.passages" :key="pi" class="ks-passage">
          <blockquote>{{ p }}</blockquote>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { EvidenceGroup } from '../utils/assistantTurn'
defineProps<{ groups: EvidenceGroup[] }>()

const showSources = ref(false) // 默认折叠收起
</script>

<style scoped>
.ks-card {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-secondary);
  padding: 14px 16px;
  margin: 8px 0;
  transition: all 0.2s ease-in-out;
}
.ks-header {
  font-family: var(--serif);
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  user-select: none;
}
.ks-toggle-icon {
  font-size: 11px;
  color: var(--text-muted);
}
.ks-body {
  margin-top: 12px;
}
.ks-group {
  margin-bottom: 10px;
}
.ks-group:last-child { margin-bottom: 0; }
.ks-source {
  font-size: 12px;
  font-weight: 600;
  color: var(--accent-dim);
  margin-bottom: 4px;
}
.ks-passage blockquote {
  margin: 0 0 4px;
  padding-left: 12px;
  border-left: 2px solid var(--border);
  color: var(--text-secondary);
  font-size: 13px;
  font-style: italic;
  line-height: 1.5;
}
</style>
