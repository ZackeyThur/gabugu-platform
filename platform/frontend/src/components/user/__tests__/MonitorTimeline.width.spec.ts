import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { MonitorTimelinePoint } from '@/api/channelMonitor'
import MonitorTimeline from '../monitor/MonitorTimeline.vue'

vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

/** 延迟取唯一值，用它验证「保留哪 14 条、顺序如何」，不依赖时钟与翻译。 */
function point(i: number, status: MonitorTimelinePoint['status'] = 'operational'): MonitorTimelinePoint {
  return {
    status,
    latency_ms: 100 + i,
    ping_latency_ms: 30,
    // 下标 0 是最近一次
    checked_at: new Date(Date.UTC(2026, 10, 21, 8, 0, 0) - i * 60_000).toISOString(),
  }
}

const mountTimeline = (buckets: MonitorTimelinePoint[], extra: Record<string, unknown> = {}) =>
  mount(MonitorTimeline, { props: { buckets, countdownSeconds: 60, ...extra } })

const bars = (wrapper: ReturnType<typeof mountTimeline>) =>
  wrapper.findAll('[data-testid="timeline-bar"]')

describe('MonitorTimeline wide-bar rendering', () => {
  it('renders exactly 14 slots for a 60-point history', () => {
    const wrapper = mountTimeline(Array.from({ length: 60 }, (_, i) => point(i)))
    expect(bars(wrapper).length).toBe(14)
    expect(wrapper.find('[data-testid="timeline-track"]').exists()).toBe(true)
  })

  it('keeps only the newest 14 samples and puts the latest on the right', () => {
    const data = Array.from({ length: 60 }, (_, i) => point(i, i === 0 ? 'failed' : 'operational'))
    const all = bars(mountTimeline(data))

    // 左起第一格 = 第 14 新的样本（latency 113）
    expect(all[0].attributes('title')).toContain('113ms')
    // 最右一格 = 最新样本（latency 100），且是本次失败状态
    const last = all[all.length - 1]
    expect(last.attributes('title')).toContain('100ms')
    expect(last.classes().join(' ')).toContain('bg-red-500')
    // 更旧的样本（114ms 起）不能被画出来
    expect(all.map((b) => b.attributes('title')).join('|')).not.toContain('114ms')
  })

  it('pads with grey placeholders when there are fewer samples than slots', () => {
    const wrapper = mountTimeline([point(0), point(1), point(2)])
    const all = bars(wrapper)

    expect(all.length).toBe(14)
    expect(all.filter((b) => b.classes().join(' ').includes('bg-gray-300')).length).toBe(11)
    // 标题只报真实条数，占位格不算检测
    expect(wrapper.text()).toContain('"n":3')
  })

  it('never appends ms to a missing latency in the tooltip', () => {
    const wrapper = mountTimeline([{ ...point(0), latency_ms: null }])
    expect(bars(wrapper).at(-1)!.attributes('title')).not.toContain('-ms')
  })

  it('shows the auto-refresh-off label instead of a countdown when refresh is off', () => {
    const wrapper = mountTimeline([point(0)], { autoRefreshOn: false, countdownSeconds: 0 })
    expect(wrapper.text()).toContain('monitorCommon.autoRefreshOff')
    expect(wrapper.text()).not.toContain('monitorCommon.nextUpdateIn')
  })
})
