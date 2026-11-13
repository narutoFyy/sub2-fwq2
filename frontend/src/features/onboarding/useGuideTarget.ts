import { onBeforeUnmount, ref, shallowRef, watch, type Ref } from 'vue'

export interface TargetRect {
  x: number
  y: number
  w: number
  h: number
}

/** 每个候选目标是一个返回元素的函数，按顺序取第一个「真的看得见」的。 */
export type TargetResolver = () => HTMLElement | null

const round = (n: number) => Math.round(n * 2) / 2

function safe(resolver: TargetResolver): HTMLElement | null {
  try {
    return resolver()
  } catch {
    return null
  }
}

/** 有尺寸，且至少有一部分在视口水平范围内（手机上收起的侧栏在屏幕外，不算可见）。 */
function isVisible(el: HTMLElement): boolean {
  const r = el.getBoundingClientRect()
  return r.width > 0 && r.height > 0 && r.right > 0 && r.left < window.innerWidth
}

/**
 * 持续跟踪引导目标的位置（侧栏展开、弹窗动画、滚动、窗口缩放都会移动它）。
 * 短暂找不到（路由切换中）时保留上一次的位置，避免聚光灯闪烁。
 */
export function useGuideTarget(resolvers: Ref<TargetResolver[]>, enabled: Ref<boolean>) {
  const rect = shallowRef<TargetRect | null>(null)
  /** 卡片贴靠的区域：目标在弹窗里时是整个弹窗面板，否则就是目标本身。 */
  const anchorRect = shallowRef<TargetRect | null>(null)
  const inDialog = ref(false)
  const el = shallowRef<HTMLElement | null>(null)
  /** 命中的是第几个候选，-1 = 都没找到。 */
  const index = ref(-1)
  /** 超过 1.5 秒都没找到：界面上给「直接前往」兜底按钮。 */
  const missing = ref(false)
  let raf = 0
  let lostAt = 0

  function scan(): { e: HTMLElement; i: number } | null {
    const list = resolvers.value
    for (let i = 0; i < list.length; i++) {
      const e = safe(list[i]!)
      if (e && isVisible(e)) return { e, i }
    }
    return null
  }

  function bringIntoView(target: HTMLElement) {
    const r = target.getBoundingClientRect()
    if (r.top < 0 || r.bottom > window.innerHeight) {
      target.scrollIntoView({ block: 'center', behavior: 'smooth' })
    }
  }

  function tick() {
    const hit = scan()
    const t = performance.now()
    if (hit) {
      lostAt = 0
      missing.value = false
      if (hit.e !== el.value) {
        el.value = hit.e
        bringIntoView(hit.e)
      }
      index.value = hit.i
      const r = hit.e.getBoundingClientRect()
      const next = { x: round(r.left), y: round(r.top), w: round(r.width), h: round(r.height) }
      const cur = rect.value
      if (!cur || cur.x !== next.x || cur.y !== next.y || cur.w !== next.w || cur.h !== next.h) rect.value = next

      const panel = hit.e.closest<HTMLElement>('.modal-content')
      inDialog.value = !!panel
      const ar = (panel ?? hit.e).getBoundingClientRect()
      const aNext = { x: round(ar.left), y: round(ar.top), w: round(ar.width), h: round(ar.height) }
      const aCur = anchorRect.value
      if (!aCur || aCur.x !== aNext.x || aCur.y !== aNext.y || aCur.w !== aNext.w || aCur.h !== aNext.h) anchorRect.value = aNext
    } else {
      if (!lostAt) lostAt = t
      if (t - lostAt > 350 && (rect.value || el.value)) {
        rect.value = null
        anchorRect.value = null
        inDialog.value = false
        el.value = null
        index.value = -1
      }
      if (t - lostAt > 1500) missing.value = true
    }
    raf = requestAnimationFrame(tick)
  }

  watch(
    [enabled, resolvers],
    ([on]) => {
      cancelAnimationFrame(raf)
      lostAt = 0
      missing.value = false
      if (on) {
        raf = requestAnimationFrame(tick)
      } else {
        rect.value = null
        anchorRect.value = null
        inDialog.value = false
        el.value = null
        index.value = -1
      }
    },
    { immediate: true }
  )

  onBeforeUnmount(() => cancelAnimationFrame(raf))

  return { rect, anchorRect, inDialog, el, index, missing }
}
