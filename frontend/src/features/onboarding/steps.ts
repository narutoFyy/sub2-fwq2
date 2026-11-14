/**
 * 新手引导的步骤与流转（纯逻辑，不依赖 DOM / 路由，便于单测）。
 *
 * 流程：欢迎 → 选工具 → 买码并兑换 → 创建 Key → 问「装了吗」→（下载）→ 一键导入 → 完成。
 * 「买码并兑换」拆成 buy（去小铺）/ returned（问买好没）/ redeem（去兑换）三步；
 * 「创建 Key」是一步，具体高亮哪个按钮由页面当前状态决定（见 GuideLayer）。
 * 选了站内聊天类用法（纯聊天 / 玄学测算）：创建 Key 之后进 chat（去聊天页），再到 finish 等第一次调用。
 */
export type GuideStepId =
  | 'welcome'
  | 'client'
  | 'buy'
  | 'returned'
  | 'redeem'
  | 'key'
  | 'ccs'
  | 'download'
  | 'import'
  | 'imported'
  | 'manual'
  | 'chat'
  | 'finish'
  | 'success'

export const STEP_ORDER: GuideStepId[] = [
  'welcome', 'client', 'buy', 'returned', 'redeem', 'key', 'ccs', 'download', 'import', 'imported', 'manual', 'chat', 'finish', 'success'
]

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
    default:
      // 问「装了吗」、下载、导入、手动配置、去聊天、完成都属于「接入工具 / 开始聊天」。
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
  /** 选的是站内聊天类用法：建好 Key 直接去聊天页，不用安装和导入。 */
  chatOnly?: boolean
}

/** 创建 Key 之后的下一步。 */
export function afterKeyStep(ctx: Pick<GuideContext, 'chatOnly'>): GuideStepId {
  return ctx.chatOnly ? 'chat' : 'ccs'
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
      return afterKeyStep(ctx)
    case 'chat': // 打开了聊天页
      return 'finish'
    case 'ccs': // 「都装好了」
    case 'download': // 「我装好了，继续」
      return 'import'
    case 'import': // 点了导入按钮
      return 'imported'
    case 'imported': // 「已导入成功」
    case 'manual': // 「我配置好了」
      return 'finish'
    case 'finish': // 等到第一次调用
    case 'success':
      return 'success'
  }
}

/** 上一步（用于「返回」按钮）。 */
export function prevStep(current: GuideStepId): GuideStepId {
  switch (current) {
    case 'client':
      return 'welcome'
    case 'returned':
      return 'buy'
    case 'download':
      return 'ccs'
    case 'imported':
      return 'import'
    case 'manual':
      return 'imported'
    default:
      return current
  }
}

export const FUND_STEPS: GuideStepId[] = ['buy', 'returned', 'redeem']
