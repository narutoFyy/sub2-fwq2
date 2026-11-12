/**
 * 柔光点阵背景（Canvas 2D，无依赖）。
 *
 * 原理：每个网格点按"强度场"取 0~1 的值，再分 6 档画成大小、深浅不同的圆点。
 * 强度场 = 两层 Simplex 噪声叠加，并用第三层噪声做域扭曲，云团会像熔岩灯一样缓慢流动、分裂、融合。
 * 档位之间用 4x4 Bayer 有序抖动过渡，渐变是点的混排而不是一圈圈色阶。
 * 光标附近会把采样点搅成漩涡并提亮，移动越快漩涡越强。
 */

export type GlowTheme = 'midnight' | 'rose'

export interface GlowDotsOptions {
  /** 光的重心在画布中的水平位置（0~1） */
  centerX?: number
  /** 光的重心在画布中的垂直位置（0~1） */
  centerY?: number
  /** 窄屏（<640px）下的垂直重心 */
  mobileCenterY?: number
  /** 图案缩放，越大云团越大 */
  scale?: number
  /** 动画速度倍率 */
  speed?: number
  /** 点阵格子大小（CSS 像素） */
  cell?: number
  /** 窄屏下的格子大小 */
  mobileCell?: number
  /** 格子总数上限，超出时自动加大格子以保证性能 */
  maxCells?: number
  /** 是否响应鼠标 */
  interactive?: boolean
  /** 噪声种子，不同种子得到不同的云团形态 */
  seed?: number
  /** 动画起始时间。默认值是实测首屏覆盖最饱满、之后也不会变空的时刻 */
  startTime?: number
  /** 整体强度系数（<1 时最亮的核心更少，画面更克制） */
  gain?: number
  /** 加载完成第一帧后回调（用于淡入） */
  onReady?: () => void
}

interface Palette {
  grid: string
  levels: string[]
}

/**
 * 6 档从弱到强。前几档是中性灰，只有最强的核心带强调色，
 * 让大面积保持中性、颜色只出现在一处（深色：青；浅色：樱粉）。
 */
export const GLOW_PALETTES: Record<GlowTheme, Palette> = {
  midnight: { grid: '#151619', levels: ['#1a1c21', '#23262d', '#2f343d', '#3b5462', '#2aa7c2', '#8beafb'] },
  rose: { grid: '#ececee', levels: ['#e9e9ec', '#dcdce1', '#c8c8cf', '#c9a2b0', '#e0587c', '#e11d48'] },
}

/** 各档圆点直径（相对格子） */
const SIZES = [0.26, 0.36, 0.48, 0.6, 0.72, 0.84]
const BAYER = [0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5]
const TAU = Math.PI * 2

const clamp01 = (v: number) => (v < 0 ? 0 : v > 1 ? 1 : v)
const smoothstep = (a: number, b: number, x: number) => {
  const t = clamp01((x - a) / (b - a))
  return t * t * (3 - 2 * t)
}

function toLevel(v: number, col: number, row: number): number {
  if (v <= 0.015) return 0
  const b = (BAYER[((row & 3) << 2) | (col & 3)] + 0.5) / 16
  if (v < 0.14) return v / 0.14 > b ? 1 : 0
  return Math.min(6, Math.floor(1 + ((v - 0.14) / 0.86) * 5 + b))
}

function mulberry32(seed: number) {
  let s = seed >>> 0
  return () => {
    let t = (s = (s + 0x6d2b79f5) >>> 0)
    t = Math.imul(t ^ (t >>> 15), 1 | t)
    t ^= t + Math.imul(t ^ (t >>> 7), 61 | t)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

const GRAD = new Float32Array([1, 1, 0, -1, 1, 0, 1, -1, 0, -1, -1, 0, 1, 0, 1, -1, 0, 1, 1, 0, -1, -1, 0, -1, 0, 1, 1, 0, -1, 1, 0, 1, -1, 0, -1, -1])

/** 3D Simplex 噪声，返回约 [-1, 1] */
function createNoise3D(seed: number) {
  const rand = mulberry32(seed)
  const p = new Uint8Array(256)
  const perm = new Uint8Array(512)
  const pm = new Uint8Array(512)
  for (let i = 0; i < 256; i++) p[i] = i
  for (let i = 255; i > 0; i--) {
    const j = Math.floor(rand() * (i + 1))
    const t = p[i]
    p[i] = p[j]
    p[j] = t
  }
  for (let i = 0; i < 512; i++) {
    perm[i] = p[i & 255]
    pm[i] = perm[i] % 12
  }
  const F = 1 / 3
  const G = 1 / 6
  return (x: number, y: number, z: number): number => {
    const s = (x + y + z) * F
    const i = Math.floor(x + s)
    const j = Math.floor(y + s)
    const k = Math.floor(z + s)
    const t = (i + j + k) * G
    const x0 = x - (i - t)
    const y0 = y - (j - t)
    const z0 = z - (k - t)
    let i1: number, j1: number, k1: number, i2: number, j2: number, k2: number
    if (x0 >= y0) {
      if (y0 >= z0) { i1 = 1; j1 = 0; k1 = 0; i2 = 1; j2 = 1; k2 = 0 }
      else if (x0 >= z0) { i1 = 1; j1 = 0; k1 = 0; i2 = 1; j2 = 0; k2 = 1 }
      else { i1 = 0; j1 = 0; k1 = 1; i2 = 1; j2 = 0; k2 = 1 }
    } else {
      if (y0 < z0) { i1 = 0; j1 = 0; k1 = 1; i2 = 0; j2 = 1; k2 = 1 }
      else if (x0 < z0) { i1 = 0; j1 = 1; k1 = 0; i2 = 0; j2 = 1; k2 = 1 }
      else { i1 = 0; j1 = 1; k1 = 0; i2 = 1; j2 = 1; k2 = 0 }
    }
    const x1 = x0 - i1 + G, y1 = y0 - j1 + G, z1 = z0 - k1 + G
    const x2 = x0 - i2 + 2 * G, y2 = y0 - j2 + 2 * G, z2 = z0 - k2 + 2 * G
    const x3 = x0 - 1 + 3 * G, y3 = y0 - 1 + 3 * G, z3 = z0 - 1 + 3 * G
    const ii = i & 255, jj = j & 255, kk = k & 255
    let n = 0
    let c: number, g: number
    c = 0.6 - x0 * x0 - y0 * y0 - z0 * z0
    if (c > 0) { g = 3 * pm[ii + perm[jj + perm[kk]]]; c *= c; n += c * c * (GRAD[g] * x0 + GRAD[g + 1] * y0 + GRAD[g + 2] * z0) }
    c = 0.6 - x1 * x1 - y1 * y1 - z1 * z1
    if (c > 0) { g = 3 * pm[ii + i1 + perm[jj + j1 + perm[kk + k1]]]; c *= c; n += c * c * (GRAD[g] * x1 + GRAD[g + 1] * y1 + GRAD[g + 2] * z1) }
    c = 0.6 - x2 * x2 - y2 * y2 - z2 * z2
    if (c > 0) { g = 3 * pm[ii + i2 + perm[jj + j2 + perm[kk + k2]]]; c *= c; n += c * c * (GRAD[g] * x2 + GRAD[g + 1] * y2 + GRAD[g + 2] * z2) }
    c = 0.6 - x3 * x3 - y3 * y3 - z3 * z3
    if (c > 0) { g = 3 * pm[ii + 1 + perm[jj + 1 + perm[kk + 1]]]; c *= c; n += c * c * (GRAD[g] * x3 + GRAD[g + 1] * y3 + GRAD[g + 2] * z3) }
    return 32 * n
  }
}

/** 柔光强度场：坐标以重心为原点、单位约为画布短边，y 向下 */
function createGlowField(seed: number) {
  const noise = createNoise3D(seed)
  return (x: number, y: number, t: number): number => {
    const m = 1 - smoothstep(0.3, 1.0, Math.hypot(x * 0.55, y * 1.35))
    if (m <= 0) return 0
    const wx = x + 0.3 * noise(x * 0.9, y * 0.9, t * 0.06)
    const wy = y + 0.3 * noise(x * 0.9 + 5, y * 0.9, t * 0.06)
    const n =
      0.7 * noise(wx * 1.4, wy * 1.4, t * 0.09) +
      0.3 * noise(wx * 3 + 3, wy * 3, t * 0.13) +
      0.3 * Math.exp((-y * y) / 0.07) // 重心带：锚点所在高度最浓
    const top = 1 - 0.65 * smoothstep(-0.15, -0.6, y) // 越往上越淡
    return top * Math.pow(smoothstep(-0.15, 0.5, n), 1.4) * m
  }
}

/* 全局共享一个 rAF 循环，多块画布同屏时只调度一次 */
const subscribers = new Set<(now: number) => void>()
let loopId = 0
function loop(now: number) {
  loopId = 0
  subscribers.forEach((fn) => fn(now))
  if (subscribers.size) loopId = requestAnimationFrame(loop)
}
function subscribe(fn: (now: number) => void) {
  subscribers.add(fn)
  if (!loopId) loopId = requestAnimationFrame(loop)
  return () => {
    subscribers.delete(fn)
    if (!subscribers.size && loopId) {
      cancelAnimationFrame(loopId)
      loopId = 0
    }
  }
}

export class GlowDotsRenderer {
  private readonly canvas: HTMLCanvasElement
  private readonly ctx: CanvasRenderingContext2D
  private readonly o: Required<Omit<GlowDotsOptions, 'onReady'>> & Pick<GlowDotsOptions, 'onReady'>
  private readonly field: (x: number, y: number, t: number) => number
  private readonly reduced: boolean
  private readonly interactive: boolean
  private readonly mouse = { tu: 0, tv: 0, u: 0, v: 0, inside: false, strength: 0, speed: 0 }
  private palette: Palette = GLOW_PALETTES.midnight
  private W = 0
  private H = 0
  private cell = 0
  private cols = 0
  private rows = 0
  private unit = 1
  private ox = 0
  private oy = 0
  private U = new Float32Array(0)
  private V = new Float32Array(0)
  private L = new Uint8Array(0)
  private grid: HTMLCanvasElement | null = null
  private time = 0
  private last = 0
  private ready = false
  private unsub: (() => void) | null = null
  private io: IntersectionObserver | null = null
  private ro: ResizeObserver | null = null
  private resizeRaf = 0
  private readonly onMove = (e: PointerEvent) => this.handleMove(e)
  private readonly onOut = (e: PointerEvent) => {
    if (!e.relatedTarget) this.mouse.inside = false
  }

  /** 浏览器不支持 Canvas 2D（如测试环境）时返回 null */
  static create(canvas: HTMLCanvasElement, options: GlowDotsOptions = {}): GlowDotsRenderer | null {
    // jsdom 没有 Canvas 实现，调用 getContext 只会刷错误日志
    if (typeof navigator !== 'undefined' && /jsdom/i.test(navigator.userAgent)) return null
    let ctx: CanvasRenderingContext2D | null = null
    try {
      ctx = canvas.getContext('2d')
    } catch {
      ctx = null
    }
    return ctx ? new GlowDotsRenderer(canvas, ctx, options) : null
  }

  private constructor(canvas: HTMLCanvasElement, ctx: CanvasRenderingContext2D, options: GlowDotsOptions) {
    this.canvas = canvas
    this.ctx = ctx
    this.o = {
      centerX: 0.58,
      centerY: 0.6,
      mobileCenterY: 0.52,
      scale: 1,
      speed: 1.3,
      cell: 5,
      mobileCell: 4,
      maxCells: 30000,
      interactive: true,
      seed: 31,
      startTime: 46,
      gain: 1,
      ...options,
    }
    this.field = createGlowField(this.o.seed)
    this.time = this.o.startTime
    const mq = (q: string) => typeof window.matchMedia === 'function' && window.matchMedia(q).matches
    this.reduced = mq('(prefers-reduced-motion: reduce)')
    this.interactive = this.o.interactive && mq('(pointer: fine)') && !this.reduced

    if (typeof IntersectionObserver !== 'undefined') {
      this.io = new IntersectionObserver(
        ([entry]) => (entry?.isIntersecting ? this.start() : this.stop()),
        { rootMargin: '120px 0px' },
      )
      this.io.observe(canvas)
    } else {
      this.start()
    }
    if (typeof ResizeObserver !== 'undefined') {
      this.ro = new ResizeObserver(() => {
        cancelAnimationFrame(this.resizeRaf)
        this.resizeRaf = requestAnimationFrame(() => this.layout())
      })
      this.ro.observe(canvas)
    }
    if (this.interactive) {
      window.addEventListener('pointermove', this.onMove, { passive: true })
      window.addEventListener('pointerout', this.onOut)
    }
    this.layout()
  }

  setTheme(theme: GlowTheme) {
    const next = GLOW_PALETTES[theme] ?? GLOW_PALETTES.midnight
    if (next === this.palette && this.grid) return
    this.palette = next
    this.paintGrid()
    this.draw()
  }

  destroy() {
    this.stop()
    this.io?.disconnect()
    this.ro?.disconnect()
    cancelAnimationFrame(this.resizeRaf)
    window.removeEventListener('pointermove', this.onMove)
    window.removeEventListener('pointerout', this.onOut)
  }

  private start() {
    if (this.unsub || this.reduced) return
    this.last = 0
    this.unsub = subscribe((now) => this.tick(now))
  }

  private stop() {
    this.unsub?.()
    this.unsub = null
  }

  private layout() {
    const rect = this.canvas.getBoundingClientRect()
    const w = Math.max(1, rect.width)
    const h = Math.max(1, rect.height)
    const dpr = Math.min(window.devicePixelRatio || 1, 2)
    const mobile = w < 640
    let c = mobile ? this.o.mobileCell : this.o.cell
    while ((w / c) * (h / c) > this.o.maxCells) c += 0.5
    this.cell = Math.max(2, Math.round(c * dpr))
    this.W = this.canvas.width = Math.round(w * dpr)
    this.H = this.canvas.height = Math.round(h * dpr)
    this.cols = Math.ceil(this.W / this.cell)
    this.rows = Math.ceil(this.H / this.cell)
    this.unit = Math.max(Math.min(this.W, this.H), this.W * 0.55) * this.o.scale * (mobile ? 1.6 : 1)
    this.ox = this.W * this.o.centerX
    this.oy = this.H * (mobile ? this.o.mobileCenterY : this.o.centerY)
    const n = this.cols * this.rows
    this.U = new Float32Array(n)
    this.V = new Float32Array(n)
    this.L = new Uint8Array(n)
    for (let y = 0, i = 0; y < this.rows; y++) {
      const v = ((y + 0.5) * this.cell - this.oy) / this.unit
      for (let x = 0; x < this.cols; x++, i++) {
        this.U[i] = ((x + 0.5) * this.cell - this.ox) / this.unit
        this.V[i] = v
      }
    }
    this.paintGrid()
    this.draw()
  }

  private paintGrid() {
    if (!this.W || !this.H) return
    const g = document.createElement('canvas')
    g.width = this.W
    g.height = this.H
    const gx = g.getContext('2d')
    if (!gx) return
    const r = Math.max(0.6, 0.08 * this.cell)
    gx.fillStyle = this.palette.grid
    gx.beginPath()
    for (let y = 0; y < this.rows; y++) {
      for (let x = 0; x < this.cols; x++) {
        const cx = (x + 0.5) * this.cell
        const cy = (y + 0.5) * this.cell
        gx.moveTo(cx + r, cy)
        gx.arc(cx, cy, r, 0, TAU)
      }
    }
    gx.fill()
    this.grid = g
  }

  private draw() {
    if (!this.cols) return
    const { mouse, ctx, cell } = this
    const warp = this.interactive && mouse.strength > 0.002
    for (let y = 0, i = 0; y < this.rows; y++) {
      for (let x = 0; x < this.cols; x++, i++) {
        let u = this.U[i]
        let v = this.V[i]
        let k = 0
        if (warp) {
          // 光标处：光被搅动成漩涡并向外推开
          const dx = u - mouse.u
          const dy = v - mouse.v
          const d2 = dx * dx + dy * dy
          if (d2 < 0.14) {
            k = Math.exp(-d2 / 0.022) * mouse.strength
            const swirl = k * (0.9 + mouse.speed * 2.5)
            u += -dy * swirl - dx * k * 0.45
            v += dx * swirl - dy * k * 0.45
          }
        }
        let s = this.field(u, v, this.time) * this.o.gain
        if (k > 0) {
          s = 1 - (1 - s) * (1 - 0.2 * k) // 周围一圈柔光
          s = Math.min(1, s + 0.55 * k * s) // 原有光团提亮
        }
        this.L[i] = toLevel(s, x, y)
      }
    }
    ctx.clearRect(0, 0, this.W, this.H)
    if (this.grid) ctx.drawImage(this.grid, 0, 0)
    for (let lv = 1; lv <= 6; lv++) {
      const r = (cell * SIZES[lv - 1]) / 2
      let any = false
      ctx.beginPath()
      for (let y = 0, i = 0; y < this.rows; y++) {
        const cy = (y + 0.5) * cell
        for (let x = 0; x < this.cols; x++, i++) {
          if (this.L[i] === lv) {
            const cx = (x + 0.5) * cell
            ctx.moveTo(cx + r, cy)
            ctx.arc(cx, cy, r, 0, TAU)
            any = true
          }
        }
      }
      if (any) {
        ctx.fillStyle = this.palette.levels[lv - 1]
        ctx.fill()
      }
    }
    if (!this.ready) {
      this.ready = true
      this.o.onReady?.()
    }
  }

  private tick(now: number) {
    if (this.last && now - this.last < 1000 / 60 - 2) return
    const dt = this.last ? Math.min((now - this.last) / 1000, 0.05) : 1 / 60
    this.last = now
    this.time += dt * this.o.speed
    if (this.interactive) {
      const m = this.mouse
      const du = (m.tu - m.u) * 0.12
      const dv = (m.tv - m.v) * 0.12
      m.u += du
      m.v += dv
      m.speed += (Math.min(1, Math.hypot(du, dv) * 25) - m.speed) * 0.1 // 移动越快，漩涡越强
      m.strength += (+m.inside - m.strength) * 0.08
    }
    this.draw()
  }

  private handleMove(e: PointerEvent) {
    const r = this.canvas.getBoundingClientRect()
    const x = e.clientX - r.left
    const y = e.clientY - r.top
    const k = this.W / Math.max(1, r.width)
    const m = this.mouse
    m.inside = x >= 0 && y >= 0 && x <= r.width && y <= r.height
    m.tu = (x * k - this.ox) / this.unit
    m.tv = (y * k - this.oy) / this.unit
    if (m.strength < 0.01) {
      m.u = m.tu
      m.v = m.tv
    }
  }
}
