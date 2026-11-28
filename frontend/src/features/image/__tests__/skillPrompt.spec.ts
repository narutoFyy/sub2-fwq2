import { describe, expect, it } from 'vitest'
import { buildSkillPrompt, imageBaseUrl, keyCanGenerateImages, maskKey, MY_IMAGE_REPO } from '../skillPrompt'

describe('my-image skill prompt', () => {
  it('appends /v1 to the site address once', () => {
    expect(imageBaseUrl('https://example.test/')).toBe('https://example.test/v1')
    expect(imageBaseUrl('https://example.test/v1')).toBe('https://example.test/v1')
  })

  it('carries the repo, base url and key in both languages', () => {
    for (const lang of ['zh', 'en'] as const) {
      const text = buildSkillPrompt({ lang, baseUrl: 'https://example.test/v1', apiKey: 'sk-test-123' })
      expect(text).toContain(MY_IMAGE_REPO)
      expect(text).toContain('https://example.test/v1')
      expect(text).toContain('sk-test-123')
      expect(text).toContain('~/.codex/skills/my-image')
      expect(text).toContain('~/.claude/skills/my-image')
    }
  })

  it('masks keys for display', () => {
    expect(maskKey('sk-abcdefghijklmnop')).toBe('sk-ab••••••••mnop')
    expect(maskKey('short')).toBe('••••••••')
  })

  it('treats active keys on an image group, or with an extra image group, as image keys', () => {
    expect(keyCanGenerateImages({ status: 'active', group: { allow_image_generation: true } })).toBe(true)
    expect(keyCanGenerateImages({ status: 'inactive', group: { allow_image_generation: true } })).toBe(false)
    expect(keyCanGenerateImages({ status: 'active', group: { allow_image_generation: false }, group_ids: [1, 7] }, new Set([7]))).toBe(true)
    expect(keyCanGenerateImages({ status: 'active', group: { allow_image_generation: false }, group_ids: [1] }, new Set([7]))).toBe(false)
  })
})
