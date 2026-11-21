import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AffiliateView from '../AffiliateView.vue'

const { copyToClipboard, getAffiliateDetail } = vi.hoisted(() => ({
  copyToClipboard: vi.fn(),
  getAffiliateDetail: vi.fn(),
}))

vi.mock('@/api/user', () => ({
  default: {
    getAffiliateDetail,
    transferAffiliateQuota: vi.fn(),
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    refreshUser: vi.fn(),
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
})
