import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import type { UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useAffiliateSummaryStore } from '@/stores/affiliateSummary'
import AffiliateView from '@/views/user/AffiliateView.vue'

const { getAffiliateDetailMock, transferAffiliateQuotaMock } = vi.hoisted(() => ({
  getAffiliateDetailMock: vi.fn(),
  transferAffiliateQuotaMock: vi.fn(),
}))

vi.mock('@/api/user', () => ({
  default: {
    getAffiliateDetail: getAffiliateDetailMock,
    transferAffiliateQuota: transferAffiliateQuotaMock,
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'zh' },
    }),
  }
})

function makeDetail(userId: number, overrides: Partial<UserAffiliateDetail> = {}): UserAffiliateDetail {
  return {
    user_id: userId,
    aff_code: `CODE-${userId}`,
    aff_count: 1,
    aff_quota: 0,
    aff_frozen_quota: 0,
    aff_history_quota: 0,
    effective_rebate_rate_percent: 5,
    granted_step_bonus_total: 0,
    granted_ladder_bonus_total: 0,
    invitees: [],
    ...overrides,
  }
}

function makeUser(id: number, role: 'user' | 'admin' = 'user') {
  return {
    id,
    email: `user${id}@qq.com`,
    username: `user${id}`,
    role,
    balance: 10,
    concurrency: 2,
    status: 'active',
    allowed_groups: [],
    created_at: '2026-11-01T00:00:00Z',
    updated_at: '2026-11-01T00:00:00Z',
  }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('useAffiliateSummaryStore', () => {
  let pinia: ReturnType<typeof createPinia>

  beforeEach(() => {
    vi.clearAllMocks()
    pinia = createPinia()
    setActivePinia(pinia)
    const appStore = useAppStore()
    appStore.cachedPublicSettings = {
      affiliate_enabled: true,
      frontend_mode: 'standard',
    } as unknown as typeof appStore.cachedPublicSettings
  })

  it('discards an in-flight response if the user logs out or switches accounts before it resolves', async () => {
    const authStore = useAuthStore()
    authStore.token = 'tok-1'
    authStore.user = makeUser(1)
    const store = useAffiliateSummaryStore()

    const firstReq = deferred<UserAffiliateDetail>()
    getAffiliateDetailMock.mockReturnValueOnce(firstReq.promise)

    const p1 = store.ensureLoaded()
    // Simulate logout + login as user 2 before user 1 response returns
    store.reset()
    authStore.user = makeUser(2)

    const secondReq = deferred<UserAffiliateDetail>()
    getAffiliateDetailMock.mockReturnValueOnce(secondReq.promise)
    const p2 = store.ensureLoaded()

    // Stale user 1 response arrives after switch
    firstReq.resolve(makeDetail(1, { aff_quota: 99 }))
    expect(await p1).toBeNull()
    expect(store.detail).toBeNull()

    // User 2 response arrives and populates store cleanly
    secondReq.resolve(makeDetail(2, { aff_quota: 12 }))
    expect(await p2).toMatchObject({ user_id: 2, aff_quota: 12 })
    expect(store.detail?.user_id).toBe(2)
    expect(store.detail?.aff_quota).toBe(12)
  })

  it('rejects syncFromDetail for mismatched users and invalidates older in-flight ensureLoaded calls', async () => {
    const authStore = useAuthStore()
    authStore.token = 'tok-1'
    authStore.user = makeUser(10)
    const store = useAffiliateSummaryStore()

    // Mismatched user payload is ignored
    store.syncFromDetail(makeDetail(999, { aff_quota: 88 }))
    expect(store.detail).toBeNull()

    // Start an earlier ensureLoaded request
    const staleReq = deferred<UserAffiliateDetail>()
    getAffiliateDetailMock.mockReturnValueOnce(staleReq.promise)
    const stalePromise = store.ensureLoaded()

    // Newer full detail arrives from AffiliateView via syncFromDetail
    store.syncFromDetail(makeDetail(10, { granted_step_bonus_total: 8, aff_quota: 20 }))
    expect(store.detail?.granted_step_bonus_total).toBe(8)

    // Older out-of-order response with lower bonus arrives later -> must NOT overwrite
    staleReq.resolve(makeDetail(10, { granted_step_bonus_total: 3, aff_quota: 5 }))
    expect(await stalePromise).toBeNull()
    expect(store.detail?.granted_step_bonus_total).toBe(8)
    expect(store.detail?.aff_quota).toBe(20)
  })

  it('starts a fresh forced request after transferQuota instead of reusing a pre-transfer in-flight promise', async () => {
    const authStore = useAuthStore()
    authStore.token = 'tok-1'
    authStore.user = makeUser(5)
    authStore.refreshUser = vi.fn().mockResolvedValue(authStore.user)
    const store = useAffiliateSummaryStore()

    // Pre-transfer in-flight request still pending
    const preTransferReq = deferred<UserAffiliateDetail>()
    const postTransferReq = deferred<UserAffiliateDetail>()
    getAffiliateDetailMock
      .mockReturnValueOnce(preTransferReq.promise)
      .mockReturnValueOnce(postTransferReq.promise)
    transferAffiliateQuotaMock.mockResolvedValueOnce({ transferred_quota: 25, balance: 35 })

    const prePromise = store.ensureLoaded()
    const transferPromise = store.transferQuota()

    // Wait a tick for transferAffiliateQuota to resolve and trigger ensureLoaded(true)
    await Promise.resolve()
    await Promise.resolve()

    expect(getAffiliateDetailMock).toHaveBeenCalledTimes(2)

    // Resolve post-transfer response first, then stale pre-transfer response
    postTransferReq.resolve(makeDetail(5, { aff_quota: 0 }))
    await transferPromise
    expect(store.detail?.aff_quota).toBe(0)

    preTransferReq.resolve(makeDetail(5, { aff_quota: 25 }))
    await prePromise
    expect(store.detail?.aff_quota).toBe(0)
  })

  it('rejects syncFromDetail with a guard captured during an in-flight transferQuota after transfer completes', async () => {
    const authStore = useAuthStore()
    authStore.token = 'tok-1'
    authStore.user = makeUser(7)
    authStore.refreshUser = vi.fn().mockResolvedValue(authStore.user)
    const store = useAffiliateSummaryStore()

    const transferPost = deferred<{ transferred_quota: number; balance: number }>()
    transferAffiliateQuotaMock.mockReturnValueOnce(transferPost.promise)
    getAffiliateDetailMock.mockResolvedValueOnce(makeDetail(7, { aff_quota: 0 }))

    // Start transferQuota on dashboard
    const transferPromise = store.transferQuota()

    // Navigate to /affiliate while POST is in flight and capture guard before GET
    const staleDetailGuard = store.captureSyncGuard()

    // Dashboard POST finishes and refreshes summary to 0
    transferPost.resolve({ transferred_quota: 25, balance: 35 })
    await transferPromise
    expect(store.detail?.aff_quota).toBe(0)

    // Earlier /affiliate GET resolves with stale 25 -> must be rejected
    expect(store.hasGuardMutationChanged(staleDetailGuard)).toBe(true)
    const accepted = store.syncFromDetail(makeDetail(7, { aff_quota: 25 }), staleDetailGuard)
    expect(accepted).toBe(false)
    expect(store.detail?.aff_quota).toBe(0)
  })

  it('resets loading immediately to false when syncFromDetail supersedes an in-flight ensureLoaded', async () => {
    const authStore = useAuthStore()
    authStore.token = 'tok-1'
    authStore.user = makeUser(8)
    const store = useAffiliateSummaryStore()

    const pendingReq = deferred<UserAffiliateDetail>()
    getAffiliateDetailMock.mockReturnValueOnce(pendingReq.promise)

    const loadPromise = store.ensureLoaded()
    expect(store.loading).toBe(true)

    const accepted = store.syncFromDetail(makeDetail(8, { aff_quota: 15 }))
    expect(accepted).toBe(true)
    expect(store.loading).toBe(false)
    expect(store.detail?.aff_quota).toBe(15)

    // Resolving the superseded request later must not overwrite or break loading
    pendingReq.resolve(makeDetail(8, { aff_quota: 1 }))
    await loadPromise
    expect(store.loading).toBe(false)
    expect(store.detail?.aff_quota).toBe(15)
  })

  it('keeps transfer refresh and converges store + remounted AffiliateView to 0 when original view unmounts mid-transfer', async () => {
    const authStore = useAuthStore()
    authStore.token = 'tok-real'
    authStore.user = makeUser(21)
    const refreshUserSpy = vi.fn().mockResolvedValue(authStore.user)
    authStore.refreshUser = refreshUserSpy
    const store = useAffiliateSummaryStore()

    const mountView = () =>
      mount(AffiliateView, {
        global: {
          plugins: [pinia],
          stubs: {
            AppLayout: { template: '<main><slot /></main>' },
            Icon: true,
            Pagination: true,
            AffiliatePosterDialog: true,
          },
        },
      })

    // 1. Initial mount loads aff_quota = 25
    getAffiliateDetailMock.mockResolvedValueOnce(makeDetail(21, { aff_quota: 25, aff_history_quota: 25 }))
    const view1 = mountView()
    await flushPromises()
    expect(store.detail?.aff_quota).toBe(25)

    // 2. Click transfer on AffiliateView #1 while POST stays pending
    const transferPost = deferred<{ transferred_quota: number; balance: number }>()
    transferAffiliateQuotaMock.mockReturnValueOnce(transferPost.promise)
    const transferBtn = view1.findAll('button').find((b) => b.text().includes('affiliate.transfer.button'))!
    await transferBtn.trigger('click')

    // 3. Unmount AffiliateView #1 and remount AffiliateView #2 whose GET will return stale 25
    view1.unmount()
    const remountStaleGet = deferred<UserAffiliateDetail>()
    getAffiliateDetailMock.mockReturnValueOnce(remountStaleGet.promise)
    const view2 = mountView()

    // 4. Transfer POST completes -> store refreshes user & calls ensureLoaded(true) returning 0
    const postTransferSummaryGet = deferred<UserAffiliateDetail>()
    const remountFreshGet = deferred<UserAffiliateDetail>()
    getAffiliateDetailMock
      .mockReturnValueOnce(postTransferSummaryGet.promise)
      .mockReturnValueOnce(remountFreshGet.promise)

    transferPost.resolve({ transferred_quota: 25, balance: 35 })
    await flushPromises()
    expect(refreshUserSpy).toHaveBeenCalledTimes(1)

    postTransferSummaryGet.resolve(makeDetail(21, { aff_quota: 0, aff_history_quota: 25 }))
    await flushPromises()
    expect(store.detail?.aff_quota).toBe(0)

    // 5. Earlier remount GET returns stale 25 -> rejected, view2 retries and converges to 0
    remountStaleGet.resolve(makeDetail(21, { aff_quota: 25, aff_history_quota: 25 }))
    await flushPromises()
    expect(store.detail?.aff_quota).toBe(0)

    remountFreshGet.resolve(makeDetail(21, { aff_quota: 0, aff_history_quota: 25 }))
    await flushPromises()
    expect(store.detail?.aff_quota).toBe(0)
    expect(view2.text()).toContain('0.00 元')
    expect(view2.text()).not.toContain('差 275.00 元')
    view2.unmount()
  })

  it('rejects stale AffiliateView GET and converges both store and view to 0 when only seq advances via ensureLoaded(true)', async () => {
    const authStore = useAuthStore()
    authStore.token = 'tok-real'
    authStore.user = makeUser(22)
    authStore.refreshUser = vi.fn().mockResolvedValue(authStore.user)
    const store = useAffiliateSummaryStore()

    // 1. AffiliateView mounts and fires GET #1 (will resolve late with stale 25)
    const staleViewGet = deferred<UserAffiliateDetail>()
    const freshSummaryGet = deferred<UserAffiliateDetail>()
    const retryViewGet = deferred<UserAffiliateDetail>()
    getAffiliateDetailMock
      .mockReturnValueOnce(staleViewGet.promise)
      .mockReturnValueOnce(freshSummaryGet.promise)
      .mockReturnValueOnce(retryViewGet.promise)

    const wrapper = mount(AffiliateView, {
      global: {
        plugins: [pinia],
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
          Pagination: true,
          AffiliatePosterDialog: true,
        },
      },
    })

    // 2. While GET #1 is in-flight, ensureLoaded(true) advances seq and updates store to 0
    const summaryPromise = store.ensureLoaded(true)
    freshSummaryGet.resolve(makeDetail(22, { aff_quota: 0, aff_history_quota: 25 }))
    await summaryPromise
    expect(store.detail?.aff_quota).toBe(0)

    // 3. Stale GET #1 resolves with 25 -> rejected (only seq changed), triggers retry
    staleViewGet.resolve(makeDetail(22, { aff_quota: 25, aff_history_quota: 25 }))
    await flushPromises()
    expect(store.detail?.aff_quota).toBe(0)
    expect(getAffiliateDetailMock).toHaveBeenCalledTimes(3)

    // 4. Retry resolves with 0 -> both store and AffiliateView converge to 0
    retryViewGet.resolve(makeDetail(22, { aff_quota: 0, aff_history_quota: 25 }))
    await flushPromises()
    expect(store.detail?.aff_quota).toBe(0)
    expect(wrapper.text()).toContain('0.00 元')
    expect(wrapper.text()).not.toContain('差 275.00 元')
    wrapper.unmount()
  })

  it('disables transfer while in-flight and refreshes remounted AffiliateView to 0 when its early GET resolved with 25 before transfer POST completed', async () => {
    const authStore = useAuthStore()
    authStore.token = 'tok-real'
    authStore.user = makeUser(23)
    const refreshUserSpy = vi.fn().mockResolvedValue(authStore.user)
    authStore.refreshUser = refreshUserSpy
    const store = useAffiliateSummaryStore()

    const mountView = () =>
      mount(AffiliateView, {
        global: {
          plugins: [pinia],
          stubs: {
            AppLayout: { template: '<main><slot /></main>' },
            Icon: true,
            Pagination: true,
            AffiliatePosterDialog: true,
          },
        },
      })

    // 1. AffiliateView #1 loads aff_quota = 25 and clicks transfer
    getAffiliateDetailMock.mockResolvedValueOnce(makeDetail(23, { aff_quota: 25, aff_history_quota: 25 }))
    const view1 = mountView()
    await flushPromises()

    const transferPost = deferred<{ transferred_quota: number; balance: number }>()
    transferAffiliateQuotaMock.mockReturnValueOnce(transferPost.promise)
    const view1TransferBtn = view1
      .findAll('button')
      .find((b) => b.text().includes('affiliate.transfer.button'))!
    await view1TransferBtn.trigger('click')
    expect(store.transferring).toBe(true)

    // 2. Unmount #1 and mount #2 while POST is still in-flight; #2's initial GET returns 25 BEFORE POST finishes
    view1.unmount()
    getAffiliateDetailMock.mockResolvedValueOnce(makeDetail(23, { aff_quota: 25, aff_history_quota: 25 }))
    const view2 = mountView()
    await flushPromises()

    // #2 currently displays 25, but transfer button MUST be disabled because store.transferring is true
    const view2BtnWhilePending = view2
      .findAll('button')
      .find(
        (b) =>
          b.text().includes('affiliate.transfer.transferring') ||
          b.text().includes('affiliate.transfer.button'),
      )!
    expect(view2BtnWhilePending.attributes('disabled')).toBeDefined()
    expect(view2BtnWhilePending.text()).toContain('affiliate.transfer.transferring')

    // 3. POST finishes -> store refreshes user & summary (0), then increments mutationCompletedTick -> #2 auto-refreshes to 0
    getAffiliateDetailMock
      .mockResolvedValueOnce(makeDetail(23, { aff_quota: 0, aff_history_quota: 25 }))
      .mockResolvedValueOnce(makeDetail(23, { aff_quota: 0, aff_history_quota: 25 }))

    transferPost.resolve({ transferred_quota: 25, balance: 35 })
    await flushPromises()

    expect(refreshUserSpy).toHaveBeenCalledTimes(1)
    expect(store.transferring).toBe(false)
    expect(store.detail?.aff_quota).toBe(0)
    expect(store.mutationCompletedTick).toBe(1)
    expect(view2.text()).toContain('0.00 元')
    expect(view2.text()).not.toContain('差 275.00 元')

    const view2BtnAfterSettled = view2
      .findAll('button')
      .find((b) => b.text().includes('affiliate.transfer.button'))!
    expect(view2BtnAfterSettled.attributes('disabled')).toBeDefined()
    view2.unmount()
  })

  it('immediately settles remounted AffiliateView to 0 and keeps transfer disabled while post-mutation GET is still pending', async () => {
    const authStore = useAuthStore()
    authStore.token = 'tok-real'
    authStore.user = makeUser(24)
    authStore.refreshUser = vi.fn().mockResolvedValue(authStore.user)
    const store = useAffiliateSummaryStore()

    const mountView = () =>
      mount(AffiliateView, {
        global: {
          plugins: [pinia],
          stubs: {
            AppLayout: { template: '<main><slot /></main>' },
            Icon: true,
            Pagination: true,
            AffiliatePosterDialog: true,
          },
        },
      })

    getAffiliateDetailMock.mockResolvedValueOnce(makeDetail(24, { aff_quota: 25, aff_history_quota: 25 }))
    const view1 = mountView()
    await flushPromises()

    const transferPost = deferred<{ transferred_quota: number; balance: number }>()
    transferAffiliateQuotaMock.mockReturnValueOnce(transferPost.promise)
    const view1Btn = view1.findAll('button').find((b) => b.text().includes('affiliate.transfer.button'))!
    await view1Btn.trigger('click')
    view1.unmount()

    // Remounted #2 resolves early GET with stale 25 while POST is still pending
    getAffiliateDetailMock.mockResolvedValueOnce(makeDetail(24, { aff_quota: 25, aff_history_quota: 25 }))
    const view2 = mountView()
    await flushPromises()

    const delayedView2Get = deferred<UserAffiliateDetail>()
    getAffiliateDetailMock
      .mockResolvedValueOnce(makeDetail(24, { aff_quota: 0, aff_history_quota: 25 }))
      .mockReturnValueOnce(delayedView2Get.promise)

    transferPost.resolve({ transferred_quota: 25, balance: 35 })
    await flushPromises()

    // Even BEFORE delayedView2Get resolves and after store.transferring resets to false,
    // view2 must immediately display 0.00 元, keep transfer disabled, and never fire a 2nd POST
    expect(store.transferring).toBe(false)
    expect(store.detail?.aff_quota).toBe(0)
    expect(view2.text()).toContain('0.00 元')
    expect(view2.text()).not.toContain('差 275.00 元')

    const btnDuringDelayedGet = view2
      .findAll('button')
      .find(
        (b) =>
          b.text().includes('affiliate.transfer.button') ||
          b.text().includes('affiliate.transfer.transferring'),
      )!
    expect(btnDuringDelayedGet.attributes('disabled')).toBeDefined()
    await btnDuringDelayedGet.trigger('click')
    expect(transferAffiliateQuotaMock).toHaveBeenCalledTimes(1)

    delayedView2Get.resolve(makeDetail(24, { aff_quota: 0, aff_history_quota: 25 }))
    await flushPromises()
    expect(view2.text()).toContain('0.00 元')
    const btnAfterSettled = view2
      .findAll('button')
      .find((b) => b.text().includes('affiliate.transfer.button'))!
    expect(btnAfterSettled.attributes('disabled')).toBeDefined()
    view2.unmount()
  })

  it('keeps remounted AffiliateView settled at 0 and transfer disabled when post-mutation completion GET fails', async () => {
    const authStore = useAuthStore()
    authStore.token = 'tok-real'
    authStore.user = makeUser(25)
    authStore.refreshUser = vi.fn().mockResolvedValue(authStore.user)
    const store = useAffiliateSummaryStore()

    const mountView = () =>
      mount(AffiliateView, {
        global: {
          plugins: [pinia],
          stubs: {
            AppLayout: { template: '<main><slot /></main>' },
            Icon: true,
            Pagination: true,
            AffiliatePosterDialog: true,
          },
        },
      })

    getAffiliateDetailMock.mockResolvedValueOnce(makeDetail(25, { aff_quota: 25, aff_history_quota: 25 }))
    const view1 = mountView()
    await flushPromises()

    const transferPost = deferred<{ transferred_quota: number; balance: number }>()
    transferAffiliateQuotaMock.mockReturnValueOnce(transferPost.promise)
    const view1Btn = view1.findAll('button').find((b) => b.text().includes('affiliate.transfer.button'))!
    await view1Btn.trigger('click')
    view1.unmount()

    getAffiliateDetailMock.mockResolvedValueOnce(makeDetail(25, { aff_quota: 25, aff_history_quota: 25 }))
    const view2 = mountView()
    await flushPromises()

    getAffiliateDetailMock
      .mockResolvedValueOnce(makeDetail(25, { aff_quota: 0, aff_history_quota: 25 }))
      .mockRejectedValueOnce(new Error('network error'))

    transferPost.resolve({ transferred_quota: 25, balance: 35 })
    await flushPromises()

    expect(store.transferring).toBe(false)
    expect(store.detail?.aff_quota).toBe(0)
    expect(view2.text()).toContain('0.00 元')
    expect(view2.text()).not.toContain('差 275.00 元')

    const btnAfterError = view2
      .findAll('button')
      .find((b) => b.text().includes('affiliate.transfer.button'))!
    expect(btnAfterError.attributes('disabled')).toBeDefined()
    await btnAfterError.trigger('click')
    expect(transferAffiliateQuotaMock).toHaveBeenCalledTimes(1)
    view2.unmount()
  })

  it('does not double-deduct aff_quota when ensureLoaded(true) is superseded by a fresher post-transfer syncFromDetail', async () => {
    const authStore = useAuthStore()
    authStore.token = 'tok-real'
    authStore.user = makeUser(26)
    authStore.refreshUser = vi.fn().mockResolvedValue(authStore.user)
    const store = useAffiliateSummaryStore()

    const mountView = () =>
      mount(AffiliateView, {
        global: {
          plugins: [pinia],
          stubs: {
            AppLayout: { template: '<main><slot /></main>' },
            Icon: true,
            Pagination: true,
            AffiliatePosterDialog: true,
          },
        },
      })

    getAffiliateDetailMock.mockResolvedValueOnce(makeDetail(26, { aff_quota: 25, aff_history_quota: 25 }))
    const view1 = mountView()
    await flushPromises()

    const transferPost = deferred<{ transferred_quota: number; balance: number }>()
    transferAffiliateQuotaMock.mockReturnValueOnce(transferPost.promise)
    const view1Btn = view1.findAll('button').find((b) => b.text().includes('affiliate.transfer.button'))!
    await view1Btn.trigger('click')
    view1.unmount()

    getAffiliateDetailMock.mockResolvedValueOnce(makeDetail(26, { aff_quota: 25, aff_history_quota: 25 }))
    const view2 = mountView()
    await flushPromises()

    const staleStoreGet = deferred<UserAffiliateDetail>()
    getAffiliateDetailMock
      .mockReturnValueOnce(staleStoreGet.promise)
      .mockRejectedValueOnce(new Error('network error'))

    // Resolve transfer POST: store kicks off ensureLoaded(true) -> staleStoreGet
    transferPost.resolve({ transferred_quota: 25, balance: 35 })
    await Promise.resolve()

    // While ensureLoaded(true) is in-flight, a fresher post-transfer syncFromDetail writes 5 (new rebate arrived)
    const synced = store.syncFromDetail(
      makeDetail(26, { aff_quota: 5, aff_history_quota: 30 }),
      store.captureSyncGuard(),
    )
    expect(synced).toBe(true)

    // Stale ensureLoaded(true) resolves later and is discarded (returns null) -> must NOT subtract 25 from 5!
    staleStoreGet.resolve(makeDetail(26, { aff_quota: 0, aff_history_quota: 25 }))
    await flushPromises()

    expect(store.detail?.aff_quota).toBe(5)
    expect(view2.text()).toContain('5.00 元')
    expect(view2.text()).toContain('差 295.00 元')
    expect(view2.text()).not.toContain('差 300.00 元')

    const btnAfterRace = view2
      .findAll('button')
      .find((b) => b.text().includes('affiliate.transfer.button'))!
    expect(btnAfterRace.attributes('disabled')).toBeUndefined()
    view2.unmount()
  })
})
