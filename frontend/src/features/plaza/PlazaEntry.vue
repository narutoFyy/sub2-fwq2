<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { CountUp, vReveal, vSpotlight } from '@/features/motion'
import { usePlazaCopy, discountEn } from './content'
import { usePlazaData } from './usePlazaData'
import { discountZh, featuredModel, formatRate, groupRate, isTokenModel, perMillion, perUnit, unitPrices, billingMode } from './pricing'
import type { ModelPlazaGroup } from '@/api/modelPlaza'

/** 首页「广场」板块：按倍率列出分组与代表模型价格，完整价目去 /plaza。 */
const PREVIEW_LIMIT = 6

const copy = usePlazaCopy()
const { locale } = useI18n()
const { data, status, init } = usePlazaData()
onMounted(init)

const groups = computed(() =>
  [...(data.value?.groups ?? [])].sort((a, b) => groupRate(a) - groupRate(b) || a.name.localeCompare(b.name))
)
const preview = computed(() =>
  groups.value.slice(0, PREVIEW_LIMIT).map((g) => ({ group: g, icon: featuredModel(g)?.name || g.platform, price: featuredPrice(g), discount: discount(g) }))
)
const hiddenCount = computed(() => Math.max(0, groups.value.length - PREVIEW_LIMIT))
const visible = computed(() => status.value === 'auth' || (status.value === 'ready' && groups.value.length > 0))

function discount(g: ModelPlazaGroup): string {
  const rate = groupRate(g)
  return locale.value.startsWith('zh') ? discountZh(rate) : discountEn(rate)
}

function featuredPrice(g: ModelPlazaGroup) {
  const m = featuredModel(g)
  if (!m) return null
  const rate = groupRate(g)
  if (isTokenModel(m)) {
    return { model: m.name, kind: 'token' as const, input: perMillion(m.pricing?.input_price, rate), output: perMillion(m.pricing?.output_price, rate) }
  }
  const first = unitPrices(m)[0]
  return {
    model: m.name,
    kind: 'unit' as const,
    price: perUnit(first?.price, rate),
    unit: billingMode(m) === 'image' ? copy.value.perImage : copy.value.perRequest
  }
}
</script>

<template>
  <section v-if="visible" v-reveal class="plaza-entry" aria-labelledby="plaza-entry-title">
    <div data-reveal class="entry-heading">
      <h2 id="plaza-entry-title">{{ copy.entryTitle }}</h2>
      <p>{{ copy.entryDesc }}</p>
    </div>

    <RouterLink v-if="status === 'auth'" data-reveal :to="{ path: '/login', query: { redirect: '/plaza' } }" class="entry-more">
      <span>{{ copy.loginRequired }}</span>
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><path d="M5 12h14m-6-6 6 6-6 6" /></svg>
    </RouterLink>

    <template v-else>
      <div data-reveal class="entry-head-row" aria-hidden="true">
        <span>{{ copy.group }}</span>
        <span>{{ copy.featured }}</span>
        <span class="num">{{ copy.input }}</span>
        <span class="num">{{ copy.output }}</span>
        <span class="num">{{ copy.rateLabel }}</span>
      </div>
      <ul class="entry-rows">
        <li v-for="({ group: g, icon, price, discount: off }, i) in preview" :key="g.id" data-reveal>
          <RouterLink v-spotlight :to="{ path: '/plaza', query: { group: String(g.id) } }" class="entry-row">
            <span class="group-cell">
              <ModelIcon :model="icon" size="22px" />
              <span class="group-text">
                <strong>{{ g.name }}</strong>
                <small>{{ copy.modelsCount(g.models.length) }}</small>
              </span>
            </span>
            <template v-if="price">
              <span class="model-cell">{{ price.model }}</span>
              <template v-if="price.kind === 'token'">
                <span class="num price price-in"><em>{{ copy.input }}</em><CountUp :value="price.input" :delay="0.15 + i * 0.07" /></span>
                <span class="num price price-out"><em>{{ copy.output }}</em><CountUp :value="price.output" :delay="0.2 + i * 0.07" /></span>
              </template>
              <span v-else class="num price price-in span-2"><CountUp :value="price.price" :delay="0.15 + i * 0.07" /><em class="unit">{{ price.unit }}</em></span>
            </template>
            <span class="num rate-cell">
              <b><CountUp :value="`×${formatRate(groupRate(g))}`" :delay="0.1 + i * 0.07" /></b>
              <small v-if="off">{{ off }}</small>
            </span>
          </RouterLink>
        </li>
      </ul>
      <div data-reveal class="entry-foot">
        <span class="unit-note">{{ copy.unitToken }}</span>
        <RouterLink to="/plaza" class="entry-more">
          <span>{{ copy.viewAll }}<small v-if="hiddenCount">{{ copy.moreGroups(hiddenCount) }}</small></span>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><path d="M5 12h14m-6-6 6 6-6 6" /></svg>
        </RouterLink>
      </div>
    </template>
  </section>
</template>

<style scoped>
.plaza-entry { max-width: 1280px; margin: 0 auto; padding: 40px 48px 96px; color: #0a0a0b; }
:global(.dark .plaza-entry) { color: #ededef; }
.entry-heading { display: flex; align-items: baseline; justify-content: space-between; gap: 24px; margin-bottom: 36px; }
h2 { font-size: clamp(26px, 3vw, 38px); line-height: 1.25; font-weight: 500; }
.entry-heading p { color: #63636b; }
:global(.dark .plaza-entry .entry-heading p) { color: #a0a4ad; }

.entry-head-row, .entry-row {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(0, 1.2fr) 120px 120px 112px;
  gap: 20px;
  align-items: center;
}
.entry-head-row { padding: 0 0 12px; font-size: 12px; color: #8a8a93; border-bottom: 1px solid #dedee1; }
:global(.dark .plaza-entry .entry-head-row) { color: #7c7f88; border-color: #2b2b30; }
.num { text-align: right; font-variant-numeric: tabular-nums; }
.entry-row { padding: 18px 0; border-bottom: 1px solid #dedee1; transition: background-color .2s; --spot-color: #e11d48; --spot-strength: 10%; }
:global(.dark .plaza-entry .entry-row) { --spot-color: #22d3ee; --spot-strength: 15%; }
:global(.dark .plaza-entry .entry-row) { border-color: #2b2b30; }
.entry-row:hover { background: color-mix(in srgb, currentColor 3%, transparent); }
.entry-row:focus-visible { outline: 2px solid currentColor; outline-offset: 4px; border-radius: 4px; }
.group-cell { display: flex; align-items: center; gap: 14px; min-width: 0; }
.group-text { min-width: 0; }
.group-text strong { display: block; font-size: 17px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.group-text small, .rate-cell small, .entry-more small { display: block; margin-top: 4px; font-size: 12px; color: #63636b; }
:global(.dark .plaza-entry :is(.group-text small, .rate-cell small, .entry-more small)) { color: #a0a4ad; }
.model-cell { font: 13px 'Geist Mono Variable', ui-monospace, monospace; color: #63636b; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
:global(.dark .plaza-entry .model-cell) { color: #a0a4ad; }
.price { font: 500 15px 'Geist Mono Variable', ui-monospace, monospace; }
.price em { display: none; font-style: normal; }
.price .unit { display: inline; margin-left: 4px; font: 12px 'Geist Variable', system-ui, sans-serif; color: #63636b; }
.span-2 { grid-column: span 2; }
.rate-cell b { font: 600 16px 'Geist Mono Variable', ui-monospace, monospace; color: #21646a; }
:global(.dark .plaza-entry .rate-cell b) { color: #a3e4e8; }
/* 黑色品牌图标（OpenAI、Grok 等）在深色下反白 */
:global(.dark .plaza-entry .model-icon path[fill='#000000']) { fill: #ededef; }

.entry-foot { display: flex; align-items: center; justify-content: space-between; gap: 24px; margin-top: 8px; }
.unit-note { font-size: 12px; color: #8a8a93; }
.entry-more { display: inline-flex; align-items: center; gap: 16px; padding: 20px 0; font-size: 18px; font-weight: 500; }
.entry-more svg { transition: transform .2s; }
.entry-more:hover svg { transform: translateX(4px); }
.entry-more:focus-visible { outline: 2px solid currentColor; outline-offset: 6px; border-radius: 4px; }

@media (max-width: 860px) {
  .plaza-entry { padding: 24px 24px 64px; }
  .entry-heading { display: block; margin-bottom: 16px; }
  .entry-heading p { margin-top: 14px; }
  .entry-head-row { display: none; }
  .entry-row { grid-template-columns: minmax(0, 1fr) auto; grid-template-areas: 'group rate' 'model model' 'in out'; gap: 8px 16px; }
  .group-cell { grid-area: group; }
  .rate-cell { grid-area: rate; }
  .model-cell { grid-area: model; padding-left: 36px; }
  .price { text-align: left; }
  .price-in { grid-area: in; padding-left: 36px; }
  .price-out { grid-area: out; }
  .price em { display: inline; margin-right: 6px; font: 12px 'Geist Variable', system-ui, sans-serif; color: #63636b; }
  .span-2 { grid-column: 1 / -1; }
  .entry-foot { flex-direction: column-reverse; align-items: flex-start; gap: 0; }
}
@media (prefers-reduced-motion: reduce) { .entry-more svg, .entry-row { transition: none; } }
</style>
