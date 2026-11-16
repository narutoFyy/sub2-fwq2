<template>
  <AppLayout>
    <div class="w-full min-w-0 space-y-5 pb-8">
      <header class="flex flex-wrap items-start justify-between gap-4">
        <div class="min-w-0">
          <h1 class="page-title">{{ t('modelRadar.configure') }}</h1>
          <p class="page-description">{{ t('modelRadar.configureSubtitle') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <span v-if="dirty" class="text-xs font-medium text-amber-700 dark:text-amber-400">{{ t('modelRadar.unsaved') }}</span>
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="load"><Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />{{ t('modelRadar.refresh') }}</button>
          <button type="button" class="btn btn-secondary" :disabled="running || !enabledGroupIds.length" @click="run(enabledGroupIds)"><Icon name="play" size="sm" />{{ running ? t('modelRadar.runningNow') : t('modelRadar.runNow') }}</button>
          <button type="button" class="btn btn-primary" :disabled="saving || !groups.length" @click="save"><Icon name="check" size="sm" />{{ t('modelRadar.save') }}</button>
        </div>
      </header>

      <div v-if="loading && !overview" class="flex justify-center py-12"><LoadingSpinner /></div>
      <div v-else-if="error" class="card p-5 text-sm text-red-700 dark:text-red-300">{{ t('modelRadar.loadFailed') }}</div>
      <template v-else>
        <RadarHowItWorks :prompt="prompt" :expected="expected" />
        <RadarSettingsPanel v-model="settings" />
        <div v-if="!groups.length" class="card p-10 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('modelRadar.empty') }}</div>
        <template v-else>
          <RadarSummary :groups="groups" />
          <div class="flex flex-wrap items-center justify-end gap-2">
            <button type="button" class="btn btn-secondary btn-sm" @click="setAll(true)">{{ t('modelRadar.enableAll') }}</button>
            <button type="button" class="btn btn-secondary btn-sm" @click="setAll(false)">{{ t('modelRadar.disableAll') }}</button>
            <button type="button" class="btn btn-secondary btn-sm text-red-700 dark:text-red-400" @click="ask({ kind: 'all' })"><Icon name="trash" size="sm" />{{ t('modelRadar.clearAll') }}</button>
          </div>
          <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
            <RadarGroupCard v-for="group in groups" :key="group.group_id" :group="group" :prompt="prompt" :confirm-rounds="settings.nerf_confirm_rounds">
              <template #badges>
                <div v-if="!group.enabled || !group.visible" class="mt-2 flex flex-wrap gap-1.5">
                  <span v-if="!group.enabled" class="rounded-full bg-gray-900/[0.06] px-2 py-0.5 text-xs text-gray-600 dark:bg-white/[0.08] dark:text-gray-300">{{ t('modelRadar.offBadge') }}</span>
                  <span v-if="!group.visible" class="inline-flex items-center gap-1 rounded-full bg-gray-900/[0.06] px-2 py-0.5 text-xs text-gray-600 dark:bg-white/[0.08] dark:text-gray-300"><Icon name="eyeOff" size="xs" />{{ t('modelRadar.hiddenBadge') }}</span>
                </div>
              </template>
              <template #config>
                <div class="mt-4 space-y-3 rounded-xl bg-gray-900/[0.03] p-4 dark:bg-white/[0.04]">
                  <div class="flex items-center justify-between gap-3">
                    <span class="text-sm font-medium text-gray-800 dark:text-gray-100">{{ t('modelRadar.enabled') }}</span>
                    <Toggle v-model="group.enabled" :aria-label="t('modelRadar.enabled')" />
                  </div>
                  <div>
                    <div class="flex items-center justify-between gap-3">
                      <span class="text-sm font-medium text-gray-800 dark:text-gray-100">{{ t('modelRadar.visible') }}</span>
                      <Toggle v-model="group.visible" :aria-label="t('modelRadar.visible')" />
                    </div>
                    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.visibleNote') }}</p>
                  </div>
                  <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
                    <label class="block sm:col-span-2"><span class="mb-1.5 block text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.model') }}</span><input v-model="group.model_id" class="input" maxlength="128" /></label>
                    <label class="block"><span class="mb-1.5 block text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.effort') }}</span><select v-model="group.reasoning_effort" class="input"><option v-for="effort in efforts" :key="effort" :value="effort">{{ effort }}</option></select></label>
                    <label class="block"><span class="mb-1.5 block text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.groupInterval') }}</span>
                      <select v-model.number="group.interval_minutes" class="input">
                        <option :value="0">{{ t('modelRadar.followGlobal', { label: intervalLabel(settings.interval_minutes, t) }) }}</option>
                        <option v-for="option in intervalOptions(group.interval_minutes)" :key="option" :value="option">{{ intervalLabel(option, t) }}</option>
                      </select>
                    </label>
                  </div>
                  <p v-if="group.reasoning_effort !== 'high'" class="text-xs text-amber-700 dark:text-amber-400">{{ t('modelRadar.effortHint') }}</p>
                  <p v-if="group.enabled && group.next_run_at" class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.nextRun', { time: new Date(group.next_run_at).toLocaleString() }) }}</p>
                </div>
              </template>
              <template #actions>
                <div class="mt-5 flex flex-wrap gap-2 border-t border-gray-900/[0.06] pt-4 dark:border-white/[0.09]">
                  <button type="button" class="btn btn-secondary btn-sm" :disabled="running" @click="run([group.group_id])"><Icon name="play" size="sm" />{{ t('modelRadar.runOne') }}</button>
                  <button v-if="group.ciallo" type="button" class="btn btn-secondary btn-sm" @click="ask({ kind: 'latest', group })">{{ t('modelRadar.deleteLatest') }}</button>
                  <button v-if="group.timeline.length || group.drawing" type="button" class="btn btn-secondary btn-sm text-red-700 dark:text-red-400" @click="ask({ kind: 'group', group })"><Icon name="trash" size="sm" />{{ t('modelRadar.clearGroup') }}</button>
                </div>
              </template>
            </RadarGroupCard>
          </div>
        </template>
      </template>
    </div>
    <ConfirmDialog :show="!!pending" :title="pendingTitle" :message="pendingMessage" danger @confirm="confirmPending" @cancel="pending = null" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import RadarGroupCard from '@/components/model-radar/RadarGroupCard.vue'
import RadarHowItWorks from '@/components/model-radar/RadarHowItWorks.vue'
import RadarSettingsPanel from '@/components/model-radar/RadarSettingsPanel.vue'
import RadarSummary from '@/components/model-radar/RadarSummary.vue'
import { intervalLabel, intervalOptions } from '@/components/model-radar/radar'
import { adminAPI } from '@/api/admin'
import type { ModelRadarGroup, ModelRadarOverview, ModelRadarSettings } from '@/api/modelRadar'
import { useAppStore } from '@/stores/app'

type Pending = { kind: 'all' } | { kind: 'group' | 'latest'; group: ModelRadarGroup }

const DEFAULT_SETTINGS: ModelRadarSettings = { interval_minutes: 30, ciallo_attempts: 3, nerf_confirm_rounds: 1, drawing_enabled: true, retention_days: 0 }

const { t } = useI18n()
const appStore = useAppStore()
const overview = ref<ModelRadarOverview | null>(null)
const groups = ref<ModelRadarGroup[]>([])
const settings = ref<ModelRadarSettings>({ ...DEFAULT_SETTINGS })
const loading = ref(false)
const saving = ref(false)
const running = ref(false)
const error = ref(false)
const pending = ref<Pending | null>(null)
const efforts = ['minimal', 'low', 'medium', 'high', 'xhigh', 'max']
const prompt = computed(() => overview.value?.prompt || 'ciallo')
const expected = computed(() => overview.value?.expected_reply || 'Ciallo～(∠・ω< )⌒☆')
const enabledGroupIds = computed(() => groups.value.filter(group => group.enabled).map(group => group.group_id))

const configOf = (group: ModelRadarGroup) => ({ group_id: group.group_id, group_name: group.group_name, model_id: group.model_id, reasoning_effort: group.reasoning_effort, enabled: group.enabled, interval_minutes: group.interval_minutes || 0, visible: group.visible })
const snapshot = () => JSON.stringify([settings.value, groups.value.map(configOf)])
const saved = ref('')
const dirty = computed(() => !!overview.value && saved.value !== snapshot())

// apply 把后端总览灌进页面。keepEdits 为真时（轮询检测结果期间）保留没保存的表单改动。
const apply = (fresh: ModelRadarOverview, keepEdits = false) => {
  const edits = new Map(groups.value.map(group => [group.group_id, configOf(group)]))
  overview.value = fresh
  groups.value = fresh.groups.map(group => ({ ...group, visible: group.visible ?? true, interval_minutes: group.interval_minutes ?? 0, ...(keepEdits ? edits.get(group.group_id) : undefined) }))
  if (!keepEdits) {
    settings.value = { ...DEFAULT_SETTINGS, ...fresh.settings }
    saved.value = snapshot()
  }
}

const load = async () => {
  loading.value = true
  error.value = false
  try {
    apply(await adminAPI.modelRadar.getOverview())
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

const saveAll = async () => {
  await adminAPI.modelRadar.updateSettings(settings.value)
  await adminAPI.modelRadar.updateConfigs(groups.value.map(configOf))
  saved.value = snapshot()
}

const save = async () => {
  saving.value = true
  try {
    await saveAll()
    appStore.showSuccess(t('modelRadar.saved'))
    await load()
  } catch {
    appStore.showError(t('modelRadar.saveFailed'))
  } finally {
    saving.value = false
  }
}

const setAll = (enabled: boolean) => groups.value.forEach(group => { group.enabled = enabled })

// 「立即检测」按页面上当前的设置跑：先存，再触发。后端在后台跑（画图题要几分钟），
// 这里轮询到每个目标分组都出了新的 ciallo 结果为止，最多等 3 分钟。
const run = async (targets: number[]) => {
  if (!targets.length) return
  running.value = true
  const startedAt = Date.now()
  try {
    await saveAll()
    await adminAPI.modelRadar.runNow(targets)
    for (let attempt = 0; attempt < 45; attempt++) {
      await new Promise(resolve => setTimeout(resolve, 4000))
      const fresh = await adminAPI.modelRadar.getOverview()
      apply(fresh, true)
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

const ask = (next: Pending) => { pending.value = next }
const pendingTitle = computed(() => {
  if (!pending.value) return ''
  return pending.value.kind === 'all' ? t('modelRadar.clearAll') : pending.value.kind === 'group' ? t('modelRadar.clearGroup') : t('modelRadar.deleteLatest')
})
const pendingMessage = computed(() => {
  const current = pending.value
  if (!current) return ''
  if (current.kind === 'all') return t('modelRadar.clearAllConfirm')
  return t(current.kind === 'group' ? 'modelRadar.clearGroupConfirm' : 'modelRadar.deleteLatestConfirm', { name: current.group.group_name })
})

const confirmPending = async () => {
  const current = pending.value
  pending.value = null
  if (!current) return
  try {
    if (current.kind === 'latest') {
      if (current.group.ciallo) await adminAPI.modelRadar.deleteResult(current.group.ciallo.id)
      appStore.showSuccess(t('modelRadar.deleted'))
    } else {
      const { deleted } = current.kind === 'all' ? await adminAPI.modelRadar.clearAllResults() : await adminAPI.modelRadar.clearGroupResults(current.group.group_id)
      appStore.showSuccess(t('modelRadar.cleared', { n: deleted }))
    }
    apply(await adminAPI.modelRadar.getOverview(), true)
  } catch {
    appStore.showError(t('modelRadar.deleteFailed'))
  }
}

onMounted(load)
</script>
