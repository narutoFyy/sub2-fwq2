<template>
  <AppLayout>
    <div class="space-y-6">
      <button type="button" class="inline-flex items-center gap-2 text-sm font-medium text-gray-600 hover:text-primary-600 dark:text-dark-300 dark:hover:text-primary-400" @click="router.push('/activities')">
        <Icon name="arrowLeft" size="sm" />
        {{ t('activities.launchRebate.back') }}
      </button>

      <div v-if="loading" class="flex justify-center py-16">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else-if="campaign">
        <section class="relative overflow-hidden rounded-2xl bg-gradient-to-r from-primary-700 via-blue-600 to-cyan-500 p-7 text-white shadow-lg sm:p-10">
          <div class="absolute -right-16 -top-24 h-72 w-72 rounded-full border border-white/20"></div>
          <div class="absolute -bottom-36 right-1/3 h-72 w-72 rounded-full bg-white/10 blur-2xl"></div>
          <div class="relative grid gap-8 lg:grid-cols-[1fr_auto] lg:items-end">
            <div>
              <div class="flex flex-wrap items-center gap-3">
                <span class="rounded-full bg-white/15 px-3 py-1 text-sm font-medium backdrop-blur">{{ statusLabel }}</span>
                <span class="text-sm text-white/75">{{ remainingText }}</span>
              </div>
              <h1 class="mt-6 text-3xl font-semibold tracking-tight sm:text-4xl">{{ campaign.name }}</h1>
              <p class="mt-3 max-w-2xl text-sm leading-6 text-white/80">{{ t('activities.launchRebate.description') }}</p>
            </div>
            <div class="rounded-2xl border border-white/20 bg-white/10 px-8 py-5 text-center backdrop-blur">
              <p class="text-sm text-white/75">{{ t('activities.launchRebate.bonusLabel') }}</p>
              <p class="mt-1 text-5xl font-semibold">{{ campaign.bonus_rate_percent }}%</p>
            </div>
          </div>
        </section>

        <section>
          <h2 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('activities.launchRebate.myStats') }}</h2>
          <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <div class="card p-5">
              <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('activities.launchRebate.rank') }}</p>
              <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">{{ rankText }}</p>
            </div>
            <div class="card p-5">
              <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('activities.launchRebate.qualifiedUsers') }}</p>
              <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ campaign.user_stats.qualified_count }}</p>
            </div>
            <div class="card p-5">
              <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('activities.launchRebate.qualifyingAmount') }}</p>
              <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatCurrency(campaign.user_stats.qualifying_amount) }}</p>
            </div>
            <div class="card p-5">
              <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('activities.launchRebate.bonusAmount') }}</p>
              <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">{{ formatCurrency(campaign.user_stats.bonus_amount) }}</p>
            </div>
          </div>
        </section>

        <section class="grid gap-6 xl:grid-cols-[0.85fr_1.15fr]">
          <div class="card p-6">
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400">
                <Icon name="users" />
              </div>
              <div>
                <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('activities.launchRebate.inviteAction') }}</h2>
                <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('activities.launchRebate.myCode') }}</p>
              </div>
            </div>
            <button type="button" class="mt-5 flex w-full items-center justify-between rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 text-left dark:border-dark-700 dark:bg-dark-900" @click="copyCode">
              <code class="font-semibold text-gray-900 dark:text-white">{{ campaign.affiliate_code }}</code>
              <Icon name="copy" size="sm" class="text-gray-500" />
            </button>
            <button type="button" class="btn btn-primary mt-3 w-full justify-center" @click="copyInviteLink">
              <Icon name="link" size="sm" />
              {{ t('activities.launchRebate.copyLink') }}
            </button>
          </div>

          <div class="card p-6">
            <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('activities.launchRebate.rulesTitle') }}</h2>
            <ol class="mt-4 space-y-3 text-sm leading-6 text-gray-600 dark:text-dark-300">
              <li v-for="(rule, index) in rules" :key="rule" class="flex gap-3">
                <span class="flex h-6 w-6 flex-none items-center justify-center rounded-full bg-primary-50 text-xs font-semibold text-primary-600 dark:bg-primary-900/30 dark:text-primary-400">{{ index + 1 }}</span>
                <span>{{ rule }}</span>
              </li>
            </ol>
          </div>
        </section>

        <section class="card overflow-hidden">
          <div class="border-b border-gray-100 px-6 py-5 dark:border-dark-700">
            <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('activities.launchRebate.leaderboard') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('activities.launchRebate.leaderboardHint') }}</p>
          </div>
          <div v-if="campaign.leaderboard.length === 0" class="px-6 py-14 text-center text-sm text-gray-500 dark:text-dark-400">
            {{ t('activities.launchRebate.leaderboardEmpty') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="w-full min-w-[720px] text-left text-sm">
              <thead class="bg-gray-50 text-gray-500 dark:bg-dark-900 dark:text-dark-400">
                <tr>
                  <th class="px-6 py-3 font-medium">{{ t('activities.launchRebate.columns.rank') }}</th>
                  <th class="px-6 py-3 font-medium">{{ t('activities.launchRebate.columns.user') }}</th>
                  <th class="px-6 py-3 text-right font-medium">{{ t('activities.launchRebate.columns.qualifiedUsers') }}</th>
                  <th class="px-6 py-3 text-right font-medium">{{ t('activities.launchRebate.columns.qualifyingAmount') }}</th>
                  <th class="px-6 py-3 text-right font-medium">{{ t('activities.launchRebate.columns.bonusAmount') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="entry in campaign.leaderboard" :key="`${entry.rank}-${entry.masked_email}`" class="border-t border-gray-100 dark:border-dark-700" :class="entry.is_current_user ? 'bg-primary-50/60 dark:bg-primary-900/10' : ''">
                  <td class="px-6 py-4 font-semibold" :class="entry.rank <= 3 ? 'text-primary-600 dark:text-primary-400' : 'text-gray-700 dark:text-dark-200'">#{{ entry.rank }}</td>
                  <td class="px-6 py-4 font-medium text-gray-900 dark:text-white">{{ entry.masked_email }}</td>
                  <td class="px-6 py-4 text-right text-gray-700 dark:text-dark-200">{{ entry.qualified_count }}</td>
                  <td class="px-6 py-4 text-right font-medium text-gray-900 dark:text-white">{{ formatCurrency(entry.qualifying_amount) }}</td>
                  <td class="px-6 py-4 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(entry.bonus_amount) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { LaunchCampaignDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateTime } from '@/utils/format'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const loading = ref(true)
const campaign = ref<LaunchCampaignDetail | null>(null)
const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | undefined

const statusLabel = computed(() => {
  if (!campaign.value) return ''
  if (campaign.value.ended) return t('activities.status.ended')
  if (campaign.value.active) return t('activities.status.active')
  return t('activities.status.upcoming')
})

const remainingText = computed(() => {
  if (!campaign.value) return ''
  const end = new Date(campaign.value.ends_at).getTime()
  const remaining = Math.max(0, end - now.value)
  if (remaining === 0) return `${t('activities.launchRebate.deadlineLabel')}: ${formatDateTime(campaign.value.ends_at)}`
  const days = Math.floor(remaining / 86400000)
  const hours = Math.floor((remaining % 86400000) / 3600000)
  return `${days} 天 ${hours} 小时后结束`
})

const rankText = computed(() => campaign.value?.user_stats.rank ? `#${campaign.value.user_stats.rank}` : t('activities.launchRebate.unranked'))
const inviteLink = computed(() => {
  if (!campaign.value) return ''
  const path = `/register?aff=${encodeURIComponent(campaign.value.affiliate_code)}`
  return typeof window === 'undefined' ? path : `${window.location.origin}${path}`
})
const rules = computed(() => [
  t('activities.launchRebate.rules.period'),
  t('activities.launchRebate.rules.eligibility'),
  t('activities.launchRebate.rules.sources'),
  t('activities.launchRebate.rules.stacking'),
  t('activities.launchRebate.rules.ranking'),
  t('activities.launchRebate.rules.prize'),
  t('activities.launchRebate.rules.review')
])

async function copyCode(): Promise<void> {
  if (!campaign.value?.affiliate_code) return
  await copyToClipboard(campaign.value.affiliate_code, t('activities.launchRebate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('activities.launchRebate.linkCopied'))
}

onMounted(async () => {
  timer = setInterval(() => { now.value = Date.now() }, 60000)
  try {
    campaign.value = await userAPI.getLaunchCampaign()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('activities.loadFailed')))
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>
