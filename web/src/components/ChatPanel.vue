<template>
  <div class="chat-shell">
    <!-- Empty state -->
    <WelcomePanel v-if="messages.length === 0 && !isLoading" @ask="handleSend" />

    <!-- Messages -->
    <div v-else class="chat-body">
      <n-scrollbar ref="scrollRef" class="messages">
        <template v-for="msg in messages" :key="msg.id">
          <AssistantTurn
            v-if="msg.role === 'assistant'"
            :message="msg"
            :isLoading="isLoading && msg === messages[messages.length - 1]"
          />
          <ChatBubble v-else :message="msg" />
        </template>
        <div v-if="isLoading" class="loading"><n-spin size="small" /></div>
      </n-scrollbar>
    </div>

    <!-- Input -->
    <div class="input-row">
      <n-input
        v-model:value="inputText"
        placeholder="请输入出生年月日时..."
        :disabled="isLoading"
        @keydown.enter="handleSend"
        size="large"
      >
        <template #suffix>
          <n-button type="primary" :disabled="!inputText.trim()" @click="handleSend">发送</n-button>
        </template>
      </n-input>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { NScrollbar, NInput, NButton, NSpin } from 'naive-ui'
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
  background: #0D0C0A;
}
.chat-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.messages {
  flex: 1;
  padding: 24px;
}
.input-row {
  padding: 16px 24px;
  border-top: 1px solid #2a2722;
}
.loading {
  text-align: center;
  padding: 16px;
}
</style>
