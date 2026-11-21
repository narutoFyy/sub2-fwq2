import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AdminAffiliateRecordsTable from '../AdminAffiliateRecordsTable.vue'

const { listInviteRecords, listRebateRecords, listTransferRecords, getUserOverview, listRewardGrants, approveRewardGrant, rejectRewardGrant } = vi.hoisted(() => ({
  listInviteRecords: vi.fn(),
  listRebateRecords: vi.fn(),
  listTransferRecords: vi.fn(),
  getUserOverview: vi.fn(),
  listRewardGrants: vi.fn(),
  approveRewardGrant: vi.fn(),
  rejectRewardGrant: vi.fn(),
}))

vi.mock('@/api/admin/affiliates', () => {
  const affiliatesAPI = { listInviteRecords, listRebateRecords, listTransferRecords, getUserOverview, listRewardGrants, approveRewardGrant, rejectRewardGrant }
  return { affiliatesAPI, default: affiliatesAPI }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const TablePageLayoutStub = {
  template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>',
}

const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <table>
      <tr v-for="(row, index) in data" :key="index" :data-test="'row-' + index">
        <td v-for="column in columns" :key="column.key" :data-test="'cell-' + column.key">
          <slot :name="'cell-' + column.key" :row="row" />
        </td>
      </tr>
    </table>
  `,
}

const WithdrawDialogStub = {
  name: 'AffiliateOfflineWithdrawDialog',
  props: ['show'],
  emits: ['close', 'success'],
  template: '<div data-test="withdraw-dialog" :data-show="String(show)" />',
}

const PaginationStub = {
  name: 'Pagination',
  props: ['total', 'page', 'pageSize'],
  emits: ['update:page', 'update:pageSize'],
  template: '<div><button data-test="next" @click="$emit(\'update:page\', page + 1)">next</button></div>',
}

const BaseDialogStub = {
  props: ['show'],
  template: '<section v-if="show"><slot /></section>',
}

function mountTable(type: 'invites' | 'rebates' | 'transfers') {
  return mount(AdminAffiliateRecordsTable, {
    props: { type },
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        Pagination: PaginationStub,
        Icon: true,
        OrderStatusBadge: true,
        BaseDialog: BaseDialogStub,
        AffiliateOfflineWithdrawDialog: WithdrawDialogStub,
      },
    },
  })
}

function page<T>(items: T[]) {
  return { items, total: items.length, page: 1, page_size: 20, pages: 1 }
}

describe('AdminAffiliateRecordsTable', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    localStorage.clear()
    listInviteRecords.mockResolvedValue(page([]))
  })

  it('renders non-order rebate accruals with empty order fields', async () => {
    listRebateRecords.mockResolvedValue(page([
      {
        order_id: null,
        out_trade_no: '',
        inviter_id: 1,
        inviter_email: 'inviter@example.com',
        inviter_username: 'inviter',
        invitee_id: 2,
        invitee_email: 'invitee@example.com',
        invitee_username: 'invitee',
        order_amount: null,
        pay_amount: null,
        rebate_amount: 2,
        payment_type: '',
        order_status: '',
        created_at: '2026-09-17T08:00:00Z',
      },
    ]))

    const wrapper = mountTable('rebates')
    await flushPromises()

    const row = wrapper.get('[data-test="row-0"]')
    expect(row.get('[data-test="cell-order"]').text()).toBe('-')
    expect(row.get('[data-test="cell-order_amount"]').text()).toBe('-')
    expect(row.get('[data-test="cell-pay_amount"]').text()).toBe('-')
    expect(row.get('[data-test="cell-payment_type"]').text()).toBe('-')
    expect(row.get('[data-test="cell-order_status"]').text()).toBe('-')
    expect(row.get('[data-test="cell-rebate_amount"]').text()).toBe('$2.00')
    expect(wrapper.find('[data-test="affiliate-withdraw-open"]').exists()).toBe(false)
  })

  it('lists offline withdrawals next to balance transfers with their type', async () => {
    listTransferRecords.mockResolvedValue(page([
      {
        ledger_id: 11,
        action: 'withdraw',
        user_id: 42,
        user_email: 'inviter@example.com',
        username: 'inviter',
        amount: 12.5,
        balance_after: 5.5,
        available_quota_after: 7.5,
        frozen_quota_after: 0,
        history_quota_after: 30,
        snapshot_available: true,
        created_at: '2026-09-17T08:00:00Z',
      },
      {
        ledger_id: 10,
        action: 'transfer',
        user_id: 42,
        user_email: 'inviter@example.com',
        username: 'inviter',
        amount: 3,
        balance_after: 5.5,
        available_quota_after: 20,
        frozen_quota_after: 0,
        history_quota_after: 30,
        snapshot_available: true,
        created_at: '2026-09-16T08:00:00Z',
      },
    ]))

    const wrapper = mountTable('transfers')
    await flushPromises()

    const withdrawRow = wrapper.get('[data-test="row-0"]')
    expect(withdrawRow.get('[data-test="cell-action"]').text()).toBe('admin.affiliates.outflowTypes.withdraw')
    expect(withdrawRow.get('[data-test="cell-amount"]').text()).toBe('$12.50')

    const transferRow = wrapper.get('[data-test="row-1"]')
    expect(transferRow.get('[data-test="cell-action"]').text()).toBe('admin.affiliates.outflowTypes.transfer')
  })

  it('opens the offline withdrawal dialog and reloads the first page after recording', async () => {
    listTransferRecords.mockResolvedValue(page([]))

    const wrapper = mountTable('transfers')
    await flushPromises()
    expect(listTransferRecords).toHaveBeenCalledTimes(1)

    const button = wrapper.get('[data-test="affiliate-withdraw-open"]')
    expect(button.text()).toBe('admin.affiliates.withdraw.button')
    await button.trigger('click')
    expect(wrapper.get('[data-test="withdraw-dialog"]').attributes('data-show')).toBe('true')

    wrapper.findComponent(WithdrawDialogStub).vm.$emit('success', { ledger_id: 12 })
    await flushPromises()

    expect(wrapper.get('[data-test="withdraw-dialog"]').attributes('data-show')).toBe('false')
    expect(listTransferRecords).toHaveBeenCalledTimes(2)
    expect(listTransferRecords.mock.calls[1][0]).toMatchObject({ page: 1 })
  })

  function grant(id: number) {
    return { id, inviter_id: id, inviter_email: `inviter-${id}@example.com`, status: 'pending_review', reward_type: 'tier_50', amount: 60 }
  }

  async function openReviews() {
    const wrapper = mountTable('invites')
    await flushPromises()
    await wrapper.get('[data-test="affiliate-reward-review-open"]').trigger('click')
    await flushPromises()
    return wrapper
  }

  it('loads rewards after the first 50 and resets the page when filters or page size change', async () => {
    listRewardGrants.mockImplementation(({ page: requestedPage, page_size }) => Promise.resolve({
      items: [grant(requestedPage === 1 ? 51 : 1)], total: 51, page: requestedPage, page_size,
    }))
    const wrapper = await openReviews()
    const pager = () => wrapper.findAllComponents(PaginationStub).find(component => component.attributes('data-testid') === 'reward-review-pagination')!
    expect(pager().props()).toMatchObject({ total: 51, page: 1, pageSize: 50 })
    await pager().get('[data-test="next"]').trigger('click')
    await flushPromises()
    expect(listRewardGrants).toHaveBeenLastCalledWith({ page: 2, page_size: 50, status: 'pending_review' })
    expect(wrapper.get('[data-testid="reward-review-dialog"]').text()).toContain('inviter-1@example.com')

    await wrapper.findAll('button').find(button => button.text() === '全部记录')!.trigger('click')
    await flushPromises()
    expect(listRewardGrants).toHaveBeenLastCalledWith({ page: 1, page_size: 50, status: undefined })
    pager().vm.$emit('update:pageSize', 20)
    await flushPromises()
    expect(listRewardGrants).toHaveBeenLastCalledWith({ page: 1, page_size: 20, status: undefined })
  })

  it.each(['通过并发放', '拒绝'])('returns to the last available page after %s removes the only final-page record', async (action) => {
    listRewardGrants
      .mockResolvedValueOnce({ items: [grant(51)], total: 51 })
      .mockResolvedValueOnce({ items: [grant(1)], total: 51 })
      .mockResolvedValueOnce({ items: [], total: 50 })
      .mockResolvedValueOnce({ items: [grant(51)], total: 50 })
    const wrapper = await openReviews()
    await wrapper.get('[data-testid="reward-review-pagination"] [data-test="next"]').trigger('click')
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === action)!.trigger('click')
    await flushPromises()
    expect(action === '拒绝' ? rejectRewardGrant : approveRewardGrant).toHaveBeenCalledWith(1, expect.any(String))
    expect(listRewardGrants).toHaveBeenLastCalledWith({ page: 1, page_size: 50, status: 'pending_review' })
    expect(wrapper.get('[data-testid="reward-review-dialog"]').text()).toContain('inviter-51@example.com')
  })

  it('ignores a stale page response after changing the filter', async () => {
    let resolvePending!: (result: unknown) => void
    listRewardGrants.mockImplementationOnce(() => new Promise(resolve => { resolvePending = resolve }))
    const wrapper = await openReviews()
    listRewardGrants.mockResolvedValueOnce({ items: [grant(2)], total: 1 })
    await wrapper.findAll('button').find(button => button.text() === '全部记录')!.trigger('click')
    await flushPromises()
    resolvePending({ items: [grant(1)], total: 51 })
    await flushPromises()
    const dialog = wrapper.get('[data-testid="reward-review-dialog"]')
    expect(dialog.text()).toContain('inviter-2@example.com')
    expect(dialog.text()).not.toContain('inviter-1@example.com')
  })

  it('restores the last successful page and filter when a reward request fails', async () => {
    listRewardGrants
      .mockResolvedValueOnce({ items: [grant(51)], total: 51, page: 1, page_size: 50 })
      .mockRejectedValueOnce(new Error('temporary failure'))
      .mockRejectedValueOnce(new Error('temporary failure'))
      .mockRejectedValueOnce(new Error('temporary failure'))
    const wrapper = await openReviews()
    const pager = () => wrapper.findAllComponents(PaginationStub).find(component => component.attributes('data-testid') === 'reward-review-pagination')!

    await pager().get('[data-test="next"]').trigger('click')
    await flushPromises()
    expect(listRewardGrants).toHaveBeenLastCalledWith({ page: 2, page_size: 50, status: 'pending_review' })
    expect(pager().props()).toMatchObject({ page: 1, total: 51 })
    expect(wrapper.get('[data-testid="reward-review-dialog"]').text()).toContain('inviter-51@example.com')

    await wrapper.findAll('button').find(button => button.text() === '全部记录')!.trigger('click')
    await flushPromises()
    expect(listRewardGrants).toHaveBeenLastCalledWith({ page: 1, page_size: 50, status: undefined })
    expect(wrapper.findAll('button').find(button => button.text() === '待人工审核')!.classes()).toContain('btn-primary')
    expect(wrapper.findAll('button').find(button => button.text() === '全部记录')!.classes()).toContain('btn-secondary')
    expect(pager().props()).toMatchObject({ page: 1, total: 51 })

    pager().vm.$emit('update:pageSize', 20)
    await flushPromises()
    expect(listRewardGrants).toHaveBeenLastCalledWith({ page: 1, page_size: 20, status: 'pending_review' })
    expect(pager().props()).toMatchObject({ page: 1, pageSize: 50, total: 51 })
  })
})
