import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import AccountOrderModal from '../AccountOrderModal.vue'

const { list, reorder } = vi.hoisted(() => ({ list: vi.fn(), reorder: vi.fn() }))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { list, reorder } } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('vue-draggable-plus', () => ({
  VueDraggable: { name: 'VueDraggable', props: ['modelValue', 'disabled'], emits: ['update:modelValue', 'end'], template: '<div><slot /></div>' }
}))

const row = (id: number) => ({ id, name: `account-${id}`, platform: 'openai', priority: 20 - id })
const page = (ids: number[], total = ids.length) => ({ items: ids.map(row), total, page: 1, page_size: 1000 })
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}
const wrappers: VueWrapper[] = []
function mountModal() {
  const wrapper = mount(AccountOrderModal, {
    props: { show: true },
    global: { stubs: {
      BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
      Icon: true
    } }
  })
  wrappers.push(wrapper)
  return wrapper
}
const ids = (wrapper: VueWrapper) => wrapper.findAll('[data-testid="account-order-row"]').map(item => Number(item.attributes('data-account-id')))
const save = (wrapper: VueWrapper) => wrapper.get<HTMLButtonElement>('[data-testid="account-order-save"]')

describe('AccountOrderModal', () => {
  beforeEach(() => {
    list.mockReset().mockResolvedValue(page([3, 1, 2]))
    reorder.mockReset().mockResolvedValue({ success: true })
  })
  afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()) })

  it('loads every server page in saved order without list filters or scheduling payloads', async () => {
    list.mockResolvedValueOnce(page([3, 1], 3)).mockResolvedValueOnce(page([2], 3))
    const wrapper = mountModal()
    expect(save(wrapper).element.disabled).toBe(true)
    await flushPromises()
    expect(ids(wrapper)).toEqual([3, 1, 2])
    const filters = { lite: '1', include_scheduler_score: '0', sort_by: 'display_order', sort_order: 'asc' }
    expect(list).toHaveBeenNthCalledWith(1, 1, 1000, filters, { signal: expect.any(AbortSignal) })
    expect(list).toHaveBeenNthCalledWith(2, 2, 1000, filters, { signal: expect.any(AbortSignal) })
    await wrapper.findAll('[data-testid="account-order-down"]')[0].trigger('click')
    await wrapper.findAll('[data-testid="account-order-up"]')[2].trigger('click')
    expect(ids(wrapper)).toEqual([1, 2, 3])
    expect(wrapper.findAll<HTMLButtonElement>('[data-testid="account-order-up"]')[0].element.disabled).toBe(true)
    expect(wrapper.findAll<HTMLButtonElement>('[data-testid="account-order-down"]')[2].element.disabled).toBe(true)
    await save(wrapper).trigger('click')
    await flushPromises()
    expect(reorder).toHaveBeenCalledTimes(1)
    expect(reorder).toHaveBeenCalledWith([1, 2, 3])
    expect(wrapper.emitted('saved')).toHaveLength(1)
  })

  it('saves the order received from a drag interaction', async () => {
    const wrapper = mountModal()
    await flushPromises()
    const drag = wrapper.getComponent({ name: 'VueDraggable' })
    // Exercise the draggable component contract while leaving pointer mechanics to browser coverage.
    drag.vm.$emit('update:modelValue', [row(2), row(3), row(1)])
    drag.vm.$emit('end')
    await flushPromises()
    expect(ids(wrapper)).toEqual([2, 3, 1])
    await save(wrapper).trigger('click')
    await flushPromises()
    expect(reorder).toHaveBeenCalledTimes(1)
    expect(reorder).toHaveBeenCalledWith([2, 3, 1])
  })

  it('cancels without a write and discards unsaved edits when reopened', async () => {
    const wrapper = mountModal()
    await flushPromises()
    await wrapper.findAll('[data-testid="account-order-down"]')[0].trigger('click')
    await wrapper.get('[data-testid="account-order-cancel"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
    expect(reorder).not.toHaveBeenCalled()
    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(ids(wrapper)).toEqual([3, 1, 2])
    expect(list).toHaveBeenCalledTimes(2)
  })

  it('keeps local edits after save failure and prevents duplicate submissions while pending', async () => {
    const pending = deferred<{ success: boolean }>()
    reorder.mockReturnValueOnce(pending.promise)
    const wrapper = mountModal()
    await flushPromises()
    await wrapper.findAll('[data-testid="account-order-down"]')[0].trigger('click')
    await save(wrapper).trigger('click')
    await save(wrapper).trigger('click')
    expect(reorder).toHaveBeenCalledTimes(1)
    expect(save(wrapper).element.disabled).toBe(true)
    expect(wrapper.get<HTMLButtonElement>('[data-testid="account-order-cancel"]').element.disabled).toBe(true)
    expect(wrapper.findAll<HTMLButtonElement>('[data-testid="account-order-up"]').every(button => button.element.disabled)).toBe(true)
    pending.reject(new Error('write failed'))
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('write failed')
    expect(ids(wrapper)).toEqual([1, 3, 2])
    expect(wrapper.emitted('saved')).toBeUndefined()
    expect(save(wrapper).element.disabled).toBe(false)
    await save(wrapper).trigger('click')
    await flushPromises()
    expect(reorder).toHaveBeenLastCalledWith([1, 3, 2])
    expect(wrapper.emitted('saved')).toHaveLength(1)
  })

  it('disables saving on load error and supports retrying the complete list', async () => {
    list.mockRejectedValueOnce(new Error('list failed'))
    const wrapper = mountModal()
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('list failed')
    expect(save(wrapper).element.disabled).toBe(true)
    expect(ids(wrapper)).toEqual([])
    await wrapper.get('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(ids(wrapper)).toEqual([3, 1, 2])
    expect(save(wrapper).element.disabled).toBe(false)
  })

  it.each([
    { name: 'duplicate IDs across pages', first: page([3, 1], 3), second: page([1], 3) },
    { name: 'changed total across pages', first: page([3, 1], 3), second: page([2], 4) },
    { name: 'truncated pagination', first: page([3, 1], 3), second: page([], 3) },
    { name: 'more than 10000 accounts', first: page([3], 10001), second: page([], 10001) }
  ])('rejects $name rather than saving an incomplete order', async ({ first, second }) => {
    list.mockResolvedValueOnce(first).mockResolvedValueOnce(second)
    const wrapper = mountModal()
    await flushPromises()
    expect(wrapper.get('[role="alert"]').exists()).toBe(true)
    expect(save(wrapper).element.disabled).toBe(true)
    expect(ids(wrapper)).toEqual([])
    expect(reorder).not.toHaveBeenCalled()
  })

  it('keeps an empty account list unsavable', async () => {
    list.mockResolvedValueOnce(page([]))
    const wrapper = mountModal()
    await flushPromises()
    expect(save(wrapper).element.disabled).toBe(true)
    expect(wrapper.text()).toContain('empty.noData')
  })

  it('ignores an old request after closing and reopening even if the transport ignores abort', async () => {
    const old = deferred<ReturnType<typeof page>>()
    list.mockReturnValueOnce(old.promise).mockResolvedValueOnce(page([8, 7]))
    const wrapper = mountModal()
    const signal = list.mock.calls[0][3].signal as AbortSignal
    await wrapper.setProps({ show: false })
    expect(signal.aborted).toBe(true)
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(ids(wrapper)).toEqual([8, 7])
    old.resolve(page([3, 1, 2]))
    await flushPromises()
    expect(ids(wrapper)).toEqual([8, 7])
    await save(wrapper).trigger('click')
    await flushPromises()
    expect(reorder).toHaveBeenCalledTimes(1)
    expect(reorder).toHaveBeenCalledWith([8, 7])
  })
})
