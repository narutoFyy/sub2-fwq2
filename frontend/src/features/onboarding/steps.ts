/**
 * 新手引导的步骤与流转（纯逻辑，不依赖 DOM / 路由，便于单测）。
 *
 * 流程：欢迎 → 选工具 → 买码并兑换 → 创建 Key → 完成。
 * 「买码并兑换」拆成 buy（去小铺）/ returned（问买好没）/ redeem（去兑换）三步；
 * 「创建 Key」是一步，具体高亮哪个按钮由页面当前状态决定（见 GuideLayer）。
 */
export type GuideStepId = 'welcome' | 'client' | 'buy' | 'returned' | 'redeem' | 'key' | 'finish'

export const STEP_ORDER: GuideStepId[] = ['welcome', 'client', 'buy', 'returned', 'redeem', 'key', 'finish']

/** 进度条的四个阶段（欢迎页不算）。 */
export const STAGES = ['client', 'fund', 'key', 'connect'] as const
export type GuideStage = (typeof STAGES)[number]

export function stageOf(step: GuideStepId): number {
  switch (step) {
    case 'welcome':
    case 'client':
      return 0
    case 'buy':
    case 'returned':
    case 'redeem':
      return 1
    case 'key':
      return 2
    case 'finish':
      return 3
  }
}

/** 购买入口：站内支付页 / 自定义菜单里的外链（如小铺）/ 没有。 */
export type PurchaseMode = 'payment' | 'external' | 'none'

export interface GuideContext {
  /** 余额大于 0 或有生效中的订阅：不用再充值。 */
  hasFunds: boolean
  purchaseMode: PurchaseMode
  /** 兑换页是否可用（简易模式会隐藏）。 */
  redeemAvailable: boolean
}

/** 充值阶段的第一步：有入口就先去买，没有入口直接去兑换，没有兑换页也跳过。 */
function firstFundStep(ctx: GuideContext): GuideStepId {
  if (ctx.hasFunds) return 'key'
  if (ctx.purchaseMode !== 'none') return 'buy'
  return ctx.redeemAvailable ? 'redeem' : 'key'
}

/** 完成当前步骤后的下一步（跳过已经不需要的步骤）。 */
export function nextStep(current: GuideStepId, ctx: GuideContext): GuideStepId {
  switch (current) {
    case 'welcome':
      return 'client'
    case 'client':
      return firstFundStep(ctx)
    case 'buy':
      return ctx.hasFunds ? 'key' : 'returned'
    case 'returned':
      return ctx.hasFunds || !ctx.redeemAvailable ? 'key' : 'redeem'
    case 'redeem':
      return 'key'
    case 'key':
      return 'finish'
    case 'finish':
      return 'finish'
  }
}

/** 上一步（用于「返回」按钮）。 */
export function prevStep(current: GuideStepId): GuideStepId {
  switch (current) {
    case 'client':
      return 'welcome'
    case 'returned':
      return 'buy'
    default:
      return current
  }
}

export const FUND_STEPS: GuideStepId[] = ['buy', 'returned', 'redeem']
