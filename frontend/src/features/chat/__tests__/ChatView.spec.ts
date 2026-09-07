import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import type { ChatConversation } from '../types'
import type { ApiKey } from '@/types'

const { session } = vi.hoisted(() => ({
  session: {} as {
    conversations: { value: ChatConversation[] }
    activeId: { value: string | null }
    keys: { value: ApiKey[] }
    models: { value: string[] }
    selectedKeyId: { value: number | null }
    selectedModel: { value: string }
    draft: { value: string }
    streaming: { value: boolean }
    loading: { value: boolean }
    modelsLoading: { value: boolean }
    sendError: { value: { code: string; message: string } | null }
    persistenceWarning: { value: boolean }
    load: ReturnType<typeof vi.fn>
    selectConversation: ReturnType<typeof vi.fn>
    newConversation: ReturnType<typeof vi.fn>
    renameConversation: ReturnType<typeof vi.fn>
    deleteConversation: ReturnType<typeof vi.fn>
    selectKey: ReturnType<typeof vi.fn>
    selectModel: ReturnType<typeof vi.fn>
    send: ReturnType<typeof vi.fn>
    retry: ReturnType<typeof vi.fn>
    stop: ReturnType<typeof vi.fn>
    activeConversation: () => ChatConversation | null
  },
}))

vi.mock('../useChatSession', async () => {
  const { ref } = await import('vue')
  session.conversations = ref([])
  session.activeId = ref(null)
  session.keys = ref([])
  session.models = ref([])
  session.selectedKeyId = ref(null)
  session.selectedModel = ref('')
  session.draft = ref('')
  session.streaming = ref(false)
  session.loading = ref(false)
  session.modelsLoading = ref(false)
  session.sendError = ref(null)
  session.persistenceWarning = ref(false)
  session.load = vi.fn()
  session.selectConversation = vi.fn()
  session.newConversation = vi.fn()
  session.renameConversation = vi.fn()
  session.deleteConversation = vi.fn()
  session.selectKey = vi.fn()
  session.selectModel = vi.fn()
  session.send = vi.fn()
  session.retry = vi.fn()
  session.stop = vi.fn()
  session.activeConversation = () => session.conversations.value.find((item) => item.id === session.activeId.value) ?? null
  return {
    useChatSession: () => session,
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ isAdmin: false, user: { id: 7 } }),
}))

const RouterLinkStub = defineComponent({
  props: {
    to: { type: [String, Object], required: true },
  },
  template: '<a :href="typeof to === \'string\' ? to : to.path"><slot /></a>',
})

async function mountChat() {
  const [{ mount }, { createI18n }, { default: ChatView }] = await Promise.all([
    import('@vue/test-utils'),
    import('vue-i18n'),
    import('../ChatView.vue'),
  ])
  const i18n = createI18n({
    legacy: false,
    locale: 'zh',
    messages: {
      zh: {
        chat: {
          title: '聊天',
          nav: '聊天',
          console: '控制台',
          newChat: '新对话',
          greeting: '今天想聊什么',
          composerPlaceholder: '写一条消息',
          send: '发送',
          stop: '停止',
          retry: '重试',
          copy: '复制',
          copied: '已复制',
          rename: '重命名',
          delete: '删除',
          confirmDelete: '删除这个对话？',
          key: '密钥',
          model: '模型',
          noKey: '还没有可用的 API 密钥。',
          createKey: '去创建密钥',
          noModel: '这把密钥暂时没有可用模型。',
          noConversations: '还没有对话',
          persistenceWarning: '当前浏览器无法保存会话，刷新后会丢失。',
          loading: '正在打开聊天',
          modelsLoading: '正在读取模型',
          emptyAssistant: '没有回复',
          you: '你',
          assistant: '助手',
          errors: {
            invalid_key: '这把密钥无效或已停用，请换一把再试。',
            billing: '余额或配额不足。',
            rate_limit: '请求过于频繁，请稍后再试。',
            upstream: '模型没有返回结果。',
          },
        },
        nav: { lightMode: '浅色', darkMode: '深色' },
      },
    },
  })
  return mount(ChatView, {
    global: {
      plugins: [i18n],
      stubs: { RouterLink: RouterLinkStub },
    },
  })
}

describe('ChatView', () => {
  beforeEach(async () => {
    await import('../ChatView.vue')
    session.conversations.value = []
    session.keys.value = []
    session.models.value = []
    session.selectedKeyId.value = null
    session.selectedModel.value = ''
    session.draft.value = ''
    session.activeId.value = null
    session.sendError.value = null
    session.streaming.value = false
    session.loading.value = false
  })

  it('sends users without a key to create one', async () => {
    const wrapper = await mountChat()
    expect(wrapper.text()).toContain('chat.noKey')
    expect(wrapper.get('a[href="/keys"]').text()).toContain('chat.createKey')
  })

  it('shows the quiet greeting when a key exists and the thread is empty', async () => {
    session.keys.value = [{ id: 40, name: 'main', status: 'active' } as ApiKey]
    session.selectedKeyId.value = 40
    session.selectedModel.value = 'claude-sonnet-4'
    session.models.value = ['claude-sonnet-4']
    const wrapper = await mountChat()
    expect(wrapper.text()).toContain('chat.greeting')
    expect(wrapper.get('button.chat-send').attributes('disabled')).toBeDefined()
  })

  it('renders mapped billing errors', async () => {
    session.keys.value = [{ id: 40, name: 'main', status: 'active' } as ApiKey]
    session.selectedKeyId.value = 40
    session.selectedModel.value = 'claude-sonnet-4'
    session.sendError.value = { code: 'billing', message: 'insufficient balance' }
    const wrapper = await mountChat()
    expect(wrapper.text()).toContain('insufficient balance')
  })
})
