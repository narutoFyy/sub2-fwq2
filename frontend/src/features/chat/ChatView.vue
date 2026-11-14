<template>
  <div class="chat-shell" :class="{ 'is-fortune': mode === 'fortune' }">
    <div class="chat-glow" aria-hidden="true" />
    <div class="relative flex h-screen min-h-0">
      <Transition name="chat-fade">
        <div v-if="railOpen" class="chat-scrim md:hidden" @click="railOpen = false" />
      </Transition>

      <!-- 侧栏：品牌、新建、会话列表 -->
      <aside class="chat-rail" :class="{ 'is-open': railOpen }">
        <div class="flex items-center justify-between gap-2 px-2 pb-5 pt-1">
          <router-link :to="consolePath" class="chat-brand" :title="t('chat.console')">
            <BrandLogo :name="brandName" :size="22" />
          </router-link>
          <div class="flex items-center gap-0.5">
            <LocaleToggle />
            <ThemeToggleIcon />
          </div>
        </div>

        <div class="grid gap-2 px-1">
          <button type="button" class="chat-new-btn" @click="newConversation('chat')">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
            <span>{{ t('chat.newChat') }}</span>
          </button>
          <button type="button" class="chat-new-btn chat-new-btn--fortune" @click="newConversation('fortune')">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 14.5A8.5 8.5 0 1 1 9.5 4a6.8 6.8 0 0 0 10.5 10.5z" /></svg>
            <span>{{ t('chat.fortuneNew') }}</span>
          </button>
        </div>

        <p v-if="!conversations.length" class="mt-6 px-3 text-[13px]" :style="{ color: 'var(--chat-muted)' }">
          {{ t('chat.noConversations') }}
        </p>
        <nav class="chat-conv-list">
          <div
            v-for="item in conversations"
            :key="item.id"
            class="chat-conv group"
            :class="{ 'is-active': item.id === activeId }"
          >
            <button
              type="button"
              class="flex min-w-0 flex-1 items-center gap-2 text-left"
              @click="selectConversation(item.id)"
              @dblclick="onRename(item)"
            >
              <span class="chat-conv-dot" :class="{ 'is-fortune': item.mode === 'fortune' }" aria-hidden="true" />
              <span class="truncate">{{ titleOf(item) }}</span>
            </button>
            <button
              type="button"
              class="chat-conv-del"
              :title="t('chat.delete')"
              :aria-label="t('chat.delete')"
              @click.stop="onDelete(item.id)"
            >
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 7h14M10 7V5h4v2M7 7l1 12h8l1-12" /></svg>
            </button>
          </div>
        </nav>

        <router-link :to="consolePath" class="chat-console-link">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M15 6l-6 6 6 6" /></svg>
          {{ t('chat.console') }}
        </router-link>
      </aside>

      <main class="flex min-w-0 flex-1 flex-col">
        <!-- 顶栏：会话标题、密钥、模型 -->
        <header class="chat-header">
          <button type="button" class="chat-icon-btn chat-menu-btn" :aria-label="t('chat.nav')" @click="railOpen = true">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h16M4 12h16M4 17h10" /></svg>
          </button>
          <div class="chat-header-title flex min-w-0 flex-1 items-center gap-2">
            <span v-if="mode === 'fortune'" class="chat-badge">{{ t('chat.fortuneBadge') }}</span>
            <span class="truncate text-sm font-medium">{{ headerTitle }}</span>
          </div>
          <label class="chat-pill-select" :title="t('chat.key')">
            <span class="chat-pill-label">{{ t('chat.key') }}</span>
            <select :value="selectedKeyId ?? ''" :disabled="!keys.length" @change="onKeyChange">
              <option v-if="!keys.length" value="">{{ t('chat.noKey') }}</option>
              <option v-for="item in keys" :key="item.id" :value="item.id">{{ item.name }}</option>
            </select>
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 10l5 5 5-5" /></svg>
          </label>
          <label class="chat-pill-select chat-pill-select--model" :title="t('chat.model')">
            <span class="chat-pill-label">{{ t('chat.model') }}</span>
            <select :value="selectedModel" :disabled="!models.length" @change="onModelChange">
              <option v-if="modelsLoading" value="">{{ t('chat.modelsLoading') }}</option>
              <option v-else-if="!models.length" value="">{{ t('chat.noModel') }}</option>
              <option v-for="model in models" :key="model" :value="model">{{ model }}</option>
            </select>
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 10l5 5 5-5" /></svg>
          </label>
        </header>

        <p v-if="persistenceWarning" class="chat-notice">{{ t('chat.persistenceWarning') }}</p>

        <section ref="scrollerRef" class="chat-scroller">
          <div v-if="loading" class="chat-empty">
            <span class="chat-typing" aria-hidden="true"><i /><i /><i /></span>
            <p class="mt-4 text-sm" :style="{ color: 'var(--chat-muted)' }">{{ t('chat.loading') }}</p>
          </div>

          <div v-else-if="!keys.length" class="chat-empty">
            <p class="chat-greeting">{{ t('chat.noKey') }}</p>
            <router-link to="/keys" class="chat-cta">{{ t('chat.createKey') }}</router-link>
          </div>

          <div v-else-if="!active?.messages.length" class="chat-empty">
            <div class="chat-orb" aria-hidden="true">
              <svg v-if="mode === 'fortune'" viewBox="0 0 24 24"><path d="M20 14.5A8.5 8.5 0 1 1 9.5 4a6.8 6.8 0 0 0 10.5 10.5z" /><path d="M16.5 3.5l.6 1.4 1.4.6-1.4.6-.6 1.4-.6-1.4-1.4-.6 1.4-.6z" /></svg>
              <svg v-else viewBox="0 0 24 24"><path d="M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z" /></svg>
            </div>
            <p class="chat-greeting">{{ mode === 'fortune' ? t('chat.fortuneGreeting') : t('chat.greeting') }}</p>
            <p class="chat-subtitle">{{ mode === 'fortune' ? t('chat.fortuneSubtitle') : t('chat.chatSubtitle') }}</p>
            <div class="chat-suggestions" :class="{ 'is-six': suggestions.length > 4 }">
              <button
                v-for="(item, i) in suggestions"
                :key="item.title"
                type="button"
                class="chat-suggestion"
                :style="{ '--i': i }"
                @click="useSuggestion(item.prompt)"
              >
                <strong>{{ item.title }}</strong>
                <span>{{ item.prompt }}</span>
              </button>
            </div>
          </div>

          <ol v-else class="chat-thread">
            <li
              v-for="message in active.messages"
              :key="message.id"
              class="chat-msg group"
              :class="message.role === 'user' ? 'is-user' : 'is-assistant'"
            >
              <template v-if="message.role === 'assistant'">
                <div class="chat-avatar" :class="{ 'is-fortune': mode === 'fortune' }" aria-hidden="true">
                  <svg v-if="mode === 'fortune'" viewBox="0 0 24 24"><path d="M20 14.5A8.5 8.5 0 1 1 9.5 4a6.8 6.8 0 0 0 10.5 10.5z" /></svg>
                  <svg v-else viewBox="0 0 24 24"><path d="M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z" /></svg>
                </div>
                <div class="min-w-0 flex-1">
                  <div class="chat-msg-name">{{ mode === 'fortune' ? t('chat.fortuneName') : t('chat.assistant') }}</div>
                  <span v-if="message.status === 'streaming' && !message.content" class="chat-typing" aria-hidden="true"><i /><i /><i /></span>
                  <div
                    v-else
                    class="chat-md"
                    :class="{ 'is-error': message.status === 'error' }"
                    v-html="renderChatMarkdown(message.content || (message.status === 'streaming' ? '' : t('chat.emptyAssistant')))"
                  />
                  <span v-if="message.status === 'streaming' && message.content" class="chat-caret" aria-hidden="true" />
                  <div v-if="message.content && message.status !== 'streaming'" class="chat-msg-actions">
                    <button type="button" @click="copyMessage(message.content)">
                      {{ copied === message.content ? t('chat.copied') : t('chat.copy') }}
                    </button>
                  </div>
                </div>
              </template>
              <template v-else>
                <div class="chat-user">{{ message.content }}</div>
              </template>
            </li>
          </ol>
        </section>

        <footer class="chat-footer">
          <p v-if="sendError" class="chat-error">{{ errorText }}</p>
          <form class="chat-composer" @submit.prevent="onSend">
            <textarea
              ref="inputRef"
              v-model="draft"
              rows="1"
              class="chat-input"
              :placeholder="mode === 'fortune' ? t('chat.fortunePlaceholder') : t('chat.composerPlaceholder')"
              :disabled="!keys.length || streaming"
              @keydown="onComposerKeydown"
              @input="resizeComposer"
            />
            <div class="flex items-center justify-between gap-2 px-3 pb-2.5">
              <span class="chat-hint">{{ t('chat.sendHint') }}</span>
              <div class="flex items-center gap-1.5">
                <button v-if="canRetry" type="button" class="chat-text-btn" @click="retry">{{ t('chat.retry') }}</button>
                <button v-if="streaming" type="button" class="chat-send is-stop" :aria-label="t('chat.stop')" :title="t('chat.stop')" @click="stop">
                  <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="7" y="7" width="10" height="10" rx="2" /></svg>
                </button>
                <button v-else type="submit" class="chat-send" :disabled="!canSend" :aria-label="t('chat.send')" :title="t('chat.send')">
                  <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 19V5M6 11l6-6 6 6" /></svg>
                </button>
              </div>
            </div>
          </form>
        </footer>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import BrandLogo from '@/components/brand/BrandLogo.vue'
import LocaleToggle from '@/components/brand/LocaleToggle.vue'
import ThemeToggleIcon from '@/components/common/ThemeToggleIcon.vue'
import { renderChatMarkdown } from './markdown'
import { suggestionsFor } from './suggestions'
import type { ChatMode } from './types'
import { useChatSession } from './useChatSession'
import './chat.css'

const { t, locale } = useI18n()
const route = useRoute() as ReturnType<typeof useRoute> | undefined
const router = useRouter() as ReturnType<typeof useRouter> | undefined
const authStore = useAuthStore()
const session = useChatSession()
const {
  conversations,
  activeId,
  keys,
  models,
  selectedKeyId,
  selectedModel,
  draft,
  streaming,
  loading,
  modelsLoading,
  sendError,
  persistenceWarning,
} = session

const scrollerRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLTextAreaElement | null>(null)
const copied = ref('')
const railOpen = ref(false)

const brandName = computed(() => {
  const name = t('home.landing.brandName')
  return name === 'home.landing.brandName' ? 'yf-mail' : name
})
const consolePath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const active = computed(() => session.activeConversation())
const mode = computed<ChatMode>(() => active.value?.mode ?? 'chat')
const suggestions = computed(() => suggestionsFor(locale.value, mode.value))
const headerTitle = computed(() => {
  const conv = active.value
  if (conv && conv.messages.length) return conv.title
  return mode.value === 'fortune' ? t('chat.fortuneNew') : t('chat.newChat')
})

/** 还没发消息的对话标题是存储层的占位 "New chat"，按模式显示成当前语言的名字。 */
function titleOf(item: { title: string; mode: ChatMode; messages: unknown[] }) {
  if (item.messages.length || item.title !== 'New chat') return item.title
  return item.mode === 'fortune' ? t('chat.fortuneNew') : t('chat.newChat')
}
const canSend = computed(() => Boolean(draft.value.trim() && keys.value.length && selectedModel.value && !streaming.value))
const canRetry = computed(() => {
  const last = active.value?.messages.at(-1)
  return Boolean(last && last.role === 'assistant' && last.status !== 'streaming' && !streaming.value)
})
const errorText = computed(() => {
  if (!sendError.value) return ''
  const mapped = t(`chat.errors.${sendError.value.code}`)
  return mapped === `chat.errors.${sendError.value.code}` ? sendError.value.message : mapped
})

function onKeyChange(event: Event) {
  const value = Number((event.target as HTMLSelectElement).value)
  if (Number.isFinite(value) && value > 0) {
    void session.selectKey(value)
  }
}

function onModelChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  if (value) {
    void session.selectModel(value)
  }
}

function onComposerKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
    event.preventDefault()
    void onSend()
  }
}

async function onSend() {
  await session.send()
  await nextTick()
  resizeComposer()
  scrollToBottom()
}

function retry() {
  void session.retry()
}

function stop() {
  session.stop()
}

function resizeComposer() {
  const el = inputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, 220)}px`
}

function scrollToBottom() {
  const root = scrollerRef.value
  if (!root) return
  root.scrollTop = root.scrollHeight
}

async function useSuggestion(prompt: string) {
  draft.value = prompt
  await nextTick()
  resizeComposer()
  const el = inputRef.value
  if (el) {
    el.focus()
    el.setSelectionRange(prompt.length, prompt.length)
  }
}

async function copyMessage(content: string) {
  try {
    await navigator.clipboard.writeText(content)
    copied.value = content
    window.setTimeout(() => {
      if (copied.value === content) copied.value = ''
    }, 1200)
  } catch {
    copied.value = ''
  }
}

async function onDelete(id: string) {
  if (window.confirm(t('chat.confirmDelete'))) {
    await session.deleteConversation(id)
  }
}

async function newConversation(next: ChatMode) {
  railOpen.value = false
  // 当前就是同类型的空对话：不再新建一个，直接用它。
  const current = active.value
  if (current && !current.messages.length && current.mode === next) return
  await session.newConversation(next)
}

function selectConversation(id: string) {
  railOpen.value = false
  void session.selectConversation(id)
}

async function onRename(item: { id: string; title: string }) {
  const next = window.prompt(t('chat.rename'), item.title)
  if (next != null) {
    await session.renameConversation(item.id, next)
  }
}

/** 从新手引导或外部链接带 ?mode=fortune 进来：加载完成后开一个玄学测算对话，再把参数去掉，刷新不会重复新建。 */
async function applyRequestedMode() {
  const requested = route?.query.mode
  if (requested !== 'fortune' && requested !== 'chat') return
  await newConversation(requested)
  void router?.replace({ path: '/chat' })
}

watch(
  loading,
  (now, before) => {
    if (before && !now) void applyRequestedMode()
  },
)
// 已在聊天页时再从引导跳过来（只改了查询参数）。
watch(
  () => route?.query.mode,
  (value) => {
    if (value && !loading.value) void applyRequestedMode()
  },
)

watch(
  () => active.value?.messages.map((item) => item.content).join('\0'),
  () => {
    void nextTick(scrollToBottom)
  },
)
</script>
