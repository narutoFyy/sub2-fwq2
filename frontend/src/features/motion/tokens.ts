/**
 * 全站动效的时长与缓动，所有动效从这里取值，不在组件里各写各的。
 * inOut 与按钮翻滚、主题切换用的是同一条曲线。
 */
type Bezier = [number, number, number, number]

export const DURATION = {
  fast: 0.2,
  base: 0.4,
  slow: 0.7,
  count: 1.2,
} as const

export const EASE: { out: Bezier; inOut: Bezier; expoOut: Bezier } = {
  out: [0.22, 1, 0.36, 1],
  inOut: [0.65, 0, 0.25, 1],
  expoOut: [0.16, 1, 0.3, 1],
}

/** 一组元素依次出场时，相邻两个的间隔（秒） */
export const STAGGER = 0.07

export const SPRING = {
  /** 磁吸按钮：跟手、松开时带一点回弹 */
  magnetic: { type: 'spring' as const, stiffness: 240, damping: 15, mass: 0.6 },
}
