import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import UsageCleanupDialog from '../UsageCleanupDialog.vue'
import Pagination from '@/components/common/Pagination.vue'

const { listCleanupTasks, createCleanupTask, showError, showSuccess } = vi.hoisted(() => ({ listCleanupTasks: vi.fn(), createCleanupTask: vi.fn(), showError: vi.fn(), showSuccess: vi.fn() }))
vi.mock('@/components/admin/usage/UsageFilters.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/api/admin/usage', () => ({ adminUsageAPI: { listCleanupTasks, createCleanupTask } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError, showSuccess }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)
beforeEach(() => { vi.clearAllMocks(); vi.spyOn(console, 'error').mockImplementation(() => {}) })
afterEach(() => { vi.restoreAllMocks() })
const page = (number: number) => ({ items: [{ id: number, status: 'succeeded', deleted_rows: number, filters: {} }], total: 15, page: number, page_size: 5 })
async function openDialog() {
  listCleanupTasks.mockResolvedValueOnce(page(1))
  const wrapper = mount(UsageCleanupDialog, {
    props: { show: false, filters: {}, startDate: '2026-09-01', endDate: '2026-09-02' },
    global: { stubs: { BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' }, UsageFilters: true, ConfirmDialog: true, Pagination: true } },
  })
  await wrapper.setProps({ show: true })
  await flushPromises()
  return wrapper
}

describe('cleanup task pagination', () => {
  it.each(['success', 'failure'])('ignores an older page request that completes with %s', async (outcome) => {
    const wrapper = await openDialog()
    let resolve!: (value: object) => void
    let reject!: (error: Error) => void
    listCleanupTasks.mockImplementationOnce(() => new Promise((res, rej) => { resolve = res; reject = rej }))
    wrapper.findComponent(Pagination).vm.$emit('update:page', 2)
    await flushPromises()
    listCleanupTasks.mockResolvedValueOnce(page(3))
    wrapper.findComponent(Pagination).vm.$emit('update:page', 3)
    await flushPromises()
    if (outcome === 'success') resolve(page(2))
    else reject(new Error('old failure'))
    await flushPromises()
    expect(wrapper.findComponent(Pagination).props('page')).toBe(3)
    expect(wrapper.text()).toContain('#3')
    expect(showError).not.toHaveBeenCalled()
  })

  it('ignores a pending result after the dialog is closed and reopened', async () => {
    const wrapper = await openDialog()
    let resolve!: (value: object) => void
    listCleanupTasks.mockImplementationOnce(() => new Promise(res => { resolve = res }))
    wrapper.findComponent(Pagination).vm.$emit('update:page', 2)
    await flushPromises()
    await wrapper.setProps({ show: false })
    listCleanupTasks.mockResolvedValueOnce(page(1))
    await wrapper.setProps({ show: true })
    await flushPromises()
    resolve(page(2))
    await flushPromises()
    expect(wrapper.findComponent(Pagination).props('page')).toBe(1)
    expect(wrapper.text()).toContain('#1')
  })
})

describe('cleanup exclude-self warning', () => {
  it('shows the cleanup warning only while the view hides self', async () => {
    listCleanupTasks.mockResolvedValue(page(1))
    createCleanupTask.mockResolvedValue({ id: 9 })
    const wrapper = mount(UsageCleanupDialog, {
      props: { show: false, filters: { exclude_self: true }, startDate: '2026-09-01', endDate: '2026-09-02', excludeSelfActive: true },
      global: { stubs: { BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' }, UsageFilters: true, ConfirmDialog: true, Pagination: true } },
    })
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(wrapper.find('[data-testid="cleanup-exclude-self-warning"]').exists()).toBe(true)
    await wrapper.setProps({ excludeSelfActive: false })
    expect(wrapper.find('[data-testid="cleanup-exclude-self-warning"]').exists()).toBe(false)
  })
})

describe('cleanup submit payload', () => {
  it('submits the full-database cleanup without exclude_self or exclude_user_id even when the view hides self', async () => {
    listCleanupTasks.mockResolvedValue(page(1))
    createCleanupTask.mockResolvedValue({ id: 9 })
    const wrapper = mount(UsageCleanupDialog, {
      props: { show: false, filters: { exclude_self: true, model: 'm' }, startDate: '2026-09-01', endDate: '2026-09-02', excludeSelfActive: true },
      global: { stubs: { BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' }, UsageFilters: true, ConfirmDialog: true, Pagination: true } },
    })
    await wrapper.setProps({ show: true })
    await flushPromises()
    await (wrapper.vm as unknown as { submitCleanup: () => Promise<void> }).submitCleanup()
    await flushPromises()
    expect(createCleanupTask).toHaveBeenCalledTimes(1)
    const payload = createCleanupTask.mock.calls[0][0] as Record<string, unknown>
    expect(payload).not.toHaveProperty('exclude_self')
    expect(payload).not.toHaveProperty('exclude_user_id')
    expect(payload).toMatchObject({ start_date: '2026-09-01', end_date: '2026-09-02', model: 'm' })
  })
})
