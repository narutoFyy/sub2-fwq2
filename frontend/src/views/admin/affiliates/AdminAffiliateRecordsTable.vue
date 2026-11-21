<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-full md:w-80">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model="filters.search" type="text" class="input pl-10" :placeholder="t('admin.affiliates.records.searchPlaceholder')" @input="debounceLoad" />
          </div>
          <input v-model="filters.start_at" type="date" class="input w-full sm:w-44" :title="t('admin.affiliates.records.startAt')" @change="reloadFromFirstPage" />
          <input v-model="filters.end_at" type="date" class="input w-full sm:w-44" :title="t('admin.affiliates.records.endAt')" @change="reloadFromFirstPage" />
          <button class="btn btn-secondary px-2 md:px-3" :disabled="loading" :title="t('common.refresh')" @click="loadRecords">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button
            v-if="props.type === 'invites'"
            type="button"
            class="btn btn-primary ml-auto"
            data-test="affiliate-reward-review-open"
            @click="openRewardReviewDialog"
          >
            奖励人工审核列表
          </button>
          <button
            v-if="props.type === 'transfers'"
            type="button"
            class="btn btn-primary ml-auto"
            data-test="affiliate-withdraw-open"
            @click="withdrawDialog = true"
          >
            {{ t('admin.affiliates.withdraw.button') }}
          </button>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="records"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          :sort-storage-key="sortStorageKey"
          @sort="handleSort"
        >
          <template #cell-inviter="{ row }">
            <UserCell
              :id="row.inviter_id"
              :email="row.inviter_email"
              :username="row.inviter_username"
              :clickable="props.type !== 'transfers'"
              @open="openUserOverview"
            />
          </template>
          <template #cell-invitee="{ row }">
            <span v-if="row.invitee_id == null" class="text-sm text-gray-400 dark:text-dark-500">-</span>
            <UserCell
              v-else
              :id="row.invitee_id"
              :email="row.invitee_email"
              :username="row.invitee_username"
              :clickable="props.type !== 'transfers'"
              @open="openUserOverview"
            />
          </template>
          <template #cell-user="{ row }">
            <UserCell
              :id="row.user_id"
              :email="row.user_email"
              :username="row.username"
              :clickable="true"
              @open="openUserOverview"
            />
          </template>
          <template #cell-aff_code="{ row }">
            <span class="font-mono text-sm text-gray-700 dark:text-gray-300">{{ row.aff_code || '-' }}</span>
          </template>
          <template #cell-paid_progress="{ row }">
            <div class="space-y-0.5 text-xs">
              <div class="text-gray-900 dark:text-white">实付: ${{ formatAmount(row.total_paid_amount) }}</div>
              <div class="text-gray-500 dark:text-dark-400">消耗: ${{ formatAmount(row.total_consumed_amount) }}</div>
            </div>
          </template>
          <template #cell-reward_status="{ row }">
            <div class="flex flex-wrap items-center gap-1 text-xs">
              <span
                class="rounded px-1.5 py-0.5 font-medium"
                :class="row.trial_bonus_status === 'active' || row.trial_bonus_status === 'used'
                  ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                  : 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-400'"
              >
                体验金: {{ row.trial_bonus_status || 'none' }}
              </span>
              <span
                class="rounded px-1.5 py-0.5 font-medium"
                :class="row.step1_reward_status === 'granted'
                  ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                  : row.step1_reward_status === 'pending_review' || row.step1_reward_status === 'pending_consume'
                    ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
                    : 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-400'"
              >
                3元: {{ row.step1_reward_status || 'none' }}
              </span>
              <span
                class="rounded px-1.5 py-0.5 font-medium"
                :class="row.step2_reward_status === 'granted'
                  ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                  : row.step2_reward_status === 'pending_review' || row.step2_reward_status === 'pending_consume'
                    ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
                    : 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-400'"
              >
                5元: {{ row.step2_reward_status || 'none' }}
              </span>
            </div>
          </template>
          <template #cell-risk_info="{ row }">
            <div class="space-y-1 text-xs">
              <div v-if="row.register_ip" class="font-mono text-gray-600 dark:text-gray-300">{{ row.register_ip }}</div>
              <div class="flex flex-wrap gap-1">
                <span
                  v-if="row.same_ip_warning"
                  class="rounded bg-amber-100 px-1.5 py-0.5 font-medium text-amber-800 dark:bg-amber-900/40 dark:text-amber-300"
                >
                  同 IP 预警
                </span>
                <span
                  v-if="row.risk_tags"
                  class="rounded bg-red-50 px-1.5 py-0.5 text-red-700 dark:bg-red-900/30 dark:text-red-300"
                >
                  {{ row.risk_tags }}
                </span>
                <span v-if="!row.register_ip && !row.same_ip_warning && !row.risk_tags" class="text-gray-400">-</span>
              </div>
            </div>
          </template>
          <template #cell-order="{ row }">
            <span v-if="row.order_id == null" class="text-sm text-gray-400 dark:text-dark-500">-</span>
            <div v-else class="space-y-0.5">
              <div class="font-mono text-sm text-gray-900 dark:text-white">#{{ row.order_id }}</div>
              <div class="max-w-56 truncate text-sm text-gray-500 dark:text-dark-400">{{ row.out_trade_no }}</div>
            </div>
          </template>
          <template #cell-payment_type="{ row }">
            <template v-if="row.payment_type">{{ t('payment.methods.' + row.payment_type, row.payment_type) }}</template>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>
          <template #cell-order_status="{ row }">
            <OrderStatusBadge v-if="row.order_status" :status="row.order_status" />
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>
          <template #cell-total_rebate="{ row }">
            <AmountText :value="row.total_rebate" />
          </template>
          <template #cell-order_amount="{ row }">
            <NullableAmountText :value="row.order_amount" />
          </template>
          <template #cell-pay_amount="{ row }">
            <span v-if="row.pay_amount == null" class="text-sm text-gray-400 dark:text-dark-500">-</span>
            <span v-else class="text-sm text-gray-900 dark:text-white">¥{{ formatAmount(row.pay_amount) }}</span>
          </template>
          <template #cell-rebate_amount="{ row }">
            <AmountText :value="row.rebate_amount" strong />
          </template>
          <template #cell-action="{ row }">
            <span :class="['badge whitespace-nowrap', row.action === 'withdraw' ? 'badge-warning' : 'badge-primary']">
              {{ outflowTypeLabel(row.action) }}
            </span>
          </template>
          <template #cell-amount="{ row }">
            <AmountText :value="row.amount" strong />
          </template>
          <template #cell-balance_after="{ row }">
            <NullableAmountText :value="row.balance_after" />
          </template>
          <template #cell-available_quota_after="{ row }">
            <NullableAmountText :value="row.available_quota_after" />
          </template>
          <template #cell-frozen_quota_after="{ row }">
            <NullableAmountText :value="row.frozen_quota_after" />
          </template>
          <template #cell-history_quota_after="{ row }">
            <NullableAmountText :value="row.history_quota_after" />
          </template>
          <template #cell-created_at="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ formatDateTime(row.created_at) }}</span>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <AffiliateOfflineWithdrawDialog
      v-if="props.type === 'transfers'"
      :show="withdrawDialog"
      @close="withdrawDialog = false"
      @success="handleWithdrawSuccess"
    />

    <!-- 阶梯奖与风控奖励人工审核列表弹窗 -->
    <BaseDialog
      :show="rewardReviewDialog"
      title="阶梯奖（50/100人档）与风控奖励人工审核"
      width="wide"
      @close="rewardReviewDialog = false"
    >
      <div class="space-y-4" data-testid="reward-review-dialog">
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2 text-sm">
            <button
              type="button"
              class="btn btn-sm"
              :class="rewardReviewFilterStatus === 'pending_review' ? 'btn-primary' : 'btn-secondary'"
              @click="changeRewardReviewStatus('pending_review')"
            >
              待人工审核
            </button>
            <button
              type="button"
              class="btn btn-sm"
              :class="rewardReviewFilterStatus === '' ? 'btn-primary' : 'btn-secondary'"
              @click="changeRewardReviewStatus('')"
            >
              全部记录
            </button>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="rewardReviewLoading" @click="loadRewardGrants">
            刷新
          </button>
        </div>

        <div v-if="rewardReviewLoading" class="flex justify-center py-8">
          <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
        </div>
        <div v-else-if="rewardGrants.length === 0" class="rounded-lg border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
          暂无匹配的奖励审核记录
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full min-w-[680px] text-left text-sm">
            <thead>
              <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                <th class="px-3 py-2 font-medium">邀请人</th>
                <th class="px-3 py-2 font-medium">奖励类型</th>
                <th class="px-3 py-2 font-medium">金额</th>
                <th class="px-3 py-2 font-medium">触发原因</th>
                <th class="px-3 py-2 font-medium">状态</th>
                <th class="px-3 py-2 font-medium text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="grant in rewardGrants"
                :key="grant.id"
                class="border-b border-gray-100 last:border-b-0 dark:border-dark-800"
              >
                <td class="px-3 py-2.5">
                  <div class="font-medium text-gray-900 dark:text-white">{{ grant.inviter_email || `#${grant.inviter_id}` }}</div>
                  <div v-if="grant.invitee_email" class="text-xs text-gray-500">关联好友: {{ grant.invitee_email }}</div>
                </td>
                <td class="px-3 py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300">
                  {{ grant.reward_type }}
                </td>
                <td class="px-3 py-2.5 font-semibold text-emerald-600 dark:text-emerald-400">
                  ${{ formatAmount(grant.amount) }}
                </td>
                <td class="px-3 py-2.5 text-xs text-gray-600 dark:text-gray-300">
                  {{ grant.review_reason || '-' }}
                </td>
                <td class="px-3 py-2.5 text-xs">
                  <span
                    class="rounded px-2 py-0.5 font-medium"
                    :class="grant.status === 'granted'
                      ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300'
                      : grant.status === 'pending_review'
                        ? 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
                        : 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300'"
                  >
                    {{ grant.status }}
                  </span>
                </td>
                <td class="px-3 py-2.5 text-right">
                  <div v-if="grant.status === 'pending_review'" class="inline-flex gap-2">
                    <button
                      type="button"
                      class="btn btn-primary btn-sm"
                      :disabled="reviewingGrantId === grant.id"
                      @click="handleApproveGrant(grant.id)"
                    >
                      通过并发放
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary btn-sm text-red-600"
                      :disabled="reviewingGrantId === grant.id"
                      @click="handleRejectGrant(grant.id)"
                    >
                      拒绝
                    </button>
                  </div>
                  <span v-else class="text-xs text-gray-400">-</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <Pagination
          v-if="rewardReviewPagination.total > 0"
          data-testid="reward-review-pagination"
          :total="rewardReviewPagination.total"
          :page="rewardReviewPagination.page"
          :page-size="rewardReviewPagination.page_size"
          @update:page="changeRewardReviewPage"
          @update:page-size="changeRewardReviewPageSize"
        />
      </div>
    </BaseDialog>

    <BaseDialog
      :show="overviewDialog"
      :title="t('admin.affiliates.overview.title')"
      width="normal"
      @close="overviewDialog = false"
    >
      <div v-if="overviewLoading" class="flex justify-center py-8">
        <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>
      <div v-else-if="selectedOverview" class="space-y-4">
        <div class="rounded-lg border border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="font-mono text-sm text-gray-900 dark:text-white">#{{ selectedOverview.user_id }}</div>
          <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ selectedOverview.email || '-' }}</div>
          <div class="mt-0.5 text-sm text-gray-500 dark:text-dark-400">{{ selectedOverview.username || '-' }}</div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <OverviewStat :label="t('admin.affiliates.overview.affCode')" :value="selectedOverview.aff_code || '-'" mono />
          <OverviewStat :label="t('admin.affiliates.overview.rebateRate')" :value="formatPercent(selectedOverview.rebate_rate_percent)" />
          <OverviewStat :label="t('admin.affiliates.overview.invitedCount')" :value="String(selectedOverview.invited_count)" />
          <OverviewStat :label="t('admin.affiliates.overview.rebatedInviteeCount')" :value="String(selectedOverview.rebated_invitee_count)" />
          <OverviewStat label="有效邀请人数" :value="String(selectedOverview.qualified_invitee_count ?? 0)" />
          <OverviewStat label="实付充值人数" :value="String(selectedOverview.paid_invitee_count ?? 0)" />
          <OverviewStat :label="t('admin.affiliates.overview.availableQuota')" :value="'$' + formatAmount(selectedOverview.available_quota)" />
          <OverviewStat :label="t('admin.affiliates.overview.historyQuota')" :value="'$' + formatAmount(selectedOverview.history_quota)" />
        </div>
        <div class="flex items-center justify-between rounded-lg border border-gray-200 p-3 text-sm dark:border-dark-700">
          <div>
            <div class="font-medium text-gray-900 dark:text-white">
              新人 1 元体验金连带风控状态：
              <span :class="selectedOverview.trial_grant_paused ? 'text-red-600' : 'text-emerald-600'">
                {{ selectedOverview.trial_grant_paused ? '已暂停发放（待复核）' : '正常发放' }}
              </span>
            </div>
            <div v-if="selectedOverview.trial_grant_pause_reason" class="mt-0.5 text-xs text-gray-500">
              {{ selectedOverview.trial_grant_pause_reason }}
            </div>
          </div>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            @click="toggleInviterTrialPause"
          >
            {{ selectedOverview.trial_grant_paused ? '解除体验金熔断' : '暂停体验金发放' }}
          </button>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderStatusBadge from '@/components/payment/OrderStatusBadge.vue'
import AffiliateOfflineWithdrawDialog from './AffiliateOfflineWithdrawDialog.vue'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import {
  affiliatesAPI,
  type AffiliateInviteRecord,
  type AffiliateRebateRecord,
  type AffiliateRewardGrantRecord,
  type AffiliateTransferRecord,
  type AffiliateUserOverview,
  type ListAffiliateRecordsParams,
} from '@/api/admin/affiliates'
import type { PaginatedResponse } from '@/types'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatDateTime as formatDisplayDateTime } from '@/utils/format'

type RecordType = 'invites' | 'rebates' | 'transfers'
type AffiliateRecord = AffiliateInviteRecord | AffiliateRebateRecord | AffiliateTransferRecord

const props = defineProps<{
  type: RecordType
}>()

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const records = ref<AffiliateRecord[]>([])
const filters = reactive({ search: '', start_at: '', end_at: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const overviewDialog = ref(false)
const overviewLoading = ref(false)
const selectedOverview = ref<AffiliateUserOverview | null>(null)
const withdrawDialog = ref(false)
const rewardReviewDialog = ref(false)
const rewardReviewLoading = ref(false)
const rewardReviewFilterStatus = ref('pending_review')
const rewardReviewPagination = reactive({ page: 1, page_size: 50, total: 0 })
const rewardGrants = ref<AffiliateRewardGrantRecord[]>([])
const reviewingGrantId = ref<number | null>(null)
let rewardReviewRequestId = 0
let lastRewardReviewState: { page: number; page_size: number; total: number; status: string } | null = null
let debounceTimer: ReturnType<typeof setTimeout> | null = null

const columns = computed<Column[]>(() => {
  if (props.type === 'invites') {
    return [
      { key: 'inviter', label: t('admin.affiliates.records.inviter'), sortable: true },
      { key: 'invitee', label: t('admin.affiliates.records.invitee'), sortable: true },
      { key: 'aff_code', label: t('admin.affiliates.records.affCode'), sortable: true },
      { key: 'paid_progress', label: '实付 / 消耗', sortable: false },
      { key: 'reward_status', label: '体验金 / 两步奖', sortable: false },
      { key: 'risk_info', label: 'IP / 风控标记', sortable: false },
      { key: 'total_rebate', label: t('admin.affiliates.records.totalRebate'), sortable: true },
      { key: 'created_at', label: t('admin.affiliates.records.invitedAt'), sortable: true },
    ]
  }
  if (props.type === 'rebates') {
    return [
      { key: 'order', label: t('admin.affiliates.records.order'), sortable: true },
      { key: 'inviter', label: t('admin.affiliates.records.inviter'), sortable: true },
      { key: 'invitee', label: t('admin.affiliates.records.invitee'), sortable: true },
      { key: 'order_amount', label: t('admin.affiliates.records.orderAmount'), sortable: true },
      { key: 'pay_amount', label: t('admin.affiliates.records.payAmount'), sortable: true },
      { key: 'rebate_amount', label: t('admin.affiliates.records.rebateAmount') },
      { key: 'payment_type', label: t('admin.affiliates.records.paymentType'), sortable: true },
      { key: 'order_status', label: t('admin.affiliates.records.orderStatus'), sortable: true },
      { key: 'created_at', label: t('admin.affiliates.records.rebatedAt'), sortable: true },
    ]
  }
  return [
    { key: 'user', label: t('admin.affiliates.records.user'), sortable: true },
    { key: 'action', label: t('admin.affiliates.records.outflowType'), sortable: true },
    { key: 'amount', label: t('admin.affiliates.records.transferAmount'), sortable: true },
    { key: 'balance_after', label: t('admin.affiliates.records.balanceAfter'), sortable: true },
    { key: 'available_quota_after', label: t('admin.affiliates.records.availableQuotaAfter'), sortable: true },
    { key: 'frozen_quota_after', label: t('admin.affiliates.records.frozenQuotaAfter'), sortable: true },
    { key: 'history_quota_after', label: t('admin.affiliates.records.historyQuotaAfter'), sortable: true },
    { key: 'created_at', label: t('admin.affiliates.records.transferredAt'), sortable: true },
  ]
})

const sortStorageKey = computed(() => `admin-affiliate-${props.type}-table-sort`)

function loadInitialSortState(): { sort_by: string; sort_order: 'asc' | 'desc' } {
  const fallback = { sort_by: 'created_at', sort_order: 'desc' as 'asc' | 'desc' }
  try {
    const raw = localStorage.getItem(sortStorageKey.value)
    if (!raw) return fallback
    const parsed = JSON.parse(raw) as { key?: string; order?: string }
    const key = typeof parsed.key === 'string' ? parsed.key : ''
    if (!columns.value.some((column) => column.key === key && column.sortable)) return fallback
    return {
      sort_by: key,
      sort_order: parsed.order === 'asc' ? 'asc' : 'desc',
    }
  } catch {
    return fallback
  }
}

const sortState = reactive(loadInitialSortState())

function userTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return 'UTC'
  }
}

function buildParams(): ListAffiliateRecordsParams {
  return {
    page: pagination.page,
    page_size: pagination.page_size,
    search: filters.search.trim() || undefined,
    start_at: filters.start_at || undefined,
    end_at: filters.end_at || undefined,
    sort_by: sortState.sort_by,
    sort_order: sortState.sort_order,
    timezone: userTimezone(),
  }
}

async function fetchRecords(params: ListAffiliateRecordsParams): Promise<PaginatedResponse<AffiliateRecord>> {
  if (props.type === 'invites') {
    return affiliatesAPI.listInviteRecords(params)
  }
  if (props.type === 'rebates') {
    return affiliatesAPI.listRebateRecords(params)
  }
  return affiliatesAPI.listTransferRecords(params)
}

async function loadRecords() {
  loading.value = true
  try {
    const res = await fetchRecords(buildParams())
    records.value = res.items || []
    pagination.total = res.total || 0
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function debounceLoad() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => reloadFromFirstPage(), 300)
}

function reloadFromFirstPage() {
  pagination.page = 1
  void loadRecords()
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadRecords()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  void loadRecords()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  void loadRecords()
}

function handleWithdrawSuccess() {
  withdrawDialog.value = false
  reloadFromFirstPage()
}

function outflowTypeLabel(action: string | null | undefined): string {
  return action === 'withdraw'
    ? t('admin.affiliates.outflowTypes.withdraw')
    : t('admin.affiliates.outflowTypes.transfer')
}

function formatAmount(value: number | null | undefined): string {
  return Number(value || 0).toFixed(2)
}

function formatPercent(value: number | null | undefined): string {
  const rounded = Math.round(Number(value || 0) * 100) / 100
  return `${Number.isInteger(rounded) ? rounded.toString() : rounded.toString()}%`
}

function formatDateTime(value: string | null | undefined): string {
  return value ? formatDisplayDateTime(value) : '-'
}

async function openUserOverview(userId: number) {
  if (!userId) return
  overviewDialog.value = true
  overviewLoading.value = true
  selectedOverview.value = null
  try {
    selectedOverview.value = await affiliatesAPI.getUserOverview(userId)
  } catch (error) {
    overviewDialog.value = false
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  } finally {
    overviewLoading.value = false
  }
}

async function toggleInviterTrialPause() {
  if (!selectedOverview.value || !affiliatesAPI.setUserTrialGrantPaused) return
  const nextPaused = !selectedOverview.value.trial_grant_paused
  const reason = nextPaused ? '管理员手动暂停新人体验金发放' : ''
  try {
    await affiliatesAPI.setUserTrialGrantPaused(selectedOverview.value.user_id, nextPaused, reason)
    selectedOverview.value.trial_grant_paused = nextPaused
    selectedOverview.value.trial_grant_pause_reason = reason
    appStore.showSuccess(nextPaused ? '已暂停该邀请人新人体验金发放' : '已解除该邀请人体验金连带熔断')
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  }
}

function openRewardReviewDialog() {
  rewardReviewDialog.value = true
  rewardReviewPagination.page = 1
  rewardReviewPagination.total = 0
  lastRewardReviewState = null
  rewardGrants.value = []
  void loadRewardGrants()
}

function changeRewardReviewStatus(status: string) {
  rewardReviewFilterStatus.value = status
  rewardReviewPagination.page = 1
  void loadRewardGrants()
}

function changeRewardReviewPage(page: number) {
  rewardReviewPagination.page = page
  void loadRewardGrants()
}

function changeRewardReviewPageSize(pageSize: number) {
  rewardReviewPagination.page_size = pageSize
  rewardReviewPagination.page = 1
  void loadRewardGrants()
}

async function loadRewardGrants() {
  if (!affiliatesAPI.listRewardGrants) return
  const requestId = ++rewardReviewRequestId
  rewardReviewLoading.value = true
  try {
    const res = await affiliatesAPI.listRewardGrants({
      page: rewardReviewPagination.page,
      page_size: rewardReviewPagination.page_size,
      status: rewardReviewFilterStatus.value || undefined,
    })
    if (requestId !== rewardReviewRequestId) return
    rewardReviewPagination.total = res.total
    const lastPage = Math.max(1, Math.ceil(res.total / rewardReviewPagination.page_size))
    if (rewardReviewPagination.page > lastPage) {
      rewardReviewPagination.page = lastPage
      await loadRewardGrants()
      return
    }
    rewardGrants.value = res.items || []
    lastRewardReviewState = { ...rewardReviewPagination, status: rewardReviewFilterStatus.value }
  } catch (error) {
    if (requestId !== rewardReviewRequestId) return
    if (lastRewardReviewState) {
      rewardReviewPagination.page = lastRewardReviewState.page
      rewardReviewPagination.page_size = lastRewardReviewState.page_size
      rewardReviewPagination.total = lastRewardReviewState.total
      rewardReviewFilterStatus.value = lastRewardReviewState.status
    } else {
      rewardReviewPagination.page = 1
      rewardReviewPagination.total = 0
      rewardGrants.value = []
    }
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  } finally {
    if (requestId === rewardReviewRequestId) rewardReviewLoading.value = false
  }
}

async function handleApproveGrant(grantId: number) {
  if (!affiliatesAPI.approveRewardGrant) return
  reviewingGrantId.value = grantId
  try {
    await affiliatesAPI.approveRewardGrant(grantId, '管理员审核通过')
    appStore.showSuccess('奖励已审核通过并发放到账')
    await Promise.all([loadRewardGrants(), loadRecords()])
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  } finally {
    reviewingGrantId.value = null
  }
}

async function handleRejectGrant(grantId: number) {
  if (!affiliatesAPI.rejectRewardGrant) return
  reviewingGrantId.value = grantId
  try {
    await affiliatesAPI.rejectRewardGrant(grantId, '管理员审核拒绝')
    appStore.showSuccess('已拒绝该笔奖励发放')
    await Promise.all([loadRewardGrants(), loadRecords()])
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  } finally {
    reviewingGrantId.value = null
  }
}

const UserCell = defineComponent({
  props: {
    id: { type: Number, required: true },
    email: { type: String, default: '' },
    username: { type: String, default: '' },
    clickable: { type: Boolean, default: false },
  },
  emits: ['open'],
  setup(cellProps, { emit }) {
    return () => h('div', { class: 'space-y-0.5' }, [
      h('div', { class: 'font-mono text-sm text-gray-900 dark:text-white' }, `#${cellProps.id}`),
      h(cellProps.clickable ? 'button' : 'div', {
        class: cellProps.clickable
          ? 'max-w-56 truncate text-left text-sm font-medium text-primary-600 hover:text-primary-700 hover:underline dark:text-primary-400 dark:hover:text-primary-300'
          : 'max-w-56 truncate text-sm text-gray-700 dark:text-gray-300',
        type: cellProps.clickable ? 'button' : undefined,
        onClick: cellProps.clickable ? () => emit('open', cellProps.id) : undefined,
      }, cellProps.email || '-'),
      h('div', { class: 'max-w-56 truncate text-sm text-gray-500 dark:text-dark-400' }, cellProps.username || '-'),
    ])
  },
})

const AmountText = defineComponent({
  props: {
    value: { type: Number, default: 0 },
    strong: { type: Boolean, default: false },
  },
  setup(amountProps) {
    return () => h('span', {
      class: amountProps.strong
        ? 'text-sm font-semibold text-emerald-600 dark:text-emerald-400'
        : 'text-sm text-gray-900 dark:text-white',
    }, `$${formatAmount(amountProps.value)}`)
  },
})

const NullableAmountText = defineComponent({
  props: {
    value: { type: Number as PropType<number | null | undefined>, default: null },
  },
  setup(amountProps) {
    return () => {
      const value = amountProps.value
      if (value === null || value === undefined) {
        return h('span', { class: 'text-sm text-gray-400 dark:text-dark-500' }, '-')
      }
      return h(AmountText, { value })
    }
  },
})

const OverviewStat = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
    mono: { type: Boolean, default: false },
  },
  setup(statProps) {
    return () => h('div', { class: 'rounded-lg border border-gray-100 bg-white p-3 dark:border-dark-700 dark:bg-dark-900' }, [
      h('div', { class: 'text-sm text-gray-500 dark:text-dark-400' }, statProps.label),
      h('div', {
        class: statProps.mono
          ? 'mt-1 font-mono text-base font-semibold text-gray-900 dark:text-white'
          : 'mt-1 text-base font-semibold text-gray-900 dark:text-white',
      }, statProps.value),
    ])
  },
})

onMounted(() => {
  void loadRecords()
})
</script>
