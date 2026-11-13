<script setup lang="ts">
import { computed } from 'vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { platformLabel } from '@/utils/platformColors'
import type { ModelPlazaGroup, PlazaModel, PlazaTimePricingPeriod } from '@/api/modelPlaza'
import { usePlazaCopy } from './content'
import {
  billingMode,
  formatRate,
  isTokenModel,
  modelRate,
  officialTiers,
  paidTiers,
  perMillion,
  perUnit,
  sortModels,
  unitPrices,
  type TokenTier
} from './pricing'

const props = defineProps<{ group: ModelPlazaGroup; models: PlazaModel[] }>()
const copy = usePlazaCopy()

type TierKey = 'input' | 'output' | 'cacheRead' | 'cacheWrite'
const columns = computed<{ key: TierKey; label: string }[]>(() => [
  { key: 'input', label: copy.value.input },
  { key: 'output', label: copy.value.output },
  { key: 'cacheRead', label: copy.value.cacheRead },
  { key: 'cacheWrite', label: copy.value.cacheWrite }
])

const sorted = computed(() => sortModels(props.models))
const tokenModels = computed(() => sorted.value.filter(isTokenModel))
const unitModels = computed(() => sorted.value.filter((m) => !isTokenModel(m)))

interface Cell { label: string; paid: string; official: string }

/** 每个价格格子：按档位一行，实付价 + 不同于实付时的官方划线价。 */
function cells(m: PlazaModel, key: TierKey): Cell[] {
  const rate = modelRate(props.group, m)
  const paid = paidTiers(m)
  const official = officialTiers(m)
  return paid.map((tier: TokenTier, i) => {
    const p = perMillion(tier[key], rate)
    const offTier = official.length === paid.length ? official[i] : official.length === 1 && paid.length === 1 ? official[0] : undefined
    const o = offTier ? perMillion(offTier[key]) : '—'
    // 1h 缓存写入价与 5m 相同时不重复展示。
    const p1h = key === 'cacheWrite' && tier.cacheWrite1h != null ? perMillion(tier.cacheWrite1h, rate) : ''
    const extra1h = p1h && p1h !== p ? ` · 1h ${p1h}` : ''
    return { label: tier.label, paid: p + extra1h, official: o !== p && o !== '—' && p !== '—' ? o : '' }
  })
}

function hasValue(m: PlazaModel, key: TierKey): boolean {
  return paidTiers(m).some((t) => t[key] != null)
}

function reasoningTags(m: PlazaModel): string[] {
  const mult = m.pricing?.reasoning_effort_multipliers
  if (!mult) return []
  return Object.entries(mult)
    .filter(([, v]) => typeof v === 'number' && Number.isFinite(v) && v > 0 && v !== 1)
    .map(([effort, v]) => copy.value.reasoning(effort, v))
}

function clock(v: string) {
  return v.replace(/^(\d{2}:\d{2}):00$/, '$1')
}

function periodTag(p: PlazaTimePricingPeriod): string {
  return `${clock(p.start_time)}–${clock(p.end_time)} ×${formatRate(p.multiplier)}`
}

function unitSuffix(m: PlazaModel) {
  return billingMode(m) === 'image' ? copy.value.perImage : copy.value.perRequest
}

function unitRows(m: PlazaModel) {
  const rate = modelRate(props.group, m)
  return unitPrices(m).map((u) => ({ label: u.label, paid: perUnit(u.price, rate) }))
}

const showPlatform = (m: PlazaModel) => props.group.platform === 'composite' || m.platform !== props.group.platform
</script>

<template>
  <div class="price-table">
    <div v-if="tokenModels.length" class="pt-grid" role="table">
      <div class="pt-row pt-head" role="row">
        <span role="columnheader">{{ copy.model }}</span>
        <span v-for="c in columns" :key="c.key" role="columnheader" class="num">{{ c.label }}</span>
      </div>
      <div v-for="m in tokenModels" :key="`${m.platform}:${m.name}`" class="pt-row" role="row">
        <span class="pt-model" role="cell">
          <ModelIcon :model="m.name" size="18px" />
          <span class="pt-model-text">
            <span class="pt-name">{{ m.name }}</span>
            <span v-if="showPlatform(m) || reasoningTags(m).length || m.time_pricing?.periods?.length" class="pt-tags">
              <span v-if="showPlatform(m)" class="tag">{{ platformLabel(m.platform) }}</span>
              <span v-for="r in reasoningTags(m)" :key="r" class="tag">{{ r }}</span>
              <span v-for="(p, i) in m.time_pricing?.periods ?? []" :key="i" class="tag" :title="m.time_pricing?.timezone">
                {{ m.time_pricing?.weekdays_only ? `${copy.weekdays} ` : '' }}{{ periodTag(p) }}
              </span>
            </span>
          </span>
        </span>
        <span v-for="(c, ci) in columns" :key="c.key" class="pt-cell num" role="cell" :class="{ 'is-empty': !hasValue(m, c.key) }">
          <em class="pt-cell-label">{{ c.label }}</em>
          <span class="pt-values">
            <span v-for="(cell, i) in cells(m, c.key)" :key="i" class="pt-value">
              <b><small v-if="cell.label" class="tier" :class="{ 'tier-repeat': ci > 0 }">{{ cell.label }}</small>{{ cell.paid }}</b>
              <s v-if="cell.official" :title="copy.officialHint">{{ cell.official }}</s>
            </span>
          </span>
        </span>
      </div>
    </div>

    <div v-if="unitModels.length" class="pt-grid pt-unit" role="table">
      <div class="pt-row pt-head" role="row">
        <span role="columnheader">{{ copy.model }}</span>
        <span role="columnheader" class="num">{{ copy.perImageLabel }} / {{ copy.perRequestLabel }}</span>
      </div>
      <div v-for="m in unitModels" :key="`${m.platform}:${m.name}`" class="pt-row" role="row">
        <span class="pt-model" role="cell">
          <ModelIcon :model="m.name" size="18px" />
          <span class="pt-model-text">
            <span class="pt-name">{{ m.name }}</span>
            <span class="pt-tags"><span class="tag">{{ billingMode(m) === 'image' ? copy.perImageLabel : copy.perRequestLabel }}</span></span>
          </span>
        </span>
        <span class="pt-cell num" role="cell">
          <span class="pt-values">
            <span v-for="(u, i) in unitRows(m)" :key="i" class="pt-value">
              <small v-if="u.label" class="tier">{{ u.label }}</small>
              <b>{{ u.paid }}</b><small class="unit">{{ unitSuffix(m) }}</small>
            </span>
            <span v-if="!unitRows(m).length" class="pt-value"><b>—</b></span>
          </span>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.price-table { --pt-line: var(--plaza-line, #dddde2); --pt-muted: var(--plaza-muted, #63636b); }
.pt-grid + .pt-grid { margin-top: 28px; }
.pt-row {
  display: grid;
  grid-template-columns: minmax(200px, 1.5fr) repeat(3, minmax(110px, 1fr)) minmax(150px, 1.2fr);
  gap: 16px;
  align-items: center;
  padding: 14px 0;
  border-bottom: 1px solid var(--pt-line);
}
.pt-unit .pt-row { grid-template-columns: minmax(220px, 1.6fr) minmax(160px, 1fr); }
.pt-head { padding: 10px 0; font-size: 12px; color: var(--pt-muted); }
.pt-row:not(.pt-head):hover { background: color-mix(in srgb, currentColor 3%, transparent); }
.num { text-align: right; }
.pt-model { display: flex; align-items: flex-start; gap: 10px; min-width: 0; }
.pt-model :deep(.model-icon), .pt-model :deep(.model-icon-fallback) { margin-top: 2px; flex-shrink: 0; }
.pt-model-text { min-width: 0; }
.pt-name { display: block; font: 500 14px 'Geist Mono Variable', ui-monospace, monospace; overflow-wrap: anywhere; }
.pt-tags { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 6px; }
.tag { font-size: 11px; line-height: 18px; padding: 0 8px; border-radius: 99px; color: var(--pt-muted); background: color-mix(in srgb, currentColor 6%, transparent); white-space: nowrap; }
.pt-cell-label { display: none; font-style: normal; }
.pt-values { display: flex; flex-direction: column; gap: 6px; align-items: flex-end; }
.pt-value { display: flex; flex-direction: column; align-items: flex-end; font-variant-numeric: tabular-nums; }
.pt-value b { font: 600 15px 'Geist Mono Variable', ui-monospace, monospace; white-space: nowrap; }
.pt-value s { font: 12px 'Geist Mono Variable', ui-monospace, monospace; color: var(--pt-muted); opacity: .8; margin-top: 2px; }
.pt-value .tier { margin-right: 8px; font: 400 11px 'Geist Variable', system-ui, sans-serif; color: var(--pt-muted); }
/* 档位标签只在输入列显示一次，其余列按行对齐即可 */
.pt-value .tier-repeat { display: none; }
.pt-value .unit { font-size: 12px; color: var(--pt-muted); margin-left: 4px; }
.pt-unit .pt-value { flex-direction: row; align-items: baseline; gap: 6px; }
.is-empty b { color: var(--pt-muted); font-weight: 400; }
:global(.dark .price-table .model-icon path[fill='#000000']) { fill: #ededef; }

@media (max-width: 760px) {
  .pt-head { display: none; }
  .pt-row, .pt-unit .pt-row { grid-template-columns: 1fr 1fr; gap: 12px 16px; padding: 18px 0; }
  .pt-model { grid-column: 1 / -1; }
  .pt-cell { display: flex; flex-direction: column; align-items: flex-start; text-align: left; gap: 4px; }
  .pt-unit .pt-cell { grid-column: 1 / -1; }
  .pt-cell-label { display: block; font-size: 12px; color: var(--pt-muted); }
  .pt-values, .pt-value { align-items: flex-start; }
  .is-empty { display: none; }
  .pt-value .tier-repeat { display: inline; }
}
</style>
