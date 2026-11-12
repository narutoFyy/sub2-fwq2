<template>
  <button type="button" class="locale-toggle" :aria-label="t('home.switchLanguage')" :title="t('home.switchLanguage')" @click="toggle">
    <span class="locale-roll" aria-hidden="true">
      <Transition :name="direction">
        <span :key="current" class="locale-label" :class="{ 'locale-label--cjk': current === 'zh' }">{{ labelOf(current) }}</span>
      </Transition>
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { availableLocales, setLocale } from '@/i18n'

const { t, locale } = useI18n()
const current = computed(() => locale.value)
const direction = ref<'roll-up' | 'roll-down'>('roll-up')
let busy = false

const LABELS: Record<string, string> = { zh: '中', en: 'EN' }
const labelOf = (code: string) => LABELS[code] ?? code.toUpperCase()

async function toggle() {
  if (busy) return
  const codes = availableLocales.map((l) => l.code)
  const idx = codes.indexOf(current.value as (typeof codes)[number])
  const next = codes[(idx + 1) % codes.length]
  direction.value = idx % 2 === 0 ? 'roll-up' : 'roll-down'
  busy = true
  const doc = document as Document & {
    startViewTransition?: (cb: () => Promise<void>) => { finished: Promise<void> }
  }
  const reduceMotion =
    typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  try {
    if (doc.startViewTransition && !reduceMotion) {
      // 整页文案柔和地交叉淡入
      document.documentElement.classList.add('locale-vt')
      const vt = doc.startViewTransition(() => setLocale(next))
      await vt.finished.catch(() => {})
      document.documentElement.classList.remove('locale-vt')
    } else {
      await setLocale(next)
    }
  } finally {
    busy = false
  }
}
</script>

<style scoped>
.locale-toggle {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  padding: 0;
  border: 0;
  border-radius: 9999px;
  background: none;
  cursor: pointer;
  color: var(--muted, inherit);
  transition:
    color 0.2s,
    box-shadow 0.2s;
  -webkit-tap-highlight-color: transparent;
}
.locale-toggle:hover {
  color: var(--ink, inherit);
}
.locale-toggle:focus {
  outline: none;
}
.locale-toggle:focus-visible {
  box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 22%, transparent);
}
.locale-roll {
  position: relative;
  display: grid;
  place-items: center;
  width: 28px;
  height: 22px;
  overflow: hidden;
}
.locale-label {
  grid-area: 1 / 1;
  font-family: 'Geist Variable', 'PingFang SC', 'HarmonyOS Sans SC', 'MiSans', 'Microsoft YaHei UI', sans-serif;
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.02em;
  line-height: 1;
}
.locale-label--cjk {
  font-size: 14px;
  letter-spacing: 0;
}

/* 文字翻牌：旧的向上滚出并模糊，新的从下方滚入（反向切换时方向相反） */
.roll-up-enter-active,
.roll-up-leave-active,
.roll-down-enter-active,
.roll-down-leave-active {
  transition:
    transform 0.55s cubic-bezier(0.65, 0, 0.25, 1),
    opacity 0.4s ease,
    filter 0.4s ease;
}
.roll-up-enter-from,
.roll-down-leave-to {
  transform: translateY(110%);
  opacity: 0;
  filter: blur(3px);
}
.roll-up-leave-to,
.roll-down-enter-from {
  transform: translateY(-110%);
  opacity: 0;
  filter: blur(3px);
}
@media (prefers-reduced-motion: reduce) {
  .roll-up-enter-active,
  .roll-up-leave-active,
  .roll-down-enter-active,
  .roll-down-leave-active {
    transition: none;
  }
}
</style>

<style>
html.locale-vt::view-transition-old(root),
html.locale-vt::view-transition-new(root) {
  animation-duration: 0.35s;
  animation-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
}
</style>
