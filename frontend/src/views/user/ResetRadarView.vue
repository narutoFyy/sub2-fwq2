<template>
  <AppLayout>
    <div class="w-full min-w-0 space-y-6 pb-8">
      <header class="page-header rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700 sm:p-6">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.16em] text-primary-600 dark:text-primary-400">OpenAI</p>
            <h1 class="page-title mt-1 text-2xl font-black text-gray-900 dark:text-white">{{ t('modelRadar.title') }}</h1>
            <p class="page-description mt-1.5 text-sm text-gray-500 dark:text-gray-400">{{ t('modelRadar.subtitle') }}</p>
          </div>
          <button type="button" class="btn-secondary inline-flex items-center gap-2" :disabled="loading" @click="load">
            <Icon name="refresh" size="sm" />{{ t('modelRadar.refresh') }}
          </button>
        </div>
      </header>

      <section>
        <div class="mb-3 flex items-end justify-between gap-3">
          <div>
            <h2 class="text-lg font-bold text-gray-900 dark:text-white">{{ t('modelRadar.groupTests') }}</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.interval', { minutes: overview?.interval_minutes ?? 30 }) }}</p>
          </div>
        </div>

        <div v-if="loading" class="flex justify-center py-12"><LoadingSpinner /></div>
        <div v-else-if="error" class="rounded-2xl border border-red-200 bg-red-50 p-5 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300">{{ t('modelRadar.loadFailed') }}</div>
        <div v-else-if="!overview?.groups.length" class="rounded-2xl border border-dashed border-gray-300 p-10 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">{{ t('modelRadar.empty') }}</div>
        <div v-else class="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <article v-for="group in overview.groups" :key="group.group_id" class="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0">
                <h3 class="truncate font-bold text-gray-900 dark:text-white">{{ group.group_name }}</h3>
                <p class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">{{ group.model_id }} · {{ group.reasoning_effort }}</p>
              </div>
              <span class="rounded-full px-2.5 py-1 text-xs font-semibold" :class="group.enabled ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'">{{ group.enabled ? t('common.enabled') : t('common.disabled') }}</span>
            </div>

            <div class="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2">
              <RadarTestResult :label="t('modelRadar.logic')" :result="group.logic" />
              <RadarTestResult :label="t('modelRadar.drawing')" :result="group.drawing" />
            </div>

            <div class="mt-5 border-t border-gray-100 pt-4 dark:border-dark-700">
              <div class="mb-2 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400"><span>{{ t('modelRadar.lastRun', { time: formatTime(group.last_run_at) }) }}</span><span>{{ group.timeline.length }} checks</span></div>
              <div class="flex h-7 items-end gap-1" :aria-label="t('modelRadar.groupTests')">
                <span v-for="slot in timeline(group.timeline)" :key="slot.key" class="min-w-0 flex-1 rounded-sm" :class="slotClass(slot.status)" :title="slot.title"></span>
              </div>
            </div>
          </article>
        </div>
      </section>

      <section class="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <article class="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
          <h2 class="font-bold text-gray-900 dark:text-white">{{ t('modelRadar.iq') }}</h2>
          <p class="mt-4 text-2xl font-black text-gray-400 dark:text-gray-500">{{ overview?.iq_status ?? t('modelRadar.notDeveloped') }}</p>
        </article>
        <article class="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
          <h2 class="font-bold text-gray-900 dark:text-white">{{ t('modelRadar.recommendation') }}</h2>
          <p class="mt-4 text-2xl font-black text-gray-400 dark:text-gray-500">{{ overview?.recommend_status ?? t('modelRadar.notDeveloped') }}</p>
        </article>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { modelRadarAPI, type ModelRadarOverview, type ModelRadarResult } from '@/api/modelRadar'

const { t } = useI18n()
const overview = ref<ModelRadarOverview | null>(null)
const loading = ref(false)
const error = ref(false)

const load = async () => {
  loading.value = true
  error.value = false
  try { overview.value = await modelRadarAPI.getOverview() } catch { error.value = true } finally { loading.value = false }
}

const formatTime = (value?: string) => value ? new Date(value).toLocaleString() : t('modelRadar.noData')
const resultLabel = (result?: ModelRadarResult) => {
  if (!result) return t('modelRadar.noData')
  return result.status === 'passed' ? t('modelRadar.passed') : result.status === 'failed' ? t('modelRadar.failed') : result.status === 'request_failed' ? t('modelRadar.requestFailed') : t('modelRadar.pendingReview')
}
const slotClass = (status: string) => status === 'passed' ? 'bg-emerald-500' : status === 'failed' ? 'bg-red-500' : status === 'pending_review' ? 'bg-amber-400' : status === 'request_failed' ? 'bg-gray-400' : 'bg-gray-200 dark:bg-dark-600'
const timeline = (items: ModelRadarResult[]) => Array.from({ length: 16 }, (_, index) => { const item = items[index]; return { key: item?.id ?? `empty-${index}`, status: item?.status ?? 'empty', title: item ? `${resultLabel(item)} · ${formatTime(item.detected_at)}` : t('modelRadar.noData') } })
const RadarTestResult = defineComponent({
  props: { label: { type: String, required: true }, result: { type: Object as () => ModelRadarResult | undefined, default: undefined } },
  setup(props) {
    const { t: childT } = useI18n()
    return () => h('div', { class: 'rounded-xl border border-gray-100 p-3 dark:border-dark-700' }, [
      h('p', { class: 'text-xs text-gray-500 dark:text-gray-400' }, props.label),
      h('div', { class: 'mt-2 flex items-center justify-between gap-2' }, [
        h('span', { class: 'text-sm font-semibold text-gray-900 dark:text-white' }, props.result ? (props.result.status === 'passed' ? childT('modelRadar.passed') : props.result.status === 'failed' ? childT('modelRadar.failed') : props.result.status === 'pending_review' ? childT('modelRadar.pendingReview') : childT('modelRadar.requestFailed')) : childT('modelRadar.noData')),
        props.result ? h('span', { class: 'text-xs text-gray-500 dark:text-gray-400' }, t('modelRadar.latency', { ms: props.result.latency_ms })) : null
      ])
    ])
  }
})
onMounted(load)
</script>
