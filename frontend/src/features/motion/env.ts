/**
 * 动效运行环境判断。
 * 测试环境（jsdom）、不支持 IntersectionObserver 的浏览器、开了「减少动态效果」的用户，一律直接显示最终状态。
 */
function media(query: string): boolean {
  return typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia(query).matches
}

export function canAnimate(): boolean {
  if (typeof window === 'undefined' || typeof navigator === 'undefined') return false
  if (/jsdom/i.test(navigator.userAgent)) return false
  if (typeof IntersectionObserver === 'undefined') return false
  return !media('(prefers-reduced-motion: reduce)')
}

/** 有鼠标这类精确指针、能悬停的设备（手机和平板上不做跟随光标的效果） */
export function hasFinePointer(): boolean {
  return media('(hover: hover) and (pointer: fine)')
}
