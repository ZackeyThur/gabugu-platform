import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AnnouncementContent from '../AnnouncementContent.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
afterEach(() => { document.body.innerHTML = ''; document.body.classList.remove('modal-open') })

describe('AnnouncementContent', () => {
  it('renders saved bitmap bytes and strips active content', async () => {
    const src = 'data:image/png;base64,YWJjZA=='
    const wrapper = mount(AnnouncementContent, { global: { stubs: { transition: true } }, props: { content: `正文\n\n<img src="${src}" alt="海报" onerror="alert(1)"><script>alert(1)</script>` } })
    expect(wrapper.get('img').attributes('src')).toBe(src)
    expect(wrapper.get('img').attributes('onerror')).toBeUndefined()
    expect(wrapper.find('script').exists()).toBe(false)
    expect(wrapper.get('img').attributes('tabindex')).toBe('0')
    await wrapper.get('img').trigger('keydown', { key: 'Enter' })
    expect(document.body.querySelector('[role=dialog] img')?.getAttribute('src')).toBe(src)
    const parentEscape = vi.fn()
    document.addEventListener('keydown', parentEscape)
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await wrapper.vm.$nextTick()
    expect(parentEscape).not.toHaveBeenCalled()
    document.removeEventListener('keydown', parentEscape)
    expect(document.body.querySelector('[role=dialog]')).toBeNull()
    await wrapper.get('img').trigger('click')
    expect(document.body.querySelector('[role=dialog] img')).not.toBeNull()
    await wrapper.setProps({ content: '新公告' })
    expect(document.body.querySelector('[role=dialog]')).toBeNull()
    wrapper.unmount()
  })
})
