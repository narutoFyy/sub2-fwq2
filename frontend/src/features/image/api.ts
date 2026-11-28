import { buildGatewayUrl } from '@/api/url'
import { GatewayError, listGatewayModels, mapGatewayError } from '@/features/chat/gateway'

export { GatewayError }

export type ImageRatio = 'square' | 'landscape' | 'portrait'
export type ImageQuality = 'auto' | 'low' | 'high'

export const RATIO_SIZE: Record<ImageRatio, string> = {
  square: '1024x1024',
  landscape: '1536x1024',
  portrait: '1024x1536'
}

export interface GeneratedImage {
  /** 可直接给 <img> 用的地址：data URL 或上游返回的 URL。 */
  src: string
}

interface ImagesResponse {
  data?: Array<{ b64_json?: string; url?: string }>
}

/**
 * 线上前面挂着 Cloudflare：一次请求超过约 100 秒会被它切断（524），
 * 网关重启或过载时也会回一整页 HTML。这些情况都不能把 HTML 原样展示给用户。
 */
export class ImageTimeoutError extends Error {
  constructor() {
    super('timeout')
    this.name = 'ImageTimeoutError'
  }
}
const TIMEOUT_STATUS = new Set([504, 522, 524])

async function readBody(response: Response): Promise<unknown> {
  const text = await response.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    const html = /^\s*</.test(text)
    return { error: { message: html ? `HTTP ${response.status}` : text.slice(0, 300) } }
  }
}

function toImages(body: unknown): GeneratedImage[] {
  const data = (body as ImagesResponse | null)?.data ?? []
  return data
    .map((item) => {
      if (item.b64_json) return { src: `data:image/png;base64,${item.b64_json}` }
      if (item.url) return { src: item.url }
      return null
    })
    .filter((item): item is GeneratedImage => item !== null)
}

async function send(path: string, apiKey: string, init: RequestInit): Promise<GeneratedImage[]> {
  const response = await fetch(buildGatewayUrl(path), {
    ...init,
    headers: { Authorization: `Bearer ${apiKey}`, ...(init.headers ?? {}) }
  })
  if (TIMEOUT_STATUS.has(response.status)) throw new ImageTimeoutError()
  const body = await readBody(response)
  if (!response.ok) {
    const mapped = mapGatewayError(response.status, body)
    throw new GatewayError(mapped.code, mapped.message)
  }
  return toImages(body)
}

export interface GenerateOptions {
  apiKey: string
  model: string
  prompt: string
  ratio: ImageRatio
  count: number
  quality: ImageQuality
  /** 有参考图时走 /images/edits。 */
  reference?: Blob | null
  signal?: AbortSignal
}

export function generateImages(options: GenerateOptions): Promise<GeneratedImage[]> {
  const size = RATIO_SIZE[options.ratio]
  if (options.reference) {
    const form = new FormData()
    form.append('model', options.model)
    form.append('prompt', options.prompt)
    form.append('size', size)
    form.append('n', String(options.count))
    if (options.quality !== 'auto') form.append('quality', options.quality)
    form.append('image', options.reference, 'reference.png')
    return send('/v1/images/edits', options.apiKey, { method: 'POST', body: form, signal: options.signal })
  }
  const payload: Record<string, unknown> = { model: options.model, prompt: options.prompt, size, n: options.count }
  if (options.quality !== 'auto') payload.quality = options.quality
  return send('/v1/images/generations', options.apiKey, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
    signal: options.signal
  })
}

/** 这把密钥能用的生图模型（按名字判断），推荐的排在最前。 */
export async function listImageModels(apiKey: string, signal?: AbortSignal): Promise<string[]> {
  const ids = await listGatewayModels({ completionsUrl: buildGatewayUrl('/v1/chat/completions'), apiKey, signal })
  return sortImageModels(ids.filter((id) => /image|imagine|imagen|dall-e|flux|seedream|banana/i.test(id)))
}

/**
 * 默认选最新的标准版 gpt-image（例如有 2.6 就不选 2.5），
 * 带后缀的渠道变体（-flare、-sunburst 等）和其他家的模型排在后面，名字顺序不变。
 */
export function sortImageModels(ids: string[]): string[] {
  const plain = (id: string) => {
    const m = /^gpt-image-(\d+(?:\.\d+)?)$/i.exec(id)
    return m ? Number(m[1]) : null
  }
  return [...ids].sort((a, b) => {
    const va = plain(a)
    const vb = plain(b)
    if (va !== null && vb !== null) return vb - va
    if (va !== null) return -1
    if (vb !== null) return 1
    return 0
  })
}

/**
 * 上游有时返回的是图床临时链接，过一阵就失效。拿到后转成 data URL 存进本地历史；
 * 跨域取不到就保留原链接。
 */
export async function toLocalSrc(src: string): Promise<string> {
  if (!/^https?:/i.test(src)) return src
  try {
    const response = await fetch(src)
    if (!response.ok) return src
    const blob = await response.blob()
    if (!blob.type.startsWith('image/')) return src
    return await new Promise<string>((resolve) => {
      const reader = new FileReader()
      reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : src)
      reader.onerror = () => resolve(src)
      reader.readAsDataURL(blob)
    })
  } catch {
    return src
  }
}

export function isAbortError(error: unknown): boolean {
  return error instanceof DOMException ? error.name === 'AbortError' : error instanceof Error && error.name === 'AbortError'
}
