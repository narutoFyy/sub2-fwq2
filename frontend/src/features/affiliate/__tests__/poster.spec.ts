import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { toCanvasMock } = vi.hoisted(() => ({
  toCanvasMock: vi.fn(),
}))

vi.mock('qrcode', () => ({
  default: {
    toCanvas: toCanvasMock,
  },
}))

import { generateAffiliatePoster } from '../poster'

function createMockCanvasContext() {
  const gradient = { addColorStop: vi.fn() }
  return {
    fillStyle: '',
    strokeStyle: '',
    lineWidth: 1,
    font: '',
    textBaseline: 'alphabetic',
    textAlign: 'left',
    fillRect: vi.fn(),
    beginPath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    quadraticCurveTo: vi.fn(),
    closePath: vi.fn(),
    fill: vi.fn(),
    stroke: vi.fn(),
    arc: vi.fn(),
    ellipse: vi.fn(),
    clip: vi.fn(),
    save: vi.fn(),
    restore: vi.fn(),
    translate: vi.fn(),
    scale: vi.fn(),
    fillText: vi.fn(),
    measureText: vi.fn((text: string) => ({ width: text.length * 10 })),
    createRadialGradient: vi.fn(() => gradient),
    createLinearGradient: vi.fn(() => gradient),
    drawImage: vi.fn(),
  }
}

describe('generateAffiliatePoster', () => {
  let mockCtx: ReturnType<typeof createMockCanvasContext>
  let createElementSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    vi.clearAllMocks()
    toCanvasMock.mockResolvedValue(undefined)
    mockCtx = createMockCanvasContext()

    const origCreateElement = document.createElement.bind(document)
    createElementSpy = vi.spyOn(document, 'createElement').mockImplementation(((tagName: string) => {
      if (tagName.toLowerCase() === 'canvas') {
        const fakeBlob = new Blob(['png-bytes'], { type: 'image/png' })
        return {
          width: 0,
          height: 0,
          getContext: vi.fn(() => mockCtx),
          toDataURL: vi.fn(() => 'data:image/png;base64,poster-mock'),
          toBlob: vi.fn((cb: (blob: Blob | null) => void) => cb(fakeBlob)),
        } as unknown as HTMLCanvasElement
      }
      return origCreateElement(tagName)
    }) as typeof document.createElement)
  })

  afterEach(() => {
    createElementSpy.mockRestore()
  })

  it('lazily invokes QRCode.toCanvas with the exact invite registration URL, preloads Geist Variable fonts, and returns PNG output', async () => {
    const fontsLoad = vi.fn().mockResolvedValue([])
    Object.defineProperty(document, 'fonts', {
      configurable: true,
      value: { load: fontsLoad },
    })

    const fontsUsed: string[] = []
    const origFillText = mockCtx.fillText
    mockCtx.fillText = vi.fn((...args: unknown[]) => {
      fontsUsed.push(mockCtx.font)
      return origFillText(...args)
    })

    const result = await generateAffiliatePoster({
      affCode: 'STONE2026',
      origin: 'https://yf-mail.com',
      brandName: '石头中转站',
    })

    expect(fontsLoad).toHaveBeenCalledWith('700 56px "Geist Variable"')
    expect(fontsLoad).toHaveBeenCalledWith('600 19px "Geist Mono Variable"')
    expect(fontsUsed.some((font) => font.includes('"Geist Variable"'))).toBe(true)
    expect(fontsUsed.some((font) => font.includes('"Geist Mono Variable"'))).toBe(true)

    expect(result.inviteLink).toBe('https://yf-mail.com/register?aff=STONE2026')
    expect(result.dataUrl).toBe('data:image/png;base64,poster-mock')
    expect(result.blob).toBeInstanceOf(Blob)

    expect(toCanvasMock).toHaveBeenCalledTimes(1)
    expect(toCanvasMock).toHaveBeenCalledWith(
      expect.anything(),
      'https://yf-mail.com/register?aff=STONE2026',
      expect.objectContaining({
        width: 220,
        margin: 1,
        errorCorrectionLevel: 'M',
        color: { dark: '#0a0a0b', light: '#ffffff' },
      }),
    )

    expect(mockCtx.drawImage).toHaveBeenCalledTimes(1)
    expect(mockCtx.fillText).toHaveBeenCalledWith('像石头一样稳的', expect.any(Number), expect.any(Number))
    expect(mockCtx.fillText).toHaveBeenCalledWith('扫码注册立领 1 元体验额度', expect.any(Number), expect.any(Number))
    expect(mockCtx.fillText).toHaveBeenCalledWith(expect.stringContaining('STONE2026'), expect.any(Number), expect.any(Number))
  })

  it('renders localized English copy when locale is en', async () => {
    await generateAffiliatePoster({
      affCode: 'ENG2026',
      origin: 'https://yf-mail.com',
      brandName: 'yf-mail',
      locale: 'en',
    })

    expect(mockCtx.fillText).toHaveBeenCalledWith('FRIEND INVITE', expect.any(Number), expect.any(Number))
    expect(mockCtx.fillText).toHaveBeenCalledWith('Rock-Solid', expect.any(Number), expect.any(Number))
    expect(mockCtx.fillText).toHaveBeenCalledWith('Scan for 1.00 CNY trial credit', expect.any(Number), expect.any(Number))
    expect(mockCtx.fillText).toHaveBeenCalledWith('Invite code: ENG2026', expect.any(Number), expect.any(Number))
  })
})
