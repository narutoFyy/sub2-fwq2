<template>
  <AppLayout>
    <div class="w-full min-w-0 space-y-6 pb-8">
      <header class="page-header rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700 sm:p-6">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div><p class="text-xs font-semibold uppercase tracking-[0.16em] text-primary-600 dark:text-primary-400">OpenAI</p><h1 class="mt-1 text-2xl font-black text-gray-900 dark:text-white">{{ t('modelRadar.configure') }}</h1><p class="mt-1.5 text-sm text-gray-500 dark:text-gray-400">{{ t('modelRadar.subtitle') }}</p></div>
          <div class="flex flex-wrap gap-2"><button class="btn-secondary inline-flex items-center gap-2" :disabled="loading" @click="load"><Icon name="refresh" size="sm" />{{ t('modelRadar.refresh') }}</button><button class="btn-primary inline-flex items-center gap-2" :disabled="saving || !groups.length" @click="save"><Icon name="check" size="sm" />{{ t('modelRadar.save') }}</button><button class="btn-secondary inline-flex items-center gap-2" :disabled="running || !groups.length" @click="run"><Icon name="play" size="sm" />{{ t('modelRadar.runNow') }}</button></div>
        </div>
      </header>

      <div v-if="error" class="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300">{{ t('modelRadar.loadFailed') }}</div>
      <div v-if="notice" class="rounded-2xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-950/30 dark:text-emerald-300">{{ notice }}</div>
      <div v-if="loading" class="flex justify-center py-12"><LoadingSpinner /></div>
      <section v-else-if="groups.length" class="space-y-4">
        <article v-for="group in groups" :key="group.group_id" class="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-3"><div><h2 class="font-bold text-gray-900 dark:text-white">{{ group.group_name }}</h2><p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ group.group_id }}</p></div><button type="button" class="relative inline-flex h-6 w-11 flex-shrink-0 rounded-full transition" :class="group.enabled ? 'bg-emerald-500' : 'bg-gray-300 dark:bg-dark-600'" :aria-label="t('modelRadar.enabled')" @click="group.enabled = !group.enabled"><span class="absolute top-1 h-4 w-4 rounded-full bg-white transition" :class="group.enabled ? 'left-6' : 'left-1'"></span></button></div>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2"><label class="block"><span class="mb-1.5 block text-xs font-semibold text-gray-600 dark:text-gray-300">{{ t('modelRadar.model') }}</span><input v-model="group.model_id" class="input w-full" maxlength="128" /></label><label class="block"><span class="mb-1.5 block text-xs font-semibold text-gray-600 dark:text-gray-300">{{ t('modelRadar.effort') }}</span><select v-model="group.reasoning_effort" class="input w-full"><option v-for="effort in efforts" :key="effort" :value="effort">{{ effort }}</option></select></label></div>
          <div class="mt-5 grid grid-cols-1 gap-3 lg:grid-cols-2"><div v-for="(result, index) in [group.logic, group.drawing]" :key="result?.id ?? index" class="rounded-xl border border-gray-100 p-3 dark:border-dark-700"><div class="flex items-center justify-between gap-2"><span class="text-xs font-semibold text-gray-600 dark:text-gray-300">{{ result?.test_type === 'drawing' ? t('modelRadar.drawing') : t('modelRadar.logic') }}</span><span class="text-xs text-gray-500 dark:text-gray-400">{{ result ? `${result.latency_ms} ms` : t('modelRadar.noData') }}</span></div><p class="mt-2 text-sm font-semibold" :class="resultClass(result?.status)">{{ statusLabel(result?.status) }}</p><div v-if="result?.test_type === 'drawing' && result.status === 'pending_review'" class="mt-3 flex gap-2"><button class="btn-secondary text-xs" @click="openReview(result)">{{ t('modelRadar.review') }}</button></div></div></div>
        </article>
      </section>
      <div v-else class="rounded-2xl border border-dashed border-gray-300 p-10 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">{{ t('modelRadar.empty') }}</div>

      <section class="grid grid-cols-1 gap-4 lg:grid-cols-2"><article class="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700"><h2 class="font-bold text-gray-900 dark:text-white">{{ t('modelRadar.iq') }}</h2><p class="mt-3 text-xl font-black text-gray-400">{{ overview?.iq_status }}</p></article><article class="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700"><h2 class="font-bold text-gray-900 dark:text-white">{{ t('modelRadar.recommendation') }}</h2><p class="mt-3 text-xl font-black text-gray-400">{{ overview?.recommend_status }}</p></article></section>

      <div v-if="reviewing" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="reviewing = null"><div class="max-h-[85vh] w-full max-w-3xl overflow-auto rounded-2xl bg-white p-5 shadow-xl dark:bg-dark-800"><div class="flex items-center justify-between gap-3"><h2 class="font-bold text-gray-900 dark:text-white">{{ t('modelRadar.review') }}</h2><button class="btn-secondary" @click="reviewing = null">{{ t('common.close') }}</button></div><pre class="mt-4 max-h-[55vh] overflow-auto whitespace-pre-wrap break-words rounded-xl bg-gray-950 p-4 text-xs text-gray-100">{{ reviewing.response_text }}</pre><div class="mt-4 flex justify-end gap-2"><button class="btn-secondary" :disabled="reviewSaving" @click="submitReview('failed')">{{ t('modelRadar.reviewFail') }}</button><button class="btn-primary" :disabled="reviewSaving" @click="submitReview('passed')">{{ t('modelRadar.reviewPass') }}</button></div></div></div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { ModelRadarGroup, ModelRadarOverview, ModelRadarResult } from '@/api/modelRadar'

const { t } = useI18n()
const overview = ref<ModelRadarOverview | null>(null)
const groups = ref<ModelRadarGroup[]>([])
const loading = ref(false); const saving = ref(false); const running = ref(false); const error = ref(false); const notice = ref(''); const reviewing = ref<ModelRadarResult | null>(null); const reviewSaving = ref(false)
const efforts = ['minimal', 'low', 'medium', 'high', 'xhigh', 'max']

const load = async () => { loading.value = true; error.value = false; try { overview.value = await adminAPI.modelRadar.getOverview(); groups.value = overview.value.groups.map(group => ({ ...group })) } catch { error.value = true } finally { loading.value = false } }
const save = async () => { saving.value = true; try { await adminAPI.modelRadar.updateConfigs(groups.value.map(group => ({ group_id: group.group_id, group_name: group.group_name, model_id: group.model_id, reasoning_effort: group.reasoning_effort, enabled: group.enabled }))); notice.value = t('modelRadar.saved'); await load() } catch { error.value = true } finally { saving.value = false } }
const run = async () => { running.value = true; try { await adminAPI.modelRadar.runNow(groups.value.filter(group => group.enabled).map(group => group.group_id)); notice.value = t('modelRadar.running'); setTimeout(load, 500) } catch { error.value = true } finally { running.value = false } }
const openReview = (result: ModelRadarResult) => { reviewing.value = result }
const submitReview = async (status: 'passed' | 'failed') => { if (!reviewing.value) return; reviewSaving.value = true; try { await adminAPI.modelRadar.reviewResult(reviewing.value.id, status); reviewing.value = null; await load() } catch { error.value = true } finally { reviewSaving.value = false } }
const statusLabel = (status?: string) => status === 'passed' ? t('modelRadar.passed') : status === 'failed' ? t('modelRadar.failed') : status === 'pending_review' ? t('modelRadar.pendingReview') : status === 'request_failed' ? t('modelRadar.requestFailed') : t('modelRadar.noData')
const resultClass = (status?: string) => status === 'passed' ? 'text-emerald-600 dark:text-emerald-400' : status === 'failed' ? 'text-red-600 dark:text-red-400' : status === 'pending_review' ? 'text-amber-600 dark:text-amber-400' : 'text-gray-500'
onMounted(load)
</script>
