import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { GUIDE_CLIENTS, GUIDE_STORAGE_PREFIX, type GuideClient } from './config'
import type { GuideStepId } from './steps'

/**
 * idle：还没开始（新用户待启动 / 老用户待邀请）；active：进行中；
 * done：完成；skipped：中途跳过；dismissed：老用户在邀请卡片上点了「不用了」。
 */
export type GuideStatus = 'idle' | 'active' | 'done' | 'skipped' | 'dismissed'

interface Persisted {
  status: GuideStatus
  step: GuideStepId
  client: GuideClient | null
  /** 引导里创建的那把 Key，用来在 Key 列表里高亮它的「导入」按钮。 */
  keyId: number | null
  /** 走过的步骤（最近的在最后），给左上角的返回箭头用。 */
  history: GuideStepId[]
}

const HISTORY_LIMIT = 30
const DEFAULT: Persisted = { status: 'idle', step: 'welcome', client: null, keyId: null, history: [] }
const key = (userId: number) => `${GUIDE_STORAGE_PREFIX}:${userId}`

function read(userId: number): Persisted {
  try {
    const raw = localStorage.getItem(key(userId))
    if (!raw) return { ...DEFAULT }
    const parsed = JSON.parse(raw) as Partial<Persisted>
    return {
      status: parsed.status ?? 'idle',
      step: parsed.step ?? 'welcome',
      client: parsed.client && GUIDE_CLIENTS.includes(parsed.client) ? parsed.client : null,
      keyId: typeof parsed.keyId === 'number' ? parsed.keyId : null,
      history: Array.isArray(parsed.history) ? parsed.history.filter((s): s is GuideStepId => typeof s === 'string').slice(-HISTORY_LIMIT) : []
    }
  } catch {
    return { ...DEFAULT }
  }
}

/** 引导进度：按账号存在 localStorage，刷新、关页、去下载后回来都能接着走。 */
export const useGuideStore = defineStore('onboardingGuide', () => {
  const userId = ref<number | null>(null)
  const status = ref<GuideStatus>('idle')
  const step = ref<GuideStepId>('welcome')
  const client = ref<GuideClient | null>(null)
  const keyId = ref<number | null>(null)
  const history = ref<GuideStepId[]>([])
  /** 兑换成功 / Key 创建成功这类页面事件的计数，GuideLayer 据此推进。 */
  const redeemedTick = ref(0)
  const keyCreatedTick = ref(0)

  const active = computed(() => status.value === 'active')

  function persist() {
    if (userId.value == null) return
    try {
      localStorage.setItem(key(userId.value), JSON.stringify({ status: status.value, step: step.value, client: client.value, keyId: keyId.value, history: history.value }))
    } catch {
      /* 存储不可用时本次会话内仍可用 */
    }
  }

  /** 切换账号 / 登录后载入该账号的进度。 */
  function load(id: number | null) {
    userId.value = id
    const saved = id == null ? { ...DEFAULT } : read(id)
    status.value = saved.status
    step.value = saved.step
    client.value = saved.client
    keyId.value = saved.keyId
    history.value = saved.history
  }

  function start() {
    status.value = 'active'
    step.value = 'welcome'
    client.value = null
    keyId.value = null
    history.value = []
    persist()
  }

  function go(next: GuideStepId) {
    if (next !== step.value) history.value = [...history.value, step.value].slice(-HISTORY_LIMIT)
    step.value = next
    persist()
  }

  /** 回到历史里的第 index 步（之后走过的都丢掉），不记入历史。 */
  function backTo(index: number) {
    const target = history.value[index]
    if (target == null) return
    history.value = history.value.slice(0, index)
    step.value = target
    persist()
  }

  function chooseClient(next: GuideClient) {
    client.value = next
    persist()
  }

  function finish() {
    status.value = 'done'
    persist()
  }

  function skip() {
    status.value = 'skipped'
    persist()
  }

  /** 老用户在邀请卡片上选「不用了」：永久不再邀请。 */
  function dismiss() {
    status.value = 'dismissed'
    persist()
  }

  function notifyRedeemed() {
    redeemedTick.value++
  }

  function notifyKeyCreated(id?: number) {
    if (typeof id === 'number') {
      keyId.value = id
      persist()
    }
    keyCreatedTick.value++
  }

  return {
    userId,
    status,
    step,
    client,
    keyId,
    history,
    active,
    redeemedTick,
    keyCreatedTick,
    load,
    start,
    go,
    backTo,
    chooseClient,
    finish,
    skip,
    dismiss,
    notifyRedeemed,
    notifyKeyCreated
  }
})
