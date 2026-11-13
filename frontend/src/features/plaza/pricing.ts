/**
 * 广场价格口径（与计费一致）：
 * - token 模型：实付单价 = 展示单价 × 生效倍率（用户专属倍率 ?? 分组倍率），按 $/1M tokens 展示；
 * - 图片 / 视频模型开启独立倍率时改乘独立倍率，按张 / 按次展示；
 * - 官方价是参考价，不乘任何倍率。
 */
import { resolveIntervalPrices } from '@/utils/pricing'
import type { UserPricingInterval } from '@/api/channels'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'

export const PER_MILLION = 1_000_000

export type PlazaBillingMode = 'token' | 'per_request' | 'image' | 'video'

export function billingMode(m: PlazaModel): PlazaBillingMode {
  return (m.pricing?.billing_mode || 'token') as PlazaBillingMode
}

export function isTokenModel(m: PlazaModel): boolean {
  return billingMode(m) === 'token'
}

/** 生效倍率 = 用户专属倍率 ?? 分组默认倍率。 */
export function groupRate(g: ModelPlazaGroup): number {
  return g.user_rate_multiplier ?? g.rate_multiplier
}

/** 单个模型实付所乘的倍率（图片 / 视频独立倍率会覆盖分组倍率）。 */
export function modelRate(g: ModelPlazaGroup, m: PlazaModel): number {
  const mode = billingMode(m)
  if (mode === 'image' && g.image_rate_independent) return g.image_rate_multiplier ?? 1
  if (mode === 'video' && g.video_rate_independent) return Math.max(0, g.video_rate_multiplier ?? 1)
  return groupRate(g)
}

/**
 * 美元金额：至少 2 位小数，最多保留 4 位有效小数并去掉末尾多余的 0，
 * 避免浮点噪声（0.30000000004）和过长的尾数。
 */
export function formatUsd(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return '—'
  if (value === 0) return '$0.00'
  const abs = Math.abs(value)
  // 小于 0.01 时按有效数字保留，保证 $0.0035 这类价格不被截成 $0.00。
  const digits = abs >= 1 ? 2 : Math.min(8, Math.max(2, Math.ceil(-Math.log10(abs)) + 2))
  let s = value.toFixed(digits)
  if (s.includes('.')) {
    s = s.replace(/0+$/, '')
    const decimals = s.length - s.indexOf('.') - 1
    if (decimals < 2) s += '0'.repeat(2 - decimals)
  }
  return `$${s}`
}

/** 每 token 单价 × 倍率 → 每百万 token 的美元价。 */
export function perMillion(price: number | null | undefined, rate = 1): string {
  if (price == null) return '—'
  return formatUsd(price * rate * PER_MILLION)
}

/** 按张 / 按次单价 × 倍率。 */
export function perUnit(price: number | null | undefined, rate = 1): string {
  if (price == null) return '—'
  return formatUsd(price * rate)
}

export interface TokenTier {
  label: string
  input: number | null
  output: number | null
  cacheRead: number | null
  cacheWrite: number | null
  cacheWrite1h: number | null
}

function sortByContext(intervals: UserPricingInterval[]): UserPricingInterval[] {
  return [...intervals].sort((a, b) => a.min_tokens - b.min_tokens)
}

function formatTokenCount(n: number): string {
  const trim = (v: number) => String(Math.round(v * 100) / 100)
  if (n >= 1_000_000) return `${trim(n / 1_000_000)}M`
  if (n >= 1_000) return `${trim(n / 1_000)}K`
  return String(n)
}

/** 档位标签：优先 tier_label，否则有上限为「≤上限」，末档为「>下限」。 */
export function tierLabel(iv: UserPricingInterval): string {
  if (iv.tier_label) return iv.tier_label
  return iv.max_tokens == null ? `>${formatTokenCount(iv.min_tokens)}` : `≤${formatTokenCount(iv.max_tokens)}`
}

/** 实付 token 档位（单档时只有一行、无标签）。 */
export function paidTiers(m: PlazaModel): TokenTier[] {
  const p = m.pricing
  if (!p) return []
  const intervals = sortByContext(p.intervals ?? [])
  if (intervals.length > 1) {
    return intervals.map((raw) => {
      const iv = resolveIntervalPrices(raw, p)
      return {
        label: tierLabel(iv),
        input: iv.input_price ?? null,
        output: iv.output_price ?? null,
        cacheRead: iv.cache_read_price ?? null,
        cacheWrite: iv.cache_write_price ?? null,
        cacheWrite1h: iv.cache_write_1h_price ?? null
      }
    })
  }
  return [{
    label: '',
    input: p.input_price ?? null,
    output: p.output_price ?? null,
    cacheRead: p.cache_read_price ?? null,
    cacheWrite: p.cache_write_price ?? null,
    cacheWrite1h: p.cache_write_1h_price ?? null
  }]
}

/** 官方 token 档位（参考价）。 */
export function officialTiers(m: PlazaModel): TokenTier[] {
  const o = m.official_pricing
  if (!o) return []
  const intervals = sortByContext(o.intervals ?? [])
  if (intervals.length > 1) {
    return intervals.map((iv) => ({
      label: tierLabel(iv),
      input: iv.input_price ?? null,
      output: iv.output_price ?? null,
      cacheRead: iv.cache_read_price ?? null,
      cacheWrite: iv.cache_write_price ?? null,
      cacheWrite1h: iv.cache_write_1h_price ?? null
    }))
  }
  return [{
    label: '',
    input: o.input_price,
    output: o.output_price,
    cacheRead: o.cache_read_price,
    cacheWrite: o.cache_write_price,
    cacheWrite1h: o.cache_write_1h_price ?? null
  }]
}

/** 按张 / 按次模型的价格档（无档位时返回单一价）。 */
export function unitPrices(m: PlazaModel): { label: string; price: number }[] {
  const p = m.pricing
  if (!p) return []
  const tiers = (p.intervals ?? []).filter((iv) => iv.per_request_price != null)
  if (tiers.length) return tiers.map((iv) => ({ label: tierLabel(iv), price: iv.per_request_price as number }))
  return p.per_request_price != null ? [{ label: '', price: p.per_request_price }] : []
}

/**
 * 展示排序：token 模型在前（按官方输出价从高到低，即旗舰在前），
 * 按张 / 按次模型沉底；同价按名称降序，新版本号在前。
 */
export function sortModels(models: PlazaModel[]): PlazaModel[] {
  return [...models].sort((a, b) => {
    const ta = isTokenModel(a)
    const tb = isTokenModel(b)
    if (ta !== tb) return ta ? -1 : 1
    const pa = a.official_pricing?.output_price ?? a.pricing?.output_price ?? null
    const pb = b.official_pricing?.output_price ?? b.pricing?.output_price ?? null
    if (pa != null && pb != null && pa !== pb) return pb - pa
    if (pa != null && pb == null) return -1
    if (pa == null && pb != null) return 1
    return b.name.localeCompare(a.name)
  })
}

/** 首页预览用的代表模型：有输入、输出价的 token 模型中排序第一个，否则取第一个模型。 */
export function featuredModel(g: ModelPlazaGroup): PlazaModel | null {
  const sorted = sortModels(g.models)
  return sorted.find((m) => isTokenModel(m) && m.pricing?.input_price != null && m.pricing?.output_price != null) ?? sorted[0] ?? null
}

/** 去掉浮点噪声的倍率展示：0.25 → "0.25"，1.5 → "1.5"。 */
export function formatRate(rate: number): string {
  return String(Math.round(rate * 1000) / 1000)
}

/**
 * 倍率换算成中文「折」：0.25 → 2.5 折，0.08 → 0.8 折；≥1 时返回空（不是折扣）。
 */
export function discountZh(rate: number): string {
  if (!(rate > 0) || rate >= 1) return ''
  return `${Math.round(rate * 100) / 10} 折`
}
