import { describe, expect, it } from 'vitest'
import { toGatewayMessages } from '../session'
import { emptyConversation } from '../storage'
import { FORTUNE_SYSTEM_PROMPT } from '../prompts'
import { suggestionsFor } from '../suggestions'

describe('fortune mode', () => {
  const messages = [{ id: 'm1', role: 'user' as const, content: '看看今年运势', createdAt: 1, status: 'complete' as const }]

  it('prepends the fortune system prompt only for fortune conversations', () => {
    const fortune = toGatewayMessages(emptyConversation({ mode: 'fortune', messages }))
    expect(fortune[0]).toEqual({ role: 'system', content: FORTUNE_SYSTEM_PROMPT })
    expect(fortune[1]).toEqual({ role: 'user', content: '看看今年运势' })

    const plain = toGatewayMessages(emptyConversation({ messages }))
    expect(plain).toEqual([{ role: 'user', content: '看看今年运势' }])
  })

  it('offers locale-specific quick prompts per mode', () => {
    expect(suggestionsFor('zh-CN', 'fortune').map((s) => s.title)).toContain('八字命盘')
    expect(suggestionsFor('en', 'chat').length).toBe(4)
  })
})
