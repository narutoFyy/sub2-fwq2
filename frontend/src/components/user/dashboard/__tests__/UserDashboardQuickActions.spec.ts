import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'
import UserDashboardQuickActions from '../UserDashboardQuickActions.vue'
import { useGuideStore } from '@/features/onboarding/guideStore'

const auth = vi.hoisted(() => ({ isAuthenticated: true, isAdmin: false }))

vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('@/composables/useBatchImageAccess', () => ({
  useBatchImageAccess: () => ({ canUseBatchImage: ref(false), refreshBatchImageAccess: vi.fn() })
}))
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key, locale: ref('zh') }) }
})

function mountActions() {
  return mount(UserDashboardQuickActions, { global: { stubs: { Icon: { template: '<span />' } } } })
}

describe('dashboard quick actions: reopen the guide', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    auth.isAuthenticated = true
    auth.isAdmin = false
    useGuideStore().load(1)
  })

  it('offers the guide again after it was skipped, and restarts it on click', async () => {
    const guide = useGuideStore()
    guide.start()
    guide.skip()
    const wrapper = mountActions()
    const entry = wrapper.get('[data-testid="reopen-guide"]')
    expect(entry.text()).toContain('新手引导')

    await entry.trigger('click')
    expect(guide.status).toBe('active')
    expect(guide.step).toBe('welcome')
    expect(wrapper.find('[data-testid="reopen-guide"]').exists()).toBe(false)
  })

  it('is hidden for administrators', () => {
    auth.isAdmin = true
    expect(mountActions().find('[data-testid="reopen-guide"]').exists()).toBe(false)
  })
})
