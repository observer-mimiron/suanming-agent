<template>
  <div class="chat-shell" :class="{ 'is-empty': messages.length === 0 && !isLoading }">
    <!-- Empty state: centered -->
    <div v-if="messages.length === 0 && !isLoading" class="empty-center">
      <WelcomePanel @ask="handleSend" />
      <div class="input-row empty-input">
        <n-input
          v-model:value="inputText"
          placeholder="请输入出生年月日时..."
          @keydown.enter="handleSend"
          size="large"
        >
          <template #suffix>
            <button
              class="send-btn"
              :class="{ active: inputText.trim() }"
              :disabled="!inputText.trim()"
              @click="handleSend"
            >
              <ArrowUp :size="18" />
            </button>
          </template>
        </n-input>
      </div>
    </div>

    <!-- Chat state -->
    <template v-else>
      <div class="chat-body">
        <n-scrollbar ref="scrollRef" class="messages">
          <template v-for="msg in messages" :key="msg.id">
            <AssistantTurn
              v-if="msg.role === 'assistant'"
              :message="msg"
              :isLoading="isLoading && msg === messages[messages.length - 1]"
            />
            <ChatBubble v-else :message="msg" />
          </template>
        </n-scrollbar>
      </div>
      <div class="input-row chat-input">
        <n-input
          v-model:value="inputText"
          placeholder="继续提问..."
          :disabled="isLoading"
          @keydown.enter="handleSend"
          size="large"
        >
          <template #suffix>
            <button
              class="send-btn"
              :class="{ active: inputText.trim() }"
              :disabled="!inputText.trim() || isLoading"
              @click="handleSend"
            >
              <ArrowUp :size="18" />
            </button>
          </template>
        </n-input>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { NScrollbar, NInput } from 'naive-ui'
import { ArrowUp } from 'lucide-vue-next'
import ChatBubble from './ChatBubble.vue'
import AssistantTurn from './AssistantTurn.vue'
import WelcomePanel from './WelcomePanel.vue'
import { useSSE } from '../composables/useSSE'

const { messages, isLoading, sendMessage } = useSSE()
const inputText = ref('')
const scrollRef = ref()

async function handleSend(t?: string) {
  const text = (typeof t === 'string' ? t : inputText.value).trim()
  if (!text || isLoading.value) return
  inputText.value = ''
  await sendMessage(text)
  await nextTick()
  scrollRef.value?.scrollTo({ top: 999999, behavior: 'smooth' })
}
</script>

<style scoped>
.chat-shell {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--bg);
}
.chat-shell.is-empty {
  justify-content: center;
}
.empty-center {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 32px;
  padding: 24px;
}
.empty-input {
  width: 100%;
  max-width: 520px;
}
.empty-input :deep(.n-input) {
  --n-height: 52px;
  --n-border-radius: 16px;
  --n-bg-color: var(--bg-secondary);
  --n-border-color: var(--border);
  --n-text-color: var(--text-primary);
  --n-placeholder-color: var(--text-muted);
}
.chat-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  align-items: center;
}
.messages {
  flex: 1;
  width: 100%;
  max-width: 680px;
  padding: 24px 24px 0;
}
.input-row {
  width: 100%;
  max-width: 680px;
  padding: 16px 24px;
}
.chat-input {
  border-top: 1px solid var(--border);
}
.chat-input :deep(.n-input) {
  --n-bg-color: var(--bg-secondary);
  --n-border-color: var(--border);
  --n-text-color: var(--text-primary);
  --n-placeholder-color: var(--text-muted);
  --n-border-radius: 14px;
}
.send-btn {
  width: 34px; height: 34px;
  border-radius: 10px;
  border: 1px solid #d4ccc0;
  background: transparent;
  color: #b0a89c;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
  flex-shrink: 0;
}
.send-btn.active {
  border-color: var(--accent);
  color: var(--accent);
  background: rgba(184,149,106,0.08);
}
.send-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--accent-dim);
}
.send-btn:disabled {
  opacity: 0.3;
  cursor: default;
}
</style>
