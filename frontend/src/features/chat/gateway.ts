import { consumeSseChunk, extractChatDelta } from './sse'
import type { ChatCompletionMessage, GatewayErrorCode, GatewayErrorShape, StreamChatResult } from './types'

export class GatewayError extends Error implements GatewayErrorShape {
  code: GatewayErrorCode

  constructor(code: GatewayErrorCode, message: string) {
    super(message)
    this.name = 'GatewayError'
    this.code = code
  }
}

function modelsUrlFromCompletions(completionsUrl: string): string {
  if (completionsUrl.endsWith('/v1/chat/completions')) {
    return `${completionsUrl.slice(0, -'/chat/completions'.length)}/models`
  }
  return completionsUrl.replace(/\/chat\/completions\/?$/, '/models')
}

function readErrorMessage(body: unknown, fallback: string): string {
  if (body && typeof body === 'object') {
    const error = (body as { error?: { message?: string } }).error
    if (error?.message) {
      return error.message
    }
  }
  return fallback
}

export function mapGatewayError(status: number, body: unknown): GatewayErrorShape {
  const message = readErrorMessage(body, `Request failed (${status})`)
  const lower = message.toLowerCase()
  if (status === 401 || lower.includes('invalid api key') || lower.includes('authentication')) {
    return { code: 'invalid_key', message }
  }
  if (
    status === 402 ||
    status === 403 ||
    lower.includes('balance') ||
    lower.includes('quota') ||
    message.includes('余额')
  ) {
    return { code: 'billing', message }
  }
  if (status === 429) {
    return { code: 'rate_limit', message }
  }
  return { code: 'upstream', message }
}

async function readJsonBody(response: Response): Promise<unknown> {
  const text = await response.text()
  if (!text) {
    return null
  }
  try {
    return JSON.parse(text)
  } catch {
    return { error: { message: text } }
  }
}

function isAbortError(error: unknown): boolean {
  return (
    (error instanceof DOMException && error.name === 'AbortError') ||
    (error instanceof Error && error.name === 'AbortError')
  )
}

export async function listGatewayModels(options: {
  completionsUrl: string
  apiKey: string
  signal?: AbortSignal
}): Promise<string[]> {
  const response = await fetch(modelsUrlFromCompletions(options.completionsUrl), {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${options.apiKey}`,
    },
    signal: options.signal,
  })
  const body = await readJsonBody(response)
  if (!response.ok) {
    const mapped = mapGatewayError(response.status, body)
    throw new GatewayError(mapped.code, mapped.message)
  }
  const data = Array.isArray((body as { data?: unknown })?.data) ? (body as { data: Array<{ id?: string }> }).data : []
  return data.map((item) => item.id).filter((id): id is string => typeof id === 'string' && id.length > 0)
}

export async function streamChatCompletion(options: {
  completionsUrl: string
  apiKey: string
  model: string
  messages: ChatCompletionMessage[]
  signal?: AbortSignal
  onDelta: (text: string) => void
}): Promise<StreamChatResult> {
  let response: Response
  try {
    response = await fetch(options.completionsUrl, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${options.apiKey}`,
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      },
      body: JSON.stringify({
        model: options.model,
        stream: true,
        messages: options.messages,
      }),
      signal: options.signal,
    })
  } catch (error) {
    if (isAbortError(error)) {
      return { finishReason: 'aborted', error: null }
    }
    throw error
  }

  if (!response.ok) {
    const body = await readJsonBody(response)
    const mapped = mapGatewayError(response.status, body)
    throw new GatewayError(mapped.code, mapped.message)
  }

  const reader = response.body?.getReader()
  if (!reader) {
    throw new GatewayError('upstream', 'No response body')
  }

  const decoder = new TextDecoder()
  let rest = ''
  let finishReason: string | null = null

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }
      const chunk = decoder.decode(value, { stream: true })
      const parsed = consumeSseChunk(rest, chunk)
      rest = parsed.rest
      for (const event of parsed.events) {
        const delta = extractChatDelta(event.data)
        if (delta.error) {
          throw new GatewayError('upstream', delta.error)
        }
        if (delta.content) {
          options.onDelta(delta.content)
        }
        if (delta.finishReason) {
          finishReason = delta.finishReason
        }
        if (delta.done) {
          return { finishReason: finishReason ?? 'stop', error: null }
        }
      }
    }
    const tail = decoder.decode()
    const flushed = consumeSseChunk(rest, `${tail}\n\n`)
    for (const event of flushed.events) {
      const delta = extractChatDelta(event.data)
      if (delta.error) {
        throw new GatewayError('upstream', delta.error)
      }
      if (delta.content) {
        options.onDelta(delta.content)
      }
      if (delta.finishReason) {
        finishReason = delta.finishReason
      }
      if (delta.done) {
        return { finishReason: finishReason ?? 'stop', error: null }
      }
    }
  } catch (error) {
    if (isAbortError(error)) {
      return { finishReason: 'aborted', error: null }
    }
    throw error
  }

  return { finishReason: finishReason ?? 'stop', error: null }
}
