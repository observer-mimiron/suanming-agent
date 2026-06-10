import { ref } from 'vue'
import { createParser, type EventSourceMessage } from 'eventsource-parser'
import type { ChatMessage } from '../types/chat'

export function useSSE() {
  const messages = ref<ChatMessage[]>([])
  const isLoading = ref(false)
  const sessionId = ref(localStorage.getItem('sessionId') || crypto.randomUUID())
  localStorage.setItem('sessionId', sessionId.value)

  async function sendMessage(content: string) {
    messages.value.push({ id: Date.now().toString(), role: 'user', segments: [{ type: 'text', content }] })
    const msg: ChatMessage = { id: (Date.now()+1).toString(), role: 'assistant', segments: [] }
    messages.value.push(msg)
    isLoading.value = true

    const parser = createParser({
      onEvent: (event: EventSourceMessage) => {
        if (event.event === 'done') { isLoading.value = false; return }
        try {
          const data = JSON.parse(event.data)
          if (event.event === 'thinking') { msg.segments.push({ type:'thinking', text:data.text, agent:data.agent }) }
          else if (event.event === 'tool_call') { msg.segments.push({ type:'tool_call', tool:data.tool, params:data.params }) }
          else if (event.event === 'component') { msg.segments.push({ type:'component', componentType:data.type, payload:data.payload }) }
          else if (event.event === 'error') { msg.segments.push({ type:'error', message:data.message }) }
          else if (event.event === 'text') {
            const last = msg.segments[msg.segments.length-1]
            if (last?.type==='text') last.content += data.content
            else msg.segments.push({ type:'text', content: data.content })
          }
        } catch { /* skip unparseable data */ }
      }
    })

    try {
      const resp = await fetch('/api/chat', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: content, session_id: sessionId.value })
      })
      if (!resp.ok || !resp.body) {
        const text = await resp.text()
        msg.segments.push({ type: 'error', message: text || '请求失败' })
        isLoading.value = false
        return
      }
      const reader = resp.body.getReader()
      const decoder = new TextDecoder()
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        parser.feed(decoder.decode(value, { stream: true }))
      }
      isLoading.value = false
    } catch (err) {
      console.error('SSE error:', err)
      msg.segments.push({ type: 'error', message: String(err) })
      isLoading.value = false
    }
  }

  return { messages, isLoading, sendMessage }
}
