<template>
  <AppLayout>
    <div class="space-y-6">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('activities.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('activities.description') }}</p>
      </div>

      <div v-if="loading" class="flex justify-center py-16">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <button
        v-else-if="campaign"
        type="button"
        class="group w-full overflow-hidden rounded-2xl border border-gray-200 bg-white text-left shadow-sm transition hover:-translate-y-0.5 hover:border-primary-300 hover:shadow-lg dark:border-dark-700 dark:bg-dark-800 dark:hover:border-primary-700"
        @click="router.push('/activities/launch-rebate')"
      >
        <div class="grid min-h-[250px] lg:grid-cols-[1.2fr_0.8fr]">
          <div class="relative overflow-hidden bg-gradient-to-br from-primary-600 via-blue-600 to-cyan-500 p-7 text-white sm:p-10">
            <div class="absolute -right-20 -top-24 h-64 w-64 rounded-full border border-white/20"></div>
            <div class="absolute -bottom-32 right-16 h-72 w-72 rounded-full bg-white/10 blur-2xl"></div>
            <div class="relative flex h-full flex-col justify-between gap-10">
              <div class="flex items-center justify-between gap-4">
                <span class="rounded-full bg-white/15 px-3 py-1 text-sm font-medium backdrop-blur">
                  {{ statusLabel }}
                </span>
                <Icon name="gift" size="lg" class="text-white/90" />
              </div>
              <div>
                <p class="text-sm font-medium text-white/75">{{ t('activities.launchRebate.bonusLabel') }}</p>
                <p class="mt-2 text-5xl font-semibold tracking-tight">{{ campaign.bonus_rate_percent }}%</p>
                <h2 class="mt-6 text-2xl font-semibold">{{ campaign.name }}</h2>
                <p class="mt-2 max-w-xl text-sm leading-6 text-white/80">
                  {{ t('activities.launchRebate.description') }}
                </p>
              </div>
            </div>
          </div>

          <div class="flex flex-col justify-between gap-8 p-7 sm:p-10">
            <div>
              <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('activities.until') }}</p>
              <p class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ formattedEndAt }}</p>
              <div class="mt-6 grid grid-cols-2 gap-3">
                <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
                  <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('activities.launchRebate.qualifiedUsers') }}</p>
                  <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ campaign.user_stats.qualified_count }}</p>
                </div>
                <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
                  <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('activities.launchRebate.rank') }}</p>
                  <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                    {{ campaign.user_stats.rank ? `#${campaign.user_stats.rank}` : '-' }}
                  </p>
                </div>
              </div>
            </div>
            <div class="flex items-center justify-between text-sm font-medium text-primary-600 dark:text-primary-400">
              <span>{{ t('activities.viewDetails') }}</span>
              <Icon name="arrowRight" size="sm" class="transition-transform group-hover:translate-x-1" />
            </div>
          </div>
        </div>
      </button>

      <div v-else class="card p-12 text-center text-sm text-gray-500 dark:text-dark-400">
        {{ t('activities.empty') }}
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { LaunchCampaignDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const loading = ref(true)
const campaign = ref<LaunchCampaignDetail | null>(null)

const formattedEndAt = computed(() => campaign.value ? formatDateTime(campaign.value.ends_at) : '-')
const statusLabel = computed(() => {
  if (!campaign.value) return ''
  if (campaign.value.ended) return t('activities.status.ended')
  if (campaign.value.active) return t('activities.status.active')
  return t('activities.status.upcoming')
})

onMounted(async () => {
  try {
    campaign.value = await userAPI.getLaunchCampaign()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('activities.loadFailed')))
  } finally {
    loading.value = false
  }
})
</script>
