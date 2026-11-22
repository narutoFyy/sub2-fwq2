import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import type { UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useAffiliateSummaryStore } from '@/stores/affiliateSummary'
import DashboardAffiliateBanner from '../DashboardAffiliateBanner.vue'

const { getAffiliateDetailMock, transferAffiliateQuotaMock } = vi.hoisted(() => ({
  getAffiliateDetailMock: vi.fn(),
  transferAffiliateQuotaMock: vi.fn(),
}))

vi.mock('@/api/user', () => ({
  default: {
    getAffiliateDetail: getAffiliateDetailMock,
    transferAffiliateQuota: transferAffiliateQuotaMock,
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key,
    }),
  }
})

function makeDetail(overrides: Partial<UserAffiliateDetail> = {}): UserAffiliateDetail {
  return {
    user_id: 10,
    aff_code: 'INVITE10',
    aff_count: 1,
    aff_quota: 0,
    aff_frozen_quota: 0,
    aff_history_quota: 0,
    effective_rebate_rate_percent: 5,
    qualified_invitee_count: 0,
    pending_step_bonus_total: 0,
    granted_step_bonus_total: 0,
    pending_ladder_bonus_total: 0,
    granted_ladder_bonus_total: 0,
    withdraw_threshold: 300,
    remind_consume_count: 0,
    ladder_tiers: [
      { tier_target: 5, cumulative_bonus: 10, delta_bonus: 10, requires_min_total_paid: 0, auto_grant: true, status: 'locked' },
    ],
    invitees: [],
    ...overrides,
  }
}

function setupStores(options: { isAdmin?: boolean; isSimpleMode?: boolean; affiliateEnabled?: boolean } = {}) {
  setActivePinia(createPinia())
  const authStore = useAuthStore()
  const appStore = useAppStore()
  authStore.token = 'tok'
  authStore.user = {
    id: 10,
    email: 'u@qq.com',
    username: 'u',
    role: options.isAdmin ? 'admin' : 'user',
    balance: 10,
    concurrency: 2,
    status: 'active',
    allowed_groups: [],
    created_at: '2026-11-01T00:00:00Z',
    updated_at: '2026-11-01T00:00:00Z',
  }
  authStore.refreshUser = vi.fn().mockResolvedValue(authStore.user)
  appStore.cachedPublicSettings = {
    site_name: 'Sub2API',
    site_logo: '',
    site_subtitle: '',
    api_base_url: '',
    contact_info: '',
    doc_url: '',
    home_content: '',
    hide_ccs_import_button: false,
    purchase_subscription_enabled: false,
    purchase_subscription_url: '',
    custom_menu_items: [],
    registration_enabled: true,
    email_verify_enabled: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: false,
    password_reset_enabled: false,
    frontend_mode: options.isSimpleMode ? 'simple' : 'standard',
    turnstile_enabled: false,
    turnstile_site_key: '',
    linuxdo_oauth_enabled: false,
    oidc_oauth_enabled: false,
    oidc_oauth_provider_name: '',
    github_oauth_enabled: false,
    google_oauth_enabled: false,
    version: '1.0.0',
    affiliate_enabled: options.affiliateEnabled ?? true,
  } as unknown as typeof appStore.cachedPublicSettings
  return { authStore, appStore, summaryStore: useAffiliateSummaryStore() }
}

function mountBanner() {
  return mount(DashboardAffiliateBanner, {
    global: {
      stubs: {
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
        Icon: true,
        AffiliatePosterDialog: true,
      },
    },
  })
}

describe('DashboardAffiliateBanner', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('does not render for admins or when affiliate feature is disabled', async () => {
    getAffiliateDetailMock.mockResolvedValue(makeDetail())

    setupStores({ isAdmin: true, affiliateEnabled: true })
    const adminWrapper = mountBanner()
    await flushPromises()
    expect(adminWrapper.find('[data-testid="dashboard-affiliate-banner"]').exists()).toBe(false)
    expect(getAffiliateDetailMock).not.toHaveBeenCalled()

    setupStores({ isAdmin: false, affiliateEnabled: false })
    const disabledWrapper = mountBanner()
    await flushPromises()
    expect(disabledWrapper.find('[data-testid="dashboard-affiliate-banner"]').exists()).toBe(false)
    expect(getAffiliateDetailMock).not.toHaveBeenCalled()
  })

  it('renders P1 consume reminder when invitees need consumption unlock', async () => {
    getAffiliateDetailMock.mockResolvedValue(
      makeDetail({ remind_consume_count: 2, pending_step_bonus_total: 6, aff_quota: 50 }),
    )
    setupStores()

    const wrapper = mountBanner()
    await flushPromises()

    const banner = wrapper.get('[data-testid="dashboard-affiliate-banner"]')
    expect(banner.attributes('data-banner-type')).toBe('consume_reminder')
    expect(banner.find('[data-testid="banner-copy-remind"]').exists()).toBe(true)
  })

  it('renders P2 transferable banner and transfers quota directly', async () => {
    getAffiliateDetailMock.mockResolvedValue(makeDetail({ aff_quota: 25 }))
    transferAffiliateQuotaMock.mockResolvedValue({ transferred_quota: 25, balance: 35 })
    const { appStore } = setupStores()
    const showSuccessSpy = vi.spyOn(appStore, 'showSuccess')

    const wrapper = mountBanner()
    await flushPromises()

    const banner = wrapper.get('[data-testid="dashboard-affiliate-banner"]')
    expect(banner.attributes('data-banner-type')).toBe('transferable')

    await banner.get('[data-testid="banner-transfer-btn"]').trigger('click')
    await flushPromises()

    expect(transferAffiliateQuotaMock).toHaveBeenCalledTimes(1)
    expect(showSuccessSpy).toHaveBeenCalledWith(expect.stringContaining('25.00 元'))
  })

  it('renders P5 default promo and persists 7-day dismissal on close', async () => {
    getAffiliateDetailMock.mockResolvedValue(makeDetail())
    setupStores()

    const wrapper = mountBanner()
    await flushPromises()

    expect(wrapper.get('[data-testid="dashboard-affiliate-banner"]').attributes('data-banner-type')).toBe('default_promo')
    await wrapper.get('[data-testid="banner-dismiss-default"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="dashboard-affiliate-banner"]').exists()).toBe(false)
    expect(window.localStorage.getItem('affiliate_banner_dismissed_until_v1:10')).toBeTruthy()
  })

  it('shows arrival toast when granted bonus increases above previously seen baseline', async () => {
    window.localStorage.setItem('affiliate_seen_bonus_v1:10', '3')
    getAffiliateDetailMock.mockResolvedValue(
      makeDetail({ granted_step_bonus_total: 8, granted_ladder_bonus_total: 0 }),
    )
    const { appStore } = setupStores()
    const showSuccessSpy = vi.spyOn(appStore, 'showSuccess')

    mountBanner()
    await flushPromises()

    expect(showSuccessSpy).toHaveBeenCalledWith(expect.stringContaining('5.00 元'))
  })
})
