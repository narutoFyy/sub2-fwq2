import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, put } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, put }
}))

import {
  getOAuthMonitorEmailConfig,
  updateOAuthMonitorEmailConfig
} from '@/api/admin/accounts'

describe('admin OAuth account monitor email API', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
  })

  it('uses the OAuth-specific email configuration endpoints', async () => {
    const config = { enabled: true, recipients: ['admin@example.com'] }
    get.mockResolvedValueOnce({ data: config })
    put.mockResolvedValueOnce({ data: config })

    await expect(getOAuthMonitorEmailConfig()).resolves.toEqual(config)
    await expect(updateOAuthMonitorEmailConfig(config)).resolves.toEqual(config)

    expect(get).toHaveBeenCalledWith('/admin/accounts/monitoring/email')
    expect(put).toHaveBeenCalledWith('/admin/accounts/monitoring/email', config)
  })
})
