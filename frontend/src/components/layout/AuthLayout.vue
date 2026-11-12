<template>
  <div class="auth-shell relative isolate flex min-h-screen">
    <!-- 桌面端柔光：铺在左侧约 70% 宽度，向右渐隐到表单左边，左右无分界 -->
    <div class="art-glow pointer-events-none absolute inset-y-0 left-0 -z-10 hidden w-[72%] lg:block" aria-hidden="true">
      <GlowDots :center-x="0.38" :center-y="0.44" :max-cells="20000" :gain="0.74" />
      <div class="art-fade absolute inset-0"></div>
    </div>

    <!-- 左：品牌与标语（桌面端），跟随深浅色主题 -->
    <aside class="auth-art relative hidden flex-col justify-between p-12 lg:flex lg:w-[52%] xl:w-[56%]">
      <BrandLogo to="/home" :name="siteName" />
      <h2 class="art-title">{{ t('auth.layout.headlineLine1') }}<br />{{ t('auth.layout.headlineLine2') }}</h2>
    </aside>

    <!-- 右：表单区，跟随深浅色主题 -->
    <main class="auth-main relative flex min-h-screen min-w-0 flex-1 flex-col">
      <!-- 窄屏：顶部一条柔光代替左侧面板 -->
      <div class="pointer-events-none absolute inset-x-0 top-0 h-56 overflow-hidden lg:hidden" aria-hidden="true">
        <GlowDots :center-x="0.5" :center-y="0.45" :mobile-center-y="0.45" :max-cells="9000" :interactive="false" :gain="0.74" />
        <div class="band-fade absolute inset-0"></div>
      </div>

      <div class="relative z-10 flex items-center justify-end gap-1 px-5 py-4 sm:px-8">
        <LocaleToggle />
        <ThemeToggleIcon />
      </div>

      <div class="relative z-10 flex flex-1 items-center justify-center px-5 pb-16 pt-4 sm:px-8">
        <div class="w-full max-w-[360px]">
          <div class="mb-10 flex justify-center lg:hidden">
            <BrandLogo to="/home" :name="siteName" :size="34" />
          </div>
          <div class="auth-form">
            <slot />
          </div>
          <div class="auth-form mt-8 text-center text-sm">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import '@fontsource-variable/geist'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import BrandLogo from '@/components/brand/BrandLogo.vue'
import LocaleToggle from '@/components/brand/LocaleToggle.vue'
import GlowDots from '@/components/visual/GlowDots.vue'
import ThemeToggleIcon from '@/components/common/ThemeToggleIcon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-shell {
  --bg: #fafafa;
  --ink: #0a0a0b;
  --muted: #63636b;
  --accent: #e11d48;
  --field: rgba(10, 10, 11, 0.045);
  --field-focus: rgba(10, 10, 11, 0.07);
  --btn-bg: #0a0a0b;
  --btn-fg: #fafafa;
  --btn-bg-hover: #26262a;
  background: var(--bg);
  color: var(--ink);
  font-family: 'Geist Variable', 'PingFang SC', 'HarmonyOS Sans SC', 'MiSans', 'Microsoft YaHei UI', 'Microsoft YaHei', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
.dark .auth-shell {
  --bg: #0a0a0b;
  --ink: #ededef;
  --muted: #8a8f98;
  --accent: #22d3ee;
  --field: rgba(255, 255, 255, 0.05);
  --field-focus: rgba(255, 255, 255, 0.08);
  --btn-bg: #e6e6e8;
  --btn-fg: #0a0a0b;
  --btn-bg-hover: #ffffff;
}

.auth-art {
  color: var(--ink);
}
/* 点阵向右逐渐消失，过渡区跨过页面中线，到表单左边已完全淡出 */
.art-glow {
  -webkit-mask-image: linear-gradient(to right, #000 26%, rgba(0, 0, 0, 0.5) 50%, transparent 80%);
  mask-image: linear-gradient(to right, #000 26%, rgba(0, 0, 0, 0.5) 50%, transparent 80%);
}
.art-fade {
  background:
    radial-gradient(ellipse 55% 42% at 16% 84%, color-mix(in srgb, var(--bg) 92%, transparent), transparent 72%),
    linear-gradient(to bottom, color-mix(in srgb, var(--bg) 85%, transparent), transparent 22%, transparent 76%, var(--bg));
}
.art-title {
  font-size: clamp(36px, 3.4vw, 52px);
  line-height: 1.15;
  letter-spacing: 0;
  font-weight: 500;
}
.art-title:lang(en) {
  letter-spacing: -0.025em;
}

.band-fade {
  background: linear-gradient(to bottom, transparent 20%, var(--bg) 96%);
}

/* ---- 表单区：去边框、黑白主按钮、强调色链接 ---- */
.auth-form :deep(h2) {
  font-size: 24px;
  font-weight: 500;
  letter-spacing: 0;
}
.auth-form :deep(.input-label) {
  color: var(--muted);
  font-weight: 500;
}
.auth-form :deep(.input) {
  height: 42px;
  border-color: transparent;
  border-radius: 10px;
  background-color: var(--field);
  box-shadow: none;
  transition:
    background-color 0.25s,
    box-shadow 0.25s;
}
.auth-form :deep(.input:hover) {
  background-color: var(--field-focus);
}
.auth-form :deep(.input:focus) {
  border-color: transparent;
  background-color: var(--field-focus);
  --tw-ring-color: color-mix(in srgb, var(--accent) 35%, transparent);
}
.auth-form :deep(.input-error) {
  --tw-ring-color: rgba(239, 68, 68, 0.35);
  box-shadow: 0 0 0 2px rgba(239, 68, 68, 0.35);
}
.auth-form :deep(.btn-primary) {
  height: 42px;
  border-radius: 9999px;
  background-image: none;
  background-color: var(--btn-bg);
  color: var(--btn-fg);
  box-shadow: none;
}
.auth-form :deep(.btn-primary:hover:not(:disabled)) {
  background-image: none;
  background-color: var(--btn-bg-hover);
  box-shadow: none;
}
.auth-form :deep(.btn-primary:focus) {
  --tw-ring-color: color-mix(in srgb, var(--ink) 22%, transparent);
  --tw-ring-offset-width: 0px;
}
.auth-form :deep(.btn-secondary) {
  border-color: transparent;
  background-color: var(--field);
  box-shadow: none;
}
.auth-form :deep(.text-primary-600),
.auth-form :deep(.text-primary-500),
.dark .auth-form :deep(.dark\:text-primary-400) {
  color: var(--accent);
}
</style>
