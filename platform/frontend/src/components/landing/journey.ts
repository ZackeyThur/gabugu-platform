/** A scroll position at which the journey has reached a given point. */
export interface JourneyStop {
  /** Pixels scrolled. */
  at: number
  /** 0..1 */
  progress: number
}

/**
 * How far the journey has come at a scroll position: between two stops it moves evenly with the scroll.
 * Stops are given from the top of the page down; one that would lie above its predecessor counts as lying
 * at it, so a part of the page that has no height does not turn the journey back.
 */
export function journeyAt(scroll: number, stops: readonly JourneyStop[]) {
  if (!stops.length) return 0
  let from = stops[0]
  if (scroll <= from.at) return from.progress
  for (let i = 1; i < stops.length; i++) {
    const to = { at: Math.max(stops[i].at, from.at), progress: stops[i].progress }
    if (scroll < to.at) return from.progress + (to.progress - from.progress) * (scroll - from.at) / (to.at - from.at)
    from = to
  }
  return from.progress
}
