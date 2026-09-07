import { describe, expect, it } from 'vitest'
import { listGatewayModels, streamChatCompletion } from '../gateway'

const key = process.env.SUB2API_TEST_KEY || ''
const completionsUrl = 'https://yf-mail.com/v1/chat/completions'

describe.skipIf(!key)('live gateway against yf-mail.com', () => {
  it('lists models and streams a short completion with the selected key', async () => {
    const models = await listGatewayModels({ completionsUrl, apiKey: key })
    expect(models.length).toBeGreaterThan(0)
    expect(models).toContain('gpt-5.4-mini')

    const chunks: string[] = []
    const result = await streamChatCompletion({
      completionsUrl,
      apiKey: key,
      model: 'gpt-5.4-mini',
      messages: [{ role: 'user', content: 'Reply with exactly: ping-ok' }],
      onDelta: (text) => {
        chunks.push(text)
      },
    })

    expect(chunks.join('')).toContain('ping-ok')
    expect(result.finishReason).toBe('stop')
    expect(result.error).toBeNull()
  }, 90000)
})
