import { computed, ref, watch } from 'vue'
import { getModelPlaza, type ModelPlazaResponse } from '@/api/modelPlaza'
import { useAppStore, useAuthStore } from '@/stores'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'

/**
 * 广场数据加载状态：
 * - disabled：后台关闭了广场；
 * - auth：后台要求登录、当前未登录（不发请求，否则 401 会被全局拦截器跳去登录页）；
 * - loading / ready / error。
 */
export type PlazaStatus = 'loading' | 'ready' | 'error' | 'disabled' | 'auth'

export function usePlazaData() {
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const data = ref<ModelPlazaResponse | null>(null)
  const fetchState = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')

  const enabled = computed(() => resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.modelPlaza))
  const needsLogin = computed(
    () => appStore.cachedPublicSettings?.model_plaza_require_auth === true && !authStore.isAuthenticated
  )

  const status = computed<PlazaStatus>(() => {
    if (!appStore.publicSettingsLoaded && fetchState.value === 'idle') return 'loading'
    if (!enabled.value) return 'disabled'
    if (needsLogin.value) return 'auth'
    if (fetchState.value === 'ready') return 'ready'
    if (fetchState.value === 'error') return 'error'
    return 'loading'
  })

  async function load() {
    if (!enabled.value || needsLogin.value || fetchState.value === 'loading') return
    fetchState.value = 'loading'
    try {
      data.value = await getModelPlaza()
      fetchState.value = 'ready'
    } catch {
      fetchState.value = 'error'
    }
  }

  async function init() {
    if (!appStore.publicSettingsLoaded) {
      try {
        await appStore.fetchPublicSettings()
      } catch {
        /* 设置加载失败时按未知处理，由接口兜底 */
      }
    }
    await load()
  }

  // 登录态变化（如在别的标签页登录）后重新取数，以拿到专属分组与倍率。
  watch(
    () => authStore.isAuthenticated,
    () => {
      fetchState.value = 'idle'
      void load()
    }
  )

  return { data, status, init, reload: load }
}
