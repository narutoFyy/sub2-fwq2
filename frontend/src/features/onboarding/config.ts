/** 新手引导的站点级配置。 */

/** 客服 QQ 群：引导每张卡片底部的「不想配置？」入口展示。 */
export const SUPPORT_QQ_GROUP = '1127453722'

/**
 * 引导里可选的用法：两个编程工具（要装软件、经 CC Switch 导入）、生图（网页里直接画，
 * 或复制一段话让 Codex / Claude Code 自己装 my-image），以及两个直接在站内聊天页使用的（纯聊天、玄学测算）。
 */
export type GuideClient = 'codex' | 'claude-code' | 'image' | 'chat' | 'fortune'

export const GUIDE_CLIENTS: GuideClient[] = ['codex', 'claude-code', 'image', 'chat', 'fortune']

/** 生图的两种方式：web 在本站生图页画；agent 把 my-image 装进 Codex / Claude Code。 */
export type ImageMode = 'web' | 'agent'

/** 站内聊天类用法：建好密钥后直接去 /chat，跳过安装和导入。 */
export function isChatClient(client: GuideClient | null | undefined): client is 'chat' | 'fortune' {
  return client === 'chat' || client === 'fortune'
}

/** 引导选的客户端 → 创建 Key 时预选的供应商（与 Key 页的分组供应商一致）；聊天类不限定，用默认。 */
export const CLIENT_PROVIDER: Partial<Record<GuideClient, 'openai' | 'anthropic'>> = {
  codex: 'openai',
  'claude-code': 'anthropic'
}

/** 聊天类用法在聊天页打开的地址。 */
export function chatPathFor(client: 'chat' | 'fortune'): string {
  return client === 'fortune' ? '/chat?mode=fortune' : '/chat'
}

/** localStorage 键（按账号区分），升版时换后缀即可让所有人重新引导。 */
export const GUIDE_STORAGE_PREFIX = 'onboarding_guide_v1'
