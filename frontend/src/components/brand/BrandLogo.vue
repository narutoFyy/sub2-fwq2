<template>
  <component :is="to ? RouterLink : 'span'" v-bind="to ? { to } : {}" class="brand-logo" :aria-label="name">
    <!-- 石头堆（cairn）：登山路上指路的中转标记。顶石用强调色，悬停时轻轻找平衡 -->
    <svg class="brand-mark" viewBox="0 0 32 32" :width="size" :height="size" aria-hidden="true">
      <g class="stone stone-base">
        <path d="M3.8 26.3C3.6 22.7 9.2 20.5 16.2 20.5C23.5 20.5 28.5 22.3 28.4 25.6C28.3 28.3 23.9 29.5 16.1 29.5C8.7 29.5 3.9 28.6 3.8 26.3Z" />
      </g>
      <g class="stone stone-mid">
        <path d="M8.3 17.1C8.1 14.4 11.7 12.7 16.8 12.7C21.8 12.7 24.9 14.1 24.7 16.6C24.5 18.6 21.4 19.3 16.3 19.3C11.4 19.3 8.5 18.7 8.3 17.1Z" />
      </g>
      <g class="stone stone-top">
        <path d="M11.4 9.5C11.1 7.1 13.3 5.2 16.6 5.2C19.8 5.2 21.7 6.7 21.5 8.8C21.3 10.6 19.2 11.3 16.1 11.3C13.3 11.3 11.6 10.7 11.4 9.5Z" />
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
  { to: undefined, size: 30, showName: true },
)
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
}
.stone {
  fill: currentColor;
  transform-box: fill-box;
  transform-origin: 50% 100%;
  transition: transform 0.7s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.stone-top {
  fill: var(--accent, #e11d48);
  transition-duration: 0.9s;
}
.brand-logo:hover .stone-top {
  transform: translate(-0.6px, -0.9px) rotate(-6deg);
}
.brand-logo:hover .stone-mid {
  transform: rotate(2.5deg);
}
.brand-name {
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.08em;
  white-space: nowrap;
}
@media (prefers-reduced-motion: reduce) {
  .stone {
    transition: none;
  }
}
</style>
