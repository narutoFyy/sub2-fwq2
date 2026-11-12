<template>
  <div data-testid="landing-home" class="landing min-h-screen">
    <!-- ============ 首屏 ============ -->
    <section class="relative isolate flex min-h-[100svh] flex-col overflow-hidden">
      <div class="absolute inset-0 -z-10" aria-hidden="true">
        <GlowDots :center-x="0.64" :center-y="0.58" />
        <div class="hero-scrim absolute inset-0"></div>
        <div class="hero-fade absolute inset-x-0 bottom-0 h-[32%]"></div>
      </div>

      <header class="px-5 py-4 sm:px-8 lg:px-12">
        <nav class="mx-auto flex max-w-7xl items-center justify-between gap-3">
          <BrandLogo to="/home" :name="siteName" />
          <div class="flex shrink-0 items-center gap-1 sm:gap-2">
            <RollLink
              v-if="showModelPlazaEntry"
              to="/model-plaza"
              :label="t('nav.modelPlaza')"
              variant="plain"
              class="mr-2 hidden sm:inline-flex"
            />
            <LocaleToggle />
            <ThemeToggleIcon />
            <RollLink :to="isAuthenticated ? dashboardPath : '/login'" :label="isAuthenticated ? t('home.dashboard') : t('home.login')" class="ml-1" />
          </div>
        </nav>
      </header>

      <div class="mx-auto mt-auto w-full max-w-7xl px-5 pb-16 sm:px-8 sm:pb-24 lg:px-12">
        <h1 class="hero-title">
          {{ t('home.landing.hero.titleLine1') }}<br />{{ t('home.landing.hero.titleLine2') }}
        </h1>
        <p class="mt-6 text-lg text-muted sm:text-xl">{{ t('home.landing.hero.subtitle') }}</p>
        <div class="mt-10 flex flex-wrap items-center gap-6">
          <RollLink
            :to="isAuthenticated ? dashboardPath : '/login'"
            :label="isAuthenticated ? t('home.goToDashboard') : t('home.getStarted')"
            size="lg"
            arrow
          />
          <RollLink
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            :label="t('home.landing.hero.viewPricing')"
            size="lg"
            variant="plain"
            arrow
          />
        </div>
      </div>
    </section>

    <!-- ============ 模型 ============ -->
    <section class="mx-auto max-w-7xl px-5 py-20 sm:px-8 sm:py-28 lg:px-12">
      <ul class="flex flex-wrap items-center justify-center gap-x-16 gap-y-8">
        <li v-for="m in models" :key="m.label" class="model-item">
          <ModelIcon :model="m.model" size="28px" />
          <span>{{ m.label }}</span>
        </li>
      </ul>
    </section>

    <!-- ============ 接入 ============ -->
    <section class="mx-auto grid max-w-7xl items-center gap-12 px-5 py-20 sm:px-8 sm:py-28 lg:grid-cols-[0.8fr_1.2fr] lg:gap-20 lg:px-12">
      <h2 class="section-title">
        {{ t('home.landing.quickstart.titleLine1') }}<br />{{ t('home.landing.quickstart.titleLine2') }}
      </h2>
      <div class="code-card">
        <div class="flex items-center justify-between gap-3 px-4 pt-4">
          <div class="flex gap-1" role="tablist">
            <button
              v-for="tab in codeTabs"
              :key="tab.id"
              type="button"
              role="tab"
              :aria-selected="activeTab === tab.id"
              class="code-tab"
              :class="{ 'code-tab--on': activeTab === tab.id }"
              @click="activeTab = tab.id"
            >
              {{ tab.label }}
            </button>
          </div>
          <button type="button" class="code-copy" :aria-label="t('home.landing.quickstart.copy')" @click="copyCode">
            <Icon :name="copied ? 'check' : 'copy'" size="sm" />
          </button>
        </div>
        <Transition name="code-fade" mode="out-in">
          <pre :key="activeTab" class="overflow-x-auto px-6 pb-7 pt-5 text-[13px] leading-[1.8]"><code>{{ activeCode }}</code></pre>
        </Transition>
      </div>
    </section>

    <!-- ============ 优势 ============ -->
    <section class="mx-auto grid max-w-7xl gap-12 px-5 py-20 sm:grid-cols-3 sm:px-8 sm:py-28 lg:px-12">
      <div v-for="f in features" :key="f.title">
        <h3 class="feature-title">{{ f.title }}</h3>
        <p class="mt-3 text-muted">{{ f.desc }}</p>
      </div>
    </section>

    <!-- ============ 底部 ============ -->
    <section class="relative isolate overflow-hidden">
      <div class="absolute inset-0 -z-10" aria-hidden="true">
        <GlowDots :center-x="0.5" :center-y="0.62" :seed="47" :max-cells="16000" />
        <div class="cta-fade absolute inset-0"></div>
      </div>
      <div class="mx-auto flex max-w-7xl flex-col items-center px-5 py-32 text-center sm:px-8 sm:py-44">
        <h2 class="hero-title !text-[clamp(40px,6.4vw,80px)]">{{ t('home.landing.finalCta.title') }}</h2>
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
      <div class="flex items-center gap-7">
        <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="footer-link">{{ t('nav.modelPlaza') }}</router-link>
        <router-link to="/key-usage" class="footer-link">{{ t('home.landing.footer.keyUsage') }}</router-link>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="footer-link">{{ t('home.docs') }}</a>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import '@fontsource-variable/geist'
import '@fontsource-variable/geist-mono'
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import BrandLogo from '@/components/brand/BrandLogo.vue'
import LocaleToggle from '@/components/brand/LocaleToggle.vue'
import RollLink from '@/components/brand/RollLink.vue'
import ThemeToggleIcon from '@/components/common/ThemeToggleIcon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import GlowDots from '@/components/visual/GlowDots.vue'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const showModelPlazaEntry = computed(
  () =>
    isFeatureFlagEnabled(FeatureFlags.modelPlaza) &&
    (isAuthenticated.value || appStore.cachedPublicSettings?.model_plaza_require_auth !== true),
)
const origin = typeof window !== 'undefined' ? window.location.origin : 'https://your-domain'

const models = [
  { model: 'claude', label: 'Claude' },
  { model: 'gpt-5', label: 'GPT' },
  { model: 'gemini', label: 'Gemini' },
]

type CodeTab = 'claudeCode' | 'curl' | 'python'
const activeTab = ref<CodeTab>('claudeCode')
const codeTabs: { id: CodeTab; label: string }[] = [
  { id: 'claudeCode', label: 'Claude Code' },
  { id: 'curl', label: 'cURL' },
  { id: 'python', label: 'Python' },
]
const codeSamples = computed<Record<CodeTab, string>>(() => {
  const key = t('home.landing.quickstart.keyPlaceholder')
  return {
    claudeCode: [`export ANTHROPIC_BASE_URL=${origin}`, `export ANTHROPIC_AUTH_TOKEN=${key}`, 'claude'].join('\n'),
    curl: [
      `curl ${origin}/v1/messages \\`,
      `  -H "x-api-key: ${key}" \\`,
      '  -H "anthropic-version: 2023-06-01" \\',
      '  -H "content-type: application/json" \\',
      `  -d '{"model":"claude-sonnet-5-5","max_tokens":512,`,
      `       "messages":[{"role":"user","content":"Hello"}]}'`,
    ].join('\n'),
    python: [
      'from openai import OpenAI',
      '',
      `client = OpenAI(base_url="${origin}/v1", api_key="${key}")`,
      'resp = client.chat.completions.create(',
      '    model="gpt-5",',
      '    messages=[{"role": "user", "content": "Hello"}],',
      ')',
      'print(resp.choices[0].message.content)',
    ].join('\n'),
  }
})
const activeCode = computed(() => codeSamples.value[activeTab.value])

const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | undefined
async function copyCode() {
  try {
    await navigator.clipboard.writeText(activeCode.value)
    copied.value = true
    clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => (copied.value = false), 1600)
  } catch {
    /* 剪贴板不可用时静默失败，用户仍可手动选中复制 */
  }
}
onBeforeUnmount(() => clearTimeout(copiedTimer))

const features = computed(() => [
  { title: t('home.landing.features.stableTitle'), desc: t('home.landing.features.stableDesc') },
  { title: t('home.landing.features.usageTitle'), desc: t('home.landing.features.usageDesc') },
  { title: t('home.landing.features.keyTitle'), desc: t('home.landing.features.keyDesc') },
])
</script>

<style scoped>
/* 浅色 = 樱粉，深色 = 午夜霓虹，和点阵配色一一对应 */
.landing {
  --bg: #fffafb;
  --ink: #1c0a10;
  --muted: rgba(28, 10, 16, 0.56);
  --accent: #e11d48;
  --code-bg: #12070b;
  background: var(--bg);
  color: var(--ink);
  font-family: 'Geist Variable', 'PingFang SC', 'HarmonyOS Sans SC', 'MiSans', 'Microsoft YaHei UI', 'Microsoft YaHei', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  font-feature-settings: 'ss01', 'cv11';
}
.dark .landing {
  --bg: #07080c;
  --ink: #f1f5f9;
  --muted: rgba(241, 245, 249, 0.56);
  --accent: #22d3ee;
  --code-bg: #0e1119;
}

.text-muted {
  color: var(--muted);
}
.hero-scrim {
  background:
    radial-gradient(ellipse 70% 62% at 20% 70%, color-mix(in srgb, var(--bg) 92%, transparent), transparent 74%),
    linear-gradient(to right, color-mix(in srgb, var(--bg) 55%, transparent), transparent 58%);
}
.hero-fade {
  background: linear-gradient(to top, var(--bg) 6%, color-mix(in srgb, var(--bg) 55%, transparent) 50%, transparent);
}
.cta-fade {
  background:
    linear-gradient(to bottom, var(--bg), transparent 30%, transparent 70%, var(--bg)),
    radial-gradient(ellipse at center, color-mix(in srgb, var(--bg) 55%, transparent), transparent 70%);
}

.hero-title {
  font-size: clamp(52px, 8.6vw, 128px);
  line-height: 0.98;
  letter-spacing: -0.045em;
  font-weight: 600;
}
.section-title {
  font-size: clamp(36px, 4.6vw, 60px);
  line-height: 1.05;
  letter-spacing: -0.04em;
  font-weight: 600;
}
.feature-title {
  font-size: clamp(26px, 2.6vw, 34px);
  letter-spacing: -0.03em;
  font-weight: 600;
}

.model-item {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 20px;
  font-weight: 600;
  letter-spacing: -0.02em;
  opacity: 0.6;
  filter: grayscale(1);
  transition:
    opacity 0.4s,
    filter 0.4s;
}
.model-item:hover {
  opacity: 1;
  filter: grayscale(0);
}

/* 代码卡片两种主题下都用深色，代码更易读；不用边框，只靠底色区分 */
.code-card {
  overflow: hidden;
  border-radius: 20px;
  background: var(--code-bg);
  color: #e2e8f0;
}
.code-tab {
  height: 32px;
  padding: 0 14px;
  border-radius: 9999px;
  font-size: 13px;
  color: rgba(226, 232, 240, 0.5);
  transition:
    background 0.3s,
    color 0.3s;
}
.code-tab:hover {
  color: #e2e8f0;
}
.code-tab--on {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}
.code-copy {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: 9999px;
  color: rgba(226, 232, 240, 0.5);
  transition: color 0.3s;
}
.code-copy:hover {
  color: #fff;
}
pre code {
  font-family: 'Geist Mono Variable', ui-monospace, 'SF Mono', Menlo, monospace;
  white-space: pre;
}
.code-fade-enter-active,
.code-fade-leave-active {
  transition:
    opacity 0.25s ease,
    transform 0.25s ease;
}
.code-fade-enter-from {
  opacity: 0;
  transform: translateY(4px);
}
.code-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.footer-link {
  transition: color 0.2s;
}
.footer-link:hover {
  color: var(--ink);
}
</style>
