<template>
  <div
    v-if="banner && detail"
    data-testid="dashboard-affiliate-banner"
    :data-banner-type="banner.type"
    class="relative overflow-hidden rounded-2xl border p-4 transition-colors sm:px-5 sm:py-4"
    :class="bannerSurfaceClass"
  >
    <!-- Subtle dot glow accent -->
    <div
      aria-hidden="true"
      class="pointer-events-none absolute -right-10 -top-10 h-32 w-32 rounded-full blur-2xl"
      :class="bannerGlowClass"
    ></div>

    <div class="relative flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex min-w-0 items-start gap-3">
        <div
          class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl"
          :class="bannerIconClass"
        >
          <Icon :name="bannerIconName" size="sm" />
        </div>

        <div class="min-w-0 space-y-0.5">
          <!-- P1: Consume reminder -->
          <template v-if="banner.type === 'consume_reminder'">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('affiliate.banner.consumeReminderTitle', { count: banner.count, amount: formatYuan(banner.pendingBonus, localeCode) }) }}
            </p>
            <p class="text-xs text-gray-600 dark:text-dark-300">
              {{ t('affiliate.banner.consumeReminderDesc') }}
            </p>
          </template>

          <!-- P2: Transferable quota -->
          <template v-else-if="banner.type === 'transferable'">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('affiliate.banner.transferableTitle', { amount: formatYuan(banner.amount, localeCode) }) }}
            </p>
            <p class="text-xs text-gray-600 dark:text-dark-300">
              {{
                banner.canWithdraw
                  ? t('affiliate.banner.transferableWithdrawDesc', { threshold: banner.withdrawThreshold })
                  : t('affiliate.banner.transferableDesc')
              }}
            </p>
          </template>

          <!-- P3: Trial active -->
          <template v-else-if="banner.type === 'trial_active'">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('affiliate.banner.trialActiveTitle', { amount: formatYuan(banner.amount, localeCode), days: banner.daysRemaining }) }}
            </p>
            <p class="text-xs text-gray-600 dark:text-dark-300">
              {{ t('affiliate.banner.trialActiveDesc') }}
            </p>
          </template>

          <!-- P4: Ladder sprint -->
          <template v-else-if="banner.type === 'ladder_sprint'">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">
              {{
                t('affiliate.banner.ladderSprintTitle', {
                  remaining: banner.remaining,
                  bonus: formatYuan(banner.tier.delta_bonus, localeCode),
                })
              }}
            </p>
            <p class="text-xs text-gray-600 dark:text-dark-300">
              {{ ladderSprintDescription }}
            </p>
          </template>

          <!-- P5: Default promo -->
          <template v-else>
            <p class="text-sm font-semibold text-gray-900 dark:text-white">
              {{
                formattedRebateRate
                  ? t('affiliate.banner.defaultPromoTitle', { rate: formattedRebateRate })
                  : t('affiliate.banner.defaultPromoTitleNoRate')
              }}
            </p>
            <p class="text-xs text-gray-600 dark:text-dark-300">
              {{ t('affiliate.banner.defaultPromoDesc') }}
            </p>
          </template>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex shrink-0 flex-wrap items-center gap-2 sm:flex-nowrap">
        <!-- P1 actions -->
        <template v-if="banner.type === 'consume_reminder'">
          <button type="button" class="btn btn-secondary btn-sm" data-testid="banner-copy-remind" @click="copyRemindText">
            {{ t('affiliate.banner.copyRemindCopy') }}
          </button>
          <router-link to="/affiliate" class="btn btn-primary btn-sm">
            {{ t('affiliate.banner.viewDetails') }}
          </router-link>
        </template>

        <!-- P2 actions -->
        <template v-else-if="banner.type === 'transferable'">
          <button
            type="button"
            class="btn btn-primary btn-sm"
            data-testid="banner-transfer-btn"
            :disabled="transferring"
            @click="handleTransferQuota"
          >
            {{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.banner.quickTransfer') }}
          </button>
          <router-link to="/affiliate" class="btn btn-secondary btn-sm">
            {{ t('affiliate.banner.viewDetails') }}
          </router-link>
        </template>

        <!-- P3 actions -->
        <template v-else-if="banner.type === 'trial_active'">
          <router-link to="/chat" class="btn btn-primary btn-sm">
            {{ t('affiliate.banner.tryInChat') }}
          </router-link>
          <router-link to="/tutorials" class="btn btn-secondary btn-sm">
            {{ t('affiliate.banner.setupGuides') }}
          </router-link>
        </template>

        <!-- P4 actions -->
        <template v-else-if="banner.type === 'ladder_sprint'">
          <button type="button" class="btn btn-primary btn-sm" data-testid="banner-copy-link" @click="copyInviteLink">
            {{ t('affiliate.copyLink') }}
          </button>
          <router-link to="/affiliate" class="btn btn-secondary btn-sm">
            {{ t('affiliate.banner.viewProgress') }}
          </router-link>
        </template>

        <!-- P5 actions -->
        <template v-else>
          <button type="button" class="btn btn-primary btn-sm" data-testid="banner-copy-link" @click="copyInviteLink">
            {{ t('affiliate.copyLink') }}
          </button>
          <button type="button" class="btn btn-secondary btn-sm" data-testid="banner-open-poster" @click="showPosterDialog = true">
            {{ t('affiliate.banner.generatePoster') }}
          </button>
          <router-link to="/affiliate" class="btn btn-ghost btn-sm">
            {{ t('affiliate.banner.moreDetails') }}
          </router-link>
          <button
            type="button"
            class="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-200/60 hover:text-gray-600 dark:text-dark-400 dark:hover:bg-white/10 dark:hover:text-gray-200"
            data-testid="banner-dismiss-default"
            :aria-label="t('common.close')"
            @click="handleDismissDefault"
          >
            <Icon name="x" size="sm" />
          </button>
        </template>
      </div>
    </div>

    <AffiliatePosterDialog
      v-if="showPosterDialog"
      :show="showPosterDialog"
      :aff-code="detail.aff_code"
      @close="showPosterDialog = false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getActivePinia } from 'pinia'
import Icon from '@/components/icons/Icon.vue'
import AffiliatePosterDialog from '@/features/affiliate/AffiliatePosterDialog.vue'
import {
  buildInviteLink,
  buildRemindConsumeCopy,
  consumeUnseenGrantedBonusDelta,
  copyTextToClipboard,
  dismissDefaultBanner,
  formatRebateRate,
  formatYuan,
  isDefaultBannerDismissed,
  resolveDashboardBanner,
} from '@/features/affiliate/helpers'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { tryUseAffiliateSummaryStore } from '@/stores/affiliateSummary'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { extractApiErrorMessage } from '@/utils/apiError'

const i18n = useI18n()
const { t } = i18n
const authStore = useAuthStore()
const appStore = getActivePinia() ? useAppStore() : null
const affiliateSummaryStore = tryUseAffiliateSummaryStore()

const showPosterDialog = ref(false)
const dismissedDefault = ref(false)

const localeCode = computed(() => {
  const raw = i18n.locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || 'zh')
  }
  return 'zh'
})

const eligible = computed(() =>
  Boolean(
    authStore.isAuthenticated &&
      authStore.user &&
      !authStore.isAdmin &&
      !authStore.isSimpleMode &&
      resolveFeatureFlag(appStore?.cachedPublicSettings, FeatureFlags.affiliate),
  ),
)

const detail = computed(() => (eligible.value ? affiliateSummaryStore?.detail ?? null : null))
const transferring = computed(() => Boolean(affiliateSummaryStore?.transferring))

const banner = computed(() =>
  resolveDashboardBanner(detail.value, { dismissedDefault: dismissedDefault.value }),
)

const formattedRebateRate = computed(() =>
  formatRebateRate(detail.value?.effective_rebate_rate_percent),
)

const ladderSprintDescription = computed(() => {
  if (banner.value?.type !== 'ladder_sprint') return ''
  const parts = [t('affiliate.banner.ladderSprintDesc', { target: banner.value.tier.tier_target })]
  if (banner.value.paidRemaining > 0) {
    parts.push(
      t('affiliate.banner.ladderSprintPaidNeed', {
        paidAmount: formatYuan(banner.value.paidRemaining, localeCode.value),
      }),
    )
  }
  if (banner.value.requiresManualReview) {
    parts.push(t('affiliate.banner.ladderSprintManualReview'))
  }
  return parts.join(' ')
})

const bannerSurfaceClass = computed(() => {
  switch (banner.value?.type) {
    case 'consume_reminder':
      return 'border-amber-200/80 bg-amber-50/70 dark:border-amber-500/25 dark:bg-amber-950/20'
    case 'transferable':
      return 'border-emerald-200/80 bg-emerald-50/70 dark:border-emerald-500/25 dark:bg-emerald-950/20'
    case 'trial_active':
    case 'ladder_sprint':
      return 'border-rose-200/70 bg-white dark:border-cyan-400/25 dark:bg-[#141417]'
    default:
      return 'border-gray-200/80 bg-white dark:border-white/[0.08] dark:bg-[#141417]'
  }
})

const bannerGlowClass = computed(() => {
  switch (banner.value?.type) {
    case 'consume_reminder':
      return 'bg-amber-400/15 dark:bg-amber-400/15'
    case 'transferable':
      return 'bg-emerald-400/15 dark:bg-emerald-400/15'
    default:
      return 'bg-rose-400/15 dark:bg-cyan-400/15'
  }
})

const bannerIconClass = computed(() => {
  switch (banner.value?.type) {
    case 'consume_reminder':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
    case 'transferable':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
    default:
      return 'bg-rose-50 text-rose-600 dark:bg-cyan-400/15 dark:text-cyan-300'
  }
})

const bannerIconName = computed(() => {
  switch (banner.value?.type) {
    case 'consume_reminder':
      return 'sparkles' as const
    case 'transferable':
      return 'dollar' as const
    case 'trial_active':
      return 'gift' as const
    case 'ladder_sprint':
      return 'trophy' as const
    default:
      return 'users' as const
  }
})

async function copyRemindText(): Promise<void> {
  const ok = await copyTextToClipboard(buildRemindConsumeCopy(undefined, localeCode.value))
  if (ok) {
    appStore?.showSuccess(t('affiliate.banner.remindCopied'))
  } else {
    appStore?.showError(t('common.copyFailed'))
  }
}

async function copyInviteLink(): Promise<void> {
  if (!detail.value?.aff_code) return
  const ok = await copyTextToClipboard(buildInviteLink(detail.value.aff_code))
  if (ok) {
    appStore?.showSuccess(t('affiliate.linkCopied'))
  } else {
    appStore?.showError(t('common.copyFailed'))
  }
}

async function handleTransferQuota(): Promise<void> {
  if (!affiliateSummaryStore || transferring.value) return
  try {
    const resp = await affiliateSummaryStore.transferQuota()
    appStore?.showSuccess(
      t('affiliate.transfer.success', { amount: formatYuan(resp.transferred_quota, localeCode.value) }),
    )
  } catch (error) {
    appStore?.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  }
}

function handleDismissDefault(): void {
  const userId = authStore.user?.id
  if (userId != null) {
    dismissDefaultBanner(userId)
  }
  dismissedDefault.value = true
}

function syncDismissAndBonus(): void {
  const userId = authStore.user?.id
  if (userId == null) return
  dismissedDefault.value = isDefaultBannerDismissed(userId)
  if (detail.value) {
    const delta = consumeUnseenGrantedBonusDelta(userId, detail.value)
    if (delta > 0) {
      appStore?.showSuccess(
        t('affiliate.banner.bonusArrivedToast', { amount: formatYuan(delta, localeCode.value) }),
      )
    }
  }
}

watch(
  [eligible, () => authStore.user?.id],
  ([isEligible, userId]) => {
    if (userId != null) {
      dismissedDefault.value = isDefaultBannerDismissed(userId)
    }
    if (isEligible) {
      void affiliateSummaryStore?.ensureLoaded()
    }
  },
  { immediate: true },
)

watch(detail, () => {
  syncDismissAndBonus()
})

onMounted(() => {
  syncDismissAndBonus()
  if (eligible.value) {
    void affiliateSummaryStore?.ensureLoaded()
  }
})
</script>
