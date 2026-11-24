import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
import Drawer from '../DashboardDrawer.vue'
enableAutoUnmount(afterEach)
afterEach(() => { document.body.innerHTML = ''; document.body.style.overflow = '' })
describe('抽屉键盘和背景锁定', () => {
  it('打开聚焦关闭按钮，Tab循环、Escape关闭后恢复焦点/背景', async () => {
    const app = document.createElement('div'); app.id = 'app'; document.body.append(app)
    const opener = document.createElement('button'); app.append(opener); opener.focus()
    document.body.style.overflow = 'auto'
    const w = mount(Drawer, { props: { open: false, title: '费用详情' }, slots: { default: '<button id="last-control">详情操作</button>' }, attachTo: app, global: { stubs: { Icon: true } } })
    await w.setProps({ open: true }); await flushPromises()
    const dialog = document.querySelector('[role="dialog"]')!
    const controls = dialog.querySelectorAll('button')
    expect(document.activeElement).toBe(controls[0])
    expect(app.inert).toBe(true)
    expect(document.body.style.overflow).toBe('hidden')
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(document.getElementById(dialog.getAttribute('aria-labelledby')!)?.textContent).toBe('费用详情')
    controls[1].focus(); document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', cancelable: true }))
    expect(document.activeElement).toBe(controls[0])
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, cancelable: true }))
    expect(document.activeElement).toBe(controls[1])
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', cancelable: true }))
    expect(w.emitted('close')).toHaveLength(1)
    await w.setProps({ open: false })
    expect(document.activeElement).toBe(opener)
    expect(app.inert).toBe(false)
    expect(document.body.style.overflow).toBe('auto')
  })
  it('卸载释放背景锁和键盘监听', async () => {
    const w = mount(Drawer, { props: { open: true, title: 'test' }, attachTo: document.body, global: { stubs: { Icon: true } } })
    await flushPromises(); w.unmount()
    expect(document.body.style.overflow).toBe('')
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(w.emitted('close')).toBeUndefined()
  })
})
