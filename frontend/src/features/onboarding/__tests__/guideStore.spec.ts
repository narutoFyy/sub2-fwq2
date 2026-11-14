import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useGuideStore } from '../guideStore'

describe('guide store', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('starts idle for an account that has never seen the guide', () => {
    const guide = useGuideStore()
    guide.load(7)
    expect(guide.status).toBe('idle')
    expect(guide.active).toBe(false)
  })

  it('persists progress per account and restores it after a reload', () => {
    const guide = useGuideStore()
    guide.load(7)
    guide.start()
    guide.chooseClient('codex')
    guide.go('redeem')

    setActivePinia(createPinia())
    const reloaded = useGuideStore()
    reloaded.load(7)
    expect(reloaded.active).toBe(true)
    expect(reloaded.step).toBe('redeem')
    expect(reloaded.client).toBe('codex')

    reloaded.load(8)
    expect(reloaded.status).toBe('idle')
    expect(reloaded.client).toBeNull()
  })

  it('remembers skip, dismiss and finish so they never auto-start again', () => {
    const guide = useGuideStore()
    for (const action of ['skip', 'dismiss', 'finish'] as const) {
      guide.load(9)
      guide.start()
      guide[action]()
      const status = guide.status
      expect(status).not.toBe('active')
      guide.load(null)
      guide.load(9)
      expect(guide.status).toBe(status)
    }
  })

  it('ignores corrupted storage', () => {
    localStorage.setItem('onboarding_guide_v1:5', '{not json')
    const guide = useGuideStore()
    guide.load(5)
    expect(guide.status).toBe('idle')
    expect(guide.step).toBe('welcome')
  })

  it('remembers the key created in the guide across reloads', () => {
    const guide = useGuideStore()
    guide.load(11)
    guide.start()
    guide.notifyKeyCreated(42)
    guide.notifyKeyCreated() // 没带 id 的通知不应清掉已记住的 key

    setActivePinia(createPinia())
    const reloaded = useGuideStore()
    reloaded.load(11)
    expect(reloaded.keyId).toBe(42)
    expect(reloaded.keyCreatedTick).toBe(0)

    reloaded.start() // 重新开始引导时丢掉旧的 key
    expect(reloaded.keyId).toBeNull()
  })

  it('bumps event counters for page notifications', () => {
    const guide = useGuideStore()
    guide.notifyRedeemed()
    guide.notifyKeyCreated()
    guide.notifyKeyCreated()
    expect(guide.redeemedTick).toBe(1)
    expect(guide.keyCreatedTick).toBe(2)
  })
})
