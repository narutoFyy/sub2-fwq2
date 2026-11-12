<template>
  <component
    :is="isExternal ? 'a' : RouterLink"
    v-bind="isExternal ? { href: to, target: '_blank', rel: 'noopener noreferrer' } : { to }"
    class="roll-link"
    :class="[`roll-link--${size}`, `roll-link--${variant}`]"
    :aria-label="label"
  >
    <!-- 悬停时文字上下翻滚、箭头横向接力，和语言切换同一套动效 -->
    <span class="roll-text">
      <span class="roll-track" aria-hidden="true">
        <span>{{ label }}</span>
        <span>{{ label }}</span>
      </span>
    </span>
    <span v-if="arrow" class="roll-arrow" aria-hidden="true">
      <svg viewBox="0 0 16 16"><path d="M3 8h9M8.5 4.5 12 8l-3.5 3.5" /></svg>
      <svg viewBox="0 0 16 16"><path d="M3 8h9M8.5 4.5 12 8l-3.5 3.5" /></svg>
    </span>
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'

const props = withDefaults(
  defineProps<{
    to: string
    label: string
    size?: 'sm' | 'lg'
    variant?: 'solid' | 'plain'
    arrow?: boolean
  }>(),
  { size: 'sm', variant: 'solid', arrow: false },
)

const isExternal = computed(() => /^https?:\/\//.test(props.to))
</script>

<style scoped>
.roll-link {
  --ease: cubic-bezier(0.65, 0, 0.25, 1);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border-radius: 9999px;
  font-family: inherit;
  font-weight: 500;
  white-space: nowrap;
  transition:
    transform 0.35s cubic-bezier(0.34, 1.56, 0.64, 1),
    background-color 0.2s,
    box-shadow 0.2s;
  -webkit-tap-highlight-color: transparent;
}
.roll-link:active {
  transform: scale(0.97);
}
.roll-link:focus {
  outline: none;
}
.roll-link:focus-visible {
  box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 22%, transparent);
}

.roll-link--solid {
  background: var(--btn-bg, #0a0a0b);
  color: var(--btn-fg, #fafafa);
}
.roll-link--solid:hover {
  background: var(--btn-bg-hover, #26262a);
}
.roll-link--plain {
  color: var(--muted, inherit);
  transition:
    color 0.2s,
    transform 0.35s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.roll-link--plain:hover {
  color: var(--ink, inherit);
}

.roll-link--sm {
  height: 36px;
  padding: 0 14px;
  font-size: 14px;
}
.roll-link--lg {
  height: 44px;
  padding: 0 20px;
  font-size: 15px;
}
.roll-link--plain.roll-link--sm,
.roll-link--plain.roll-link--lg {
  padding: 0 8px;
}

.roll-text {
  display: inline-block;
  height: 1.3em;
  line-height: 1.3em;
  overflow: hidden;
}
.roll-track {
  display: flex;
  flex-direction: column;
  transition: transform 0.55s var(--ease);
}
.roll-link:hover .roll-track {
  transform: translateY(-50%);
}

.roll-arrow {
  position: relative;
  display: inline-block;
  width: 16px;
  height: 16px;
  overflow: hidden;
}
.roll-arrow svg {
  position: absolute;
  inset: 0;
  width: 16px;
  height: 16px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.5;
  stroke-linecap: round;
  stroke-linejoin: round;
  transition: transform 0.55s var(--ease);
}
.roll-arrow svg:last-child {
  transform: translateX(-120%);
}
.roll-link:hover .roll-arrow svg:first-child {
  transform: translateX(120%);
}
.roll-link:hover .roll-arrow svg:last-child {
  transform: translateX(0);
}

@media (prefers-reduced-motion: reduce) {
  .roll-track,
  .roll-arrow svg {
    transition: none;
  }
}
</style>
