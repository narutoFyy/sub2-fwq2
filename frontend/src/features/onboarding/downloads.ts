/**
 * 下载页要展示的内容：纯函数，输入「系统 + 资源信息」，输出界面块（便于单测）。
 *
 * 来源约定：
 * - CC Switch：优先自托管（/downloads/cc-switch/，由 manifest.json 声明），否则回退 GitHub 发布页的固定版本；
 * - Codex 桌面版：agentsmirror 国内可直连镜像；
 * - Claude Code：npm 安装，源用 npmmirror；Node.js 安装包也取 npmmirror 镜像。
 */
import type { Env } from './platform'
import type { GuideBlock, GuideLink } from './viewModel'
import type { GuideCopy } from './copy'

export interface HostedManifest {
  version: string
  files: { name: string; size: number; sha256: string }[]
}

/** 校验自托管 manifest；形状不对（例如 SPA 回退返回了 HTML）一律当作未自托管。 */
export function parseManifest(raw: unknown): HostedManifest | null {
  if (!raw || typeof raw !== 'object') return null
  const { version, files } = raw as { version?: unknown; files?: unknown }
  if (typeof version !== 'string' || !/^v?\d/.test(version) || !Array.isArray(files)) return null
  const valid = files.filter(
    (f): f is HostedManifest['files'][number] =>
      !!f && typeof f.name === 'string' && /^[\w.-]+$/.test(f.name) && typeof f.size === 'number' && typeof f.sha256 === 'string'
  )
  return valid.length ? { version, files: valid } : null
}

// ---------------------------------------------------------------- CC Switch

export const CC_SWITCH_FALLBACK_VERSION = 'v4.0.31'
const CC_SWITCH_RELEASES = 'https://github.com/farion1231/cc-switch/releases'
export const CC_SWITCH_HOSTED_BASE = '/downloads/cc-switch'

type CcAsset = 'win-x64' | 'win-arm64' | 'mac' | 'linux-deb'
const CC_ASSETS: Record<CcAsset, { suffix: string; size: string }> = {
  'win-x64': { suffix: 'Windows.msi', size: '20 MB' },
  'win-arm64': { suffix: 'Windows-arm64.msi', size: '20 MB' },
  mac: { suffix: 'macOS.dmg', size: '41 MB' },
  'linux-deb': { suffix: 'Linux-x86_64.deb', size: '21 MB' }
}

function ccOrder(env: Env): CcAsset[] {
  switch (env.os) {
    case 'windows':
      return env.arch === 'arm64' ? ['win-arm64', 'win-x64'] : ['win-x64', 'win-arm64']
    case 'mac':
      return ['mac']
    case 'linux':
      return ['linux-deb']
    default:
      return ['win-x64', 'mac', 'win-arm64', 'linux-deb']
  }
}

export function buildCcSwitchBlock(env: Env, manifest: HostedManifest | null, c: GuideCopy): GuideBlock {
  const version = manifest?.version ?? CC_SWITCH_FALLBACK_VERSION
  const links = ccOrder(env).map((key): GuideLink => {
    const name = `CC-Switch-${version}-${CC_ASSETS[key].suffix}`
    const hosted = manifest?.files.find((f) => f.name === name)
    return {
      label: c.download.ccsAsset[key],
      href: hosted ? `${CC_SWITCH_HOSTED_BASE}/${name}` : `${CC_SWITCH_RELEASES}/download/${version}/${name}`,
      meta: hosted ? `${Math.max(1, Math.round(hosted.size / 1e6))} MB` : CC_ASSETS[key].size
    }
  })
  const [primary, ...alternates] = links
  const hostedPrimary = primary?.href.startsWith(CC_SWITCH_HOSTED_BASE) ?? false
  return {
    type: 'download',
    heading: c.download.ccsHeading,
    primary,
    alternates,
    note: env.os === 'mac' ? c.download.ccsMacNote : hostedPrimary ? undefined : c.download.ccsGithubNote,
    backup: { label: c.download.ccsBackup, href: `${CC_SWITCH_RELEASES}/latest` }
  }
}

// ---------------------------------------------------------------- Codex 桌面版

const CODEX_BASE = 'https://codexapp.agentsmirror.com/latest'
export const CODEX_CHECKSUMS = `${CODEX_BASE}/checksums`

type CodexAsset = 'win-x64' | 'win-arm64' | 'mac-arm64' | 'mac-intel'
const CODEX_ASSETS: Record<CodexAsset, { size: string }> = {
  'win-x64': { size: '约 1.0 GB' },
  'win-arm64': { size: '约 1.1 GB' },
  'mac-arm64': { size: '约 910 MB' },
  'mac-intel': { size: '约 950 MB' }
}

function codexOrder(env: Env): CodexAsset[] {
  if (env.os === 'windows') return env.arch === 'arm64' ? ['win-arm64', 'win-x64'] : ['win-x64', 'win-arm64']
  if (env.os === 'mac') {
    if (env.arch === 'x64') return ['mac-intel', 'mac-arm64']
    return ['mac-arm64', 'mac-intel']
  }
  return []
}

export const CODEX_NPM_COMMAND = 'npm install -g @openai/codex --registry=https://registry.npmmirror.com'

export function buildCodexBlocks(env: Env, c: GuideCopy): GuideBlock[] {
  const order = codexOrder(env)
  if (!order.length) {
    // Linux / 未知系统：桌面版没有对应安装包，给命令行版。
    return [
      { type: 'note', text: c.download.codexNoDesktop },
      { type: 'command', heading: c.download.codexCliHeading, command: CODEX_NPM_COMMAND, hint: c.download.nodeNeeded }
    ]
  }
  const links = order.map(
    (key): GuideLink => ({ label: c.download.codexAsset[key], href: `${CODEX_BASE}/${key}`, meta: `${CODEX_ASSETS[key].size}` })
  )
  const [primary, ...alternates] = links
  const macUnsure = env.os === 'mac' && env.arch === 'unknown'
  return [
    {
      type: 'download',
      heading: c.download.codexHeading,
      primary,
      alternates,
      note: macUnsure ? c.download.macChipHint : env.os === 'windows' ? c.download.codexWinNote : c.download.codexBigNote,
      verify: { label: c.download.verify, href: CODEX_CHECKSUMS }
    }
  ]
}

// ---------------------------------------------------------------- Claude Code（npm + npmmirror）

export const NPM_MIRROR = 'https://registry.npmmirror.com'
export const NODE_LISTING_URL = `${NPM_MIRROR}/-/binary/node/latest-v22.x/`
export const NODE_FALLBACK_VERSION = '22.20.0'
export const CLAUDE_CODE_COMMAND = `npm install -g @anthropic-ai/claude-code --registry=${NPM_MIRROR}`

/** 从 npmmirror 的目录列表里挑出最高的 Node 版本（按数字比较，不是字符串比较）。 */
export function latestNodeVersion(listing: unknown): string | null {
  if (!Array.isArray(listing)) return null
  let best: number[] | null = null
  let bestText: string | null = null
  for (const item of listing) {
    const m = typeof item?.name === 'string' ? /^node-v(\d+)\.(\d+)\.(\d+)-x64\.msi$/.exec(item.name) : null
    if (!m) continue
    const parts = [Number(m[1]), Number(m[2]), Number(m[3])]
    if (!best || parts[0]! > best[0]! || (parts[0] === best[0] && (parts[1]! > best[1]! || (parts[1] === best[1] && parts[2]! > best[2]!)))) {
      best = parts
      bestText = `${m[1]}.${m[2]}.${m[3]}`
    }
  }
  return bestText
}

function nodeLink(env: Env, version: string, c: GuideCopy): GuideLink | null {
  const file = (name: string) => `${NODE_LISTING_URL}${name}`
  if (env.os === 'windows') {
    const arm = env.arch === 'arm64'
    return { label: arm ? c.download.nodeWinArm : c.download.nodeWin, href: file(`node-v${version}-${arm ? 'arm64' : 'x64'}.msi`), meta: `Node ${version}` }
  }
  if (env.os === 'mac') return { label: c.download.nodeMac, href: file(`node-v${version}.pkg`), meta: `Node ${version}` }
  return null
}

export function buildClaudeCodeBlocks(env: Env, nodeVersion: string, c: GuideCopy): GuideBlock[] {
  const node = nodeLink(env, nodeVersion, c)
  const otherWin: GuideLink[] =
    env.os === 'windows'
      ? [
          {
            label: env.arch === 'arm64' ? c.download.nodeWin : c.download.nodeWinArm,
            href: `${NODE_LISTING_URL}node-v${nodeVersion}-${env.arch === 'arm64' ? 'x64' : 'arm64'}.msi`,
            meta: `Node ${nodeVersion}`
          }
        ]
      : []
  return [
    node
      ? { type: 'download', heading: c.download.nodeHeading, primary: node, alternates: otherWin, note: c.download.nodeNote }
      : { type: 'note', text: c.download.nodeLinux },
    { type: 'command', heading: c.download.claudeHeading, command: CLAUDE_CODE_COMMAND, hint: env.os === 'windows' ? c.download.shellWin : c.download.shellUnix },
    { type: 'command', heading: c.download.claudeCheck, command: 'claude --version', hint: c.download.claudeCheckHint }
  ]
}

/** 下载页完整内容：CC Switch + 所选工具；手机上只给提示和网址。 */
export function buildDownloadBlocks(
  env: Env,
  client: 'codex' | 'claude-code',
  resources: { manifest: HostedManifest | null; nodeVersion: string; siteUrl: string },
  c: GuideCopy
): GuideBlock[] {
  if (env.os === 'mobile') {
    return [
      { type: 'note', text: c.download.mobile },
      { type: 'copy', label: c.download.siteUrl, value: resources.siteUrl }
    ]
  }
  return [
    buildCcSwitchBlock(env, resources.manifest, c),
    ...(client === 'codex' ? buildCodexBlocks(env, c) : buildClaudeCodeBlocks(env, resources.nodeVersion, c))
  ]
}
