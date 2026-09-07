export type ChatRole = 'user' | 'assistant'
export type ChatMessageStatus = 'complete' | 'streaming' | 'error' | 'aborted'

export interface ChatMessage {
  id: string
  role: ChatRole
  content: string
  createdAt: number
  status: ChatMessageStatus
}

export interface ChatConversation {
  id: string
  title: string
  createdAt: number
  updatedAt: number
  keyId: number | null
  model: string
  messages: ChatMessage[]
}

export interface ChatPrefs {
  selectedKeyId: number | null
  selectedModel: string
}

export type GatewayErrorCode = 'invalid_key' | 'billing' | 'rate_limit' | 'upstream'

export interface GatewayErrorShape {
  code: GatewayErrorCode
  message: string
}

export interface ChatCompletionMessage {
  role: ChatRole | 'system'
  content: string
}

export interface StreamChatResult {
  finishReason: string | null
  error: string | null
}

export interface ChatRepository {
  persisted: boolean
  listConversations(userId: number): Promise<ChatConversation[]>
  getConversation(userId: number, id: string): Promise<ChatConversation | null>
  saveConversation(userId: number, conversation: ChatConversation): Promise<ChatConversation>
  deleteConversation(userId: number, id: string): Promise<void>
  getPrefs(userId: number): Promise<ChatPrefs>
  savePrefs(userId: number, prefs: ChatPrefs): Promise<void>
  clearAll(): Promise<void>
}
