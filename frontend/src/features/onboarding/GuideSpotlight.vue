<script setup lang="ts">
import { computed } from 'vue'
import type { TargetRect } from './useGuideTarget'

const props = withDefaults(defineProps<{ rect: TargetRect | null; cursor?: boolean; dim?: boolean }>(), { dim: true })
const PAD = 6

const holeStyle = computed(() => {
  const r = props.rect
  if (!r) return { opacity: 0, transform: 'translate3d(50vw, 50vh, 0)', width: '0px', height: '0px' }
  return {
    opacity: 1,
    transform: `translate3d(${r.x - PAD}px, ${r.y - PAD}px, 0)`,
    width: `${r.w + PAD * 2}px`,
    height: `${r.h + PAD * 2}px`
  }
})

const cursorStyle = computed(() => {
  const r = props.rect
  if (!r) return { opacity: 0 }
  return { opacity: 1, transform: `translate3d(${r.x + r.w - 18}px, ${r.y + r.h - 8}px, 0)` }
})
</script>

<template>
  <div class="og-hole" :class="{ 'og-hole--clear': !dim }" :style="holeStyle" aria-hidden="true"><span class="og-ring"></span></div>
  <div v-if="cursor" class="og-cursor" :style="cursorStyle" aria-hidden="true">
    <span class="og-ripple"></span>
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none"><path d="M5 3l14 8-6 1.6L9.6 19 5 3z" fill="currentColor" stroke="var(--og-bg)" stroke-width="1.4" stroke-linejoin="round" /></svg>
  </div>
</template>
