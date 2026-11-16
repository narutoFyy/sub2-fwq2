<template>
  <div class="space-y-3">
    <iframe
      class="block aspect-video w-full rounded-xl bg-white"
      :srcdoc="previewDocument"
      sandbox="allow-scripts"
      loading="lazy"
      :title="title"
    />
    <details>
      <summary class="cursor-pointer text-xs text-gray-500 dark:text-gray-400">{{ t('modelRadar.rawOutput') }}</summary>
      <pre class="mt-2 max-h-72 overflow-auto whitespace-pre-wrap break-words rounded-xl bg-gray-950 p-3 text-xs leading-5 text-gray-100">{{ responseText }}</pre>
    </details>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ responseText: string; title: string }>()
const { t } = useI18n()

// 模型输出当成不可信内容：放进沙箱 iframe，并用 CSP 掐掉一切外联。
const previewDocument = computed(() => {
  const response = stripMarkdownFence(props.responseText)
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
