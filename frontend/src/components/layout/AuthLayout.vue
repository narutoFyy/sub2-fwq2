<template>
  <div class="auth-shell flex min-h-screen">
    <!-- 左：柔光面板（桌面端），始终用深色的午夜霓虹 -->
    <aside class="auth-art relative isolate hidden flex-col justify-between overflow-hidden p-12 lg:flex lg:w-[52%] xl:w-[56%]">
      <div class="absolute inset-0 -z-10" aria-hidden="true">
        <GlowDots theme="midnight" :center-x="0.62" :center-y="0.4" :max-cells="18000" />
        <div class="art-fade absolute inset-0"></div>
      </div>
      <BrandLogo to="/home" :name="siteName" />
      <h2 class="art-title">{{ t('auth.layout.headlineLine1') }}<br />{{ t('auth.layout.headlineLine2') }}</h2>
    </aside>

    <!-- 右：表单区，跟随深浅色主题 -->
    <main class="auth-main relative flex min-h-screen min-w-0 flex-1 flex-col">
      <!-- 窄屏：顶部一条柔光代替左侧面板 -->
      <div class="pointer-events-none absolute inset-x-0 top-0 h-56 overflow-hidden lg:hidden" aria-hidden="true">
        <GlowDots :center-x="0.5" :center-y="0.45" :mobile-center-y="0.45" :max-cells="9000" :interactive="false" />
        <div class="band-fade absolute inset-0"></div>
      </div>

      <div class="relative z-10 flex items-center justify-end gap-1 px-5 py-4 sm:px-8">
        <LocaleToggle />
        <ThemeToggleIcon />
      </div>

      <div class="relative z-10 flex flex-1 items-center justify-center px-5 pb-16 pt-4 sm:px-8">
        <div class="w-full max-w-[380px]">
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
  --accent: #e11d48;
  --field: rgba(28, 10, 16, 0.045);
  --field-focus: rgba(28, 10, 16, 0.07);
  background: #fffafb;
  color: #1c0a10;
  font-family: 'Geist Variable', 'PingFang SC', 'HarmonyOS Sans SC', 'MiSans', 'Microsoft YaHei UI', 'Microsoft YaHei', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
.dark .auth-shell {
  --accent: #22d3ee;
  --field: rgba(255, 255, 255, 0.05);
  --field-focus: rgba(255, 255, 255, 0.08);
  background: #07080c;
  color: #f1f5f9;
}

.auth-art {
  --accent: #22d3ee;
  background: #07080c;
  color: #f1f5f9;
}
.art-fade {
  background:
    radial-gradient(ellipse 75% 55% at 25% 78%, rgba(7, 8, 12, 0.85), transparent 72%),
    linear-gradient(to bottom, rgba(7, 8, 12, 0.85), transparent 24%, transparent 70%, rgba(7, 8, 12, 0.92)),
    linear-gradient(to right, transparent 72%, rgba(7, 8, 12, 0.6));
}
.art-title {
  font-size: clamp(44px, 4.6vw, 72px);
  line-height: 1;
  letter-spacing: -0.045em;
  font-weight: 600;
}

.band-fade {
  background: linear-gradient(to bottom, transparent 20%, #fffafb 96%);
}
.dark .band-fade {
  background: linear-gradient(to bottom, transparent 20%, #07080c 96%);
}

/* ---- 表单区：去边框、黑白主按钮、强调色链接 ---- */
.auth-form :deep(h2) {
  font-size: 30px;
  font-weight: 600;
  letter-spacing: -0.03em;
}
.auth-form :deep(.input) {
  height: 48px;
  border-color: transparent;
  border-radius: 14px;
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
  height: 48px;
  border-radius: 9999px;
  background-image: none;
  background-color: #1c0a10;
  color: #fffafb;
  box-shadow: none;
  letter-spacing: 0.02em;
}
.auth-form :deep(.btn-primary:hover:not(:disabled)) {
  background-image: none;
  background-color: #3a1520;
  box-shadow: none;
}
.dark .auth-form :deep(.btn-primary) {
  background-color: #f1f5f9;
  color: #07080c;
}
.dark .auth-form :deep(.btn-primary:hover:not(:disabled)) {
  background-color: #ffffff;
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
