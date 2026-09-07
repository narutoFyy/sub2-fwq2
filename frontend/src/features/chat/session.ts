import type { ApiKey } from '@/types'
import { GatewayError, listGatewayModels, streamChatCompletion } from './gateway'
import { createBrowserChatRepository, createChatId, emptyConversation, titleFromFirstMessage } from './storage'
import type {
  ChatCompletionMessage,
  ChatConversation,
  ChatMessage,
  ChatRepository,
  GatewayErrorShape,
  StreamChatResult,
} from './types'

export interface ChatEngineDeps {
  getUserId: () => number | null
  repository: ChatRepository
  listKeys: () => Promise<ApiKey[]>
  listModels: typeof listGatewayModels
  stream: typeof streamChatCompletion
  completionsUrl: () => string
}

export interface ChatEngineState {
  conversations: ChatConversation[]
  activeId: string | null
  keys: ApiKey[]
  models: string[]
  selectedKeyId: number | null
  selectedModel: string
  draft: string
  streaming: boolean
  loading: boolean
  modelsLoading: boolean
  sendError: GatewayErrorShape | null
  persistenceWarning: boolean
}

export function createInitialChatState(persisted = true): ChatEngineState {
  return {
    conversations: [],
    activeId: null,
    keys: [],
    models: [],
    selectedKeyId: null,
    selectedModel: '',
    draft: '',
    streaming: false,
    loading: false,
    modelsLoading: false,
    sendError: null,
    persistenceWarning: !persisted,
  }
}

function toGatewayMessages(conversation: ChatConversation): ChatCompletionMessage[] {
  return conversation.messages
    .filter((message) => message.role === 'user' || message.status === 'complete')
    .filter((message) => message.content.trim().length > 0)
    .map((message) => ({ role: message.role, content: message.content }))
}

function activeKey(state: ChatEngineState): ApiKey | undefined {
  return state.keys.find((key) => key.id === state.selectedKeyId)
}

function toErrorShape(error: unknown): GatewayErrorShape {
  if (error instanceof GatewayError) {
    return { code: error.code, message: error.message }
  }
  if (error instanceof Error && error.message) {
    return { code: 'upstream', message: error.message }
  }
  return { code: 'upstream', message: 'Request failed' }
}

function isAbortError(error: unknown): boolean {
  return (
    (error instanceof DOMException && error.name === 'AbortError') ||
    (error instanceof Error && error.name === 'AbortError')
  )
}

export function createChatEngine(deps: ChatEngineDeps, state: ChatEngineState = createInitialChatState(deps.repository.persisted)) {

  let abortController: AbortController | null = null
  let modelsRequest = 0
  const discarded = new Set<string>()

  function userId(): number {
    const id = deps.getUserId()
    if (!id) {
      throw new Error('not_authenticated')
    }
    return id
  }

  function activeConversation(): ChatConversation | null {
    return state.conversations.find((item) => item.id === state.activeId) ?? null
  }

  async function persist(conversation: ChatConversation) {
    if (discarded.has(conversation.id)) {
      return conversation
    }
    await deps.repository.saveConversation(userId(), conversation)
    if (discarded.has(conversation.id)) {
      await deps.repository.deleteConversation(userId(), conversation.id)
      return conversation
    }
    const index = state.conversations.findIndex((item) => item.id === conversation.id)
    if (index < 0) {
      state.conversations.unshift(conversation)
    }
    state.conversations.sort((a, b) => b.updatedAt - a.updatedAt)
    return conversation
  }

  async function persistPrefs() {
    await deps.repository.savePrefs(userId(), {
      selectedKeyId: state.selectedKeyId,
      selectedModel: state.selectedModel,
    })
  }

  async function refreshModels() {
    const requestId = ++modelsRequest
    const key = activeKey(state)
    if (!key?.key) {
      if (requestId === modelsRequest) {
        state.models = []
      }
      return
    }
    const keyId = key.id
    state.modelsLoading = true
    try {
      const models = await deps.listModels({
        completionsUrl: deps.completionsUrl(),
        apiKey: key.key,
      })
      if (requestId !== modelsRequest || state.selectedKeyId !== keyId) {
        return
      }
      state.models = models
      if (state.selectedModel && !models.includes(state.selectedModel)) {
        state.selectedModel = models[0] ?? ''
      } else if (!state.selectedModel) {
        state.selectedModel = models[0] ?? ''
      }
      await persistPrefs()
    } catch (error) {
      if (requestId !== modelsRequest || isAbortError(error)) {
        return
      }
      state.models = []
      state.sendError = toErrorShape(error)
    } finally {
      if (requestId === modelsRequest) {
        state.modelsLoading = false
      }
    }
  }

  async function load() {
    state.loading = true
    state.sendError = null
    try {
      const id = userId()
      const [keys, conversations, prefs] = await Promise.all([
        deps.listKeys(),
        deps.repository.listConversations(id),
        deps.repository.getPrefs(id),
      ])
      state.keys = keys.filter((key) => key.status === 'active')
      state.conversations = conversations
      state.persistenceWarning = !deps.repository.persisted
      const preferredKey = state.keys.find((key) => key.id === prefs.selectedKeyId)
      state.selectedKeyId = preferredKey?.id ?? state.keys[0]?.id ?? null
      state.selectedModel = prefs.selectedModel
      if (state.conversations[0] && !state.activeId) {
        state.activeId = state.conversations[0].id
      }
      await refreshModels()
    } catch (error) {
      state.sendError = toErrorShape(error)
    } finally {
      state.loading = false
    }
  }

  async function selectConversation(id: string | null) {
    state.activeId = id
    state.sendError = null
  }

  async function newConversation() {
    const conversation = emptyConversation({
      keyId: state.selectedKeyId,
      model: state.selectedModel,
    })
    state.activeId = conversation.id
    state.draft = ''
    state.sendError = null
    await persist(conversation)
  }

  async function renameConversation(id: string, title: string) {
    const conversation = state.conversations.find((item) => item.id === id)
    if (!conversation) {
      return
    }
    conversation.title = title.trim() || titleFromFirstMessage(conversation.messages.find((item) => item.role === 'user')?.content ?? '')
    conversation.updatedAt = Date.now()
    await persist(conversation)
  }

  async function deleteConversation(id: string) {
    discarded.add(id)
    if (state.activeId === id) {
      stop()
    }
    await deps.repository.deleteConversation(userId(), id)
    state.conversations = state.conversations.filter((item) => item.id !== id)
    if (state.activeId === id) {
      state.activeId = state.conversations[0]?.id ?? null
    }
  }

  async function selectKey(keyId: number) {
    state.selectedKeyId = keyId
    await persistPrefs()
    await refreshModels()
    const conversation = activeConversation()
    if (conversation) {
      conversation.keyId = keyId
      conversation.updatedAt = Date.now()
      await persist(conversation)
    }
  }

  async function selectModel(model: string) {
    state.selectedModel = model
    await persistPrefs()
    const conversation = activeConversation()
    if (conversation) {
      conversation.model = model
      conversation.updatedAt = Date.now()
      await persist(conversation)
    }
  }

  function conversationById(id: string): ChatConversation | undefined {
    return state.conversations.find((item) => item.id === id)
  }

  function liveAssistant(conversation: ChatConversation, assistantId: string): ChatMessage | undefined {
    return conversation.messages.find((message) => message.id === assistantId)
  }

  async function finishAssistant(conversation: ChatConversation, assistantId: string, result: StreamChatResult | GatewayErrorShape, aborted: boolean) {
    if (discarded.has(conversation.id)) {
      return
    }
    const assistant = liveAssistant(conversation, assistantId)
    if (!assistant) {
      return
    }
    if ('code' in result) {
      assistant.status = 'error'
      if (!assistant.content) {
        assistant.content = result.message
      }
      state.sendError = result
    } else if (aborted || result.finishReason === 'aborted') {
      assistant.status = 'aborted'
    } else {
      assistant.status = 'complete'
    }
    conversation.updatedAt = Date.now()
    await persist(conversation)
  }

  async function runCompletion(conversation: ChatConversation, assistantId: string) {
    const key = activeKey(state)
    if (!key?.key) {
      await finishAssistant(conversation, assistantId, { code: 'invalid_key', message: 'No API key selected' }, false)
      return
    }
    abortController = new AbortController()
    state.streaming = true
    state.sendError = null
    try {
      const result = await deps.stream({
        completionsUrl: deps.completionsUrl(),
        apiKey: key.key,
        model: state.selectedModel || conversation.model,
        messages: toGatewayMessages(conversation),
        signal: abortController.signal,
        onDelta: (text) => {
          if (discarded.has(conversation.id)) {
            return
          }
          const liveConversation = conversationById(conversation.id) ?? conversation
          const live = liveAssistant(liveConversation, assistantId)
          if (!live) {
            return
          }
          live.content += text
          live.status = 'streaming'
        },
      })
      await finishAssistant(conversation, assistantId, result, false)
    } catch (error) {
      if (isAbortError(error)) {
        await finishAssistant(conversation, assistantId, { finishReason: 'aborted', error: null }, true)
      } else {
        await finishAssistant(conversation, assistantId, toErrorShape(error), false)
      }
    } finally {
      state.streaming = false
      abortController = null
    }
  }

  async function send() {
    const content = state.draft.trim()
    if (!content || state.streaming) {
      return
    }
    if (!activeKey(state)) {
      state.sendError = { code: 'invalid_key', message: 'No API key selected' }
      return
    }
    if (!state.selectedModel) {
      state.sendError = { code: 'upstream', message: 'No model selected' }
      return
    }

    let conversation = activeConversation()
    if (!conversation) {
      conversation = emptyConversation({
        keyId: state.selectedKeyId,
        model: state.selectedModel,
      })
      state.activeId = conversation.id
    }

    const now = Date.now()
    const userMessage: ChatMessage = {
      id: createChatId(),
      role: 'user',
      content,
      createdAt: now,
      status: 'complete',
    }
    const assistant: ChatMessage = {
      id: createChatId(),
      role: 'assistant',
      content: '',
      createdAt: now + 1,
      status: 'streaming',
    }
    conversation.messages.push(userMessage, assistant)
    const live = liveAssistant(conversation, assistant.id) ?? assistant
    if (conversation.title === 'New chat') {
      conversation.title = titleFromFirstMessage(content)
    }
    conversation.keyId = state.selectedKeyId
    conversation.model = state.selectedModel
    conversation.updatedAt = now
    state.draft = ''
    await persist(conversation)
    await runCompletion(conversationById(conversation.id) ?? conversation, live.id)
  }

  async function retry() {
    if (state.streaming) {
      return
    }
    const conversation = activeConversation()
    if (!conversation) {
      return
    }
    const last = conversation.messages[conversation.messages.length - 1]
    if (!last || last.role !== 'assistant') {
      return
    }
    last.content = ''
    last.status = 'streaming'
    last.createdAt = Date.now()
    await persist(conversation)
    await runCompletion(conversationById(conversation.id) ?? conversation, last.id)
  }

  function stop() {
    abortController?.abort()
  }

  return {
    state,
    load,
    selectConversation,
    newConversation,
    renameConversation,
    deleteConversation,
    selectKey,
    selectModel,
    send,
    retry,
    stop,
    activeConversation,
  }
}

export function defaultChatEngineDeps(overrides: Partial<ChatEngineDeps> = {}): ChatEngineDeps {
  return {
    getUserId: () => null,
    repository: createBrowserChatRepository(),
    listKeys: async () => [],
    listModels: listGatewayModels,
    stream: streamChatCompletion,
    completionsUrl: () => '/v1/chat/completions',
    ...overrides,
  }
}
