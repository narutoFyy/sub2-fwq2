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

export interface GuideAction {
  label: string
  run: () => void
}

export interface GuideClientOption {
  id: GuideClient
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
