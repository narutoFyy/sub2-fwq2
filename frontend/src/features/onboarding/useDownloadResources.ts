import { ref } from 'vue'
import { CC_SWITCH_HOSTED_BASE, NODE_FALLBACK_VERSION, NODE_LISTING_URL, latestNodeVersion, parseManifest, type HostedManifest } from './downloads'

async function fetchJson(url: string, timeoutMs = 4000): Promise<unknown | null> {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), timeoutMs)
  try {
    const response = await fetch(url, { signal: controller.signal, cache: 'no-store' })
    if (!response.ok || !/json/i.test(response.headers.get('content-type') ?? '')) return null
    return await response.json()
  } catch {
    return null
  } finally {
    clearTimeout(timer)
  }
}

/**
 * 下载页需要的两份动态信息，只在进入下载步骤时加载一次：
 * - 自托管 manifest（没部署时 404 / 返回 HTML，视为未自托管，回退 GitHub）；
 * - npmmirror 上最新的 Node.js 22 版本（取不到就用写死的兜底版本）。
 */
export function useDownloadResources() {
  const manifest = ref<HostedManifest | null>(null)
  const nodeVersion = ref(NODE_FALLBACK_VERSION)
  const ready = ref(false)
  let started = false

  async function load() {
    if (started) return
    started = true
    const [hosted, listing] = await Promise.all([
      fetchJson(`${CC_SWITCH_HOSTED_BASE}/manifest.json`),
      fetchJson(NODE_LISTING_URL)
    ])
    manifest.value = parseManifest(hosted)
    nodeVersion.value = latestNodeVersion(listing) ?? NODE_FALLBACK_VERSION
    ready.value = true
  }

  return { manifest, nodeVersion, ready, load }
}
