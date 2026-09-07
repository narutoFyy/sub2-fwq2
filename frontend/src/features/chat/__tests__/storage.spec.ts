import { beforeEach, describe, expect, it } from 'vitest'
import {
  createMemoryChatRepository,
  sanitizeConversation,
  titleFromFirstMessage,
} from '../storage'
import type { ChatConversation } from '../types'

function conversation(overrides: Partial<ChatConversation> = {}): ChatConversation {
  return {
    id: 'conv-1',
    title: 'Hello',
    createdAt: 1,
    updatedAt: 2,
    keyId: 40,
    model: 'claude-sonnet-4',
    messages: [
      {
        id: 'm1',
        role: 'user',
        content: 'Hello there',
        createdAt: 1,
        status: 'complete',
      },
    ],
    ...overrides,
  }
}

describe('titleFromFirstMessage', () => {
  it('uses the first line of the first user message, trimmed and capped', () => {
    expect(titleFromFirstMessage('  锈冠醒来，门里的人没有影子\nsecond')).toBe('锈冠醒来，门里的人没有影子')
    expect(titleFromFirstMessage('a'.repeat(80))).toHaveLength(24)
    expect(titleFromFirstMessage('   \n  ')).toBe('New chat')
  })
})

describe('sanitizeConversation', () => {
  it('keeps only known fields and drops API key secrets', () => {
    const dirty = {
      ...conversation(),
      apiKey: 'sk-secret',
      key: 'sk-secret',
      extra: true,
      messages: [
        {
          id: 'm1',
          role: 'user',
          content: 'Hi',
          createdAt: 1,
          status: 'complete',
          apiKey: 'sk-in-message',
        },
      ],
    }
    const clean = sanitizeConversation(dirty)
    expect(clean).toEqual({
      id: 'conv-1',
      title: 'Hello',
      createdAt: 1,
      updatedAt: 2,
      keyId: 40,
      model: 'claude-sonnet-4',
      messages: [
        {
          id: 'm1',
          role: 'user',
          content: 'Hi',
          createdAt: 1,
          status: 'complete',
        },
      ],
    })
    expect(JSON.stringify(clean)).not.toContain('sk-')
  })
})

describe('createMemoryChatRepository', () => {
  const repo = createMemoryChatRepository()

  beforeEach(async () => {
    await repo.clearAll()
  })

  it('isolates conversations by user id', async () => {
    await repo.saveConversation(1, conversation({ id: 'a', title: 'A', updatedAt: 10 }))
    await repo.saveConversation(2, conversation({ id: 'a', title: 'B', updatedAt: 11 }))

    const user1 = await repo.listConversations(1)
    const user2 = await repo.listConversations(2)
    expect(user1.map((item) => item.title)).toEqual(['A'])
    expect(user2.map((item) => item.title)).toEqual(['B'])
    expect(await repo.getConversation(1, 'a')).toMatchObject({ title: 'A' })
    expect(await repo.getConversation(2, 'missing')).toBeNull()
  })

  it('lists newest conversations first and can delete one', async () => {
    await repo.saveConversation(1, conversation({ id: 'old', title: 'Old', updatedAt: 1 }))
    await repo.saveConversation(1, conversation({ id: 'new', title: 'New', updatedAt: 9 }))
    expect((await repo.listConversations(1)).map((item) => item.id)).toEqual(['new', 'old'])
    await repo.deleteConversation(1, 'new')
    expect((await repo.listConversations(1)).map((item) => item.id)).toEqual(['old'])
  })

  it('does not persist injected API key secrets', async () => {
    await repo.saveConversation(7, {
      ...conversation(),
      apiKey: 'sk-should-not-persist',
    } as ChatConversation & { apiKey: string })
    const stored = await repo.getConversation(7, 'conv-1')
    expect(stored && 'apiKey' in stored).toBe(false)
    expect(JSON.stringify(stored)).not.toContain('sk-')
  })

  it('stores prefs as key id and model only', async () => {
    await repo.savePrefs(3, { selectedKeyId: 40, selectedModel: 'gpt-4.1' })
    expect(await repo.getPrefs(3)).toEqual({ selectedKeyId: 40, selectedModel: 'gpt-4.1' })
    expect(await repo.getPrefs(9)).toEqual({ selectedKeyId: null, selectedModel: '' })
  })
})
