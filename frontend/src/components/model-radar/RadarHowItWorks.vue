<template>
  <section class="card grid grid-cols-1 gap-5 p-5 sm:p-6 lg:grid-cols-[auto_1fr_1fr] lg:items-center lg:gap-8">
    <div>
      <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('modelRadar.howSend') }}</p>
      <p class="mt-2 inline-block rounded-2xl rounded-br-md bg-gray-950 px-4 py-2 font-mono text-base text-gray-50 dark:bg-gray-200 dark:text-gray-950">{{ prompt }}</p>
    </div>
    <div class="min-w-0">
      <p class="flex items-center gap-2 text-xs font-medium text-emerald-700 dark:text-emerald-400">
        <span class="h-2 w-2 rounded-full bg-emerald-500"></span>{{ t('modelRadar.howGood') }}
        <span class="font-normal text-gray-500 dark:text-gray-400">{{ t('modelRadar.howGoodNote') }}</span>
      </p>
      <p class="mt-2 inline-block max-w-full break-words rounded-2xl rounded-bl-md bg-emerald-500/10 px-4 py-2 text-base text-gray-900 dark:text-gray-100">{{ expected }}</p>
    </div>
    <div class="min-w-0">
      <p class="flex items-center gap-2 text-xs font-medium text-red-700 dark:text-red-400">
        <span class="h-2 w-2 rounded-full bg-red-500"></span>{{ t('modelRadar.howBad') }}
        <span class="font-normal text-gray-500 dark:text-gray-400">{{ t('modelRadar.howBadNote') }}</span>
      </p>
      <p class="mt-2 inline-block max-w-full break-words rounded-2xl rounded-bl-md bg-red-500/10 px-4 py-2 text-base text-gray-900 dark:text-gray-100">{{ t('modelRadar.badExample') }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{ prompt: string; expected: string }>()
const { t } = useI18n()
</script>
