// How the Landing moves from one screen to the next. Pure functions; useLandingPager applies them.

export type PagingMode =
  /** A wheel gesture or a key turns exactly one screen: for a mouse or a trackpad. */
  | 'paged'
  /** The browser's own scroll snapping: for touch, where the page follows the finger. */
  | 'snap'
  /** Ordinary scrolling. */
  | 'free'

/**
 * The page turns screen by screen only where every screen fits in the window; a visitor who asked for
 * less motion scrolls as on any other page.
 */
export function choosePagingMode(page: { fits: boolean, reducedMotion: boolean, touch: boolean }): PagingMode {
  if (!page.fits || page.reducedMotion) return 'free'
  return page.touch ? 'snap' : 'paged'
}

/** The stop that lies nearest to a scroll position. */
export function nearestStop(scroll: number, stops: readonly number[]) {
  let nearest = 0
  for (let index = 1; index < stops.length; index++) {
    if (Math.abs(stops[index] - scroll) < Math.abs(stops[nearest] - scroll)) nearest = index
  }
  return nearest
}

/**
 * The stop one step on from a scroll position, down (1) or up (-1). A position between two stops is on
 * its way already, so the step ends at the next stop in that direction; at either end it stays there.
 */
export function stopBeyond(scroll: number, stops: readonly number[], direction: 1 | -1, slack = 2) {
  if (direction > 0) {
    const index = stops.findIndex(stop => stop > scroll + slack)
    return index < 0 ? Math.max(0, stops.length - 1) : index
  }
  for (let index = stops.length - 1; index >= 0; index--) {
    if (stops[index] < scroll - slack) return index
  }
  return 0
}

/** Seconds a move takes: one screen in well under a second; a longer way takes longer, but not in proportion. */
export function turnSeconds(distance: number, screen: number) {
  if (distance <= 0 || screen <= 0) return 0
  return Math.min(1.6, 0.3 + 0.45 * Math.sqrt(distance / screen))
}

/** Slow out of the old screen and slow into the new one. */
export function easeTurn(t: number) {
  if (t <= 0) return 0
  if (t >= 1) return 1
  return t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2
}

/** What is remembered of the wheel between two of its events. */
export interface WheelTrace {
  /** When the last event came, in milliseconds. */
  at: number
  /** Down (1) or up (-1). */
  direction: 1 | -1
  /** When the gesture began. */
  beganAt: number
  /** The largest of the gesture's events, in pixels. */
  peak: number
  /** How small its events have become; follows them down at once and up only slowly. */
  low: number
  /** How far the gesture has asked to scroll so far. */
  sum: number
  /** The gesture has turned the page. */
  spent: boolean
}

/** Events this far apart belong to two gestures. */
const WHEEL_PAUSE_MS = 140
/** A gesture's own first events grow; growth counts as a new gesture only after this long. */
const WHEEL_REGROW_MS = 280
/** A wheel that is turned on and on without a pause begins a new gesture after this long... */
const WHEEL_HOLD_MS = 1100
/** ...if its events are still this large. */
const WHEEL_HOLD_SIZE = 60
/** A gesture turns the page once it has asked for this much; a finger brushing the trackpad does not. */
const WHEEL_TOLERANCE = 10

/**
 * Takes one wheel event and says whether the page should turn. One gesture turns one screen, however
 * long it lasts. A trackpad goes on sending events for a second or more after the fingers have left it,
 * each smaller than the one before: those belong to the gesture that has already turned the page. A new
 * gesture shows as a pause before it, as a change of direction, as events that had died down and grow
 * again (a second swipe), or as a wheel that keeps turning at full size (it turns again after a while).
 */
export function readWheel(trace: WheelTrace | undefined, now: number, size: number, direction: 1 | -1): { turn: boolean, trace: WheelTrace } {
  let fresh = true
  if (trace && now - trace.at <= WHEEL_PAUSE_MS && direction === trace.direction) {
    const since = now - trace.beganAt
    const again = since > WHEEL_REGROW_MS && trace.low < trace.peak * 0.5 && size > trace.low * 2.2 + 20
    const held = since > WHEEL_HOLD_MS && size >= WHEEL_HOLD_SIZE && trace.low >= trace.peak * 0.7
    fresh = again || held
  }
  const next: WheelTrace = fresh || !trace
    ? { at: now, direction, beganAt: now, peak: size, low: size, sum: size, spent: false }
    : { at: now, direction, beganAt: trace.beganAt, peak: Math.max(trace.peak, size), low: Math.min(trace.low * 1.12 + 0.5, size), sum: trace.sum + size, spent: trace.spent }
  const turn = !next.spent && next.sum >= WHEEL_TOLERANCE
  if (turn) next.spent = true
  return { turn, trace: next }
}
