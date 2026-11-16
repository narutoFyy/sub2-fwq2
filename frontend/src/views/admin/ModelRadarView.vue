<template>
  <AppLayout>
    <div class="w-full min-w-0 space-y-5 pb-8">
      <header class="flex flex-wrap items-start justify-between gap-4">
        <div class="min-w-0">
          <h1 class="page-title">{{ t('modelRadar.configure') }}</h1>
          <p class="page-description">{{ t('modelRadar.configureSubtitle') }}</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="load"><Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />{{ t('modelRadar.refresh') }}</button>
          <button type="button" class="btn btn-secondary" :disabled="running || !enabledGroupIds.length" @click="run"><Icon name="play" size="sm" />{{ running ? t('modelRadar.runningNow') : t('modelRadar.runNow') }}</button>
          <button type="button" class="btn btn-primary" :disabled="saving || !groups.length" @click="save"><Icon name="check" size="sm" />{{ t('modelRadar.save') }}</button>
        </div>
      </header>

      <RadarHowItWorks :prompt="prompt" :expected="expected" />

      <div v-if="loading && !overview" class="flex justify-center py-12"><LoadingSpinner /></div>
      <div v-else-if="error" class="card p-5 text-sm text-red-700 dark:text-red-300">{{ t('modelRadar.loadFailed') }}</div>
      <div v-else-if="!groups.length" class="card p-10 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('modelRadar.empty') }}</div>
      <template v-else>
        <RadarSummary :groups="groups" />
        <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <RadarGroupCard v-for="group in groups" :key="group.group_id" :group="group" :prompt="prompt">
            <template #config>
              <div class="mt-4 rounded-xl bg-gray-900/[0.03] p-4 dark:bg-white/[0.04]">
                <label class="flex cursor-pointer items-center justify-between gap-3">
                  <span class="text-sm font-medium text-gray-800 dark:text-gray-100">{{ t('modelRadar.enabled') }}</span>
                  <button type="button" role="switch" :aria-checked="group.enabled" class="relative inline-flex h-6 w-11 shrink-0 rounded-full transition" :class="group.enabled ? 'bg-emerald-500' : 'bg-gray-300 dark:bg-white/20'" @click="group.enabled = !group.enabled"><span class="absolute top-1 h-4 w-4 rounded-full bg-white transition-all" :class="group.enabled ? 'left-6' : 'left-1'"></span></button>
                </label>
                <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <label class="block"><span class="mb-1.5 block text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.model') }}</span><input v-model="group.model_id" class="input" maxlength="128" /></label>
                  <label class="block"><span class="mb-1.5 block text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.effort') }}</span><select v-model="group.reasoning_effort" class="input"><option v-for="effort in efforts" :key="effort" :value="effort">{{ effort }}</option></select></label>
                </div>
                <p v-if="group.reasoning_effort !== 'high'" class="mt-2 text-xs text-amber-700 dark:text-amber-400">{{ t('modelRadar.effortHint') }}</p>
                <p v-if="group.enabled && group.next_run_at" class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.nextRun', { time: new Date(group.next_run_at).toLocaleString() }) }}</p>
              </div>
            </template>
          </RadarGroupCard>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import RadarGroupCard from '@/components/model-radar/RadarGroupCard.vue'
import RadarHowItWorks from '@/components/model-radar/RadarHowItWorks.vue'
import RadarSummary from '@/components/model-radar/RadarSummary.vue'
import { adminAPI } from '@/api/admin'
import type { ModelRadarGroup, ModelRadarOverview } from '@/api/modelRadar'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()
const overview = ref<ModelRadarOverview | null>(null)
const groups = ref<ModelRadarGroup[]>([])
const loading = ref(false)
const saving = ref(false)
const running = ref(false)
const error = ref(false)
const efforts = ['minimal', 'low', 'medium', 'high', 'xhigh', 'max']
const prompt = computed(() => overview.value?.prompt || 'ciallo')
const expected = computed(() => overview.value?.expected_reply || 'Ciallo～(∠・ω< )⌒☆')
const enabledGroupIds = computed(() => groups.value.filter(group => group.enabled).map(group => group.group_id))

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

const saveConfigs = () => adminAPI.modelRadar.updateConfigs(groups.value.map(group => ({ group_id: group.group_id, group_name: group.group_name, model_id: group.model_id, reasoning_effort: group.reasoning_effort, enabled: group.enabled })))

const save = async () => {
  saving.value = true
  try {
    await saveConfigs()
    appStore.showSuccess(t('modelRadar.saved'))
    await load()
  } catch {
    appStore.showError(t('modelRadar.saveFailed'))
  } finally {
    saving.value = false
  }
}

// 「立即检测」按页面上当前的模型和强度跑：先存配置，再触发。后端在后台跑（画图题要几分钟），
// 这里轮询到每个分组都出了新的 ciallo 结果为止，最多等 3 分钟。
const run = async () => {
  running.value = true
  const startedAt = Date.now()
  const targets = enabledGroupIds.value
  try {
    await saveConfigs()
    await adminAPI.modelRadar.runNow(targets)
    for (let attempt = 0; attempt < 45; attempt++) {
      await new Promise(resolve => setTimeout(resolve, 4000))
      const fresh = await adminAPI.modelRadar.getOverview()
      overview.value = fresh
      groups.value = fresh.groups.map(group => ({ ...group }))
      const done = targets.every(id => {
        const latest = fresh.groups.find(group => group.group_id === id)?.ciallo
        return latest && new Date(latest.detected_at).getTime() >= startedAt
      })
      if (done) {
        appStore.showSuccess(t('modelRadar.ran', { n: targets.length }))
        return
      }
    }
    appStore.showError(t('modelRadar.runFailed'))
  } catch {
    appStore.showError(t('modelRadar.runFailed'))
  } finally {
    running.value = false
  }
}

onMounted(load)
</script>
