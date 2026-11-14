export type OS = 'windows' | 'mac' | 'linux' | 'mobile' | 'unknown'
export type Arch = 'arm64' | 'x64' | 'unknown'

export interface Env {
  os: OS
  arch: Arch
}

export function detectOS(
  ua: string = navigator.userAgent,
  platform: string = navigator.platform ?? '',
  touchPoints: number = navigator.maxTouchPoints ?? 0
): OS {
  if (/Android|iPhone|iPad|iPod|Mobile/i.test(ua)) return 'mobile'
  // iPadOS 的桌面模式 UA 自称 Macintosh，但有触摸点。
  if (/Mac/i.test(platform) && touchPoints > 1) return 'mobile'
  if (/Win/i.test(platform) || /Windows/i.test(ua)) return 'windows'
  if (/Mac/i.test(platform) || /Macintosh|Mac OS X/i.test(ua)) return 'mac'
  if (/Linux|X11/i.test(ua)) return 'linux'
  return 'unknown'
}

function webglRenderer(): string {
  try {
    const gl = document.createElement('canvas').getContext('webgl')
    const ext = gl?.getExtension('WEBGL_debug_renderer_info')
    return ext && gl ? String(gl.getParameter(ext.UNMASKED_RENDERER_WEBGL)) : ''
  } catch {
    return ''
  }
}

/**
 * 判断 CPU 架构。浏览器不直接暴露，只能尽力推断：
 * Chromium 有 userAgentData；Mac 上看显卡名（Apple M 系列 = Apple 芯片）。
 * 推断不出来返回 unknown，由界面把几个版本都列出来让用户自己选。
 */
export async function detectArch(os: OS): Promise<Arch> {
  const uad = (navigator as Navigator & { userAgentData?: { getHighEntropyValues?: (hints: string[]) => Promise<{ architecture?: string }> } }).userAgentData
  if (uad?.getHighEntropyValues) {
    try {
      const { architecture } = await uad.getHighEntropyValues(['architecture'])
      if (architecture === 'arm') return 'arm64'
      if (architecture === 'x86') return 'x64'
    } catch {
      /* 被拒绝就走下面的推断 */
    }
  }
  if (os === 'mac') {
    const renderer = webglRenderer()
    if (/Apple M\d/i.test(renderer)) return 'arm64'
    if (/Intel|AMD|Radeon|NVIDIA/i.test(renderer)) return 'x64'
    return 'unknown'
  }
  if (os === 'windows' || os === 'linux') return 'x64'
  return 'unknown'
}
