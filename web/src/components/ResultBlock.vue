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
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-secondary);
  overflow: hidden;
  transition: box-shadow 0.2s;
}
.result-block:hover {
  box-shadow: var(--shadow-card);
}
.result-block__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px 0;
}
.result-block__title {
  font-family: var(--serif);
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
}
.result-block__body {
  padding: 14px 16px 16px;
}
</style>
