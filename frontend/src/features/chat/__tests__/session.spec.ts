import { describe, expect, it, vi } from 'vitest'
import { reactive, watch } from 'vue'
import { createChatEngine, createInitialChatState } from '../session'
import { createMemoryChatRepository } from '../storage'
import { GatewayError } from '../gateway'
import type { ApiKey } from '@/types'

function key(overrides: Partial<ApiKey> = {}): ApiKey {
  return {
    id: 40,
    user_id: 1,
    key: 'sk-live',
    name: 'main',
    group_id: 1,
    status: 'active',
    ip_whitelist: [],
    ip_blacklist: [],
    last_used_at: null,
    last_used_ip: null,
    quota: 0,
    quota_used: 0,
    expires_at: null,
    created_at: '',
    updated_at: '',
    current_concurrency: 0,
    rate_limit_5h: 0,
    rate_limit_1d: 0,
    rate_limit_7d: 0,
    usage_5h: 0,
    usage_1d: 0,
    usage_7d: 0,
    window_5h_start: null,
    window_1d_start: null,
    window_7d_start: null,
    reset_5h_at: null,
    reset_1d_at: null,
    reset_7d_at: null,
    ...overrides,
  }
}

function engine(options?: {
  keys?: ApiKey[]
  stream?: ReturnType<typeof vi.fn>
  listModels?: ReturnType<typeof vi.fn>
}) {
  const repository = createMemoryChatRepository()
  const stream = options?.stream ?? vi.fn().mockImplementation(async ({ onDelta }) => {
    onDelta('Hello')
    return { finishReason: 'stop', error: null }
  })
  const created = createChatEngine({
    getUserId: () => 7,
    repository,
    listKeys: async () => options?.keys ?? [key()],
    listModels: options?.listModels ?? vi.fn().mockResolvedValue(['claude-sonnet-4', 'gpt-4.1']),
    stream,
    completionsUrl: () => 'https://example.test/v1/chat/completions',
  })
  return { ...created, repository, stream }
}

describe('createChatEngine', () => {
  it('loads keys and models, ignoring inactive keys', async () => {
    const { state, load } = engine({
      keys: [key({ id: 1, status: 'inactive', key: 'sk-dead' }), key({ id: 2, name: 'live' })],
    })
    await load()
    expect(state.keys.map((item) => item.id)).toEqual([2])
    expect(state.selectedKeyId).toBe(2)
    expect(state.models).toEqual(['claude-sonnet-4', 'gpt-4.1'])
    expect(state.selectedModel).toBe('claude-sonnet-4')
  })

  it('keeps the live assistant message so streamed tokens stay on the same object', async () => {
    const box: { current?: ReturnType<typeof createChatEngine> } = {}
    const stream = vi.fn().mockImplementation(async ({ onDelta }) => {
      const assistant = box.current?.state.conversations[0].messages.at(-1)
      onDelta('Hel')
      expect(assistant).toBe(box.current?.state.conversations[0].messages.at(-1))
      expect(assistant?.content).toBe('Hel')
      onDelta('lo')
      return { finishReason: 'stop', error: null }
    })
    const created = engine({ stream })
    box.current = created
    await created.load()
    created.state.draft = 'Hi'
    await created.send()
    expect(created.state.conversations[0].messages.at(-1)?.content).toBe('Hello')
  })

  it('notifies Vue when streaming into reactive engine state', async () => {
    const seen: string[] = []
    const repository = createMemoryChatRepository()
    const state = reactive(createInitialChatState(false))
    const created = createChatEngine(
      {
        getUserId: () => 7,
        repository,
        listKeys: async () => [key()],
        listModels: vi.fn().mockResolvedValue(['claude-sonnet-4']),
        stream: vi.fn().mockImplementation(async ({ onDelta }) => {
          onDelta('Hel')
          onDelta('lo')
          return { finishReason: 'stop', error: null }
        }),
        completionsUrl: () => 'https://example.test/v1/chat/completions',
      },
      state,
    )
    watch(
      () => created.activeConversation()?.messages.at(-1)?.content ?? '',
      (value) => {
        if (value) seen.push(value)
      },
      { flush: 'sync' },
    )
    await created.load()
    created.state.draft = 'Hi'
    await created.send()
    expect(seen).toEqual(['Hel', 'Hello'])
  })

  it('repairs leftover streaming rows so retry is available after reload', async () => {
    const { state, load, send, repository } = engine({
      stream: vi.fn().mockImplementation(async () => new Promise(() => undefined)),
    })
    await load()
    state.draft = 'hang'
    void send()
    await vi.waitFor(() => {
      expect(state.conversations[0]?.messages.at(-1)?.status).toBe('streaming')
    })
    const reloaded = createChatEngine({
      getUserId: () => 7,
      repository,
      listKeys: async () => [key()],
      listModels: vi.fn().mockResolvedValue(['claude-sonnet-4']),
      stream: vi.fn().mockResolvedValue({ finishReason: 'stop', error: null }),
      completionsUrl: () => 'https://example.test/v1/chat/completions',
    })
    await reloaded.load()
    expect(reloaded.state.conversations[0].messages.at(-1)?.status).toBe('aborted')
  })

  it('does not resurrect a conversation deleted while streaming', async () => {
    let release: () => void = () => undefined
    const { state, load, send, deleteConversation, repository } = engine({
      stream: vi.fn().mockImplementation(
        () =>
          new Promise((resolve) => {
            release = () => resolve({ finishReason: 'stop', error: null })
          }),
      ),
    })
    await load()
    state.draft = 'bye'
    const sending = send()
    await vi.waitFor(() => {
      expect(state.conversations).toHaveLength(1)
    })
    const id = state.conversations[0].id
    await deleteConversation(id)
    release()
    await sending
    expect(state.conversations.map((item) => item.id)).not.toContain(id)
    expect(await repository.getConversation(7, id)).toBeNull()
  })

  it('marks network failures on the assistant instead of throwing', async () => {
    const { state, load, send } = engine({
      stream: vi.fn().mockRejectedValue(new TypeError('Failed to fetch')),
    })
    await load()
    state.draft = 'hello'
    await send()
    expect(state.sendError).toEqual({ code: 'upstream', message: 'Failed to fetch' })
    expect(state.conversations[0].messages.at(-1)).toMatchObject({
      status: 'error',
      content: 'Failed to fetch',
    })
  })

  it('ignores a slower model list from the previous key', async () => {
    let finishFirst: (models: string[]) => void = () => undefined
    const first = new Promise<string[]>((resolve) => {
      finishFirst = resolve
    })
    const listModels = vi
      .fn()
      .mockImplementationOnce(() => first)
      .mockResolvedValueOnce(['fast-model'])
    const { state, load, selectKey } = engine({
      keys: [key({ id: 1, name: 'slow' }), key({ id: 2, name: 'fast', key: 'sk-fast' })],
      listModels,
    })
    const loading = load()
    await vi.waitFor(() => expect(listModels).toHaveBeenCalledTimes(1))
    state.selectedKeyId = 2
    await selectKey(2)
    finishFirst(['slow-model'])
    await loading
    expect(state.models).toEqual(['fast-model'])
    expect(state.selectedModel).toBe('fast-model')
  })

  it('sends with the selected key secret in memory and never persists it', async () => {
    const { state, load, send, repository, stream } = engine()
    await load()
    state.draft = '锈冠醒来'
    await send()

    expect(stream).toHaveBeenCalledWith(
      expect.objectContaining({
        apiKey: 'sk-live',
        model: 'claude-sonnet-4',
        messages: [{ role: 'user', content: '锈冠醒来' }],
      }),
    )
    const stored = await repository.listConversations(7)
    expect(stored).toHaveLength(1)
    expect(stored[0].title).toBe('锈冠醒来')
    expect(stored[0].messages[1]).toMatchObject({ role: 'assistant', content: 'Hello', status: 'complete' })
    expect(JSON.stringify(stored)).not.toContain('sk-live')
  })

  it('retries the last assistant without duplicating the user turn', async () => {
    const stream = vi.fn()
      .mockResolvedValueOnce({ finishReason: 'stop', error: null })
      .mockImplementationOnce(async ({ onDelta }) => {
        onDelta('again')
        return { finishReason: 'stop', error: null }
      })
    const { state, load, send, retry } = engine({ stream })
    await load()
    state.draft = 'Hi'
    await send()
    await retry()
    expect(state.conversations[0].messages.filter((item) => item.role === 'user')).toHaveLength(1)
    expect(state.conversations[0].messages.at(-1)).toMatchObject({ content: 'again', status: 'complete' })
  })

  it('records billing errors on the assistant message', async () => {
    const { state, load, send } = engine({
      stream: vi.fn().mockRejectedValue(new GatewayError('billing', 'insufficient balance')),
    })
    await load()
    state.draft = 'pay me'
    await send()
    expect(state.sendError).toEqual({ code: 'billing', message: 'insufficient balance' })
    expect(state.conversations[0].messages.at(-1)).toMatchObject({
      role: 'assistant',
      status: 'error',
      content: 'insufficient balance',
    })
  })

  it('isolates history per user id', async () => {
    const repository = createMemoryChatRepository()
    const first = createChatEngine({
      getUserId: () => 1,
      repository,
      listKeys: async () => [key()],
      listModels: vi.fn().mockResolvedValue(['m']),
      stream: vi.fn().mockResolvedValue({ finishReason: 'stop', error: null }),
      completionsUrl: () => 'https://example.test/v1/chat/completions',
    })
    await first.load()
    first.state.draft = 'from user 1'
    await first.send()

    const second = createChatEngine({
      getUserId: () => 2,
      repository,
      listKeys: async () => [key()],
      listModels: vi.fn().mockResolvedValue(['m']),
      stream: vi.fn().mockResolvedValue({ finishReason: 'stop', error: null }),
      completionsUrl: () => 'https://example.test/v1/chat/completions',
    })
    await second.load()
    expect(second.state.conversations).toEqual([])
    expect((await repository.listConversations(1))[0].title).toBe('from user 1')
  })
})
