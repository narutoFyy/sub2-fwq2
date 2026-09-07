<template>
  <div class="chat-shell">
    <div class="flex min-h-screen">
      <div
        v-if="railOpen"
        class="fixed inset-0 z-20 bg-black/20 md:hidden"
        @click="railOpen = false"
      />
      <aside
        class="z-30 flex w-[17.5rem] shrink-0 flex-col border-r px-3 py-4 max-md:fixed max-md:inset-y-0 max-md:left-0"
        :class="railOpen ? 'max-md:flex' : 'max-md:hidden md:flex'"
        :style="{ borderColor: 'var(--chat-line)', background: 'var(--chat-bg-raised)' }"
      >
        <div class="mb-5 flex items-center justify-between gap-2 px-1">
          <router-link :to="consolePath" class="text-[13px] tracking-wide" :style="{ color: 'var(--chat-muted)' }">
            {{ t('chat.console') }}
          </router-link>
          <div class="flex items-center gap-1">
            <button type="button" class="chat-icon-btn" :title="isDark ? t('nav.lightMode') : t('nav.darkMode')" @click="toggleTheme">
              {{ isDark ? 'Light' : 'Dark' }}
            </button>
            <button type="button" class="chat-icon-btn" @click="toggleLocale">
              {{ locale === 'zh' ? 'EN' : '中' }}
            </button>
          </div>
        </div>

        <button type="button" class="chat-new-btn mb-4" @click="newConversation">
          {{ t('chat.newChat') }}
        </button>

        <p v-if="!conversations.length" class="px-2 text-sm" :style="{ color: 'var(--chat-muted)' }">
          {{ t('chat.noConversations') }}
        </p>
        <nav class="min-h-0 flex-1 space-y-0.5 overflow-y-auto">
          <div
            v-for="item in conversations"
            :key="item.id"
            class="group flex items-center rounded-lg px-2 py-2 text-sm"
            :class="item.id === activeId ? 'chat-conv-active' : 'chat-conv'"
          >
            <button
              type="button"
              class="min-w-0 flex-1 truncate text-left"
              @click="selectConversation(item.id)"
              @dblclick="onRename(item)"
            >
              {{ item.title }}
            </button>
            <button
              type="button"
              class="ml-1 hidden text-[11px] group-hover:inline"
              :style="{ color: 'var(--chat-muted)' }"
              @click.stop="onDelete(item.id)"
            >
              {{ t('chat.delete') }}
            </button>
          </div>
        </nav>
      </aside>

      <main class="flex min-w-0 flex-1 flex-col">
        <header class="flex flex-wrap items-center gap-3 border-b px-6 py-3" :style="{ borderColor: 'var(--chat-line)' }">
          <button type="button" class="chat-icon-btn md:hidden" @click="railOpen = true">{{ t('chat.nav') }}</button>
          <label class="flex min-w-[12rem] flex-1 items-center gap-2 text-xs" :style="{ color: 'var(--chat-muted)' }">
            <span class="shrink-0 whitespace-nowrap">{{ t('chat.key') }}</span>
            <select
              class="chat-select"
              :value="selectedKeyId ?? ''"
              :disabled="!keys.length"
              @change="onKeyChange"
            >
              <option v-if="!keys.length" value="">{{ t('chat.noKey') }}</option>
              <option v-for="item in keys" :key="item.id" :value="item.id">{{ item.name }}</option>
            </select>
          </label>
          <label class="flex min-w-[12rem] flex-1 items-center gap-2 text-xs" :style="{ color: 'var(--chat-muted)' }">
            <span class="shrink-0 whitespace-nowrap">{{ t('chat.model') }}</span>
            <select
              class="chat-select"
              :value="selectedModel"
              :disabled="!models.length"
              @change="onModelChange"
            >
              <option v-if="modelsLoading" value="">{{ t('chat.modelsLoading') }}</option>
              <option v-else-if="!models.length" value="">{{ t('chat.noModel') }}</option>
              <option v-for="model in models" :key="model" :value="model">{{ model }}</option>
            </select>
          </label>
        </header>

        <p v-if="persistenceWarning" class="px-6 pt-3 text-xs" :style="{ color: 'var(--chat-muted)' }">
          {{ t('chat.persistenceWarning') }}
        </p>

        <section ref="scrollerRef" class="min-h-0 flex-1 overflow-y-auto px-6 py-8">
          <div v-if="loading" class="pt-24 text-center text-sm" :style="{ color: 'var(--chat-muted)' }">
            {{ t('chat.loading') }}
          </div>
          <div v-else-if="!keys.length" class="mx-auto max-w-xl pt-24">
            <p class="chat-greeting">{{ t('chat.noKey') }}</p>
            <router-link to="/keys" class="mt-4 inline-block text-sm">{{ t('chat.createKey') }}</router-link>
          </div>
          <div v-else-if="!active?.messages.length" class="mx-auto max-w-2xl pt-24">
            <p class="chat-greeting">{{ t('chat.greeting') }}</p>
          </div>
          <ol v-else class="mx-auto max-w-2xl list-none space-y-8 p-0">
            <li v-for="message in active.messages" :key="message.id" class="group">
              <div class="mb-2 flex items-baseline justify-between gap-3 text-[11px] uppercase tracking-[0.14em]" :style="{ color: 'var(--chat-muted)' }">
                <span>{{ message.role === 'user' ? t('chat.you') : t('chat.assistant') }}</span>
                <button
                  v-if="message.content"
                  type="button"
                  class="opacity-0 group-hover:opacity-100"
                  @click="copyMessage(message.content)"
                >
                  {{ copied === message.content ? t('chat.copied') : t('chat.copy') }}
                </button>
              </div>
              <div
                v-if="message.role === 'assistant'"
                class="chat-md"
                v-html="renderChatMarkdown(message.content || (message.status === 'streaming' ? '' : t('chat.emptyAssistant')))"
              />
              <div v-else class="chat-user">{{ message.content }}</div>
              <span v-if="message.status === 'streaming'" class="chat-caret" aria-hidden="true" />
            </li>
          </ol>
        </section>

        <footer class="px-6 pb-6 pt-2">
          <p v-if="sendError" class="mx-auto mb-2 max-w-2xl text-sm" :style="{ color: 'var(--chat-danger)' }">
            {{ errorText }}
          </p>
          <form class="chat-composer mx-auto max-w-2xl" @submit.prevent="onSend">
            <textarea
              v-model="draft"
              rows="1"
              class="chat-input"
              :placeholder="t('chat.composerPlaceholder')"
              :disabled="!keys.length || streaming"
              @keydown="onComposerKeydown"
              @input="resizeComposer"
            />
            <div class="flex items-center justify-end gap-2 px-2 pb-2">
              <button v-if="canRetry" type="button" class="chat-text-btn" @click="retry">{{ t('chat.retry') }}</button>
              <button v-if="streaming" type="button" class="chat-text-btn" @click="stop">{{ t('chat.stop') }}</button>
              <button v-else type="submit" class="chat-send" :disabled="!canSend">{{ t('chat.send') }}</button>
            </div>
          </form>
        </footer>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { getLocale, setLocale } from '@/i18n'
import { renderChatMarkdown } from './markdown'
import { useChatSession } from './useChatSession'
import './chat.css'

const { t, locale } = useI18n()
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
const copied = ref('')
const railOpen = ref(false)
const isDark = ref(document.documentElement.classList.contains('dark'))

const consolePath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const active = computed(() => session.activeConversation())
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

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

async function toggleLocale() {
  await setLocale(getLocale() === 'zh' ? 'en' : 'zh')
}

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
  if (event.key === 'Enter' && !event.shiftKey) {
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

function resizeComposer(event?: Event) {
  const el = (event?.target as HTMLTextAreaElement | undefined) ?? document.querySelector<HTMLTextAreaElement>('.chat-input')
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, 200)}px`
}

function scrollToBottom() {
  const root = scrollerRef.value
  if (!root) return
  root.scrollTop = root.scrollHeight
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

function newConversation() {
  void session.newConversation()
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

watch(
  () => active.value?.messages.map((item) => item.content).join('\0'),
  () => {
    void nextTick(scrollToBottom)
  },
)
</script>

<style scoped>
.chat-icon-btn,
.chat-text-btn {
  border: 0;
  background: transparent;
  color: var(--chat-muted);
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  cursor: pointer;
}

.chat-new-btn,
.chat-send {
  border: 1px solid var(--chat-line);
  background: transparent;
  color: var(--chat-ink);
  border-radius: 999px;
  padding: 0.55rem 0.9rem;
  font-size: 0.8125rem;
  cursor: pointer;
}

.chat-new-btn {
  width: 100%;
  text-align: left;
}

.chat-send:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.chat-select {
  width: 100%;
  border: 0;
  border-bottom: 1px solid var(--chat-line);
  background: transparent;
  color: var(--chat-ink);
  padding: 0.35rem 0;
  font-size: 0.875rem;
}

.chat-conv {
  color: var(--chat-muted);
}

.chat-conv-active {
  background: color-mix(in oklch, var(--chat-line) 55%, transparent);
  color: var(--chat-ink);
}

.chat-greeting {
  font-family: var(--chat-serif);
  font-size: clamp(2rem, 4vw, 3.25rem);
  font-weight: 400;
  line-height: 1.2;
  letter-spacing: -0.03em;
}

.chat-user {
  font-family: var(--chat-serif);
  font-size: 1.0625rem;
  line-height: 1.7;
  white-space: pre-wrap;
}

.chat-composer {
  border: 1px solid var(--chat-line);
  border-radius: 1.15rem;
  background: var(--chat-bg-raised);
}

.chat-input {
  width: 100%;
  resize: none;
  border: 0;
  background: transparent;
  color: var(--chat-ink);
  padding: 0.95rem 1rem 0.35rem;
  font-family: var(--chat-serif);
  font-size: 1.05rem;
  line-height: 1.6;
  outline: none;
}

.chat-caret {
  display: inline-block;
  width: 0.45rem;
  height: 1.05em;
  margin-left: 0.15rem;
  background: var(--chat-ink);
  animation: chat-blink 1s steps(1) infinite;
  vertical-align: text-bottom;
}

@keyframes chat-blink {
  50% {
    opacity: 0;
  }
}
</style>
