import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { usePlazaData } from '../usePlazaData'

const { getModelPlaza, app, auth } = vi.hoisted(() => ({
  getModelPlaza: vi.fn(),
  app: {
    publicSettingsLoaded: true,
    cachedPublicSettings: null as Record<string, unknown> | null,
    fetchPublicSettings: vi.fn()
  },
  auth: { isAuthenticated: false }
}))

vi.mock('@/api/modelPlaza', () => ({ getModelPlaza }))
vi.mock('@/stores', () => ({ useAppStore: () => app, useAuthStore: () => auth }))

function setup(settings: Record<string, unknown>, authenticated: boolean) {
  app.publicSettingsLoaded = true
  app.cachedPublicSettings = reactive({ ...settings })
  auth.isAuthenticated = authenticated
  return usePlazaData()
}

describe('usePlazaData', () => {
  beforeEach(() => {
    getModelPlaza.mockReset()
    getModelPlaza.mockResolvedValue({ description: '', groups: [] })
  })

  it('asks anonymous visitors to sign in without calling the API when sign-in is required', async () => {
    const plaza = setup({ model_plaza_enabled: true, model_plaza_require_auth: true }, false)
    await plaza.init()

    expect(plaza.status.value).toBe('auth')
    // 不发请求：否则后端返回 401，会被全局拦截器直接跳去登录页。
    expect(getModelPlaza).not.toHaveBeenCalled()
  })

  it('loads the plaza for signed-in users when sign-in is required', async () => {
    const plaza = setup({ model_plaza_enabled: true, model_plaza_require_auth: true }, true)
    await plaza.init()

    expect(getModelPlaza).toHaveBeenCalledTimes(1)
    expect(plaza.status.value).toBe('ready')
  })

  it('stays public for anonymous visitors only when sign-in is not required', async () => {
    const plaza = setup({ model_plaza_enabled: true, model_plaza_require_auth: false }, false)
    await plaza.init()

    expect(getModelPlaza).toHaveBeenCalledTimes(1)
    expect(plaza.status.value).toBe('ready')
  })

  it('reports disabled without calling the API when the plaza is switched off', async () => {
    const plaza = setup({ model_plaza_enabled: false, model_plaza_require_auth: false }, true)
    await plaza.init()

    expect(plaza.status.value).toBe('disabled')
    expect(getModelPlaza).not.toHaveBeenCalled()
  })
})
