<script setup lang="ts">
/** 接通后的庆祝：一小撮彩纸从卡片上方迸开再落下（纯 CSS，约 1.8 秒）。减少动态效果时不渲染。 */
const COLORS = ['var(--og-accent)', '#f5a524', '#f31260', '#7c5cff', '#17c964', '#0070f0']
const pieces = Array.from({ length: 42 }, (_, i) => {
  const angle = (Math.PI * 2 * i) / 42 + Math.random() * 0.5
  const power = 120 + Math.random() * 190
  return {
    x: Math.round(Math.cos(angle) * power),
    y: Math.round(Math.sin(angle) * power * 0.7 - 90),
    r: Math.round(Math.random() * 720 - 360),
    d: Math.round(Math.random() * 160),
    w: 6 + Math.round(Math.random() * 6),
    c: COLORS[i % COLORS.length]
  }
})
const reduced = typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
</script>

<template>
  <div v-if="!reduced" class="og-confetti" aria-hidden="true">
    <i v-for="(p, i) in pieces" :key="i" :style="{ '--x': p.x, '--y': p.y, '--r': `${p.r}deg`, '--d': `${p.d}ms`, '--w': `${p.w}px`, background: p.c }"></i>
  </div>
</template>
