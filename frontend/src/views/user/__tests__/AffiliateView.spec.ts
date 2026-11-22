import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AffiliateView from '../AffiliateView.vue'

const PaginationStub = {
  name: 'Pagination',
  props: ['total', 'page', 'pageSize'],
  emits: ['update:page', 'update:pageSize'],
  template: '<div />',
}

const { copyToClipboard, getAffiliateDetail, transferAffiliateQuota, syncFromDetail, authState, showError, showSuccess } =
  vi.hoisted(() => ({
    copyToClipboard: vi.fn(),
    getAffiliateDetail: vi.fn(),
    transferAffiliateQuota: vi.fn(),
    syncFromDetail: vi.fn(() => true),
    authState: {
      user: { id: 1 } as { id: number } | null,
      refreshUser: vi.fn().mockResolvedValue(undefined),
    },
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }))

vi.mock('@/api/user', () => ({
  default: {
    getAffiliateDetail,
    transferAffiliateQuota,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState,
}))

vi.mock('@/stores/affiliateSummary', () => ({
  tryUseAffiliateSummaryStore: () => ({
    syncFromDetail,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('AffiliateView', () => {
  const affiliateCode = 'affiliate-code-that-is-long-enough-to-overflow-a-mobile-viewport'

  beforeEach(() => {
    vi.clearAllMocks()
    authState.user = { id: 1 }
    copyToClipboard.mockResolvedValue(true)
    getAffiliateDetail.mockResolvedValue({
      user_id: 1,
      aff_code: affiliateCode,
      inviter_id: null,
      aff_count: 0,
      aff_quota: 0,
      aff_frozen_quota: 0,
      aff_history_quota: 0,
      effective_rebate_rate_percent: 10,
      invitees: [],
    })
  })

  it('stacks long values and copy controls on mobile while retaining desktop rows', async () => {
    const wrapper = mount(AffiliateView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
        },
      },
    })

    await flushPromises()

    const values = wrapper.findAll('code')
    expect(values).toHaveLength(2)
    for (const value of values) {
      expect(value.classes()).toEqual(expect.arrayContaining([
        'min-w-0',
        'break-all',
        'sm:flex-1',
        'sm:truncate',
      ]))
      expect(Array.from(value.element.parentElement?.classList ?? [])).toEqual(expect.arrayContaining([
        'flex-col',
        'items-stretch',
        'sm:flex-row',
        'sm:items-center',
      ]))
    }

    const copyButtons = wrapper.findAll('button').filter((button) =>
      ['affiliate.copyCode', 'affiliate.copyLink'].includes(button.text()),
    )
    expect(copyButtons).toHaveLength(2)
    for (const button of copyButtons) {
      expect(button.classes()).toEqual(expect.arrayContaining([
        'w-full',
        'sm:w-auto',
        'sm:shrink-0',
      ]))
    }

    await copyButtons[0].trigger('click')
    await copyButtons[1].trigger('click')
    await flushPromises()

    expect(copyToClipboard).toHaveBeenNthCalledWith(1, affiliateCode, 'affiliate.codeCopied')
    expect(copyToClipboard).toHaveBeenNthCalledWith(
      2,
      `${window.location.origin}/register?aff=${encodeURIComponent(affiliateCode)}`,
      'affiliate.linkCopied',
    )
  })

  it('renders v2 ladder tiers, withdraw guide and API Key unlock reminder when invitee has paid 10 without consuming 3', async () => {
    getAffiliateDetail.mockResolvedValue({
      user_id: 1,
      aff_code: 'YFMAIL2026',
      inviter_id: null,
      aff_count: 2,
      aff_quota: 320,
      aff_frozen_quota: 4,
      aff_history_quota: 324,
      effective_rebate_rate_percent: 5,
      qualified_invitee_count: 1,
      qualified_invitees_paid_total: 60,
      pending_step_bonus_total: 3,
      granted_step_bonus_total: 8,
      pending_ladder_bonus_total: 0,
      granted_ladder_bonus_total: 0,
      withdraw_threshold: 300,
      can_apply_withdraw: true,
      remind_consume_count: 1,
      ladder_tiers: [
        { tier_target: 5, cumulative_bonus: 10, delta_bonus: 10, requires_min_total_paid: 0, auto_grant: true, status: 'locked' },
        { tier_target: 20, cumulative_bonus: 40, delta_bonus: 30, requires_min_total_paid: 0, auto_grant: true, status: 'locked' },
        { tier_target: 50, cumulative_bonus: 100, delta_bonus: 60, requires_min_total_paid: 0, auto_grant: false, status: 'locked' },
        { tier_target: 100, cumulative_bonus: 200, delta_bonus: 100, requires_min_total_paid: 3000, auto_grant: false, status: 'locked' },
      ],
      invitees: [
        {
          user_id: 101,
          email: 'a***@qq.com',
          username: 'friendA',
          total_rebate: 0.5,
          total_paid_amount: 10,
          total_consumed_amount: 1.2,
          step1_reward_status: 'pending_consume',
          step2_reward_status: 'none',
          pending_bonus_amount: 3,
          granted_bonus_amount: 0,
          display_status: 'pending_consume_3',
          needs_consume_reminder: true,
        },
      ],
    })

    const wrapper = mount(AffiliateView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('[data-test="affiliate-consume-reminder"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('提醒好友配置 API Key 试用即可解锁 3 元')
    expect(wrapper.find('[data-test="affiliate-ladder-card"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="affiliate-withdraw-guide"]').text()).toContain('已满 300 元可申请')
  })

  it('paginates invitees while keeping full-history totals and reminders', async () => {
    getAffiliateDetail.mockImplementation(({ page, page_size }) => Promise.resolve({
      user_id: 1, aff_code: 'TEST', aff_count: 101, aff_quota: 0, aff_frozen_quota: 0, aff_history_quota: 0,
      effective_rebate_rate_percent: 5, qualified_invitee_count: 101, qualified_invitees_paid_total: 1010,
      remind_consume_count: 8, pending_step_bonus_total: 24,
      invitees_total: 101, invitees_page: page, invitees_page_size: page_size,
      invitees: [{ user_id: page, email: `page-${page}@example.com`, total_rebate: 0, needs_consume_reminder: false }],
    }))
    const wrapper = mount(AffiliateView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true, Pagination: PaginationStub } },
    })
    await flushPromises()
    expect(getAffiliateDetail).toHaveBeenLastCalledWith({ page: 1, page_size: 20 })
    expect(wrapper.get('[data-test="affiliate-consume-reminder"]').text()).toContain('8 位好友')
    expect(wrapper.get('[data-test="affiliate-ladder-card"]').text()).toContain('101 / 100')
    const pager = wrapper.findComponent(PaginationStub)
    expect(pager.props()).toMatchObject({ total: 101, page: 1, pageSize: 20 })
    pager.vm.$emit('update:page', 6)
    await flushPromises()
    expect(getAffiliateDetail).toHaveBeenLastCalledWith({ page: 6, page_size: 20 })
    expect(wrapper.get('[data-testid="affiliate-invitees"]').text()).toContain('page-6@example.com')
    expect(wrapper.get('[data-test="affiliate-ladder-card"]').text()).toContain('101 / 100')
    expect(wrapper.get('[data-test="affiliate-consume-reminder"]').text()).toContain('8 位好友')
    pager.vm.$emit('update:pageSize', 50)
    await flushPromises()
    expect(getAffiliateDetail).toHaveBeenLastCalledWith({ page: 1, page_size: 50 })
  })

  it('fetches all pages for cross-page filtering and resets pagination to filtered total', async () => {
    getAffiliateDetail.mockImplementation(({ page, page_size }) => {
      if (page_size === 20) {
        return Promise.resolve({
          user_id: 1,
          aff_code: 'MULTI',
          aff_count: 120,
          aff_quota: 0,
          aff_frozen_quota: 0,
          aff_history_quota: 0,
          effective_rebate_rate_percent: 5,
          invitees_total: 120,
          invitees_page: page,
          invitees_page_size: page_size,
          invitees: [
            { user_id: 1, email: 'page1-unpaid@qq.com', username: 'u1', total_rebate: 0, total_paid_amount: 0, needs_consume_reminder: false },
          ],
        })
      }
      if (page === 1) {
        return Promise.resolve({
          user_id: 1,
          aff_code: 'MULTI',
          aff_count: 120,
          aff_quota: 0,
          aff_frozen_quota: 0,
          aff_history_quota: 0,
          effective_rebate_rate_percent: 5,
          invitees_total: 120,
          invitees_page: 1,
          invitees_page_size: 100,
          invitees: Array.from({ length: 100 }, (_, idx) => ({
            user_id: idx + 1,
            email: `u${idx + 1}@qq.com`,
            username: `u${idx + 1}`,
            total_rebate: 0,
            total_paid_amount: 0,
            needs_consume_reminder: false,
          })),
        })
      }
      return Promise.resolve({
        user_id: 1,
        aff_code: 'MULTI',
        aff_count: 120,
        aff_quota: 0,
        aff_frozen_quota: 0,
        aff_history_quota: 0,
        effective_rebate_rate_percent: 5,
        invitees_total: 120,
        invitees_page: 2,
        invitees_page_size: 100,
        invitees: [
          {
            user_id: 105,
            email: 'page2-pending@qq.com',
            username: 'pendingFriend',
            total_rebate: 0.5,
            total_paid_amount: 10,
            total_consumed_amount: 1,
            display_status: 'pending_consume_3',
            needs_consume_reminder: true,
          },
        ],
      })
    })

    const wrapper = mount(AffiliateView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true, Pagination: PaginationStub } },
    })
    await flushPromises()

    await wrapper.get('[data-testid="affiliate-filter-pending_consume"]').trigger('click')
    await flushPromises()

    expect(getAffiliateDetail).toHaveBeenCalledWith({ page: 1, page_size: 100 })
    expect(getAffiliateDetail).toHaveBeenCalledWith({ page: 2, page_size: 100 })
    expect(wrapper.get('[data-testid="affiliate-invitees"]').text()).toContain('page2-pending@qq.com')
    const pager = wrapper.findComponent(PaginationStub)
    expect(pager.props()).toMatchObject({ total: 1, page: 1 })
  })

  it('does not pollute affiliateSummary store when unmounted or switched to another user before response resolves', async () => {
    let resolvePending!: (val: unknown) => void
    getAffiliateDetail.mockImplementationOnce(
      () => new Promise((resolve) => { resolvePending = resolve }),
    )

    const wrapper = mount(AffiliateView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true } },
    })
    wrapper.unmount()

    resolvePending({
      user_id: 1,
      aff_code: 'STALE',
      aff_count: 0,
      aff_quota: 99,
      aff_frozen_quota: 0,
      aff_history_quota: 99,
      effective_rebate_rate_percent: 5,
      invitees: [],
    })
    await flushPromises()
    expect(syncFromDetail).not.toHaveBeenCalled()

    // Now test user switch while mounted
    let resolveSwitch!: (val: unknown) => void
    getAffiliateDetail.mockImplementationOnce(
      () => new Promise((resolve) => { resolveSwitch = resolve }),
    )
    const wrapper2 = mount(AffiliateView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true } },
    })
    authState.user = { id: 2 }
    resolveSwitch({
      user_id: 1,
      aff_code: 'OLD-USER',
      aff_count: 0,
      aff_quota: 50,
      aff_frozen_quota: 0,
      aff_history_quota: 50,
      effective_rebate_rate_percent: 5,
      invitees: [],
    })
    await flushPromises()
    expect(syncFromDetail).not.toHaveBeenCalled()
    wrapper2.unmount()
  })

  it('keeps server page and pageSize aligned when changing page size inside a filter and switching back to all', async () => {
    getAffiliateDetail.mockImplementation(({ page, page_size }) => {
      if (page_size === 20) {
        return Promise.resolve({
          user_id: 1,
          aff_code: 'DECPL',
          aff_count: 120,
          aff_quota: 0,
          aff_frozen_quota: 0,
          aff_history_quota: 0,
          effective_rebate_rate_percent: 5,
          invitees_total: 120,
          invitees_page: page,
          invitees_page_size: 20,
          invitees: [
            { user_id: page * 1000 + 1, email: `server-p${page}@qq.com`, username: `p${page}`, total_rebate: 0, total_paid_amount: 0 },
          ],
        })
      }
      return Promise.resolve({
        user_id: 1,
        aff_code: 'DECPL',
        aff_count: 120,
        aff_quota: 0,
        aff_frozen_quota: 0,
        aff_history_quota: 0,
        effective_rebate_rate_percent: 5,
        invitees_total: 120,
        invitees_page: page,
        invitees_page_size: 100,
        invitees:
          page === 1
            ? Array.from({ length: 100 }, (_, i) => ({
                user_id: i + 1,
                email: `u${i + 1}@qq.com`,
                username: `u${i + 1}`,
                total_rebate: 0,
                total_paid_amount: 0,
              }))
            : Array.from({ length: 20 }, (_, i) => ({
                user_id: 101 + i,
                email: `u${101 + i}@qq.com`,
                username: `u${101 + i}`,
                total_rebate: 0,
                total_paid_amount: 0,
              })),
      })
    })

    const wrapper = mount(AffiliateView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true, Pagination: PaginationStub } },
    })
    await flushPromises()

    const pager = wrapper.findComponent(PaginationStub)
    // Navigate to server page 3 (pageSize 20)
    pager.vm.$emit('update:page', 3)
    await flushPromises()
    expect(pager.props()).toMatchObject({ total: 120, page: 3, pageSize: 20 })
    expect(wrapper.get('[data-testid="affiliate-invitees"]').text()).toContain('server-p3@qq.com')

    // Switch to 'unpaid' filter and change filter pageSize to 100
    await wrapper.get('[data-testid="affiliate-filter-unpaid"]').trigger('click')
    await flushPromises()
    const filteredPager = wrapper.findComponent(PaginationStub)
    filteredPager.vm.$emit('update:pageSize', 100)
    await flushPromises()
    expect(wrapper.findComponent(PaginationStub).props()).toMatchObject({ total: 120, page: 1, pageSize: 100 })

    // Switch back to 'all': server page 3 and pageSize 20 must remain intact
    await wrapper.get('[data-testid="affiliate-filter-all"]').trigger('click')
    await flushPromises()
    expect(wrapper.findComponent(PaginationStub).props()).toMatchObject({ total: 120, page: 3, pageSize: 20 })
    expect(wrapper.get('[data-testid="affiliate-invitees"]').text()).toContain('server-p3@qq.com')
  })

  it('refreshes the full cross-page filter cache after transferQuota while filtered', async () => {
    let quota = 30
    transferAffiliateQuota.mockImplementation(async () => {
      quota = 0
      return { transferred_quota: 30, balance: 40 }
    })
    getAffiliateDetail.mockImplementation(({ page, page_size }) => {
      if (page_size === 20) {
        return Promise.resolve({
          user_id: 1,
          aff_code: 'FILT-XFER',
          aff_count: 120,
          aff_quota: quota,
          aff_frozen_quota: 0,
          aff_history_quota: 30,
          effective_rebate_rate_percent: 5,
          invitees_total: 120,
          invitees_page: page,
          invitees_page_size: 20,
          invitees: [
            { user_id: 1, email: 'page1-only@qq.com', username: 'u1', total_rebate: 0, total_paid_amount: 0, needs_consume_reminder: false },
          ],
        })
      }
      if (page === 1) {
        return Promise.resolve({
          user_id: 1,
          aff_code: 'FILT-XFER',
          aff_count: 120,
          aff_quota: quota,
          aff_frozen_quota: 0,
          aff_history_quota: 30,
          effective_rebate_rate_percent: 5,
          invitees_total: 120,
          invitees_page: 1,
          invitees_page_size: 100,
          invitees: Array.from({ length: 100 }, (_, i) => ({
            user_id: i + 1,
            email: `p1-${i + 1}@qq.com`,
            username: `u${i + 1}`,
            total_rebate: 0,
            total_paid_amount: 0,
            needs_consume_reminder: false,
          })),
        })
      }
      return Promise.resolve({
        user_id: 1,
        aff_code: 'FILT-XFER',
        aff_count: 120,
        aff_quota: quota,
        aff_frozen_quota: 0,
        aff_history_quota: 30,
        effective_rebate_rate_percent: 5,
        invitees_total: 120,
        invitees_page: 2,
        invitees_page_size: 100,
        invitees: [
          {
            user_id: 110,
            email: 'page2-target@qq.com',
            username: 'targetFriend',
            total_rebate: 1,
            total_paid_amount: 10,
            total_consumed_amount: 1,
            display_status: 'pending_consume_3',
            needs_consume_reminder: true,
          },
        ],
      })
    })

    const wrapper = mount(AffiliateView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: true, Pagination: PaginationStub } },
    })
    await flushPromises()

    await wrapper.get('[data-testid="affiliate-filter-pending_consume"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="affiliate-invitees"]').text()).toContain('page2-target@qq.com')

    // Click transfer button while still on filtered tab
    const transferBtn = wrapper.findAll('button').find((b) => b.text().includes('affiliate.transfer.button'))!
    await transferBtn.trigger('click')
    await flushPromises()

    expect(transferAffiliateQuota).toHaveBeenCalledTimes(1)
    expect(showSuccess).toHaveBeenCalledTimes(1)
    expect(showError).not.toHaveBeenCalled()
    // Filter cache must be re-fetched across pages, NOT degraded to page 1 only
    expect(wrapper.get('[data-testid="affiliate-invitees"]').text()).toContain('page2-target@qq.com')
    expect(wrapper.findComponent(PaginationStub).props()).toMatchObject({ total: 1, page: 1 })
  })
})
