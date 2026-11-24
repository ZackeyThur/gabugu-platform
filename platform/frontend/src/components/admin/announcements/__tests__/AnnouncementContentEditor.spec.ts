import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import Editor from '../AnnouncementContentEditor.vue'
import { joinAnnouncementImages, splitAnnouncementImages } from '@/utils/announcementImages'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

function editor(content = '正文') {
  return mount(defineComponent({
    components: { Editor },
    setup() { return { value: ref(content) } },
    template: '<Editor v-model="value" />'
  }))
}

describe('AnnouncementContentEditor', () => {
  it('adds an image, reloads its stored content, edits the body, and removes it', async () => {
    const wrapper = editor()
    await wrapper.get('input[type=text]').setValue('https://gabugu.com/poster.png')
    await wrapper.findAll('button').find(button => button.text().includes('addUrl'))!.trigger('click')
    const saved = (wrapper.vm as unknown as { value: string }).value
    const reopened = editor(saved)
    expect(reopened.get('img').attributes('src')).toBe('https://gabugu.com/poster.png')
    expect(reopened.get('textarea').element.value).toBe('正文')
    await reopened.get('textarea').setValue('修改正文')
    expect(splitAnnouncementImages((reopened.vm as unknown as { value: string }).value).images).toHaveLength(1)
    await reopened.findAll('button').find(button => button.text() === 'common.remove')!.trigger('click')
    expect((reopened.vm as unknown as { value: string }).value).toBe('修改正文')
    wrapper.unmount(); reopened.unmount()
  })

  it('keeps image descriptions and reordered images after saving', async () => {
    const wrapper = editor(joinAnnouncementImages('正文', [{ src: '/a.png', alt: 'A' }, { src: '/b.png', alt: 'B' }]))
    await wrapper.findAll('input[maxlength="200"]')[0]!.setValue('第一张')
    await wrapper.findAll('li')[1]!.findAll('button')[0]!.trigger('click')
    expect(splitAnnouncementImages((wrapper.vm as unknown as { value: string }).value).images).toEqual([{ src: '/b.png', alt: 'B' }, { src: '/a.png', alt: '第一张' }])
    wrapper.unmount()
  })

  it('rejects unsafe URLs and SVG uploads without changing existing content', async () => {
    const wrapper = editor()
    await wrapper.get('input[type=text]').setValue('javascript:alert(1)')
    await wrapper.findAll('button').find(button => button.text().includes('addUrl'))!.trigger('click')
    expect(wrapper.get('[role=alert]').text()).toContain('invalidUrl')
    const input = wrapper.get('input[type=file]')
    Object.defineProperty(input.element, 'files', { value: [new File(['<svg/>'], 'bad.svg', { type: 'image/svg+xml' })], configurable: true })
    await input.trigger('change')
    expect(wrapper.get('[role=alert]').text()).toContain('invalidType')
    expect((wrapper.vm as unknown as { value: string }).value).toBe('正文')
    wrapper.unmount()
  })
})
