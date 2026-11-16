import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const frontendRoot = resolve(__dirname, '../../../..')
const stripeConsumers = [
  'src/views/user/StripePaymentView.vue',
  'src/views/user/StripePopupView.vue',
  'src/components/payment/StripePaymentInline.vue',
]

function readFrontendFile(path: string): string {
  return readFileSync(resolve(frontendRoot, path), 'utf8')
}

describe('Stripe lazy-loading contract', () => {
  it.each(stripeConsumers)('%s uses the side-effect-free Stripe loader', (path) => {
    const source = readFrontendFile(path)

    expect(source).toContain("await import('@stripe/stripe-js/pure')")
    expect(source).not.toMatch(/await import\(['"]@stripe\/stripe-js['"]\)/)
  })

  it('keeps Stripe out of the shared vendor chunk', () => {
    const viteConfig = readFrontendFile('vite.config.ts')
    const stripeRule = viteConfig.indexOf("id.includes('/@stripe/stripe-js/')")

    expect(stripeRule).toBeGreaterThan(-1)
    expect(viteConfig.slice(stripeRule, stripeRule + 120)).toContain("return 'vendor-stripe'")
    // 不再有把剩余第三方库合并在一起的公共包，避免 Stripe 等大库被首屏间接引入。
    expect(viteConfig).not.toContain("return 'vendor-misc'")
  })
})
