import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { useFirstCallWatcher } from '../useFirstCallWatcher'

const list = vi.hoisted(() => vi.fn())
vi.mock('@/api/usage', () => ({ usageAPI: { list } }))

function setup(keyId: number | null) {
  const active = ref(true)
  const key = ref<number | null>(keyId)
  const onDetected = vi.fn()
  const wrapper = mount(
    defineComponent({
      setup() {
        useFirstCallWatcher({ active, keyId: key, onDetected })
        return () => null
      }
    })
  )
  return { active, key, onDetected, wrapper }
}

async function tick(ms = 5000) {
  await vi.advanceTimersByTimeAsync(ms)
}

describe('useFirstCallWatcher', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    list.mockReset()
    Object.defineProperty(document, 'hidden', { value: false, configurable: true })
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('fires once as soon as the created key has a usage record', async () => {
    list.mockResolvedValueOnce({ total: 0 }).mockResolvedValue({ total: 1 })
    const { onDetected, wrapper } = setup(42)
    await tick(0)
    expect(list).toHaveBeenCalledWith(1, 1, 42)
    expect(onDetected).not.toHaveBeenCalled()
    await tick()
    expect(onDetected).toHaveBeenCalledTimes(1)
    await tick(20000)
    expect(onDetected).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('uses the initial total as a baseline when the key is unknown', async () => {
    list.mockResolvedValueOnce({ total: 3 }).mockResolvedValueOnce({ total: 3 }).mockResolvedValue({ total: 4 })
    const { onDetected, wrapper } = setup(null)
    await tick(0)
    expect(list).toHaveBeenCalledWith(1, 1, undefined)
    await tick()
    expect(onDetected).not.toHaveBeenCalled()
    await tick()
    expect(onDetected).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('does nothing while inactive and stops when deactivated', async () => {
    list.mockResolvedValue({ total: 0 })
    const { active, wrapper } = setup(7)
    await tick(0)
    const calls = list.mock.calls.length
    active.value = false
    await nextTick()
    await tick(30000)
    expect(list.mock.calls.length).toBe(calls)
    wrapper.unmount()
  })

  it('skips polling while the tab is hidden and checks right away when it comes back', async () => {
    list.mockResolvedValue({ total: 1 })
    Object.defineProperty(document, 'hidden', { value: true, configurable: true })
    const { onDetected, wrapper } = setup(9)
    await tick(15000)
    expect(list).not.toHaveBeenCalled()

    Object.defineProperty(document, 'hidden', { value: false, configurable: true })
    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true })
    document.dispatchEvent(new Event('visibilitychange'))
    await tick(0)
    expect(onDetected).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('survives a failed request and keeps polling', async () => {
    list.mockRejectedValueOnce(new Error('network')).mockResolvedValue({ total: 2 })
    const { onDetected, wrapper } = setup(5)
    await tick(0)
    expect(onDetected).not.toHaveBeenCalled()
    await tick()
    expect(onDetected).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})
