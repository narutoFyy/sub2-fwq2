import type { GuideClient } from './config'
import type { TargetResolver } from './useGuideTarget'

export interface GuideLink {
  label: string
  href: string
  meta?: string
}

/** 卡片里的内容块（下载页等用）。 */
export type GuideBlock =
  | { type: 'download'; heading: string; primary?: GuideLink; alternates?: GuideLink[]; note?: string; backup?: GuideLink; verify?: GuideLink }
  | { type: 'command'; heading: string; command: string; hint?: string }
  | { type: 'copy'; heading?: string; label: string; value: string; secret?: boolean }
  | { type: 'note'; text: string }
  /** 带脉冲小点的等待状态（如「正在等待第一次调用」）。 */
  | { type: 'status'; text: string }
  /** 一段要复制给智能体的话：展示时密钥打码，复制完整版本。 */
  | { type: 'prompt'; heading: string; shown: string; value: string; note?: string }
  /** 推荐模型：名字 + 一句话能力介绍。 */
  | { type: 'models'; heading: string; hint?: string; items: { name: string; pitch: string; tag?: string }[] }

export interface GuideAction {
  label: string
  run: () => void
  /** 「跳过这步 / 我已经有…」这类次要出口：放在卡片最底一行（左边是客服入口），小字，不和主操作抢眼。 */
  quiet?: boolean
}

/** 选项卡片：选用法（GuideClient），或选生图方式（image-web / image-agent）。 */
export type GuideOptionId = GuideClient | 'image-web' | 'image-agent'

export interface GuideClientOption {
  id: GuideOptionId
  name: string
  desc: string
}

/** 当前这一屏要展示的内容（由 GuideLayer 按步骤 + 页面状态算出来）。 */
export interface GuideView {
  /** modal：居中卡片 + 背景虚化；spot：聚光灯照着页面上的目标，卡片贴在旁边。 */
  kind: 'modal' | 'spot'
  /** 内容切换动画用的键。 */
  key: string
  title: string
  body?: string
  hint?: string
  list?: string[]
  /** 小步骤清单（建密钥弹窗里的「起名字 → 选分组 → 点创建」），做到哪一步打勾到哪一步。 */
  checklist?: { label: string; state: 'done' | 'active' | 'todo' }[]
  options?: GuideClientOption[]
  blocks?: GuideBlock[]
  /** 宽一点的卡片（下载页内容多）。 */
  wide?: boolean
  /** 允许收起成角落的小胶囊（用户去安装软件时不挡着页面）。 */
  collapsible?: boolean
  /** 收起后胶囊上的文字。 */
  pillLabel?: string
  primary?: GuideAction
  secondary?: GuideAction
  /** 目标找不到时才出现的「直接前往」按钮。 */
  fallback?: GuideAction
  back?: GuideAction
  /** 目标是要点击的：显示点击提示的指针。 */
  click?: boolean
  resolvers: TargetResolver[]
  /** false：不压暗页面（目标在弹窗里，弹窗自己已有遮罩，再压暗会太黑）。 */
  dim?: boolean
  /** 目标在侧栏里：手机上需要先展开侧栏抽屉。 */
  inSidebar?: boolean
}
