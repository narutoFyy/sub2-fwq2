<template>
  <BaseDialog :show="show" title="OAuth 账号监控" width="wide" @close="emit('close')">
    <form class="space-y-5" @submit.prevent="submit">
      <div class="rounded-lg bg-blue-50 p-4 text-sm text-blue-800 dark:bg-blue-900/20 dark:text-blue-200">
        已选择 {{ accountIds.length }} 个 OpenAI/Codex OAuth 账号。额度按主窗口或次窗口任一剩余百分比低于阈值触发。
      </div>
      <label class="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-dark-600">
        <span class="font-medium text-gray-800 dark:text-gray-100">启用账号监控</span>
        <input v-model="form.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />
      </label>
      <div class="grid gap-4 sm:grid-cols-2">
        <label class="input-label">额度剩余阈值 (%)
          <input v-model.number="form.quota_threshold_percent" type="number" min="0" max="100" step="1" class="input mt-1 w-full" />
        </label>
        <label class="input-label">检查间隔 (分钟)
          <input v-model.number="form.interval_minutes" type="number" min="1" max="1440" step="1" class="input mt-1 w-full" />
        </label>
      </div>
      <div class="space-y-2 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
        <div class="text-sm font-semibold text-gray-800 dark:text-gray-100">告警条件</div>
        <label class="flex items-center gap-2 text-sm"><input v-model="form.notify_on_error" type="checkbox" class="rounded text-primary-600" />账号状态异常</label>
        <label class="flex items-center gap-2 text-sm"><input v-model="form.notify_on_quota" type="checkbox" class="rounded text-primary-600" />额度剩余低于阈值</label>
        <label class="flex items-center gap-2 text-sm"><input v-model="form.notify_on_recovery" type="checkbox" class="rounded text-primary-600" />恢复时发送通知</label>
      </div>
      <div class="space-y-2 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
        <div class="text-sm font-semibold text-gray-800 dark:text-gray-100">通知渠道</div>
        <label class="flex items-center gap-2 text-sm"><input v-model="form.notify_email" type="checkbox" class="rounded text-primary-600" />邮件（沿用 Ops 收件人）</label>
        <label class="flex items-center gap-2 text-sm"><input v-model="form.notify_pushplus" type="checkbox" class="rounded text-primary-600" />PushPlus</label>
        <label class="input-label mt-2 block">PushPlus Token
          <input v-model="push.token" type="password" autocomplete="off" placeholder="留空保持原 token" class="input mt-1 w-full" />
        </label>
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="input-label">模板
            <select v-model="push.template" class="input mt-1 w-full"><option value="html">html</option><option value="markdown">markdown</option><option value="txt">txt</option></select>
          </label>
          <label class="input-label">渠道（可选）
            <input v-model="push.channel" class="input mt-1 w-full" placeholder="wechat" />
          </label>
        </div>
        <label class="flex items-center gap-2 text-sm"><input v-model="push.enabled" type="checkbox" class="rounded text-primary-600" />启用 PushPlus</label>
        <p class="text-xs text-gray-500 dark:text-gray-400">持续异常按每次检查发送，默认每 5 分钟一次。</p>
      </div>
      <div class="flex justify-end gap-2 border-t border-gray-200 pt-4 dark:border-dark-600">
        <button type="button" class="btn btn-secondary" @click="emit('close')">取消</button>
        <button type="submit" class="btn btn-primary" :disabled="saving">{{ saving ? '保存中...' : '保存监控设置' }}</button>
      </div>
    </form>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import type { OAuthAccountMonitorConfig, OAuthMonitorPushPlusConfig } from '@/api/admin/accounts'

const props = defineProps<{ show: boolean; accountIds: number[]; config: OAuthAccountMonitorConfig | null; pushConfig?: OAuthMonitorPushPlusConfig | null; saving?: boolean }>()
const emit = defineEmits<{ close: []; save: [config: OAuthAccountMonitorConfig, push: OAuthMonitorPushPlusConfig] }>()
const form = reactive<OAuthAccountMonitorConfig>({ enabled: true, account_ids: [], quota_threshold_percent: 20, interval_minutes: 5, notify_on_error: true, notify_on_quota: true, notify_on_recovery: true, notify_email: true, notify_pushplus: true, repeat_every_check: true })
const push = reactive<OAuthMonitorPushPlusConfig>({ enabled: false, token: '', topic: '', template: 'html', channel: '' })
watch(() => [props.show, props.config, props.accountIds, props.pushConfig], () => { Object.assign(form, props.config ?? {}, { account_ids: [...props.accountIds] }); Object.assign(push, props.pushConfig ?? {}) }, { immediate: true, deep: true })
const submit = () => emit('save', { ...form, account_ids: [...props.accountIds] }, { ...push })
</script>
