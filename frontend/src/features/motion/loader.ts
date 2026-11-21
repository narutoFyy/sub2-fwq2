/**
 * 动效引擎按需加载：页面先渲染，引擎在后台下载，不占首屏的包。
 */
type Engine = typeof import('./engine')

let pending: Promise<Engine> | null = null

export function loadMotion(): Promise<Engine> {
  if (!pending) {
    pending = import('./engine').catch((err) => {
      pending = null
      throw err
    })
  }
  return pending
}
