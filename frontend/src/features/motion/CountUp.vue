<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { canAnimate } from './env'
import { loadMotion } from './loader'
import { formatNumeric, parseNumeric } from './numeric'
import { DURATION, EASE } from './tokens'

/**
 * 数字滚动：进入视口时从 0 滚到目标值，滚完显示原文。
 * 不能解析成数字的文字（如「—」）、测试环境、减少动态效果时直接显示原文。
 */
const props = withDefaults(defineProps<{ value: string; delay?: number }>(), { delay: 0 })

const parts = parseNumeric(props.value)
const animated = canAnimate() && parts !== null && parts.value > 0
const shown = ref(animated && parts ? formatNumeric(parts, 0) : props.value)
const el = ref<HTMLElement | null>(null)

let stopView: VoidFunction | undefined
let running: { stop: () => void } | undefined
let settled = !animated

function settle() {
  settled = true
  stopView?.()
  running?.stop()
  shown.value = props.value
}

onMounted(async () => {
  if (settled || !parts || !el.value) return
  try {
    const { animate, inView } = await loadMotion()
    if (settled || !el.value) return
    stopView = inView(
      el.value,
      () => {
        stopView?.()
        running = animate(0, parts.value, {
          duration: DURATION.count,
          delay: props.delay,
          ease: EASE.expoOut,
          onUpdate: (v) => {
            if (!settled) shown.value = formatNumeric(parts, v)
          },
          onComplete: settle,
        })
      },
      { amount: 0.6 },
    )
  } catch {
    settle()
  }
})

// 数据刷新或切换语言后值变了：不再重播，直接显示新值
watch(() => props.value, settle)
onBeforeUnmount(() => {
  stopView?.()
  running?.stop()
})
</script>

<template>
  <span ref="el">{{ shown }}</span>
</template>
