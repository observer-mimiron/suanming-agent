import { ref } from 'vue'
import type { ChatMessage } from '../types/chat'

export function useSSE() {
  const messages = ref<ChatMessage[]>([])
  const isLoading = ref(false)
  const sessionId = ref(crypto.randomUUID())

  function sendMessage(content: string) {
    const msgs = messages.value
    msgs.push({ id: Date.now().toString(), role: 'user', segments: [{ type: 'text', content }] })
    const assistIdx = msgs.length
    msgs.push({ id: (Date.now()+1).toString(), role: 'assistant', segments: [] })
    isLoading.value = true

    const xhr = new XMLHttpRequest()
    xhr.open('POST', '/api/chat')
    xhr.setRequestHeader('Content-Type', 'application/json')
    let prevLen = 0
    let timer: ReturnType<typeof setInterval> | null = null

    const parseChunk = () => {
      const raw = xhr.responseText
      if (!raw || raw.length <= prevLen) return
      const chunk = raw.substring(prevLen)
      prevLen = raw.length

      let currentEvent = ''
      const lines = chunk.split('\n')
      const target = msgs[assistIdx]
      for (const line of lines) {
        if (!line) continue
        if (line.startsWith('event: ')) { currentEvent = line.slice(7).trim(); continue }
        if (!line.startsWith('data: ')) continue
        try {
          const data = JSON.parse(line.slice(6))
          if (currentEvent === 'thinking') { target.segments.push({ type:'thinking', text:data.text, agent:data.agent }) }
          else if (currentEvent === 'tool_call') { target.segments.push({ type:'tool_call', tool:data.tool, params:data.params }) }
          else if (currentEvent === 'component') { target.segments.push({ type:'component', componentType:data.type, payload:data.payload }) }
          else if (currentEvent === 'error') { target.segments.push({ type:'error', message:data.message }) }
          else if (currentEvent === 'text') {
            const segs = target.segments
            const lastIdx = segs.length - 1
            if (lastIdx >= 0 && segs[lastIdx].type === 'text') {
              segs[lastIdx] = { ...segs[lastIdx], content: segs[lastIdx].content + data.content }
            } else {
              segs.push({ type:'text', content: data.content })
            }
          }
        } catch { /* skip */ }
      }
    }

    xhr.onreadystatechange = () => {
      if (xhr.readyState === 2) {
        // Start polling as soon as headers are received
        timer = setInterval(parseChunk, 50)
      }
      if (xhr.readyState === 4) {
        if (timer) { clearInterval(timer); timer = null }
        parseChunk()
        isLoading.value = false
      }
    }

    xhr.onerror = () => {
      if (timer) { clearInterval(timer); timer = null }
      msgs[assistIdx].segments.push({ type: 'error', message: '网络请求失败' })
      isLoading.value = false
    }

    xhr.send(JSON.stringify({ message: content, session_id: sessionId.value }))
  }

  return { messages, isLoading, sendMessage }
}
