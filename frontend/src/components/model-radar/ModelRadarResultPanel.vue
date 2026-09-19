<template>
  <section class="min-w-0 border-t border-gray-100 pt-4 first:border-t-0 first:pt-0 dark:border-dark-700">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-sm font-bold text-gray-900 dark:text-white">{{ label }}</h4>
      <div v-if="result" class="flex items-center gap-2 text-xs">
        <span class="font-semibold" :class="statusClass">{{ statusLabel }}</span>
        <span class="text-gray-500 dark:text-gray-400">{{ t('modelRadar.latency', { ms: result.latency_ms }) }}</span>
      </div>
    </div>

    <div class="mt-3">
      <p class="text-xs font-semibold text-gray-500 dark:text-gray-400">{{ t('modelRadar.question') }}</p>
      <p class="mt-1 whitespace-pre-wrap break-words text-sm leading-6 text-gray-800 dark:text-gray-200">{{ result?.prompt || fallbackPrompt }}</p>
    </div>

    <div class="mt-4">
      <p class="text-xs font-semibold text-gray-500 dark:text-gray-400">{{ t('modelRadar.result') }}</p>
      <div v-if="!result" class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('modelRadar.noData') }}</div>
      <div v-else-if="result.test_type === 'drawing' && result.response_text" class="mt-2 space-y-3">
        <iframe
          class="block aspect-video w-full border border-gray-200 bg-white dark:border-dark-600"
          :srcdoc="previewDocument"
          sandbox="allow-scripts"
          loading="lazy"
          :title="label"
        />
        <details class="border-t border-gray-100 pt-3 dark:border-dark-700">
          <summary class="cursor-pointer text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('modelRadar.rawOutput') }}</summary>
          <pre class="mt-2 max-h-72 overflow-auto whitespace-pre-wrap break-words bg-gray-950 p-3 text-xs leading-5 text-gray-100">{{ result.response_text }}</pre>
        </details>
      </div>
      <pre v-else-if="result.response_text" class="mt-2 max-h-72 overflow-auto whitespace-pre-wrap break-words border border-gray-200 bg-gray-50 p-3 text-sm leading-6 text-gray-800 dark:border-dark-600 dark:bg-dark-900 dark:text-gray-100">{{ result.response_text }}</pre>
      <p v-else class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ result.error_message || t('modelRadar.previewUnavailable') }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelRadarResult } from '@/api/modelRadar'

const props = withDefaults(defineProps<{
  label: string
  result?: ModelRadarResult
  fallbackPrompt?: string
}>(), {
  result: undefined,
  fallbackPrompt: ''
})

const { t } = useI18n()

const statusLabel = computed(() => {
  if (!props.result) return t('modelRadar.noData')
  if (props.result.status === 'passed') return t('modelRadar.passed')
  if (props.result.status === 'failed') return t('modelRadar.failed')
  if (props.result.status === 'request_failed') return t('modelRadar.requestFailed')
  return t('modelRadar.passed')
})

const statusClass = computed(() => {
  if (props.result?.status === 'passed' || props.result?.status === 'pending_review') return 'text-emerald-600 dark:text-emerald-400'
  if (props.result?.status === 'failed') return 'text-red-600 dark:text-red-400'
  return 'text-gray-500 dark:text-gray-400'
})

const previewDocument = computed(() => {
  const response = stripMarkdownFence(props.result?.response_text || '')
  const security = `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; font-src data:; media-src 'none'; connect-src 'none'; frame-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'">`
  const sizing = '<style>html,body{width:100%;height:100%;margin:0;overflow:hidden;background:#fff}body{display:grid;place-items:center}svg{display:block;max-width:100%;max-height:100%}</style>'

  if (/<head[\s>]/i.test(response)) {
    return response.replace(/<head([^>]*)>/i, `<head$1>${security}${sizing}`)
  }
  if (/<html[\s>]/i.test(response)) {
    return response.replace(/<html([^>]*)>/i, `<html$1><head>${security}${sizing}</head>`)
  }
  return `<!doctype html><html><head>${security}${sizing}</head><body>${response}</body></html>`
})

function stripMarkdownFence(value: string): string {
  const trimmed = value.trim()
  const match = trimmed.match(/```(?:html|svg|xml)?\s*([\s\S]*?)\s*```/i)
  return match?.[1] ?? trimmed
}
</script>
