import { afterEach, describe, expect, it, vi } from 'vitest'
import { generateImages, ImageTimeoutError, sortImageModels } from '../api'

describe('sortImageModels', () => {
  it('puts the newest plain gpt-image first and channel variants after', () => {
    const ids = ['gpt-image-2.5', 'gpt-image-2.5-flare', 'gpt-image-2.5-sunburst', 'gpt-image-2.6', 'gpt-image-2.6-flare', 'gpt-image-2.6-sunburst']
    expect(sortImageModels(ids)).toEqual(['gpt-image-2.6', 'gpt-image-2.5', 'gpt-image-2.5-flare', 'gpt-image-2.5-sunburst', 'gpt-image-2.6-flare', 'gpt-image-2.6-sunburst'])
  })

  it('keeps other providers in their original order', () => {
    expect(sortImageModels(['grok-imagine', 'flux-pro', 'gpt-image-2'])).toEqual(['gpt-image-2', 'grok-imagine', 'flux-pro'])
  })
})

describe('generateImages errors', () => {
  afterEach(() => vi.unstubAllGlobals())
  const base = { apiKey: 'sk-x', model: 'gpt-image-2.6', prompt: 'cat', ratio: 'square' as const, count: 1, quality: 'auto' as const }

  it('turns a Cloudflare 524 into a timeout error instead of showing HTML', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('<!DOCTYPE html><html>timeout</html>', { status: 524 })))
    await expect(generateImages(base)).rejects.toBeInstanceOf(ImageTimeoutError)
  })

  it('never shows an HTML error page as the message', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('<html><body>Bad gateway</body></html>', { status: 502 })))
    await expect(generateImages(base)).rejects.toMatchObject({ message: 'HTTP 502' })
  })

  it('reads url results', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ data: [{ url: 'https://cdn.example/a.png' }] }), { status: 200 })))
    await expect(generateImages(base)).resolves.toEqual([{ src: 'https://cdn.example/a.png' }])
  })
})
