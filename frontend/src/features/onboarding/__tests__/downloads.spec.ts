import { describe, expect, it } from 'vitest'
import { guideCopyEn, guideCopyZh } from '../copy'
import {
  CC_SWITCH_FALLBACK_VERSION,
  CLAUDE_CODE_COMMAND,
  CODEX_NPM_COMMAND,
  buildCcSwitchBlock,
  buildClaudeCodeBlocks,
  buildCodexBlocks,
  buildDownloadBlocks,
  latestNodeVersion,
  parseManifest,
  type HostedManifest
} from '../downloads'
import { detectOS, type Env } from '../platform'
import type { GuideBlock } from '../viewModel'

const c = guideCopyZh
const win: Env = { os: 'windows', arch: 'x64' }
const winArm: Env = { os: 'windows', arch: 'arm64' }
const macArm: Env = { os: 'mac', arch: 'arm64' }
const macUnknown: Env = { os: 'mac', arch: 'unknown' }
const linux: Env = { os: 'linux', arch: 'x64' }

const hosted: HostedManifest = {
  version: 'v4.0.40',
  files: [
    { name: 'CC-Switch-v4.0.40-Windows.msi', size: 21_000_000, sha256: 'a' },
    { name: 'CC-Switch-v4.0.40-macOS.dmg', size: 42_000_000, sha256: 'b' }
  ]
}

function download(block: GuideBlock) {
  if (block.type !== 'download') throw new Error(`expected a download block, got ${block.type}`)
  return block
}

describe('parseManifest', () => {
  it('accepts a well-formed manifest and drops malformed files', () => {
    const parsed = parseManifest({ version: 'v1.2.3', files: [{ name: 'a.msi', size: 1, sha256: 'x' }, { name: '../evil', size: 1, sha256: 'x' }, { nope: true }] })
    expect(parsed?.files.map((f) => f.name)).toEqual(['a.msi'])
  })

  it('rejects anything else (e.g. an HTML fallback page parsed as null)', () => {
    expect(parseManifest(null)).toBeNull()
    expect(parseManifest('<html>')).toBeNull()
    expect(parseManifest({ version: 'abc', files: [] })).toBeNull()
    expect(parseManifest({ version: 'v1', files: [] })).toBeNull()
  })
})

describe('CC Switch block', () => {
  it('falls back to a pinned GitHub release when nothing is self-hosted', () => {
    const block = download(buildCcSwitchBlock(win, null, c))
    expect(block.primary?.href).toBe(`https://github.com/farion1231/cc-switch/releases/download/${CC_SWITCH_FALLBACK_VERSION}/CC-Switch-${CC_SWITCH_FALLBACK_VERSION}-Windows.msi`)
    expect(block.note).toBe(c.download.ccsGithubNote)
    expect(block.alternates?.[0]?.label).toBe(c.download.ccsAsset['win-arm64'])
  })

  it('prefers the self-hosted file for the manifest version and keeps GitHub as backup', () => {
    const block = download(buildCcSwitchBlock(win, hosted, c))
    expect(block.primary?.href).toBe('/downloads/cc-switch/CC-Switch-v4.0.40-Windows.msi')
    expect(block.primary?.meta).toBe('21 MB')
    expect(block.note).toBeUndefined()
    expect(block.backup?.href).toContain('github.com/farion1231/cc-switch/releases')
  })

  it('uses GitHub for platforms the manifest does not host', () => {
    const block = download(buildCcSwitchBlock(winArm, hosted, c))
    expect(block.primary?.href).toBe('https://github.com/farion1231/cc-switch/releases/download/v4.0.40/CC-Switch-v4.0.40-Windows-arm64.msi')
    expect(block.alternates?.[0]?.href).toBe('/downloads/cc-switch/CC-Switch-v4.0.40-Windows.msi')
  })

  it('adds the Gatekeeper hint on macOS', () => {
    expect(download(buildCcSwitchBlock(macArm, hosted, c)).note).toBe(c.download.ccsMacNote)
  })
})

describe('Codex block', () => {
  it('offers the mirror build for the detected chip and lists the other as an alternative', () => {
    const block = download(buildCodexBlocks(macArm, c)[0]!)
    expect(block.primary?.href).toBe('https://codexapp.agentsmirror.com/latest/mac-arm64')
    expect(block.alternates?.[0]?.href).toBe('https://codexapp.agentsmirror.com/latest/mac-intel')
    expect(block.verify?.href).toBe('https://codexapp.agentsmirror.com/latest/checksums')
  })

  it('explains how to tell the chip when it cannot be detected', () => {
    expect(download(buildCodexBlocks(macUnknown, c)[0]!).note).toBe(c.download.macChipHint)
  })

  it('picks the ARM Windows build on ARM', () => {
    expect(download(buildCodexBlocks(winArm, c)[0]!).primary?.href).toBe('https://codexapp.agentsmirror.com/latest/win-arm64')
  })

  it('falls back to the CLI on systems without a desktop build', () => {
    const blocks = buildCodexBlocks(linux, c)
    expect(blocks[0]).toEqual({ type: 'note', text: c.download.codexNoDesktop })
    expect(blocks[1]).toMatchObject({ type: 'command', command: CODEX_NPM_COMMAND })
  })
})

describe('Claude Code blocks', () => {
  it('installs Node from npmmirror, then Claude Code through the npmmirror registry', () => {
    const blocks = buildClaudeCodeBlocks(win, '22.20.0', c)
    expect(download(blocks[0]!).primary?.href).toBe('https://registry.npmmirror.com/-/binary/node/latest-v22.x/node-v22.20.0-x64.msi')
    expect(blocks[1]).toMatchObject({ type: 'command', command: CLAUDE_CODE_COMMAND })
    expect(CLAUDE_CODE_COMMAND).toContain('--registry=https://registry.npmmirror.com')
    expect(blocks[2]).toMatchObject({ type: 'command', command: 'claude --version' })
  })

  it('uses the pkg on macOS and just a note on Linux', () => {
    expect(download(buildClaudeCodeBlocks(macArm, '22.20.0', c)[0]!).primary?.href).toBe('https://registry.npmmirror.com/-/binary/node/latest-v22.x/node-v22.20.0.pkg')
    expect(buildClaudeCodeBlocks(linux, '22.20.0', c)[0]).toEqual({ type: 'note', text: c.download.nodeLinux })
  })
})

describe('download page content', () => {
  const resources = { manifest: null, nodeVersion: '22.20.0', siteUrl: 'https://yf-mail.com' }

  it('puts CC Switch first, then the chosen tool', () => {
    const codex = buildDownloadBlocks(win, 'codex', resources, c)
    expect(download(codex[0]!).heading).toBe(c.download.ccsHeading)
    expect(download(codex[1]!).heading).toBe(c.download.codexHeading)
    const claude = buildDownloadBlocks(win, 'claude-code', resources, c)
    expect(claude).toHaveLength(4)
  })

  it('tells phone users to continue on a computer and offers the site address', () => {
    const blocks = buildDownloadBlocks({ os: 'mobile', arch: 'unknown' }, 'codex', resources, guideCopyEn)
    expect(blocks[0]).toEqual({ type: 'note', text: guideCopyEn.download.mobile })
    expect(blocks[1]).toMatchObject({ type: 'copy', value: 'https://yf-mail.com' })
  })
})

describe('latestNodeVersion', () => {
  it('compares versions numerically, not alphabetically', () => {
    const listing = ['node-v22.9.0-x64.msi', 'node-v22.20.0-x64.msi', 'node-v22.18.1-x64.msi', 'node-v22.20.0.pkg'].map((name) => ({ name }))
    expect(latestNodeVersion(listing)).toBe('22.20.0')
  })

  it('returns null for unusable listings', () => {
    expect(latestNodeVersion(null)).toBeNull()
    expect(latestNodeVersion([{ name: 'readme.txt' }])).toBeNull()
  })
})

describe('detectOS', () => {
  it('recognises the common desktop and mobile platforms', () => {
    expect(detectOS('Mozilla/5.0 (Windows NT 10.0; Win64; x64)', 'Win32', 0)).toBe('windows')
    expect(detectOS('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', 'MacIntel', 0)).toBe('mac')
    expect(detectOS('Mozilla/5.0 (X11; Linux x86_64)', 'Linux x86_64', 0)).toBe('linux')
    expect(detectOS('Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)', 'iPhone', 5)).toBe('mobile')
    expect(detectOS('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', 'MacIntel', 5)).toBe('mobile')
  })
})
