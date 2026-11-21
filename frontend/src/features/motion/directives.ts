import type { Directive } from 'vue'
import { canAnimate, hasFinePointer } from './env'
import { hideForEntrance, playEntrance } from './entrance'
import { loadMotion } from './loader'
import { SPRING } from './tokens'

const cleanups = new WeakMap<HTMLElement, () => void>()

function teardown(el: HTMLElement) {
  cleanups.get(el)?.()
  cleanups.delete(el)
}

/**
 * v-reveal：区块滚动进入视口时，里面标了 data-reveal 的元素依次出场（没标就整块出场）。
 * 绑定值是额外延迟（秒）。只播一次。
 */
export const vReveal: Directive<HTMLElement, number | undefined> = {
  mounted(el, binding) {
    if (!canAnimate()) return
    const marked = Array.from(el.querySelectorAll<HTMLElement>('[data-reveal]'))
    const items = marked.length ? marked : [el]
    hideForEntrance(items)
    let alive = true
    let stopView: VoidFunction | undefined
    cleanups.set(el, () => {
      alive = false
      stopView?.()
    })
    loadMotion()
      .then(({ inView }) => {
        if (!alive) return
        stopView = inView(
          el,
          () => {
            stopView?.()
            void playEntrance(items, { delay: binding.value ?? 0 })
          },
          { amount: 0.15, margin: '0px 0px -8% 0px' },
        )
      })
      .catch(() => void playEntrance(items))
  },
  unmounted: teardown,
}

/**
 * v-spotlight：光标所在处亮起一圈柔光（样式见 motion.css 的 [data-spotlight]）。
 * 只记录光标坐标到 CSS 变量，不用动效引擎；手机上不启用。
 */
export const vSpotlight: Directive<HTMLElement> = {
  mounted(el) {
    if (!hasFinePointer()) return
    let raf = 0
    let x = 0
    let y = 0
    const onMove = (e: PointerEvent) => {
      const rect = el.getBoundingClientRect()
      x = e.clientX - rect.left
      y = e.clientY - rect.top
      if (raf) return
      raf = requestAnimationFrame(() => {
        raf = 0
        el.style.setProperty('--spot-x', `${x}px`)
        el.style.setProperty('--spot-y', `${y}px`)
      })
    }
    el.setAttribute('data-spotlight', '')
    el.addEventListener('pointermove', onMove, { passive: true })
    cleanups.set(el, () => {
      cancelAnimationFrame(raf)
      el.removeEventListener('pointermove', onMove)
    })
  },
  unmounted: teardown,
}

/**
 * v-magnetic：光标在按钮上时按钮朝光标轻轻偏移，移开后弹回原位。
 * 用 CSS translate 属性而不是 transform，不和按钮自己的按下缩放打架。绑定值是跟随比例（默认 0.3）。
 */
export const vMagnetic: Directive<HTMLElement, number | undefined> = {
  mounted(el, binding) {
    if (!canAnimate() || !hasFinePointer()) return
    const ratio = binding.value ?? 0.3
    const limit = 10
    const pos = { x: 0, y: 0 }
    let running: { stop: () => void } | undefined
    let raf = 0
    let target = { x: 0, y: 0 }

    const apply = () => {
      el.style.translate = `${pos.x.toFixed(2)}px ${pos.y.toFixed(2)}px`
    }
    const springTo = (tx: number, ty: number) => {
      loadMotion()
        .then(({ animate }) => {
          running?.stop()
          running = animate(pos, { x: tx, y: ty }, { ...SPRING.magnetic, onUpdate: apply })
        })
        .catch(() => undefined)
    }
    const clamp = (v: number) => Math.max(-limit, Math.min(limit, v))
    const onMove = (e: PointerEvent) => {
      const rect = el.getBoundingClientRect()
      target = {
        x: clamp((e.clientX - (rect.left + rect.width / 2)) * ratio),
        y: clamp((e.clientY - (rect.top + rect.height / 2)) * ratio),
      }
      if (raf) return
      raf = requestAnimationFrame(() => {
        raf = 0
        springTo(target.x, target.y)
      })
    }
    const onLeave = () => {
      cancelAnimationFrame(raf)
      raf = 0
      springTo(0, 0)
    }
    el.addEventListener('pointermove', onMove, { passive: true })
    el.addEventListener('pointerleave', onLeave)
    cleanups.set(el, () => {
      cancelAnimationFrame(raf)
      running?.stop()
      el.removeEventListener('pointermove', onMove)
      el.removeEventListener('pointerleave', onLeave)
      el.style.translate = ''
    })
  },
  unmounted: teardown,
}
