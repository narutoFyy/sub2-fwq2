<script setup lang="ts">
import '@fontsource-variable/geist'
import '@fontsource-variable/geist-mono'
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import BrandLogo from '@/components/brand/BrandLogo.vue'
import LocaleToggle from '@/components/brand/LocaleToggle.vue'
import RollLink from '@/components/brand/RollLink.vue'
import ThemeToggleIcon from '@/components/common/ThemeToggleIcon.vue'
import GlowDots from '@/components/visual/GlowDots.vue'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import PlazaPriceTable from './PlazaPriceTable.vue'
import { usePlazaCopy, discountEn } from './content'
import { usePlazaData } from './usePlazaData'
import { discountZh, formatRate, groupRate } from './pricing'

const route = useRoute()
const router = useRouter()
const app = useAppStore()
const auth = useAuthStore()
const { t, locale } = useI18n()
const copy = usePlazaCopy()
const { data, status, init } = usePlazaData()

const siteName = computed(() => t('home.landing.brandName'))
const isAuthenticated = computed(() => auth.isAuthenticated)
const dashboardPath = computed(() => (auth.isAdmin ? '/admin/dashboard' : '/dashboard'))
const loginPath = '/login?redirect=%2Fplaza'

const search = ref('')
const selected = ref<number | 'all'>('all')

const groups = computed(() =>
  [...(data.value?.groups ?? [])].sort((a, b) => groupRate(a) - groupRate(b) || a.name.localeCompare(b.name))
)

/** 当前筛选下要展示的分组：选中分组 + 模型名搜索（分组内只留命中的模型）。 */
const shown = computed(() => {
  const q = search.value.trim().toLowerCase()
  return groups.value
    .filter((g) => selected.value === 'all' || g.id === selected.value)
    .map((g) => ({ group: g, models: q ? g.models.filter((m) => m.name.toLowerCase().includes(q)) : g.models }))
    .filter((x) => x.models.length > 0)
})

const descriptionHtml = computed(() => {
  const md = data.value?.description?.trim()
  return md ? DOMPurify.sanitize(marked.parse(md) as string) : ''
})

function discount(g: ModelPlazaGroup): string {
  const rate = groupRate(g)
  if (rate === 1) return copy.value.sameAsOfficial
  if (rate > 1) return copy.value.premium
  return locale.value.startsWith('zh') ? copy.value.discount(discountZh(rate)) : discountEn(rate)
}

function notes(g: ModelPlazaGroup): string[] {
  const list: string[] = []
  if (hasPeakRate(g)) {
    const window = formatPeakRateWindow(g, serverTimezoneLabel(app.cachedPublicSettings?.server_utc_offset))
    list.push(copy.value.peak(window, formatRate(g.peak_rate_multiplier)))
  }
  const tiered = g.models.some((m) => (m.pricing?.intervals?.length ?? 0) > 1)
  if (tiered) list.push(copy.value.longContext)
  if (g.long_context_pricing_enabled === false && g.models.some((m) => (m.official_pricing?.intervals?.length ?? 0) > 1)) {
    list.push(copy.value.longContextOff)
  }
  if (g.models.some((m) => m.time_pricing?.periods?.length)) list.push(copy.value.timePricing)
  return list
}

function select(id: number | 'all') {
  selected.value = id
  void router.replace({ query: { ...route.query, group: id === 'all' ? undefined : String(id) } })
}

watch(
  () => route.query.group,
  (v) => {
    const id = Number(v)
    selected.value = Number.isFinite(id) && id > 0 ? id : 'all'
  },
  { immediate: true }
)

// 链接带的分组不存在（已删除或无权限）时回到全部。
watch(groups, (list) => {
  if (selected.value !== 'all' && list.length && !list.some((g) => g.id === selected.value)) select('all')
})

watch(status, async (s) => {
  if (s !== 'ready' || selected.value === 'all') return
  await nextTick()
  document.querySelector('.plaza-tabs [aria-pressed="true"]')?.scrollIntoView({ block: 'nearest', inline: 'center' })
})

onMounted(init)
</script>

<template>
  <div class="plaza-surface">
    <header class="plaza-header">
      <nav class="plaza-nav">
        <BrandLogo to="/home" :name="siteName" />
        <div class="plaza-nav-actions">
          <RouterLink to="/home" class="back-home">{{ copy.home }}</RouterLink>
          <LocaleToggle />
          <ThemeToggleIcon />
          <RollLink :to="isAuthenticated ? dashboardPath : loginPath" :label="isAuthenticated ? copy.dashboard : copy.login" class="ml-2" />
        </div>
      </nav>
    </header>

    <main class="plaza-main">
      <section class="plaza-hero">
        <div class="hero-glow" aria-hidden="true"><GlowDots :center-x="0.82" :center-y="0.4" :gain="0.46" :max-cells="9000" /></div>
        <h1>{{ copy.title }}</h1>
        <p>{{ copy.intro }}</p>
        <p v-if="!isAuthenticated && status === 'ready'" class="hint">{{ copy.anonymousHint }}</p>
      </section>

      <div v-if="status === 'loading'" class="state" role="status"><span class="spinner" aria-hidden="true"></span></div>
      <div v-else-if="status === 'auth'" class="state">
        <p>{{ copy.loginRequired }}</p>
        <RollLink :to="loginPath" :label="copy.login" size="lg" arrow />
      </div>
      <div v-else-if="status === 'error' || status === 'disabled'" class="state"><p>{{ status === 'error' ? copy.loadFailed : copy.empty }}</p></div>

      <template v-else>
        <div class="plaza-controls">
          <div class="plaza-tabs" role="group" :aria-label="copy.allGroups">
            <button type="button" :aria-pressed="selected === 'all'" @click="select('all')">{{ copy.allGroups }}</button>
            <button v-for="g in groups" :key="g.id" type="button" :aria-pressed="selected === g.id" @click="select(g.id)">
              {{ g.name }}<span>×{{ formatRate(groupRate(g)) }}</span>
            </button>
          </div>
          <label class="plaza-search">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" /></svg>
            <input v-model="search" type="search" :placeholder="copy.search" :aria-label="copy.search" />
          </label>
        </div>

        <p class="unit-line">{{ copy.unitToken }} · {{ copy.officialHint }} · {{ copy.sortHint }}</p>

        <details v-if="descriptionHtml" class="plaza-note">
          <summary>{{ copy.noteTitle }}</summary>
          <div class="plaza-note-body" v-html="descriptionHtml"></div>
        </details>

        <p v-if="!shown.length" class="state">{{ search.trim() ? copy.noResult : copy.empty }}</p>

        <section v-for="{ group: g, models } in shown" :key="g.id" class="plaza-group" :aria-labelledby="`plaza-group-${g.id}`">
          <header class="group-head">
            <div class="group-title">
              <h2 :id="`plaza-group-${g.id}`">{{ g.name }}</h2>
              <span v-if="g.is_exclusive" class="badge">{{ copy.exclusive }}</span>
              <span v-if="g.subscription_type === 'subscription'" class="badge">{{ copy.subscription }}</span>
            </div>
            <div class="group-rate">
              <b>×{{ formatRate(groupRate(g)) }}</b>
              <span>{{ discount(g) }}</span>
            </div>
          </header>
          <p class="group-formula">
            {{ copy.formula(formatRate(groupRate(g))) }}
            <template v-if="g.user_rate_multiplier != null && g.user_rate_multiplier !== g.rate_multiplier"> · {{ copy.customRate(formatRate(g.rate_multiplier)) }}</template>
          </p>
          <p v-if="g.description" class="group-desc">{{ g.description }}</p>
          <ul v-if="notes(g).length" class="group-notes">
            <li v-for="n in notes(g)" :key="n">{{ n }}</li>
          </ul>
          <PlazaPriceTable :group="g" :models="models" />
        </section>
      </template>
    </main>
  </div>
</template>

<style scoped>
.plaza-surface {
  --plaza-bg: #fafafa; --plaza-ink: #0a0a0b; --plaza-muted: #63636b; --plaza-line: #dddde2; --plaza-panel: #f2f2f4;
  --plaza-accent: #21646a;
  min-height: 100vh; overflow-x: clip; background: var(--plaza-bg); color: var(--plaza-ink);
  font-family: 'Geist Variable', 'PingFang SC', 'Microsoft YaHei', system-ui, sans-serif; -webkit-font-smoothing: antialiased;
}
:global(.dark .plaza-surface) {
  --plaza-bg: #0a0a0b; --plaza-ink: #ededef; --plaza-muted: #a0a4ad; --plaza-line: #2b2b30; --plaza-panel: #141417;
  --plaza-accent: #a3e4e8;
}
.plaza-surface :is(a, button, summary, input):focus-visible { outline: 2px solid var(--plaza-accent); outline-offset: 4px; border-radius: 6px; }
.plaza-header { border-bottom: 1px solid var(--plaza-line); }
.plaza-nav { max-width: 1280px; margin: auto; padding: 0 48px; min-height: 80px; display: flex; justify-content: space-between; align-items: center; gap: 16px; }
.plaza-nav-actions { display: flex; align-items: center; gap: 12px; }
.back-home { font-size: 13px; margin-right: 16px; color: var(--plaza-muted); }
.plaza-main { max-width: 1184px; margin: auto; padding: 0 48px 120px; }

.plaza-hero { position: relative; isolation: isolate; padding: 64px 0 40px; }
.hero-glow { position: absolute; inset: 0 -48px; z-index: -1; opacity: .9; pointer-events: none; mask-image: linear-gradient(to right, transparent 30%, #000 75%); }
.plaza-hero h1 { font-size: clamp(32px, 4.5vw, 52px); font-weight: 500; line-height: 1.15; letter-spacing: -.025em; max-width: 720px; }
.plaza-hero p { margin-top: 18px; font-size: 16px; line-height: 1.8; color: var(--plaza-muted); max-width: 640px; }
.plaza-hero .hint { margin-top: 8px; font-size: 13px; }

.state { min-height: 240px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 24px; color: var(--plaza-muted); text-align: center; }
.spinner { width: 28px; height: 28px; border-radius: 50%; border: 2px solid var(--plaza-line); border-top-color: var(--plaza-ink); animation: spin .8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.plaza-controls {
  position: sticky; top: 0; z-index: 5; display: flex; align-items: center; gap: 16px;
  padding: 14px 0; background: color-mix(in srgb, var(--plaza-bg) 92%, transparent); backdrop-filter: blur(8px);
  border-bottom: 1px solid var(--plaza-line);
}
.plaza-tabs { flex: 1; min-width: 0; display: flex; gap: 6px; overflow-x: auto; scrollbar-width: none; padding: 2px; }
.plaza-tabs::-webkit-scrollbar { display: none; }
.plaza-tabs button { flex-shrink: 0; display: inline-flex; align-items: baseline; gap: 6px; padding: 8px 14px; border-radius: 99px; font-size: 13px; color: var(--plaza-muted); background: var(--plaza-panel); transition: background-color .15s, color .15s; }
.plaza-tabs button span { font: 12px 'Geist Mono Variable', ui-monospace, monospace; opacity: .8; }
.plaza-tabs button:hover { color: var(--plaza-ink); }
.plaza-tabs button[aria-pressed='true'] { background: var(--plaza-ink); color: var(--plaza-bg); }
.plaza-search { flex-shrink: 0; display: flex; align-items: center; gap: 8px; width: 220px; padding: 8px 14px; border-radius: 99px; background: var(--plaza-panel); color: var(--plaza-muted); }
.plaza-search input { flex: 1; min-width: 0; background: transparent; border: 0; outline: none; font-size: 13px; color: var(--plaza-ink); padding: 0; box-shadow: none; }
.plaza-search input::placeholder { color: var(--plaza-muted); }

.unit-line { margin-top: 16px; font-size: 12px; color: var(--plaza-muted); }
.plaza-note { margin-top: 20px; padding: 16px 20px; border-radius: 12px; background: var(--plaza-panel); font-size: 14px; }
.plaza-note summary { cursor: pointer; font-weight: 500; }
.plaza-note-body { margin-top: 12px; line-height: 1.75; color: var(--plaza-muted); overflow-wrap: anywhere; }
.plaza-note-body :deep(a) { text-decoration: underline; text-underline-offset: 4px; color: var(--plaza-ink); }
.plaza-note-body :deep(ul) { list-style: disc; padding-left: 20px; }
.plaza-note-body :deep(ol) { list-style: decimal; padding-left: 20px; }
.plaza-note-body :deep(p + p) { margin-top: 8px; }

.plaza-group { padding-top: 56px; scroll-margin-top: 80px; }
.group-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; }
.group-title { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; min-width: 0; }
.group-title h2 { font-size: clamp(22px, 2.4vw, 28px); font-weight: 500; line-height: 1.3; overflow-wrap: anywhere; }
.badge { font-size: 12px; padding: 2px 10px; border-radius: 99px; background: var(--plaza-panel); color: var(--plaza-muted); }
.group-rate { flex-shrink: 0; display: flex; align-items: baseline; gap: 10px; }
.group-rate b { font: 600 28px 'Geist Mono Variable', ui-monospace, monospace; color: var(--plaza-accent); }
.group-rate span { font-size: 13px; color: var(--plaza-muted); }
.group-formula { margin-top: 8px; font-size: 13px; color: var(--plaza-muted); }
.group-desc { margin-top: 12px; font-size: 14px; line-height: 1.75; color: var(--plaza-muted); max-width: 760px; white-space: pre-line; }
.group-notes { margin-top: 12px; display: flex; flex-wrap: wrap; gap: 6px 20px; font-size: 12px; color: var(--plaza-muted); }
.group-notes li::before { content: '· '; }
.plaza-group :deep(.price-table) { margin-top: 20px; border-top: 1px solid var(--plaza-ink); }

@media (max-width: 760px) {
  .plaza-nav { padding: 0 16px; min-height: 64px; }
  .back-home { display: none; }
  .plaza-main { padding: 0 16px 80px; }
  .plaza-hero { padding: 40px 0 28px; }
  .hero-glow { inset: 0 -16px; }
  .plaza-controls { flex-direction: column; align-items: stretch; gap: 10px; }
  .plaza-search { width: auto; }
  .plaza-group { padding-top: 40px; }
  .group-head { flex-direction: column; align-items: flex-start; gap: 8px; }
  .group-rate b { font-size: 22px; }
}
@media (prefers-reduced-motion: reduce) { .spinner { animation-duration: 2.4s; } .plaza-tabs button { transition: none; } }
</style>
