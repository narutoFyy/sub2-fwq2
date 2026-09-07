import { describe, expect, it } from 'vitest'
import { consumeSseChunk, extractChatDelta } from '../sse'

describe('consumeSseChunk', () => {
  it('holds incomplete lines and emits complete data payloads', () => {
    const first = consumeSseChunk('', 'data: {"choices":[{"delta":{"content":"Hel')
    expect(first.events).toEqual([])
    const second = consumeSseChunk(first.rest, 'lo"}}]}\n\ndata: [DONE]\n\n')
    expect(second.events).toEqual([
      { data: '{"choices":[{"delta":{"content":"Hello"}}]}' },
      { data: '[DONE]' },
    ])
    expect(second.rest).toBe('')
  })

  it('ignores comments and event names used by some proxies', () => {
    const result = consumeSseChunk('', ': keep-alive\nevent: delta\ndata: {"ok":true}\n\n')
    expect(result.events).toEqual([{ data: '{"ok":true}' }])
  })
})

describe('extractChatDelta', () => {
  it('reads streamed content and finish reason', () => {
    expect(extractChatDelta('{"choices":[{"delta":{"content":"Hi"}}]}')).toEqual({
      content: 'Hi',
      finishReason: null,
      error: null,
      done: false,
    })
    expect(extractChatDelta('{"choices":[{"delta":{},"finish_reason":"stop"}]}')).toEqual({
      content: '',
      finishReason: 'stop',
      error: null,
      done: false,
    })
    expect(extractChatDelta('[DONE]')).toEqual({
      content: '',
      finishReason: null,
      error: null,
      done: true,
    })
  })

  it('reads OpenAI-style error objects', () => {
    expect(
      extractChatDelta('{"error":{"type":"authentication_error","message":"Invalid API key"}}'),
    ).toEqual({
      content: '',
      finishReason: null,
      error: 'Invalid API key',
      done: false,
    })
  })
})
