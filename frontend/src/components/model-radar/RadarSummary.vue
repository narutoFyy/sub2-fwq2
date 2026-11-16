<template>
  <div class="grid grid-cols-3 gap-3">
    <div v-for="tile in tiles" :key="tile.key" class="card px-4 py-3 sm:px-5 sm:py-4">
      <p class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400"><span class="h-2 w-2 rounded-full" :class="tile.dot"></span>{{ tile.label }}</p>
      <p class="mt-1 text-2xl font-semibold tabular-nums sm:text-3xl" :class="tile.count ? 'text-gray-900 dark:text-white' : 'text-gray-300 dark:text-gray-600'">{{ tile.count }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelRadarGroup } from '@/api/modelRadar'
import { summarize } from './radar'

const props = defineProps<{ groups: ModelRadarGroup[] }>()
const { t } = useI18n()

const tiles = computed(() => {
  const counts = summarize(props.groups)
  return [
    { key: 'ok', label: t('modelRadar.summaryOk'), count: counts.ok, dot: 'bg-emerald-500' },
    { key: 'nerfed', label: t('modelRadar.summaryNerfed'), count: counts.nerfed, dot: 'bg-red-500' },
    { key: 'unknown', label: t('modelRadar.summaryUnknown'), count: counts.unknown, dot: 'bg-gray-400' }
  ]
})
</script>
