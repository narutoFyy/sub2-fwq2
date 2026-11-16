<template>
  <AppLayout>
    <div class="w-full min-w-0 space-y-5 pb-8">
      <header class="flex flex-wrap items-start justify-between gap-4">
        <div class="min-w-0">
          <h1 class="page-title">{{ t('modelRadar.title') }}</h1>
          <p class="page-description">{{ t('modelRadar.subtitle', { minutes: overview?.interval_minutes ?? 30, prompt }) }}</p>
        </div>
        <button type="button" class="btn btn-secondary" :disabled="loading" @click="load">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />{{ t('modelRadar.refresh') }}
        </button>
      </header>

      <RadarHowItWorks :prompt="prompt" :expected="expected" />

      <div v-if="loading && !overview" class="flex justify-center py-12"><LoadingSpinner /></div>
      <div v-else-if="error" class="card p-5 text-sm text-red-700 dark:text-red-300">{{ t('modelRadar.loadFailed') }}</div>
      <div v-else-if="!overview?.groups.length" class="card p-10 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('modelRadar.empty') }}</div>
      <template v-else>
        <RadarSummary :groups="overview.groups" />
        <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <RadarGroupCard v-for="group in overview.groups" :key="group.group_id" :group="group" :prompt="prompt" />
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
import { modelRadarAPI, type ModelRadarOverview } from '@/api/modelRadar'

const { t } = useI18n()
const overview = ref<ModelRadarOverview | null>(null)
const loading = ref(false)
const error = ref(false)
const prompt = computed(() => overview.value?.prompt || 'ciallo')
const expected = computed(() => overview.value?.expected_reply || 'Ciallo～(∠・ω< )⌒☆')

const load = async () => {
  loading.value = true
  error.value = false
  try { overview.value = await modelRadarAPI.getOverview() } catch { error.value = true } finally { loading.value = false }
}

onMounted(load)
</script>
