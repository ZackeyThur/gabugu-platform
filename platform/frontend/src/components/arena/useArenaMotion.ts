import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

type GsapModule = typeof import('@/components/landing/landingGsap')
type Gsap = GsapModule['gsap']

const CLEAR = 'transform,opacity'

/**
 * Progressive-enhancement motion controller for the Arena leaderboard.
 *
 * Uses the shared on-demand `vendor-gsap` chunk so no other page pays its download cost.
 * Strictly honors `prefers-reduced-motion` and never leaves elements hidden when GSAP is
 * unavailable or slow to load.
 */
export function useArenaMotion(rootRef: Ref<HTMLElement | null>) {
  const reducedMotion = ref(true)
  let gsapInstance: Gsap | null = null
  let loadPromise: Promise<Gsap | null> | null = null
  let motionQuery: MediaQueryList | undefined
  let mounted = false
  const activeTweens = new Set<{ kill(): void }>()

  function loadGsap(): Promise<Gsap | null> {
    if (gsapInstance) return Promise.resolve(gsapInstance)
    if (!loadPromise) {
      loadPromise = import('@/components/landing/landingGsap')
        .then((mod) => {
          gsapInstance = mod.gsap
          return mod.gsap
        })
        .catch(() => null)
    }
    return loadPromise
  }

  function track<T extends { kill(): void }>(tween: T): T {
    activeTweens.add(tween)
    return tween
  }

  function killAll() {
    activeTweens.forEach((tw) => {
      try {
        tw.kill()
      } catch {
        // Ignore already-reverted tweens.
      }
    })
    activeTweens.clear()
  }

  /**
   * Choreographed initial entrance for Hero + Tabs + Podium + Insights + first rows.
   * Never hides elements beforehand: animates cleanly from subtle offset once ready.
   */
  async function playIntro() {
    if (!mounted || reducedMotion.value) return
    const root = rootRef.value
    if (!root) return
    const gsap = await loadGsap()
    if (!gsap || !mounted || reducedMotion.value || !root.isConnected) return

    const tl = gsap.timeline({ defaults: { ease: 'expo.out' } })
    track(tl)

    const heroParts = root.querySelectorAll('[data-arena-intro="hero"]')
    if (heroParts.length) {
      tl.fromTo(
        heroParts,
        { opacity: 0, y: 16 },
        { opacity: 1, y: 0, duration: 0.72, stagger: 0.06, clearProps: CLEAR },
        0
      )
    }

    const tabsAndMeta = root.querySelectorAll('[data-arena-intro="chrome"]')
    if (tabsAndMeta.length) {
      tl.fromTo(
        tabsAndMeta,
        { opacity: 0, y: 10 },
        { opacity: 1, y: 0, duration: 0.56, stagger: 0.05, clearProps: CLEAR },
        0.12
      )
    }

    const podiumCards = root.querySelectorAll('[data-arena-anim="podium"]')
    if (podiumCards.length) {
      tl.fromTo(
        podiumCards,
        { opacity: 0, y: 20 },
        { opacity: 1, y: 0, duration: 0.68, stagger: 0.07, clearProps: CLEAR },
        0.18
      )
    }

    const insights = root.querySelectorAll('[data-arena-anim="insight"]')
    if (insights.length) {
      tl.fromTo(
        insights,
        { opacity: 0, y: 12 },
        { opacity: 1, y: 0, duration: 0.52, stagger: 0.05, clearProps: CLEAR },
        0.32
      )
    }

    const rows = Array.from(root.querySelectorAll('[data-arena-anim="row"]')).slice(0, 14)
    if (rows.length) {
      tl.fromTo(
        rows,
        { opacity: 0, y: 10 },
        { opacity: 1, y: 0, duration: 0.44, stagger: 0.018, ease: 'power3.out', clearProps: CLEAR },
        0.36
      )
    }
  }

  /**
   * Snappy track-switch choreography when switching between Text / Code / Vision / Image.
   */
  async function playBoardSwitch() {
    if (!mounted || reducedMotion.value) return
    const root = rootRef.value
    if (!root) return
    const gsap = await loadGsap()
    if (!gsap || !mounted || reducedMotion.value || !root.isConnected) return

    const podiumCards = root.querySelectorAll('[data-arena-anim="podium"]')
    if (podiumCards.length) {
      track(
        gsap.fromTo(
          podiumCards,
          { opacity: 0, y: 14 },
          { opacity: 1, y: 0, duration: 0.46, stagger: 0.05, ease: 'expo.out', overwrite: true, clearProps: CLEAR }
        )
      )
    }

    const insights = root.querySelectorAll('[data-arena-anim="insight"]')
    if (insights.length) {
      track(
        gsap.fromTo(
          insights,
          { opacity: 0, y: 8 },
          { opacity: 1, y: 0, duration: 0.36, stagger: 0.04, ease: 'power3.out', overwrite: true, clearProps: CLEAR }
        )
      )
    }

    const rows = Array.from(root.querySelectorAll('[data-arena-anim="row"]')).slice(0, 14)
    if (rows.length) {
      track(
        gsap.fromTo(
          rows,
          { opacity: 0, y: 8 },
          { opacity: 1, y: 0, duration: 0.34, stagger: 0.014, ease: 'power2.out', overwrite: true, clearProps: CLEAR }
        )
      )
    }
  }

  /**
   * Smooth tabular number tweening (e.g. rolling podium scores into place without layout jitter).
   */
  async function tweenNumbers(targets: number[], onUpdate: (values: number[]) => void, previous?: number[]) {
    if (!mounted || reducedMotion.value || targets.length === 0) {
      onUpdate(targets)
      return
    }
    const gsap = await loadGsap()
    if (!gsap || !mounted || reducedMotion.value) {
      onUpdate(targets)
      return
    }

    const state: Record<string, number> = {}
    const toProps: Record<string, number> = {}
    targets.forEach((target, i) => {
      const start = previous && typeof previous[i] === 'number' && previous[i] > 0 ? previous[i] : Math.max(0, target - 48)
      state[`v${i}`] = start
      toProps[`v${i}`] = target
    })

    track(
      gsap.to(state, {
        ...toProps,
        duration: 0.68,
        ease: 'expo.out',
        overwrite: true,
        onUpdate: () => {
          onUpdate(targets.map((_, i) => state[`v${i}`]))
        },
        onComplete: () => {
          onUpdate(targets)
        }
      })
    )
  }

  function updateMotionPref() {
    reducedMotion.value = motionQuery?.matches ?? true
  }

  onMounted(() => {
    mounted = true
    motionQuery =
      typeof window !== 'undefined' && typeof window.matchMedia === 'function'
        ? window.matchMedia('(prefers-reduced-motion: reduce)')
        : undefined
    reducedMotion.value = motionQuery?.matches ?? true
    motionQuery?.addEventListener?.('change', updateMotionPref)
    if (!reducedMotion.value) {
      void loadGsap()
    }
  })

  onBeforeUnmount(() => {
    mounted = false
    motionQuery?.removeEventListener?.('change', updateMotionPref)
    killAll()
  })

  return {
    reducedMotion,
    playIntro,
    playBoardSwitch,
    tweenNumbers
  }
}

/**
 * Measures the active child button inside a horizontal strip (tabs or segmented control)
 * and outputs CSS custom properties / pixel geometry for a smoothly gliding indicator.
 */
export function measureIndicator(container: HTMLElement | null, selector: string): { left: number; width: number; ready: boolean } {
  if (!container) return { left: 0, width: 0, ready: false }
  const active = container.querySelector<HTMLElement>(selector)
  if (!active) return { left: 0, width: 0, ready: false }
  const left = active.offsetLeft
  const width = active.offsetWidth
  return { left, width, ready: width > 0 }
}
