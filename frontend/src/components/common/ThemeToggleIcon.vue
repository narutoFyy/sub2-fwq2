<template>
  <button
    type="button"
    class="theme-toggle-icon"
    :class="{ 'is-dark': isDark, 'no-anim': !animReady }"
    :title="label"
    :aria-label="label"
    @click="toggle"
  >
    <!-- 同一个 SVG 在太阳与月亮之间变形：深色显示月亮，浅色显示太阳 -->
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <mask :id="maskId" maskUnits="userSpaceOnUse" x="-6" y="-6" width="36" height="36">
        <rect x="-6" y="-6" width="36" height="36" fill="#fff" />
        <circle class="cut" cx="17.5" cy="6.5" r="6.8" fill="#000" />
      </mask>
      <g :mask="`url(#${maskId})`">
        <circle class="core" cx="12" cy="12" r="8.6" fill="currentColor" />
      </g>
      <g class="rays" stroke="currentColor" stroke-width="2" stroke-linecap="round">
        <line
          v-for="(ray, i) in rays"
          :key="i"
          :style="{ '--i': i }"
          :x1="ray.x1"
          :y1="ray.y1"
          :x2="ray.x2"
          :y2="ray.y2"
        />
      </g>
    </svg>
  </button>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const isDark = ref(typeof document !== 'undefined' && document.documentElement.classList.contains('dark'))
const animReady = ref(false)
const maskId = `theme-moon-cut-${Math.random().toString(36).slice(2, 8)}`
let busy = false
let observer: MutationObserver | null = null

const label = computed(() => (isDark.value ? t('home.switchToLight') : t('home.switchToDark')))

/** 8 根光芒的端点（半径 8.4 → 10.6），直接算坐标，避免嵌套旋转和 CSS 变换叠加 */
const rays = Array.from({ length: 8 }, (_, i) => {
  const a = (i * Math.PI) / 4
  const pt = (r: number) => [12 + r * Math.sin(a), 12 - r * Math.cos(a)].map((v) => +v.toFixed(2))
  const [x1, y1] = pt(10.6)
  const [x2, y2] = pt(8.4)
  return { x1, y1, x2, y2 }
})

function applyTheme(dark: boolean) {
  document.documentElement.classList.toggle('dark', dark)
  isDark.value = dark
  try {
    localStorage.setItem('theme', dark ? 'dark' : 'light')
  } catch {
    /* 隐私模式等场景下写入失败不影响切换 */
  }
}

function toggle(event: MouseEvent) {
  if (busy) return
  const next = !document.documentElement.classList.contains('dark')
  const reduceMotion =
    typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const doc = document as Document & {
    startViewTransition?: (cb: () => Promise<void> | void) => {
      ready: Promise<void>
      finished: Promise<void>
    }
  }
  if (!doc.startViewTransition || reduceMotion) {
    applyTheme(next)
    return
  }
  // 新主题从图标位置以圆形扩散铺满整页
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const x = rect.left + rect.width / 2
  const y = rect.top + rect.height / 2
  const end = Math.hypot(Math.max(x, window.innerWidth - x), Math.max(y, window.innerHeight - y))
  const root = document.documentElement
  busy = true
  root.classList.add('theme-vt')
  const vt = doc.startViewTransition(async () => {
    applyTheme(next)
    await nextTick()
  })
  vt.ready
    .then(() => {
      root.animate(
        { clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${end}px at ${x}px ${y}px)`] },
        { duration: 750, easing: 'cubic-bezier(.65, 0, .25, 1)', pseudoElement: '::view-transition-new(root)' },
      )
    })
    .catch(() => {})
  vt.finished.finally(() => {
    busy = false
    root.classList.remove('theme-vt')
  })
}

onMounted(() => {
  // 其他地方切换主题时同步图标
  if (typeof MutationObserver !== 'undefined') {
    observer = new MutationObserver(() => {
      isDark.value = document.documentElement.classList.contains('dark')
    })
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  }
  // 首帧不播放变形动画
  if (typeof requestAnimationFrame === 'function') {
    requestAnimationFrame(() => requestAnimationFrame(() => (animReady.value = true)))
  } else {
    animReady.value = true
  }
})

onBeforeUnmount(() => observer?.disconnect())
</script>

<style scoped>
.theme-toggle-icon {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  padding: 0;
  border: 0;
  border-radius: 9999px;
  background: none;
  color: inherit;
  cursor: pointer;
  opacity: 0.82;
  transition: opacity 0.2s;
  -webkit-tap-highlight-color: transparent;
}
.theme-toggle-icon:hover {
  opacity: 1;
}
.theme-toggle-icon:focus-visible {
  outline: 2px solid currentColor;
  outline-offset: 2px;
}
.theme-toggle-icon svg {
  width: 22px;
  height: 22px;
  overflow: visible;
  transform-origin: 50% 50%;
  transform: rotate(180deg);
  transition: transform 0.7s cubic-bezier(0.65, 0, 0.25, 1);
}
.theme-toggle-icon svg * {
  transform-box: view-box;
  transform-origin: 12px 12px;
}

/* 浅色 = 太阳：月牙缺口移走，圆心缩小，光芒依次弹出 */
.core {
  transform: scale(0.56);
  transition: transform 0.6s cubic-bezier(0.34, 1.45, 0.64, 1);
}
.cut {
  transform: translate(8px, -8px);
  transition: transform 0.55s cubic-bezier(0.65, 0, 0.25, 1);
}
.rays line {
  opacity: 1;
  transform: rotate(0deg) scale(1);
  transition:
    transform 0.5s cubic-bezier(0.34, 1.56, 0.64, 1),
    opacity 0.3s ease;
  transition-delay: calc(0.16s + var(--i) * 0.035s);
}

/* 深色 = 月亮 */
.is-dark svg {
  transform: rotate(0deg);
}
.is-dark .core {
  transform: scale(1);
}
.is-dark .cut {
  transform: translate(0, 0);
}
.is-dark .rays line {
  opacity: 0;
  transform: rotate(-60deg) scale(0.3);
  transition:
    transform 0.35s cubic-bezier(0.4, 0, 1, 1),
    opacity 0.2s ease;
  transition-delay: 0s;
}

.no-anim svg,
.no-anim svg * {
  transition: none !important;
}
@media (prefers-reduced-motion: reduce) {
  .theme-toggle-icon svg,
  .theme-toggle-icon svg * {
    transition-duration: 0.01s !important;
    transition-delay: 0s !important;
  }
}
</style>

<style>
/* 整页圆形扩散切换（View Transitions），需要全局作用域 */
html.theme-vt::view-transition-old(root),
html.theme-vt::view-transition-new(root) {
  animation: none;
  mix-blend-mode: normal;
}
html.theme-vt,
html.theme-vt body {
  transition: none !important;
}
</style>
