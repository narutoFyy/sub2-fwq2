import { ref } from 'vue'
import { defineStore, getActivePinia } from 'pinia'
import userAPI from '@/api/user'
import type { AffiliateTransferResponse, UserAffiliateDetail } from '@/types'
import { useAuthStore } from './auth'
import { useAppStore } from './app'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'

const CACHE_TTL_MS = 5 * 60 * 1000

export interface AffiliateSyncGuard {
  epoch: number
  seq: number
  mutation: number
}

export const useAffiliateSummaryStore = defineStore('affiliateSummary', () => {
  const detail = ref<UserAffiliateDetail | null>(null)
  const loading = ref(false)
  const transferring = ref(false)
  const lastFetchedAt = ref(0)
  const lastUserId = ref<number | null>(null)
  const mutationCompletedTick = ref(0)
  const lastMutationTransferredQuota = ref(0)

  let sessionEpoch = 0
  let latestSeq = 0
  let mutationVersion = 0
  let transferSeq = 0
  let detailWriteSeq = 0
  let inFlightPromise: Promise<UserAffiliateDetail | null> | null = null
  let inFlightUserId: number | null = null
  let inFlightEpoch = -1

  function isEligible(): boolean {
    const authStore = useAuthStore()
    const appStore = useAppStore()
    return Boolean(
      authStore.isAuthenticated &&
        authStore.user &&
        !authStore.isAdmin &&
        !authStore.isSimpleMode &&
        resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.affiliate),
    )
  }

  function clearInFlight(): void {
    inFlightPromise = null
    inFlightUserId = null
    inFlightEpoch = -1
    loading.value = false
  }

  function reset(): void {
    sessionEpoch += 1
    latestSeq += 1
    mutationVersion += 1
    transferSeq += 1
    detail.value = null
    transferring.value = false
    lastFetchedAt.value = 0
    lastUserId.value = null
    lastMutationTransferredQuota.value = 0
    clearInFlight()
  }

  function recordMutation(): void {
    mutationVersion += 1
    latestSeq += 1
    clearInFlight()
  }

  function captureSyncGuard(): AffiliateSyncGuard {
    return {
      epoch: sessionEpoch,
      seq: latestSeq,
      mutation: mutationVersion,
    }
  }

  function hasGuardMutationChanged(guard?: AffiliateSyncGuard | null): boolean {
    if (!guard) return false
    return guard.mutation !== mutationVersion || guard.epoch !== sessionEpoch
  }

  function hasGuardStale(guard?: AffiliateSyncGuard | null): boolean {
    if (!guard) return false
    return (
      guard.epoch !== sessionEpoch ||
      guard.mutation !== mutationVersion ||
      guard.seq !== latestSeq
    )
  }

  function syncFromDetail(nextDetail: UserAffiliateDetail, guard?: AffiliateSyncGuard): boolean {
    if (!isEligible()) {
      return false
    }
    const authStore = useAuthStore()
    const currentUserId = authStore.user?.id ?? null
    if (currentUserId === null || nextDetail.user_id !== currentUserId) {
      return false
    }
    if (
      guard &&
      (guard.epoch !== sessionEpoch ||
        guard.mutation !== mutationVersion ||
        guard.seq !== latestSeq)
    ) {
      return false
    }
    if (lastUserId.value !== null && lastUserId.value !== currentUserId) {
      sessionEpoch += 1
    }
    latestSeq += 1
    clearInFlight()
    detail.value = nextDetail
    detailWriteSeq += 1
    lastUserId.value = currentUserId
    lastFetchedAt.value = Date.now()
    return true
  }

  async function ensureLoaded(force = false): Promise<UserAffiliateDetail | null> {
    if (!isEligible()) {
      return null
    }
    const authStore = useAuthStore()
    const currentUserId = authStore.user?.id ?? null
    if (currentUserId === null) {
      return null
    }

    if (lastUserId.value !== null && lastUserId.value !== currentUserId) {
      reset()
    }

    const now = Date.now()
    if (
      !force &&
      detail.value &&
      lastUserId.value === currentUserId &&
      now - lastFetchedAt.value < CACHE_TTL_MS
    ) {
      return detail.value
    }

    if (
      !force &&
      inFlightPromise &&
      inFlightUserId === currentUserId &&
      inFlightEpoch === sessionEpoch
    ) {
      return inFlightPromise
    }

    const reqEpoch = sessionEpoch
    const reqMutation = mutationVersion
    const reqSeq = ++latestSeq
    const reqUserId = currentUserId
    loading.value = true

    const promise = userAPI
      .getAffiliateDetail({ page: 1, page_size: 1 })
      .then((result) => {
        const liveAuth = useAuthStore()
        if (
          reqEpoch === sessionEpoch &&
          reqMutation === mutationVersion &&
          reqSeq === latestSeq &&
          liveAuth.isAuthenticated &&
          liveAuth.user?.id === reqUserId &&
          (!result?.user_id || result.user_id === reqUserId)
        ) {
          detail.value = result
          detailWriteSeq += 1
          lastUserId.value = reqUserId
          lastFetchedAt.value = Date.now()
          return result
        }
        return null
      })
      .catch(() => null)
      .finally(() => {
        if (inFlightPromise === promise) {
          clearInFlight()
        }
      })

    inFlightPromise = promise
    inFlightUserId = reqUserId
    inFlightEpoch = reqEpoch
    return promise
  }

  async function transferQuota(): Promise<AffiliateTransferResponse> {
    const authStore = useAuthStore()
    const reqUserId = authStore.user?.id ?? null
    const reqEpoch = sessionEpoch
    const reqTransferSeq = ++transferSeq
    const preTransferWriteSeq = detailWriteSeq
    mutationVersion += 1
    transferring.value = true
    try {
      const resp = await userAPI.transferAffiliateQuota()
      const liveAuth = useAuthStore()
      if (
        reqEpoch === sessionEpoch &&
        liveAuth.isAuthenticated &&
        reqUserId !== null &&
        liveAuth.user?.id === reqUserId
      ) {
        mutationVersion += 1
        const postMutationWriteSeq = detailWriteSeq
        const [, refreshedDetail] = await Promise.all([
          Promise.resolve(liveAuth.refreshUser?.()).catch(() => undefined),
          ensureLoaded(true),
        ])
        const liveAuthAfter = useAuthStore()
        if (
          reqEpoch === sessionEpoch &&
          liveAuthAfter.isAuthenticated &&
          liveAuthAfter.user?.id === reqUserId
        ) {
          const transferred = Math.max(0, resp.transferred_quota ?? 0)
          if (
            !refreshedDetail &&
            detailWriteSeq === postMutationWriteSeq &&
            detail.value &&
            detail.value.user_id === reqUserId
          ) {
            if (detailWriteSeq === preTransferWriteSeq) {
              detail.value = {
                ...detail.value,
                aff_quota: Math.max(0, Number((detail.value.aff_quota - transferred).toFixed(8))),
              }
            } else {
              detail.value = null
            }
            lastFetchedAt.value = 0
          }
          lastMutationTransferredQuota.value = transferred
          mutationCompletedTick.value += 1
        }
      }
      return resp
    } finally {
      if (reqTransferSeq === transferSeq && reqEpoch === sessionEpoch) {
        transferring.value = false
      }
    }
  }

  return {
    detail,
    loading,
    transferring,
    lastFetchedAt,
    lastUserId,
    mutationCompletedTick,
    lastMutationTransferredQuota,
    ensureLoaded,
    syncFromDetail,
    captureSyncGuard,
    hasGuardMutationChanged,
    hasGuardStale,
    recordMutation,
    transferQuota,
    reset,
  }
})

export function tryUseAffiliateSummaryStore() {
  return getActivePinia() ? useAffiliateSummaryStore() : null
}
