import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAnnouncementStore } from '../announcements'
import type { UserAnnouncement } from '@/types'

const list = vi.hoisted(() => vi.fn())
const markRead = vi.hoisted(() => vi.fn())
vi.mock('@/api', () => ({ announcementsAPI: { list, markRead } }))

const notice = (id: number): UserAnnouncement => ({
  id, title: `Notice ${id}`, content: 'Content', notify_mode: 'popup',
  created_at: '2026-09-18T00:00:00Z', updated_at: '2026-09-18T00:00:00Z'
})

function pendingList() {
  let resolve!: (value: UserAnnouncement[]) => void
  let reject!: (reason: Error) => void
  const promise = new Promise<UserAnnouncement[]>((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.resetAllMocks()
  vi.spyOn(console, 'error').mockImplementation(() => {})
})
afterEach(() => vi.restoreAllMocks())

describe('announcement fetch ownership', () => {
  it('does not restore announcements or popups after logout resets the store', async () => {
    const store = useAnnouncementStore()
    const old = pendingList()
    list.mockReturnValueOnce(old.promise)
    const request = store.fetchAnnouncements()
    store.reset()
    old.resolve([notice(1)])
    await request
    expect(store.announcements).toEqual([])
    expect(store.currentPopup).toBeNull()
    expect(store.loading).toBe(false)
  })

  it('keeps the new session loading when an old request finishes', async () => {
    const store = useAnnouncementStore()
    const old = pendingList()
    const current = pendingList()
    list.mockReturnValueOnce(old.promise).mockReturnValueOnce(current.promise)
    const oldRequest = store.fetchAnnouncements()
    store.reset()
    const currentRequest = store.fetchAnnouncements()
    old.resolve([])
    await oldRequest
    expect(store.loading).toBe(true)
    current.resolve([notice(2)])
    await currentRequest
    expect(store.announcements.map(a => a.id)).toEqual([2])
    expect(store.loading).toBe(false)
  })

  it('keeps the latest forced refresh when responses arrive out of order', async () => {
    const store = useAnnouncementStore()
    const old = pendingList()
    list.mockReturnValueOnce(old.promise).mockResolvedValueOnce([notice(2)])
    const oldRequest = store.fetchAnnouncements()
    await store.fetchAnnouncements(true)
    old.resolve([notice(1)])
    await oldRequest
    expect(store.announcements.map(a => a.id)).toEqual([2])
    expect(store.currentPopup?.id).toBe(2)
  })

  it('does not clear the new session throttle when an old request rejects', async () => {
    const store = useAnnouncementStore()
    const old = pendingList()
    list.mockReturnValueOnce(old.promise).mockResolvedValue([notice(2)])
    const oldRequest = store.fetchAnnouncements()
    store.reset()
    await store.fetchAnnouncements()
    old.reject(new Error('old request failed'))
    await oldRequest
    await store.fetchAnnouncements()
    expect(list).toHaveBeenCalledTimes(2)
    expect(store.announcements.map(a => a.id)).toEqual([2])
  })

  it('still retries a failed current request', async () => {
    const store = useAnnouncementStore()
    list.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce([notice(1)])
    await store.fetchAnnouncements()
    await store.fetchAnnouncements()
    expect(list).toHaveBeenCalledTimes(2)
    expect(store.currentPopup?.id).toBe(1)
  })

  it('still throttles ordinary concurrent fetches', async () => {
    const store = useAnnouncementStore()
    const current = pendingList()
    list.mockReturnValueOnce(current.promise)
    const request = store.fetchAnnouncements()
    await store.fetchAnnouncements()
    expect(list).toHaveBeenCalledTimes(1)
    expect(store.loading).toBe(true)
    current.resolve([])
    await request
    expect(store.loading).toBe(false)
  })

  it('extracts dashboardPosters before the 20-item bell cap, sorts stably, keeps read posters, and clears on reset', async () => {
    const store = useAnnouncementStore()
    const items: UserAnnouncement[] = Array.from({ length: 24 }, (_, idx) => ({
      id: idx + 1,
      title: `Regular ${idx + 1}`,
      content: 'Plain text',
      notify_mode: 'silent',
      show_on_dashboard: false,
      created_at: `2026-11-${String(Math.min(28, idx + 1)).padStart(2, '0')}T00:00:00Z`,
      updated_at: '2026-11-01T00:00:00Z',
    }))
    // 25th item (beyond bell slice(0, 20)) with poster image + internal link
    items.push({
      id: 99,
      title: 'Deep Poster',
      content: 'Promo\n![Banner](https://gabugu.com/banner-99.png)',
      notify_mode: 'silent',
      show_on_dashboard: true,
      dashboard_link: '/purchase',
      created_at: '2026-11-20T08:00:00Z',
      updated_at: '2026-11-20T08:00:00Z',
    })
    // Newer poster without link, and a show_on_dashboard item with NO image (must be skipped)
    items.push({
      id: 100,
      title: 'Newest Poster',
      content: '![Hero](/brand/hero.png)',
      notify_mode: 'silent',
      show_on_dashboard: true,
      dashboard_link: 'javascript:alert(1)', // unsafe -> sanitized to null (falls back to modal)
      created_at: '2026-11-22T08:00:00Z',
      updated_at: '2026-11-22T08:00:00Z',
    })
    items.push({
      id: 101,
      title: 'No Image Poster',
      content: 'Text only without image',
      notify_mode: 'silent',
      show_on_dashboard: true,
      created_at: '2026-11-23T08:00:00Z',
      updated_at: '2026-11-23T08:00:00Z',
    })

    list.mockResolvedValueOnce(items)
    markRead.mockResolvedValueOnce(undefined)

    await store.fetchAnnouncements(true)

    expect(store.announcements).toHaveLength(20)
    expect(store.dashboardPosters.map(p => p.announcement.id)).toEqual([100, 99])
    expect(store.dashboardPosters[0].link).toBeNull()
    expect(store.dashboardPosters[1].link).toEqual({ kind: 'internal', href: '/purchase' })

    // Marking read preserves poster order and updates read_at
    await store.markAsRead(100)
    expect(store.dashboardPosters.map(p => p.announcement.id)).toEqual([100, 99])
    expect(store.dashboardPosters[0].announcement.read_at).toBeTruthy()

    store.reset()
    expect(store.dashboardPosters).toEqual([])
  })
})
