<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { useTutorialCopy, tutorialClients, clientName, type TutorialClient } from './content'
import './tutorials.css'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const copy = useTutorialCopy()
const router = useRouter()
const body = ref<HTMLElement>()
async function openGuide(client: TutorialClient) {
  await router.push(`/tutorials/${client}`)
  emit('close')
}
function trapTab(event: KeyboardEvent) {
  if (event.key !== 'Tab' || !body.value) return
  const controls = body.value.closest('[role="dialog"]')?.querySelectorAll<HTMLElement>('button, a[href]')
  if (!controls?.length) return
  const first = controls[0]!
  const last = controls[controls.length - 1]!
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() }
  if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
}
watch(() => props.show, async show => {
  if (show) { await nextTick(); body.value?.querySelector<HTMLElement>('button')?.focus() }
}, { immediate: true })
</script>

<template>
  <BaseDialog :show="show" :title="copy.welcomeTitle" :z-index="130" :show-close-button="false" @close="emit('close')">
    <div ref="body" class="tutorial-surface welcome-content" @keydown="trapTab">
      <p class="welcome-desc">{{ copy.welcomeDesc }}</p>
      <ol class="welcome-path"><li v-for="(step, i) in [copy.steps[0], copy.steps[1], copy.steps[3]]" :key="step"><span>{{ i + 1 }}</span>{{ step }}</li></ol>
      <div class="welcome-guides"><button v-for="client in tutorialClients" :key="client" type="button" @click="openGuide(client)"><ModelIcon :model="client === 'codex' ? 'gpt-5' : 'claude'" :class="{ 'tutorial-codex-icon': client === 'codex' }" size="28px" /><span>{{ clientName(client) }}<small>{{ copy.read }}</small></span><span aria-hidden="true">↗</span></button></div>
      <p class="welcome-foot">{{ copy.welcomeFoot }}</p>
      <button type="button" class="welcome-later" @click="emit('close')">{{ copy.later }}</button>
    </div>
  </BaseDialog>
</template>

<style scoped>
:global(.modal-overlay:has(.welcome-content) .modal-content) { background: #fafafa; color: #0a0a0b; }
:global(.dark .modal-overlay:has(.welcome-content) .modal-content) { background: #141417; color: #ededef; border-color: #2b2b30; }
:global(.dark .modal-overlay:has(.welcome-content) .modal-header) { border-color: #2b2b30; }
.welcome-content { min-height: 0; background: transparent; }
.welcome-desc { font-size: 14px; color: var(--guide-muted); line-height: 1.9; }
.welcome-path { display: flex; align-items: center; gap: 16px; padding: 24px 0; flex-wrap: wrap; }
.welcome-path li { display: flex; gap: 7px; align-items: center; font-size: 11px; color: var(--guide-muted); }
.welcome-path li > span { background: var(--guide-panel); border: 1px solid var(--guide-line); width: 20px; height: 20px; border-radius: 50%; display: grid; place-items: center; }
.welcome-guides { display: grid; gap: 12px; }
.welcome-guides button { width: 100%; display: flex; align-items: center; gap: 16px; border-radius: 12px; border: 1px solid var(--guide-line); padding: 20px; text-align: left; transition: background .15s; }
.welcome-guides button:hover { background: var(--guide-panel); }
.welcome-guides button > span:last-child { margin-left: auto; }
.welcome-guides small { display: block; color: var(--guide-muted); font-size: 12px; margin-top: 5px; }
.welcome-foot { font-size: 12px; color: var(--guide-muted); text-align: center; line-height: 1.8; margin: 24px 0 8px; }
.welcome-later { display: block; margin: auto; padding: 12px 24px; font-size: 13px; }
@media (prefers-reduced-motion: reduce) { .welcome-guides button { transition: none; } }
</style>
