import { describe, expect, it } from 'vitest'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import { discountZh, featuredModel, formatUsd, modelRate, paidTiers, perMillion, sortModels } from '../pricing'
import { discountEn } from '../content'

function model(name: string, input: number | null, output: number | null, extra: Partial<PlazaModel> = {}): PlazaModel {
  return {
    name,
    platform: 'openai',
    pricing: {
      billing_mode: 'token', input_price: input, output_price: output, cache_write_price: null, cache_read_price: null,
      image_input_price: null, image_output_price: null, per_request_price: null, intervals: []
    },
    official_pricing: { input_price: input, output_price: output, cache_write_price: null, cache_read_price: null },
    ...extra
  }
}

function group(extra: Partial<ModelPlazaGroup> = {}): ModelPlazaGroup {
  return {
    id: 1, name: 'g', description: '', platform: 'openai', subscription_type: 'standard', rate_multiplier: 0.25,
    peak_rate_enabled: false, peak_start: '', peak_end: '', peak_rate_multiplier: 1, is_exclusive: false,
    image_rate_independent: false, image_rate_multiplier: 1, video_rate_independent: false, video_rate_multiplier: 1,
    long_context_pricing_enabled: true, models: [], ...extra
  }
}

describe('plaza pricing', () => {
  it('formats USD without float noise and keeps small prices visible', () => {
    expect(formatUsd(0.30000000004)).toBe('$0.30')
    expect(formatUsd(3)).toBe('$3.00')
    expect(formatUsd(0.0375)).toBe('$0.0375')
    expect(formatUsd(0.0035)).toBe('$0.0035')
    expect(formatUsd(12.345)).toBe('$12.35')
    expect(formatUsd(null)).toBe('—')
  })

  it('converts per-token prices to USD per million tokens times the rate', () => {
    expect(perMillion(3e-6, 0.25)).toBe('$0.75')
    expect(perMillion(1.5e-5, 0.08)).toBe('$1.20')
    expect(perMillion(null, 1)).toBe('—')
  })

  it('uses the personal rate and independent media rates', () => {
    const g = group({ user_rate_multiplier: 0.2, image_rate_independent: true, image_rate_multiplier: 0.5 })
    const img = model('gpt-image', null, null)
    img.pricing!.billing_mode = 'image'
    expect(modelRate(g, model('gpt-5', 1e-6, 2e-6))).toBe(0.2)
    expect(modelRate(g, img)).toBe(0.5)
  })

  it('puts flagship token models first and picks them as featured', () => {
    const cheap = model('gpt-mini', 1e-7, 4e-7)
    const flagship = model('gpt-5.5', 5e-6, 3e-5)
    const img = model('gpt-image', null, null)
    img.pricing!.billing_mode = 'image'
    expect(sortModels([img, cheap, flagship]).map((m) => m.name)).toEqual(['gpt-5.5', 'gpt-mini', 'gpt-image'])
    expect(featuredModel(group({ models: [img, cheap, flagship] }))?.name).toBe('gpt-5.5')
  })

  it('expands multi-tier pricing with labels', () => {
    const m = model('gpt-5.4', 2.5e-6, 1.5e-5)
    m.pricing!.intervals = [
      { min_tokens: 0, max_tokens: 272000, input_price: 2.5e-6, output_price: 1.5e-5, cache_write_price: null, cache_read_price: null, per_request_price: null },
      { min_tokens: 272000, max_tokens: null, input_price: 5e-6, output_price: 2.25e-5, cache_write_price: null, cache_read_price: null, per_request_price: null }
    ]
    expect(paidTiers(m).map((t) => t.label)).toEqual(['≤272K', '>272K'])
  })

  it('describes discounts in Chinese and English', () => {
    expect(discountZh(0.25)).toBe('2.5 折')
    expect(discountZh(0.08)).toBe('0.8 折')
    expect(discountZh(1.5)).toBe('')
    expect(discountEn(0.25)).toBe('75% off list')
  })
})
