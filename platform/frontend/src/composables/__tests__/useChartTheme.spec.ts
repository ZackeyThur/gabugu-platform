import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import {
  __getChartThemeObserverStateForTests,
  DARK_CHART_TOKENS,
  LIGHT_CHART_TOKENS,
  useChartTheme,
} from '../useChartTheme'
import { createCrosshairPlugin, formatAdaptiveCurrency } from '@/utils/chartPlugins'

describe('useChartTheme', () => {
  afterEach(() => {
    document.documentElement.classList.remove('dark')
  })

  it('reactively updates tokens when .dark class toggles on documentElement and cleans up observer on unmount', async () => {
    document.documentElement.classList.remove('dark')

    const Probe = defineComponent({
      setup() {
        const theme = useChartTheme()
        return { theme }
      },
      template: '<div :data-dark="theme.isDark.value" :data-primary="theme.tokens.value.primary" />',
    })

    const w1 = mount(Probe)
    const w2 = mount(Probe)

    expect(__getChartThemeObserverStateForTests().refCount).toBe(2)
    expect(__getChartThemeObserverStateForTests().active).toBe(true)
    expect(w1.attributes('data-dark')).toBe('false')
    expect(w1.attributes('data-primary')).toBe(LIGHT_CHART_TOKENS.primary)

    document.documentElement.classList.add('dark')
    await new Promise((resolve) => setTimeout(resolve, 0))
    await nextTick()

    expect(w1.attributes('data-dark')).toBe('true')
    expect(w1.attributes('data-primary')).toBe(DARK_CHART_TOKENS.primary)
    expect(w2.attributes('data-primary')).toBe(DARK_CHART_TOKENS.primary)

    document.documentElement.classList.remove('dark')
    await new Promise((resolve) => setTimeout(resolve, 0))
    await nextTick()

    expect(w1.attributes('data-dark')).toBe('false')
    expect(w1.attributes('data-primary')).toBe(LIGHT_CHART_TOKENS.primary)

    w1.unmount()
    expect(__getChartThemeObserverStateForTests().refCount).toBe(1)
    expect(__getChartThemeObserverStateForTests().active).toBe(true)

    w2.unmount()
    expect(__getChartThemeObserverStateForTests().refCount).toBe(0)
    expect(__getChartThemeObserverStateForTests().active).toBe(false)
  })

  it('wraps series colors seamlessly beyond palette length', () => {
    const Probe = defineComponent({
      setup() {
        return { theme: useChartTheme() }
      },
      template: '<div />',
    })
    const wrapper = mount(Probe)
    const theme = (wrapper.vm as any).theme
    const colors = theme.seriesColors(15)
    expect(colors).toHaveLength(15)
    expect(colors[8]).toBe(colors[0])
    expect(colors[14]).toBe(colors[6])
    wrapper.unmount()
  })

  it('formats micro-currency ticks adaptively without collapsing to $0.00', () => {
    expect(formatAdaptiveCurrency(0, 0.005)).toBe('$0')
    expect(formatAdaptiveCurrency(0.0025, 0.005)).toBe('$0.0025')
    expect(formatAdaptiveCurrency(0.045, 0.08)).toBe('$0.045')
    expect(formatAdaptiveCurrency(12.5, 50)).toBe('$12.50')
    expect(formatAdaptiveCurrency(1500, 2000)).toBe('$1.5K')
  })

  it('does not leak MutationObserver refCount when invoked outside an effect scope', () => {
    expect(__getChartThemeObserverStateForTests().refCount).toBe(0)
    const theme = useChartTheme()
    expect(theme.tokens.value.primary).toBe(LIGHT_CHART_TOKENS.primary)
    expect(__getChartThemeObserverStateForTests().refCount).toBe(0)
    expect(__getChartThemeObserverStateForTests().active).toBe(false)
  })

  it('draws crosshair using chart active elements when tooltip active elements are empty (keyboard nav)', () => {
    const plugin = createCrosshairPlugin(() => '#2D5890')
    const stroke = vi.fn()
    const moveTo = vi.fn()
    const lineTo = vi.fn()
    const fakeChart = {
      tooltip: { getActiveElements: () => [] },
      getActiveElements: () => [{ element: { x: 120 } }],
      chartArea: { top: 10, bottom: 200 },
      ctx: {
        save: vi.fn(),
        restore: vi.fn(),
        beginPath: vi.fn(),
        setLineDash: vi.fn(),
        moveTo,
        lineTo,
        stroke,
        lineWidth: 0,
        strokeStyle: '',
      },
    }
    ;(plugin.afterDatasetsDraw as any)(fakeChart)
    expect(moveTo).toHaveBeenCalledWith(120, 10)
    expect(lineTo).toHaveBeenCalledWith(120, 200)
    expect(stroke).toHaveBeenCalledTimes(1)
  })
})
