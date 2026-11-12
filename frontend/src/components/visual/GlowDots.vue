<template>
  <canvas
    ref="canvasRef"
    aria-hidden="true"
    class="glow-dots"
    :class="{ 'glow-dots--ready': ready }"
  ></canvas>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { GlowDotsRenderer, type GlowDotsOptions, type GlowTheme } from '@/utils/glowDots'

const props = withDefaults(
  defineProps<{
    /** auto：跟随 <html> 上的 dark 类（深色=午夜霓虹，浅色=樱粉） */
    theme?: GlowTheme | 'auto'
    centerX?: number
    centerY?: number
    mobileCenterY?: number
    scale?: number
    cell?: number
    maxCells?: number
    interactive?: boolean
    seed?: number
  }>(),
  {
    theme: 'auto',
    centerX: 0.58,
    centerY: 0.6,
    mobileCenterY: 0.52,
    scale: 1,
    cell: 5,
    maxCells: 30000,
    interactive: true,
    seed: 31,
  },
)

const canvasRef = ref<HTMLCanvasElement | null>(null)
const ready = ref(false)
let renderer: GlowDotsRenderer | null = null
let classObserver: MutationObserver | null = null

function resolveTheme(): GlowTheme {
  if (props.theme !== 'auto') return props.theme
  return document.documentElement.classList.contains('dark') ? 'midnight' : 'rose'
}

onMounted(() => {
  if (!canvasRef.value) return
  const options: GlowDotsOptions = {
    centerX: props.centerX,
    centerY: props.centerY,
    mobileCenterY: props.mobileCenterY,
    scale: props.scale,
    cell: props.cell,
    maxCells: props.maxCells,
    interactive: props.interactive,
    seed: props.seed,
    onReady: () => {
      ready.value = true
    },
  }
  renderer = GlowDotsRenderer.create(canvasRef.value, options)
  if (!renderer) return
  renderer.setTheme(resolveTheme())
  if (typeof MutationObserver !== 'undefined') {
    classObserver = new MutationObserver(() => renderer?.setTheme(resolveTheme()))
    classObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  }
})

watch(
  () => props.theme,
  () => renderer?.setTheme(resolveTheme()),
)

onBeforeUnmount(() => {
  classObserver?.disconnect()
  renderer?.destroy()
  renderer = null
})
</script>

<style scoped>
.glow-dots {
  display: block;
  width: 100%;
  height: 100%;
  opacity: 0;
  transition: opacity 1.2s ease-out;
}
.glow-dots--ready {
  opacity: 1;
}
</style>
