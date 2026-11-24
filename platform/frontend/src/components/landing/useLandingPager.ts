import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue'
import { easeTurn, nearestStop, readWheel, stopBeyond, turnSeconds, type PagingMode, type WheelTrace } from './paging'

/** After scrolling that the pager did not start has been at rest this long, the page is set on a stop. */
const SETTLE_AFTER_MS = 170
/** A turn gives way when the page is found this far from where the turn last put it: someone else scrolled. */
const FOREIGN_SCROLL_PX = 3
/** Keys that are left to the control that has the focus: typing, a list's own arrows, a field of its own. */
const OWN_KEYS = 'input, textarea, select, [contenteditable=""], [contenteditable="true"], [role="tablist"], [role="listbox"], [role="menu"]'
/** An element that can be scrolled by the keyboard keeps the keys; one that merely takes focus does not. */
const SCROLL_OWNER = '[tabindex="0"], [tabindex="-1"]'
const OWN_SPACE = 'button, a[href], summary, [role="button"]'

export interface LandingPagerOptions {
  /** Scroll positions of the page's stops, from the top down: the top of each screen, then the page's end. */
  stops: () => readonly number[]
  /** Height of one screen in pixels. */
  screen: () => number
  mode: Ref<PagingMode>
  /** Whether the Landing is the page in front of the visitor. */
  active: () => boolean
  /** The visitor asked for less motion: moves happen at once. */
  reducedMotion: () => boolean
  /**
   * Called at once whenever the pager has put the page somewhere, in the same frame. The browser's own
   * scroll event comes a frame later, which is too late for anything that is drawn against the scroll.
   */
  moved?: () => void
}

/**
 * Turns the Landing screen by screen. The page stays an ordinary scrolling document, so links to a
 * section, the Back button, find-in-page and a screen reader work as anywhere; this only decides where
 * scrolling comes to rest. Which stop the page is at is always read from the scroll position.
 * It needs no animation library, so it works from the first paint on.
 */
export function useLandingPager(options: LandingPagerOptions) {
  /** The stop the page is at, or the nearest one while it moves. */
  const current = ref(0)
  let turning = false
  /** The stop a move in progress is heading for. */
  let heading = -1
  let frame = 0
  /** What was asked for during a turn: a direction, or a stop (`queuedTo >= 0`), for when the turn ends. */
  let queued: 1 | -1 | 0 = 0
  let queuedTo = -1
  let trace: WheelTrace | undefined
  let settleTimer: ReturnType<typeof setTimeout> | undefined

  const paged = () => options.mode.value === 'paged' && options.active()
  const place = (top: number) => {
    // A page that is already there is left alone; what follows the scroll is told all the same.
    if (Math.abs(window.scrollY - top) > 0.01) window.scrollTo({ top, behavior: 'instant' })
    options.moved?.()
  }

  /** Stops a move in progress where it is. */
  function cancel() {
    cancelAnimationFrame(frame)
    clearTimeout(settleTimer)
    frame = 0
    turning = false
    heading = -1
    queued = 0
    queuedTo = -1
  }

  function moveTo(index: number, instant = false) {
    const stops = options.stops()
    if (!stops.length) return
    const to = Math.min(stops.length - 1, Math.max(0, index))
    const target = stops[to]
    cancel()
    const from = window.scrollY
    if (instant || options.reducedMotion()) return place(target)
    // Where the browser snaps by itself it also animates by itself: put there at once, because a smooth
    // scroll may stop at the first snap point it passes rather than the one that was asked for.
    if (options.mode.value !== 'paged') return place(target)
    const seconds = turnSeconds(Math.abs(target - from), options.screen())
    if (Math.abs(target - from) < 1 || !seconds) return place(target)
    turning = true
    heading = to
    const began = performance.now()
    let put = from
    const step = (now: number) => {
      // The scrollbar, a finger or the browser moved the page meanwhile: it is theirs.
      if (Math.abs(window.scrollY - put) > FOREIGN_SCROLL_PX) {
        cancel()
        return scrolled()
      }
      const t = Math.min(1, (now - began) / (seconds * 1000))
      place(from + (target - from) * easeTurn(t))
      put = window.scrollY
      if (t < 1) {
        frame = requestAnimationFrame(step)
        return
      }
      frame = 0
      turning = false
      heading = -1
      const next = queued
      const stop = queuedTo
      queued = 0
      queuedTo = -1
      if (stop >= 0) moveTo(stop)
      else if (next) turn(next)
    }
    frame = requestAnimationFrame(step)
  }

  /** One screen down (1) or up (-1). */
  function turn(direction: 1 | -1) {
    moveTo(stopBeyond(window.scrollY, options.stops(), direction))
  }

  /** Asks for a stop: at once, or kept for the end of the turn in progress — one, not a queue. */
  function goTo(index: number, instant = false) {
    if (instant || !turning) return moveTo(index, instant)
    queued = 0
    queuedTo = index
  }

  /**
   * The stops have moved (the window was resized, the fonts arrived): a turn in progress is aimed again
   * at the screen it was heading for, which is no longer where it was going to land.
   */
  function retarget() {
    if (!turning || heading < 0) return
    // A turn aimed again is not a new request: what was asked for during it stays asked for.
    const keep: [1 | -1 | 0, number] = [queued, queuedTo]
    moveTo(heading)
    queued = keep[0]
    queuedTo = keep[1]
  }

  /** Sets a page that came to rest between two stops on the nearer one. */
  function settle() {
    clearTimeout(settleTimer)
    if (turning || !paged()) return
    const stops = options.stops()
    if (!stops.length) return
    const index = nearestStop(window.scrollY, stops)
    if (Math.abs(stops[index] - window.scrollY) > 1) moveTo(index)
  }

  /** Reads which stop the page is at; to be called when the stops themselves have changed. */
  function sync() {
    const stops = options.stops()
    if (stops.length) current.value = nearestStop(window.scrollY, stops)
  }

  function scrolled() {
    sync()
    if (turning || !paged()) return
    // Scrolling that came from elsewhere: the scrollbar, find-in-page, the browser showing what has the focus.
    clearTimeout(settleTimer)
    settleTimer = setTimeout(settle, SETTLE_AFTER_MS)
  }

  function wheel(event: WheelEvent) {
    // With Ctrl the wheel zooms the page; that stays the browser's.
    if (!paged() || event.ctrlKey) return
    const size = Math.abs(event.deltaY) * (event.deltaMode === 1 ? 40 : event.deltaMode === 2 ? 800 : 1)
    // A sideways gesture belongs to whatever scrolls sideways under the pointer...
    if (size <= Math.abs(event.deltaX)) return
    // ...and an event too small to scroll anything is left alone rather than consumed.
    if (size < 1) return
    event.preventDefault()
    const direction = event.deltaY > 0 ? 1 : -1
    const read = readWheel(trace, performance.now(), size, direction)
    trace = read.trace
    if (!read.turn) return
    // A second gesture during a turn is kept for when the turn has ended; more than one is not.
    if (turning) queued = direction
    else turn(direction)
  }

  /** Whether the focused element has a scroll of its own that the arrows and the page keys would move. */
  function scrollsItself(target: Element | null) {
    const owner = target?.closest<HTMLElement>(SCROLL_OWNER)
    return !!owner && (owner.scrollHeight > owner.clientHeight + 1 || owner.scrollWidth > owner.clientWidth + 1)
  }

  function key(event: KeyboardEvent) {
    if (!paged() || event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) return
    const target = event.target instanceof Element ? event.target : null
    if (target?.closest(OWN_KEYS) || scrollsItself(target)) return
    let direction: 1 | -1
    switch (event.key) {
      case 'PageDown':
      case 'ArrowDown':
        direction = 1
        break
      case 'PageUp':
      case 'ArrowUp':
        direction = -1
        break
      case ' ':
        if (target?.closest(OWN_SPACE)) return
        direction = event.shiftKey ? -1 : 1
        break
      case 'Home':
        event.preventDefault()
        return goTo(0)
      case 'End':
        event.preventDefault()
        return goTo(options.stops().length - 1)
      default:
        return
    }
    event.preventDefault()
    // A key pressed during a turn is kept for its end, as a wheel's is: two presses go two screens.
    if (turning) queued = direction
    else turn(direction)
  }

  /** A finger on the page takes it over; the pager lets go and sets the page afterwards. */
  function taken() {
    if (!turning) return
    cancel()
    scrolled()
  }

  onMounted(() => {
    window.addEventListener('wheel', wheel, { passive: false })
    window.addEventListener('keydown', key)
    window.addEventListener('scroll', scrolled, { passive: true })
    window.addEventListener('touchstart', taken, { passive: true })
  })
  onBeforeUnmount(() => {
    cancel()
    clearTimeout(settleTimer)
    window.removeEventListener('wheel', wheel)
    window.removeEventListener('keydown', key)
    window.removeEventListener('scroll', scrolled)
    window.removeEventListener('touchstart', taken)
  })

  return {
    current,
    moveTo,
    turn,
    goTo,
    retarget,
    settle,
    sync,
    turning: () => turning,
    /** The stop a turn in progress is heading for; -1 while the page rests or is scrolled by hand. */
    heading: () => heading
  }
}
