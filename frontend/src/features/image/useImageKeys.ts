import { computed, ref } from 'vue'
import { keysAPI, userGroupsAPI } from '@/api'
import type { ApiKey } from '@/types'
import { keyCanGenerateImages } from './skillPrompt'

/** 读取当前账号能生图的密钥（主分组或附加分组开了生图）。 */
export function useImageKeys() {
  const keys = ref<ApiKey[]>([])
  const loading = ref(false)
  const loaded = ref(false)

  async function load() {
    loading.value = true
    try {
      const [page, groups] = await Promise.all([
        keysAPI.list(1, 100, { status: 'active' }),
        userGroupsAPI.getAvailable().catch(() => [])
      ])
      const imageGroups = new Set(groups.filter((g) => g.allow_image_generation).map((g) => g.id))
      keys.value = page.items.filter((key) => keyCanGenerateImages(key, imageGroups))
    } catch {
      keys.value = []
    } finally {
      loading.value = false
      loaded.value = true
    }
  }

  return { keys, loading, loaded, load, hasKeys: computed(() => keys.value.length > 0) }
}
