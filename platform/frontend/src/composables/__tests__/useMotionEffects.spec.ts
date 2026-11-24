import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { useMotionEffects } from '../useMotionEffects'

enableAutoUnmount(afterEach)

function mockMatchMedia(opts: { reduced?: boolean; fineHover?: boolean } = {}) {
  const { reduced = false, fineHover = true } = opts
  vi.stubGlobal(
    'matchMedia',
    vi.fn((query: string) => ({
      matches: query.includes('prefers-reduced-motion')
        ? reduced
        : query.includes('hover: hover')
          ? fineHover
          : false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
  )
}

function makeHost() {
  return defineComponent({
    setup() {
      useMotionEffects()
      return {}
    },
    template: `
      <div>
        <div class="card" data-tilt data-testid="card">
          <button type="button" data-magnet data-spark data-testid="cta">Go</button>
        </div>
        <div class="studio-root">
          <div class="card" data-testid="studio-card">
            <button type="button" data-spark data-testid="studio-btn">Studio</button>
          </div>
        </div>
      </div>
    `,
  })
}

function mockRect(el: HTMLElement, rect: Partial<DOMRect>) {
  vi.spyOn(el, 'getBoundingClientRect').mockReturnValue({
    x: rect.left ?? 0,
    y: rect.top ?? 0,
    left: rect.left ?? 0,
    top: rect.top ?? 0,
    right: (rect.left ?? 0) + (rect.width ?? 200),
    bottom: (rect.top ?? 0) + (rect.height ?? 100),
    width: rect.width ?? 200,
    height: rect.height ?? 100,
    toJSON: () => ({}),
  })
}

describe('useMotionEffects', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockMatchMedia({ reduced: false, fineHover: true })
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
      cb(performance.now())
      return 1
    })
    vi.stubGlobal('cancelAnimationFrame', vi.fn())
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
    document.querySelectorAll('[data-testid="g-spark-burst"]').forEach((el) => el.remove())
  })

  it('tracks spotlight, tilt, and magnet variables and resets cleanly on window pointerleave', () => {
    const wrapper = mount(makeHost(), { attachTo: document.body })
    const card = wrapper.get('[data-testid="card"]').element as HTMLElement
    const cta = wrapper.get('[data-testid="cta"]').element as HTMLElement
    mockRect(card, { left: 100, top: 100, width: 200, height: 100 })
    mockRect(cta, { left: 150, top: 120, width: 100, height: 40 })

    const moveEvent = new MouseEvent('pointermove', {
      clientX: 240,
      clientY: 130,
      bubbles: true,
    })
    cta.dispatchEvent(moveEvent)

    expect(card.getAttribute('data-spot-active')).toBe('1')
    expect(card.style.getPropertyValue('--spot-x')).toBe('140.0px')
    expect(card.style.getPropertyValue('--spot-y')).toBe('30.0px')
    expect(card.getAttribute('data-tilt-active')).toBe('1')
    expect(card.style.getPropertyValue('--tilt-rx')).not.toBe('')
    expect(cta.getAttribute('data-magnet-active')).toBe('1')
    expect(cta.style.getPropertyValue('--mag-x')).not.toBe('')

    document.documentElement.dispatchEvent(new MouseEvent('pointerleave'))
    expect(card.getAttribute('data-spot-active')).toBeNull()
    expect(card.getAttribute('data-tilt-active')).toBeNull()
    expect(cta.getAttribute('data-magnet-active')).toBeNull()
  })

  it('spawns and auto-cleans click spark bursts without leaking DOM nodes', () => {
    const wrapper = mount(makeHost(), { attachTo: document.body })
    const cta = wrapper.get('[data-testid="cta"]').element as HTMLElement

    cta.dispatchEvent(new MouseEvent('click', { clientX: 200, clientY: 150, bubbles: true }))
    const burst = document.querySelector('[data-testid="g-spark-burst"]')
    expect(burst).not.toBeNull()
    expect(burst?.querySelectorAll('.g-spark-ray')).toHaveLength(8)

    vi.advanceTimersByTime(500)
    expect(document.querySelector('[data-testid="g-spark-burst"]')).toBeNull()
  })

  it('completely ignores excluded zones like .studio-root and respects prefers-reduced-motion', () => {
    const wrapper = mount(makeHost(), { attachTo: document.body })
    const studioCard = wrapper.get('[data-testid="studio-card"]').element as HTMLElement
    const studioBtn = wrapper.get('[data-testid="studio-btn"]').element as HTMLElement
    mockRect(studioCard, { left: 0, top: 0, width: 200, height: 100 })

    studioCard.dispatchEvent(new MouseEvent('pointermove', { clientX: 50, clientY: 50, bubbles: true }))
    expect(studioCard.getAttribute('data-spot-active')).toBeNull()

    studioBtn.dispatchEvent(new MouseEvent('click', { clientX: 50, clientY: 50, bubbles: true }))
    expect(document.querySelector('[data-testid="g-spark-burst"]')).toBeNull()

    mockMatchMedia({ reduced: true, fineHover: true })
    const cta = wrapper.get('[data-testid="cta"]').element as HTMLElement
    cta.dispatchEvent(new MouseEvent('click', { clientX: 120, clientY: 120, bubbles: true }))
    expect(document.querySelector('[data-testid="g-spark-burst"]')).toBeNull()
  })
})
