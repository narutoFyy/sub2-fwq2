import { describe, expect, it } from 'vitest'
import { FUND_STEPS, nextStep, prevStep, stageOf, type GuideContext } from '../steps'

const base: GuideContext = { hasFunds: false, purchaseMode: 'external', redeemAvailable: true }

describe('onboarding steps', () => {
  it('walks the full path for a user with no credit and an external shop', () => {
    expect(nextStep('welcome', base)).toBe('client')
    expect(nextStep('client', base)).toBe('buy')
    expect(nextStep('buy', base)).toBe('returned')
    expect(nextStep('returned', base)).toBe('redeem')
    expect(nextStep('redeem', base)).toBe('key')
    expect(nextStep('key', base)).toBe('finish')
    expect(nextStep('finish', base)).toBe('finish')
  })

  it('skips the funding steps when the user already has credit', () => {
    const ctx = { ...base, hasFunds: true }
    expect(nextStep('client', ctx)).toBe('key')
    expect(nextStep('buy', ctx)).toBe('key')
    expect(nextStep('returned', ctx)).toBe('key')
  })

  it('goes straight to redeem when there is no purchase entry', () => {
    expect(nextStep('client', { ...base, purchaseMode: 'none' })).toBe('redeem')
  })

  it('skips funding entirely when there is neither a purchase entry nor a redeem page', () => {
    expect(nextStep('client', { ...base, purchaseMode: 'none', redeemAvailable: false })).toBe('key')
    expect(nextStep('returned', { ...base, redeemAvailable: false })).toBe('key')
  })

  it('maps steps to the four progress stages', () => {
    expect(stageOf('client')).toBe(0)
    for (const step of FUND_STEPS) expect(stageOf(step)).toBe(1)
    expect(stageOf('key')).toBe(2)
    expect(stageOf('finish')).toBe(3)
  })

  it('goes back one step where it makes sense', () => {
    expect(prevStep('client')).toBe('welcome')
    expect(prevStep('returned')).toBe('buy')
    expect(prevStep('key')).toBe('key')
  })
})
