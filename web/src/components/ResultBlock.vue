<template>
  <div class="result-block">
    <header v-if="$slots.title || $slots.icon" class="result-block__header">
      <span v-if="$slots.icon" class="result-block__icon">
        <slot name="icon" />
      </span>
      <div v-if="$slots.title" class="result-block__title">
        <slot name="title" />
      </div>
    </header>
    <div class="result-block__body">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.result-block {
  margin: 0 0 14px;
  border: 1px solid var(--line-soft);
  border-radius: 10px;
  background: var(--bg-card);
  overflow: hidden;
}

.result-block__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px 0;
}

.result-block__icon {
  display: flex;
  align-items: center;
  color: var(--accent);
  flex-shrink: 0;
}

.result-block__title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  letter-spacing: 0.02em;
}

.result-block__body {
  padding: 12px 16px 16px;
}
</style>
