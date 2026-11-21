import { loadMotion } from './loader'
import { DURATION, EASE, STAGGER } from './tokens'

/** 出场前先藏起来（行内样式，挂载时同步设置，首帧就不会闪一下） */
export function hideForEntrance(els: HTMLElement[]): void {
  for (const el of els) el.style.opacity = '0'
}

function clearEntrance(els: HTMLElement[]): void {
  for (const el of els) {
    el.style.opacity = ''
    el.style.transform = ''
    el.style.filter = ''
  }
}

/**
 * 一组元素依次带模糊上浮出现。
 * 结束后清掉行内样式，元素原来的 transform（按钮按下缩放、悬停效果）照常生效。
 * 引擎加载失败时直接显示，绝不让内容一直藏着。
 */
export async function playEntrance(els: HTMLElement[], options: { delay?: number; distance?: number } = {}): Promise<void> {
  if (!els.length) return
  try {
    const { animate, stagger } = await loadMotion()
    await animate(
      els,
      {
        opacity: [0, 1],
        y: [options.distance ?? 14, 0],
        filter: ['blur(6px)', 'blur(0px)'],
      },
      { duration: DURATION.slow, ease: EASE.out, delay: stagger(STAGGER, { startDelay: options.delay ?? 0 }) },
    ).finished
    // 引擎在结束后的下一帧才把终态写成行内样式，等它写完再清，否则会残留 transform: none
    await new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve())))
  } catch {
    /* 引擎没加载出来就直接显示 */
  }
  clearEntrance(els)
}
