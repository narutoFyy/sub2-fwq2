<template>
  <div data-testid="landing-home" class="landing min-h-screen">
    <!-- ============ 首屏 ============ -->
    <section class="relative isolate flex min-h-[100svh] flex-col overflow-hidden">
      <div class="absolute inset-0 -z-10" aria-hidden="true">
        <GlowDots :center-x="0.76" :center-y="0.5" :cell="4" :max-cells="55000" :gain="0.84" reveal-selector="[data-reveal-title]" @reveal="titleShown = true" />
        <div class="hero-scrim absolute inset-0"></div>
        <div class="hero-fade absolute inset-x-0 bottom-0 h-[32%]"></div>
      </div>

      <header class="hero-gutter-nav">
        <nav class="flex h-16 items-center justify-between gap-3">
          <BrandLogo to="/home" :name="siteName" />
          <div class="flex shrink-0 items-center gap-1">
            <span class="hidden sm:inline-flex"><RollLink to="/tutorials" :label="tutorialCopy.nav" variant="plain" class="mr-1" /></span>
            <RollLink
              v-if="showModelPlazaEntry"
              to="/plaza"
              :label="t('nav.modelPlaza')"
              variant="plain"
              class="mr-1 hidden sm:inline-flex"
            />
            <LocaleToggle />
            <ThemeToggleIcon />
            <RollLink :to="isAuthenticated ? dashboardPath : '/login'" :label="isAuthenticated ? t('home.dashboard') : t('home.login')" class="ml-2" />
          </div>
        </nav>
      </header>

      <div class="hero-gutter my-auto w-full">
        <h1 class="hero-title hero-title--stack" :class="{ 'hero-title--hidden': !titleShown }" data-reveal-title>
          <span class="block" data-reveal-line>{{ t('home.landing.hero.titleLine1') }}</span>
          <span class="block" data-reveal-line>{{ t('home.landing.hero.titleLine2') }}</span>
        </h1>
        <p class="hero-sub">{{ t('home.landing.hero.subtitle') }}</p>
        <div class="mt-9 flex flex-wrap items-center gap-4">
          <span class="sm:hidden"><RollLink to="/tutorials" :label="tutorialCopy.nav" size="lg" variant="plain" arrow /></span>
          <RollLink
            :to="isAuthenticated ? dashboardPath : '/login'"
            :label="isAuthenticated ? t('home.goToDashboard') : t('home.getStarted')"
            size="lg"
            arrow
          />
          <RollLink
            v-if="showModelPlazaEntry"
            to="/plaza"
            :label="t('home.landing.hero.viewPricing')"
            size="lg"
            variant="plain"
            arrow
          />
        </div>
      </div>
    </section>

    <TutorialEntry />
    <PlazaEntry v-if="showModelPlazaEntry" />

    <!-- ============ 底部 ============ -->
    <section class="relative isolate overflow-hidden">
      <div class="absolute inset-0 -z-10" aria-hidden="true">
        <GlowDots :center-x="0.5" :center-y="0.62" :seed="47" :max-cells="16000" :gain="0.74" />
        <div class="cta-fade absolute inset-0"></div>
      </div>
      <div class="mx-auto flex max-w-7xl flex-col items-center px-5 py-32 text-center sm:px-8 sm:py-44">
        <h2 class="hero-title !text-[clamp(36px,4.8vw,64px)]">{{ t('home.landing.finalCta.title') }}</h2>
        <RollLink
          :to="isAuthenticated ? dashboardPath : '/login'"
          :label="isAuthenticated ? t('home.goToDashboard') : t('home.getStarted')"
          size="lg"
          arrow
          class="mt-10"
        />
      </div>
    </section>

    <footer class="mx-auto flex max-w-7xl flex-col items-center justify-between gap-4 px-5 py-10 text-sm text-muted sm:flex-row sm:px-8 lg:px-12">
      <BrandLogo :name="siteName" :size="22" />
      <div class="flex flex-wrap items-center justify-center gap-x-7 gap-y-3">
        <router-link to="/tutorials" class="footer-link">{{ tutorialCopy.nav }}</router-link>
        <router-link v-if="showModelPlazaEntry" to="/plaza" class="footer-link">{{ t('nav.modelPlaza') }}</router-link>
        <router-link to="/key-usage" class="footer-link">{{ t('home.landing.footer.keyUsage') }}</router-link>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="footer-link">{{ t('home.docs') }}</a>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import '@fontsource-variable/geist'
import '@fontsource-variable/geist-mono'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import BrandLogo from '@/components/brand/BrandLogo.vue'
import LocaleToggle from '@/components/brand/LocaleToggle.vue'
import RollLink from '@/components/brand/RollLink.vue'
import ThemeToggleIcon from '@/components/common/ThemeToggleIcon.vue'
import GlowDots from '@/components/visual/GlowDots.vue'
import { hasRevealPlayed } from '@/utils/glowDots'
import TutorialEntry from '@/features/tutorials/TutorialEntry.vue'
import PlazaEntry from '@/features/plaza/PlazaEntry.vue'
import { useTutorialCopy } from '@/features/tutorials/content'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const tutorialCopy = useTutorialCopy()

// 品牌名跟随界面语言：中文「石头中转站」，英文 yf-mail
const siteName = computed(() => t('home.landing.brandName'))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
// 标题从点阵里浮现：动画期间真实文字保持透明，交接后淡入；减少动态效果 / 非浏览器环境 / 已播放过则直接显示
const canReveal =
  typeof window !== 'undefined' &&
  !/jsdom/i.test(navigator.userAgent) &&
  !(typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches) &&
  !hasRevealPlayed()
const titleShown = ref(!canReveal)
let revealGuard: ReturnType<typeof setTimeout> | undefined
onMounted(() => {
  // 兜底：任何原因没收到交接信号，6 秒后也要把标题显示出来
  revealGuard = setTimeout(() => (titleShown.value = true), 6000)
})
onBeforeUnmount(() => clearTimeout(revealGuard))

// 强制登录时广场页自己提示登录，入口照常展示。
const showModelPlazaEntry = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza))
</script>

<style scoped>
/* 中性底色；强调色只出现在点阵最亮的核心（浅色樱粉、深色青） */
.landing {
  --bg: #fafafa;
  --ink: #0a0a0b;
  --muted: #63636b;
  --accent: #e11d48;
  --btn-bg: #0a0a0b;
  --btn-fg: #fafafa;
  --btn-bg-hover: #26262a;
  background: var(--bg);
  color: var(--ink);
  font-family: 'Geist Variable', 'PingFang SC', 'HarmonyOS Sans SC', 'MiSans', 'Microsoft YaHei UI', 'Microsoft YaHei', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  font-feature-settings: 'ss01', 'cv11';
}
.dark .landing {
  --bg: #0a0a0b;
  --ink: #ededef;
  --muted: #8a8f98;
  --accent: #22d3ee;
  --btn-bg: #e6e6e8;
  --btn-fg: #0a0a0b;
  --btn-bg-hover: #ffffff;
}

.text-muted {
  color: var(--muted);
}
.hero-scrim {
  background:
    linear-gradient(to right, color-mix(in srgb, var(--bg) 22%, transparent), transparent 50%),
    linear-gradient(to bottom, color-mix(in srgb, var(--bg) 60%, transparent), transparent 22%);
}
.hero-fade {
  background: linear-gradient(to top, var(--bg) 6%, color-mix(in srgb, var(--bg) 55%, transparent) 50%, transparent);
}
.cta-fade {
  background:
    linear-gradient(to bottom, var(--bg), transparent 30%, transparent 70%, var(--bg)),
    radial-gradient(ellipse at center, color-mix(in srgb, var(--bg) 55%, transparent), transparent 70%);
}

/* 中文字距保持 0（方块字一收紧就挤）；英文略收 */
.hero-title {
  font-size: clamp(40px, 12.4vw, 168px);
  line-height: 1.05;
  letter-spacing: 0;
  font-weight: 600;
  white-space: nowrap;
}
.hero-title {
  transition: opacity 0.5s ease;
}
.hero-title--hidden {
  opacity: 0;
}
/* 首屏不做居中限宽：Logo 与大标题都按视口比例贴向左侧，导航右侧控件对称 */
.hero-gutter-nav {
  padding-inline: clamp(20px, 5.6vw, 140px);
}
.hero-gutter {
  padding-inline: clamp(20px, 8.7vw, 220px);
  /* 上多下少，把整块内容压到画面偏下；按屏高比例取值，矮窗口里按钮也不会被挤出首屏 */
  padding-top: max(0px, min(31.3svh, calc(100svh - 17vw - 244px)));
  padding-bottom: 32px;
}
/* 首屏两行大标题：上方是站名小标，下方是一行说明 */
.hero-title--stack {
  font-size: clamp(40px, 8.2vw, 160px);
  line-height: 1.04;
  font-weight: 700;
}
.hero-sub {
  margin-top: 1rem;
  max-width: 36rem;
  font-size: 17px;
  line-height: 1.625;
  color: var(--muted);
}
.hero-title:lang(en) {
  letter-spacing: -0.04em;
}

.footer-link {
  transition: color 0.2s;
}
.footer-link:hover {
  color: var(--ink);
}
</style>
