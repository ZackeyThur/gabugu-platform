import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import type { DashboardPosterItem } from '@/stores/announcements'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

const routerPush = vi.hoisted(() => vi.fn())
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush }),
}))

import DashboardPosterShowcase from '../DashboardPosterShowcase.vue'

enableAutoUnmount(afterEach)

const makePoster = (
  id: number,
  link: DashboardPosterItem['link'] = null
): DashboardPosterItem => ({
  announcement: {
    id,
    title: `Promo ${id}`,
    content: `![Poster ${id}](https://gabugu.com/p${id}.png)`,
    notify_mode: 'silent',
    show_on_dashboard: true,
    created_at: '2026-11-20T00:00:00Z',
    updated_at: '2026-11-20T00:00:00Z',
  },
  image: { src: `https://gabugu.com/p${id}.png`, alt: `Poster ${id}` },
  link,
})

describe('DashboardPosterShowcase 16:9 海报橱窗', () => {
  it('无海报时不渲染任何占位元素', () => {
    const w = mount(DashboardPosterShowcase, {
      props: { posters: [] },
      global: { stubs: { Icon: true } },
    })
    expect(w.find('.overview-poster-card').exists()).toBe(false)
  })

  it('单张海报隐藏轮播控件，无链接时点击弹出公告详情', async () => {
    const poster = makePoster(1, null)
    const w = mount(DashboardPosterShowcase, {
      props: { posters: [poster] },
      global: { stubs: { Icon: true } },
    })
    expect(w.find('.overview-poster-edge').exists()).toBe(false)
    expect(w.get('.overview-poster-img').attributes('src')).toBe('https://gabugu.com/p1.png')

    await w.get('.overview-poster-stage').trigger('click')
    expect(w.emitted('open-announcement')?.[0]).toEqual([poster.announcement])
    await w.get('.overview-poster-dock').trigger('click')
    expect(w.emitted('open-announcement')?.[1]).toEqual([poster.announcement])
  })

  it('多张海报支持切换、站内路由跳转与安全外链新窗口打开', async () => {
    routerPush.mockReset()
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const posters = [
      makePoster(1, { kind: 'internal', href: '/purchase?amount=100' }),
      makePoster(2, { kind: 'external', href: 'https://gabugu.com/event' }),
    ]
    const w = mount(DashboardPosterShowcase, {
      props: { posters },
      global: { stubs: { Icon: true } },
    })

    expect(w.findAll('.overview-poster-dot')).toHaveLength(2)
    await w.get('.overview-poster-stage').trigger('click')
    expect(routerPush).toHaveBeenCalledWith('/purchase?amount=100')

    await w.get('.overview-poster-edge.is-next').trigger('click')
    expect(w.get('.overview-poster-title').text()).toBe('Promo 2')
    await w.get('.overview-poster-stage').trigger('click')
    expect(openSpy).toHaveBeenCalledWith('https://gabugu.com/event', '_blank', 'noopener,noreferrer')
    openSpy.mockRestore()
  })

  it('图片加载失败时平滑回退到品牌兜底画布且保留点击查看公告', async () => {
    const poster = makePoster(9, null)
    const w = mount(DashboardPosterShowcase, {
      props: { posters: [poster] },
      global: { stubs: { Icon: true } },
    })

    await w.get('.overview-poster-img').trigger('error')
    expect(w.find('[data-testid="poster-image-fallback"]').exists()).toBe(true)
    await w.get('.overview-poster-stage').trigger('click')
    expect(w.emitted('open-announcement')?.[0]).toEqual([poster.announcement])
  })

  it('多张海报支持键盘左右方向键切换', async () => {
    const posters = [makePoster(1), makePoster(2)]
    const w = mount(DashboardPosterShowcase, {
      props: { posters },
      global: { stubs: { Icon: true } },
    })
    expect(w.get('.overview-poster-title').text()).toBe('Promo 1')
    await w.get('.overview-poster-card').trigger('keydown', { key: 'ArrowRight' })
    expect(w.get('.overview-poster-title').text()).toBe('Promo 2')
    await w.get('.overview-poster-card').trigger('keydown', { key: 'ArrowLeft' })
    expect(w.get('.overview-poster-title').text()).toBe('Promo 1')
  })
})
