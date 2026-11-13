/** 新手引导的站点级配置。 */

/** 客服 QQ 群：引导每张卡片底部的「不想配置？」入口展示。 */
export const SUPPORT_QQ_GROUP = '1127453722'

export type GuideClient = 'codex' | 'claude-code'

/** 引导选的客户端 → 创建 Key 时预选的供应商（与 Key 页的分组供应商一致）。 */
export const CLIENT_PROVIDER: Record<GuideClient, 'openai' | 'anthropic'> = {
  codex: 'openai',
  'claude-code': 'anthropic'
}

/** localStorage 键（按账号区分），升版时换后缀即可让所有人重新引导。 */
export const GUIDE_STORAGE_PREFIX = 'onboarding_guide_v1'
