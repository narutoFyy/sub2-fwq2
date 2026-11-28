import { describe, expect, it } from 'vitest'
import { GUIDE_CLIENTS } from '../config'
import { nextStep, prevStep, stageOf, type GuideContext } from '../steps'

const base: GuideContext = { hasFunds: false, purchaseMode: 'external', redeemAvailable: true }

describe('onboarding image path', () => {
  it('lists image as the third choice', () => {
    expect(GUIDE_CLIENTS.indexOf('image')).toBe(2)
  })

  it('asks web or agent right after picking image, then funds', () => {
    const ctx = { ...base, imageClient: true, imageMode: null }
    expect(nextStep('client', ctx)).toBe('imageMode')
    expect(nextStep('imageMode', ctx)).toBe('buy')
    expect(nextStep('imageMode', { ...ctx, hasFunds: true })).toBe('key')
    expect(prevStep('imageMode')).toBe('client')
    expect(stageOf('imageMode')).toBe(0)
  })

  it('goes to the image page after the key for web, and to the skill prompt for agents', () => {
    expect(nextStep('key', { ...base, imageClient: true, imageMode: 'web' })).toBe('imageGo')
    expect(nextStep('key', { ...base, imageClient: true, imageMode: 'agent' })).toBe('skill')
    expect(nextStep('imageGo', base)).toBe('finish')
    expect(nextStep('skill', base)).toBe('finish')
    expect(stageOf('skill')).toBe(3)
  })

  it('leaves the other paths unchanged', () => {
    expect(nextStep('client', base)).toBe('buy')
    expect(nextStep('key', base)).toBe('ccs')
    expect(nextStep('key', { ...base, chatOnly: true })).toBe('chat')
  })
})
