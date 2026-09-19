<template>
  <AppLayout>
    <div class="w-full min-w-0 space-y-6 pb-8">
      <header class="page-header rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700 sm:p-6">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p class="text-xs font-semibold uppercase text-primary-600 dark:text-primary-400">OpenAI</p>
            <h1 class="mt-1 text-2xl font-black text-gray-900 dark:text-white">{{ t('modelRadar.configure') }}</h1>
            <p class="mt-1.5 text-sm text-gray-500 dark:text-gray-400">{{ t('modelRadar.subtitle') }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button class="btn-secondary inline-flex items-center gap-2" :disabled="loading" @click="load"><Icon name="refresh" size="sm" />{{ t('modelRadar.refresh') }}</button>
            <button class="btn-primary inline-flex items-center gap-2" :disabled="saving || !groups.length" @click="save"><Icon name="check" size="sm" />{{ t('modelRadar.save') }}</button>
            <button class="btn-secondary inline-flex items-center gap-2" :disabled="running || !groups.length" @click="run"><Icon name="play" size="sm" />{{ t('modelRadar.runNow') }}</button>
          </div>
        </div>
      </header>

      <div v-if="error" class="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300">{{ t('modelRadar.loadFailed') }}</div>
      <div v-if="notice" class="rounded-2xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-950/30 dark:text-emerald-300">{{ notice }}</div>
      <div v-if="loading" class="flex justify-center py-12"><LoadingSpinner /></div>
      <section v-else-if="groups.length" class="space-y-4">
        <article v-for="group in groups" :key="group.group_id" class="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div><h2 class="font-bold text-gray-900 dark:text-white">{{ group.group_name }}</h2><p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ group.group_id }}</p></div>
            <button type="button" class="relative inline-flex h-6 w-11 flex-shrink-0 rounded-full transition" :class="group.enabled ? 'bg-emerald-500' : 'bg-gray-300 dark:bg-dark-600'" :aria-label="t('modelRadar.enabled')" @click="group.enabled = !group.enabled"><span class="absolute top-1 h-4 w-4 rounded-full bg-white transition" :class="group.enabled ? 'left-6' : 'left-1'"></span></button>
          </div>

          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <label class="block"><span class="mb-1.5 block text-xs font-semibold text-gray-600 dark:text-gray-300">{{ t('modelRadar.model') }}</span><input v-model="group.model_id" class="input w-full" maxlength="128" /></label>
            <label class="block"><span class="mb-1.5 block text-xs font-semibold text-gray-600 dark:text-gray-300">{{ t('modelRadar.effort') }}</span><select v-model="group.reasoning_effort" class="input w-full"><option v-for="effort in efforts" :key="effort" :value="effort">{{ effort }}</option></select></label>
          </div>

          <div class="mt-5 space-y-5">
            <ModelRadarResultPanel :label="t('modelRadar.logic')" :result="group.logic" :fallback-prompt="logicPrompt" />
            <ModelRadarResultPanel :label="t('modelRadar.drawing')" :result="group.drawing" :fallback-prompt="drawingPrompt" />
          </div>
        </article>
      </section>
      <div v-else class="rounded-2xl border border-dashed border-gray-300 p-10 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">{{ t('modelRadar.empty') }}</div>

      <section class="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <article class="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700"><h2 class="font-bold text-gray-900 dark:text-white">{{ t('modelRadar.iq') }}</h2><p class="mt-3 text-xl font-black text-gray-400">{{ overview?.iq_status }}</p></article>
        <article class="rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700"><h2 class="font-bold text-gray-900 dark:text-white">{{ t('modelRadar.recommendation') }}</h2><p class="mt-3 text-xl font-black text-gray-400">{{ overview?.recommend_status }}</p></article>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelRadarResultPanel from '@/components/model-radar/ModelRadarResultPanel.vue'
import { adminAPI } from '@/api/admin'
import type { ModelRadarGroup, ModelRadarOverview } from '@/api/modelRadar'

const { t } = useI18n()
const overview = ref<ModelRadarOverview | null>(null)
const groups = ref<ModelRadarGroup[]>([])
const loading = ref(false)
const saving = ref(false)
const running = ref(false)
const error = ref(false)
const notice = ref('')
const efforts = ['minimal', 'low', 'medium', 'high', 'xhigh', 'max']
const logicPrompt = 'Solve this logic question. What is the next number in the sequence 1, 3, 6, 10, 15, ? Explain briefly, then finish with exactly "Answer: 21".'
const drawingPrompt = '创建一个HTML，内容是SVG绘制一个鹈鹕骑自行车的2D动画'

const load = async () => {
  loading.value = true
  error.value = false
  try {
    overview.value = await adminAPI.modelRadar.getOverview()
    groups.value = overview.value.groups.map(group => ({ ...group }))
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

const save = async () => {
  saving.value = true
  try {
    await adminAPI.modelRadar.updateConfigs(groups.value.map(group => ({ group_id: group.group_id, group_name: group.group_name, model_id: group.model_id, reasoning_effort: group.reasoning_effort, enabled: group.enabled })))
    notice.value = t('modelRadar.saved')
    await load()
  } catch {
    error.value = true
  } finally {
    saving.value = false
  }
}

const run = async () => {
  running.value = true
  try {
    await adminAPI.modelRadar.runNow(groups.value.filter(group => group.enabled).map(group => group.group_id))
    notice.value = t('modelRadar.running')
    setTimeout(load, 500)
  } catch {
    error.value = true
  } finally {
    running.value = false
  }
}

onMounted(load)
</script>
