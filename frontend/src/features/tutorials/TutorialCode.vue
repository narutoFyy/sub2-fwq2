<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { useTutorialCopy } from './content'
const props = defineProps<{ code: string; label: string }>()
const copy = useTutorialCopy()
const status = ref<'idle' | 'copied' | 'error'>('idle')
let timer: ReturnType<typeof setTimeout> | undefined
async function copyText() {
  try { await navigator.clipboard.writeText(props.code); status.value = 'copied' }
  catch { status.value = 'error' }
  clearTimeout(timer)
  timer = setTimeout(() => { status.value = 'idle' }, 2500)
}
onBeforeUnmount(() => clearTimeout(timer))
</script>

<template>
  <div class="guide-code">
    <div class="code-label"><span>{{ label }}</span><button type="button" @click="copyText">{{ status === 'copied' ? copy.copied : copy.copy }}</button></div>
    <pre><code>{{ code }}</code></pre>
    <p v-if="status !== 'idle'" class="copy-status" role="status">{{ status === 'error' ? copy.copyFailed : copy.copied }}</p>
  </div>
</template>

<style scoped>
.guide-code { background: #111115; color: #eeeef1; border-radius: 10px; margin: 20px 0; overflow: hidden; }
.code-label { display: flex; align-items: center; justify-content: space-between; padding: 10px 18px 0; font-size: 11px; color: #a2a5b0; }
button { min-height: 36px; padding: 0 8px; color: #d8d8e0; border-radius: 4px; }
button:hover { color: white; background: #25252c; }
button:focus-visible { outline: 2px solid #bceef2; }
pre { padding: 14px 18px 20px; overflow-x: auto; font: 13px/1.8 'Geist Mono Variable', monospace; }
.copy-status { padding: 0 18px 12px; font-size: 12px; color: #bceef2; }
</style>
