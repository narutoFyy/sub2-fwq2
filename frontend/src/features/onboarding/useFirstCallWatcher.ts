import { onBeforeUnmount, watch, type Ref } from 'vue'
import { usageAPI } from '@/api/usage'

const INTERVAL_MS = 5000

/**
 * 等用户的第一次调用：active 期间每 5 秒查一次使用记录，查到就回调一次。
 * - 知道引导里创建的那把 Key：它有任何记录就算；
 * - 不知道（老用户重走引导）：以进入时的记录总数为基线，之后增加才算，避免被历史记录误触发。
 * 标签页在后台时不查，切回来立刻查一次——用户装好工具、去别的窗口试用，回来就能看到结果。
 */
export function useFirstCallWatcher(options: { active: Ref<boolean>; keyId: Ref<number | null>; onDetected: () => void }) {
  let timer: ReturnType<typeof setTimeout> | undefined
  let baseline: number | null = null
  let token = 0

  async function check(current: number) {
    if (current !== token || document.hidden) return
    try {
      const keyId = options.keyId.value
      const { total } = await usageAPI.list(1, 1, keyId ?? undefined)
      if (current !== token) return
      if (keyId != null ? total > 0 : baseline !== null && total > baseline) {
        stop()
        options.onDetected()
        return
      }
      if (keyId == null && baseline === null) baseline = total
    } catch {
      /* 偶发失败下一轮再查 */
    }
  }

  function schedule(current: number) {
    timer = setTimeout(async () => {
      await check(current)
      if (current === token) schedule(current)
    }, INTERVAL_MS)
  }

  function start() {
    stop()
    const current = ++token
    baseline = null
    void check(current)
    schedule(current)
  }

  function stop() {
    token++
    clearTimeout(timer)
    timer = undefined
  }

  function onVisible() {
    if (document.visibilityState === 'visible' && options.active.value) void check(token)
  }

  watch(
    options.active,
    (on) => {
      if (on) start()
      else stop()
    },
    { immediate: true }
  )
  document.addEventListener('visibilitychange', onVisible)
  onBeforeUnmount(() => {
    stop()
    document.removeEventListener('visibilitychange', onVisible)
  })
}
