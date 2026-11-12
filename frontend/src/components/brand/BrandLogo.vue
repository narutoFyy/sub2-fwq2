<template>
  <component :is="to ? RouterLink : 'span'" v-bind="to ? { to } : {}" class="brand-logo" :aria-label="name">
    <!-- 一块被斜线切开的卵石：石头 + 向前的"中转"。单色，悬停时上半块轻轻抬起 -->
    <svg class="brand-mark" viewBox="0 0 32 32" :width="size" :height="size" aria-hidden="true">
      <defs>
        <clipPath :id="`${uid}-top`"><polygon points="-4,-4 36,-4 36,12.2 -4,15.9" /></clipPath>
        <clipPath :id="`${uid}-bottom`"><polygon points="-4,18.5 36,14.8 36,36 -4,36" /></clipPath>
      </defs>
      <g class="half half-top">
        <path :clip-path="`url(#${uid}-top)`" :d="PEBBLE" />
      </g>
      <g class="half half-bottom">
        <path :clip-path="`url(#${uid}-bottom)`" :d="PEBBLE" />
      </g>
    </svg>
    <span v-if="showName" class="brand-name">{{ name }}</span>
  </component>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'

withDefaults(
  defineProps<{
    name: string
    to?: string
    size?: number
    showName?: boolean
  }>(),
  { to: undefined, size: 24, showName: true },
)

const PEBBLE = 'M16 5.5C23.9 5.5 29.5 9.3 29.5 15.6C29.5 22 23.7 26.5 16 26.5C8.3 26.5 2.5 22 2.5 15.6C2.5 9.3 8.1 5.5 16 5.5Z'
const uid = `brand-${Math.random().toString(36).slice(2, 8)}`
</script>

<style scoped>
.brand-logo {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: inherit;
  -webkit-tap-highlight-color: transparent;
}
.brand-mark {
  flex-shrink: 0;
  overflow: visible;
  fill: currentColor;
}
.half {
  transition: transform 0.45s cubic-bezier(0.22, 1, 0.36, 1);
}
.brand-logo:hover .half-top {
  transform: translate(0.8px, -0.9px);
}
.brand-name {
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.01em;
  white-space: nowrap;
}
@media (prefers-reduced-motion: reduce) {
  .half {
    transition: none;
  }
}
</style>
