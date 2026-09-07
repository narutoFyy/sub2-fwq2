import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  listGatewayModels,
  mapGatewayError,
  streamChatCompletion,
} from '../gateway'

function sseResponse(chunks: string[], status = 200, contentType = 'text/event-stream'): Response {
  const encoder = new TextEncoder()
  let index = 0
  const body = new ReadableStream<Uint8Array>({
    pull(controller) {
      if (index >= chunks.length) {
        controller.close()
        return
      }
      controller.enqueue(encoder.encode(chunks[index]))
      index += 1
    },
  })
  return new Response(body, { status, headers: { 'Content-Type': contentType } })
}

describe('mapGatewayError', () => {
  it('prefers the OpenAI error message and classifies common statuses', () => {
    expect(mapGatewayError(401, { error: { message: 'Invalid API key', type: 'authentication_error' } })).toEqual({
      code: 'invalid_key',
      message: 'Invalid API key',
    })
    expect(mapGatewayError(403, { error: { message: 'insufficient balance' } })).toEqual({
      code: 'billing',
      message: 'insufficient balance',
    })
    expect(mapGatewayError(429, { error: { message: 'rate limited' } })).toEqual({
      code: 'rate_limit',
      message: 'rate limited',
    })
    expect(mapGatewayError(500, { error: { message: 'upstream failed' } })).toEqual({
      code: 'upstream',
      message: 'upstream failed',
    })
  })
})

describe('listGatewayModels', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('requests /v1/models with the selected API key and returns ids', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ data: [{ id: 'gpt-4.1' }, { id: 'claude-sonnet-4' }] }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const models = await listGatewayModels({
      completionsUrl: 'https://yf-mail.com/v1/chat/completions',
      apiKey: 'sk-user',
    })

    expect(models).toEqual(['gpt-4.1', 'claude-sonnet-4'])
    expect(fetchMock).toHaveBeenCalledWith(
      'https://yf-mail.com/v1/models',
      expect.objectContaining({
        headers: expect.objectContaining({ Authorization: 'Bearer sk-user' }),
      }),
    )
    const headers = fetchMock.mock.calls[0][1].headers as Record<string, string>
    expect(JSON.stringify(headers)).not.toMatch(/auth_token/i)
  })
})

describe('streamChatCompletion', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends the selected key and yields streamed text until DONE', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      sseResponse(['data: {"choices":[{"delta":{"content":"Hi"}}]}\n\n', 'data: [DONE]\n\n']),
    )
    vi.stubGlobal('fetch', fetchMock)
    const deltas: string[] = []

    const result = await streamChatCompletion({
      completionsUrl: 'https://yf-mail.com/v1/chat/completions',
      apiKey: 'sk-user',
      model: 'claude-sonnet-4',
      messages: [{ role: 'user', content: 'Hello' }],
      onDelta: (text) => deltas.push(text),
    })

    expect(deltas).toEqual(['Hi'])
    expect(result).toEqual({ finishReason: 'stop', error: null })
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(init.method).toBe('POST')
    expect(init.headers).toMatchObject({ Authorization: 'Bearer sk-user' })
    expect(JSON.parse(String(init.body))).toMatchObject({
      model: 'claude-sonnet-4',
      stream: true,
      messages: [{ role: 'user', content: 'Hello' }],
    })
  })

  it('flushes a final SSE event that has no trailing blank line', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      sseResponse(['data: {"choices":[{"delta":{"content":"Hi"}}]}']),
    )
    vi.stubGlobal('fetch', fetchMock)
    const deltas: string[] = []
    const result = await streamChatCompletion({
      completionsUrl: 'https://yf-mail.com/v1/chat/completions',
      apiKey: 'sk-user',
      model: 'claude-sonnet-4',
      messages: [{ role: 'user', content: 'Hello' }],
      onDelta: (text) => deltas.push(text),
    })
    expect(deltas).toEqual(['Hi'])
    expect(result.finishReason).toBe('stop')
  })

  it('maps non-stream JSON errors before any delta', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { message: 'Invalid API key', type: 'authentication_error' } }), {
          status: 401,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )

    await expect(
      streamChatCompletion({
        completionsUrl: 'https://yf-mail.com/v1/chat/completions',
        apiKey: 'sk-bad',
        model: 'gpt-4.1',
        messages: [{ role: 'user', content: 'Hello' }],
        onDelta: () => undefined,
      }),
    ).rejects.toMatchObject({ code: 'invalid_key', message: 'Invalid API key' })
  })

  it('treats abort as a controlled stop rather than an upstream error', async () => {
    const abort = new AbortController()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((_url: string, init?: RequestInit) => {
        return new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () => {
            const error = new DOMException('Aborted', 'AbortError')
            reject(error)
          })
          abort.abort()
        })
      }),
    )

    const result = await streamChatCompletion({
      completionsUrl: 'https://yf-mail.com/v1/chat/completions',
      apiKey: 'sk-user',
      model: 'gpt-4.1',
      messages: [{ role: 'user', content: 'Hello' }],
      signal: abort.signal,
      onDelta: () => undefined,
    })
    expect(result).toEqual({ finishReason: 'aborted', error: null })
  })
})
