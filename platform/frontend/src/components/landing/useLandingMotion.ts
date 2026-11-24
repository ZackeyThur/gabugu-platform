import { onBeforeUnmount, onMounted, ref, watch, type Ref, type WatchStopHandle } from 'vue'

/** How the first screen appears: in full on a first visit, briefly on a return, or already in place. */
export type LandingOpening = 'full' | 'short' | 'none'

type Library = typeof import('./landingGsap')
type Gsap = Library['gsap']
type Timeline = ReturnType<Gsap['timeline']>

const INTRO_PENDING = 'intro-pending'
const INTRO_CUE = 'intro-cue'
/** The headline waits this long at most for its typeface; text is never held back by the picture. */
const FONT_WAIT_MS = 600
/** ...and this long at most for the animation library; after that the words simply stand there. */
const LIBRARY_WAIT_MS = 1500
/** What an entrance leaves on an element is taken off again, so the stylesheet alone decides how it rests. */
const CLEAR = 'transform,opacity,visibility'
/** A heading is wiped in from its upper edge; the insets leave room for ascenders and descenders. */
const WIPE_FROM = 'inset(-10% -5% 110% -5%)'
const WIPE_TO = 'inset(-10% -5% -20% -5%)'

/**
 * Motion of the Landing, with GSAP. Progressive enhancement: GSAP is downloaded on demand, and without it,
 * or where the visitor asked for less motion, every part of the page is simply visible.
 * The page tells this when it reaches a screen (`arrive`); what the screen holds then comes in, once.
 * How a screen fades while the page leaves it is the stylesheet's (see --screen-near in landing.css).
 */
export function useLandingMotion(
  landingRoot: Ref<HTMLElement | null>,
  activeProtocol: Ref<string>,
  opening: LandingOpening = 'full'
) {
  const motionReduced = ref(false)
  /** performance.now() at the moment the first-screen text began to appear; 0 until then. */
  const introStartedAt = ref(0)
  let library: Library | undefined
  let loading: Promise<Library | undefined> | undefined
  let context: ReturnType<Gsap['context']> | undefined
  let motionQuery: MediaQueryList | undefined
  let stopRootWatch: WatchStopHandle | undefined
  const cleanups: VoidFunction[] = []
  /** Screens the page has been on; their entrances are over, or were never needed. */
  const arrived = new Set<number>([0])
  /** Entrances that wait for the page to reach their screen. */
  const entrances = new Map<number, Timeline>()
  let mounted = false
  let introPlayed = opening === 'none'

  function load() {
    loading ??= import('./landingGsap').then(module => (library = module), () => undefined)
    return loading
  }

  /** Resolves once the page is in front of the visitor; a tab opened in the background waits here. */
  function pageShown() {
    if (!document.hidden) return Promise.resolve()
    return new Promise<void>(resolve => {
      const shown = () => {
        if (document.hidden) return
        document.removeEventListener('visibilitychange', shown)
        resolve()
      }
      document.addEventListener('visibilitychange', shown)
    })
  }

  /** Resolves when the headline's typeface can be drawn, or after a short wait. */
  function headlineTypeface(root: HTMLElement) {
    const headline = root.querySelector<HTMLElement>('.landing-hero h1')
    if (!headline || typeof document.fonts?.load !== 'function') return Promise.resolve()
    const { fontWeight, fontFamily } = getComputedStyle(headline)
    return Promise.race([
      document.fonts.load(`${fontWeight} 1em ${fontFamily}`, headline.textContent ?? '').catch(() => undefined),
      new Promise(resolve => setTimeout(resolve, FONT_WAIT_MS))
    ])
  }

  /** The first screen as it rests, without an entrance. */
  function showFirstScreen(root: HTMLElement) {
    root.classList.remove(INTRO_PENDING)
    introPlayed = true
    if (!introStartedAt.value) introStartedAt.value = performance.now()
  }

  /** The one orchestrated moment: the headline rises piece by piece, then the rest of the first screen. */
  function intro(root: HTMLElement, gsap: Gsap) {
    const short = opening === 'short'
    root.classList.remove(INTRO_PENDING)
    introPlayed = true
    introStartedAt.value = performance.now()
    // Once everything has settled, the arrow under the headline dips twice (see landing.css).
    const timeline = gsap.timeline({ onComplete: () => root.classList.add(INTRO_CUE) })
    timeline.fromTo(root.querySelectorAll('.hero-piece'), { yPercent: 112 }, {
      yPercent: 0, duration: short ? 0.8 : 1.15, ease: 'expo.out', stagger: short ? 0.02 : 0.05, clearProps: 'transform'
    }, 0.1)
    timeline.fromTo(root.querySelectorAll('.hero-eyebrow, .hero-description, .hero-actions'), { opacity: 0, y: 18 }, {
      opacity: 1, y: 0, duration: 0.9, ease: 'power3.out', stagger: 0.12, clearProps: CLEAR
    }, short ? 0.25 : 0.6)
    timeline.fromTo(root.querySelectorAll('.hero-foot, .screen-dots'), { opacity: 0 }, {
      opacity: 1, duration: 0.9, ease: 'power2.out', clearProps: CLEAR
    }, short ? 0.55 : 1.4)
  }

  /** The elements of a part that come in one after another. */
  function piecesOf(part: Element): Element[] {
    if (part.matches('.screen-head, .closing-content, .provider-list, .tool-steps')) return Array.from(part.children)
    if (part.matches('.faq')) return Array.from(part.querySelectorAll('.faq-questions button, .faq-answers'))
    return [part]
  }

  /** What a screen holds comes in once, when the page reaches the screen; coming back does not replay it. */
  function prepareEntrances(root: HTMLElement, gsap: Gsap) {
    root.querySelectorAll<HTMLElement>('.landing-screen').forEach((screen, index) => {
      // Nothing a visitor has already seen disappears to be animated in.
      if (arrived.has(index)) return
      const pieces = Array.from(screen.querySelectorAll('[data-reveal-part]')).flatMap(piecesOf)
      if (!pieces.length) return
      const timeline = gsap.timeline({ paused: true })
      // Opacity, never visibility: what has not come in yet can still be reached with the keyboard and is
      // there for a screen reader.
      timeline.fromTo(pieces, { opacity: 0, y: 22 }, {
        opacity: 1, y: 0, duration: 0.8, ease: 'power3.out', stagger: 0.055, clearProps: CLEAR
      }, 0)
      const heading = screen.querySelector('h2')
      if (heading) {
        timeline.fromTo(heading, { clipPath: WIPE_FROM }, { clipPath: WIPE_TO, duration: 1.05, ease: 'power4.out', clearProps: 'clipPath' }, 0.05)
      }
      // The formula is read term by term, each with the line that explains it.
      const terms = screen.querySelectorAll('.formula-part')
      if (terms.length) {
        timeline.fromTo(terms, { opacity: 0.2 }, { opacity: 1, duration: 0.6, ease: 'power2.out', stagger: 0.26, clearProps: 'opacity' }, 0.3)
        timeline.fromTo(screen.querySelectorAll('.formula-note'), { opacity: 0 }, { opacity: 1, duration: 0.6, ease: 'power2.out', stagger: 0.26, clearProps: 'opacity' }, 0.56)
      }
      // The sample key's meter runs up to what has been used, and the record is filled in field by field.
      const meter = screen.querySelector('.sample-meter span')
      if (meter) {
        timeline.fromTo(meter, { scaleX: 0 }, { scaleX: 1, duration: 1.2, ease: 'power2.out', clearProps: 'transform' }, 0.45)
        timeline.fromTo(screen.querySelectorAll('.sample-record>div'), { opacity: 0 }, { opacity: 1, duration: 0.5, ease: 'power2.out', stagger: 0.12, clearProps: 'opacity' }, 0.9)
      }
      entrances.set(index, timeline)
      // Focus that gets there before the page does (Tab, a screen reader's cursor) shows the screen at once.
      const focused = () => arrive(index, true)
      screen.addEventListener('focusin', focused)
      cleanups.push(() => screen.removeEventListener('focusin', focused))
    })
  }

  /** The page has reached this screen (counted from 0). `atOnce`: without the entrance. */
  function arrive(index: number, atOnce = false) {
    const first = !arrived.has(index)
    arrived.add(index)
    const timeline = entrances.get(index)
    if (!timeline) return
    if (atOnce) timeline.progress(1)
    else if (first) timeline.play()
  }

  function build(root: HTMLElement, module: Library, withIntro: boolean) {
    clearMotion()
    context = module.gsap.context(() => {
      if (withIntro) intro(root, module.gsap)
      else showFirstScreen(root)
      prepareEntrances(root, module.gsap)
    }, root)
  }

  function clearMotion() {
    cleanups.splice(0).forEach(cleanup => cleanup())
    entrances.clear()
    context?.revert()
    context = undefined
    landingRoot.value?.classList.remove(INTRO_PENDING, INTRO_CUE)
  }

  function initializeRoot(root: HTMLElement | null) {
    clearMotion()
    if (!root) return
    const current = () => mounted && landingRoot.value === root && !motionReduced.value && root.isConnected
    if (motionReduced.value || introPlayed) {
      showFirstScreen(root)
      if (!motionReduced.value) void load().then(module => { if (module && current()) build(root, module, false) })
      return
    }
    // The first screen holds its opening pose until the headline's typeface and the animation library have
    // arrived and the page is in view: the headline never shows in a fallback face, and the opening is not
    // spent on a hidden tab. Neither wait is open-ended.
    root.classList.add(INTRO_PENDING)
    const inTime = Promise.race([load(), new Promise<undefined>(resolve => setTimeout(resolve, LIBRARY_WAIT_MS))])
    void Promise.all([inTime, headlineTypeface(root), pageShown()]).then(([module]) => {
      if (!root.classList.contains(INTRO_PENDING)) return
      if (module && current()) return build(root, module, true)
      showFirstScreen(root)
      void load().then(late => { if (late && current()) build(root, late, false) })
    })
  }

  function applyMotionPreference() {
    motionReduced.value = motionQuery?.matches ?? true
    // A preference change must not replay the first screen.
    introPlayed = true
    initializeRoot(landingRoot.value)
  }

  watch(activeProtocol, () => {
    const root = landingRoot.value
    if (!mounted || !root || motionReduced.value || !library) return
    const code = root.querySelector<HTMLElement>('.console-pane.active code')
    if (code) library.gsap.fromTo(code, { opacity: 0 }, { opacity: 1, duration: 0.22, ease: 'power1.out', overwrite: true, clearProps: 'opacity' })
  }, { flush: 'post' })

  onMounted(() => {
    mounted = true
    motionQuery = typeof window.matchMedia === 'function'
      ? window.matchMedia('(prefers-reduced-motion: reduce)')
      : undefined
    motionReduced.value = motionQuery?.matches ?? true
    motionQuery?.addEventListener?.('change', applyMotionPreference)
    stopRootWatch = watch(landingRoot, initializeRoot, { immediate: true, flush: 'post' })
  })

  onBeforeUnmount(() => {
    mounted = false
    stopRootWatch?.()
    motionQuery?.removeEventListener?.('change', applyMotionPreference)
    clearMotion()
  })

  return {
    motionReduced,
    introStartedAt,
    arrive
  }
}
