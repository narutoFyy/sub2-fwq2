import type { AffiliateLadderTier, UserAffiliateDetail } from '@/types'

const BANNER_DISMISS_STORAGE_PREFIX = 'affiliate_banner_dismissed_until_v1:'
const SEEN_BONUS_STORAGE_PREFIX = 'affiliate_seen_bonus_v1:'
const DEFAULT_BANNER_DISMISS_TTL_MS = 7 * 24 * 60 * 60 * 1000
const BONUS_DELTA_EPSILON = 0.0001
const MS_PER_DAY = 86_400_000

/**
 * 与 backend/internal/service/affiliate_v2.go (L73-92) 保持完全一致的 18 个主流邮箱域名。
 */
export const MAINSTREAM_TRIAL_EMAIL_DOMAINS = [
  'qq.com',
  'foxmail.com',
  'vip.qq.com',
  '163.com',
  '126.com',
  'yeah.net',
  'vip.163.com',
  'gmail.com',
  'googlemail.com',
  'outlook.com',
  'hotmail.com',
  'live.com',
  'msn.com',
  'outlook.cn',
  'live.cn',
  'icloud.com',
  'me.com',
  'mac.com',
] as const

const MAINSTREAM_TRIAL_EMAIL_DOMAIN_SET = new Set<string>(MAINSTREAM_TRIAL_EMAIL_DOMAINS)

export function isZhLocale(locale?: string | null): boolean {
  if (!locale) return true
  return locale.trim().toLowerCase().startsWith('zh')
}

export function formatRebateRate(value?: number | null): string | null {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) return null
  const rounded = Math.round(value * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
}

export function formatYuan(value: number, locale = 'zh'): string {
  const unit = isZhLocale(locale) ? '元' : 'CNY'
  if (!Number.isFinite(value)) return `0.00 ${unit}`
  const absRounded = Math.round(Math.abs(value) * 100 + 1e-9) / 100
  const signed = absRounded === 0 ? 0 : value < 0 ? -absRounded : absRounded
  return `${signed.toFixed(2)} ${unit}`
}

export function hasCompleteEmailDomain(email: string): boolean {
  const normalized = email.trim()
  const at = normalized.lastIndexOf('@')
  if (at <= 0 || at >= normalized.length - 1) return false
  const domain = normalized.slice(at + 1).trim()
  if (!domain || /\s/.test(domain)) return false
  const dot = domain.lastIndexOf('.')
  return dot > 0 && dot < domain.length - 1
}

export function isMainstreamTrialEmail(email: string): boolean {
  const normalized = email.trim().toLowerCase()
  const at = normalized.lastIndexOf('@')
  if (at <= 0 || at >= normalized.length - 1) return false
  const domain = normalized.slice(at + 1).trim()
  return MAINSTREAM_TRIAL_EMAIL_DOMAIN_SET.has(domain)
}

export function buildInviteLink(affCode: string, origin?: string): string {
  const code = affCode.trim()
  const resolvedOrigin = (
    origin !== undefined ? origin : typeof window !== 'undefined' ? window.location.origin : ''
  ).replace(/\/+$/, '')
  const path = `/register?aff=${encodeURIComponent(code)}`
  return resolvedOrigin ? `${resolvedOrigin}${path}` : path
}

export function buildShareCopy(
  affCode: string,
  origin?: string,
  brandName?: string,
  locale = 'zh',
): string {
  const link = buildInviteLink(affCode, origin)
  const zh = isZhLocale(locale)
  const name = brandName?.trim() || (zh ? '石头中转站' : 'yf-mail')
  if (!zh) {
    return `I've been using ${name} for Claude Code, Codex, and web AI chat — super stable and reliable. Sign up with my invite link using a mainstream email (Gmail / Outlook / iCloud / QQ / 163) to claim 1.00 CNY trial credit: ${link}`
  }
  return `最近在用「${name}」接 Claude Code / Codex 和网页直聊，稳定省心。通过我的专属链接注册（QQ / 163 / Gmail / Outlook / iCloud 等主流邮箱）立领 1 元体验额度，可以先试再用：${link}`
}

export function buildRemindConsumeCopy(friendName?: string, locale = 'zh'): string {
  const trimmed = friendName?.trim()
  if (!isZhLocale(locale)) {
    const prefix = trimmed ? `Hey ${trimmed}, ` : 'Hey! '
    return `${prefix}saw your top-up went through — remember to create an API Key for Claude Code / Codex or spend 3.00 CNY in Web Chat to unlock my referral reward. Ping me anytime if you need help setting up!`
  }
  const prefix = trimmed ? `${trimmed}，` : ''
  return `${prefix}看你已经充值啦～记得在后台创建 API Key 接入 Claude Code / Codex，或者直接在网页聊天页试用满 3 元，就能顺手帮我解锁邀请奖励，配置遇到问题随时问我！`
}

/**
 * 不依赖 Pinia 的安全剪贴板复制工具。
 * 优先使用 Clipboard API，失败或非安全上下文降级为 textarea + execCommand，并确保在 finally 中清理节点。
 */
export async function copyTextToClipboard(text: string): Promise<boolean> {
  if (!text) return false

  if (
    typeof navigator !== 'undefined' &&
    navigator.clipboard &&
    typeof navigator.clipboard.writeText === 'function' &&
    typeof window !== 'undefined' &&
    window.isSecureContext
  ) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // 降级到 textarea 复制
    }
  }

  if (typeof document === 'undefined' || !document.body) {
    return false
  }

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', 'true')
  textarea.style.cssText = 'position:fixed;left:0;top:0;width:1px;height:1px;opacity:0;pointer-events:none'
  document.body.appendChild(textarea)
  try {
    textarea.focus({ preventScroll: true })
    textarea.select()
    textarea.setSelectionRange(0, textarea.value.length)
    return Boolean(document.execCommand('copy'))
  } catch {
    return false
  } finally {
    if (textarea.parentNode) {
      textarea.parentNode.removeChild(textarea)
    }
  }
}

export function getTrialDaysRemaining(expiresAt?: string | null, now = Date.now()): number {
  if (!expiresAt) return 0
  const expiresMs = new Date(expiresAt).getTime()
  if (!Number.isFinite(expiresMs) || expiresMs <= now) return 0
  return Math.ceil((expiresMs - now) / MS_PER_DAY)
}

export function isTrialBonusActive(
  detail?: Pick<UserAffiliateDetail, 'trial_bonus_status' | 'trial_expires_at'> | null,
  now = Date.now(),
): boolean {
  return detail?.trial_bonus_status === 'active' && getTrialDaysRemaining(detail.trial_expires_at, now) > 0
}

export interface LadderSprintInfo {
  tier: AffiliateLadderTier
  remaining: number
  paidRemaining: number
  requiresManualReview: boolean
}

export function resolveNextLadderSprint(
  detail?: Pick<
    UserAffiliateDetail,
    'ladder_tiers' | 'qualified_invitee_count' | 'qualified_invitees_paid_total'
  > | null,
): LadderSprintInfo | null {
  const tiers = detail?.ladder_tiers
  if (!tiers || tiers.length === 0) return null
  const qualified = detail.qualified_invitee_count ?? 0
  const nextLocked = [...tiers]
    .filter((tier) => tier.status === 'locked')
    .sort((a, b) => a.tier_target - b.tier_target)[0]
  if (!nextLocked) return null
  const remaining = nextLocked.tier_target - qualified
  if (remaining >= 1 && remaining <= 2) {
    const requiredPaid = nextLocked.requires_min_total_paid ?? 0
    const currentPaid = detail.qualified_invitees_paid_total ?? 0
    const paidRemaining = Math.max(0, requiredPaid - currentPaid)
    return {
      tier: nextLocked,
      remaining,
      paidRemaining,
      requiresManualReview: !nextLocked.auto_grant,
    }
  }
  return null
}

export type DashboardBannerState =
  | {
      type: 'consume_reminder'
      count: number
      pendingBonus: number
    }
  | {
      type: 'transferable'
      amount: number
      withdrawThreshold: number
      canWithdraw: boolean
    }
  | {
      type: 'trial_active'
      daysRemaining: number
      amount: number
    }
  | {
      type: 'ladder_sprint'
      tier: AffiliateLadderTier
      remaining: number
      paidRemaining: number
      requiresManualReview: boolean
    }
  | {
      type: 'default_promo'
      rebateRatePercent: number | null
    }

function getRemindConsumeCount(detail: Pick<UserAffiliateDetail, 'remind_consume_count' | 'invitees'>): number {
  if (typeof detail.remind_consume_count === 'number') {
    return detail.remind_consume_count
  }
  return (detail.invitees ?? []).filter((item) => item.needs_consume_reminder).length
}

export function resolveDashboardBanner(
  detail: UserAffiliateDetail | null | undefined,
  options: { dismissedDefault: boolean; now?: number },
): DashboardBannerState | null {
  if (!detail) return null
  const now = options.now ?? Date.now()

  // P1: 好友已充未消费满 3 元
  const remindCount = getRemindConsumeCount(detail)
  if (remindCount > 0) {
    return {
      type: 'consume_reminder',
      count: remindCount,
      pendingBonus: detail.pending_step_bonus_total ?? 0,
    }
  }

  // P2: 有可转余额的返利额度
  if (detail.aff_quota > 0) {
    const withdrawThreshold = detail.withdraw_threshold ?? 300
    return {
      type: 'transferable',
      amount: detail.aff_quota,
      withdrawThreshold,
      canWithdraw: detail.aff_quota >= withdrawThreshold,
    }
  }

  // P3: 受邀新人 1 元体验金生效中
  if (isTrialBonusActive(detail, now)) {
    return {
      type: 'trial_active',
      daysRemaining: getTrialDaysRemaining(detail.trial_expires_at, now),
      amount: detail.trial_bonus_amount ?? 1,
    }
  }

  // P4: 阶梯奖临门一脚（差 1~2 人）
  const sprint = resolveNextLadderSprint(detail)
  if (sprint) {
    return {
      type: 'ladder_sprint',
      tier: sprint.tier,
      remaining: sprint.remaining,
      paidRemaining: sprint.paidRemaining,
      requiresManualReview: sprint.requiresManualReview,
    }
  }

  // P5: 默认推广横幅（7 天内未手动关闭）
  if (!options.dismissedDefault) {
    const rate =
      typeof detail.effective_rebate_rate_percent === 'number' &&
      Number.isFinite(detail.effective_rebate_rate_percent) &&
      detail.effective_rebate_rate_percent > 0
        ? detail.effective_rebate_rate_percent
        : null
    return {
      type: 'default_promo',
      rebateRatePercent: rate,
    }
  }

  return null
}

export interface SidebarAffiliateBadge {
  key: 'pendingUnlock' | 'transferable' | 'defaultPromo'
  text: string
  tone: 'amber' | 'emerald' | 'neutral'
}

export function resolveSidebarBadge(detail?: UserAffiliateDetail | null): SidebarAffiliateBadge {
  if (detail && getRemindConsumeCount(detail) > 0) {
    return { key: 'pendingUnlock', text: '待解锁', tone: 'amber' }
  }
  if (detail && detail.aff_quota > 0) {
    return { key: 'transferable', text: '可转余额', tone: 'emerald' }
  }
  return { key: 'defaultPromo', text: '奖 8 元', tone: 'neutral' }
}

export function isDefaultBannerDismissed(userId: number | string, now = Date.now()): boolean {
  if (typeof window === 'undefined') return false
  const key = `${BANNER_DISMISS_STORAGE_PREFIX}${userId}`
  try {
    const raw = window.localStorage.getItem(key)
    if (!raw) return false
    const until = Number(raw)
    if (!Number.isFinite(until) || until <= now) {
      window.localStorage.removeItem(key)
      return false
    }
    return true
  } catch {
    return false
  }
}

export function dismissDefaultBanner(userId: number | string, now = Date.now()): void {
  if (typeof window === 'undefined') return
  const key = `${BANNER_DISMISS_STORAGE_PREFIX}${userId}`
  try {
    window.localStorage.setItem(key, String(now + DEFAULT_BANNER_DISMISS_TTL_MS))
  } catch {
    // 忽略浏览器存储异常
  }
}

export function consumeUnseenGrantedBonusDelta(
  userId: number | string,
  detail?: Pick<UserAffiliateDetail, 'granted_step_bonus_total' | 'granted_ladder_bonus_total'> | null,
): number {
  if (typeof window === 'undefined' || !detail) return 0
  const current = (detail.granted_step_bonus_total ?? 0) + (detail.granted_ladder_bonus_total ?? 0)
  if (!Number.isFinite(current) || current < 0) return 0

  const key = `${SEEN_BONUS_STORAGE_PREFIX}${userId}`
  try {
    const raw = window.localStorage.getItem(key)
    if (raw === null) {
      window.localStorage.setItem(key, String(current))
      return 0
    }
    const seen = Number(raw)
    if (!Number.isFinite(seen)) {
      window.localStorage.setItem(key, String(current))
      return 0
    }
    const delta = current - seen
    if (delta > BONUS_DELTA_EPSILON) {
      window.localStorage.setItem(key, String(current))
      return Math.round(delta * 100) / 100
    }
    if (current < seen) {
      window.localStorage.setItem(key, String(current))
    }
    return 0
  } catch {
    return 0
  }
}
