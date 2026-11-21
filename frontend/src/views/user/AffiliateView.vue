<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <template v-else-if="detail">
        <!-- 活动主视觉口号横幅 -->
        <div class="relative overflow-hidden rounded-2xl bg-gradient-to-r from-primary-700 via-blue-600 to-indigo-600 p-6 text-white shadow-md">
          <div class="relative flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div class="space-y-2">
              <div class="inline-flex items-center gap-2 rounded-full bg-white/15 px-3 py-1 text-xs font-medium backdrop-blur">
                <span>拉新返利活动 · 正式版 v2</span>
              </div>
              <h2 class="text-xl font-bold tracking-tight sm:text-2xl">
                好友注册领 1 元体验，邀 1 人最高得 8 元额度 + {{ formattedRebateRate }}% 永久返现
              </h2>
              <p class="max-w-3xl text-sm text-white/85">
                好友通过邀请链接注册并完成验证立得 1 元体验金（7 天有效）；好友实付满 10 元且消耗满 3 元您得 3 元余额，实付满 50 元再得 5 元（合计 8 元）；拉满阶梯人数另有最高 200 元余额大奖！
              </p>
            </div>
            <div class="flex shrink-0 flex-wrap gap-3">
              <div class="rounded-xl border border-white/20 bg-white/10 px-4 py-3 text-center backdrop-blur">
                <p class="text-xs text-white/75">单人两步奖待到账</p>
                <p class="mt-1 text-xl font-bold text-amber-200">{{ formatCurrency(pendingStepBonus) }}</p>
              </div>
              <div class="rounded-xl border border-white/20 bg-white/10 px-4 py-3 text-center backdrop-blur">
                <p class="text-xs text-white/75">活动余额已到账</p>
                <p class="mt-1 text-xl font-bold text-emerald-200">{{ formatCurrency(totalGrantedActivityBonus) }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- 好友已充10元但未消费满3元时的醒目解锁提示 -->
        <div
          v-if="remindConsumeCount > 0"
          data-test="affiliate-consume-reminder"
          class="flex flex-col gap-3 rounded-xl border border-amber-300 bg-amber-50 p-4 text-amber-900 dark:border-amber-700/60 dark:bg-amber-950/30 dark:text-amber-200 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="flex items-start gap-3">
            <Icon name="sparkles" size="md" class="mt-0.5 shrink-0 text-amber-600 dark:text-amber-400" />
            <div class="text-sm">
              <p class="font-semibold">
                您有 {{ remindConsumeCount }} 位好友已充值达标，待消耗满 3 元即可解锁奖励（待到账 {{ formatCurrency(pendingStepBonus) }}）
              </p>
              <p class="mt-0.5 text-amber-800 dark:text-amber-300">
                提醒好友配置 API Key 试用即可解锁 3 元（若首充已满 50 元可一次性解锁 8 元）！
              </p>
            </div>
          </div>
        </div>

        <!-- 顶部 4 项核心指标 -->
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <p class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
              <Icon name="dollar" size="sm" class="text-primary-500" />
              {{ t('affiliate.stats.rebateRate') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">
              {{ formattedRebateRate }}<span class="ml-0.5 text-base font-medium">%</span>
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('affiliate.stats.rebateRateHint') }}（冻结 72 小时解冻）
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.invitedUsers') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCount(detail.aff_count) }}
            </p>
            <p class="mt-1 text-xs text-emerald-600 dark:text-emerald-400">
              有效邀请（满 10 元 + 耗 3 元）：{{ formatCount(qualifiedInviteeCount) }} 人
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.availableQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">
              {{ formatCurrency(detail.aff_quota) }}
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              随时 1:1 转余额 · 满 {{ withdrawThreshold }} 元可联系客服提现
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.totalQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCurrency(detail.aff_history_quota) }}
            </p>
            <p v-if="detail.aff_frozen_quota > 0" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {{ t('affiliate.stats.frozenQuota') }}（72h 内解冻）: {{ formatCurrency(detail.aff_frozen_quota) }}
            </p>
            <p v-else class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              两步奖已到账 {{ formatCurrency(grantedStepBonus) }} · 待到账 {{ formatCurrency(pendingStepBonus) }}
            </p>
          </div>
        </div>

        <!-- 邀请码与邀请链接卡片 -->
        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.title') }}</h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.description') }}</p>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.yourCode') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm font-semibold text-gray-900 dark:text-white sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm text-gray-700 dark:text-gray-300 sm:flex-1 sm:truncate">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-900/40 dark:bg-primary-900/20">
            <p class="text-sm font-medium text-primary-800 dark:text-primary-200">{{ t('affiliate.tips.title') }}</p>
            <ul class="mt-2 space-y-1 text-sm text-primary-700 dark:text-primary-300">
              <li>1. {{ t('affiliate.tips.line1') }}（主流邮箱通过邀请链接注册立送 1 元体验金，7 天有效）。</li>
              <li>2. 单人两步奖：好友注册 30 天内累计实付满 10 元且消耗满 3 元，您得 3 元余额；累计实付满 50 元且消耗满 3 元，再得 5 元余额（合计 8 元）。</li>
              <li>3. {{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}（每笔冻结 72 小时后进入可用返利，可随时 1:1 转余额，或满 300 元联系客服人工提现）。</li>
              <li v-if="detail.aff_frozen_quota > 0">4. {{ t('affiliate.tips.line4') }}</li>
            </ul>
          </div>
        </div>

        <!-- 阶梯奖进度条（按档补差额，封顶 200 元余额） -->
        <div class="card p-6" data-test="affiliate-ladder-card">
          <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">
                阶梯达标大奖（按档补差额，最高 200 元站内余额）
              </h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                「有效邀请」定义：通过邀请链接注册，且在注册后 30 天内满足「累计实付满 10 元 + 累计消耗满 3 元」的新用户。
              </p>
            </div>
            <div class="text-right">
              <span class="inline-flex items-center rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
                当前有效邀请：{{ qualifiedInviteeCount }} / 100 人
              </span>
            </div>
          </div>

          <!-- 进度条 -->
          <div class="mt-4">
            <div class="h-3 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-800">
              <div
                class="h-full rounded-full bg-gradient-to-r from-primary-500 to-emerald-500 transition-all duration-500"
                :style="{ width: `${ladderProgressPercent}%` }"
              ></div>
            </div>
            <div class="mt-2 flex flex-wrap items-center justify-between text-xs text-gray-500 dark:text-dark-400">
              <span>已解锁阶梯奖：{{ formatCurrency(grantedLadderBonus) }}（待审核：{{ formatCurrency(pendingLadderBonus) }}）</span>
              <span>有效邀请人累计实付：{{ formatCurrency(qualifiedInviteesPaidTotal) }}（100 人档需满 3000 元）</span>
            </div>
          </div>

          <!-- 4 档阶梯卡片 -->
          <div class="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div
              v-for="tier in ladderTiers"
              :key="tier.tier_target"
              class="rounded-xl border p-4 transition-colors"
              :class="tierCardClass(tier)"
            >
              <div class="flex items-center justify-between">
                <span class="text-sm font-semibold text-gray-900 dark:text-white">
                  累计达 {{ tier.tier_target }} 人
                </span>
                <span class="rounded-full px-2 py-0.5 text-xs font-medium" :class="tierBadgeClass(tier)">
                  {{ tierStatusLabel(tier) }}
                </span>
              </div>
              <p class="mt-2 text-2xl font-bold text-primary-600 dark:text-primary-400">
                +{{ tier.delta_bonus }} 元
              </p>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                本档累计总奖 {{ tier.cumulative_bonus }} 元 · {{ tier.auto_grant ? '自动发放' : '人工审核发放' }}
              </p>
              <p v-if="tier.requires_min_total_paid > 0" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
                需这 100 人累计实付满 {{ tier.requires_min_total_paid }} 元
              </p>
            </div>
          </div>
        </div>

        <!-- 可用返利 1:1 转余额 & 满 300 元人工提现指引 -->
        <div class="grid gap-4 lg:grid-cols-2">
          <div class="card p-6">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.transfer.title') }}</h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  {{ t('affiliate.transfer.description') }}（无门槛随时 1:1 兑换为站内可用余额）
                </p>
              </div>
              <button
                class="btn btn-primary"
                :disabled="transferring || detail.aff_quota <= 0"
                @click="transferQuota"
              >
                <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
                <Icon v-else name="dollar" size="sm" />
                <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
              </button>
            </div>
            <p v-if="detail.aff_quota <= 0" class="mt-3 text-sm text-amber-600 dark:text-amber-400">
              {{ t('affiliate.transfer.empty') }}
            </p>
          </div>

          <div class="card p-6" data-test="affiliate-withdraw-guide">
            <div class="flex items-start justify-between gap-3">
              <div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-white">
                  满 {{ withdrawThreshold }} 元人工核验提现（全场唯一可提现渠道）
                </h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  5% 长期返利冻结 72 小时解冻后进入可用返利额度；攒满 {{ withdrawThreshold }} 元可联系客服人工核验订单与行为记录后转账。
                </p>
              </div>
              <span
                class="shrink-0 rounded-full px-2.5 py-1 text-xs font-semibold"
                :class="detail.aff_quota >= withdrawThreshold
                  ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300'
                  : 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300'"
              >
                {{ detail.aff_quota >= withdrawThreshold ? '已满 300 元可申请' : `差 ${formatCurrency(Math.max(0, withdrawThreshold - detail.aff_quota))}` }}
              </span>
            </div>
            <div class="mt-3 rounded-lg bg-gray-50 p-3 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">
              <span v-if="detail.aff_quota >= withdrawThreshold" class="font-medium text-emerald-700 dark:text-emerald-300">
                您的可用返利额度已达 {{ formatCurrency(detail.aff_quota) }}！请联系官方客服或加入官方 QQ 群（1127453722）提交您的账号邮箱登记人工审核转账。
              </span>
              <span v-else>
                提现指引：当前可用返利 {{ formatCurrency(detail.aff_quota) }} / {{ formatCurrency(withdrawThreshold) }}。未满 {{ withdrawThreshold }} 元也可随时点击左侧「转入余额」无门槛 1:1 转为站内余额直接调用模型。
              </span>
            </div>
          </div>
        </div>

        <!-- 邀请好友明细表（含待到账/已到账状态与唤醒提醒） -->
        <div class="card p-6" data-testid="affiliate-invitees">
          <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.invitees.title') }}（待到账 / 已到账明细）</h3>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                按实付金额统计两步奖达标节点；消耗满 3 元不区分体验金与实付余额。
              </p>
            </div>
          </div>

          <div v-if="inviteesLoading" class="flex justify-center py-8" role="status">
            <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
          </div>
          <div v-else-if="detail.invitees.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[760px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-3 py-2 font-medium">实付 / 消耗进度</th>
                  <th class="px-3 py-2 font-medium">单人两步奖状态</th>
                  <th class="px-3 py-2 font-medium text-right">{{ t('affiliate.invitees.columns.rebate') }}（{{ formattedRebateRate }}%）</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in detail.invitees"
                  :key="item.user_id"
                  class="border-b border-gray-100 last:border-b-0 dark:border-dark-800"
                >
                  <td class="px-3 py-3 text-gray-900 dark:text-white">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3 text-xs text-gray-600 dark:text-gray-300">
                    <div>实付：{{ formatCurrency(item.total_paid_amount ?? 0) }}</div>
                    <div class="mt-0.5 text-gray-500 dark:text-dark-400">
                      消耗：{{ formatCurrency(item.total_consumed_amount ?? 0) }} / $3.00
                    </div>
                  </td>
                  <td class="px-3 py-3">
                    <div class="flex flex-col items-start gap-1">
                      <span class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium" :class="inviteeBadgeClass(item)">
                        {{ inviteeStatusText(item) }}
                      </span>
                      <span
                        v-if="item.needs_consume_reminder"
                        class="text-xs font-medium text-amber-600 dark:text-amber-400"
                      >
                        提醒好友配置 API Key 试用即可解锁 3 元
                      </span>
                    </div>
                  </td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <Pagination
            v-if="inviteesTotal > 0"
            data-testid="affiliate-invitees-pagination"
            :total="inviteesTotal"
            :page="inviteesPage"
            :page-size="inviteesPageSize"
            @update:page="changeInviteesPage"
            @update:page-size="changeInviteesPageSize"
          />
        </div>

        <!-- 清晰版活动规则与边界口径说明 -->
        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">活动规则与防刷风控声明</h3>
          <div class="mt-3 grid gap-4 text-sm text-gray-600 dark:text-dark-300 md:grid-cols-2">
            <div class="space-y-2">
              <p class="font-medium text-gray-900 dark:text-white">一、奖励明细与达标口径</p>
              <ul class="list-disc space-y-1.5 pl-5 text-xs leading-5">
                <li><strong>新人 1 元体验金</strong>：好友通过邀请链接注册，完成人机验证及主流邮箱（QQ、163、126、Foxmail、Gmail、Outlook/Hotmail、iCloud）校验后立即到账 1 元体验余额，自注册起 7 天有效，到期未使用部分自动清零，不可提现。</li>
                <li><strong>单人两步奖（最高 8 元余额）</strong>：好友注册 30 天内累计实付满 10 元即显示「待到账 3 元」，累计消耗满 3 元后转为「已到账 3 元」；累计实付满 50 元且消耗满 3 元再加 5 元（合计 8 元站内余额，不可提现）。</li>
                <li><strong>优惠码实付口径</strong>：使用 <span class="font-mono">STONE-*</span> 等优惠码打折充值时，满 10 元/50 元达标判断及 5% 返利均按实际支付金额计算，不按到账额度计算。</li>
              </ul>
            </div>
            <div class="space-y-2">
              <p class="font-medium text-gray-900 dark:text-white">二、核销提现与违规追责条款</p>
              <ul class="list-disc space-y-1.5 pl-5 text-xs leading-5">
                <li><strong>5% 长期返利</strong>：好友后续每次实付充值，邀请人获得实付金额的 {{ formattedRebateRate }}%，每笔冻结 72 小时后解冻；支持随时无门槛 1:1 转为站内余额，或累计可用返利满 300 元联系客服人工核验转账。</li>
                <li><strong>退款与拒付处理</strong>：已触发邀请奖励或体验金转化的充值订单不支持无理由退款；如发生异常退款或支付争议，系统同步扣回双方的赠送与邀请奖励（余额不足扣为负值并限制账号调用）。</li>
                <li><strong>违规追责条款</strong>：一经判定存在批量注册、自邀套利等作弊行为，官方有权追回全部奖励并封禁相关账号。</li>
              </ul>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { AffiliateInvitee, AffiliateLadderTier, UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)
const inviteesPage = ref(1)
const inviteesPageSize = ref(20)
const inviteesLoading = ref(false)
const inviteesTotal = computed(() => detail.value?.invitees_total ?? detail.value?.invitees.length ?? 0)
let detailRequestId = 0

const DEFAULT_LADDER_TIERS: AffiliateLadderTier[] = [
  { tier_target: 5, cumulative_bonus: 10, delta_bonus: 10, requires_min_total_paid: 0, auto_grant: true, status: 'locked' },
  { tier_target: 20, cumulative_bonus: 40, delta_bonus: 30, requires_min_total_paid: 0, auto_grant: true, status: 'locked' },
  { tier_target: 50, cumulative_bonus: 100, delta_bonus: 60, requires_min_total_paid: 0, auto_grant: false, status: 'locked' },
  { tier_target: 100, cumulative_bonus: 200, delta_bonus: 100, requires_min_total_paid: 3000, auto_grant: false, status: 'locked' },
]

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

// Rebate rate is a percentage in the range [0, 100]; backend already clamps it.
// We trim trailing zeros (e.g. 20.00 → "20", 12.50 → "12.5") for a cleaner UI.
const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

const withdrawThreshold = computed(() => detail.value?.withdraw_threshold ?? 300)
const qualifiedInviteeCount = computed(() => detail.value?.qualified_invitee_count ?? 0)
const qualifiedInviteesPaidTotal = computed(() => detail.value?.qualified_invitees_paid_total ?? 0)
const pendingStepBonus = computed(() => detail.value?.pending_step_bonus_total ?? 0)
const grantedStepBonus = computed(() => detail.value?.granted_step_bonus_total ?? 0)
const pendingLadderBonus = computed(() => detail.value?.pending_ladder_bonus_total ?? 0)
const grantedLadderBonus = computed(() => detail.value?.granted_ladder_bonus_total ?? 0)
const totalGrantedActivityBonus = computed(() => grantedStepBonus.value + grantedLadderBonus.value)

const ladderTiers = computed<AffiliateLadderTier[]>(() => {
  if (detail.value?.ladder_tiers && detail.value.ladder_tiers.length > 0) {
    return detail.value.ladder_tiers
  }
  return DEFAULT_LADDER_TIERS
})

const ladderProgressPercent = computed(() => {
  const count = qualifiedInviteeCount.value
  return Math.min(100, Math.max(0, Math.round((count / 100) * 100)))
})

const remindConsumeCount = computed(() =>
  detail.value?.remind_consume_count ?? (detail.value?.invitees ?? []).filter((item) => item.needs_consume_reminder).length,
)

function formatCount(value: number): string {
  return value.toLocaleString()
}

function tierStatusLabel(tier: AffiliateLadderTier): string {
  switch (tier.status) {
    case 'granted':
      return '已到账'
    case 'pending_review':
      return '人工审核中'
    case 'rejected':
      return '未通过'
    default:
      return `差 ${Math.max(0, tier.tier_target - qualifiedInviteeCount.value)} 人`
  }
}

function tierCardClass(tier: AffiliateLadderTier): string {
  switch (tier.status) {
    case 'granted':
      return 'border-emerald-300 bg-emerald-50/50 dark:border-emerald-700/60 dark:bg-emerald-950/20'
    case 'pending_review':
      return 'border-amber-300 bg-amber-50/50 dark:border-amber-700/60 dark:bg-amber-950/20'
    default:
      return 'border-gray-200 bg-gray-50/50 dark:border-dark-700 dark:bg-dark-900/40'
  }
}

function tierBadgeClass(tier: AffiliateLadderTier): string {
  switch (tier.status) {
    case 'granted':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300'
    case 'pending_review':
      return 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
    default:
      return 'bg-gray-200 text-gray-700 dark:bg-dark-700 dark:text-dark-300'
  }
}

function inviteeStatusText(item: AffiliateInvitee): string {
  switch (item.display_status) {
    case 'pending_consume_8':
      return '待到账 8 元（待消耗满 3 元）'
    case 'pending_consume_3':
      return '待到账 3 元（待消耗满 3 元）'
    case 'granted_8':
      return '已到账 8 元'
    case 'granted_3_pending_5':
      return '已到账 3 元（满 50 再得 5 元）'
    case 'pending_review':
      return '达标审核中'
    case 'expired_window':
      return '已过 30 天达标期'
    default:
      return '已注册（满 10 元解锁 3 元）'
  }
}

function inviteeBadgeClass(item: AffiliateInvitee): string {
  switch (item.display_status) {
    case 'granted_8':
    case 'granted_3_pending_5':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300'
    case 'pending_consume_8':
    case 'pending_consume_3':
      return 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
    case 'pending_review':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300'
  }
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  const requestId = ++detailRequestId
  inviteesLoading.value = true
  if (!silent) {
    loading.value = true
  }
  try {
    const result = await userAPI.getAffiliateDetail({ page: inviteesPage.value, page_size: inviteesPageSize.value })
    if (requestId !== detailRequestId) return
    detail.value = result
    inviteesPage.value = result.invitees_page ?? inviteesPage.value
    inviteesPageSize.value = result.invitees_page_size ?? inviteesPageSize.value
  } catch (error) {
    if (requestId !== detailRequestId) return
    inviteesPage.value = detail.value?.invitees_page ?? 1
    inviteesPageSize.value = detail.value?.invitees_page_size ?? 20
    appStore.showError(extractApiErrorMessage(error, t('affiliate.loadFailed')))
  } finally {
    if (requestId === detailRequestId) {
      inviteesLoading.value = false
      loading.value = false
    }
  }
}

function changeInviteesPage(page: number): void {
  inviteesPage.value = page
  void loadAffiliateDetail(true)
}

function changeInviteesPageSize(pageSize: number): void {
  inviteesPageSize.value = pageSize
  inviteesPage.value = 1
  void loadAffiliateDetail(true)
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
