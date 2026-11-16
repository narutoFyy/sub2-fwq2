<template>
  <section class="card p-5 sm:p-6">
    <div class="flex flex-wrap items-baseline justify-between gap-2">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('modelRadar.settingsTitle') }}</h2>
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.settingsHint') }}</p>
    </div>
    <div class="mt-4 grid grid-cols-1 gap-x-6 gap-y-5 sm:grid-cols-2 xl:grid-cols-3">
      <label class="block">
        <span class="mb-1.5 block text-sm font-medium text-gray-800 dark:text-gray-100">{{ t('modelRadar.interval') }}</span>
        <select v-model.number="model.interval_minutes" class="input">
          <option v-for="option in intervalOptions(model.interval_minutes)" :key="option" :value="option">{{ intervalLabel(option, t) }}</option>
        </select>
        <span class="mt-1.5 block text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('modelRadar.intervalNote') }}</span>
      </label>
      <label class="block">
        <span class="mb-1.5 block text-sm font-medium text-gray-800 dark:text-gray-100">{{ t('modelRadar.attempts') }}</span>
        <select v-model.number="model.ciallo_attempts" class="input">
          <option v-for="n in 5" :key="n" :value="n">{{ t('modelRadar.times', { n }) }}</option>
        </select>
        <span class="mt-1.5 block text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('modelRadar.attemptsNote') }}</span>
      </label>
      <label class="block">
        <span class="mb-1.5 block text-sm font-medium text-gray-800 dark:text-gray-100">{{ t('modelRadar.confirmRounds') }}</span>
        <select v-model.number="model.nerf_confirm_rounds" class="input">
          <option v-for="n in 5" :key="n" :value="n">{{ t('modelRadar.rounds', { n }) }}</option>
        </select>
        <span class="mt-1.5 block text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('modelRadar.confirmRoundsNote') }}</span>
      </label>
      <label class="block">
        <span class="mb-1.5 block text-sm font-medium text-gray-800 dark:text-gray-100">{{ t('modelRadar.retention') }}</span>
        <select v-model.number="model.retention_days" class="input">
          <option v-for="option in retentionOptions" :key="option" :value="option">{{ option ? t('modelRadar.days', { n: option }) : t('modelRadar.retentionForever') }}</option>
        </select>
        <span class="mt-1.5 block text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('modelRadar.retentionNote') }}</span>
      </label>
      <div class="sm:col-span-2">
        <div class="flex items-center justify-between gap-3">
          <span class="text-sm font-medium text-gray-800 dark:text-gray-100">{{ t('modelRadar.drawingEnabled') }}</span>
          <Toggle v-model="model.drawing_enabled" :aria-label="t('modelRadar.drawingEnabled')" />
        </div>
        <span class="mt-1.5 block text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('modelRadar.drawingEnabledNote') }}</span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import type { ModelRadarSettings } from '@/api/modelRadar'
import { intervalLabel, intervalOptions } from './radar'

const model = defineModel<ModelRadarSettings>({ required: true })
const { t } = useI18n()

const retentionOptions = computed(() => [...new Set([0, 7, 14, 30, 90, 180, model.value.retention_days])].sort((a, b) => a - b))
</script>
