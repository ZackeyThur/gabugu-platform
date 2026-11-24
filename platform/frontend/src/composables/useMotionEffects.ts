import { onBeforeUnmount, onMounted } from 'vue'

/**
 * Global ReactBits-inspired interactive motion engine:
 * - SpotlightCard: pointer-tracking specular light on Soft Elevation cards
 * - TiltedCard: restrained 3D perspective micro-tilt on [data-tilt] cards
 * - Magnet: gentle magnetic pull on [data-magnet] primary actions
 * - ClickSpark: crisp 8-ray micro-spark burst on [data-spark] / primary CTA clicks
 *
 * Designed as a single passive rAF-throttled listener so zero per-card watchers are needed.
 * Automatically disabled inside Studio / Landing / Auth surfaces, coarse-pointer touch screens
 * (for hover effects), and when prefers-reduced-motion: reduce is active.
 */

const EXCLUDED_ZONE_SELECTOR = '.studio-root, .gabugu-landing, .gabugu-auth, .arena-page, .payment-qr-surface'
const SPOTLIGHT_SELECTOR =
  '.card:not(.bg-emerald-50):not(.bg-red-50):not(.bg-amber-50):not([data-no-spotlight]), ' +
  '.ui-soft-card:not([data-no-spotlight]), ' +
  '.overview-surface:not([data-no-spotlight]), ' +
  '.overview-savings:not([data-no-spotlight]), ' +
  '.usage-discount-card:not([data-no-spotlight])'
const TILT_SELECTOR = '[data-tilt]'
const MAGNET_SELECTOR = '[data-magnet]'
const SPARK_SELECTOR =
  '[data-spark], .btn-primary:not([data-no-spark]), .overview-primary-button:not([data-no-spark]), ' +
  '.btn-alipay:not([data-no-spark]), .btn-wxpay:not([data-no-spark]), .btn-stripe:not([data-no-spark])'

const LIGHT_SPARK_PALETTE = ['#2D5890', '#B65D36', '#4A79B4', '#C68A3C']
const DARK_SPARK_PALETTE = ['#7CA2FF', '#A78BFA', '#38D3B4', '#F6D494']

const SPARK_RAY_COUNT = 8
const SPARK_LIFETIME_MS = 440

function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return true
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function supportsFineHover(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false
  return window.matchMedia('(hover: hover) and (pointer: fine)').matches
}

function isExcluded(el: Element | null): boolean {
  return !el || Boolean(el.closest(EXCLUDED_ZONE_SELECTOR))
}

function isDisabledControl(el: Element): boolean {
  if (el instanceof HTMLButtonElement || el instanceof HTMLInputElement) {
    return el.disabled
  }
  return el.getAttribute('aria-disabled') === 'true'
}

export function spawnClickSpark(clientX: number, clientY: number): HTMLElement | null {
  if (typeof document === 'undefined' || prefersReducedMotion()) return null
  const isDark = document.documentElement.classList.contains('dark')
  const palette = isDark ? DARK_SPARK_PALETTE : LIGHT_SPARK_PALETTE

  const container = document.createElement('span')
  container.className = 'g-spark-burst'
  container.setAttribute('aria-hidden', 'true')
  container.setAttribute('data-testid', 'g-spark-burst')
  container.style.left = `${Math.round(clientX)}px`
  container.style.top = `${Math.round(clientY)}px`

  for (let i = 0; i < SPARK_RAY_COUNT; i++) {
    const ray = document.createElement('i')
    ray.className = 'g-spark-ray'
    const angle = (360 / SPARK_RAY_COUNT) * i + (i % 2 === 0 ? 6 : -4)
    const distance = i % 2 === 0 ? 22 : 17
    ray.style.setProperty('--spark-angle', `${angle}deg`)
    ray.style.setProperty('--spark-dist', `${distance}px`)
    ray.style.setProperty('--spark-color', palette[i % palette.length])
    container.appendChild(ray)
  }

  document.body.appendChild(container)
  return container
}

export function useMotionEffects(): void {
  let activeSpot: HTMLElement | null = null
  let activeTilt: HTMLElement | null = null
  let activeMagnet: HTMLElement | null = null
  let rafId = 0
  let pendingX = 0
  let pendingY = 0
  let pendingTarget: Element | null = null
  const activeBursts = new Set<{ el: HTMLElement; timer: ReturnType<typeof setTimeout> }>()

  function clearSpot(el: HTMLElement | null) {
    if (!el) return
    el.removeAttribute('data-spot-active')
  }

  function clearTilt(el: HTMLElement | null) {
    if (!el) return
    el.removeAttribute('data-tilt-active')
    el.style.removeProperty('--tilt-rx')
    el.style.removeProperty('--tilt-ry')
  }

  function clearMagnet(el: HTMLElement | null) {
    if (!el) return
    el.removeAttribute('data-magnet-active')
    el.style.removeProperty('--mag-x')
    el.style.removeProperty('--mag-y')
  }

  function clearAllHoverStates() {
    clearSpot(activeSpot)
    clearTilt(activeTilt)
    clearMagnet(activeMagnet)
    activeSpot = null
    activeTilt = null
    activeMagnet = null
  }

  function flushPointerFrame() {
    rafId = 0
    if (prefersReducedMotion() || !supportsFineHover()) {
      clearAllHoverStates()
      return
    }

    const target = pendingTarget
    pendingTarget = null
    if (!target || isExcluded(target)) {
      clearAllHoverStates()
      return
    }

    // 1. SpotlightCard tracking
    const nextSpot = target.closest(SPOTLIGHT_SELECTOR) as HTMLElement | null
    if (nextSpot !== activeSpot) {
      clearSpot(activeSpot)
      activeSpot = nextSpot && !isExcluded(nextSpot) ? nextSpot : null
    }
    if (activeSpot) {
      const rect = activeSpot.getBoundingClientRect()
      if (rect.width > 0 && rect.height > 0) {
        const x = pendingX - rect.left
        const y = pendingY - rect.top
        activeSpot.style.setProperty('--spot-x', `${x.toFixed(1)}px`)
        activeSpot.style.setProperty('--spot-y', `${y.toFixed(1)}px`)
        activeSpot.setAttribute('data-spot-active', '1')
      }
    }

    // 2. TiltedCard 3D micro-perspective tracking
    const nextTilt = target.closest(TILT_SELECTOR) as HTMLElement | null
    if (nextTilt !== activeTilt) {
      clearTilt(activeTilt)
      activeTilt = nextTilt && !isExcluded(nextTilt) ? nextTilt : null
    }
    if (activeTilt) {
      const rect = activeTilt.getBoundingClientRect()
      if (rect.width > 0 && rect.height > 0) {
        const nx = Math.max(-1, Math.min(1, ((pendingX - rect.left) / rect.width - 0.5) * 2))
        const ny = Math.max(-1, Math.min(1, ((pendingY - rect.top) / rect.height - 0.5) * 2))
        const maxDeg = Number.parseFloat(activeTilt.dataset.tiltMax || '') || 2.6
        const rx = (-ny * maxDeg).toFixed(2)
        const ry = (nx * maxDeg).toFixed(2)
        activeTilt.style.setProperty('--tilt-rx', `${rx}deg`)
        activeTilt.style.setProperty('--tilt-ry', `${ry}deg`)
        activeTilt.setAttribute('data-tilt-active', '1')
      }
    }

    // 3. Magnet CTA pull tracking
    const nextMagnet = target.closest(MAGNET_SELECTOR) as HTMLElement | null
    const validMagnet =
      nextMagnet && !isExcluded(nextMagnet) && !isDisabledControl(nextMagnet) ? nextMagnet : null
    if (validMagnet !== activeMagnet) {
      clearMagnet(activeMagnet)
      activeMagnet = validMagnet
    }
    if (activeMagnet) {
      const rect = activeMagnet.getBoundingClientRect()
      if (rect.width > 0 && rect.height > 0) {
        const dx = (pendingX - (rect.left + rect.width / 2)) * 0.14
        const dy = (pendingY - (rect.top + rect.height / 2)) * 0.18
        const clampX = Math.max(-3, Math.min(3, dx))
        const clampY = Math.max(-2.5, Math.min(2.5, dy))
        activeMagnet.style.setProperty('--mag-x', `${clampX.toFixed(2)}px`)
        activeMagnet.style.setProperty('--mag-y', `${clampY.toFixed(2)}px`)
        activeMagnet.setAttribute('data-magnet-active', '1')
      }
    }
  }

  function onPointerMove(e: PointerEvent) {
    if (e.pointerType && e.pointerType !== 'mouse' && e.pointerType !== 'pen') return
    pendingX = e.clientX
    pendingY = e.clientY
    pendingTarget = e.target instanceof Element ? e.target : null
    if (!rafId && typeof window.requestAnimationFrame === 'function') {
      rafId = window.requestAnimationFrame(flushPointerFrame)
    }
  }

  function onClick(e: MouseEvent) {
    if (prefersReducedMotion()) return
    const target = e.target instanceof Element ? e.target : null
    if (!target || isExcluded(target)) return
    const trigger = target.closest(SPARK_SELECTOR) as HTMLElement | null
    if (!trigger || isExcluded(trigger) || isDisabledControl(trigger)) return

    let cx = e.clientX
    let cy = e.clientY
    if (cx === 0 && cy === 0) {
      const rect = trigger.getBoundingClientRect()
      cx = rect.left + rect.width / 2
      cy = rect.top + rect.height / 2
    }

    const burst = spawnClickSpark(cx, cy)
    if (!burst) return
    const record = {
      el: burst,
      timer: setTimeout(() => {
        burst.remove()
        activeBursts.delete(record)
      }, SPARK_LIFETIME_MS),
    }
    activeBursts.add(record)
  }

  function onPointerLeaveWindow() {
    if (rafId) {
      window.cancelAnimationFrame(rafId)
      rafId = 0
    }
    pendingTarget = null
    clearAllHoverStates()
  }

  onMounted(() => {
    if (typeof window === 'undefined' || typeof document === 'undefined') return
    window.addEventListener('pointermove', onPointerMove, { passive: true })
    document.addEventListener('click', onClick, { passive: true })
    document.documentElement.addEventListener('pointerleave', onPointerLeaveWindow, { passive: true })
    window.addEventListener('blur', onPointerLeaveWindow, { passive: true })
  })

  onBeforeUnmount(() => {
    if (typeof window === 'undefined' || typeof document === 'undefined') return
    window.removeEventListener('pointermove', onPointerMove)
    document.removeEventListener('click', onClick)
    document.documentElement.removeEventListener('pointerleave', onPointerLeaveWindow)
    window.removeEventListener('blur', onPointerLeaveWindow)
    if (rafId) {
      window.cancelAnimationFrame(rafId)
      rafId = 0
    }
    clearAllHoverStates()
    for (const record of activeBursts) {
      clearTimeout(record.timer)
      record.el.remove()
    }
    activeBursts.clear()
  })
}
