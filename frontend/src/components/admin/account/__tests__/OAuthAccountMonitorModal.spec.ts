import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import OAuthAccountMonitorModal from '../OAuthAccountMonitorModal.vue'
import type { OAuthAccountMonitorOverview } from '@/api/admin/accounts'

function createOverview(): OAuthAccountMonitorOverview {
  return {
    config: {
      enabled: true,
      account_ids: [7],
      quota_threshold_percent: 20,
      interval_minutes: 5,
      notify_on_error: false,
      notify_on_quota: true,
      notify_on_recovery: false,
      notify_email: false,
      notify_pushplus: false,
      repeat_every_check: false,
      notify_on_unavailable: true,
      unavailable_failure_threshold: 3,
      unavailable_window_minutes: 15,
      unavailable_reminder_minutes: 60
    },
    email: { enabled: false, recipients: ['admin@example.com'] },
    pushplus: { enabled: false, token: '', template: 'html' },
    accounts: [{
      account_id: 7,
      account_name: 'Codex OAuth',
      account_status: 'active',
      schedulable: true,
      selected: true,
      eligible: true,
      monitored: true,
      effective: true,
      monitor_status: 'error',
      state: {
        account_id: 7,
        health_status: 'error',
        quota_status: 'ok',
        consecutive_failures: 3,
        last_error_code: '502',
        failure_sequence: 3,
        last_notified_failure_sequence: 3,
        quota_alert_active: false,
        updated_at: '2026-09-03T01:00:00Z'
      }
    }]
  }
}

describe('OAuthAccountMonitorModal', () => {
  it('shows only low-quota and sustained-unavailable alert rules', () => {
    const wrapper = mount(OAuthAccountMonitorModal, {
      props: { show: true, accountIds: [7], overview: createOverview() },
      global: { stubs: { Teleport: true, Icon: true } }
    })

    expect(wrapper.text()).toContain('低额度提醒')
    expect(wrapper.text()).toContain('持续不可用提醒')
    expect(wrapper.text()).not.toContain('账号错误时告警')
    expect(wrapper.text()).not.toContain('恢复时发送通知')
    expect(wrapper.text()).not.toContain('持续异常每次检查通知')
    expect(wrapper.text()).toContain('连续失败 3 次')
    expect(wrapper.text()).toContain('错误码 502')
    expect(wrapper.find('details').attributes('open')).toBeUndefined()

    wrapper.unmount()
  })

  it('emits the OAuth-specific email shape when saving', async () => {
    const wrapper = mount(OAuthAccountMonitorModal, {
      props: { show: true, accountIds: [7], overview: createOverview() },
      global: { stubs: { Teleport: true, Icon: true } }
    })

    await wrapper.get('form').trigger('submit')
    const emitted = wrapper.emitted('save')
    expect(emitted).toHaveLength(1)
    expect(emitted?.[0]?.[2]).toEqual({ enabled: false, recipients: ['admin@example.com'] })
    expect(emitted?.[0]?.[0]).toMatchObject({
      notify_on_error: false,
      notify_on_recovery: false,
      repeat_every_check: false,
      notify_on_quota: true,
      notify_on_unavailable: true
    })

    wrapper.unmount()
  })
})
