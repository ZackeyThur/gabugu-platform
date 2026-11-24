import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import UserErrorRequestsTable from '../UserErrorRequestsTable.vue'
import UserErrorDetailModal from '../UserErrorDetailModal.vue'
import type { UserErrorRequest } from '@/types'

vi.mock('@/api/usage', () => ({ getMyErrorDetail: vi.fn() }))
vi.mock('vue-i18n', async importOriginal => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({
  t: (key: string, params?: Record<string, unknown>) => key === 'usageRecordDetails.openDetails'
    ? `View ${params?.model} at ${params?.time}` : key,
}) }))

// Render only the columns the real table receives, so hiding model also removes
// its slot. The details action must remain usable independently of that column.
const tableStub = defineComponent({
  props: ['columns', 'data'],
  template: '<div><div v-for="row in data" :key="row.id"><template v-for="column in columns" :key="column.key"><slot :name="`cell-${column.key}`" :row="row" /></template></div></div>',
})

describe('user error details action', () => {
  it('remains focusable and opens the correct model-less request when model is hidden', async () => {
    const rows: UserErrorRequest[] = [41, 82].map(id => ({
      id, created_at: '2026-11-09T00:00:00Z', model: '', inbound_endpoint: '',
      status_code: 401, category: 'auth', platform: '', message: 'Authentication failed',
      key_name: '', key_deleted: false,
    }))
    const wrapper = mount(UserErrorRequestsTable, {
      attachTo: document.body,
      props: {
        rows, total: 2, loading: false, page: 1, pageSize: 20,
        timezone: 'Asia/Shanghai', visibleColumnKeys: ['status', 'created_at'],
      },
      global: { stubs: {
        DataTable: tableStub,
        Pagination: true,
        UserErrorDetailModal: { props: ['show', 'errorId', 'timezone'], template: '<div />' },
        IpGeoBatchToolbar: true,
        IpGeoCell: true,
        ModelIcon: true,
      } },
    })
    try {
      const actions = wrapper.findAll('button')
      expect(actions).toHaveLength(2)
      const target = actions[1]
      expect(target.attributes('type')).toBe('button')
      expect(target.attributes('aria-label')).toContain('View — at ')
      expect((target.element as HTMLButtonElement).tabIndex).toBe(0)
      ;(target.element as HTMLButtonElement).focus()
      expect(document.activeElement).toBe(target.element)
      // A native button's activation click is the same handler used by Enter/Space.
      await target.trigger('click')
      expect(wrapper.findComponent(UserErrorDetailModal).props()).toMatchObject({
        show: true, errorId: 82, timezone: 'Asia/Shanghai',
      })
    } finally {
      wrapper.unmount()
    }
  })
})
