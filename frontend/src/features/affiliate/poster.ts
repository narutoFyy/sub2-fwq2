import { buildInviteLink, isZhLocale } from './helpers'

export interface GenerateAffiliatePosterOptions {
  affCode: string
  inviteLink?: string
  brandName?: string
  origin?: string
  locale?: string
}

export interface GeneratedAffiliatePoster {
  dataUrl: string
  blob: Blob | null
  inviteLink: string
}

const POSTER_WIDTH = 750
const POSTER_HEIGHT = 1060
const PEBBLE_SVG_PATH =
  'M16 5.5C23.9 5.5 29.5 9.3 29.5 15.6C29.5 22 23.7 26.5 16 26.5C8.3 26.5 2.5 22 2.5 15.6C2.5 9.3 8.1 5.5 16 5.5Z'
const FONT_SANS =
  '"Geist Variable", "Geist", -apple-system, BlinkMacSystemFont, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
const FONT_MONO =
  '"Geist Mono Variable", "Geist Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace'
const FONT_PRELOAD_TIMEOUT_MS = 800

function drawRoundedRectPath(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number,
): void {
  const radius = Math.min(r, w / 2, h / 2)
  ctx.beginPath()
  ctx.moveTo(x + radius, y)
  ctx.lineTo(x + w - radius, y)
  ctx.quadraticCurveTo(x + w, y, x + w, y + radius)
  ctx.lineTo(x + w, y + h - radius)
  ctx.quadraticCurveTo(x + w, y + h, x + w - radius, y + h)
  ctx.lineTo(x + radius, y + h)
  ctx.quadraticCurveTo(x, y + h, x, y + h - radius)
  ctx.lineTo(x, y + radius)
  ctx.quadraticCurveTo(x, y, x + radius, y)
  ctx.closePath()
}

function drawSplitPebbleLogo(ctx: CanvasRenderingContext2D, x: number, y: number, size: number): void {
  const scale = size / 32
  const drawPebble = () => {
    if (typeof Path2D !== 'undefined') {
      ctx.fill(new Path2D(PEBBLE_SVG_PATH))
    } else {
      ctx.beginPath()
      ctx.ellipse(16, 16, 13.5, 10.5, 0, 0, Math.PI * 2)
      ctx.fill()
    }
  }

  // Top half (clipped by BrandLogo.vue top polygon)
  ctx.save()
  ctx.translate(x, y)
  ctx.scale(scale, scale)
  ctx.beginPath()
  ctx.moveTo(-4, -4)
  ctx.lineTo(36, -4)
  ctx.lineTo(36, 12.2)
  ctx.lineTo(-4, 15.9)
  ctx.closePath()
  ctx.clip()
  ctx.fillStyle = '#ffffff'
  drawPebble()
  ctx.restore()

  // Bottom half (clipped by BrandLogo.vue bottom polygon)
  ctx.save()
  ctx.translate(x, y)
  ctx.scale(scale, scale)
  ctx.beginPath()
  ctx.moveTo(-4, 18.5)
  ctx.lineTo(36, 14.8)
  ctx.lineTo(36, 36)
  ctx.lineTo(-4, 36)
  ctx.closePath()
  ctx.clip()
  ctx.fillStyle = '#ffffff'
  drawPebble()
  ctx.restore()
}

function drawGlowDotField(ctx: CanvasRenderingContext2D): void {
  const topGlow = ctx.createRadialGradient(620, 130, 10, 620, 130, 420)
  topGlow.addColorStop(0, 'rgba(34, 211, 238, 0.24)')
  topGlow.addColorStop(0.5, 'rgba(34, 211, 238, 0.08)')
  topGlow.addColorStop(1, 'rgba(34, 211, 238, 0)')
  ctx.fillStyle = topGlow
  ctx.fillRect(0, 0, POSTER_WIDTH, POSTER_HEIGHT)

  const bottomGlow = ctx.createRadialGradient(140, 960, 10, 140, 960, 440)
  bottomGlow.addColorStop(0, 'rgba(34, 211, 238, 0.16)')
  bottomGlow.addColorStop(0.55, 'rgba(163, 228, 232, 0.05)')
  bottomGlow.addColorStop(1, 'rgba(34, 211, 238, 0)')
  ctx.fillStyle = bottomGlow
  ctx.fillRect(0, 0, POSTER_WIDTH, POSTER_HEIGHT)

  const step = 18
  for (let gx = 18; gx < POSTER_WIDTH; gx += step) {
    for (let gy = 18; gy < POSTER_HEIGHT; gy += step) {
      const d1 = Math.hypot(gx - 610, gy - 150)
      const d2 = Math.hypot(gx - 150, gy - 920)
      const intensity = Math.max(0, 1 - d1 / 380) * 0.48 + Math.max(0, 1 - d2 / 360) * 0.26
      if (intensity < 0.03) continue
      ctx.fillStyle = `rgba(163, 228, 232, ${Math.min(0.45, intensity).toFixed(3)})`
      ctx.beginPath()
      ctx.arc(gx, gy, 1.35, 0, Math.PI * 2)
      ctx.fill()
    }
  }
}

function extractShortDomain(url: string): string {
  try {
    const parsed = new URL(url)
    return parsed.host
  } catch {
    return url.replace(/^https?:\/\//i, '').split('/')[0] || 'yf-mail.com'
  }
}

async function preloadPosterFonts(): Promise<void> {
  if (typeof document === 'undefined' || typeof document.fonts?.load !== 'function') {
    return
  }
  await Promise.race([
    Promise.allSettled([
      document.fonts.load('700 56px "Geist Variable"'),
      document.fonts.load('600 19px "Geist Mono Variable"'),
    ]),
    new Promise<void>((resolve) => {
      setTimeout(resolve, FONT_PRELOAD_TIMEOUT_MS)
    }),
  ])
}

export async function generateAffiliatePoster(
  options: GenerateAffiliatePosterOptions,
): Promise<GeneratedAffiliatePoster> {
  const affCode = options.affCode.trim()
  const inviteLink = options.inviteLink?.trim() || buildInviteLink(affCode, options.origin)
  const zh = isZhLocale(options.locale)
  const brandName = options.brandName?.trim() || (zh ? '石头中转站' : 'yf-mail')

  const copy = zh
    ? {
        badge: '好友专属邀请',
        heroLine1: '像石头一样稳的',
        heroLine2: 'AI 模型中转站',
        subtitle: '支持 Claude Code · Codex · 网页直聊',
        cardEyebrow: 'NEW USER TRIAL',
        cardTitle: '扫码注册立领 1 元体验额度',
        cardSubline: '7 天有效 · 主流邮箱自动到账',
        tags: ['Claude / GPT / Gemini', '低倍率按量计费', '一键导入 CC Switch'],
        dockLine1: '微信 / 相机扫码',
        dockLine2: '直达注册页',
        codeLabel: `专属邀请码：${affCode}`,
      }
    : {
        badge: 'FRIEND INVITE',
        heroLine1: 'Rock-Solid',
        heroLine2: 'AI API Gateway',
        subtitle: 'Claude Code · Codex · Web Chat',
        cardEyebrow: 'NEW USER TRIAL',
        cardTitle: 'Scan for 1.00 CNY trial credit',
        cardSubline: 'Valid 7 days · Auto-credited for mainstream emails',
        tags: ['Claude / GPT / Gemini', 'Low-rate billing', '1-click CC Switch'],
        dockLine1: 'Scan with camera',
        dockLine2: 'to sign up now',
        codeLabel: `Invite code: ${affCode}`,
      }

  // Dynamic import keeps qrcode completely out of the main entry bundle.
  const [qrModule] = await Promise.all([import('qrcode'), preloadPosterFonts()])
  const QRCode = (qrModule.default ?? qrModule) as typeof import('qrcode')

  const canvas = document.createElement('canvas')
  canvas.width = POSTER_WIDTH
  canvas.height = POSTER_HEIGHT
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    throw new Error('Canvas 2D context is unavailable')
  }

  // 1. Base charcoal background + dot-matrix glow
  ctx.fillStyle = '#0a0a0b'
  ctx.fillRect(0, 0, POSTER_WIDTH, POSTER_HEIGHT)
  drawGlowDotField(ctx)

  // Subtle outer hairline frame
  drawRoundedRectPath(ctx, 20, 20, POSTER_WIDTH - 40, POSTER_HEIGHT - 40, 28)
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.06)'
  ctx.lineWidth = 1.5
  ctx.stroke()

  // 2. Top bar: split pebble logo + brand name + exclusive invite pill
  drawSplitPebbleLogo(ctx, 52, 54, 42)
  ctx.fillStyle = '#ffffff'
  ctx.font = `600 26px ${FONT_SANS}`
  ctx.textBaseline = 'middle'
  ctx.fillText(brandName, 108, 76)

  const pillW = 178
  const pillH = 38
  const pillX = POSTER_WIDTH - 52 - pillW
  const pillY = 57
  drawRoundedRectPath(ctx, pillX, pillY, pillW, pillH, 19)
  ctx.fillStyle = 'rgba(34, 211, 238, 0.12)'
  ctx.fill()
  ctx.strokeStyle = 'rgba(163, 228, 232, 0.34)'
  ctx.lineWidth = 1.2
  ctx.stroke()

  ctx.fillStyle = '#a3e4e8'
  ctx.font = `500 18px ${FONT_SANS}`
  ctx.textAlign = 'center'
  ctx.fillText(copy.badge, pillX + pillW / 2, pillY + pillH / 2 + 1)
  ctx.textAlign = 'left'

  // 3. Hero headline + subtitle
  ctx.textBaseline = 'top'
  ctx.fillStyle = '#fafafa'
  ctx.font = `700 56px ${FONT_SANS}`
  ctx.fillText(copy.heroLine1, 52, 166)
  ctx.fillStyle = '#ffffff'
  ctx.fillText(copy.heroLine2, 52, 238)

  ctx.fillStyle = 'rgba(228, 228, 231, 0.72)'
  ctx.font = `400 24px ${FONT_SANS}`
  ctx.fillText(copy.subtitle, 52, 326)

  // 4. Dark glass benefit card
  const cardX = 52
  const cardY = 406
  const cardW = POSTER_WIDTH - 104
  const cardH = 278
  drawRoundedRectPath(ctx, cardX, cardY, cardW, cardH, 26)
  ctx.fillStyle = '#141417'
  ctx.fill()
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.1)'
  ctx.lineWidth = 1.5
  ctx.stroke()

  // Soft top accent bar inside card
  const accentGrad = ctx.createLinearGradient(cardX + 36, cardY, cardX + 280, cardY)
  accentGrad.addColorStop(0, '#22d3ee')
  accentGrad.addColorStop(1, 'rgba(34, 211, 238, 0)')
  ctx.fillStyle = accentGrad
  ctx.fillRect(cardX + 36, cardY, 240, 2.5)

  // Card label
  ctx.fillStyle = '#a3e4e8'
  ctx.font = `600 18px ${FONT_SANS}`
  ctx.fillText(copy.cardEyebrow, cardX + 36, cardY + 34)

  // Card main perk
  ctx.fillStyle = '#ffffff'
  ctx.font = `700 34px ${FONT_SANS}`
  ctx.fillText(copy.cardTitle, cardX + 36, cardY + 74)

  ctx.fillStyle = 'rgba(212, 212, 216, 0.8)'
  ctx.font = `400 21px ${FONT_SANS}`
  ctx.fillText(copy.cardSubline, cardX + 36, cardY + 132)

  // 3 feature tags
  let tagX = cardX + 36
  const tagY = cardY + 194
  ctx.font = `500 17px ${FONT_SANS}`
  for (const tag of copy.tags) {
    const measured = typeof ctx.measureText === 'function' ? ctx.measureText(tag).width : tag.length * 14
    const tagW = Math.ceil(measured) + 28
    const tagH = 40
    drawRoundedRectPath(ctx, tagX, tagY, tagW, tagH, 12)
    ctx.fillStyle = 'rgba(255, 255, 255, 0.06)'
    ctx.fill()
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.1)'
    ctx.lineWidth = 1
    ctx.stroke()

    ctx.fillStyle = '#e4e4e7'
    ctx.textBaseline = 'middle'
    ctx.fillText(tag, tagX + 14, tagY + tagH / 2 + 1)
    ctx.textBaseline = 'top'
    tagX += tagW + 12
  }

  // 5. High-contrast white QR dock at bottom
  const dockX = 52
  const dockY = 724
  const dockW = POSTER_WIDTH - 104
  const dockH = 282
  drawRoundedRectPath(ctx, dockX, dockY, dockW, dockH, 28)
  ctx.fillStyle = '#ffffff'
  ctx.fill()

  // Left text inside dock
  ctx.fillStyle = '#09090b'
  ctx.font = `700 26px ${FONT_SANS}`
  ctx.fillText(copy.dockLine1, dockX + 34, dockY + 42)
  ctx.fillText(copy.dockLine2, dockX + 34, dockY + 80)

  // Code pill
  drawRoundedRectPath(ctx, dockX + 34, dockY + 138, 336, 48, 12)
  ctx.fillStyle = '#f4f4f5'
  ctx.fill()
  ctx.strokeStyle = '#e4e4e7'
  ctx.lineWidth = 1
  ctx.stroke()

  ctx.fillStyle = '#18181b'
  ctx.font = `600 19px ${FONT_MONO}`
  ctx.textBaseline = 'middle'
  ctx.fillText(
    copy.codeLabel.length > 24 ? `${copy.codeLabel.slice(0, 23)}…` : copy.codeLabel,
    dockX + 48,
    dockY + 162,
  )
  ctx.textBaseline = 'top'

  // Domain footer
  ctx.fillStyle = '#71717a'
  ctx.font = `500 18px ${FONT_MONO}`
  ctx.fillText(extractShortDomain(inviteLink), dockX + 34, dockY + 216)

  // Right QR code with ample white quiet zone
  const qrSize = 220
  const qrCanvas = document.createElement('canvas')
  qrCanvas.width = qrSize
  qrCanvas.height = qrSize
  await QRCode.toCanvas(qrCanvas, inviteLink, {
    width: qrSize,
    margin: 1,
    errorCorrectionLevel: 'M',
    color: {
      dark: '#0a0a0b',
      light: '#ffffff',
    },
  })

  const qrDrawX = dockX + dockW - qrSize - 32
  const qrDrawY = dockY + Math.round((dockH - qrSize) / 2)
  ctx.drawImage(qrCanvas, qrDrawX, qrDrawY, qrSize, qrSize)

  const dataUrl = canvas.toDataURL('image/png')
  const blob = await new Promise<Blob | null>((resolve) => {
    if (typeof canvas.toBlob !== 'function') {
      resolve(null)
      return
    }
    canvas.toBlob((result) => resolve(result), 'image/png')
  })

  return {
    dataUrl,
    blob,
    inviteLink,
  }
}
