import type { ChatConversation, ChatMessage, ChatPrefs, ChatRepository, ChatRole, ChatMessageStatus } from './types'

const TITLE_MAX = 24
const EMPTY_TITLE = 'New chat'
const DB_NAME = 'sub2api-chat'
const DB_VERSION = 1
const CONV_STORE = 'conversations'
const PREF_STORE = 'prefs'

const ROLES: ChatRole[] = ['user', 'assistant']
const STATUSES: ChatMessageStatus[] = ['complete', 'streaming', 'error', 'aborted']

export function titleFromFirstMessage(content: string): string {
  const line = content.replace(/\r/g, '').split('\n').find((item) => item.trim())?.trim() ?? ''
  if (!line) {
    return EMPTY_TITLE
  }
  return Array.from(line).slice(0, TITLE_MAX).join('')
}

export function createChatId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `chat_${Date.now().toString(36)}_${Math.random().toString(16).slice(2)}`
}

function asRole(value: unknown): ChatRole {
  return ROLES.includes(value as ChatRole) ? (value as ChatRole) : 'assistant'
}

function asStatus(value: unknown): ChatMessageStatus {
  if (value === 'streaming') {
    return 'aborted'
  }
  return STATUSES.includes(value as ChatMessageStatus) ? (value as ChatMessageStatus) : 'complete'
}

function sanitizeMessage(input: unknown): ChatMessage | null {
  if (!input || typeof input !== 'object') {
    return null
  }
  const raw = input as Record<string, unknown>
  const id = typeof raw.id === 'string' && raw.id ? raw.id : null
  const content = typeof raw.content === 'string' ? raw.content : null
  const createdAt = typeof raw.createdAt === 'number' && Number.isFinite(raw.createdAt) ? raw.createdAt : null
  if (!id || content === null || createdAt === null) {
    return null
  }
  return {
    id,
    role: asRole(raw.role),
    content,
    createdAt,
    status: asStatus(raw.status),
  }
}

export function sanitizeConversation(input: unknown): ChatConversation {
  const raw = input && typeof input === 'object' ? (input as Record<string, unknown>) : {}
  const messages = Array.isArray(raw.messages)
    ? raw.messages.map(sanitizeMessage).filter((item): item is ChatMessage => item !== null)
    : []
  return {
    id: typeof raw.id === 'string' && raw.id ? raw.id : createChatId(),
    title: typeof raw.title === 'string' && raw.title.trim() ? raw.title : EMPTY_TITLE,
    createdAt: typeof raw.createdAt === 'number' && Number.isFinite(raw.createdAt) ? raw.createdAt : Date.now(),
    updatedAt: typeof raw.updatedAt === 'number' && Number.isFinite(raw.updatedAt) ? raw.updatedAt : Date.now(),
    keyId: typeof raw.keyId === 'number' && Number.isFinite(raw.keyId) ? raw.keyId : null,
    model: typeof raw.model === 'string' ? raw.model : '',
    messages,
  }
}

export function sanitizePrefs(input: unknown): ChatPrefs {
  const raw = input && typeof input === 'object' ? (input as Record<string, unknown>) : {}
  return {
    selectedKeyId: typeof raw.selectedKeyId === 'number' && Number.isFinite(raw.selectedKeyId) ? raw.selectedKeyId : null,
    selectedModel: typeof raw.selectedModel === 'string' ? raw.selectedModel : '',
  }
}

export function emptyConversation(overrides: Partial<ChatConversation> = {}): ChatConversation {
  const now = Date.now()
  return sanitizeConversation({
    id: createChatId(),
    title: EMPTY_TITLE,
    createdAt: now,
    updatedAt: now,
    keyId: null,
    model: '',
    messages: [],
    ...overrides,
  })
}

function conversationKey(userId: number, id: string): string {
  return `${userId}\u0000${id}`
}

export function createMemoryChatRepository(): ChatRepository {
  const conversations = new Map<string, ChatConversation>()
  const prefs = new Map<number, ChatPrefs>()

  return {
    persisted: false,
    async listConversations(userId: number) {
      return [...conversations.entries()]
        .filter(([key]) => key.startsWith(`${userId}\u0000`))
        .map(([, value]) => sanitizeConversation(value))
        .sort((a, b) => b.updatedAt - a.updatedAt)
    },
    async getConversation(userId: number, id: string) {
      const stored = conversations.get(conversationKey(userId, id))
      return stored ? sanitizeConversation(stored) : null
    },
    async saveConversation(userId: number, conversation: ChatConversation) {
      const clean = sanitizeConversation(conversation)
      conversations.set(conversationKey(userId, clean.id), clean)
      return clean
    },
    async deleteConversation(userId: number, id: string) {
      conversations.delete(conversationKey(userId, id))
    },
    async getPrefs(userId: number) {
      return sanitizePrefs(prefs.get(userId))
    },
    async savePrefs(userId: number, next: ChatPrefs) {
      prefs.set(userId, sanitizePrefs(next))
    },
    async clearAll() {
      conversations.clear()
      prefs.clear()
    },
  }
}

function requestToPromise<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

function openChatDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(CONV_STORE)) {
        const store = db.createObjectStore(CONV_STORE, { keyPath: 'recordKey' })
        store.createIndex('userId', 'userId', { unique: false })
      }
      if (!db.objectStoreNames.contains(PREF_STORE)) {
        db.createObjectStore(PREF_STORE, { keyPath: 'userId' })
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

export function createIndexedDBChatRepository(): ChatRepository {
  async function withStore<T>(
    storeName: string,
    mode: IDBTransactionMode,
    run: (store: IDBObjectStore) => Promise<T> | T,
  ): Promise<T> {
    const db = await openChatDb()
    try {
      const tx = db.transaction(storeName, mode)
      const done = new Promise<void>((resolve, reject) => {
        tx.oncomplete = () => resolve()
        tx.onerror = () => reject(tx.error)
        tx.onabort = () => reject(tx.error)
      })
      const result = await run(tx.objectStore(storeName))
      await done
      return result
    } finally {
      db.close()
    }
  }

  return {
    persisted: true,
    async listConversations(userId: number) {
      return withStore(CONV_STORE, 'readonly', async (store) => {
        const index = store.index('userId')
        const rows = await requestToPromise(index.getAll(userId))
        return rows
          .map((row) => sanitizeConversation((row as { conversation?: unknown }).conversation))
          .sort((a, b) => b.updatedAt - a.updatedAt)
      })
    },
    async getConversation(userId: number, id: string) {
      return withStore(CONV_STORE, 'readonly', async (store) => {
        const row = await requestToPromise(store.get(conversationKey(userId, id)))
        if (!row) {
          return null
        }
        return sanitizeConversation((row as { conversation?: unknown }).conversation)
      })
    },
    async saveConversation(userId: number, conversation: ChatConversation) {
      const clean = sanitizeConversation(conversation)
      await withStore(CONV_STORE, 'readwrite', async (store) => {
        store.put({
          recordKey: conversationKey(userId, clean.id),
          userId,
          conversation: clean,
        })
        return clean
      })
      return clean
    },
    async deleteConversation(userId: number, id: string) {
      await withStore(CONV_STORE, 'readwrite', (store) => {
        store.delete(conversationKey(userId, id))
      })
    },
    async getPrefs(userId: number) {
      return withStore(PREF_STORE, 'readonly', async (store) => {
        const row = await requestToPromise(store.get(userId))
        return sanitizePrefs(row)
      })
    },
    async savePrefs(userId: number, next: ChatPrefs) {
      const clean = sanitizePrefs(next)
      await withStore(PREF_STORE, 'readwrite', (store) => {
        store.put({ userId, ...clean })
      })
    },
    async clearAll() {
      await withStore(CONV_STORE, 'readwrite', (store) => store.clear())
      await withStore(PREF_STORE, 'readwrite', (store) => store.clear())
    },
  }
}

export function createBrowserChatRepository(): ChatRepository {
  if (typeof indexedDB === 'undefined') {
    return createMemoryChatRepository()
  }
  return createIndexedDBChatRepository()
}
