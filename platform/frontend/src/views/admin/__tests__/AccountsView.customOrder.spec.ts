import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount, type VueWrapper } from '@vue/test-utils'
import { defineComponent } from 'vue'
import AccountsView from '../AccountsView.vue'

const { list, listWithEtag, showSuccess, getBatchTodayStats, getUpstreamBillingProbeSettings, getAllProxies, getAllGroups } = vi.hoisted(() => ({
  list: vi.fn(), listWithEtag: vi.fn(), showSuccess: vi.fn(), getBatchTodayStats: vi.fn(),
  getUpstreamBillingProbeSettings: vi.fn(), getAllProxies: vi.fn(), getAllGroups: vi.fn()
}))
vi.mock('@/api/admin', () => ({ adminAPI: {
  accounts: { list, listWithEtag, getBatchTodayStats, getUpstreamBillingProbeSettings },
  proxies: { getAll: getAllProxies },
  groups: { getAll: getAllGroups }
} }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError: vi.fn(), showWarning: vi.fn(), showInfo: vi.fn() }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ token: 'test-token', isSimpleMode: false }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const OrderModal = defineComponent({
  name: 'AccountOrderModal', props: { show: Boolean }, emits: ['saved', 'close'],
  template: '<div v-if="show" data-test="order-modal"><button data-test="order-saved" @click="$emit(\'saved\')">saved</button><button data-test="order-close" @click="$emit(\'close\')">close</button></div>'
})
const DataTable = defineComponent({
  name: 'DataTable', props: ['data', 'columns'], emits: ['sort'],
  template: '<div><button data-test="sort-name" @click="$emit(\'sort\', \'name\', \'desc\')">name</button><div v-for="row in data" :key="row.id"><span data-test="account">{{ row.id }}</span><span data-test="display-order">{{ row.display_order }}</span></div></div>'
})
const Pagination = defineComponent({
  name: 'Pagination', emits: ['update:page'],
  template: '<button data-test="next-page" @click="$emit(\'update:page\', 2)">page 2</button>'
})
const EditAccountModal = defineComponent({
  name: 'EditAccountModal', props: ['show', 'account', 'proxies', 'groups'], emits: ['updated', 'close'],
  template: '<div />'
})
const wrappers: VueWrapper[] = []
function mountView() {
  const wrapper = shallowMount(AccountsView, { global: { stubs: {
    AppLayout: { template: '<div><slot /></div>' },
    TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
    AccountTableActions: { template: '<div><slot name="after" /></div>' },
    AccountOrderModal: OrderModal, EditAccountModal, DataTable, Pagination
  } } })
  wrappers.push(wrapper)
  return wrapper
}

describe('AccountsView custom display order integration', () => {
  beforeEach(() => {
    localStorage.clear()
    list.mockReset().mockResolvedValue({ items: [{ id: 3, name: 'three', platform: 'openai', group_ids: [] }], total: 40, page: 1, page_size: 20, pages: 2 })
    listWithEtag.mockReset().mockResolvedValue({ notModified: true, etag: null, data: null })
    getBatchTodayStats.mockReset().mockResolvedValue({ stats: {} })
    getUpstreamBillingProbeSettings.mockReset().mockResolvedValue({ enabled: true })
    getAllProxies.mockReset().mockResolvedValue([])
    getAllGroups.mockReset().mockResolvedValue([])
    showSuccess.mockReset()
  })
  afterEach(() => {
    wrappers.splice(0).forEach(wrapper => wrapper.unmount())
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('uses saved display order by default while retaining existing column sorting', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(1, 20, expect.objectContaining({ sort_by: 'display_order', sort_order: 'asc' }), expect.any(Object))
    await wrapper.get('[data-test="sort-name"]').trigger('click')
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(1, 20, expect.objectContaining({ sort_by: 'name', sort_order: 'desc' }), expect.any(Object))
  })

  it('preserves an explicit existing sort preference on initial load and cancellation', async () => {
    const preference = JSON.stringify({ key: 'priority', order: 'desc' })
    localStorage.setItem('account-table-sort', preference)
    const wrapper = mountView()
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(1, 20, expect.objectContaining({ sort_by: 'priority', sort_order: 'desc' }), expect.any(Object))
    const loads = list.mock.calls.length
    await wrapper.get('[data-testid="account-order-open"]').trigger('click')
    expect(wrapper.find('[data-test="order-modal"]').exists()).toBe(true)
    await wrapper.get('[data-test="order-close"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-test="order-modal"]').exists()).toBe(false)
    expect(localStorage.getItem('account-table-sort')).toBe(preference)
    expect(list).toHaveBeenCalledTimes(loads)
  })

  it('returns to page one with display order after saving and persists that preference for the next visit', async () => {
    localStorage.setItem('account-table-sort', JSON.stringify({ key: 'name', order: 'desc' }))
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="next-page"]').trigger('click')
    await flushPromises()
    expect(list.mock.calls[list.mock.calls.length - 1][0]).toBe(2)
    await wrapper.get('[data-testid="account-order-open"]').trigger('click')
    await wrapper.get('[data-test="order-saved"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-test="order-modal"]').exists()).toBe(false)
    expect(list).toHaveBeenLastCalledWith(1, 20, expect.objectContaining({ sort_by: 'display_order', sort_order: 'asc' }), expect.any(Object))
    expect(JSON.parse(localStorage.getItem('account-table-sort')!)).toEqual({ key: 'display_order', order: 'asc' })
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.customOrder.saved')
    const secondVisit = mountView()
    await flushPromises()
    expect(secondVisit.get('[data-test="account"]').text()).toBe('3')
    expect(list).toHaveBeenLastCalledWith(1, 20, expect.objectContaining({ sort_by: 'display_order', sort_order: 'asc' }), expect.any(Object))
  })

  it('pauses automatic refresh while the ordering dialog is open and resumes after cancellation', async () => {
    vi.useFakeTimers()
    vi.spyOn(document, 'hidden', 'get').mockReturnValue(false)
    localStorage.setItem('account-auto-refresh', JSON.stringify({ enabled: true, interval_seconds: 5 }))
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="account-order-open"]').trigger('click')
    await vi.advanceTimersByTimeAsync(6000)
    await flushPromises()
    expect(listWithEtag).not.toHaveBeenCalled()
    await wrapper.get('[data-test="order-close"]').trigger('click')
    await vi.advanceTimersByTimeAsync(6000)
    await flushPromises()
    expect(listWithEtag).toHaveBeenCalledWith(1, 20, expect.objectContaining({ sort_by: 'display_order', sort_order: 'asc' }), expect.any(Object))
  })

  it('refreshes display numbers saved in another browser when IDs and updated_at are unchanged', async () => {
    vi.useFakeTimers()
    vi.spyOn(document, 'hidden', 'get').mockReturnValue(false)
    localStorage.setItem('account-auto-refresh', JSON.stringify({ enabled: true, interval_seconds: 5 }))
    const initialRows = [3, 7].map(id => ({
      id, name: `account-${id}`, platform: 'openai', group_ids: [],
      updated_at: '2026-10-27T08:00:00Z', display_order: 0
    }))
    const initialPage = { items: initialRows, total: 2, page: 1, page_size: 20, pages: 1 }
    list.mockResolvedValueOnce(initialPage)
    listWithEtag.mockResolvedValueOnce({
      notModified: false, etag: 'new-display-order',
      data: { ...initialPage, items: initialRows.map((row, index) => ({ ...row, display_order: index + 1 })) }
    })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.findAll('[data-test="display-order"]').map(cell => cell.text())).toEqual(['0', '0'])

    await vi.advanceTimersByTimeAsync(6000)
    await flushPromises()

    expect(listWithEtag).toHaveBeenCalledTimes(1)
    expect(list).toHaveBeenCalledTimes(1)
    expect(wrapper.findAll('[data-test="account"]').map(cell => cell.text())).toEqual(['3', '7'])
    expect(wrapper.findAll('[data-test="display-order"]').map(cell => cell.text())).toEqual(['1', '2'])
  })

  it.each([
    { name: 'an old backend omits display_order', returnedOrder: undefined, expectedOrder: '12' },
    { name: 'a new backend explicitly returns zero', returnedOrder: 0, expectedOrder: '0' }
  ])('merges an ordinary account edit when $name', async ({ returnedOrder, expectedOrder }) => {
    const legacyResponse = { id: 3, name: 'three', platform: 'openai', group_ids: [] }
    const original = { ...legacyResponse, display_order: 12 }
    list.mockResolvedValueOnce({ items: [original], total: 1, page: 1, page_size: 20, pages: 1 })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.get('[data-test="display-order"]').text()).toBe('12')

    const updatedAccount = { ...legacyResponse, name: 'edited', ...(returnedOrder === undefined ? {} : { display_order: returnedOrder }) }
    wrapper.getComponent(EditAccountModal).vm.$emit('updated', updatedAccount)
    await flushPromises()

    expect(wrapper.getComponent(DataTable).props('data')[0].name).toBe('edited')
    expect(wrapper.get('[data-test="display-order"]').text()).toBe(expectedOrder)
    expect(list).toHaveBeenCalledTimes(1)
  })
})
