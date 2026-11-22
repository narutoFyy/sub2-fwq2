import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { UserAffiliateDetail } from '@/types'
import {
  MAINSTREAM_TRIAL_EMAIL_DOMAINS,
  buildInviteLink,
  buildRemindConsumeCopy,
  buildShareCopy,
  consumeUnseenGrantedBonusDelta,
  copyTextToClipboard,
  dismissDefaultBanner,
  formatYuan,
  getTrialDaysRemaining,
  hasCompleteEmailDomain,
  isDefaultBannerDismissed,
  isMainstreamTrialEmail,
  isTrialBonusActive,
  resolveDashboardBanner,
  resolveNextLadderSprint,
  resolveSidebarBadge,
} from '../helpers'

function makeDetail(overrides: Partial<UserAffiliateDetail> = {}): UserAffiliateDetail {
  return {
    user_id: 42,
    aff_code: 'STONE42',
    aff_count: 0,
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
      { tier_target: 20, cumulative_bonus: 40, delta_bonus: 30, requires_min_total_paid: 0, auto_grant: true, status: 'locked' },
    ],
    invitees: [],
    ...overrides,
  }
}

describe('affiliate helpers', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  it('formats CNY values in zh/en, normalizes negative tiny decimals to zero, and guards non-finite inputs', () => {
    expect(formatYuan(3)).toBe('3.00 元')
    expect(formatYuan(3, 'en')).toBe('3.00 CNY')
    expect(formatYuan(8.5)).toBe('8.50 元')
    expect(formatYuan(-0)).toBe('0.00 元')
    expect(formatYuan(-0.001)).toBe('0.00 元')
    expect(formatYuan(-0.0049)).toBe('0.00 元')
    expect(formatYuan(-0.005)).toBe('-0.01 元')
    expect(formatYuan(-0.005, 'en-US')).toBe('-0.01 CNY')
    expect(formatYuan(Number.NaN)).toBe('0.00 元')
    expect(formatYuan(Number.POSITIVE_INFINITY, 'en')).toBe('0.00 CNY')
  })

  it('matches all 18 mainstream trial email domains case-insensitively and guards incomplete domains', () => {
    expect(MAINSTREAM_TRIAL_EMAIL_DOMAINS).toHaveLength(18)
    for (const domain of MAINSTREAM_TRIAL_EMAIL_DOMAINS) {
      expect(isMainstreamTrialEmail(`User@${domain.toUpperCase()}`)).toBe(true)
      expect(hasCompleteEmailDomain(`user@${domain}`)).toBe(true)
    }
    expect(isMainstreamTrialEmail('user@company.example')).toBe(false)
    expect(hasCompleteEmailDomain('user@company.example')).toBe(true)
    expect(hasCompleteEmailDomain('user@')).toBe(false)
    expect(hasCompleteEmailDomain('user@qq')).toBe(false)
    expect(hasCompleteEmailDomain('user@qq.')).toBe(false)
    expect(hasCompleteEmailDomain('user')).toBe(false)
  })

  it('builds invite links, share copy, and reminder copy', () => {
    const link = buildInviteLink('CODE+1', 'https://yf-mail.com/')
    expect(link).toBe('https://yf-mail.com/register?aff=CODE%2B1')

    const share = buildShareCopy('STONE88', 'https://yf-mail.com', '石头中转站')
    expect(share).toContain('石头中转站')
    expect(share).toContain('1 元')
    expect(share).toContain('https://yf-mail.com/register?aff=STONE88')

    expect(buildRemindConsumeCopy('小明')).toContain('小明，')
    expect(buildRemindConsumeCopy()).toContain('API Key')
    expect(buildRemindConsumeCopy()).toContain('3 元')
  })

  it('calculates remaining trial days and active trial status', () => {
    const now = Date.UTC(2026, 10, 21, 0, 0, 0)
    const twoDaysLater = new Date(now + 2 * 86_400_000 - 1000).toISOString()
    const expired = new Date(now - 1000).toISOString()

    expect(getTrialDaysRemaining(twoDaysLater, now)).toBe(2)
    expect(getTrialDaysRemaining(expired, now)).toBe(0)
    expect(getTrialDaysRemaining(null, now)).toBe(0)

    expect(isTrialBonusActive({ trial_bonus_status: 'active', trial_expires_at: twoDaysLater }, now)).toBe(true)
    expect(isTrialBonusActive({ trial_bonus_status: 'active', trial_expires_at: expired }, now)).toBe(false)
    expect(isTrialBonusActive({ trial_bonus_status: 'consumed', trial_expires_at: twoDaysLater }, now)).toBe(false)
  })

  it('detects ladder sprint with paid threshold gap and manual-review flags', () => {
    expect(resolveNextLadderSprint(makeDetail({ qualified_invitee_count: 2 }))).toBeNull()
    expect(resolveNextLadderSprint(makeDetail({ qualified_invitee_count: 3 }))).toMatchObject({
      remaining: 2,
      paidRemaining: 0,
      requiresManualReview: false,
      tier: { tier_target: 5, delta_bonus: 10 },
    })
    expect(resolveNextLadderSprint(makeDetail({ qualified_invitee_count: 4 }))).toMatchObject({
      remaining: 1,
      paidRemaining: 0,
      requiresManualReview: false,
      tier: { tier_target: 5, delta_bonus: 10 },
    })

    const tier100Sprint = resolveNextLadderSprint(
      makeDetail({
        qualified_invitee_count: 99,
        qualified_invitees_paid_total: 1850,
        ladder_tiers: [
          {
            tier_target: 100,
            cumulative_bonus: 200,
            delta_bonus: 100,
            requires_min_total_paid: 3000,
            auto_grant: false,
            status: 'locked',
          },
        ],
      }),
    )
    expect(tier100Sprint).toMatchObject({
      remaining: 1,
      paidRemaining: 1150,
      requiresManualReview: true,
    })
  })

  it('resolves dashboard banner in strict priority order P1 > P2 > P3 > P4 > P5', () => {
    const now = Date.UTC(2026, 10, 21, 0, 0, 0)
    const activeExpiry = new Date(now + 3 * 86_400_000).toISOString()

    // P1 beats P2/P3/P4/P5
    const p1 = resolveDashboardBanner(
      makeDetail({
        remind_consume_count: 2,
        pending_step_bonus_total: 6,
        aff_quota: 320,
        trial_bonus_status: 'active',
        trial_expires_at: activeExpiry,
        qualified_invitee_count: 4,
      }),
      { dismissedDefault: false, now },
    )
    expect(p1).toEqual({ type: 'consume_reminder', count: 2, pendingBonus: 6 })

    // P2 beats P3/P4/P5 and flags withdraw eligibility
    const p2 = resolveDashboardBanner(
      makeDetail({
        aff_quota: 320,
        withdraw_threshold: 300,
        trial_bonus_status: 'active',
        trial_expires_at: activeExpiry,
        qualified_invitee_count: 4,
      }),
      { dismissedDefault: false, now },
    )
    expect(p2).toEqual({ type: 'transferable', amount: 320, withdrawThreshold: 300, canWithdraw: true })

    // P3 beats P4/P5
    const p3 = resolveDashboardBanner(
      makeDetail({
        trial_bonus_status: 'active',
        trial_bonus_amount: 1,
        trial_expires_at: activeExpiry,
        qualified_invitee_count: 4,
      }),
      { dismissedDefault: false, now },
    )
    expect(p3).toEqual({ type: 'trial_active', daysRemaining: 3, amount: 1 })

    // P4 beats P5
    const p4 = resolveDashboardBanner(
      makeDetail({ qualified_invitee_count: 4 }),
      { dismissedDefault: false, now },
    )
    expect(p4?.type).toBe('ladder_sprint')

    // P5 shown when not dismissed, hidden when dismissed
    const p5 = resolveDashboardBanner(makeDetail(), { dismissedDefault: false, now })
    expect(p5).toEqual({ type: 'default_promo', rebateRatePercent: 5 })
    expect(resolveDashboardBanner(makeDetail(), { dismissedDefault: true, now })).toBeNull()
  })

  it('resolves sidebar badges by priority', () => {
    expect(resolveSidebarBadge(makeDetail({ remind_consume_count: 1, aff_quota: 50 }))).toMatchObject({
      key: 'pendingUnlock',
      text: '待解锁',
      tone: 'amber',
    })
    expect(resolveSidebarBadge(makeDetail({ aff_quota: 12 }))).toMatchObject({
      key: 'transferable',
      text: '可转余额',
      tone: 'emerald',
    })
    expect(resolveSidebarBadge(makeDetail())).toMatchObject({
      key: 'defaultPromo',
      text: '奖 8 元',
      tone: 'neutral',
    })
    expect(resolveSidebarBadge(null)).toMatchObject({
      key: 'defaultPromo',
      tone: 'neutral',
    })
  })

  it('persists default banner dismissal for 7 days and expires afterwards', () => {
    const start = Date.UTC(2026, 10, 21, 0, 0, 0)
    expect(isDefaultBannerDismissed(7, start)).toBe(false)
    dismissDefaultBanner(7, start)
    expect(isDefaultBannerDismissed(7, start + 6 * 86_400_000)).toBe(true)
    expect(isDefaultBannerDismissed(7, start + 7 * 86_400_000 + 1)).toBe(false)
  })

  it('records baseline on first bonus check and reports only positive granted increments', () => {
    // First load writes baseline without firing toast
    expect(
      consumeUnseenGrantedBonusDelta(99, { granted_step_bonus_total: 8, granted_ladder_bonus_total: 0 }),
    ).toBe(0)
    // Same total -> 0
    expect(
      consumeUnseenGrantedBonusDelta(99, { granted_step_bonus_total: 8, granted_ladder_bonus_total: 0 }),
    ).toBe(0)
    // Positive increment -> returns delta and updates stored baseline
    expect(
      consumeUnseenGrantedBonusDelta(99, { granted_step_bonus_total: 11, granted_ladder_bonus_total: 10 }),
    ).toBe(13)
    expect(
      consumeUnseenGrantedBonusDelta(99, { granted_step_bonus_total: 11, granted_ladder_bonus_total: 10 }),
    ).toBe(0)
  })

  it('builds localized English share and remind copy', () => {
    const shareEn = buildShareCopy('STONE88', 'https://yf-mail.com', 'yf-mail', 'en')
    expect(shareEn).toContain('1.00 CNY')
    expect(shareEn).toContain('https://yf-mail.com/register?aff=STONE88')

    const remindEn = buildRemindConsumeCopy('Alex', 'en')
    expect(remindEn).toContain('Alex')
    expect(remindEn).toContain('3.00 CNY')
  })

  it('returns false and cleans up fallback textarea when copyTextToClipboard fails or throws', async () => {
    const origExec = document.execCommand
    const execFail = vi.fn(() => false)
    document.execCommand = execFail

    const ok1 = await copyTextToClipboard('hello')
    expect(ok1).toBe(false)
    expect(document.querySelectorAll('textarea')).toHaveLength(0)

    document.execCommand = vi.fn(() => {
      throw new Error('copy blocked')
    })
    const ok2 = await copyTextToClipboard('hello')
    expect(ok2).toBe(false)
    expect(document.querySelectorAll('textarea')).toHaveLength(0)

    document.execCommand = origExec
  })
})
