<template>
  <Teleport to="body">
    <Transition name="oauth-monitor-drawer">
      <div
        v-if="show"
        class="fixed inset-0 z-[70] bg-gray-950/45"
        role="presentation"
        @mousedown.self="emit('close')"
      >
        <aside
          ref="drawerRef"
          class="absolute inset-y-0 right-0 flex w-full max-w-[720px] flex-col border-l border-gray-200 bg-white shadow-2xl dark:border-dark-600 dark:bg-dark-900"
          role="dialog"
          aria-modal="true"
          aria-labelledby="oauth-monitor-title"
          tabindex="-1"
        >
          <header class="flex min-h-[72px] flex-none items-center justify-between border-b border-gray-200 px-5 dark:border-dark-600 sm:px-6">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <Icon name="shield" size="md" class="text-primary-600" />
                <h2 id="oauth-monitor-title" class="truncate text-lg font-semibold text-gray-950 dark:text-white">
                  OAuth 账号监控
                </h2>
              </div>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ selectedSummary }}
              </p>
            </div>
            <div class="ml-4 flex flex-none items-center gap-1">
              <button
                type="button"
                class="inline-flex h-9 w-9 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-dark-700 dark:hover:text-white"
                :disabled="loading || running"
                title="刷新监控状态"
                aria-label="刷新监控状态"
                @click="emit('refresh')"
              >
                <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              </button>
              <button
                type="button"
                class="inline-flex h-9 w-9 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-dark-700 dark:hover:text-white"
                aria-label="关闭 OAuth 监控"
                @click="emit('close')"
              >
                <Icon name="x" size="md" />
              </button>
            </div>
          </header>

          <div class="min-h-0 flex-1 overflow-y-auto">
            <div v-if="error || notice" class="space-y-2 px-5 pt-5 sm:px-6">
              <div v-if="error" class="flex items-start gap-3 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800 dark:border-red-800/60 dark:bg-red-950/30 dark:text-red-200">
                <Icon name="exclamationTriangle" size="sm" class="mt-0.5 flex-none" />
                <div class="min-w-0 flex-1">
                  <p class="break-words">{{ error }}</p>
                  <button type="button" class="mt-2 font-medium underline underline-offset-2" :disabled="loading" @click="emit('retry')">
                    重新加载
                  </button>
                </div>
              </div>
              <div v-if="notice" class="flex items-start gap-3 rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800 dark:border-emerald-800/60 dark:bg-emerald-950/30 dark:text-emerald-200">
                <Icon name="checkCircle" size="sm" class="mt-0.5 flex-none" />
                <p class="min-w-0 flex-1 break-words">{{ notice }}</p>
              </div>
            </div>

            <div v-if="loading && !overview" class="space-y-4 p-5 sm:p-6">
              <div v-for="index in 4" :key="index" class="h-24 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-700" />
            </div>

            <div v-else-if="!overview" class="px-5 py-10 text-center sm:px-6">
              <Icon name="shield" size="xl" class="mx-auto text-gray-300 dark:text-gray-600" />
              <p class="mt-3 text-sm font-medium text-gray-800 dark:text-gray-200">监控数据暂未加载</p>
              <button type="button" class="btn btn-secondary mt-4" :disabled="loading" @click="emit('retry')">
                重新加载
              </button>
            </div>

            <form v-else id="oauth-monitor-form" @submit.prevent="submit">
              <section class="border-b border-gray-200 px-5 py-5 dark:border-dark-600 sm:px-6">
                <div class="grid grid-cols-3 gap-px overflow-hidden rounded-lg border border-gray-200 bg-gray-200 dark:border-dark-600 dark:bg-dark-600">
                  <div class="bg-white px-3 py-3 dark:bg-dark-900">
                    <div class="text-xs text-gray-500 dark:text-gray-400">全局状态</div>
                    <div class="mt-1 flex items-center gap-1.5 text-sm font-semibold" :class="form.enabled ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">
                      <span class="h-2 w-2 rounded-full" :class="form.enabled ? 'bg-emerald-500' : 'bg-amber-500'" />
                      {{ form.enabled ? '运行中' : '已暂停' }}
                    </div>
                  </div>
                  <div class="bg-white px-3 py-3 dark:bg-dark-900">
                    <div class="text-xs text-gray-500 dark:text-gray-400">监控账号</div>
                    <div class="mt-1 text-sm font-semibold tabular-nums text-gray-900 dark:text-white">{{ monitoredCount }}</div>
                  </div>
                  <div class="bg-white px-3 py-3 dark:bg-dark-900">
                    <div class="text-xs text-gray-500 dark:text-gray-400">生效规则</div>
                    <div class="mt-1 text-sm font-semibold tabular-nums text-gray-900 dark:text-white">
                      {{ form.quota_threshold_percent }}% / {{ form.interval_minutes }} 分钟
                    </div>
                  </div>
                </div>

                <label class="mt-4 flex items-center justify-between gap-4">
                  <span>
                    <span class="block text-sm font-semibold text-gray-900 dark:text-white">启用全局监控</span>
                    <span class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">关闭后保留名单，所有账号显示为“已加入 · 全局暂停”。</span>
                  </span>
                  <input v-model="form.enabled" type="checkbox" class="h-5 w-5 flex-none rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                </label>
              </section>

              <section class="border-b border-gray-200 dark:border-dark-600">
                <div class="flex items-center justify-between px-5 pb-3 pt-5 sm:px-6">
                  <div>
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">账号名单</h3>
                    <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">开关表示该账号是否加入监控，不会改变账号启用和调度状态。</p>
                  </div>
                  <span class="text-xs tabular-nums text-gray-500 dark:text-gray-400">{{ overview.accounts.length }} 个账号</span>
                </div>

                <div v-if="overview.accounts.length === 0" class="px-5 pb-6 sm:px-6">
                  <div class="rounded-lg border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
                    当前没有已监控或选中的 OAuth 账号。
                  </div>
                </div>

                <div v-else class="divide-y divide-gray-100 border-t border-gray-100 dark:divide-dark-700 dark:border-dark-700">
                  <div v-for="account in overview.accounts" :key="account.account_id" class="px-5 py-4 sm:px-6">
                    <div class="flex items-start gap-3">
                      <input
                        :checked="isMonitored(account.account_id)"
                        :disabled="!account.eligible"
                        type="checkbox"
                        class="mt-0.5 h-4 w-4 flex-none rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-40"
                        :aria-label="`设置账号 ${account.account_name} 的监控状态`"
                        @change="toggleAccount(account.account_id, ($event.target as HTMLInputElement).checked)"
                      />
                      <div class="min-w-0 flex-1">
                        <div class="flex flex-wrap items-center gap-2">
                          <span class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ account.account_name }}</span>
                          <span class="text-xs tabular-nums text-gray-400">#{{ account.account_id }}</span>
                          <span
                            class="rounded px-1.5 py-0.5 text-[11px] font-medium"
                            :class="statusMeta(account).className"
                          >
                            {{ statusMeta(account).label }}
                          </span>
                          <span v-if="account.selected" class="rounded bg-blue-50 px-1.5 py-0.5 text-[11px] font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">本次选中</span>
                        </div>
                        <div v-if="account.eligible" class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                          <span>主窗口 {{ formatRemaining(account.state?.primary_remaining_percent) }}</span>
                          <span>次窗口 {{ formatRemaining(account.state?.secondary_remaining_percent) }}</span>
                          <span>最近检查 {{ formatCheckedAt(account.state?.last_checked_at) }}</span>
                        </div>
                        <p v-if="account.state?.error_message" class="mt-2 line-clamp-2 text-xs text-red-600 dark:text-red-400" :title="account.state.error_message">
                          {{ account.state.error_message }}
                        </p>
                        <p v-else-if="!account.eligible" class="mt-2 text-xs text-amber-700 dark:text-amber-300">
                          仅 OpenAI/Codex OAuth 账号可加入监控名单。
                        </p>
                      </div>
                    </div>
                  </div>
                </div>
              </section>

              <section class="border-b border-gray-200 px-5 py-5 dark:border-dark-600 sm:px-6">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">检查与告警规则</h3>
                <div class="mt-4 grid gap-4 sm:grid-cols-2">
                  <label class="input-label">
                    额度剩余阈值 (%)
                    <input v-model.number="form.quota_threshold_percent" type="number" min="0" max="100" step="1" class="input mt-1 w-full" />
                  </label>
                  <label class="input-label">
                    检查间隔 (分钟)
                    <input v-model.number="form.interval_minutes" type="number" min="1" max="1440" step="1" class="input mt-1 w-full" />
                  </label>
                </div>
                <div class="mt-4 grid gap-3 sm:grid-cols-2">
                  <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"><input v-model="form.notify_on_error" type="checkbox" class="rounded text-primary-600" />账号错误时告警</label>
                  <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"><input v-model="form.notify_on_quota" type="checkbox" class="rounded text-primary-600" />额度低于阈值时告警</label>
                  <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"><input v-model="form.notify_on_recovery" type="checkbox" class="rounded text-primary-600" />恢复时发送通知</label>
                  <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"><input v-model="form.repeat_every_check" type="checkbox" class="rounded text-primary-600" />持续异常每次检查通知</label>
                </div>
              </section>

              <section class="px-5 py-5 sm:px-6">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">通知渠道</h3>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">OAuth 告警的邮件收件人与 PushPlus 均在这里维护。</p>

                <div class="mt-4 grid gap-4 lg:grid-cols-2">
                  <article class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
                    <div class="flex items-start justify-between gap-3">
                      <div class="flex items-center gap-2">
                        <Icon name="mail" size="sm" class="text-blue-600 dark:text-blue-400" />
                        <div>
                          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">邮件通知</h4>
                          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">使用现有 SMTP 服务发送。</p>
                        </div>
                      </div>
                      <input v-model="emailChannelEnabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />
                    </div>

                    <div class="mt-4">
                      <label class="input-label">管理员收件人</label>
                      <div class="mt-1 flex gap-2">
                        <input
                          v-model="recipientInput"
                          type="email"
                          class="input min-w-0 flex-1"
                          placeholder="admin@example.com"
                          @keydown.enter.prevent="addRecipient"
                        />
                        <button type="button" class="btn btn-secondary px-3" @click="addRecipient">添加</button>
                      </div>
                      <p v-if="recipientError" class="mt-1 text-xs text-red-600 dark:text-red-400">{{ recipientError }}</p>
                      <div v-if="email.alert.recipients.length" class="mt-3 flex flex-wrap gap-2">
                        <span v-for="recipient in email.alert.recipients" :key="recipient" class="inline-flex items-center gap-1.5 rounded bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200">
                          {{ recipient }}
                          <button type="button" class="text-gray-400 hover:text-red-600" :aria-label="`移除 ${recipient}`" @click="removeRecipient(recipient)">
                            <Icon name="x" size="xs" />
                          </button>
                        </span>
                      </div>
                    </div>
                  </article>

                  <article class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
                    <div class="flex items-start justify-between gap-3">
                      <div class="flex items-center gap-2">
                        <Icon name="bell" size="sm" class="text-amber-600 dark:text-amber-400" />
                        <div>
                          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">PushPlus</h4>
                          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Token 保存后仅脱敏展示。</p>
                        </div>
                      </div>
                      <input v-model="pushChannelEnabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />
                    </div>

                    <label class="input-label mt-4 block">
                      Token
                      <input v-model.trim="push.token" type="password" autocomplete="off" class="input mt-1 w-full" placeholder="PushPlus Token" />
                    </label>
                    <div class="mt-3 grid grid-cols-2 gap-3">
                      <label class="input-label">
                        模板
                        <select v-model="push.template" class="input mt-1 w-full">
                          <option value="html">html</option>
                          <option value="markdown">markdown</option>
                          <option value="txt">txt</option>
                        </select>
                      </label>
                      <label class="input-label">
                        渠道
                        <input v-model.trim="push.channel" class="input mt-1 w-full" placeholder="wechat" />
                      </label>
                    </div>
                    <label class="input-label mt-3 block">
                      群组编码（可选）
                      <input v-model.trim="push.topic" class="input mt-1 w-full" />
                    </label>
                  </article>
                </div>
                <p v-if="channelValidationError" class="mt-3 text-xs text-red-600 dark:text-red-400">{{ channelValidationError }}</p>
              </section>
            </form>
          </div>

          <footer class="flex flex-none flex-wrap items-center justify-between gap-3 border-t border-gray-200 bg-white px-5 py-4 dark:border-dark-600 dark:bg-dark-900 sm:px-6">
            <p class="hidden text-xs text-gray-500 dark:text-gray-400 sm:block">保存不会关闭面板，可立即确认账号状态。</p>
            <div class="ml-auto flex flex-wrap justify-end gap-2">
              <button type="button" class="btn btn-secondary" @click="emit('close')">关闭</button>
              <button
                type="button"
                class="btn btn-secondary inline-flex items-center gap-1.5"
                :disabled="manualRunDisabled"
                @click="emit('run')"
              >
                <Icon name="play" size="sm" />
                {{ running ? '检查中...' : '立即检查' }}
              </button>
              <button form="oauth-monitor-form" type="submit" class="btn btn-primary" :disabled="saveDisabled">
                {{ saving ? '保存中...' : '保存监控设置' }}
              </button>
            </div>
          </footer>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type {
  OAuthAccountMonitorAccountOverview,
  OAuthAccountMonitorConfig,
  OAuthAccountMonitorOverview,
  OAuthMonitorPushPlusConfig
} from '@/api/admin/accounts'
import type { EmailNotificationConfig } from '@/api/admin/ops'

const props = withDefaults(defineProps<{
  show: boolean
  accountIds: number[]
  overview: OAuthAccountMonitorOverview | null
  loading?: boolean
  saving?: boolean
  running?: boolean
  error?: string
  notice?: string
}>(), {
  loading: false,
  saving: false,
  running: false,
  error: '',
  notice: ''
})

const emit = defineEmits<{
  close: []
  save: [config: OAuthAccountMonitorConfig, push: OAuthMonitorPushPlusConfig, email: EmailNotificationConfig]
  refresh: []
  retry: []
  run: []
}>()

const defaultConfig = (): OAuthAccountMonitorConfig => ({
  enabled: false,
  account_ids: [],
  quota_threshold_percent: 20,
  interval_minutes: 5,
  notify_on_error: true,
  notify_on_quota: true,
  notify_on_recovery: true,
  notify_email: true,
  notify_pushplus: true,
  repeat_every_check: true
})

const defaultPush = (): OAuthMonitorPushPlusConfig => ({
  enabled: false,
  token: '',
  topic: '',
  template: 'html',
  channel: ''
})

const defaultEmail = (): EmailNotificationConfig => ({
  alert: {
    enabled: false,
    recipients: [],
    min_severity: '',
    rate_limit_per_hour: 100,
    batching_window_seconds: 0,
    include_resolved_alerts: true
  },
  report: {
    enabled: false,
    recipients: [],
    daily_summary_enabled: false,
    daily_summary_schedule: '0 9 * * *',
    weekly_summary_enabled: false,
    weekly_summary_schedule: '0 9 * * 1',
    error_digest_enabled: false,
    error_digest_schedule: '0 * * * *',
    error_digest_min_count: 1,
    account_health_enabled: false,
    account_health_schedule: '0 9 * * *',
    account_health_error_rate_threshold: 10
  }
})

const form = reactive<OAuthAccountMonitorConfig>(defaultConfig())
const push = reactive<OAuthMonitorPushPlusConfig>(defaultPush())
const email = reactive<EmailNotificationConfig>(defaultEmail())
const recipientInput = ref('')
const recipientError = ref('')
const drawerRef = ref<HTMLElement | null>(null)

const monitoredIDs = computed(() => new Set(form.account_ids))
const monitoredCount = computed(() => form.account_ids.length)
const selectedSummary = computed(() => {
  if (props.accountIds.length === 0) return '查看与维护全部 OAuth 监控账号'
  return `本次选择 ${props.accountIds.length} 个账号，支持的 OAuth 账号会预先加入名单`
})

const emailChannelEnabled = computed({
  get: () => form.notify_email && email.alert.enabled,
  set: (enabled: boolean) => {
    form.notify_email = enabled
    email.alert.enabled = enabled
  }
})

const pushChannelEnabled = computed({
  get: () => form.notify_pushplus && push.enabled,
  set: (enabled: boolean) => {
    form.notify_pushplus = enabled
    push.enabled = enabled
  }
})

const channelValidationError = computed(() => {
  if (emailChannelEnabled.value && email.alert.recipients.length === 0) {
    return '启用邮件通知后至少需要一个管理员收件人。'
  }
  if (pushChannelEnabled.value && !push.token.trim()) {
    return '启用 PushPlus 后需要填写 Token。'
  }
  return ''
})

const saveDisabled = computed(() => !props.overview || props.loading || props.saving || props.running || Boolean(props.error) || Boolean(channelValidationError.value))
const manualRunDisabled = computed(() => {
  const overview = props.overview
  return props.loading || props.saving || props.running || !overview || !overview.config.enabled || overview.config.account_ids.length === 0
})

watch(
  () => props.overview,
  (overview) => {
    if (!overview) return
    Object.assign(form, defaultConfig(), overview.config)
    form.account_ids = overview.accounts
      .filter((account) => account.eligible && (account.monitored || account.selected))
      .map((account) => account.account_id)

    Object.assign(push, defaultPush(), overview.pushplus)
    const fallbackEmail = defaultEmail()
    Object.assign(email.alert, fallbackEmail.alert, overview.email?.alert ?? {})
    email.alert.recipients = [...(overview.email?.alert?.recipients ?? [])]
    Object.assign(email.report, fallbackEmail.report, overview.email?.report ?? {})
    email.report.recipients = [...(overview.email?.report?.recipients ?? [])]
  },
  { immediate: true }
)

watch(
  () => props.show,
  async (show) => {
    document.body.classList.toggle('modal-open', show)
    if (show) {
      await nextTick()
      drawerRef.value?.focus()
    }
  },
  { immediate: true }
)

function handleEscape(event: KeyboardEvent) {
  if (props.show && event.key === 'Escape') emit('close')
}

onMounted(() => document.addEventListener('keydown', handleEscape))
onUnmounted(() => {
  document.removeEventListener('keydown', handleEscape)
  document.body.classList.remove('modal-open')
})

function isMonitored(accountID: number) {
  return monitoredIDs.value.has(accountID)
}

function toggleAccount(accountID: number, monitored: boolean) {
  const ids = new Set(form.account_ids)
  if (monitored) ids.add(accountID)
  else ids.delete(accountID)
  form.account_ids = [...ids]
}

function currentStatus(account: OAuthAccountMonitorAccountOverview) {
  if (!account.eligible) return 'ineligible'
  if (!isMonitored(account.account_id)) return 'not_monitored'
  if (!form.enabled) return 'paused'
  if (!account.state?.last_checked_at) return 'pending'
  if (account.state.last_condition_key?.includes('quota_low:')) return 'quota_low'
  if (account.state.health_status === 'error' || account.state.last_condition_key) return 'error'
  return 'healthy'
}

function statusMeta(account: OAuthAccountMonitorAccountOverview) {
  const statuses = {
    ineligible: { label: '不支持监控', className: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300' },
    not_monitored: { label: '未监控', className: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300' },
    paused: { label: '已加入 · 全局暂停', className: 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300' },
    pending: { label: '等待首次检查', className: 'bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300' },
    healthy: { label: '监控中', className: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' },
    error: { label: '健康异常', className: 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300' },
    quota_low: { label: '额度低于阈值', className: 'bg-orange-50 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300' }
  } as const
  return statuses[currentStatus(account)]
}

function formatRemaining(value?: number | null) {
  return value == null ? '未知' : `${Math.round(value)}%`
}

function formatCheckedAt(value?: string | null) {
  if (!value) return '尚未检查'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '未知' : date.toLocaleString()
}

function addRecipient() {
  const value = recipientInput.value.trim().toLowerCase()
  recipientError.value = ''
  if (!value) return
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
    recipientError.value = '请输入有效的邮箱地址。'
    return
  }
  if (!email.alert.recipients.includes(value)) email.alert.recipients.push(value)
  recipientInput.value = ''
}

function removeRecipient(recipient: string) {
  email.alert.recipients = email.alert.recipients.filter((value) => value !== recipient)
}

function submit() {
  if (saveDisabled.value) return
  emit(
    'save',
    { ...form, account_ids: [...form.account_ids] },
    { ...push },
    {
      alert: { ...email.alert, recipients: [...email.alert.recipients] },
      report: { ...email.report, recipients: [...email.report.recipients] }
    }
  )
}
</script>

<style scoped>
.oauth-monitor-drawer-enter-active,
.oauth-monitor-drawer-leave-active {
  transition: background-color 180ms ease;
}

.oauth-monitor-drawer-enter-active aside,
.oauth-monitor-drawer-leave-active aside {
  transition: transform 220ms ease;
}

.oauth-monitor-drawer-enter-from,
.oauth-monitor-drawer-leave-to {
  background-color: transparent;
}

.oauth-monitor-drawer-enter-from aside,
.oauth-monitor-drawer-leave-to aside {
  transform: translateX(100%);
}
</style>
