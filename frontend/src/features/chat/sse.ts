export interface SseEvent {
  data: string
}

export interface ChatDelta {
  content: string
  finishReason: string | null
  error: string | null
  done: boolean
}

export function consumeSseChunk(rest: string, chunk: string): { events: SseEvent[]; rest: string } {
  const combined = `${rest}${chunk}`.replace(/\r\n/g, '\n')
  const blocks = combined.split('\n\n')
  const incomplete = blocks.pop() ?? ''
  const events: SseEvent[] = []

  for (const block of blocks) {
    const dataLines: string[] = []
    for (const line of block.split('\n')) {
      if (!line || line.startsWith(':') || line.startsWith('event:')) {
        continue
      }
      if (line.startsWith('data:')) {
        dataLines.push(line.slice(5).trimStart())
      }
    }
    if (dataLines.length > 0) {
      events.push({ data: dataLines.join('\n') })
    }
  }

  return { events, rest: incomplete }
}

export function extractChatDelta(data: string): ChatDelta {
  const trimmed = data.trim()
  if (!trimmed) {
    return { content: '', finishReason: null, error: null, done: false }
  }
  if (trimmed === '[DONE]') {
    return { content: '', finishReason: null, error: null, done: true }
  }

  try {
    const payload = JSON.parse(trimmed) as {
      error?: { message?: string }
      choices?: Array<{ delta?: { content?: string }; finish_reason?: string | null }>
    }
    if (payload.error?.message) {
      return { content: '', finishReason: null, error: payload.error.message, done: false }
    }
    const choice = payload.choices?.[0]
    return {
      content: typeof choice?.delta?.content === 'string' ? choice.delta.content : '',
      finishReason: choice?.finish_reason ?? null,
      error: null,
      done: false,
    }
  } catch {
    return { content: '', finishReason: null, error: null, done: false }
  }
}
