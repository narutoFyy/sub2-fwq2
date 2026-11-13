import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { GUIDE_STORAGE_PREFIX, type GuideClient } from './config'
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
}

const DEFAULT: Persisted = { status: 'idle', step: 'welcome', client: null }
const key = (userId: number) => `${GUIDE_STORAGE_PREFIX}:${userId}`

function read(userId: number): Persisted {
  try {
    const raw = localStorage.getItem(key(userId))
    if (!raw) return { ...DEFAULT }
    const parsed = JSON.parse(raw) as Partial<Persisted>
    return {
      status: parsed.status ?? 'idle',
      step: parsed.step ?? 'welcome',
      client: parsed.client === 'codex' || parsed.client === 'claude-code' ? parsed.client : null
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
  /** 兑换成功 / Key 创建成功这类页面事件的计数，GuideLayer 据此推进。 */
  const redeemedTick = ref(0)
  const keyCreatedTick = ref(0)

  const active = computed(() => status.value === 'active')

  function persist() {
    if (userId.value == null) return
    try {
      localStorage.setItem(key(userId.value), JSON.stringify({ status: status.value, step: step.value, client: client.value }))
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
  }

  function start() {
    status.value = 'active'
    step.value = 'welcome'
    client.value = null
    persist()
  }

  function go(next: GuideStepId) {
    step.value = next
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

  function notifyKeyCreated() {
    keyCreatedTick.value++
  }

  return {
    userId,
    status,
    step,
    client,
    active,
    redeemedTick,
    keyCreatedTick,
    load,
    start,
    go,
    chooseClient,
    finish,
    skip,
    dismiss,
    notifyRedeemed,
    notifyKeyCreated
  }
})
