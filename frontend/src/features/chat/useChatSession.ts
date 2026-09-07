import { onMounted, onUnmounted, reactive, toRefs } from 'vue'
import { keysAPI } from '@/api'
import { buildGatewayUrl } from '@/api/url'
import { useAuthStore } from '@/stores/auth'
import { listGatewayModels, streamChatCompletion } from './gateway'
import { createChatEngine, createInitialChatState, type ChatEngineDeps } from './session'
import { createBrowserChatRepository } from './storage'

export function useChatSession(overrides: Partial<ChatEngineDeps> = {}) {
  const authStore = useAuthStore()
  const repository = overrides.repository ?? createBrowserChatRepository()
  const state = reactive(createInitialChatState(repository.persisted))
  const engine = createChatEngine(
    {
      getUserId: () => authStore.user?.id ?? null,
      repository,
      listKeys: async () => {
        const page = await keysAPI.list(1, 100, { status: 'active' })
        return page.items
      },
      listModels: listGatewayModels,
      stream: streamChatCompletion,
      completionsUrl: () => buildGatewayUrl('/v1/chat/completions'),
      ...overrides,
    },
    state,
  )

  onMounted(() => {
    void engine.load().catch((error: unknown) => {
      state.sendError = {
        code: 'upstream',
        message: error instanceof Error ? error.message : 'Failed to load chat',
      }
    })
  })

  onUnmounted(() => {
    engine.stop()
  })

  return {
    ...toRefs(state),
    load: engine.load,
    selectConversation: engine.selectConversation,
    newConversation: engine.newConversation,
    renameConversation: engine.renameConversation,
    deleteConversation: engine.deleteConversation,
    selectKey: engine.selectKey,
    selectModel: engine.selectModel,
    send: engine.send,
    retry: engine.retry,
    stop: engine.stop,
    activeConversation: engine.activeConversation,
  }
}
